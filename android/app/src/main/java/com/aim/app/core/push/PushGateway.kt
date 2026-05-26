package com.aim.app.core.push

/**
 * 推送网关预留给后续厂商通道接入。
 * V0.1 测试包只保留契约，不直接依赖 FCM、华为、小米等具体 SDK。
 */
interface PushGateway {
    suspend fun registerDevice(request: PushRegisterRequest): Result<PushRegisterResult>

    suspend fun updateSettings(settings: PushSettings): Result<Unit>
}

data class PushRegisterRequest(
    val userId: Long?,
    val deviceId: String,
    val pushToken: String,
    val provider: PushProvider,
    val appVersionName: String,
)

data class PushRegisterResult(
    val bound: Boolean,
    val serverTokenId: String? = null,
)

data class PushSettings(
    val messageEnabled: Boolean,
    val mentionEnabled: Boolean,
    val aiReplyEnabled: Boolean,
    val quietHoursEnabled: Boolean,
)

enum class PushProvider {
    Fcm,
    Huawei,
    Xiaomi,
    Oppo,
    Vivo,
    Honor,
    Meizu,
    Unknown,
}

object NoopPushGateway : PushGateway {
    override suspend fun registerDevice(request: PushRegisterRequest): Result<PushRegisterResult> =
        Result.success(PushRegisterResult(bound = false))

    override suspend fun updateSettings(settings: PushSettings): Result<Unit> =
        Result.success(Unit)
}
