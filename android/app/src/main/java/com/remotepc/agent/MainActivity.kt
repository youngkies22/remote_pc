package com.remotepc.agent

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.lifecycleScope
import androidx.lifecycle.repeatOnLifecycle
import com.remotepc.agent.databinding.ActivityMainBinding
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var prefs: Prefs

    private val notifPermissionLauncher =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { startService() }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)
        prefs = Prefs(this)

        binding.inputNickname.setText(prefs.nickname)
        binding.inputHost.setText(prefs.host)
        binding.inputPort.setText(prefs.port.toString())

        binding.btnConnect.setOnClickListener { onConnectClicked() }
        binding.btnDisconnect.setOnClickListener {
            startService(Intent(this, AgentService::class.java).apply { action = AgentService.ACTION_STOP })
        }

        lifecycleScope.launch {
            repeatOnLifecycle(Lifecycle.State.STARTED) {
                AgentService.stateFlow.collect { state -> renderState(state) }
            }
        }
    }

    private fun onConnectClicked() {
        prefs.nickname = binding.inputNickname.text?.toString()?.trim().orEmpty()
        prefs.host = binding.inputHost.text?.toString()?.trim().orEmpty()
        prefs.port = binding.inputPort.text?.toString()?.trim()?.toIntOrNull() ?: 9000

        if (prefs.host.isEmpty()) {
            binding.textStatus.text = "Isi dulu IP/host server"
            return
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS)
            != PackageManager.PERMISSION_GRANTED
        ) {
            notifPermissionLauncher.launch(Manifest.permission.POST_NOTIFICATIONS)
            return
        }
        startService()
    }

    private fun startService() {
        val intent = Intent(this, AgentService::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent)
        } else {
            startService(intent)
        }
    }

    private fun renderState(state: ConnState) {
        binding.textStatus.text = when (state) {
            ConnState.DISCONNECTED -> "Status: terputus"
            ConnState.CONNECTING -> "Status: menghubungkan…"
            ConnState.ONLINE -> "Status: terhubung ke guru"
        }
    }
}
