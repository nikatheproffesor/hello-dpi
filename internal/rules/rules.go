package rules

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Action defines how a connection to a given host should be routed
type Action int

const (
	ActionDefault  Action = iota // Standard routing
	ActionDirect                 // Direct pass-through: zero manipulation, native latency
	ActionProxyDPI               // DPI bypass: fragmentation and DoH resolution
)

// RuleSet defines the structured rules loaded dynamically or embedded
type RuleSet struct {
	Version      string   `json:"version"`
	UpdatedAt    string   `json:"updated_at"`
	DirectList   []string `json:"direct_list"`
	InterceptList []string `json:"intercept_list"`
}

// Engine evaluates hostnames against smart routing rules
type Engine struct {
	mu           sync.RWMutex
	directSet    map[string]bool
	directSuffix []string
	interSet     map[string]bool
	interSuffix  []string
	version      string
	lastSync     time.Time
	cachePath    string
}

// DefaultRules embedded into the binary
var defaultDirectList = []string{
	// Turkish Banking & Financial Institutions (0ms ping, untampered)
	"ziraatbank.com.tr",
	"isbank.com.tr",
	"garantibbva.com.tr",
	"yapikredi.com.tr",
	"akbank.com.tr",
	"vakifbank.com.tr",
	"halkbank.com.tr",
	"qnb.com.tr",
	"qnbfinansbank.com",
	"denizbank.com.tr",
	"teb.com.tr",
	"ing.com.tr",
	"kuveytturk.com.tr",
	"turkiyefinans.com.tr",
	"enpara.com",
	"papara.com",
	"tosla.com",
	"bkm.com.tr",
	"fast.tcmb.gov.tr",

	// Turkish Government & Municipal Portals
	"turkiye.gov.tr",
	"gib.gov.tr",
	"egm.gov.tr",
	"meb.gov.tr",
	"saglik.gov.tr",
	"enabiz.gov.tr",
	"icisleri.gov.tr",
	"adalet.gov.tr",
	"gelirler.gov.tr",
	"btk.gov.tr",
	"kyk.gov.tr",
	"gsb.gov.tr",
	"wifi.gsb.gov.tr",

	// Captive Portals & Network Health
	"captive.apple.com",
	"connectivitycheck.gstatic.com",
	"connectivitycheck.android.com",
	"msftconnecttest.com",
	"ipv6.msftconnecttest.com",

	// High-performance gaming servers & low-latency CDNs
	"valve.net",
	"valvesoftware.com",
	"steamserver.net",
	"riotgames.com",
	"riotcdn.net",
	"pvp.net",
	"epicgames.com",
	"unrealengine.com",
	"ea.com",
	"origin.com",
	"blizzard.com",
	"battle.net",
}

var defaultInterceptList = []string{
	// Discord
	"discord.com",
	"discord.gg",
	"discordapp.com",
	"discordapp.net",
	"discord.media",
	"discord-attachments-uploads-prd.storage.googleapis.com",

	// Roblox
	"roblox.com",
	"rbxcdn.com",
	"setup.rbxcdn.com",
	"versioncompatibility.api.roblox.com",
	"clientsettingscdn.roblox.com",
	"api.roblox.com",

	// Censored / Throttled Media
	"imgur.com",
	"pastebin.com",
	"wattpad.com",
	"dailymotion.com",
	"streamable.com",

	// Video CDNs
	"youtube.com",
	"googlevideo.com",
	"ytimg.com",
	"youtu.be",
}

// NewEngine initializes the smart rule engine with embedded defaults and local cache
func NewEngine() *Engine {
	eng := &Engine{
		directSet:    make(map[string]bool),
		interSet:     make(map[string]bool),
		version:      "4.0.0-embedded",
		lastSync:     time.Now(),
	}

	// Setup local cache path in user config dir
	if configDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(configDir, "hellodpi")
		_ = os.MkdirAll(dir, 0755)
		eng.cachePath = filepath.Join(dir, "rules.json")
	}

	// Load defaults first
	eng.loadLists(defaultDirectList, defaultInterceptList)

	// Attempt to load from disk cache if exists
	if eng.cachePath != "" {
		if data, err := os.ReadFile(eng.cachePath); err == nil {
			var rs RuleSet
			if json.Unmarshal(data, &rs) == nil && len(rs.DirectList) > 0 {
				eng.loadLists(rs.DirectList, rs.InterceptList)
				eng.version = rs.Version
			}
		}
	}

	return eng
}

// Evaluate determines the routing action for a given hostname
func (e *Engine) Evaluate(host string) Action {
	h := strings.ToLower(strings.TrimSpace(host))
	if colon := strings.IndexByte(h, ':'); colon != -1 {
		h = h[:colon]
	}
	h = strings.TrimSuffix(h, ".")

	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".lan") || strings.HasSuffix(h, ".home") {
		return ActionDirect
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. Direct pass-through checks (Exact + Suffix)
	if e.directSet[h] {
		return ActionDirect
	}
	if strings.HasSuffix(h, ".gov.tr") {
		return ActionDirect
	}
	for _, suf := range e.directSuffix {
		if strings.HasSuffix(h, suf) {
			return ActionDirect
		}
	}

	// 2. Intercept / DPI bypass checks (Exact + Suffix)
	if e.interSet[h] {
		return ActionProxyDPI
	}
	for _, suf := range e.interSuffix {
		if strings.HasSuffix(h, suf) {
			return ActionProxyDPI
		}
	}

	return ActionDefault
}

// SyncRemote updates rules from a remote URL or GitHub Releases
func (e *Engine) SyncRemote(url string) error {
	if url == "" {
		url = "https://raw.githubusercontent.com/nikatheproffesor/hello-dpi/main/rules.json"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var rs RuleSet
	if err := json.Unmarshal(data, &rs); err != nil {
		return err
	}

	e.mu.Lock()
	e.loadLists(rs.DirectList, rs.InterceptList)
	e.version = rs.Version
	e.lastSync = time.Now()
	e.mu.Unlock()

	// Save to local cache
	if e.cachePath != "" {
		_ = os.WriteFile(e.cachePath, data, 0644)
	}

	return nil
}

// loadLists loads and compiles direct and intercept lists into fast maps and suffixes
func (e *Engine) loadLists(direct, intercept []string) {
	e.directSet = make(map[string]bool)
	e.directSuffix = nil
	e.interSet = make(map[string]bool)
	e.interSuffix = nil

	for _, d := range direct {
		item := strings.ToLower(strings.TrimSpace(d))
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "*.") {
			e.directSuffix = append(e.directSuffix, item[1:]) // e.g. ".bank.com.tr"
		} else {
			e.directSet[item] = true
			e.directSuffix = append(e.directSuffix, "."+item)
		}
	}

	for _, i := range intercept {
		item := strings.ToLower(strings.TrimSpace(i))
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "*.") {
			e.interSuffix = append(e.interSuffix, item[1:])
		} else {
			e.interSet[item] = true
			e.interSuffix = append(e.interSuffix, "."+item)
		}
	}
}

// Stats returns the active counts of direct and intercepted domains
func (e *Engine) Stats() (version string, directCount int, interceptCount int, lastSync time.Time) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.version, len(e.directSet), len(e.interSet), e.lastSync
}
