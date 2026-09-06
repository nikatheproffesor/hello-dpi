import Foundation
import NetworkExtension

class VPNManager: ObservableObject {
    @Published var isConnected: Bool = false
    private var providerManager: NETunnelProviderManager?

    init() {
        loadManager()
    }

    func loadManager() {
        NETunnelProviderManager.loadAllFromPreferences { [weak self] managers, error in
            guard let self = self else { return }
            if let manager = managers?.first {
                self.providerManager = manager
            } else {
                let manager = NETunnelProviderManager()
                let config = NETunnelProviderProtocol()
                config.providerBundleIdentifier = "com.hellodpi.app.PacketTunnel"
                config.serverAddress = "127.0.0.1"
                manager.protocolConfiguration = config
                manager.localizedDescription = "Hello DPI"
                manager.isEnabled = true
                self.providerManager = manager
            }
            
            self.updateStatus()
            
            NotificationCenter.default.addObserver(
                self,
                selector: #selector(self.statusDidChange),
                name: .NEVPNStatusDidChange,
                object: self.providerManager?.connection
            )
        }
    }

    @objc private func statusDidChange() {
        updateStatus()
    }

    private func updateStatus() {
        DispatchQueue.main.async {
            self.isConnected = (self.providerManager?.connection.status == .connected)
        }
    }

    func toggleConnection() {
        guard let manager = providerManager else { return }

        if isConnected {
            manager.connection.stopVPNTunnel()
        } else {
            manager.saveToPreferences { error in
                if error == nil {
                    manager.loadFromPreferences { _ in
                        try? manager.connection.startVPNTunnel()
                    }
                }
            }
        }
    }
}
