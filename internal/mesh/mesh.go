package mesh

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

var (
	ErrNoPeerAvailable   = errors.New("no peer available in HelloMesh")
	ErrPeerUnreachable   = errors.New("peer is unreachable")
	ErrDecryptionFailure = errors.New("mesh payload decryption failed")
)

// PeerNode represents a discovered Hello DPI peer node
type PeerNode struct {
	ID        string    `json:"id"`
	Addr      string    `json:"addr"`
	LastSeen  time.Time `json:"last_seen"`
	LatencyMs int64     `json:"latency_ms"`
	IsRelay   bool      `json:"is_relay"`
}

// Node operates a decentralized, zero-server P2P micro-relay
type Node struct {
	mu         sync.RWMutex
	privKey    ed25519.PrivateKey
	pubKey     ed25519.PublicKey
	nodeID     string
	peers      map[string]*PeerNode
	listenPort int
	sharedKey  []byte
}

// NewNode initializes a new HelloMesh P2P node
func NewNode(listenPort int) (*Node, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256(pub)
	nodeID := hex.EncodeToString(h[:8])

	// Derive per-node encryption key from the private key material (unique per instance)
	keyMaterial := sha256.Sum256(priv.Seed())

	node := &Node{
		privKey:    priv,
		pubKey:     pub,
		nodeID:     nodeID,
		peers:      make(map[string]*PeerNode),
		listenPort: listenPort,
		sharedKey:  keyMaterial[:16], // AES-128 key
	}

	return node, nil
}

// NodeID returns the hex-encoded identity of this node
func (n *Node) NodeID() string {
	return n.nodeID
}

// AddPeer registers or updates a peer in the mesh routing table
func (n *Node) AddPeer(peer *PeerNode) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.peers[peer.ID] = peer
}

// GetActivePeers returns a slice of currently known active peers
func (n *Node) GetActivePeers() []*PeerNode {
	n.mu.RLock()
	defer n.mu.RUnlock()
	var list []*PeerNode
	for _, p := range n.peers {
		list = append(list, p)
	}
	return list
}

// SelectBestPeer picks the lowest latency active peer
func (n *Node) SelectBestPeer() (*PeerNode, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var best *PeerNode
	for _, p := range n.peers {
		if best == nil || p.LatencyMs < best.LatencyMs {
			best = p
		}
	}
	if best == nil {
		return nil, ErrNoPeerAvailable
	}
	return best, nil
}

// EncryptPayload wraps data with AES-GCM and a random 12-byte nonce
func (n *Node) EncryptPayload(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(n.sharedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptPayload decrypts data using AES-GCM and the embedded nonce
func (n *Node) DecryptPayload(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(n.sharedKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, ErrDecryptionFailure
	}
	nonce := ciphertext[:gcm.NonceSize()]
	actualCipher := ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, actualCipher, nil)
	if err != nil {
		return nil, ErrDecryptionFailure
	}
	return plaintext, nil
}

// RelayConnection forwards an emergency stream through an unblocked peer
func (n *Node) RelayConnection(peer *PeerNode, targetAddr string, clientConn net.Conn) error {
	peerConn, err := net.DialTimeout("tcp", peer.Addr, 3*time.Second)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPeerUnreachable, err)
	}
	defer peerConn.Close()

	log.Printf("[HelloMesh] Emergency P2P Hop: Routing %s through peer node %s (%s)",
		targetAddr, peer.ID, peer.Addr)

	// Bidirectional tunnel between client and mesh relay peer
	errCh := make(chan error, 2)
	go func() {
		_, err := io.Copy(peerConn, clientConn)
		errCh <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, peerConn)
		errCh <- err
	}()

	<-errCh
	return nil
}
