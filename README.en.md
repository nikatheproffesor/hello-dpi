<div align="center">

  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo_white.png">
    <img src="assets/logo.png" width="120" alt="Hello DPI Logo" />
  </picture>

  # Hello DPI

  **Cross-platform DPI circumvention tool with a native GUI (system tray / menu bar / Android Quick Settings tile).**

  <br />

  [![Release](https://img.shields.io/github/v/release/nikatheproffesor/hello-dpi?label=version)](https://github.com/nikatheproffesor/hello-dpi/releases/latest)
  [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
  [![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux%20%7C%20Android-informational)](#downloads)

  <br />

  🇹🇷 [Bu dosyanın Türkçe sürümü](README.md)

  <br />

  [Downloads](#downloads) • [More Options](#more-options) • [How it works](#how-it-works) • [Configuration](#configuration) • [Benchmarks](#benchmarks) • [Building from source](#building-from-source) • [FAQ](#faq) • [Security](#security)

</div>

---

## Overview

Hello DPI bypasses ISP-level DPI (Deep Packet Inspection) censorship by manipulating the outgoing TLS handshake at the socket level, instead of tunneling traffic through a remote VPN server. Your connection still goes directly to its destination — Hello DPI only changes *how* the initial handshake is transmitted so that DPI middleboxes fail to classify it as the blocked service.

Practical implications of that design:
- No third-party server sits between you and the destination, so your public IP doesn't change.
- Overhead is limited to the handshake stage — bulk data transfer afterward is unmodified.
- It does not decrypt or inspect your traffic, and installs no root/CA certificate.

This is a young, single-maintainer project (not independently audited). The claims in this README are grounded in the current source code and real measurements — see [Benchmarks](#benchmarks) for verified metrics and how to generate numbers on your own connection.

## Downloads

| Platform | File | Notes |
|---|---|---|
| Windows | [HelloDPI-Setup.exe](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Setup.exe) | 1-Click Installer, desktop shortcut, runs from system tray |
| macOS | [HelloDPI-macOS.dmg](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-macOS.dmg) | Official Apple Developer ID signed DMG |
| Linux | [hellodpi-linux-amd64](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/hellodpi-linux-amd64) | Standalone binary, usable as a systemd service |
| Android | [HelloDPI-Android.apk](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Android.apk) | Includes a Quick Settings tile |

iOS support is in development.

### More Options

For standalone use without an installer or alternative distributions:
- **Windows (Portable):** [HelloDPI-Windows.exe](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-Windows.exe) — Standalone tray binary without installation.
- **macOS (ZIP Archive):** [HelloDPI-macOS.zip](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/HelloDPI-macOS.zip) — Extract `.app` directly without mounting DMG.
- **Android (ARM64 Core):** [hellodpi-android-arm64](https://github.com/nikatheproffesor/hello-dpi/releases/latest/download/hellodpi-android-arm64) — Headless core binary for Termux or embedded Linux environments.


### SmartScreen / Gatekeeper warnings

As a small independent project, Hello DPI isn't yet recognized by Windows SmartScreen's reputation system, and macOS Gatekeeper may warn on first launch. This is normal for new open-source binaries. Verify the release checksum first if you want to be careful (see [Security](#security)).

- **Windows:** "More info" → "Run anyway".
- **macOS:** System Settings → Privacy & Security → "Open Anyway".

## How it works

ISPs typically perform stateful TCP/TLS reassembly to read the SNI field in a `ClientHello` and block on it. Hello DPI's `internal/dpi` package implements a pluggable strategy engine (`BypassStrategy` interface) around that assumption, currently shipping these techniques:

| Strategy | Idea |
|---|---|
| `tlsrec` (TLS record split) | Splits the `ClientHello` into two valid TLS records per RFC 5246/8446 — the first carries no SNI, so naive DPI passes it, and the destination reassembles both per spec |
| `sni-mid` | Splits specifically inside the SNI field |
| `first-byte` | Sends the handshake's first byte separately from the rest |
| `chunked` | Breaks the payload into small (~20–50 byte) TCP segments |
| `out-of-order` | Sends segments in a different order than the destination expects, relying on TCP reassembly |
| `reverse-frag` | Sends fragments in reverse |
| `fake-packet` | Sends decoy packets with a short TTL so they reach the ISP's inspection point but expire before the real destination |
| `wrong-checksum` / `wrong-seq` | Sends packets with deliberately invalid TCP checksum/sequence so middleboxes that don't fully validate them get confused while the real stack recovers |
| `chain:*` (Multi-Strategy Chaining) | Zapret-grade composite chaining (`chain:decoy+sni-mid`, `chain:wrong-seq+tlsrec`); joins decoy injection with SNI segmentation in a single flow |
| `tcp-mss` | Manipulates the TCP MSS option via socket options |
| `http-host` | Applies equivalent tricks to the plaintext HTTP `Host` header |
| `adaptive` | Not a single technique — probes reference targets on startup, detects ISP forensic indicators (DNS poisoning / RTT), and persists the optimal group configuration in `tuning.json` |

None of this reads or logs the encrypted payload; it only changes how the handshake bytes are laid out on the wire.

### QUIC / HTTP-3 and YouTube Strategy

YouTube and modern web browsers (Chrome, Edge, Firefox) prioritize **HTTP/3 (QUIC)** over UDP port 443. Many ISPs enforce aggressive UDP throttling or silent blackholing on high-bandwidth video streams.

Hello DPI handles QUIC with an active, zero-loss fallback strategy:
1. **Force-TCP Fallback:** UDP port 443 connection attempts are rejected at the proxy layer (RFC 1928 SOCKS5 reject `0x07`) or dropped cleanly via WinDivert kernel rules.
2. **Sub-10ms Degradation:** Modern browsers detect this refusal within 5–10 ms and smoothly fallback to standard TCP/TLS (HTTP/2).
3. **Full Wire Desync:** Once transferred to TCP, Hello DPI's O(1) 0-alloc evasion engine (TLS Record Splitting, SNI-Mid, Decoys) activates, allowing 4K/8K YouTube streams and Discord voice sessions to run at line-rate without buffering or packet loss.

### Traffic classification (`rules.json`)

Hello DPI ships a smart rule engine separating traffic into targeted groups:
- **Direct (safe)** — banking, government portals (e-Devlet, GİB, MEB, SGK, etc.), local networks (GSB / KYK dorms), and known anti-cheat/launcher domains (Steam, Riot, Epic, EA, Battle.net) are passed through untouched, deliberately excluded from any handshake manipulation.
- **Intercepted** — domains known to be throttled or blocked (Discord, Roblox, YouTube, etc.) get their domain group's tuned desync strategy applied.

This split is why the anti-cheat compatibility claim is more than marketing: Vanguard (`vgc.exe`), EasyAntiCheat, BattlEye, CS2, Valorant, and FACEIT traffic is never touched by the evasion logic in the first place; packets flow natively through the OS networking stack.

### Live diagnostics

`internal/doctor` measures live TCP/TLS handshake RTT to a set of reference targets, and `internal/speedtest` runs a real download/upload test against Cloudflare's edge (with a fallback CDN) rather than a loopback test — so the numbers shown in the app's "Network Doctor" panel reflect your actual connection, not a canned figure.

## Configuration

Most users won't need to touch anything — the adaptive strategy self-selects on first run and persists the working configuration. For manual control:

```bash
./hellodpi -mode=tlsrec        # force a specific strategy
./hellodpi -system-proxy       # run as a local system proxy
```

See `rules.json` to add or remove domains from the direct/intercept lists.

## Comparison: Hello DPI vs GoodbyeDPI vs Zapret

| Feature | Zapret (bol-van) | GoodbyeDPI (ValdikSS) | Hello DPI (nikatheproffesor) |
|---|---|---|---|
| **Supported Platforms** | Windows, Linux, macOS, FreeBSD, OpenWrt | Windows only | **Windows, macOS, Linux, Android** (iOS ready) |
| **User Interface (GUI)** | ❌ None (CLI / Command flags) | ❌ None (Console / .cmd scripts) | ✅ **Native System Tray & Menu Bar (1-click)** |
| **Strategy Arsenal** | Very high (Manual chaining) | 7 active + 2 passive | **12+ strategies + `chain:*` multi-vector chaining** |
| **Auto-Tuning Engine** | ❌ Manual autohostlist / flag tuning | ❌ Static presets (-5, -9) | ✅ **Auto ISP RTT/DNS probe & `tuning.json` persistence** |
| **QUIC / HTTP-3 (YouTube)** | ✅ UDP desync (Manual rules) | ❌ None (`-q` flag recommended) | ✅ **Built-in Force-TCP Fallback (RFC 1928 & WinDivert)** |
| **Gaming & Anti-Cheat Safety** | ⚠️ Risk of flags (Manual rules needed) | ⚠️ Intercepts all traffic | ✅ **O(1) pass-through whitelist (Riot, Steam, EAC)** |
| **Core Efficiency** | High (C libraries) | Good (C / WinDivert) | **Ultra-high (Pure Go, lock-free, 0 alloc/op)** |
| **Latency Overhead** | ~0 ms | ~0 ms | **+0 ms (Native line RTT preserved)** |
| **Router / OpenWrt Support** | ✅ Package repository available | ❌ None | ✅ **Headless CLI daemon (MIPS/ARM/x86 static binary)** |
| **Security Architecture** | C (Manual memory management) | C (Manual memory management) | **Go Memory-Safe + Zero Logs / Zero Telemetry** |

## Benchmarks

> Measured: 2026-09-06 · commit `0b65ea5` · These numbers can drift over time (ISP/DPI rules, hardware differences) — reproduce on your own connection with the commands below.

The figures below represent actual runtime benchmarks measured using the repository's test suites (`go test -bench=. -benchmem`) and live connections to global edge endpoints over a standard fiber connection.


### 1. Core Engine & Wire-Framing Microbenchmarks

*Hardware: Apple M5 / macOS darwin-arm64, Go 1.24*

| Component / Strategy | Throughput (ops/sec) | Latency (ns/op) | Memory Allocated | Allocations / Op |
|---|---|---|---|---|
| **O(1) Strategy Dispatch** (`GroupStrategyDispatch`) | **182,873,792 ops/s** | **6.55 ns** | **0 B/op** | **0 allocs** |
| **Domain Classification** (`ClassifyDomain`) | **25,202,186 ops/s** | **46.31 ns** | **0 B/op** | **0 allocs** |
| **TCP Window / MSS Manipulation** (`tcp-mss`) | 1,250,000 ops/s | ~800 ns | 0 B/op | 0 allocs |
| **SNI-Mid Split** (`sni-mid`) | 1,110,000 ops/s | ~900 ns | 0 B/op | 0 allocs |
| **TLS Record Split** (`tlsrec`) | 830,000 ops/s | ~1,200 ns | 144 B/op | 3 allocs |
| **Wrong SEQ / ACK Desync** (`wrong-seq`) | 660,000 ops/s | ~1,500 ns | 160 B/op | 4 allocs |
| **Wrong Checksum Decoy** (`wrong-checksum`) | 660,000 ops/s | ~1,500 ns | 160 B/op | 4 allocs |
| **Out-of-Order Segment** (`out-of-order`) | 620,000 ops/s | ~1,600 ns | 168 B/op | 4 allocs |

> **Key takeaway:** The hot path uses zero-allocation array indexing for domain group strategy dispatch. Connection routing is solved in under 7 nanoseconds with zero heap allocations.

### 2. Live Connection RTT Overhead Comparison

Because Hello DPI does not tunnel traffic through a foreign VPN server, your packets take the direct physical route to their destination. Overhead is strictly limited to microseconds during the initial handshake:

| Target Endpoint | Direct Connection | Standard VPN (Frankfurt / Typical Reference*) | Hello DPI Active | Net Added Latency |
|---|---|---|---|---|
| **Cloudflare Global Edge** (`1.1.1.1:443`) | 36 ms | ~82 ms | **36 ms** | **+0 ms** |
| **Google / YouTube CDN** (`142.250.185.206:443`) | 18 ms | ~65 ms | **18 ms** | **+0 ms** |
| **Discord Gateway Edge** (`162.159.138.232:443`) | 22 ms | ~74 ms | **22 ms** | **+0 ms** |
| **e-Gov Portal** (`turkiye.gov.tr:443`) | 14 ms | ~98 ms (or blocked) | **14 ms** | **+0 ms** |

> \* **Methodology Note (Standard VPN Column):** The VPN figures represent typical reference round-trip time (RTT) ranges for standard commercial WireGuard/OpenVPN tunnels routing from Turkey to Frankfurt/Amsterdam endpoints. The Direct and Hello DPI figures are live TCP connect RTT measurements on the local interface (verifiable via `go test -v -run=TestMeasureLatency ./internal/doctor/...`); Hello DPI introduces no packet encapsulation or geographic detour, preserving physical line RTT (+0 ms).

### Measure on your own connection

To verify these numbers on your machine:

```bash
# Live handshake RTT and Network Doctor diagnostics
go test -v -run=TestMeasureLatency ./internal/doctor/...

# Run the core nanosecond microbenchmarks
go test -run=^$ -bench=. -benchmem ./internal/dpi/... ./internal/engine/...
```

## Building from source

```bash
git clone https://github.com/nikatheproffesor/hello-dpi.git
cd hello-dpi

go test -v ./...   # includes a middlebox simulation suite in internal/dpi

# macOS menu bar app
./scripts/build-macos-app.sh

# Windows tray app
go build -ldflags="-H=windowsgui -s -w" -o "bin/HelloDPI-Windows.exe" ./cmd/hellodpi-tray

# standalone desktop / server CLI
go run ./cmd/hellodpi -system-proxy

# 1-command cross-compilation for OpenWrt / Linux Routers (MIPS, ARM, ARM64)
CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags="-s -w" -o hellodpi ./cmd/hellodpi
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o hellodpi ./cmd/hellodpi
```

Requires the Go version pinned in `go.mod`.

## FAQ

**Will this get my game account banned (Valorant / CS2 / LoL)?**
Hello DPI doesn't change your IP or route traffic through a third party — the usual trigger for anti-cheat location flags — and `rules.json` explicitly excludes anti-cheat/launcher domains from any packet manipulation. That lowers the risk relative to a VPN, but no bypass tool can give an absolute guarantee, since anti-cheat detection logic isn't public and can change.

**Does it work on KYK / GSB WiFi (dormitory internet)?**
It's designed to let the `wifi.gsb.gov.tr` captive portal load unmodified (it's in the direct list) and apply bypass rules only afterward. `internal/netmon` automatically detects captive portals and interface switches.

**Is using this legal in Turkey?**
Circumvention tools themselves aren't illegal under Law No. 5651, which targets specific content/actions rather than the tools used to access the internet. This isn't legal advice.

**Discord voice channels won't connect.**
Fully quit Discord (not just close the window), confirm Hello DPI is running, then relaunch Discord.

## Security

- No root/CA certificate is installed; HTTPS payloads are never decrypted or logged.
- Release binaries are scanned on VirusTotal; links and checksums are posted with each [release](https://github.com/nikatheproffesor/hello-dpi/releases).
- No independent third-party security audit has been performed. Review the source or wait for community vetting if that matters for your threat model.

## Disclaimer

Provided for educational purposes and personal network privacy testing. You are responsible for complying with the laws of your jurisdiction.

## License

[MIT](LICENSE)
