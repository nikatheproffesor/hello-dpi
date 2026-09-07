package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hellodpi/hellodpi/internal/doh"
	"github.com/hellodpi/hellodpi/internal/dpi"
	"github.com/hellodpi/hellodpi/internal/mesh"
	"github.com/hellodpi/hellodpi/internal/proxy"
	"github.com/hellodpi/hellodpi/internal/sysproxy"
	"github.com/hellodpi/hellodpi/internal/tun"
	"github.com/hellodpi/hellodpi/internal/version"
)

func printBanner() {
	banner := `
  _    _      _ _         _____  _____ _____ 
 | |  | |    | | |       |  __ \|  __ \_   _|
 | |__| | ___| | | ___   | |  | | |__) || |  
 |  __  |/ _ \ | |/ _ \  | |  | |  ___/ | |  
 | |  | |  __/ | | (_) | | |__| | |    _| |_ 
 |_|  |_|\___|_|_|\___/  |_____/|_|   |_____|
  Cross-Platform Zero-Overhead DPI Bypass (v%s)
`
	fmt.Printf(banner, version.Version)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("  [DIRECT]   Direct Connection (Zero VPN speed loss or ping penalty)")
	fmt.Println("  [FRAGMENT] TLS SNI & Record Layer DPI Evasion")
	fmt.Println("  [QUIC/UDP] RFC 9000 Initial Mangling & Discord Voice Shield")
	fmt.Println("  [ECH/JA4]  Encrypted Client Hello & Chrome/Safari JA4 Masquerade")
	fmt.Println("  [TUN/ZERO] Transparent Driver Mode (Zero System Proxy Needed)")
	fmt.Println("  [AI/MUT]   Self-Healing Dynamic Mutation Heuristics Engine")
	fmt.Println("  [MESH]     HelloMesh Serverless Decentralized P2P Fallback")
	fmt.Println("---------------------------------------------------------")
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "Listen address for Hello DPI local proxy")
	mode := flag.String("mode", "sni", "Bypass strategy: 'adaptive', 'sni', 'tlsrec', 'decoy', 'chunked', 'first-byte', 'wrong-seq', 'wrong-chk', 'out-of-order', 'chain:<s1>+<s2>'")
	delay := flag.Int("delay", 2, "Delay between packet fragments in milliseconds")
	enableDoH := flag.Bool("doh", true, "Enable DNS-over-HTTPS resolution")
	dohServer := flag.String("doh-server", string(doh.Cloudflare), "DoH resolver URL (e.g. Cloudflare, Google, Quad9)")
	autoSysProxy := flag.Bool("system-proxy", false, "Automatically configure and toggle OS system proxy")
	enableTUN := flag.Bool("tun", false, "Enable transparent TUN driver mode (Zero system proxy needed)")
	enableMesh := flag.Bool("mesh", false, "Enable HelloMesh decentralized serverless P2P emergency fallback")
	resetNetwork := flag.Bool("reset-network", false, "Emergency reset of all system proxy and network configurations")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *resetNetwork {
		fmt.Println("[Hello DPI] Performing emergency network reset...")
		_ = sysproxy.ClearSystemProxy()
		fmt.Println("✓ System proxy and network settings have been cleanly restored to factory direct defaults.")
		return
	}

	if *showVersion {
		fmt.Printf("Hello DPI v%s\n", version.Version)
		return
	}

	printBanner()

	// Parse host and port
	host, portStr, err := net.SplitHostPort(*addr)
	if err != nil {
		log.Fatalf("Invalid listen address %s: %v", *addr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port: %v", err)
	}

	// Initialize proxy server
	cfg := proxy.Config{
		Addr:        *addr,
		SplitMode:   dpi.SplitMode(*mode),
		DelayMs:     *delay,
		DoHEndpoint: *dohServer,
		EnableDoH:   *enableDoH,
	}
	server := proxy.NewServer(cfg)

	sysproxy.RegisterExitCleanup()
	defer sysproxy.RecoverAndClear()

	// Set system proxy if requested
	if *autoSysProxy {
		log.Println("[Hello DPI] Activating system-wide proxy settings...")
		if err := sysproxy.SetSystemProxy(host, port); err != nil {
			log.Printf("[Hello DPI] Warning: Failed to set system proxy: %v", err)
		}
	}

	// Transparent TUN Mode
	var tunEng *tun.Engine
	if *enableTUN {
		tunDev, err := tun.OpenDevice("hellotun0")
		if err == nil {
			tunEng = tun.NewEngine(tunDev, server.Orchestrator.Strategy)
			_ = tunEng.Start()
		} else {
			log.Printf("[Hello DPI TUN] Warning: could not initialize TUN: %v", err)
		}
	}

	// HelloMesh Decentralized Fallback
	var meshNode *mesh.Node
	if *enableMesh {
		mNode, err := mesh.NewNode(0)
		if err == nil {
			meshNode = mNode
			log.Printf("[HelloMesh] P2P Node active (ID: %s)", meshNode.NodeID())
		}
	}

	// Setup graceful shutdown listener
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stopChan
		fmt.Println("\n[Hello DPI] Shutting down...")
		if tunEng != nil {
			_ = tunEng.Stop()
		}
		if *autoSysProxy {
			log.Println("[Hello DPI] Restoring system proxy settings...")
			_ = sysproxy.ClearSystemProxy()
		}
		_ = server.Close()
		os.Exit(0)
	}()

	fmt.Printf("\n[OK] Hello DPI is running!\n")
	fmt.Printf("    • Local Proxy Address : %s\n", *addr)
	fmt.Printf("    • Fragmentation Mode  : %s\n", *mode)
	fmt.Printf("    • DNS-over-HTTPS      : %v (%s)\n", *enableDoH, *dohServer)
	if *enableTUN {
		fmt.Printf("    • Transparent Driver  : Active (Zero-Proxy Mode / L3 Packet Routing)\n")
	} else if *autoSysProxy {
		fmt.Printf("    • System Proxy Mode   : Active (All browsers automatically routed)\n")
	} else {
		fmt.Printf("    • System Proxy Mode   : Manual (Point browser HTTP/SOCKS5 proxy to %s)\n", *addr)
	}
	if meshNode != nil {
		fmt.Printf("    • HelloMesh P2P       : Active (Node ID: %s)\n", meshNode.NodeID())
	}
	fmt.Printf("\nPress Ctrl+C to stop.\n\n")

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
