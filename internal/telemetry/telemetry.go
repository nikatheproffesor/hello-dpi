package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hellodpi/hellodpi/internal/probe"
)

// Record holds a single zero-PII anonymous event
type Record struct {
	ISPEstimate       string `json:"isp_estimate"`
	RTTBand           string `json:"rtt_band"`
	DNSPoisoned       bool   `json:"dns_poisoned"`
	Strategy          string `json:"strategy"`
	DomainGroup       string `json:"domain_group"`
	Success           bool   `json:"success"`
	FallbackTriggered bool   `json:"fallback_triggered"`
	Timestamp         int64  `json:"timestamp"`
}

// ISPStats holds aggregated reliability metrics for an ISP profile
type ISPStats struct {
	TotalProbes  int            `json:"total_probes"`
	SuccessCount int            `json:"success_count"`
	SuccessRate  float64        `json:"success_rate"`
	TopStrategy  string         `json:"top_strategy"`
	StrategyWins map[string]int `json:"strategy_wins"`
	LastSeen     time.Time      `json:"last_seen"`
}

// Collector manages opt-in anonymous telemetry and local ISP benchmarking
type Collector struct {
	mu         sync.RWMutex
	enabled    bool
	configPath string
	history    []Record
	ispTable   map[string]*ISPStats
}

var (
	defaultCollector *Collector
	once             sync.Once
)

// Default returns the singleton telemetry collector
func Default() *Collector {
	once.Do(func() {
		defaultCollector = NewCollector()
	})
	return defaultCollector
}

// NewCollector creates a telemetry collector (opt-in, default disabled)
func NewCollector() *Collector {
	c := &Collector{
		enabled:  false, // Strictly disabled by default
		ispTable: make(map[string]*ISPStats),
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(configDir, "hellodpi")
		_ = os.MkdirAll(dir, 0755)
		c.configPath = filepath.Join(dir, "telemetry.json")
	}

	c.loadConfig()
	return c
}

// SetEnabled toggles opt-in state and persists preference
func (c *Collector) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
	c.saveConfig()
}

// IsEnabled checks whether the user has opted into anonymous telemetry
func (c *Collector) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// RecordProbe ingests a completed auto-tune result without any PII
func (c *Collector) RecordProbe(res *probe.Result) {
	if res == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ispName := res.ISPName
	if ispName == "" {
		ispName = "Bilinmeyen İSS"
	}

	stats, exists := c.ispTable[ispName]
	if !exists {
		stats = &ISPStats{
			StrategyWins: make(map[string]int),
		}
		c.ispTable[ispName] = stats
	}

	stats.TotalProbes++
	if res.BypassVerified {
		stats.SuccessCount++
	}
	if stats.TotalProbes > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalProbes)
	}
	if res.BestStrategy != "" {
		stats.StrategyWins[res.BestStrategy]++
		// Re-evaluate top strategy
		top := ""
		topWins := -1
		for strat, wins := range stats.StrategyWins {
			if wins > topWins {
				topWins = wins
				top = strat
			}
		}
		stats.TopStrategy = top
	}
	stats.LastSeen = time.Now()

	// If opt-in is active, buffer the anonymized event
	if c.enabled {
		rec := Record{
			ISPEstimate: res.ISPName,
			RTTBand:     res.ISPFingerprint.RTTBand,
			DNSPoisoned: res.ISPFingerprint.DNSPoisoned,
			Strategy:    res.BestStrategy,
			DomainGroup: "global",
			Success:     res.BypassVerified,
			Timestamp:   time.Now().Unix(),
		}
		c.history = append(c.history, rec)
		if len(c.history) > 200 {
			c.history = c.history[len(c.history)-200:]
		}
	}
}

// RecordFallback logs an automatic runtime fallback event
func (c *Collector) RecordFallback(group probe.DomainGroup, fromStrat, toStrat string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return
	}

	rec := Record{
		DomainGroup:       string(group),
		Strategy:          toStrat,
		FallbackTriggered: true,
		Success:           true,
		Timestamp:         time.Now().Unix(),
	}
	c.history = append(c.history, rec)
	if len(c.history) > 200 {
		c.history = c.history[len(c.history)-200:]
	}
}

// GetISPSuccessTable returns the current local ISP reliability benchmarks
func (c *Collector) GetISPSuccessTable() map[string]ISPStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make(map[string]ISPStats)
	for k, v := range c.ispTable {
		res[k] = *v
	}
	return res
}

// GetStats returns summary telemetry state
func (c *Collector) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"opt_in":        c.enabled,
		"total_events":  len(c.history),
		"known_isps":    len(c.ispTable),
		"zero_pii":      true,
		"privacy_note":  "Sıfır kişisel veri: IP, hesap, URL, domain içeriği kaydedilmez.",
	}
}

type configData struct {
	Enabled bool `json:"enabled"`
}

func (c *Collector) loadConfig() {
	if c.configPath == "" {
		return
	}
	data, err := os.ReadFile(c.configPath)
	if err != nil {
		return
	}
	var cfg configData
	if json.Unmarshal(data, &cfg) == nil {
		c.enabled = cfg.Enabled
	}
}

func (c *Collector) saveConfig() {
	if c.configPath == "" {
		return
	}
	cfg := configData{Enabled: c.enabled}
	if data, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		_ = os.WriteFile(c.configPath, data, 0644)
	}
}
