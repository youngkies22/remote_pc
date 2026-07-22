package com.remotepc.agent

import com.google.gson.JsonElement
import com.google.gson.annotations.SerializedName

/**
 * Struktur pesan berikut sengaja dibuat identik (nama field JSON) dengan
 * internal/protocol dan internal/model di sisi server Go, supaya server tidak
 * perlu tahu bahwa lawan bicaranya adalah agent Android, bukan agent Windows.
 */

data class Envelope(
    val id: String,
    val type: String,
    @SerializedName("device_id") val deviceId: String? = null,
    val timestamp: Long,
    val payload: JsonElement? = null,
    val error: String? = null
)

data class RegisterInfo(
    @SerializedName("device_id") val deviceId: String,
    val token: String,
    val hostname: String,
    val username: String,
    val ip: String,
    val mac: String,
    val os: String,
    @SerializedName("windows_version") val osVersion: String,
    val arch: String
)

data class RegisterResult(
    @SerializedName("device_id") val deviceId: String = "",
    val token: String = "",
    val accepted: Boolean = false,
    val message: String? = null
)

data class Metrics(
    @SerializedName("cpu_percent") val cpuPercent: Double,
    @SerializedName("ram_percent") val ramPercent: Double,
    @SerializedName("ram_total_mb") val ramTotalMb: Long,
    @SerializedName("ram_used_mb") val ramUsedMb: Long,
    @SerializedName("disk_percent") val diskPercent: Double,
    @SerializedName("disk_total_gb") val diskTotalGb: Long,
    @SerializedName("disk_used_gb") val diskUsedGb: Long,
    @SerializedName("uptime_sec") val uptimeSec: Long,
    @SerializedName("battery_percent") val batteryPercent: Double,
    @SerializedName("network_type") val networkType: String
)

data class Heartbeat(
    val hostname: String,
    val username: String,
    val ip: String,
    val mac: String,
    val metrics: Metrics
)

// --- Tahap A3: pesan dari guru ---
data class MessageRequest(
    val title: String,
    val text: String
)

// --- Tahap A4: Live Screen ---
data class ScreenShot(
    val format: String,
    val width: Int,
    val height: Int,
    val data: String
)

data class ScreenQualityRequest(
    val quality: String
)

object MessageType {
    const val REGISTER = "register"
    const val REGISTER_RESULT = "register_result"
    const val HEARTBEAT = "heartbeat"
    const val ERROR = "error"
    const val MESSAGE = "message"
    const val SCREEN_START = "screen.start"
    const val SCREEN_STOP = "screen.stop"
    const val SCREEN_QUALITY = "screen.quality"
    const val SCREEN_FRAME = "screen.frame"
}
