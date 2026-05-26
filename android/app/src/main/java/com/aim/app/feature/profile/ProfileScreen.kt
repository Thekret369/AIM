package com.aim.app.feature.profile

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Sync
import androidx.compose.material.icons.filled.SystemUpdate
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.aim.app.core.auth.AuthSession
import com.aim.app.core.ui.AimAvatar
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimMainScaffold
import com.aim.app.core.ui.AimTopBar
import com.aim.app.model.AimRoute

@Composable
fun ProfileScreen(
    currentRoute: AimRoute,
    onNavigate: (AimRoute) -> Unit,
    session: AuthSession?,
) {
    var pushEnabled by rememberSaveable { mutableStateOf(true) }
    var betaUpdates by rememberSaveable { mutableStateOf(true) }

    AimMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                AimTopBar(title = "我的", subtitle = "测试包 · 内部渠道")
                ProfileHeader(session = session)
            }
            item {
                ToggleRow(
                    icon = Icons.Default.Notifications,
                    title = "消息推送",
                    subtitle = "接口已预留，后续接入厂商 Push token 上报",
                    checked = pushEnabled,
                    onCheckedChange = { pushEnabled = it },
                )
            }
            item {
                ToggleRow(
                    icon = Icons.Default.SystemUpdate,
                    title = "测试包更新",
                    subtitle = "接口已预留，后续返回 APK 地址和强更策略",
                    checked = betaUpdates,
                    onCheckedChange = { betaUpdates = it },
                )
            }
            item {
                LinkRow(
                    icon = Icons.Default.Sync,
                    title = "同步状态",
                    subtitle = "本地 UI 演示 · 暂未连接真实后端",
                )
            }
            item { Spacer(Modifier.height(88.dp)) }
        }
    }
}

@Composable
private fun ProfileHeader(session: AuthSession?) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = AimColors.Surface),
        border = BorderStroke(1.dp, AimColors.Line),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            AimAvatar(text = "测", color = AimColors.Primary, background = AimColors.PrimarySoft, online = true)
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = session?.user?.displayName ?: "测试账号",
                    color = AimColors.Text,
                    fontSize = 17.sp,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = session?.user?.username ?: "com.aim.app.debug · v0.2.0-debug",
                    color = AimColors.Muted,
                    fontSize = 12.sp,
                )
            }
        }
    }
}

@Composable
private fun ToggleRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = AimColors.Surface),
        border = BorderStroke(1.dp, AimColors.Line),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(icon, contentDescription = null, tint = AimColors.Primary)
            Column(modifier = Modifier.weight(1f)) {
                Text(title, color = AimColors.Text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                Text(subtitle, color = AimColors.Muted, fontSize = 11.sp, lineHeight = 16.sp)
            }
            Switch(
                checked = checked,
                onCheckedChange = onCheckedChange,
                colors = SwitchDefaults.colors(checkedThumbColor = AimColors.Primary),
            )
        }
    }
}

@Composable
private fun LinkRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
) {
    Card(
        onClick = {},
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = AimColors.Surface),
        border = BorderStroke(1.dp, AimColors.Line),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 15.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(icon, contentDescription = null, tint = AimColors.Primary)
            Column(modifier = Modifier.weight(1f)) {
                Text(title, color = AimColors.Text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                Text(subtitle, color = AimColors.Muted, fontSize = 11.sp)
            }
            Icon(Icons.Default.ChevronRight, contentDescription = null, tint = AimColors.Subtle)
        }
    }
}
