package divert

import "sync"

// KernelOptions specifies desync and diversion parameters for the L3/L4 kernel engine
type KernelOptions struct {
	StrategyName  string `json:"strategy_name"`
	WrongSeq      bool   `json:"wrong_seq"`
	WrongChecksum bool   `json:"wrong_checksum"`
	DropQUIC      bool   `json:"drop_quic"`
	FakeTTL       int    `json:"fake_ttl"`
	DNSAddr       string `json:"dns_addr"`
	DNSPort       string `json:"dns_port"`
}

// Status represents the current state of the Kernel Divert engine
type Status struct {
	Supported bool   `json:"supported"`
	Active    bool   `json:"active"`
	PID       int    `json:"pid"`
	Message   string `json:"message"`
}

var (
	stateMu sync.RWMutex
)
