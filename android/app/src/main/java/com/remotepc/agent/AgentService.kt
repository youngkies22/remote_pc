package com.remotepc.agent

import android.app.Activity
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.content.pm.ServiceInfo
import android.graphics.Bitmap
import android.graphics.PixelFormat
import android.graphics.drawable.Icon
import android.hardware.display.DisplayManager
import android.hardware.display.VirtualDisplay
import android.media.Image
import android.media.ImageReader
import android.media.projection.MediaProjection
import android.media.projection.MediaProjectionManager
import android.os.Build
import android.os.IBinder
import android.util.Base64
import android.util.Log
import com.google.gson.Gson
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import okio.ByteString
import java.io.ByteArrayOutputStream
import java.util.UUID
import java.util.concurrent.TimeUnit

enum class ConnState { DISCONNECTED, CONNECTING, ONLINE }

/**
 * Foreground service yang menjaga koneksi WebSocket ke server RemotePC tetap
 * hidup: connect -> register -> heartbeat berkala -> reconnect otomatis saat
 * putus. Juga menangani pesan dari guru (Tahap A3) dan Live Screen (Tahap A4).
 * Meniru pola internal/agent/client.go + dispatch.go di sisi Windows.
 */
class AgentService : Service() {

    private val gson = Gson()
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var supervisorJob: Job? = null
    private var heartbeatJob: Job? = null
    private var screenJob: Job? = null
    private var activeWebSocket: WebSocket? = null
    private lateinit var prefs: Prefs

    // --- Live Screen (MediaProjection) ---
    private var mediaProjection: MediaProjection? = null
    private var virtualDisplay: VirtualDisplay? = null
    private var imageReader: ImageReader? = null
    private var screenQuality: String = "normal"
    private var messageNotifCounter = 100

    private val client = OkHttpClient.Builder()
        .pingInterval(20, TimeUnit.SECONDS)
        .build()

    override fun onCreate() {
        super.onCreate()
        prefs = Prefs(this)
        createNotificationChannels()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopSelf()
                return START_NOT_STICKY
            }
            ACTION_SCREEN_GRANTED -> {
                handleScreenGranted(intent)
                return START_STICKY
            }
        }
        startForegroundDataSync(buildStatusNotification(ConnState.CONNECTING))
        if (supervisorJob?.isActive != true) {
            state.value = ConnState.CONNECTING
            supervisorJob = scope.launch { connectionLoop() }
        }
        return START_STICKY
    }

    override fun onDestroy() {
        state.value = ConnState.DISCONNECTED
        supervisorJob?.cancel()
        heartbeatJob?.cancel()
        teardownProjection()
        activeWebSocket?.close(1000, "service dihentikan")
        scope.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    /**
     * Foreground service tipe "dataSync" saja dipakai sejak koneksi awal —
     * ini aman & tak butuh syarat apa pun. Tipe "mediaProjection" BARU
     * ditambahkan (lewat [promoteForegroundForProjection]) tepat sesudah izin
     * MediaProjection didapat, karena Android 14+ mewajibkan consent sudah ada
     * SEBELUM service boleh berjalan sebagai foreground type mediaProjection —
     * mendeklarasikannya sejak awal (sebelum ada consent) melempar exception
     * yang bisa meng-crash service ini sebelum sempat connect ke server sama
     * sekali.
     */
    private fun startForegroundDataSync(notification: Notification) {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(NOTIF_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC)
        } else {
            startForeground(NOTIF_ID, notification)
        }
    }

    /** Dipanggil hanya setelah user menyetujui dialog sistem MediaProjection. */
    private fun promoteForegroundForProjection(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q) return true
        return try {
            startForeground(
                NOTIF_ID, buildStatusNotification(state.value),
                ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC or ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION
            )
            true
        } catch (e: Exception) {
            Log.w(TAG, "gagal promote foreground service ke mediaProjection: ${e.message}")
            false
        }
    }

    // ================= Koneksi & registrasi (Tahap A1) =================

    private suspend fun connectionLoop() {
        while (true) {
            try {
                connectOnce()
            } catch (e: Exception) {
                Log.w(TAG, "koneksi berakhir: ${e.message}")
            }
            state.value = ConnState.CONNECTING
            updateStatusNotification(ConnState.CONNECTING)
            delay(RECONNECT_MS)
        }
    }

    private suspend fun connectOnce() {
        val host = prefs.host.trim()
        val port = prefs.port
        if (host.isEmpty()) {
            delay(RECONNECT_MS)
            return
        }
        val url = "ws://$host:$port/ws/agent"
        val closed = CompletableDeferred<Unit>()

        val request = Request.Builder().url(url).build()
        val ws = client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: okhttp3.Response) {
                activeWebSocket = webSocket
                sendRegister(webSocket)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                handleMessage(text)
            }

            override fun onMessage(webSocket: WebSocket, bytes: ByteString) {
                // Server tidak mengirim frame biner ke agent.
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                if (!closed.isCompleted) closed.complete(Unit)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: okhttp3.Response?) {
                Log.w(TAG, "websocket gagal: ${t.message}")
                if (!closed.isCompleted) closed.complete(Unit)
            }
        })

        closed.await()
        activeWebSocket = null
        heartbeatJob?.cancel()
        heartbeatJob = null
        stopScreenCapture()
    }

    private fun sendRegister(ws: WebSocket) {
        val info = RegisterInfo(
            deviceId = prefs.deviceId,
            token = prefs.deviceToken,
            hostname = prefs.nickname.ifBlank { DeviceInfo.model() },
            username = DeviceInfo.model(),
            ip = DeviceInfo.localIp(),
            mac = "",
            os = "Android",
            osVersion = DeviceInfo.osVersion(),
            arch = DeviceInfo.arch()
        )
        sendEnvelope(ws, MessageType.REGISTER, null, gson.toJsonTree(info))
    }

    private fun handleMessage(text: String) {
        val env = try {
            gson.fromJson(text, Envelope::class.java)
        } catch (e: Exception) {
            Log.w(TAG, "pesan tidak valid: ${e.message}")
            return
        }
        when (env.type) {
            MessageType.REGISTER_RESULT -> onRegisterResult(env)
            MessageType.MESSAGE -> onTeacherMessage(env)
            MessageType.SCREEN_START -> onScreenStart()
            MessageType.SCREEN_STOP -> onScreenStop()
            MessageType.SCREEN_QUALITY -> onScreenQuality(env)
            MessageType.ERROR -> Log.w(TAG, "error dari server: ${env.error}")
            else -> Log.d(TAG, "tipe pesan belum didukung di agent Android: ${env.type}")
        }
    }

    private fun onRegisterResult(env: Envelope) {
        val ws = activeWebSocket ?: return
        val payload = env.payload
        val result = if (payload == null) null else try {
            gson.fromJson(payload, RegisterResult::class.java)
        } catch (e: Exception) {
            null
        }
        if (result == null || !result.accepted) {
            Log.w(TAG, "registrasi ditolak: ${result?.message}")
            ws.close(1000, "registrasi ditolak")
            return
        }
        prefs.saveIdentity(result.deviceId, result.token)
        state.value = ConnState.ONLINE
        updateStatusNotification(ConnState.ONLINE)
        heartbeatJob?.cancel()
        heartbeatJob = scope.launch { heartbeatLoop(result.deviceId) }
    }

    private suspend fun heartbeatLoop(deviceId: String) {
        while (true) {
            val ws = activeWebSocket ?: return
            val hb = Heartbeat(
                hostname = prefs.nickname.ifBlank { DeviceInfo.model() },
                username = DeviceInfo.model(),
                ip = DeviceInfo.localIp(),
                mac = "",
                metrics = DeviceInfo.collectMetrics(applicationContext)
            )
            sendEnvelope(ws, MessageType.HEARTBEAT, deviceId, gson.toJsonTree(hb))
            delay(HEARTBEAT_MS)
        }
    }

    private fun sendEnvelope(ws: WebSocket, type: String, deviceId: String?, payload: com.google.gson.JsonElement?) {
        val env = Envelope(
            id = UUID.randomUUID().toString(),
            type = type,
            deviceId = deviceId,
            timestamp = System.currentTimeMillis(),
            payload = payload
        )
        ws.send(gson.toJson(env))
    }

    // ================= Tahap A3: pesan dari guru =================

    private fun onTeacherMessage(env: Envelope) {
        val payload = env.payload ?: return
        val req = try {
            gson.fromJson(payload, MessageRequest::class.java)
        } catch (e: Exception) {
            null
        } ?: return
        if (req.text.isBlank()) return
        showAlertNotification(
            req.title.ifBlank { "Pesan dari Guru" },
            req.text,
            messageNotifCounter++
        )
    }

    // ================= Tahap A4: Live Screen =================

    private fun onScreenStart() {
        val projection = mediaProjection
        if (projection != null) {
            // Izin MediaProjection masih berlaku (belum di-stop total) — tinggal
            // buat ulang VirtualDisplay/ImageReader kalau sempat dilepas saat
            // screen.stop sebelumnya, TANPA minta izin sistem lagi.
            if (virtualDisplay == null) setupVirtualDisplay(projection)
            beginScreenCapture()
            return
        }
        showAlertNotification(
            "Guru ingin melihat layar Anda",
            "Ketuk notifikasi ini, lalu pilih \"Mulai sekarang\" pada dialog sistem untuk mengizinkan.",
            NOTIF_ID_SCREEN_REQUEST,
            contentIntent = PendingIntent.getActivity(
                this, 0,
                Intent(this, ScreenCaptureActivity::class.java).apply {
                    addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
                },
                PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
            )
        )
    }

    // screen.stop cuma menghentikan CAPTURE (VirtualDisplay/ImageReader) — token
    // MediaProjection SENGAJA dibiarkan hidup supaya screen.start berikutnya
    // (pindah dari grid /hp/live ke tampilan individual, balik lagi, dst) tak
    // perlu minta izin sistem berulang-ulang. Token baru benar-benar dilepas di
    // onDestroy() (service berhenti) atau bila sistem sendiri yang mencabutnya
    // (MediaProjection.Callback.onStop di bawah).
    private fun onScreenStop() {
        stopScreenCapture()
        virtualDisplay?.release()
        virtualDisplay = null
        imageReader?.close()
        imageReader = null
    }

    private fun onScreenQuality(env: Envelope) {
        val payload = env.payload ?: return
        val req = try {
            gson.fromJson(payload, ScreenQualityRequest::class.java)
        } catch (e: Exception) {
            null
        } ?: return
        screenQuality = if (req.quality == "hd") "hd" else "normal"
    }

    private fun handleScreenGranted(intent: Intent) {
        val resultCode = intent.getIntExtra(EXTRA_RESULT_CODE, Activity.RESULT_CANCELED)
        val data = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            intent.getParcelableExtra(EXTRA_RESULT_DATA, Intent::class.java)
        } else {
            @Suppress("DEPRECATION")
            intent.getParcelableExtra(EXTRA_RESULT_DATA)
        }
        if (resultCode != Activity.RESULT_OK || data == null) return
        if (!promoteForegroundForProjection()) return
        val mgr = getSystemService(MEDIA_PROJECTION_SERVICE) as MediaProjectionManager
        val projection = mgr.getMediaProjection(resultCode, data) ?: return
        mediaProjection = projection
        projection.registerCallback(object : MediaProjection.Callback() {
            override fun onStop() {
                teardownProjection()
            }
        }, null)
        setupVirtualDisplay(projection)
        beginScreenCapture()
    }

    private fun setupVirtualDisplay(projection: MediaProjection) {
        val metrics = resources.displayMetrics
        val width = metrics.widthPixels
        val height = metrics.heightPixels
        val density = metrics.densityDpi
        val reader = ImageReader.newInstance(width, height, PixelFormat.RGBA_8888, 2)
        imageReader = reader
        virtualDisplay = projection.createVirtualDisplay(
            "RemotePCCapture", width, height, density,
            DisplayManager.VIRTUAL_DISPLAY_FLAG_AUTO_MIRROR,
            reader.surface, null, null
        )
    }

    private fun beginScreenCapture() {
        screenJob?.cancel()
        screenJob = scope.launch { screenCaptureLoop() }
    }

    private fun stopScreenCapture() {
        screenJob?.cancel()
        screenJob = null
    }

    private fun teardownProjection() {
        stopScreenCapture()
        virtualDisplay?.release()
        virtualDisplay = null
        imageReader?.close()
        imageReader = null
        mediaProjection?.stop()
        mediaProjection = null
    }

    private suspend fun screenCaptureLoop() {
        while (true) {
            val ws = activeWebSocket
            val deviceId = prefs.deviceId
            val reader = imageReader
            if (ws != null && reader != null) {
                captureFrame(reader)?.let { shot ->
                    sendEnvelope(ws, MessageType.SCREEN_FRAME, deviceId, gson.toJsonTree(shot))
                }
            }
            delay(SCREEN_INTERVAL_MS)
        }
    }

    private fun captureFrame(reader: ImageReader): ScreenShot? {
        val image: Image = reader.acquireLatestImage() ?: return null
        try {
            val plane = image.planes[0]
            val buffer = plane.buffer
            val pixelStride = plane.pixelStride
            val rowStride = plane.rowStride
            val rowPadding = rowStride - pixelStride * image.width

            val bitmap = Bitmap.createBitmap(
                image.width + rowPadding / pixelStride, image.height, Bitmap.Config.ARGB_8888
            )
            bitmap.copyPixelsFromBuffer(buffer)

            val cropped = if (rowPadding == 0) bitmap else
                Bitmap.createBitmap(bitmap, 0, 0, image.width, image.height)

            val quality = if (screenQuality == "hd") 92 else 60
            val out = ByteArrayOutputStream()
            cropped.compress(Bitmap.CompressFormat.JPEG, quality, out)
            val base64 = Base64.encodeToString(out.toByteArray(), Base64.NO_WRAP)
            return ScreenShot(format = "jpeg", width = image.width, height = image.height, data = base64)
        } finally {
            image.close()
        }
    }

    // ================= Notifikasi =================

    private fun createNotificationChannels() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val nm = getSystemService(NotificationManager::class.java)
            nm.createNotificationChannel(
                NotificationChannel(
                    CHANNEL_ID_STATUS, getString(R.string.notif_channel_name),
                    NotificationManager.IMPORTANCE_LOW
                )
            )
            nm.createNotificationChannel(
                NotificationChannel(
                    CHANNEL_ID_ALERT, getString(R.string.notif_channel_alert),
                    NotificationManager.IMPORTANCE_HIGH
                )
            )
        }
    }

    private fun buildStatusNotification(connState: ConnState): Notification {
        val title = if (connState == ConnState.ONLINE)
            getString(R.string.notif_title_connected) else getString(R.string.notif_title_connecting)

        val stopIntent = Intent(this, AgentService::class.java).apply { action = ACTION_STOP }
        val stopPending = PendingIntent.getService(
            this, 0, stopIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        return Notification.Builder(this, CHANNEL_ID_STATUS)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(getString(R.string.notif_text))
            .setOngoing(true)
            .addAction(
                Notification.Action.Builder(
                    Icon.createWithResource(this, R.drawable.ic_notification),
                    getString(R.string.notif_action_stop),
                    stopPending
                ).build()
            )
            .build()
    }

    private fun updateStatusNotification(connState: ConnState) {
        val nm = getSystemService(NotificationManager::class.java)
        nm.notify(NOTIF_ID, buildStatusNotification(connState))
    }

    private fun showAlertNotification(
        title: String,
        text: String,
        notifId: Int,
        contentIntent: PendingIntent? = null
    ) {
        val nm = getSystemService(NotificationManager::class.java)
        val builder = Notification.Builder(this, CHANNEL_ID_ALERT)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(Notification.BigTextStyle().bigText(text))
            .setAutoCancel(true)
        if (contentIntent != null) builder.setContentIntent(contentIntent)
        nm.notify(notifId, builder.build())
    }

    companion object {
        private const val TAG = "AgentService"
        private const val CHANNEL_ID_STATUS = "remotepc_status"
        private const val CHANNEL_ID_ALERT = "remotepc_alert"
        private const val NOTIF_ID = 1
        private const val NOTIF_ID_SCREEN_REQUEST = 2
        private const val RECONNECT_MS = 5000L
        private const val HEARTBEAT_MS = 2000L
        private const val SCREEN_INTERVAL_MS = 300L
        const val ACTION_STOP = "com.remotepc.agent.STOP"
        const val ACTION_SCREEN_GRANTED = "com.remotepc.agent.SCREEN_GRANTED"
        const val EXTRA_RESULT_CODE = "result_code"
        const val EXTRA_RESULT_DATA = "result_data"

        val state = MutableStateFlow(ConnState.DISCONNECTED)
        val stateFlow = state.asStateFlow()
    }
}
