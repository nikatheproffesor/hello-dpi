package dpi

import (
	"net"
)

// AdaptiveStrategy inspects the protocol type of the incoming payload and dynamically
// applies the optimal bypass strategy (TLS record split for HTTPS, Host trick for HTTP,
// and first-byte split for raw/unknown protocols).
type AdaptiveStrategy struct {
	tlsStrategy  BypassStrategy
	httpStrategy BypassStrategy
	rawStrategy  BypassStrategy
}

// NewAdaptiveStrategy creates an adaptive multi-protocol strategy.
func NewAdaptiveStrategy() *AdaptiveStrategy {
	return &AdaptiveStrategy{
		tlsStrategy:  NewTLSRecordSplitStrategy(5, 5),
		httpStrategy: NewHTTPHostTrickStrategy(TrickHostCaseMix, 5),
		rawStrategy:  NewFirstByteSplitStrategy(3),
	}
}

func (s *AdaptiveStrategy) Name() string {
	return string(SplitAuto)
}

func (s *AdaptiveStrategy) Apply(conn net.Conn, data []byte, info ParsedInfo) error {
	switch info.Type {
	case TypeTLSClientHello:
		return s.tlsStrategy.Apply(conn, data, info)
	case TypeHTTPRequest:
		return s.httpStrategy.Apply(conn, data, info)
	default:
		return s.rawStrategy.Apply(conn, data, info)
	}
}

func init() {
	RegisterStrategy(NewAdaptiveStrategy())
}
