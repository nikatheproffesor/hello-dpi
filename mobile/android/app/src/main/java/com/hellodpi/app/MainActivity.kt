package com.hellodpi.app

import android.app.Activity
import android.content.Intent
import android.net.VpnService
import android.os.Bundle
import android.widget.Button
import android.widget.TextView

/**
 * Minimalist, monochrome activity for Hello DPI Android client.
 */
class MainActivity : Activity() {

    private val VPN_REQUEST_CODE = 0x4844 // "HD"

    private lateinit var btnToggle: Button
    private lateinit var txtStatus: TextView
    private var isConnected = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Clean programmatic monochrome layout (zero XML dependencies required)
        val layout = android.widget.LinearLayout(this).apply {
            orientation = android.widget.LinearLayout.VERTICAL
            setBackgroundColor(0xFF000000.toInt())
            gravity = android.view.Gravity.CENTER
            setPadding(48, 48, 48, 48)
        }

        val title = TextView(this).apply {
            text = "HELLO DPI"
            textSize = 24f
            setTextColor(0xFFFFFFFF.toInt())
            typeface = android.graphics.Typeface.DEFAULT_BOLD
            gravity = android.view.Gravity.CENTER
        }

        val subtitle = TextView(this).apply {
            text = "0 ms Ek Gecikme | DPI & Sansur Atlama"
            textSize = 12f
            setTextColor(0xFFA1A1AA.toInt())
            gravity = android.view.Gravity.CENTER
            setPadding(0, 8, 0, 48)
        }

        txtStatus = TextView(this).apply {
            text = "BAGLANTI KAPALI"
            textSize = 14f
            setTextColor(0xFF71717A.toInt())
            gravity = android.view.Gravity.CENTER
            setPadding(0, 0, 0, 32)
        }

        btnToggle = Button(this).apply {
            text = "BAGLAN"
            setTextColor(0xFF000000.toInt())
            setBackgroundColor(0xFFFFFFFF.toInt())
            textSize = 14f
            setPadding(32, 16, 32, 16)
            setOnClickListener {
                if (isConnected) {
                    disconnectVpn()
                } else {
                    prepareAndConnectVpn()
                }
            }
        }

        layout.addView(title)
        layout.addView(subtitle)
        layout.addView(txtStatus)
        layout.addView(btnToggle)

        setContentView(layout)
    }

    private fun prepareAndConnectVpn() {
        val intent = VpnService.prepare(this)
        if (intent != null) {
            startActivityForResult(intent, VPN_REQUEST_CODE)
        } else {
            onActivityResult(VPN_REQUEST_CODE, RESULT_OK, null)
        }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == VPN_REQUEST_CODE && resultCode == RESULT_OK) {
            val intent = Intent(this, HelloDpiVpnService::class.java).apply {
                action = HelloDpiVpnService.ACTION_START
            }
            startService(intent)
            isConnected = true
            updateUi()
        }
    }

    private fun disconnectVpn() {
        val intent = Intent(this, HelloDpiVpnService::class.java).apply {
            action = HelloDpiVpnService.ACTION_STOP
        }
        startService(intent)
        isConnected = false
        updateUi()
    }

    private fun updateUi() {
        if (isConnected) {
            txtStatus.text = "TUNEL DEVREDE (0 MS)"
            txtStatus.setTextColor(0xFFFFFFFF.toInt())
            btnToggle.text = "BAGLANTIYI KES"
        } else {
            txtStatus.text = "BAGLANTI KAPALI"
            txtStatus.setTextColor(0xFF71717A.toInt())
            btnToggle.text = "BAGLAN"
        }
    }
}
