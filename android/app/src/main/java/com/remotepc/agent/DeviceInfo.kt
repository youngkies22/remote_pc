package com.remotepc.agent

import android.app.ActivityManager
import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.BatteryManager
import android.os.Build
import android.os.Environment
import android.os.StatFs
import android.os.SystemClock
import java.net.NetworkInterface
import java.util.Collections

/**
 * Kumpulan informasi perangkat & metrik runtime. CPU percent SENGAJA tidak
 * diisi (selalu 0.0) karena sejak Android 8 aplikasi biasa tidak lagi bisa
 * membaca /proc/stat sistem tanpa root — beda dari Windows yang bebas lewat WMI.
 */
object DeviceInfo {

    fun arch(): String = Build.SUPPORTED_ABIS.firstOrNull() ?: "unknown"

    fun osVersion(): String = "Android ${Build.VERSION.RELEASE} (SDK ${Build.VERSION.SDK_INT})"

    fun model(): String = "${Build.MANUFACTURER} ${Build.MODEL}".trim()

    /** Ambil alamat IPv4 lokal pertama yang bukan loopback (Wi-Fi atau data seluler). */
    fun localIp(): String {
        return try {
            val interfaces = Collections.list(NetworkInterface.getNetworkInterfaces())
            for (intf in interfaces) {
                val addrs = Collections.list(intf.inetAddresses)
                for (addr in addrs) {
                    if (!addr.isLoopbackAddress && addr.hostAddress?.contains(':') == false) {
                        return addr.hostAddress ?: ""
                    }
                }
            }
            ""
        } catch (e: Exception) {
            ""
        }
    }

    fun networkType(context: Context): String {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
            ?: return "none"
        val caps = cm.getNetworkCapabilities(cm.activeNetwork) ?: return "none"
        return when {
            caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "wifi"
            caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "cellular"
            caps.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "wifi"
            else -> "none"
        }
    }

    fun batteryPercent(context: Context): Double {
        val bm = context.getSystemService(Context.BATTERY_SERVICE) as? BatteryManager ?: return 0.0
        val pct = bm.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY)
        return if (pct in 0..100) pct.toDouble() else 0.0
    }

    fun collectMetrics(context: Context): Metrics {
        val am = context.getSystemService(Context.ACTIVITY_SERVICE) as ActivityManager
        val memInfo = ActivityManager.MemoryInfo()
        am.getMemoryInfo(memInfo)
        val ramTotalMb = memInfo.totalMem / (1024 * 1024)
        val ramUsedMb = ramTotalMb - (memInfo.availMem / (1024 * 1024))
        val ramPercent = if (ramTotalMb > 0) (ramUsedMb.toDouble() / ramTotalMb.toDouble()) * 100.0 else 0.0

        val stat = StatFs(Environment.getDataDirectory().path)
        val diskTotalGb = (stat.totalBytes) / (1024L * 1024 * 1024)
        val diskFreeGb = (stat.availableBytes) / (1024L * 1024 * 1024)
        val diskUsedGb = diskTotalGb - diskFreeGb
        val diskPercent = if (diskTotalGb > 0) (diskUsedGb.toDouble() / diskTotalGb.toDouble()) * 100.0 else 0.0

        return Metrics(
            cpuPercent = 0.0,
            ramPercent = round1(ramPercent),
            ramTotalMb = ramTotalMb,
            ramUsedMb = ramUsedMb,
            diskPercent = round1(diskPercent),
            diskTotalGb = diskTotalGb,
            diskUsedGb = diskUsedGb,
            uptimeSec = SystemClock.elapsedRealtime() / 1000,
            batteryPercent = batteryPercent(context),
            networkType = networkType(context)
        )
    }

    private fun round1(v: Double): Double = Math.round(v * 10.0) / 10.0
}
