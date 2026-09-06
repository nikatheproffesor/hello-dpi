package rules

import (
	"testing"
)

func TestEvaluateDomains(t *testing.T) {
	eng := NewEngine()

	// Turkish Banking & Gov -> Direct
	if act := eng.Evaluate("isbank.com.tr"); act != ActionDirect {
		t.Errorf("Expected isbank.com.tr to be ActionDirect, got %d", act)
	}
	if act := eng.Evaluate("subdomain.turkiye.gov.tr"); act != ActionDirect {
		t.Errorf("Expected *.gov.tr to be ActionDirect, got %d", act)
	}

	// Captive portal -> Direct
	if act := eng.Evaluate("wifi.gsb.gov.tr"); act != ActionDirect {
		t.Errorf("Expected wifi.gsb.gov.tr to be ActionDirect, got %d", act)
	}

	// Discord -> ProxyDPI
	if act := eng.Evaluate("discord.com"); act != ActionProxyDPI {
		t.Errorf("Expected discord.com to be ActionProxyDPI, got %d", act)
	}
	if act := eng.Evaluate("gateway.discord.gg"); act != ActionProxyDPI {
		t.Errorf("Expected *.discord.gg to be ActionProxyDPI, got %d", act)
	}
}

func TestEvaluateCIDRs(t *testing.T) {
	eng := NewEngine()

	// Private IPs should be ActionDirect
	if act := eng.Evaluate("192.168.1.1"); act != ActionDirect {
		t.Errorf("Expected 192.168.1.1 to be ActionDirect, got %d", act)
	}
	if act := eng.Evaluate("10.50.2.1"); act != ActionDirect {
		t.Errorf("Expected 10.50.2.1 to be ActionDirect, got %d", act)
	}
	if act := eng.Evaluate("127.0.0.1"); act != ActionDirect {
		t.Errorf("Expected 127.0.0.1 to be ActionDirect, got %d", act)
	}
}

func TestAntiCheatProtection(t *testing.T) {
	eng := NewEngine()

	// Vanguard & EAC must be detected and NEVER diverted
	if !IsAntiCheatProcess("vgc.exe") {
		t.Errorf("Expected vgc.exe to be detected as anti-cheat")
	}
	if !IsAntiCheatProcess(`C:\Riot Games\VALORANT\live\valorant.exe`) {
		t.Errorf("Expected valorant.exe path to be detected as anti-cheat")
	}

	// Even if target is somehow in intercept list, anti-cheat process forces ActionDirect
	act := eng.EvaluateTarget("discord.com", 443, "valorant.exe")
	if act != ActionDirect {
		t.Errorf("Expected anti-cheat process to force ActionDirect, got %d", act)
	}
}

func TestRequiresKernel(t *testing.T) {
	eng := NewEngine()

	if !eng.RequiresKernel("RobloxPlayerBeta.exe", "roblox.com") {
		t.Errorf("Expected Roblox to require kernel")
	}
	if eng.RequiresKernel("vgc.exe", "discord.com") {
		t.Errorf("Anti-cheat should never require kernel")
	}
}
