package com.lanline.app.feature.setup

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLinePrimaryButton
import com.lanline.app.core.ui.LanLineTextField
import com.lanline.app.core.ui.FilterChipRow
import com.lanline.app.core.config.ServerConfig

@Composable
fun SetupScreen(
    config: ServerConfig,
    onSaved: (ServerConfig) -> Unit,
) {
    var serverUrl by rememberSaveable(config.apiBaseUrl) { mutableStateOf(config.apiBaseUrl) }
    var websocketUrl by rememberSaveable(config.websocketUrl) { mutableStateOf(config.websocketUrl) }
    var environment by rememberSaveable { mutableStateOf("公网测试服") }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(LanLineColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 32.dp, vertical = 72.dp),
    ) {
        Text(
            text = "LL",
            modifier = Modifier
                .background(LanLineColors.Primary, RoundedCornerShape(8.dp))
                .padding(horizontal = 18.dp, vertical = 14.dp),
            color = LanLineColors.Surface,
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(24.dp))
        Text(
            text = "测试包连接配置",
            color = LanLineColors.Text,
            fontSize = 26.sp,
            lineHeight = 34.sp,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = "输入服务器地址，优先验证 API、WebSocket 和上传链路。",
            color = LanLineColors.Muted,
            fontSize = 14.sp,
            lineHeight = 22.sp,
            modifier = Modifier.padding(top = 10.dp),
        )

        Spacer(Modifier.height(128.dp))
        LanLineTextField(
            label = "服务器地址",
            value = serverUrl,
            onValueChange = { serverUrl = it },
        )
        Spacer(Modifier.height(14.dp))
        LanLineTextField(
            label = "WebSocket",
            value = websocketUrl,
            onValueChange = { websocketUrl = it },
        )
        Spacer(Modifier.height(20.dp))
        FilterChipRow(
            items = listOf("公网测试服", "局域网", "HTTPS"),
            selected = environment,
            onSelected = { environment = it },
        )
        Spacer(Modifier.height(20.dp))
        ConnectionCheckCard()
        Spacer(Modifier.height(36.dp))
        LanLinePrimaryButton(
            text = "保存并进入登录",
            onClick = { onSaved(ServerConfig.fromInput(serverUrl, websocketUrl)) },
        )
    }
}

@Composable
private fun ConnectionCheckCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = LanLineColors.Surface),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("连接检查", color = LanLineColors.Text, fontWeight = FontWeight.Bold)
            CheckLine("API /api/login", "待登录", LanLineColors.Warning)
            CheckLine("WebSocket 鉴权", "待登录", LanLineColors.Warning)
            CheckLine("文件上传目录", "待测试", LanLineColors.Warning)
        }
    }
}

@Composable
private fun CheckLine(label: String, value: String, valueColor: androidx.compose.ui.graphics.Color) {
    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, color = LanLineColors.Muted, fontSize = 13.sp)
        Text(value, color = valueColor, fontSize = 13.sp, fontWeight = FontWeight.Bold)
    }
}
