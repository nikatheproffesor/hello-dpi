package com.hellodpi.app

import android.content.Intent
import android.net.VpnService
import android.os.ParcelFileDescriptor
import java.io.FileInputStream
import java.io.FileOutputStream
import java.nio.ByteBuffer

/**
 * HelloDpiVpnService creates a local TUN virtual interface and routes TCP/UDP traffic
 * through the embedded Go hellocore engine without requiring root access.
 */
class HelloDpiVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
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
            // Configure Android VpnService TUN Builder
            val builder = Builder()
                .setSession("Hello DPI")
                .addAddress("10.0.0.2", 24)
                .addDnsServer("1.1.1.1")
                .addDnsServer("8.8.8.8")
                .addRoute("0.0.0.0", 0)
                .setMtu(1500)
                .setBlocking(false)

            vpnInterface = builder.establish()
            isRunning = true

            // Notify native Go Mobile engine
            // hellocore.Hellocore.startMobileDefault(8080)
        } catch (e: Exception) {
            e.printStackTrace()
            stopVpn()
        }
    }

    private fun stopVpn() {
        isRunning = false
        try {
            vpnInterface?.close()
            vpnInterface = null
            // hellocore.Hellocore.stopMobileDefault()
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
