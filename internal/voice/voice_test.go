package voice

import (
	"encoding/binary"
	"testing"
)

func TestBuildSTUNBindingRequest(t *testing.T) {
	req := BuildSTUNBindingRequest()
	if len(req) != 20 {
		t.Fatalf("Expected 20 bytes STUN request, got %d", len(req))
	}

	msgType := binary.BigEndian.Uint16(req[0:2])
	if msgType != 0x0001 {
		t.Errorf("Expected STUN message type 0x0001, got 0x%04x", msgType)
	}

	magicCookie := binary.BigEndian.Uint32(req[4:8])
	if magicCookie != 0x2112A442 {
		t.Errorf("Expected magic cookie 0x2112A442, got 0x%08x", magicCookie)
	}
}

func TestIsDiscordVoiceHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"rotterdam.discord.media", true},
		{"frankfurt123.discord.gg", true},
		{"voice.discordapp.net", true},
		{"google.com", false},
		{"roblox.com", false},
	}

	for _, tt := range tests {
		got := IsDiscordVoiceHost(tt.host)
		if got != tt.want {
			t.Errorf("IsDiscordVoiceHost(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}
