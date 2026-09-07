package tlsfingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Profile represents a target browser fingerprint profile
type Profile string

const (
	ProfileChrome130 Profile = "chrome_130"
	ProfileSafari18  Profile = "safari_18"
	ProfileFirefox   Profile = "firefox_130"
)

// Chrome 130 standard cipher suites
var Chrome130CipherSuites = []uint16{
	0x1301, // TLS_AES_128_GCM_SHA256
	0x1302, // TLS_AES_256_GCM_SHA384
	0x1303, // TLS_CHACHA20_POLY1305_SHA256
	0xc02b, // TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	0xc02f, // TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
	0xc02c, // TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
	0xc030, // TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
	0xcca9, // TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256
	0xcca8, // TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
	0xc013, // TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA
	0xc014, // TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA
	0x009c, // TLS_RSA_WITH_AES_128_GCM_SHA256
	0x009d, // TLS_RSA_WITH_AES_256_GCM_SHA384
	0x002f, // TLS_RSA_WITH_AES_128_CBC_SHA
	0x0035, // TLS_RSA_WITH_AES_256_CBC_SHA
}

// Chrome 130 standard extension ordering
var Chrome130Extensions = []uint16{
	0x0000, // server_name
	0x0017, // extended_master_secret
	0xff01, // renegotiation_info
	0x000a, // supported_groups
	0x000b, // ec_point_formats
	0x0023, // session_ticket
	0x0010, // application_layer_protocol_negotiation (ALPN)
	0x0005, // status_request
	0x000d, // signature_algorithms
	0x0033, // key_share
	0x002d, // psk_key_exchange_modes
	0x002b, // supported_versions
	0x001b, // compress_certificate
	0xfe0d, // encrypted_client_hello (when present)
}

// Safari 18 standard cipher suites
var Safari18CipherSuites = []uint16{
	0x1301, 0x1302, 0x1303,
	0xc02c, 0xc02b, 0xcca9,
	0xc030, 0xc02f, 0xcca8,
	0xc024, 0xc028, 0x009d, 0x009c,
}

// Fingerprint holds parsed JA3 / JA4 telemetry
type Fingerprint struct {
	JA3Raw     string
	JA3Hash    string
	JA4        string
	MatchedBrowser string
}

// CalculateJA4 computes the JA4 fingerprint for a given set of TLS attributes
// Format: [protocol][version][sni][cipher_count][ext_count][alpn]_[hash_ciphers]_[hash_exts]
func CalculateJA4(isTCP bool, tlsVersion uint16, hasSNI bool, ciphers []uint16, exts []uint16, alpn string) string {
	var sb strings.Builder

	// 1. Protocol: 't' for TCP, 'q' for QUIC
	if isTCP {
		sb.WriteByte('t')
	} else {
		sb.WriteByte('q')
	}

	// 2. TLS Version: 13 for TLS 1.3, 12 for TLS 1.2
	if tlsVersion == 0x0304 {
		sb.WriteString("13")
	} else {
		sb.WriteString("12")
	}

	// 3. SNI present: 'd' for domain, 'i' for IP
	if hasSNI {
		sb.WriteByte('d')
	} else {
		sb.WriteByte('i')
	}

	// 4. Cipher count (2 digits)
	cCount := len(ciphers)
	if cCount > 99 {
		cCount = 99
	}
	sb.WriteString(fmt.Sprintf("%02d", cCount))

	// 5. Extension count (2 digits)
	eCount := len(exts)
	if eCount > 99 {
		eCount = 99
	}
	sb.WriteString(fmt.Sprintf("%02d", eCount))

	// 6. ALPN (2 chars: e.g. "h2", "11", "00")
	if strings.HasPrefix(alpn, "h2") {
		sb.WriteString("h2")
	} else if strings.HasPrefix(alpn, "http/1.1") {
		sb.WriteString("11")
	} else {
		sb.WriteString("00")
	}

	sb.WriteByte('_')

	// 7. Hash of sorted ciphers (truncated 12 hex chars)
	var cipherStr strings.Builder
	for i, c := range ciphers {
		if i > 0 {
			cipherStr.WriteByte(',')
		}
		cipherStr.WriteString(fmt.Sprintf("%04x", c))
	}
	cHash := sha256.Sum256([]byte(cipherStr.String()))
	sb.WriteString(hex.EncodeToString(cHash[:6]))

	sb.WriteByte('_')

	// 8. Hash of sorted extensions (truncated 12 hex chars)
	var extStr strings.Builder
	for i, e := range exts {
		if i > 0 {
			extStr.WriteByte(',')
		}
		extStr.WriteString(fmt.Sprintf("%04x", e))
	}
	eHash := sha256.Sum256([]byte(extStr.String()))
	sb.WriteString(hex.EncodeToString(eHash[:6]))

	return sb.String()
}

// Impersonator normalizes TLS parameters to resemble real browsers
type Impersonator struct {
	profile Profile
}

// NewImpersonator creates a browser impersonator with specified profile
func NewImpersonator(profile Profile) *Impersonator {
	if profile == "" {
		profile = ProfileChrome130
	}
	return &Impersonator{profile: profile}
}

// TargetCipherSuites returns the list of cipher suites for the configured profile
func (imp *Impersonator) TargetCipherSuites() []uint16 {
	switch imp.profile {
	case ProfileSafari18:
		return Safari18CipherSuites
	default:
		return Chrome130CipherSuites
	}
}

// TargetExtensions returns the extension ordering for the profile
func (imp *Impersonator) TargetExtensions() []uint16 {
	return Chrome130Extensions
}

// GenerateSignature generates the JA4 signature for the configured browser profile
func (imp *Impersonator) GenerateSignature(hasSNI bool, isTCP bool) string {
	ciphers := imp.TargetCipherSuites()
	exts := imp.TargetExtensions()
	return CalculateJA4(isTCP, 0x0304, hasSNI, ciphers, exts, "h2")
}
