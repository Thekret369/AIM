package com.aim.app.core.config

data class ServerConfig(
    val apiBaseUrl: String = DefaultApiBaseUrl,
    val websocketUrl: String = DefaultWebSocketUrl,
) {
    val loginUrl: String
        get() = "${apiBaseUrl.trimEnd('/')}/api/login"

    companion object {
        const val DefaultApiBaseUrl = "http://47.109.109.164:8080"
        const val DefaultWebSocketUrl = "ws://47.109.109.164:8080/ws"

        fun fromInput(apiInput: String, websocketInput: String): ServerConfig {
            val apiBase = normalizeApiBase(apiInput)
            val ws = websocketInput.trim().ifBlank { inferWebSocketUrl(apiBase) }
            return ServerConfig(apiBaseUrl = apiBase, websocketUrl = ws)
        }

        private fun normalizeApiBase(input: String): String {
            val clean = input.trim().trimEnd('/')
            return when {
                clean.endsWith("/api/login") -> clean.removeSuffix("/api/login")
                clean.endsWith("/login") -> clean.removeSuffix("/login")
                else -> clean
            }.ifBlank { DefaultApiBaseUrl }
        }

        private fun inferWebSocketUrl(apiBase: String): String {
            val withoutTrailingSlash = apiBase.trimEnd('/')
            return when {
                withoutTrailingSlash.startsWith("https://") ->
                    withoutTrailingSlash.replaceFirst("https://", "wss://") + "/ws"
                withoutTrailingSlash.startsWith("http://") ->
                    withoutTrailingSlash.replaceFirst("http://", "ws://") + "/ws"
                else -> DefaultWebSocketUrl
            }
        }
    }
}
