package mesh

import (
	"bytes"
	"testing"
	"time"
)

func TestMeshNodeCreationAndID(t *testing.T) {
	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	if len(node.NodeID()) != 16 {
		t.Errorf("expected 16-char hex node ID, got %s", node.NodeID())
	}
}

func TestMeshPeerRouting(t *testing.T) {
	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	peer1 := &PeerNode{
		ID:        "peer-01",
		Addr:      "192.168.1.50:9090",
		LatencyMs: 45,
		LastSeen:  time.Now(),
	}
	peer2 := &PeerNode{
		ID:        "peer-02",
		Addr:      "192.168.1.60:9090",
		LatencyMs: 15, // Faster
		LastSeen:  time.Now(),
	}

	node.AddPeer(peer1)
	node.AddPeer(peer2)

	peers := node.GetActivePeers()
	if len(peers) != 2 {
		t.Fatalf("expected 2 active peers, got %d", len(peers))
	}

	best, err := node.SelectBestPeer()
	if err != nil {
		t.Fatalf("failed to select best peer: %v", err)
	}
	if best.ID != "peer-02" {
		t.Errorf("expected peer-02 (15ms), got %s (%dms)", best.ID, best.LatencyMs)
	}
}

func TestMeshPayloadEncryptionRoundtrip(t *testing.T) {
	node, err := NewNode(0)
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	raw := []byte("secret-emergency-packet-payload-12345")
	encrypted, err := node.EncryptPayload(raw)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if bytes.Equal(raw, encrypted) {
		t.Fatalf("encrypted data must not match plaintext")
	}

	decrypted, err := node.DecryptPayload(encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if !bytes.Equal(raw, decrypted) {
		t.Errorf("decrypted data %s does not match original %s", decrypted, raw)
	}
}
