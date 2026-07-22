package com.remotepc.agent

import android.content.Intent
import android.media.projection.MediaProjectionManager
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.result.contract.ActivityResultContracts

/**
 * Activity tak berujud (tanpa UI sendiri) yang cuma memicu dialog izin sistem
 * "Mulai merekam atau menyiarkan layar?" milik Android (MediaProjection), lalu
 * meneruskan hasilnya ke AgentService dan langsung menutup diri. Dibuka dari
 * notifikasi "Guru ingin melihat layar Anda" (Tahap A4) — siswa WAJIB
 * mengetuk & menyetujui tiap sesi, tidak ada cara otomatis/diam-diam.
 */
class ScreenCaptureActivity : ComponentActivity() {

    private val launcher = registerForActivityResult(ActivityResultContracts.StartActivityForResult()) { result ->
        val data = result.data
        if (result.resultCode == RESULT_OK && data != null) {
            val serviceIntent = Intent(this, AgentService::class.java).apply {
                action = AgentService.ACTION_SCREEN_GRANTED
                putExtra(AgentService.EXTRA_RESULT_CODE, result.resultCode)
                putExtra(AgentService.EXTRA_RESULT_DATA, data)
            }
            startService(serviceIntent)
        }
        finish()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val mgr = getSystemService(MEDIA_PROJECTION_SERVICE) as MediaProjectionManager
        launcher.launch(mgr.createScreenCaptureIntent())
    }
}
