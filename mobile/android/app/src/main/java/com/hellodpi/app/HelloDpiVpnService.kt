package com.hellodpi.app

import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import java.io.File

/**
 * HelloDpiVpnService creates a local TUN virtual interface and routes TCP/UDP traffic
 * through the embedded native Hello DPI engine without requiring root access.
 */
class HelloDpiVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
    private var engineProcess: Process? = null
    private var isRunning = false

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val action = intent?.action
        if (action == ACTION_STOP) {
            stopVpn()
            return START_NOT_STICKY
        }

        startVpn()
        return START_STICKY
    }

    private fun startVpn() {
        if (isRunning) return

        try {
            // 1. Start embedded native Hello DPI core daemon
            startNativeEngine()

            // 2. Configure Android VpnService TUN Builder
            val builder = Builder()
                .setSession("Hello DPI")
                .addAddress("10.0.0.2", 24)
                .addDnsServer("1.1.1.1")
                .addDnsServer("8.8.8.8")
                .addRoute("0.0.0.0", 0)
                .setMtu(1500)
                .setBlocking(false)

            // Avoid routing proxy's own outbound traffic back into VPN
            try {
                builder.addDisallowedApplication(packageName)
            } catch (ignored: Exception) {}

            vpnInterface = builder.establish()
            isRunning = true
        } catch (e: Exception) {
            e.printStackTrace()
            stopVpn()
        }
    }

    private fun startNativeEngine() {
        try {
            val libDir = applicationInfo.nativeLibraryDir
            val nativeBin = File(libDir, "libhellodpi.so")

            if (nativeBin.exists()) {
                val pb = ProcessBuilder(
                    nativeBin.absolutePath,
                    "-addr", "127.0.0.1:8080"
                )
                pb.redirectErrorStream(true)
                engineProcess = pb.start()
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    private fun stopVpn() {
        isRunning = false
        try {
            engineProcess?.destroy()
            engineProcess = null

            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Exception) {
            e.printStackTrace()
        }
        stopSelf()
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }

    companion object {
        const val ACTION_START = "com.hellodpi.app.START"
        const val ACTION_STOP = "com.hellodpi.app.STOP"
    }
}
