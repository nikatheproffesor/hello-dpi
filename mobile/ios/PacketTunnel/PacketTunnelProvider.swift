import NetworkExtension
import HelloCore

class PacketTunnelProvider: NEPacketTunnelProvider {

    override func startTunnel(options: [String : NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        // 1. Start the Hello DPI Core Proxy Engine on localhost:18080
        let status = HelloCoreStart(18080)
        guard status == 0 else {
            completionHandler(NSError(
                domain: "com.hellodpi.app",
                code: 1,
                userInfo: [NSLocalizedDescriptionKey: "Hello DPI core engine failed to start"]
            ))
            return
        }

        // 2. Configure local virtual TUN network settings
        let tunnelNetworkSettings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        
        // Virtual IPv4 loopback routing
        let ipv4Settings = NEIPv4Settings(addresses: ["10.0.0.2"], subnetMasks: ["255.255.255.0"])
        ipv4Settings.includedRoutes = [NEIPv4Route.default()]
        tunnelNetworkSettings.ipv4Settings = ipv4Settings
        
        // Encrypted DNS fallback (Cloudflare & Google DoH)
        let dnsSettings = NEDNSSettings(servers: ["1.1.1.1", "8.8.8.8"])
        dnsSettings.matchDomains = [""]
        tunnelNetworkSettings.dnsSettings = dnsSettings
        
        // Direct all device HTTP & HTTPS traffic into the Hello DPI engine
        let proxySettings = NEProxySettings()
        proxySettings.httpServer = NEProxyServer(address: "127.0.0.1", port: 18080)
        proxySettings.httpsServer = NEProxyServer(address: "127.0.0.1", port: 18080)
        proxySettings.httpEnabled = true
        proxySettings.httpsEnabled = true
        proxySettings.exceptionList = [
            "localhost",
            "127.0.0.1",
            "captive.apple.com",
            "wifi.gsb.gov.tr" // KYK / GSB Captive portal direct bypass
        ]
        tunnelNetworkSettings.proxySettings = proxySettings
        tunnelNetworkSettings.mtu = 1500

        // 3. Apply settings and complete tunnel initialization
        setTunnelNetworkSettings(tunnelNetworkSettings) { error in
            if let error = error {
                _ = HelloCoreStop()
                completionHandler(error)
                return
            }
            completionHandler(nil)
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        // Stop background Hello DPI proxy engine
        _ = HelloCoreStop()
        completionHandler()
    }
}
