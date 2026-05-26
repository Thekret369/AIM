package com.lanline.app.feature.ai

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lanline.app.core.ui.LanLineAvatar
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLineMainScaffold
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.core.ui.FilterChipRow
import com.lanline.app.model.LanLineRoute

@Composable
fun AiScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    onBack: () -> Unit,
) {
    var selected by rememberSaveable { mutableStateOf("生成测试用例") }
    var input by rememberSaveable { mutableStateOf("") }

    LanLineMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        Column(modifier = Modifier.padding(padding)) {
            LanLineTopBar(
                title = "LanLine 助手",
                subtitle = "已连接自部署模型 · 上下文 12 条",
                leading = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "返回")
                    }
                },
                trailing = {
                    IconButton(onClick = {}) {
                        Icon(Icons.Default.Settings, contentDescription = "AI 设置")
                    }
                },
            )
            LazyColumn(
                modifier = Modifier
                    .weight(1f)
                    .padding(horizontal = 20.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
                item {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .background(LanLineColors.AiSoft)
                            .border(1.dp, Color(0xFFB9D8EA), RoundedCornerShape(8.dp))
                            .padding(16.dp),
                    ) {
                        Text("默认助手", color = LanLineColors.Ai, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                        Text("用于群聊总结、接口排查和部署说明。", color = LanLineColors.Muted, fontSize = 12.sp)
                    }
                }
                item {
                    FilterChipRow(
                        items = listOf("总结聊天", "生成测试用例", "解释报错"),
                        selected = selected,
                        aiChip = selected,
                        onSelected = { selected = it },
                    )
                }
                item {
                    AiMessageBubble(text = "帮我整理一下安卓测试包需要验证的点。", fromMe = true)
                }
                item {
                    Row(verticalAlignment = Alignment.Top) {
                        LanLineAvatar("AI", modifier = Modifier.size(34.dp), color = LanLineColors.Ai, background = LanLineColors.AiSoft)
                        Spacer(Modifier.width(10.dp))
                        AiMessageBubble(
                            text = "建议按登录、会话、单聊、群聊、AI 流式回复、上传、断网重连、返回键八类验证。",
                            fromMe = false,
                        )
                    }
                }
                item {
                    Text(
                        text = "正在继续生成 ···",
                        modifier = Modifier
                            .clip(RoundedCornerShape(8.dp))
                            .background(LanLineColors.Surface)
                            .border(1.dp, LanLineColors.Line, RoundedCornerShape(8.dp))
                            .padding(horizontal = 14.dp, vertical = 12.dp),
                        color = LanLineColors.Ai,
                        fontSize = 13.sp,
                        fontWeight = FontWeight.Bold,
                    )
                }
                item {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .background(LanLineColors.Surface)
                            .border(1.dp, LanLineColors.Line, RoundedCornerShape(8.dp))
                            .padding(16.dp),
                    ) {
                        Text("引用上下文", color = LanLineColors.Text, fontWeight = FontWeight.Bold)
                        Text(
                            "最近 3 条上传失败日志、当前 WebSocket 状态、用户测试服配置。",
                            color = LanLineColors.Muted,
                            fontSize = 12.sp,
                            modifier = Modifier.padding(top = 8.dp),
                        )
                    }
                }
            }
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(LanLineColors.Surface)
                    .padding(14.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                IconButton(onClick = {}) {
                    Icon(Icons.Default.Add, contentDescription = "添加上下文", tint = LanLineColors.Primary)
                }
                OutlinedTextField(
                    value = input,
                    onValueChange = { input = it },
                    modifier = Modifier.weight(1f),
                    placeholder = { Text("向 LanLine 助手提问") },
                    singleLine = true,
                    shape = RoundedCornerShape(8.dp),
                )
                Button(
                    onClick = { input = "" },
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
    }
}

@Composable
private fun AiMessageBubble(text: String, fromMe: Boolean) {
    Box(
        modifier = Modifier
            .fillMaxWidth(),
        contentAlignment = if (fromMe) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Text(
            text = text,
            modifier = Modifier
                .clip(RoundedCornerShape(8.dp))
                .background(if (fromMe) LanLineColors.Primary else LanLineColors.AiSoft)
                .border(
                    width = if (fromMe) 0.dp else 1.dp,
                    color = if (fromMe) Color.Transparent else Color(0xFFB9D8EA),
                    shape = RoundedCornerShape(8.dp),
                )
                .padding(12.dp),
            color = if (fromMe) LanLineColors.Surface else LanLineColors.Text,
            fontSize = 13.sp,
            lineHeight = 20.sp,
        )
    }
}
