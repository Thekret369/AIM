package com.lanline.app.model

import androidx.compose.ui.graphics.Color
import com.lanline.app.core.ui.LanLineColors

enum class LanLineRoute {
    Setup,
    Login,
    Chats,
    Chat,
    Ai,
    Contacts,
    Profile,
}

enum class ConversationType {
    User,
    Group,
    Ai,
    Broadcast,
}

enum class MessageStatus(val label: String) {
    Sending("发送中"),
    Sent("已发送"),
    Read("已读"),
    Failed("发送失败"),
}

data class ConversationUi(
    val id: Long,
    val type: ConversationType,
    val title: String,
    val avatarText: String,
    val lastMessage: String,
    val timeText: String,
    val unreadCount: Int = 0,
    val online: Boolean = false,
    val accent: Color = LanLineColors.Primary,
    val accentSoft: Color = LanLineColors.PrimarySoft,
)

data class ChatMessageUi(
    val id: Long,
    val content: String,
    val fromMe: Boolean,
    val timeText: String,
    val status: MessageStatus = MessageStatus.Sent,
    val isAi: Boolean = false,
    val isImage: Boolean = false,
)

object LanLineSampleData {
    val conversations = listOf(
        ConversationUi(
            id = 1,
            type = ConversationType.User,
            title = "蓝莓",
            avatarText = "蓝",
            lastMessage = "图片已收到，晚上继续确认。",
            timeText = "23:18",
            unreadCount = 2,
            online = true,
            accent = LanLineColors.Accent,
            accentSoft = LanLineColors.AccentSoft,
        ),
        ConversationUi(
            id = 2,
            type = ConversationType.Group,
            title = "LanLine 项目组",
            avatarText = "项",
            lastMessage = "@你 后端测试服已经上线。",
            timeText = "22:40",
            unreadCount = 5,
        ),
        ConversationUi(
            id = 3,
            type = ConversationType.Ai,
            title = "LanLine 助手",
            avatarText = "AI",
            lastMessage = "可以，我会根据接口文档整理。",
            timeText = "21:06",
            unreadCount = 1,
            accent = LanLineColors.Ai,
            accentSoft = LanLineColors.AiSoft,
        ),
        ConversationUi(
            id = 4,
            type = ConversationType.User,
            title = "王工",
            avatarText = "王",
            lastMessage = "WebSocket 断线重连我再跑一轮。",
            timeText = "昨天",
            online = true,
            accent = LanLineColors.Warning,
            accentSoft = Color(0xFFFFF4E6),
        ),
        ConversationUi(
            id = 5,
            type = ConversationType.Broadcast,
            title = "测试公告",
            avatarText = "测",
            lastMessage = "本周包体只验证 Compose UI。",
            timeText = "周日",
            accent = LanLineColors.Muted,
            accentSoft = LanLineColors.SurfaceAlt,
        ),
    )

    val chatMessages = listOf(
        ChatMessageUi(1, "测试包打开后，聊天列表和输入框都正常。", false, "23:13"),
        ChatMessageUi(2, "我这边再验证一下文件上传和返回键。", true, "23:15", MessageStatus.Read),
        ChatMessageUi(3, "图片选择器能调起，但 Android 13 以上需要单独测权限。", false, "23:17"),
        ChatMessageUi(4, "上传预览图", true, "23:18", MessageStatus.Read, isImage = true),
        ChatMessageUi(5, "收到，先以测试包可用为准。", true, "23:20", MessageStatus.Read),
    )
}
