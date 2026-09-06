package rules

import (
	"testing"
)

func TestRuleEngine_Evaluate(t *testing.T) {
	eng := NewEngine()

	tests := []struct {
		host     string
		expected Action
	}{
		// Direct banking & government
		{"ziraatbank.com.tr", ActionDirect},
		{"sub.ziraatbank.com.tr", ActionDirect},
		{"isbank.com.tr", ActionDirect},
		{"turkiye.gov.tr", ActionDirect},
		{"vatandas.uyap.gov.tr", ActionDirect},
		{"fast.tcmb.gov.tr", ActionDirect},

		// Direct gaming / low-latency
		{"valve.net", ActionDirect},
		{"riotgames.com", ActionDirect},

		// Local networks
		{"localhost", ActionDirect},
		{"printer.local", ActionDirect},

		// DPI bypass targets
		{"discord.com", ActionProxyDPI},
		{"gateway.discord.gg", ActionProxyDPI},
		{"media.discordapp.net", ActionProxyDPI},
		{"roblox.com", ActionProxyDPI},
		{"setup.rbxcdn.com", ActionProxyDPI},
		{"imgur.com", ActionProxyDPI},
		{"pastebin.com", ActionProxyDPI},
		{"youtube.com", ActionProxyDPI},
		{"googlevideo.com", ActionProxyDPI},

		// Normal untracked host defaults
		{"example.com", ActionDefault},
		{"github.com", ActionDefault},
	}

	for _, tt := range tests {
		got := eng.Evaluate(tt.host)
		if got != tt.expected {
			t.Errorf("Evaluate(%q) = %v, want %v", tt.host, got, tt.expected)
		}
	}
}
