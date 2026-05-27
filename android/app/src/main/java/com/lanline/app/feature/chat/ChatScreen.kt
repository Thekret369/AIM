package com.lanline.app.feature.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lanline.app.core.realtime.RealtimeChatMessage
import com.lanline.app.core.ui.LanLineAvatar
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.model.ChatMessageUi
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.MessageStatus

@Composable
fun ChatScreen(
    onBack: () -> Unit,
    conversation: ConversationUi?,
    historyMessages: List<ChatMessageUi>,
    historyStatus: String,
    realtimeMessages: List<RealtimeChatMessage>,
    currentUserId: Long?,
    onSendText: (ConversationUi, String) -> Boolean,
) {
    var failedMessages by remember { mutableStateOf<List<ChatMessageUi>>(emptyList()) }
    var input by rememberSaveable { mutableStateOf("") }
    val listState = rememberLazyListState()
    val liveMessages = remember(realtimeMessages, conversation, currentUserId) {
        realtimeMessages
            .filter { message -> conversation != null && message.belongsTo(conversation, currentUserId) }
            .map { it.toChatMessage(currentUserId) }
    }
    val messages = remember(historyMessages, liveMessages, failedMessages) {
        (historyMessages + liveMessages + failedMessages)
            .distinctBy { it.id }
            .sortedBy { it.id }
    }

    LaunchedEffect(conversation?.type, conversation?.id) {
        failedMessages = emptyList()
        input = ""
    }

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.lastIndex)
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(LanLineColors.Background),
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            LanLineTopBar(
                title = conversation?.title ?: "会话",
                subtitle = historyStatus,
                leading = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "返回")
                    }
                },
                trailing = {
                    IconButton(onClick = {}) {
                        Icon(Icons.Default.MoreVert, contentDescription = "更多")
                    }
                },
            )
            LazyColumn(
                state = listState,
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 20.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                if (messages.isEmpty()) {
                    item {
                        Text(
                            text = if (conversation == null) "请先选择会话" else "暂无消息",
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 24.dp),
                            color = LanLineColors.Muted,
                            fontSize = 13.sp,
                            textAlign = TextAlign.Center,
                        )
                    }
                }
                items(messages, key = { "${it.id}:${it.fromMe}" }) { message ->
                    MessageBubble(message)
                }
                item { Spacer(Modifier.height(12.dp)) }
            }
            ChatComposer(
                value = input,
                enabled = conversation != null && conversation.type != ConversationType.Broadcast,
                onValueChange = { input = it },
                onSend = {
                    val activeConversation = conversation
                    val text = input.trim()
                    if (activeConversation != null && text.isNotEmpty()) {
                        val sent = onSendText(activeConversation, text)
                        if (sent) {
                            input = ""
                        } else {
                            failedMessages = failedMessages + ChatMessageUi(
                                id = System.currentTimeMillis(),
                                content = text,
                                fromMe = true,
                                timeText = "刚刚",
                                status = MessageStatus.Failed,
                            )
                        }
                    }
                },
            )
        }
    }
}

@Composable
private fun MessageBubble(message: ChatMessageUi) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = if (message.fromMe) Arrangement.End else Arrangement.Start,
        verticalAlignment = Alignment.Top,
    ) {
        if (!message.fromMe) {
            LanLineAvatar(
                text = if (message.isAi) "AI" else "L",
                modifier = Modifier.size(34.dp),
                color = if (message.isAi) LanLineColors.Ai else LanLineColors.Accent,
                background = if (message.isAi) LanLineColors.AiSoft else LanLineColors.AccentSoft,
            )
            Spacer(Modifier.width(10.dp))
        }
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .background(
                    when {
                        message.fromMe -> LanLineColors.Primary
                        message.isAi -> LanLineColors.AiSoft
                        else -> LanLineColors.Surface
                    },
                )
                .border(
                    width = if (message.fromMe) 0.dp else 1.dp,
                    color = if (message.isAi) Color(0xFFB9D8EA) else LanLineColors.Line,
                    shape = RoundedCornerShape(8.dp),
                )
                .padding(12.dp),
        ) {
            Column {
                if (message.isImage) {
                    Text("图片消息", color = if (message.fromMe) LanLineColors.Surface else LanLineColors.Text, fontWeight = FontWeight.Bold)
                } else {
                    Text(
                        text = message.content,
                        color = if (message.fromMe) LanLineColors.Surface else LanLineColors.Text,
                        fontSize = 13.sp,
                        lineHeight = 20.sp,
                    )
                }
                Text(
                    text = if (message.fromMe) "${message.status.label} ${message.timeText}" else message.timeText,
                    modifier = Modifier.padding(top = 6.dp),
                    color = if (message.fromMe) Color(0xFFE0F3F0) else LanLineColors.Subtle,
                    fontSize = 10.sp,
                    textAlign = if (message.fromMe) TextAlign.End else TextAlign.Start,
                )
            }
        }
    }
}

@Composable
private fun ChatComposer(
    value: String,
    enabled: Boolean,
    onValueChange: (String) -> Unit,
    onSend: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(LanLineColors.Surface)
            .imePadding()
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            enabled = enabled,
            modifier = Modifier.weight(1f),
            placeholder = { Text(if (enabled) "输入消息" else "当前会话不可发送") },
            singleLine = true,
            shape = RoundedCornerShape(8.dp),
        )
        Button(
            onClick = onSend,
            enabled = enabled,
            modifier = Modifier
                .width(62.dp)
                .height(48.dp),
            shape = RoundedCornerShape(8.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = LanLineColors.Primary,
                contentColor = LanLineColors.Surface,
            ),
            contentPadding = PaddingValues(0.dp),
        ) {
            Text("发送", fontSize = 13.sp, fontWeight = FontWeight.Bold)
        }
    }
}

private fun RealtimeChatMessage.belongsTo(conversation: ConversationUi, currentUserId: Long?): Boolean =
    when (conversation.type) {
        ConversationType.Group -> groupId == conversation.id
        ConversationType.User, ConversationType.Ai -> {
            val peerId = if (currentUserId != null && fromUserId == currentUserId) toUserId else fromUserId
            peerId == conversation.id
        }
        ConversationType.Broadcast -> groupId == null && toUserId == null
    }

private fun RealtimeChatMessage.toChatMessage(currentUserId: Long?): ChatMessageUi =
    ChatMessageUi(
        id = if (id > 0) id else receivedAtEpochMillis,
        content = preview,
        fromMe = currentUserId != null && fromUserId == currentUserId,
        timeText = createdAt.toShortTime().ifBlank { "实时" },
        status = MessageStatus.Sent,
        isAi = conversationTitle.contains("AI", ignoreCase = true),
        isImage = type == "image",
    )

private fun String.toShortTime(): String =
    when {
        length >= 16 -> substring(11, 16)
        isNotBlank() -> this
        else -> ""
    }
