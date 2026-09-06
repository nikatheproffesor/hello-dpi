package dpi

import (
	"net"
	"strings"
)

// ShouldBlockQUIC checks if a connection attempt targets QUIC / HTTP-3 (UDP port 443)
// and should be dropped or rejected to force the client to fallback to TCP+TLS.
func ShouldBlockQUIC(network, targetAddr string) bool {
	if !strings.HasPrefix(strings.ToLower(network), "udp") {
		return false
	}

	_, port, err := net.SplitHostPort(targetAddr)
	if err != nil {
		return strings.HasSuffix(targetAddr, ":443")
	}

	return port == "443"
}

// SOCKS5QUICRejectReply returns the RFC 1928 response for rejecting UDP ASSOCIATE (Command not supported)
func SOCKS5QUICRejectReply() []byte {
	return []byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0}
}
