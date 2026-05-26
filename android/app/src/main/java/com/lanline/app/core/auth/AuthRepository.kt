package com.lanline.app.core.auth

import com.lanline.app.core.config.ServerConfig
import java.net.HttpURLConnection
import java.net.URL
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

class AuthRepository(
    private val config: ServerConfig,
) {
    suspend fun login(username: String, password: String): Result<AuthSession> =
        withContext(Dispatchers.IO) {
            runCatching {
                val requestBody = JSONObject()
                    .put("username", username)
                    .put("password", password)
                    .toString()

                val connection = (URL(config.loginUrl).openConnection() as HttpURLConnection).apply {
                    requestMethod = "POST"
                    connectTimeout = 8_000
                    readTimeout = 12_000
                    doOutput = true
                    setRequestProperty("Accept", "application/json")
                    setRequestProperty("Content-Type", "application/json; charset=utf-8")
                }

                try {
                    connection.outputStream.use { output ->
                        output.write(requestBody.toByteArray(StandardCharsets.UTF_8))
                    }

                    val statusCode = connection.responseCode
                    val responseText = connection.readResponseBody(statusCode)
                    if (statusCode !in 200..299) {
                        throw LanLineApiException(statusCode, parseErrorMessage(responseText, statusCode))
                    }

                    parseLoginResponse(responseText)
                } finally {
                    connection.disconnect()
                }
            }
        }

    private fun HttpURLConnection.readResponseBody(statusCode: Int): String {
        val stream = if (statusCode in 200..299) inputStream else errorStream
        return stream?.bufferedReader(StandardCharsets.UTF_8)?.use { it.readText() }.orEmpty()
    }

    private fun parseLoginResponse(responseText: String): AuthSession {
        val root = JSONObject(responseText)
        val user = root.getJSONObject("user")
        return AuthSession(
            token = root.getString("token"),
            user = AuthUser(
                id = user.optLong("id"),
                username = user.optString("username"),
                nickname = user.optString("nickname"),
                avatar = user.optString("avatar"),
                bio = user.optString("bio"),
                isAi = user.optBoolean("is_ai"),
            ),
            apiBaseUrl = config.apiBaseUrl,
            websocketUrl = config.websocketUrl,
        )
    }

    private fun parseErrorMessage(responseText: String, statusCode: Int): String {
        if (responseText.isBlank()) return "登录失败，HTTP $statusCode"
        return runCatching {
            val root = JSONObject(responseText)
            root.optString("error").ifBlank {
                root.optString("message").ifBlank { "登录失败，HTTP $statusCode" }
            }
        }.getOrDefault(responseText.take(120))
    }
}
