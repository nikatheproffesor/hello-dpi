package rules

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	ActionKernel                 // Kernel divert required (L3/L4 WinDivert for direct socket apps)
	ActionBlock                  // Block request (e.g. telemetry or rogue UDP QUIC)
)

// RuleSet defines the structured rules loaded dynamically or embedded
type RuleSet struct {
	Version       string   `json:"version"`
	UpdatedAt     string   `json:"updated_at"`
	DirectList    []string `json:"direct_list"`
	InterceptList []string `json:"intercept_list"`
	KernelList    []string `json:"kernel_list,omitempty"`
	DirectCIDRs   []string `json:"direct_cidrs,omitempty"`
}

// Engine evaluates hostnames, IP/CIDR, process names, and ports against smart routing rules
type Engine struct {
	mu           sync.RWMutex
	directSet    map[string]bool
	directSuffix []string
	interSet     map[string]bool
	interSuffix  []string
	kernelSet    map[string]bool
	directCIDRs  []*net.IPNet
	version      string
	lastSync     time.Time
	cachePath    string
}

// Anti-cheat process deny-list: NEVER intercept or divert these processes to prevent bans
var antiCheatProcesses = map[string]bool{
	"vgc.exe":                     true, // Riot Vanguard
	"vgtray.exe":                  true,
	"easyanticheat.exe":           true, // EasyAntiCheat
	"easyanticheat_eos.exe":       true,
	"beservice.exe":               true, // BattlEye
	"bedaisy.sys":                 true,
	"cs2.exe":                     true, // Counter-Strike 2
	"valorant.exe":                true, // Valorant
	"valorant-win64-shipping.exe": true, // Valorant main game and voice engine
	"riotclientux.exe":            true,
	"riotclientservices.exe":      true,
	"r5apex.exe":                  true, // Apex Legends
	"leagueclient.exe":            true, // League of Legends
	"faceit.exe":                  true, // FACEIT Anti-Cheat
	"faceitservice.exe":           true,
	"pubg.exe":                    true,
	"tslgame.exe":                 true, // PUBG
	"genshinimpact.exe":           true,
	"overwatch.exe":               true,
	"destiny2.exe":                true,
	"warzone.exe":                 true, // Call of Duty Ricochet
	"cod.exe":                     true,
	"rainbowsix.exe":              true, // Rainbow Six Siege
	"rainbowsix_vulkan.exe":       true,
	"deadbydaylight.exe":          true,
	"epicgameslauncher.exe":       true,
	"steam.exe":                   true,
}

// Default rules embedded into binary
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

	// High-performance gaming servers & low-latency CDNs (Valorant & Vivox voice 100% direct)
	"valve.net",
	"valvesoftware.com",
	"steamserver.net",
	"riotgames.com",
	"riotcdn.net",
	"pvp.net",
	"valorant.com",
	"leagueoflegends.com",
	"vivox.com",
	"vrtx.riotgames.com",
	"voice.riotgames.com",
	"rchat.riotgames.com",
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

var defaultKernelList = []string{
	"robloxplayerbeta.exe",
	"robloxplayerlauncher.exe",
	"roblox.exe",
}

var defaultDirectCIDRs = []string{
	"127.0.0.0/8",    // IPv4 loopback
	"::1/128",        // IPv6 loopback
	"10.0.0.0/8",     // RFC1918 (GSB WiFi / KYK dorm networks)
	"172.16.0.0/12",  // RFC1918
	"192.168.0.0/16", // RFC1918
	"169.254.0.0/16", // RFC3927 link-local
}

// NewEngine initializes the smart rule engine with embedded defaults and local cache
func NewEngine() *Engine {
	eng := &Engine{
		directSet: make(map[string]bool),
		interSet:  make(map[string]bool),
		kernelSet: make(map[string]bool),
		version:   "5.0.0-embedded",
		lastSync:  time.Now(),
	}

	// Setup local cache path in user config dir
	if configDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(configDir, "hellodpi")
		_ = os.MkdirAll(dir, 0755)
		eng.cachePath = filepath.Join(dir, "rules.json")
	}

	// Load defaults
	eng.loadLists(defaultDirectList, defaultInterceptList, defaultKernelList, defaultDirectCIDRs)

	// Attempt to load from disk cache if exists
	if eng.cachePath != "" {
		if data, err := os.ReadFile(eng.cachePath); err == nil {
			var rs RuleSet
			if json.Unmarshal(data, &rs) == nil && len(rs.DirectList) > 0 {
				eng.loadLists(rs.DirectList, rs.InterceptList, rs.KernelList, rs.DirectCIDRs)
				eng.version = rs.Version
			}
		}
	}

	return eng
}

func cleanProcessBase(processName string) string {
	p := strings.ToLower(strings.TrimSpace(processName))
	if idx := strings.LastIndexAny(p, `/\`); idx != -1 {
		p = p[idx+1:]
	}
	return p
}

// IsAntiCheatProcess checks if a process is on the strict deny-list
func IsAntiCheatProcess(processName string) bool {
	p := cleanProcessBase(processName)
	return antiCheatProcesses[p]
}

// RequiresKernel checks whether traffic from this process requires L3/L4 WinDivert
func (e *Engine) RequiresKernel(processName string, host string) bool {
	if IsAntiCheatProcess(processName) {
		return false
	}
	p := cleanProcessBase(processName)

	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.kernelSet[p] {
		return true
	}
	if strings.Contains(strings.ToLower(host), "roblox") {
		return true
	}
	return false
}

// Evaluate determines the routing action for a given hostname
func (e *Engine) Evaluate(host string) Action {
	return e.EvaluateTarget(host, 0, "")
}

// EvaluateTarget determines the routing action considering host, port, and process name
func (e *Engine) EvaluateTarget(host string, port int, processName string) Action {
	// 1. Anti-cheat protection check
	if processName != "" && IsAntiCheatProcess(processName) {
		return ActionDirect
	}

	h := strings.ToLower(strings.TrimSpace(host))
	if hostPart, _, err := net.SplitHostPort(h); err == nil {
		h = hostPart
	}
	h = strings.Trim(h, "[]")
	h = strings.TrimSuffix(h, ".")

	if h == "localhost" || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".lan") || strings.HasSuffix(h, ".home") {
		return ActionDirect
	}

	// 2. IP / CIDR check
	if ip := net.ParseIP(h); ip != nil {
		if e.isDirectIP(ip) {
			return ActionDirect
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// 3. Process check
	if processName != "" {
		p := cleanProcessBase(processName)
		if e.kernelSet[p] {
			return ActionKernel
		}
	}

	// 4. Direct pass-through checks (Exact + Suffix)
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

	// 5. Intercept / DPI bypass checks (Exact + Suffix)
	if e.interSet[h] {
		if strings.Contains(h, "roblox") {
			return ActionKernel
		}
		return ActionProxyDPI
	}
	for _, suf := range e.interSuffix {
		if strings.HasSuffix(h, suf) {
			if strings.Contains(suf, "roblox") {
				return ActionKernel
			}
			return ActionProxyDPI
		}
	}

	return ActionDefault
}

func (e *Engine) isDirectIP(ip net.IP) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, block := range e.directCIDRs {
		if block.Contains(ip) {
			return true
		}
	}
	return false
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
		return fmt.Errorf("remote rules sync failed with HTTP status %d", resp.StatusCode)
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
	e.loadLists(rs.DirectList, rs.InterceptList, rs.KernelList, rs.DirectCIDRs)
	e.version = rs.Version
	e.lastSync = time.Now()
	e.mu.Unlock()

	// Save to local cache
	if e.cachePath != "" {
		_ = os.WriteFile(e.cachePath, data, 0644)
	}

	return nil
}

// loadLists loads and compiles direct and intercept lists into fast maps, suffixes, and CIDR blocks
func (e *Engine) loadLists(direct, intercept, kernel, cidrs []string) {
	e.directSet = make(map[string]bool)
	e.directSuffix = nil
	e.interSet = make(map[string]bool)
	e.interSuffix = nil
	e.kernelSet = make(map[string]bool)
	e.directCIDRs = nil

	for _, d := range direct {
		item := strings.ToLower(strings.TrimSpace(d))
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "*.") {
			e.directSuffix = append(e.directSuffix, item[1:])
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

	for _, k := range kernel {
		item := strings.ToLower(strings.TrimSpace(k))
		if item != "" {
			e.kernelSet[item] = true
		}
	}

	if len(cidrs) == 0 {
		cidrs = defaultDirectCIDRs
	}
	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			e.directCIDRs = append(e.directCIDRs, block)
		}
	}
}

// Stats returns the active counts of direct and intercepted domains
func (e *Engine) Stats() (version string, directCount int, interceptCount int, lastSync time.Time) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.version, len(e.directSet), len(e.interSet), e.lastSync
}
