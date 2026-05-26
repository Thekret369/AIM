package com.aim.app.feature.setup

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
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimPrimaryButton
import com.aim.app.core.ui.AimTextField
import com.aim.app.core.ui.FilterChipRow

@Composable
fun SetupScreen(onSaved: () -> Unit) {
    var serverUrl by rememberSaveable { mutableStateOf("http://192.168.1.8:8080") }
    var websocketUrl by rememberSaveable { mutableStateOf("ws://192.168.1.8:8080/ws") }
    var environment by rememberSaveable { mutableStateOf("局域网") }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(AimColors.Background)
            .verticalScroll(rememberScrollState())
            .padding(horizontal = 32.dp, vertical = 72.dp),
    ) {
        Text(
            text = "AI",
            modifier = Modifier
                .background(AimColors.Primary, RoundedCornerShape(8.dp))
                .padding(horizontal = 18.dp, vertical = 14.dp),
            color = AimColors.Surface,
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(24.dp))
        Text(
            text = "测试包连接配置",
            color = AimColors.Text,
            fontSize = 26.sp,
            lineHeight = 34.sp,
            fontWeight = FontWeight.Bold,
        )
        Text(
            text = "输入服务器地址，优先验证 API、WebSocket 和上传链路。",
            color = AimColors.Muted,
            fontSize = 14.sp,
            lineHeight = 22.sp,
            modifier = Modifier.padding(top = 10.dp),
        )

        Spacer(Modifier.height(128.dp))
        AimTextField(
            label = "服务器地址",
            value = serverUrl,
            onValueChange = { serverUrl = it },
        )
        Spacer(Modifier.height(14.dp))
        AimTextField(
            label = "WebSocket",
            value = websocketUrl,
            onValueChange = { websocketUrl = it },
        )
        Spacer(Modifier.height(20.dp))
        FilterChipRow(
            items = listOf("局域网", "测试服", "公网 HTTPS"),
            selected = environment,
            onSelected = { environment = it },
        )
        Spacer(Modifier.height(20.dp))
        ConnectionCheckCard()
        Spacer(Modifier.height(36.dp))
        AimPrimaryButton(text = "保存并进入登录", onClick = onSaved)
    }
}

@Composable
private fun ConnectionCheckCard() {
    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = AimColors.Surface),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Text("连接检查", color = AimColors.Text, fontWeight = FontWeight.Bold)
            CheckLine("API /api/login", "通过", AimColors.Primary)
            CheckLine("WebSocket 鉴权", "待登录", AimColors.Warning)
            CheckLine("文件上传目录", "待测试", AimColors.Warning)
        }
    }
}

@Composable
private fun CheckLine(label: String, value: String, valueColor: androidx.compose.ui.graphics.Color) {
    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, color = AimColors.Muted, fontSize = 13.sp)
        Text(value, color = valueColor, fontSize = 13.sp, fontWeight = FontWeight.Bold)
    }
}
