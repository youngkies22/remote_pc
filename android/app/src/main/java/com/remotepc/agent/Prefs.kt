package com.remotepc.agent

import android.content.Context

/** Penyimpanan sederhana untuk konfigurasi & identitas device (mirip agent.yaml di sisi Windows). */
class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("remotepc_agent", Context.MODE_PRIVATE)

    var nickname: String
        get() = sp.getString(KEY_NICKNAME, "") ?: ""
        set(value) = sp.edit().putString(KEY_NICKNAME, value).apply()

    var host: String
        get() = sp.getString(KEY_HOST, "") ?: ""
        set(value) = sp.edit().putString(KEY_HOST, value).apply()

    var port: Int
        get() = sp.getInt(KEY_PORT, 9000)
        set(value) = sp.edit().putInt(KEY_PORT, value).apply()

    // Diisi server saat registrasi pertama kali, lalu dipakai ulang tiap reconnect
    // supaya device_id di dashboard tetap sama (bukan bikin device baru terus).
    var deviceId: String
        get() = sp.getString(KEY_DEVICE_ID, "") ?: ""
        set(value) = sp.edit().putString(KEY_DEVICE_ID, value).apply()

    var deviceToken: String
        get() = sp.getString(KEY_DEVICE_TOKEN, "") ?: ""
        set(value) = sp.edit().putString(KEY_DEVICE_TOKEN, value).apply()

    fun saveIdentity(deviceId: String, token: String) {
        sp.edit()
            .putString(KEY_DEVICE_ID, deviceId)
            .putString(KEY_DEVICE_TOKEN, token)
            .apply()
    }

    companion object {
        private const val KEY_NICKNAME = "nickname"
        private const val KEY_HOST = "host"
        private const val KEY_PORT = "port"
        private const val KEY_DEVICE_ID = "device_id"
        private const val KEY_DEVICE_TOKEN = "device_token"
    }
}
