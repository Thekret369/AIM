package com.lanline.app.core.auth

import android.content.Context
import com.lanline.app.core.config.ServerConfig
import org.json.JSONObject

class AuthSessionStore(context: Context) {
    private val preferences = context.getSharedPreferences(StoreName, Context.MODE_PRIVATE)

    fun load(): AuthSession? {
        val raw = preferences.getString(SessionKey, null) ?: return null
        return runCatching {
            val root = JSONObject(raw)
            val user = root.getJSONObject("user")
            AuthSession(
                token = root.getString("token"),
                user = AuthUser(
                    id = user.optLong("id"),
                    username = user.optString("username"),
                    nickname = user.optString("nickname"),
                    avatar = user.optString("avatar"),
                    bio = user.optString("bio"),
                    isAi = user.optBoolean("is_ai"),
                ),
                apiBaseUrl = root.optString("api_base_url").ifBlank { ServerConfig.DefaultApiBaseUrl },
                websocketUrl = root.optString("websocket_url").ifBlank { ServerConfig.DefaultWebSocketUrl },
            )
        }.getOrNull()
    }

    fun save(session: AuthSession) {
        val raw = JSONObject()
            .put("token", session.token)
            .put(
                "user",
                JSONObject()
                    .put("id", session.user.id)
                    .put("username", session.user.username)
                    .put("nickname", session.user.nickname)
                    .put("avatar", session.user.avatar)
                    .put("bio", session.user.bio)
                    .put("is_ai", session.user.isAi),
            )
            .put("api_base_url", session.apiBaseUrl)
            .put("websocket_url", session.websocketUrl)
            .toString()
        preferences.edit().putString(SessionKey, raw).apply()
    }

    fun clear() {
        preferences.edit().remove(SessionKey).apply()
    }

    companion object {
        private const val StoreName = "lanline_auth"
        private const val SessionKey = "session"
    }
}
