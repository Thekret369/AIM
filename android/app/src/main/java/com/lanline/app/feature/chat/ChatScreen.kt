package com.lanline.app.feature.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.CameraAlt
import androidx.compose.material.icons.filled.Image
import androidx.compose.material.icons.filled.InsertDriveFile
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
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
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lanline.app.core.realtime.RealtimeChatMessage
import com.lanline.app.core.ui.LanLineAvatar
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLinePrimaryButton
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.model.ChatMessageUi
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.MessageStatus

@OptIn(ExperimentalMaterial3Api::class)
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
    var localMessages by remember { mutableStateOf<List<ChatMessageUi>>(emptyList()) }
    var input by rememberSaveable { mutableStateOf("") }
    var showAttachmentSheet by rememberSaveable { mutableStateOf(false) }
    val listState = rememberLazyListState()
    val liveMessages = remember(realtimeMessages, conversation, currentUserId) {
        realtimeMessages
            .filter { message -> conversation != null && message.belongsTo(conversation, currentUserId) }
            .map { it.toChatMessage(currentUserId) }
    }
    val messages = remember(historyMessages, liveMessages, localMessages) {
        (historyMessages + liveMessages + localMessages).distinctBy { it.id }
    }

    LaunchedEffect(conversation?.type, conversation?.id) {
        localMessages = emptyList()
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
                item {
                    Text(
                        text = if (conversation == null) "未选择会话" else "后端历史 + 实时消息",
                        modifier = Modifier.fillMaxWidth(),
                        color = LanLineColors.Subtle,
                        fontSize = 11.sp,
                        textAlign = TextAlign.Center,
                    )
                }
                if (messages.isEmpty()) {
                    item {
                        Text(
                            text = if (conversation == null) {
                                "请先从消息或联系人列表进入一个会话。"
                            } else {
                                "暂无后端消息，发送或收到 WebSocket 消息后会显示在这里。"
                            },
                            modifier = Modifier.fillMaxWidth(),
                            color = LanLineColors.Muted,
                            fontSize = 13.sp,
                            textAlign = TextAlign.Center,
                        )
                    }
                }
                items(messages, key = { it.id }) { message ->
                    MessageBubble(message)
                }
                item { Spacer(Modifier.height(12.dp)) }
            }
            ChatComposer(
                value = input,
                enabled = conversation != null && conversation.type != ConversationType.Broadcast,
                onValueChange = { input = it },
                onAttachment = { showAttachmentSheet = true },
                onSend = {
                    val activeConversation = conversation ?: return@ChatComposer
                    val text = input.trim()
                    if (text.isNotEmpty()) {
                        val sent = onSendText(activeConversation, text)
                        localMessages = localMessages + ChatMessageUi(
                            id = System.currentTimeMillis(),
                            content = text,
                            fromMe = true,
                            timeText = "刚刚",
                            status = if (sent) MessageStatus.Sent else MessageStatus.Failed,
                        )
                        if (sent) input = ""
                    }
                },
            )
        }

        if (showAttachmentSheet) {
            ModalBottomSheet(
                onDismissRequest = { showAttachmentSheet = false },
                sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
                containerColor = LanLineColors.Surface,
            ) {
                AttachmentSheetContent(
                    onSelected = { showAttachmentSheet = false },
                    onCancel = { showAttachmentSheet = false },
                )
            }
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
                    Box(
                        modifier = Modifier
                            .size(width = 180.dp, height = 112.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .background(LanLineColors.AccentSoft),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text("图片消息", color = LanLineColors.Accent, fontWeight = FontWeight.Bold)
                    }
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
    onAttachment: () -> Unit,
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
        IconButton(
            onClick = onAttachment,
            enabled = enabled,
            modifier = Modifier
                .size(42.dp)
                .border(1.dp, LanLineColors.Line, RoundedCornerShape(8.dp)),
        ) {
            Icon(Icons.Default.Add, contentDescription = "添加附件", tint = LanLineColors.Primary)
        }
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            enabled = enabled,
            modifier = Modifier.weight(1f),
            placeholder = { Text(if (enabled) "输入消息" else "当前会话暂不可发送") },
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

@Composable
private fun AttachmentSheetContent(
    onSelected: () -> Unit,
    onCancel: () -> Unit,
) {
    Column(
        modifier = Modifier.padding(horizontal = 24.dp, vertical = 8.dp),
    ) {
        Text("添加到聊天", color = LanLineColors.Text, fontSize = 17.sp, fontWeight = FontWeight.Bold)
        Text(
            text = "图片、拍照、文件和语音入口已保留，文件上传使用 /api/upload 接口。",
            color = LanLineColors.Muted,
            fontSize = 12.sp,
            modifier = Modifier.padding(top = 8.dp, bottom = 18.dp),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            UploadAction("相册", "图片消息", Icons.Default.Image, Modifier.weight(1f), onSelected)
            UploadAction("拍照", "调起相机", Icons.Default.CameraAlt, Modifier.weight(1f), onSelected)
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            UploadAction("文件", "走上传接口", Icons.Default.InsertDriveFile, Modifier.weight(1f), onSelected)
            UploadAction("语音", "入口保留", Icons.Default.Mic, Modifier.weight(1f), onSelected)
        }
        Spacer(Modifier.height(18.dp))
        LanLinePrimaryButton(text = "取消", dark = true, onClick = onCancel)
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun UploadAction(
    title: String,
    subtitle: String,
    icon: ImageVector,
    modifier: Modifier,
    onClick: () -> Unit,
) {
    Row(
        modifier = modifier
            .height(64.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(LanLineColors.Background)
            .border(1.dp, LanLineColors.Line, RoundedCornerShape(8.dp))
            .padding(10.dp)
            .clickableNoRipple(onClick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = title, tint = LanLineColors.Primary, modifier = Modifier.size(24.dp))
        Spacer(Modifier.width(10.dp))
        Column {
            Text(title, color = LanLineColors.Text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
            Text(subtitle, color = LanLineColors.Muted, fontSize = 11.sp)
        }
    }
}

private fun Modifier.clickableNoRipple(onClick: () -> Unit): Modifier =
    this.clickable(onClick = onClick)

private fun RealtimeChatMessage.belongsTo(conversation: ConversationUi, currentUserId: Long?): Boolean =
    when (conversation.type) {
        ConversationType.Group -> groupId == conversation.id
        ConversationType.User, ConversationType.Ai ->
            fromUserId == conversation.id ||
                toUserId == conversation.id ||
                (currentUserId != null && fromUserId == currentUserId && toUserId == conversation.id)
        ConversationType.Broadcast -> groupId == null && toUserId == null
    }

private fun RealtimeChatMessage.toChatMessage(currentUserId: Long?): ChatMessageUi =
    ChatMessageUi(
        id = if (id > 0) RealtimeMessageIdOffset + id else receivedAtEpochMillis,
        content = preview,
        fromMe = currentUserId != null && fromUserId == currentUserId,
        timeText = "实时",
        status = MessageStatus.Sent,
        isAi = conversationTitle.contains("AI", ignoreCase = true),
        isImage = type == "image",
    )

private const val RealtimeMessageIdOffset = 1_000_000_000L
