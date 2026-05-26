package com.aim.app.feature.chat

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
import com.aim.app.core.ui.AimAvatar
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimPrimaryButton
import com.aim.app.core.ui.AimTopBar
import com.aim.app.model.AimSampleData
import com.aim.app.model.ChatMessageUi
import com.aim.app.model.MessageStatus

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatScreen(onBack: () -> Unit) {
    var messages by remember { mutableStateOf(AimSampleData.chatMessages) }
    var input by rememberSaveable { mutableStateOf("") }
    var showAttachmentSheet by rememberSaveable { mutableStateOf(false) }
    val listState = rememberLazyListState()

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.lastIndex)
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(AimColors.Background),
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            AimTopBar(
                title = "蓝莓",
                subtitle = "在线 · AIM Android 测试",
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
                        text = "今天 23:12",
                        modifier = Modifier.fillMaxWidth(),
                        color = AimColors.Subtle,
                        fontSize = 11.sp,
                        textAlign = TextAlign.Center,
                    )
                }
                items(messages, key = { it.id }) { message ->
                    MessageBubble(message)
                }
                item { Spacer(Modifier.height(12.dp)) }
            }
            ChatComposer(
                value = input,
                onValueChange = { input = it },
                onAttachment = { showAttachmentSheet = true },
                onSend = {
                    val text = input.trim()
                    if (text.isNotEmpty()) {
                        messages = messages + ChatMessageUi(
                            id = System.currentTimeMillis(),
                            content = text,
                            fromMe = true,
                            timeText = "刚刚",
                            status = MessageStatus.Read,
                        )
                        input = ""
                    }
                },
            )
        }

        if (showAttachmentSheet) {
            ModalBottomSheet(
                onDismissRequest = { showAttachmentSheet = false },
                sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true),
                containerColor = AimColors.Surface,
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
            AimAvatar(
                text = if (message.isAi) "AI" else "蓝",
                modifier = Modifier.size(34.dp),
                color = if (message.isAi) AimColors.Ai else AimColors.Accent,
                background = if (message.isAi) AimColors.AiSoft else AimColors.AccentSoft,
            )
            Spacer(Modifier.width(10.dp))
        }
        Box(
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .background(
                    when {
                        message.fromMe -> AimColors.Primary
                        message.isAi -> AimColors.AiSoft
                        else -> AimColors.Surface
                    },
                )
                .border(
                    width = if (message.fromMe) 0.dp else 1.dp,
                    color = if (message.isAi) Color(0xFFD8D2FF) else AimColors.Line,
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
                            .background(AimColors.AccentSoft),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text("上传预览图", color = AimColors.Accent, fontWeight = FontWeight.Bold)
                    }
                } else {
                    Text(
                        text = message.content,
                        color = if (message.fromMe) AimColors.Surface else AimColors.Text,
                        fontSize = 13.sp,
                        lineHeight = 20.sp,
                    )
                }
                Text(
                    text = if (message.fromMe) "${message.status.label} ${message.timeText}" else message.timeText,
                    modifier = Modifier.padding(top = 6.dp),
                    color = if (message.fromMe) Color(0xFFD8F5EC) else AimColors.Subtle,
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
    onValueChange: (String) -> Unit,
    onAttachment: () -> Unit,
    onSend: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(AimColors.Surface)
            .imePadding()
            .padding(14.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        IconButton(
            onClick = onAttachment,
            modifier = Modifier
                .size(42.dp)
                .border(1.dp, AimColors.Line, RoundedCornerShape(8.dp)),
        ) {
            Icon(Icons.Default.Add, contentDescription = "添加附件", tint = AimColors.Primary)
        }
        OutlinedTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.weight(1f),
            placeholder = { Text("输入消息") },
            singleLine = true,
            shape = RoundedCornerShape(8.dp),
        )
        Button(
            onClick = onSend,
            modifier = Modifier
                .width(62.dp)
                .height(48.dp),
            shape = RoundedCornerShape(8.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = AimColors.Primary,
                contentColor = AimColors.Surface,
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
        Text("添加到聊天", color = AimColors.Text, fontSize = 17.sp, fontWeight = FontWeight.Bold)
        Text(
            text = "测试包优先支持图片、拍照、文件和语音入口。",
            color = AimColors.Muted,
            fontSize = 12.sp,
            modifier = Modifier.padding(top = 8.dp, bottom = 18.dp),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            UploadAction("相册", "图片消息", Icons.Default.Image, Modifier.weight(1f), onSelected)
            UploadAction("拍照", "调起相机", Icons.Default.CameraAlt, Modifier.weight(1f), onSelected)
        }
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            UploadAction("文件", "最大 50MB", Icons.Default.InsertDriveFile, Modifier.weight(1f), onSelected)
            UploadAction("语音", "后续实现", Icons.Default.Mic, Modifier.weight(1f), onSelected)
        }
        Spacer(Modifier.height(18.dp))
        AimPrimaryButton(text = "取消", dark = true, onClick = onCancel)
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
            .background(AimColors.Background)
            .border(1.dp, AimColors.Line, RoundedCornerShape(8.dp))
            .padding(10.dp)
            .clickableNoRipple(onClick),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Icon(icon, contentDescription = title, tint = AimColors.Primary, modifier = Modifier.size(24.dp))
        Spacer(Modifier.width(10.dp))
        Column {
            Text(title, color = AimColors.Text, fontSize = 13.sp, fontWeight = FontWeight.Bold)
            Text(subtitle, color = AimColors.Muted, fontSize = 11.sp)
        }
    }
}

private fun Modifier.clickableNoRipple(onClick: () -> Unit): Modifier =
    this.clickable(onClick = onClick)
