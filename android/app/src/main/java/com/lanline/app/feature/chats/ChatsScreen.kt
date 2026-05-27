package com.lanline.app.feature.chats

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.lanline.app.core.realtime.RealtimeChatMessage
import com.lanline.app.core.realtime.RealtimeConnectionState
import com.lanline.app.core.ui.ConversationRow
import com.lanline.app.core.ui.FilterChipRow
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLineMainScaffold
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.LanLineRoute

@Composable
fun ChatsScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    realtimeState: RealtimeConnectionState,
    realtimeMessages: List<RealtimeChatMessage>,
    currentUserId: Long?,
    conversations: List<ConversationUi>,
    backendStatus: String,
    onOpenChat: (ConversationUi) -> Unit,
) {
    var selected by rememberSaveable { mutableStateOf("全部") }
    val liveConversations = realtimeMessages
        .groupBy { it.conversationKey(currentUserId) }
        .values
        .mapNotNull { messages ->
            val latest = messages.maxByOrNull { it.sortValue }
            latest?.toConversation(currentUserId, messages.count { it.fromUserId != currentUserId })
        }
    val sourceConversations = mergeConversations(conversations, liveConversations)
        .sortedWith(compareByDescending<ConversationUi> { it.type == ConversationType.Broadcast }.thenByDescending { it.sortEpochMillis })
    val filteredConversations = sourceConversations.filter {
        when (selected) {
            "未读" -> it.unreadCount > 0
            "群聊" -> it.type == ConversationType.Group
            "AI" -> it.type == ConversationType.Ai
            else -> true
        }
    }

    LanLineMainScaffold(
        currentRoute = currentRoute,
        onNavigate = onNavigate,
    ) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LanLineTopBar(
                    title = "消息",
                    subtitle = "${realtimeState.label} · $backendStatus",
                )
                FilterChipRow(
                    items = listOf("全部", "未读", "群聊", "AI"),
                    selected = selected,
                    aiChip = "AI",
                    onSelected = { selected = it },
                )
            }
            if (filteredConversations.isEmpty()) {
                item {
                    Text(
                        text = "暂无会话",
                        color = LanLineColors.Muted,
                        modifier = Modifier.padding(top = 18.dp),
                    )
                }
            }
            items(filteredConversations, key = { "${it.type}:${it.id}" }) { conversation ->
                ConversationRow(
                    conversation = conversation,
                    onClick = { onOpenChat(conversation) },
                )
            }
            item { Spacer(Modifier.height(88.dp)) }
        }
    }
}

private fun mergeConversations(
    baseConversations: List<ConversationUi>,
    liveConversations: List<ConversationUi>,
): List<ConversationUi> {
    val merged = linkedMapOf<String, ConversationUi>()
    baseConversations.forEach { conversation ->
        merged[conversation.conversationKey()] = conversation
    }
    liveConversations.forEach { live ->
        val key = live.conversationKey()
        val base = merged[key]
        merged[key] = if (base == null) {
            live
        } else {
            base.copy(
                lastMessage = live.lastMessage,
                timeText = live.timeText,
                sortEpochMillis = live.sortEpochMillis,
                unreadCount = live.unreadCount,
                online = base.online || live.online,
            )
        }
    }
    return merged.values.toList()
}

private fun RealtimeChatMessage.toConversation(currentUserId: Long?, unreadCount: Int): ConversationUi =
    ConversationUi(
        id = conversationTargetId(currentUserId),
        type = if (groupId != null && groupId > 0) ConversationType.Group else ConversationType.User,
        title = displayTitle(currentUserId),
        avatarText = displayTitle(currentUserId).take(1).ifBlank { "L" },
        lastMessage = preview,
        timeText = createdAt.toShortTime().ifBlank { "刚刚" },
        sortEpochMillis = sortValue,
        unreadCount = unreadCount,
        online = true,
        accent = LanLineColors.Accent,
        accentSoft = LanLineColors.AccentSoft,
    )

private val RealtimeChatMessage.sortValue: Long
    get() = id.takeIf { it > 0 } ?: receivedAtEpochMillis

private fun RealtimeChatMessage.conversationTargetId(currentUserId: Long?): Long =
    when {
        groupId != null && groupId > 0 -> groupId
        currentUserId != null && fromUserId == currentUserId -> toUserId ?: 0L
        else -> fromUserId
    }

private fun RealtimeChatMessage.conversationKey(currentUserId: Long?): String =
    when {
        groupId != null && groupId > 0 -> "group:$groupId"
        toUserId == null -> "broadcast"
        else -> "user:${conversationTargetId(currentUserId)}"
    }

private fun RealtimeChatMessage.displayTitle(currentUserId: Long?): String =
    when {
        groupId != null && groupId > 0 -> conversationTitle
        currentUserId != null && fromUserId == currentUserId -> "用户 #${conversationTargetId(currentUserId)}"
        else -> conversationTitle
    }

private fun ConversationUi.conversationKey(): String =
    when (type) {
        ConversationType.Group -> "group:$id"
        ConversationType.Broadcast -> "broadcast"
        ConversationType.User, ConversationType.Ai -> "user:$id"
    }

private fun String.toShortTime(): String =
    when {
        length >= 16 -> substring(11, 16)
        isNotBlank() -> this
        else -> ""
    }
