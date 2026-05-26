package com.lanline.app.core.realtime

enum class RealtimeConnectionState(val label: String) {
    Idle("未连接"),
    Connecting("连接中"),
    Connected("已连接"),
    Reconnecting("重连中"),
    Closed("已断开"),
    Failed("连接失败"),
}

data class RealtimeChatMessage(
    val id: Long,
    val type: String,
    val fromUserId: Long,
    val fromName: String,
    val toUserId: Long?,
    val groupId: Long?,
    val content: String,
    val fileName: String,
    val createdAt: String,
    val receivedAtEpochMillis: Long,
) {
    val conversationTitle: String
        get() = when {
            groupId != null && groupId > 0 -> "群聊 #$groupId"
            fromName.isNotBlank() -> fromName
            fromUserId > 0 -> "用户 #$fromUserId"
            else -> "LanLine 消息"
        }

    val preview: String
        get() = content.ifBlank { fileName.ifBlank { "收到一条${type}消息" } }
}
