package com.hellodpi.app

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.net.VpnService
import android.os.Build

/**
 * BootReceiver automatically re-enables Hello DPI on device boot if the user previously had it running.
 */
class BootReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action == Intent.ACTION_BOOT_COMPLETED) {
            val prefs = context.getSharedPreferences(HelloDpiVpnService.PREFS_NAME, Context.MODE_PRIVATE)
            val wasActive = prefs.getBoolean(HelloDpiVpnService.KEY_IS_ACTIVE, false)
            val autoStart = prefs.getBoolean(HelloDpiVpnService.KEY_AUTO_START, true)

            if (wasActive && autoStart) {
                // Ensure VPN permission is still granted
                if (VpnService.prepare(context) == null) {
                    val vpnIntent = Intent(context, HelloDpiVpnService::class.java).apply {
                        action = HelloDpiVpnService.ACTION_START
                    }
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        context.startForegroundService(vpnIntent)
                    } else {
                        context.startService(vpnIntent)
                    }
                }
            }
        }
    }
}
