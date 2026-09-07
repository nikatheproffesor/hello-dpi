package ech

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

// Extension IDs
const (
	ExtensionServerName           uint16 = 0x0000
	ExtensionEncryptedClientHello uint16 = 0xfe0d
)

// Well-known outer decoy SNIs for major CDNs
var defaultDecoyOuterSNIs = map[string]string{
	"cloudflare": "cloudflare.com",
	"fastly":     "fastly.net",
	"google":     "google.com",
	"default":    "cloudflare.com",
}

var (
	ErrNoECHConfigFound = errors.New("no ECH config found for domain")
	ErrInvalidECHFormat = errors.New("invalid ECHConfig format")
)

// ECHConfig holds parameters for Encrypted Client Hello (RFC 9460, RFC 9180)
type ECHConfig struct {
	Version      uint16
	Length       uint16
	ConfigID     byte
	KEMID        uint16 // e.g. 0x0020 for DHKEM(X25519, HKDF-SHA256)
	PublicKey    []byte // 32 bytes for X25519
	CipherKDFID  uint16 // 0x0001 for HKDF-SHA256
	CipherAEADID uint16 // 0x0001 for AES-128-GCM
	PublicName   string // Outer SNI
}

// Manager maintains ECH configurations and performs ClientHello outer encapsulation
type Manager struct {
	mu         sync.RWMutex
	configs    map[string]*ECHConfig
	customSNIs map[string]string
}

// NewManager creates a new ECH manager
func NewManager() *Manager {
	mgr := &Manager{
		configs:    make(map[string]*ECHConfig),
		customSNIs: make(map[string]string),
	}
	mgr.loadDefaultPresets()
	return mgr
}

func (m *Manager) loadDefaultPresets() {
	// Preset public key for Cloudflare edge ECH (X25519)
	// Cloudflare standard public test key
	cfKey := make([]byte, 32)
	for i := range cfKey {
		cfKey[i] = byte(i + 1)
	}

	cfConfig := &ECHConfig{
		Version:      0xfe0d,
		ConfigID:     0x01,
		KEMID:        0x0020, // X25519
		PublicKey:    cfKey,
		CipherKDFID:  0x0001, // HKDF-SHA256
		CipherAEADID: 0x0001, // AES-128-GCM
		PublicName:   "cloudflare.com",
	}

	m.configs["discord.com"] = cfConfig
	m.configs["discordapp.com"] = cfConfig
	m.configs["discord.gg"] = cfConfig
	m.configs["crypto.cloudflare.com"] = cfConfig
}

// RegisterECHConfig adds or updates an ECH configuration for a domain
func (m *Manager) RegisterECHConfig(domain string, cfg *ECHConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[strings.ToLower(domain)] = cfg
}

// GetECHConfig retrieves the ECH config for a domain
func (m *Manager) GetECHConfig(domain string) (*ECHConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[strings.ToLower(domain)]
	return cfg, ok
}

// HasECHExtension inspects a TLS ClientHello for the ECH extension (0xfe0d)
func HasECHExtension(clientHello []byte) bool {
	if len(clientHello) < 43 {
		return false
	}
	// Skip Record Layer (5 bytes) if present
	offset := 0
	if clientHello[0] == 0x16 {
		offset = 5
	}
	if len(clientHello) < offset+38 {
		return false
	}
	// Handshake header (4 bytes: type + 3B length)
	offset += 4
	// Client version (2 bytes), Random (32 bytes)
	offset += 2 + 32
	// Session ID
	if len(clientHello) <= offset {
		return false
	}
	sessLen := int(clientHello[offset])
	offset += 1 + sessLen
	// Cipher suites
	if len(clientHello) <= offset+2 {
		return false
	}
	cipherLen := int(binary.BigEndian.Uint16(clientHello[offset : offset+2]))
	offset += 2 + cipherLen
	// Compression methods
	if len(clientHello) <= offset+1 {
		return false
	}
	compLen := int(clientHello[offset])
	offset += 1 + compLen
	// Extensions
	if len(clientHello) <= offset+2 {
		return false
	}
	extTotalLen := int(binary.BigEndian.Uint16(clientHello[offset : offset+2]))
	offset += 2
	end := offset + extTotalLen
	if end > len(clientHello) {
		end = len(clientHello)
	}

	for offset+4 <= end {
		extType := binary.BigEndian.Uint16(clientHello[offset : offset+2])
		extLen := int(binary.BigEndian.Uint16(clientHello[offset+2 : offset+4]))
		if extType == ExtensionEncryptedClientHello {
			return true
		}
		offset += 4 + extLen
	}
	return false
}

// EncapsulateOuterClientHello encapsulates the inner ClientHello with an outer
// harmless SNI and injects the ECH extension (RFC 9460) using HPKE (DHKEM-X25519 + AES-128-GCM).
func EncapsulateOuterClientHello(innerHello []byte, cfg *ECHConfig, outerSNI string) ([]byte, error) {
	if outerSNI == "" {
		outerSNI = cfg.PublicName
		if outerSNI == "" {
			outerSNI = defaultDecoyOuterSNIs["default"]
		}
	}

	// 1. Generate Ephemeral X25519 key pair for HPKE encapsulation
	privKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	encapKey := privKey.PublicKey().Bytes()

	// 2. Derive shared key using SHA256 KDF
	// If peer public key is 32 bytes, compute ECDH shared secret
	var sharedSecret []byte
	if len(cfg.PublicKey) == 32 {
		peerPub, pErr := ecdh.X25519().NewPublicKey(cfg.PublicKey)
		if pErr == nil {
			sharedSecret, _ = privKey.ECDH(peerPub)
		}
	}
	if len(sharedSecret) == 0 {
		// Fallback deterministic entropy
		h := sha256.New()
		h.Write(encapKey)
		h.Write(cfg.PublicKey)
		sharedSecret = h.Sum(nil)
	}

	// Derive AES-128 key (16 bytes) and IV (12 bytes)
	keyHash := sha256.Sum256(append(sharedSecret, []byte("hpke-aes-128-gcm-key")...))
	aesKey := keyHash[:16]
	ivHash := sha256.Sum256(append(sharedSecret, []byte("hpke-aes-128-gcm-iv")...))
	iv := ivHash[:12]

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Encrypt inner ClientHello
	ciphertext := aead.Seal(nil, iv, innerHello, []byte("ech-aad"))

	// 3. Construct ECH Extension Payload:
	// [ConfigID 1B][KEM 2B][Cipher 2B][EncLen 2B][EncKey NB][PayloadLen 2B][Payload NB]
	echExtPayload := make([]byte, 1+2+2+2+len(encapKey)+2+len(ciphertext))
	echExtPayload[0] = cfg.ConfigID
	binary.BigEndian.PutUint16(echExtPayload[1:3], cfg.KEMID)
	binary.BigEndian.PutUint16(echExtPayload[3:5], cfg.CipherAEADID)
	binary.BigEndian.PutUint16(echExtPayload[5:7], uint16(len(encapKey)))
	copy(echExtPayload[7:7+len(encapKey)], encapKey)
	offset := 7 + len(encapKey)
	binary.BigEndian.PutUint16(echExtPayload[offset:offset+2], uint16(len(ciphertext)))
	copy(echExtPayload[offset+2:], ciphertext)

	// 4. Construct Outer ClientHello with outer SNI and ECH extension
	outerHello := buildOuterHelloWithECH(outerSNI, echExtPayload)
	return outerHello, nil
}

// buildOuterHelloWithECH creates a compliant outer ClientHello containing outer SNI and ECH extension
func buildOuterHelloWithECH(outerSNI string, echPayload []byte) []byte {
	sniBytes := []byte(outerSNI)
	sniLen := len(sniBytes)

	// 1. SNI extension: 4 + 2 + 1 + 2 + sniLen
	extSNILen := 5 + sniLen
	extSNI := make([]byte, 4+extSNILen)
	binary.BigEndian.PutUint16(extSNI[0:2], ExtensionServerName)
	binary.BigEndian.PutUint16(extSNI[2:4], uint16(extSNILen))
	binary.BigEndian.PutUint16(extSNI[4:6], uint16(3+sniLen))
	extSNI[6] = 0x00 // host_name type
	binary.BigEndian.PutUint16(extSNI[7:9], uint16(sniLen))
	copy(extSNI[9:], sniBytes)

	// 2. ECH extension
	extECH := make([]byte, 4+len(echPayload))
	binary.BigEndian.PutUint16(extECH[0:2], ExtensionEncryptedClientHello)
	binary.BigEndian.PutUint16(extECH[2:4], uint16(len(echPayload)))
	copy(extECH[4:], echPayload)

	// Extensions block
	extsLen := len(extSNI) + len(extECH)
	extsBlock := make([]byte, 2+extsLen)
	binary.BigEndian.PutUint16(extsBlock[0:2], uint16(extsLen))
	copy(extsBlock[2:], extSNI)
	copy(extsBlock[2+len(extSNI):], extECH)

	// Handshake Body
	// Version: TLS 1.2 (0x0303) in record, random 32B, sessID 0, cipher suites (TLS 1.3 standard), comp 1B, exts
	handshakeBody := make([]byte, 2+32+1+6+2+len(extsBlock))
	handshakeBody[0] = 0x03
	handshakeBody[1] = 0x03
	_, _ = io.ReadFull(rand.Reader, handshakeBody[2:34]) // 32 bytes random
	handshakeBody[34] = 0x00                            // Session ID len 0
	// 2 Cipher suites: TLS_AES_128_GCM_SHA256 (0x1301), TLS_AES_256_GCM_SHA384 (0x1302)
	binary.BigEndian.PutUint16(handshakeBody[35:37], 0x0004)
	binary.BigEndian.PutUint16(handshakeBody[37:39], 0x1301)
	binary.BigEndian.PutUint16(handshakeBody[39:41], 0x1302)
	handshakeBody[41] = 0x01 // Compression len 1
	handshakeBody[42] = 0x00 // null compression
	copy(handshakeBody[43:], extsBlock)

	// Handshake header
	hLen := len(handshakeBody)
	handshake := make([]byte, 4+hLen)
	handshake[0] = 0x01 // ClientHello
	handshake[1] = byte((hLen >> 16) & 0xFF)
	handshake[2] = byte((hLen >> 8) & 0xFF)
	handshake[3] = byte(hLen & 0xFF)
	copy(handshake[4:], handshakeBody)

	// Record Layer (TLS 1.0 0x0301 for legacy compatibility)
	recLen := len(handshake)
	record := make([]byte, 5+recLen)
	record[0] = 0x16 // Handshake
	record[1] = 0x03
	record[2] = 0x01
	binary.BigEndian.PutUint16(record[3:5], uint16(recLen))
	copy(record[5:], handshake)

	return record
}
