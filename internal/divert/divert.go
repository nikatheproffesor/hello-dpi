package divert

import "sync"

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
