package com.lanline.app.feature.profile

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
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
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
import com.lanline.app.core.app.LanLineBuildInfo
import com.lanline.app.core.auth.AuthSession
import com.lanline.app.core.realtime.RealtimeConnectionState
import com.lanline.app.core.ui.LanLineAvatar
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLineMainScaffold
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.core.update.AppUpdateInfo
import com.lanline.app.core.update.UpdateRequirement
import com.lanline.app.model.LanLineRoute

@Composable
fun ProfileScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    session: AuthSession?,
    realtimeState: RealtimeConnectionState,
    updateInfo: AppUpdateInfo?,
) {
    var notificationEnabled by rememberSaveable { mutableStateOf(true) }

    LanLineMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                LanLineTopBar(title = "设置", subtitle = "账号、通知与版本")
                ProfileHeader(session = session)
            }
            item {
                SettingRow(
                    icon = Icons.Default.Notifications,
                    title = "消息提醒",
                    subtitle = "收到实时消息时显示本地通知",
                    trailing = {
                        Switch(
                            checked = notificationEnabled,
                            onCheckedChange = { notificationEnabled = it },
                            colors = SwitchDefaults.colors(checkedThumbColor = LanLineColors.Primary),
                        )
                    },
                )
            }
            item {
                SettingRow(
                    icon = Icons.Default.SystemUpdate,
                    title = "版本更新",
                    subtitle = updateInfo.describeUpdate(),
                )
            }
            item {
                SettingRow(
                    icon = Icons.Default.Sync,
                    title = "连接状态",
                    subtitle = "WebSocket ${realtimeState.label} · 当前 ${LanLineBuildInfo.VersionName}",
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
        colors = CardDefaults.cardColors(containerColor = LanLineColors.Surface),
        border = BorderStroke(1.dp, LanLineColors.Line),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            LanLineAvatar(
                text = session?.user?.displayName?.take(1).orEmpty().ifBlank { "我" },
                color = LanLineColors.Primary,
                background = LanLineColors.PrimarySoft,
                online = true,
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = session?.user?.displayName ?: "未登录",
                    color = LanLineColors.Text,
                    fontSize = 17.sp,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = session?.user?.username ?: "LanLine",
                    color = LanLineColors.Muted,
                    fontSize = 12.sp,
                )
            }
            Icon(Icons.Default.Person, contentDescription = null, tint = LanLineColors.Subtle)
        }
    }
}

@Composable
private fun SettingRow(
    icon: ImageVector,
    title: String,
    subtitle: String,
    trailing: @Composable (() -> Unit)? = null,
) {
    Card(
        shape = RoundedCornerShape(8.dp),
        colors = CardDefaults.cardColors(containerColor = LanLineColors.Surface),
        border = BorderStroke(1.dp, LanLineColors.Line),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 14.dp, vertical = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Icon(icon, contentDescription = null, tint = LanLineColors.Primary)
            Column(modifier = Modifier.weight(1f)) {
                Text(title, color = LanLineColors.Text, fontSize = 14.sp, fontWeight = FontWeight.Bold)
                Text(subtitle, color = LanLineColors.Muted, fontSize = 11.sp, lineHeight = 16.sp)
            }
            trailing?.invoke()
        }
    }
}

private fun AppUpdateInfo?.describeUpdate(): String {
    if (this == null) return "正在检查版本 · 当前 ${LanLineBuildInfo.VersionName}"
    if (!hasUpdate) return "已是最新 · 当前 ${LanLineBuildInfo.VersionName}"
    val requirementText = when (requirement) {
        UpdateRequirement.Required -> "强制更新"
        UpdateRequirement.Recommended -> "建议更新"
        UpdateRequirement.Optional -> "可选更新"
    }
    return "$requirementText · 最新 $latestVersionName"
}
