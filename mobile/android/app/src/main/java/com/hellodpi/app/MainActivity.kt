package com.hellodpi.app

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.VpnService
import android.os.Build
import android.os.Bundle
import android.widget.Button
import android.widget.CheckBox
import android.widget.TextView

/**
 * Minimalist, monochrome activity for Hello DPI Android client.
 */
class MainActivity : Activity() {

    private val VPN_REQUEST_CODE = 0x4844 // "HD"

    private lateinit var btnToggle: Button
    private lateinit var txtStatus: TextView
    private lateinit var chkAutoStart: CheckBox

    private val statusReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            updateUi()
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val prefs = getSharedPreferences(HelloDpiVpnService.PREFS_NAME, Context.MODE_PRIVATE)

        // Programmatic monochrome layout (clean, zero XML overhead)
        val layout = android.widget.LinearLayout(this).apply {
            orientation = android.widget.LinearLayout.VERTICAL
            setBackgroundColor(0xFF000000.toInt())
            gravity = android.view.Gravity.CENTER
            setPadding(48, 48, 48, 48)
        }

        val title = TextView(this).apply {
            text = "HELLO DPI"
            textSize = 26f
            setTextColor(0xFFFFFFFF.toInt())
            typeface = android.graphics.Typeface.DEFAULT_BOLD
            gravity = android.view.Gravity.CENTER
        }

        val subtitle = TextView(this).apply {
            text = "0 ms Ek Gecikme · Sessiz Arka Plan Koruması"
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
            setPadding(36, 18, 36, 18)
            setOnClickListener {
                if (HelloDpiVpnService.isVpnRunning) {
                    disconnectVpn()
                } else {
                    prepareAndConnectVpn()
                }
            }
        }

        chkAutoStart = CheckBox(this).apply {
            text = "Cihaz Açıldığında Otomatik Başlat"
            setTextColor(0xFFA1A1AA.toInt())
            textSize = 12f
            isChecked = prefs.getBoolean(HelloDpiVpnService.KEY_AUTO_START, true)
            setPadding(8, 24, 8, 8)
            setOnCheckedChangeListener { _, isChecked ->
                prefs.edit().putBoolean(HelloDpiVpnService.KEY_AUTO_START, isChecked).apply()
            }
        }

        layout.addView(title)
        layout.addView(subtitle)
        layout.addView(txtStatus)
        layout.addView(btnToggle)
        layout.addView(chkAutoStart)

        setContentView(layout)
    }

    override fun onResume() {
        super.onResume()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(statusReceiver, IntentFilter(HelloDpiVpnService.ACTION_STATUS_CHANGED), RECEIVER_NOT_EXPORTED)
        } else {
            registerReceiver(statusReceiver, IntentFilter(HelloDpiVpnService.ACTION_STATUS_CHANGED))
        }
        updateUi()
    }

    override fun onPause() {
        super.onPause()
        try {
            unregisterReceiver(statusReceiver)
        } catch (ignored: Exception) {}
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
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent)
            } else {
                startService(intent)
            }
            updateUi()
        }
    }

    private fun disconnectVpn() {
        val intent = Intent(this, HelloDpiVpnService::class.java).apply {
            action = HelloDpiVpnService.ACTION_STOP
        }
        startService(intent)
        updateUi()
    }

    private fun updateUi() {
        val isRunning = HelloDpiVpnService.isVpnRunning
        if (isRunning) {
            txtStatus.text = "TUNEL DEVREDE (0 MS)"
            txtStatus.setTextColor(0xFFFFFFFF.toInt())
            btnToggle.text = "KORUMAYI DURDUR"
        } else {
            txtStatus.text = "BAGLANTI KAPALI"
            txtStatus.setTextColor(0xFF71717A.toInt())
            btnToggle.text = "BAGLAN"
        }
    }
}
