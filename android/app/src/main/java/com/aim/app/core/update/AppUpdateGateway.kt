package com.aim.app.core.update

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
