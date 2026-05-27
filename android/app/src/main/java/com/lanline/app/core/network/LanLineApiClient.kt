package com.lanline.app.core.network

import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

class LanLineApiClient(
    private val apiBaseUrl: String,
    private val token: String? = null,
) {
    suspend fun get(path: String, query: Map<String, String?> = emptyMap()): JSONObject =
        request("GET", path.withQuery(query))

    suspend fun post(path: String, body: JSONObject = JSONObject()): JSONObject =
        request("POST", path, body)

    suspend fun put(path: String, body: JSONObject = JSONObject()): JSONObject =
        request("PUT", path, body)

    suspend fun delete(path: String): JSONObject =
        request("DELETE", path)

    suspend fun uploadBytes(
        path: String,
        fileName: String,
        contentType: String,
        bytes: ByteArray,
    ): JSONObject = withContext(Dispatchers.IO) {
        val boundary = "LanLineBoundary${System.currentTimeMillis()}"
        val connection = (URL(fullUrl(path)).openConnection() as HttpURLConnection).apply {
            requestMethod = "POST"
            connectTimeout = 10_000
            readTimeout = 30_000
            doOutput = true
            setRequestProperty("Accept", "application/json")
            setRequestProperty("Content-Type", "multipart/form-data; boundary=$boundary")
            token?.takeIf { it.isNotBlank() }?.let { setRequestProperty("Authorization", "Bearer $it") }
        }
        try {
            connection.outputStream.use { output ->
                output.writeMultipartFile(boundary, fileName, contentType, bytes)
            }
            connection.readJsonResponse()
        } finally {
            connection.disconnect()
        }
    }

    private suspend fun request(method: String, path: String, body: JSONObject? = null): JSONObject =
        withContext(Dispatchers.IO) {
            val connection = (URL(fullUrl(path)).openConnection() as HttpURLConnection).apply {
                requestMethod = method
                connectTimeout = 8_000
                readTimeout = 15_000
                setRequestProperty("Accept", "application/json")
                token?.takeIf { it.isNotBlank() }?.let { setRequestProperty("Authorization", "Bearer $it") }
                if (body != null) {
                    doOutput = true
                    setRequestProperty("Content-Type", "application/json; charset=utf-8")
                }
            }
            try {
                if (body != null) {
                    connection.outputStream.use { output ->
                        output.write(body.toString().toByteArray(StandardCharsets.UTF_8))
                    }
                }
                connection.readJsonResponse()
            } finally {
                connection.disconnect()
            }
        }

    private fun fullUrl(path: String): String {
        if (path.startsWith("http://") || path.startsWith("https://")) return path
        val cleanBase = apiBaseUrl.trimEnd('/')
        val apiRoot = if (cleanBase.endsWith("/api")) cleanBase else "$cleanBase/api"
        return "$apiRoot/${path.trimStart('/')}"
    }

    private fun String.withQuery(query: Map<String, String?>): String {
        val params = query.filterValues { !it.isNullOrBlank() }
        if (params.isEmpty()) return this
        val queryText = params.entries.joinToString("&") { (key, value) ->
            "${key.urlEncode()}=${value.orEmpty().urlEncode()}"
        }
        val separator = if (contains("?")) "&" else "?"
        return "$this$separator$queryText"
    }

    private fun String.urlEncode(): String =
        URLEncoder.encode(this, StandardCharsets.UTF_8.name())

    private fun HttpURLConnection.readJsonResponse(): JSONObject {
        val statusCode = responseCode
        val stream = if (statusCode in 200..299) inputStream else errorStream
        val text = stream?.bufferedReader(StandardCharsets.UTF_8)?.use { it.readText() }.orEmpty()
        if (statusCode !in 200..299) {
            throw LanLineNetworkException(statusCode, parseErrorMessage(text, statusCode))
        }
        return if (text.isBlank()) JSONObject() else JSONObject(text)
    }

    private fun parseErrorMessage(text: String, statusCode: Int): String {
        if (text.isBlank()) return "接口请求失败，HTTP $statusCode"
        return runCatching {
            val json = JSONObject(text)
            json.optString("error").ifBlank {
                json.optString("message").ifBlank { "接口请求失败，HTTP $statusCode" }
            }
        }.getOrDefault(text.take(160))
    }

    private fun java.io.OutputStream.writeMultipartFile(
        boundary: String,
        fileName: String,
        contentType: String,
        bytes: ByteArray,
    ) {
        val safeType = contentType.ifBlank { "application/octet-stream" }
        val header = buildString {
            append("--$boundary\r\n")
            append("Content-Disposition: form-data; name=\"file\"; filename=\"$fileName\"\r\n")
            append("Content-Type: $safeType\r\n\r\n")
        }
        val footer = "\r\n--$boundary--\r\n"
        write(header.toByteArray(StandardCharsets.UTF_8))
        write(bytes)
        write(footer.toByteArray(StandardCharsets.UTF_8))
    }
}

class LanLineNetworkException(
    val statusCode: Int,
    message: String,
) : Exception(message)

fun jsonBody(vararg pairs: Pair<String, Any?>): JSONObject =
    JSONObject().apply {
        pairs.forEach { (key, value) ->
            if (value != null) put(key, value)
        }
    }
