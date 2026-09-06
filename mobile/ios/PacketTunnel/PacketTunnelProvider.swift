import NetworkExtension

class PacketTunnelProvider: NEPacketTunnelProvider {

    override func startTunnel(options: [String : NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        // 1. Configure local virtual TUN network settings
        let tunnelNetworkSettings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        
        // 2. IPv4 virtual address
        let ipv4Settings = NEIPv4Settings(addresses: ["10.0.0.2"], subnetMasks: ["255.255.255.0"])
        ipv4Settings.includedRoutes = [NEIPv4Route.default()]
        tunnelNetworkSettings.ipv4Settings = ipv4Settings
        
        // 3. DNS-over-HTTPS fallback (Cloudflare & Google)
        let dnsSettings = NEDNSSettings(servers: ["1.1.1.1", "8.8.8.8"])
        dnsSettings.matchDomains = [""] // Catch all DNS queries
        tunnelNetworkSettings.dnsSettings = dnsSettings
        
        tunnelNetworkSettings.mtu = 1500

        // 4. Apply settings and start packet loop
        setTunnelNetworkSettings(tunnelNetworkSettings) { error in
            if let error = error {
                completionHandler(error)
                return
            }
            
            // Start reading packets from the virtual TUN interface
            self.readPacketLoop()
            completionHandler(nil)
        }
    }

    private func readPacketLoop() {
        packetFlow.readPackets { [weak self] packets, protocols in
            guard let self = self else { return }
            
            // Forward raw IP packets to Hello DPI engine (or echo back)
            // In full integration, packets are passed to C-Go hellocore
            self.packetFlow.writePackets(packets, withProtocols: protocols)
            
            // Continue reading next batch
            self.readPacketLoop()
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        completionHandler()
    }
}
