package com.hellodpi.app

import android.content.Intent
import android.net.VpnService
import android.os.Build
import android.service.quicksettings.Tile
import android.service.quicksettings.TileService
import androidx.annotation.RequiresApi

/**
 * HelloDpiTileService enables 1-tap instant toggling from Android's Quick Settings notification shade.
 */
@RequiresApi(Build.VERSION_CODES.N)
class HelloDpiTileService : TileService() {

    override fun onStartListening() {
        super.onStartListening()
        updateTileState()
    }

    override fun onClick() {
        super.onClick()

        if (HelloDpiVpnService.isVpnRunning) {
            // Stop VPN
            val intent = Intent(this, HelloDpiVpnService::class.java).apply {
                action = HelloDpiVpnService.ACTION_STOP
            }
            startService(intent)
        } else {
            // Check VPN permission before starting
            val prepareIntent = VpnService.prepare(this)
            if (prepareIntent != null) {
                // Needs user confirmation: open main activity
                val appIntent = Intent(this, MainActivity::class.java).apply {
                    addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                }
                startActivityAndCollapse(appIntent)
                return
            }

            // Start VPN directly
            val intent = Intent(this, HelloDpiVpnService::class.java).apply {
                action = HelloDpiVpnService.ACTION_START
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent)
            } else {
                startService(intent)
            }
        }

        qsTile?.state = if (HelloDpiVpnService.isVpnRunning) Tile.STATE_INACTIVE else Tile.STATE_ACTIVE
        qsTile?.updateTile()
    }

    private fun updateTileState() {
        val tile = qsTile ?: return
        val isRunning = HelloDpiVpnService.isVpnRunning

        tile.state = if (isRunning) Tile.STATE_ACTIVE else Tile.STATE_INACTIVE
        tile.label = "Hello DPI"
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            tile.subtitle = if (isRunning) "Aktif (0 ms)" else "Kapalı"
        }
        tile.updateTile()
    }
}
