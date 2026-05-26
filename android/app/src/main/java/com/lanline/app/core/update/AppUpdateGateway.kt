package com.lanline.app.core.update

import com.lanline.app.core.config.ServerConfig
import java.net.HttpURLConnection
import java.net.URLEncoder
import java.net.URL
import java.nio.charset.StandardCharsets
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.json.JSONObject

/**
 * 测试包更新检查接口。
 * 后续可以接自部署更新服务，返回 APK 下载地址、强更策略和更新说明。
 */
interface AppUpdateGateway {
    suspend fun checkLatest(request: AppUpdateCheckRequest): Result<AppUpdateInfo>
}

data class AppUpdateCheckRequest(
    val versionCode: Int,
    val versionName: String,
    val channel: UpdateChannel,
    val deviceAbi: String,
)

data class AppUpdateInfo(
    val hasUpdate: Boolean,
    val latestVersionCode: Int,
    val latestVersionName: String,
    val downloadUrl: String? = null,
    val checksumSha256: String? = null,
    val requirement: UpdateRequirement = UpdateRequirement.Optional,
    val releaseNotes: List<String> = emptyList(),
)

enum class UpdateChannel {
    Internal,
    Alpha,
    Beta,
    Release,
}

enum class UpdateRequirement {
    Optional,
    Recommended,
    Required,
}

object NoopAppUpdateGateway : AppUpdateGateway {
    override suspend fun checkLatest(request: AppUpdateCheckRequest): Result<AppUpdateInfo> =
        Result.success(
            AppUpdateInfo(
                hasUpdate = false,
                latestVersionCode = request.versionCode,
                latestVersionName = request.versionName,
            ),
        )
}

class HttpAppUpdateGateway(
    private val config: ServerConfig,
) : AppUpdateGateway {
    override suspend fun checkLatest(request: AppUpdateCheckRequest): Result<AppUpdateInfo> =
        withContext(Dispatchers.IO) {
            runCatching {
                val connection = (URL(buildUrl(request)).openConnection() as HttpURLConnection).apply {
                    requestMethod = "GET"
                    connectTimeout = 6_000
                    readTimeout = 8_000
                    setRequestProperty("Accept", "application/json")
                }
                try {
                    val statusCode = connection.responseCode
                    val stream = if (statusCode in 200..299) connection.inputStream else connection.errorStream
                    val body = stream?.bufferedReader(StandardCharsets.UTF_8)?.use { it.readText() }.orEmpty()
                    if (statusCode !in 200..299) error("更新检查失败，HTTP $statusCode")
                    parseInfo(body)
                } finally {
                    connection.disconnect()
                }
            }
        }

    private fun buildUrl(request: AppUpdateCheckRequest): String {
        val query = listOf(
            "version_code" to request.versionCode.toString(),
            "version_name" to request.versionName,
            "channel" to request.channel.name,
            "device_abi" to request.deviceAbi,
        ).joinToString("&") { (key, value) ->
            "$key=${URLEncoder.encode(value, StandardCharsets.UTF_8.name())}"
        }
        return "${config.androidUpdateUrl}?$query"
    }

    private fun parseInfo(body: String): AppUpdateInfo {
        val root = JSONObject(body)
        val notes = root.optJSONArray("release_notes")
        return AppUpdateInfo(
            hasUpdate = root.optBoolean("has_update"),
            latestVersionCode = root.optInt("latest_version_code"),
            latestVersionName = root.optString("latest_version_name"),
            downloadUrl = root.optString("download_url").ifBlank { null },
            checksumSha256 = root.optString("checksum_sha256").ifBlank { null },
            requirement = runCatching {
                UpdateRequirement.valueOf(root.optString("requirement", UpdateRequirement.Optional.name))
            }.getOrDefault(UpdateRequirement.Optional),
            releaseNotes = buildList {
                if (notes != null) {
                    for (index in 0 until notes.length()) add(notes.optString(index))
                }
            }.filter { it.isNotBlank() },
        )
    }
}
