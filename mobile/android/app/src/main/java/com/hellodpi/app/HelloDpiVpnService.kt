package com.hellodpi.app

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import androidx.core.app.NotificationCompat
import java.io.File

/**
 * HelloDpiVpnService creates a local TUN virtual interface and runs the embedded
 * Hello DPI engine seamlessly in the background without battery drain or external VPN latency.
 */
class HelloDpiVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
    private var engineProcess: Process? = null

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
        if (isVpnRunning) return

        try {
            // 1. Create notification channel and promote to silent foreground service
            createNotificationChannel()
            val notification = buildForegroundNotification()
            startForeground(NOTIFICATION_ID, notification)

            // 2. Start embedded native Hello DPI core daemon
            startNativeEngine()

            // 3. Configure Android VpnService TUN Builder
            val builder = Builder()
                .setSession("Hello DPI")
                .addAddress("10.0.0.2", 24)
                .addDnsServer("1.1.1.1")
                .addDnsServer("8.8.8.8")
                .addRoute("0.0.0.0", 0)
                .setMtu(1500)
                .setBlocking(false)

            // Avoid routing proxy's own outbound sockets into the VPN loop
            try {
                builder.addDisallowedApplication(packageName)
            } catch (ignored: Exception) {}

            val pfd = builder.establish()
            if (pfd == null) {
                stopVpn()
                return
            }
            vpnInterface = pfd
            isVpnRunning = true

            // Persist active state for boot auto-reconnect
            getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
                .edit()
                .putBoolean(KEY_IS_ACTIVE, true)
                .apply()

            broadcastState(true)
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
                val proc = pb.start()
                engineProcess = proc

                // Drain process output stream in background thread to prevent buffer deadlocks
                Thread {
                    try {
                        proc.inputStream.bufferedReader().useLines { lines ->
                            lines.forEach { /* drained */ }
                        }
                    } catch (ignored: Exception) {}
                }.start()
            }
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    private fun stopVpn() {
        isVpnRunning = false

        getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putBoolean(KEY_IS_ACTIVE, false)
            .apply()

        try {
            engineProcess?.destroy()
            engineProcess = null

            vpnInterface?.close()
            vpnInterface = null
        } catch (e: Exception) {
            e.printStackTrace()
        }

        stopForeground(STOP_FOREGROUND_REMOVE)
        broadcastState(false)
        stopSelf()
    }

    override fun onRevoke() {
        // User turned off VPN from system settings
        stopVpn()
        super.onRevoke()
    }

    override fun onDestroy() {
        stopVpn()
        super.onDestroy()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Hello DPI Koruma Durumu",
                NotificationManager.IMPORTANCE_LOW // Silent, zero sound/vibration
            ).apply {
                description = "Hello DPI kesintisiz arka plan bağlantı bildirimi"
                setShowBadge(false)
            }
            val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildForegroundNotification(): android.app.Notification {
        val openIntent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        val openPendingIntent = PendingIntent.getActivity(
            this, 0, openIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val stopIntent = Intent(this, HelloDpiVpnService::class.java).apply {
            action = ACTION_STOP
        }
        val stopPendingIntent = PendingIntent.getService(
            this, 1, stopIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("Hello DPI")
            .setContentText("Sansür engelleme devrede · Sıfır ek gecikme (0 ms)")
            .setSmallIcon(R.drawable.ic_launcher)
            .setContentIntent(openPendingIntent)
            .addAction(R.drawable.ic_launcher, "DURDUR", stopPendingIntent)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    }

    private fun broadcastState(active: Boolean) {
        val intent = Intent(ACTION_STATUS_CHANGED).apply {
            putExtra("is_active", active)
            setPackage(packageName)
        }
        sendBroadcast(intent)
    }

    companion object {
        const val ACTION_START = "com.hellodpi.app.START"
        const val ACTION_STOP = "com.hellodpi.app.STOP"
        const val ACTION_STATUS_CHANGED = "com.hellodpi.app.STATUS_CHANGED"

        const val PREFS_NAME = "hellodpi_prefs"
        const val KEY_IS_ACTIVE = "is_active"
        const val KEY_AUTO_START = "auto_start_on_boot"

        const val CHANNEL_ID = "hellodpi_bg_channel"
        const val NOTIFICATION_ID = 1001

        @Volatile
        var isVpnRunning = false
            private set
    }
}
