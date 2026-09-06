import SwiftUI

struct ContentView: View {
    @StateObject private var vpn = VPNManager()

    var body: some View {
        ZStack {
            Color.black.ignoresSafeArea()

            VStack(spacing: 32) {
                // Header
                VStack(spacing: 8) {
                    Text("HELLO DPI")
                        .font(.system(size: 28, weight: .bold, design: .monospaced))
                        .foregroundColor(.white)
                        .tracking(2)

                    Text("0 ms Ek Gecikme | Mobil DPI & Sansur Atlama")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundColor(Color(white: 0.65))
                }
                .padding(.top, 40)

                Spacer()

                // Status Indicator
                VStack(spacing: 12) {
                    Circle()
                        .fill(vpn.isConnected ? Color.white : Color(white: 0.2))
                        .frame(width: 80, height: 80)
                        .overlay(
                            Circle()
                                .stroke(Color(white: 0.4), lineWidth: 1)
                        )

                    Text(vpn.isConnected ? "TUNEL AKTIF (0 MS)" : "BAGLANTI KAPALI")
                        .font(.system(size: 14, weight: .semibold, design: .monospaced))
                        .foregroundColor(vpn.isConnected ? .white : Color(white: 0.5))
                }

                Spacer()

                // Connect / Disconnect Button
                Button(action: {
                    vpn.toggleConnection()
                }) {
                    Text(vpn.isConnected ? "BAGLANTIYI KES" : "BAGLAN")
                        .font(.system(size: 15, weight: .bold, design: .monospaced))
                        .foregroundColor(vpn.isConnected ? .white : .black)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 18)
                        .background(vpn.isConnected ? Color(white: 0.15) : Color.white)
                        .cornerRadius(12)
                        .overlay(
                            RoundedRectangle(cornerRadius: 12)
                                .stroke(Color(white: 0.3), lineWidth: vpn.isConnected ? 1 : 0)
                        )
                }
                .padding(.horizontal, 32)
                .padding(.bottom, 48)
            }
        }
    }
}
