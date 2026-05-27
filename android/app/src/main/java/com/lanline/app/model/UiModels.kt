package com.lanline.app.model

import androidx.compose.ui.graphics.Color
import com.lanline.app.core.ui.LanLineColors

enum class LanLineRoute {
    Login,
    Chats,
    Chat,
    Contacts,
    Groups,
    Ai,
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
    val sortEpochMillis: Long = 0L,
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

data class ContactUi(
    val id: Long,
    val name: String,
    val avatar: String,
    val subtitle: String,
    val group: String,
    val online: Boolean,
)

data class GroupUi(
    val id: Long,
    val name: String,
    val avatar: String,
    val description: String,
    val memberCount: Int,
    val doNotDisturb: Boolean,
)

data class FriendRequestUi(
    val id: Long,
    val userId: Long,
    val name: String,
    val message: String,
)

data class BackendSnapshot(
    val conversations: List<ConversationUi> = emptyList(),
    val contacts: List<ContactUi> = emptyList(),
    val groups: List<GroupUi> = emptyList(),
    val pendingFriends: List<FriendRequestUi> = emptyList(),
    val statusText: String = "未连接后端",
)
