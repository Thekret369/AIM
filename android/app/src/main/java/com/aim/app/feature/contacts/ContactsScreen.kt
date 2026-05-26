package com.aim.app.feature.contacts

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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material.icons.filled.PersonAdd
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.aim.app.core.ui.AimAvatar
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimMainScaffold
import com.aim.app.core.ui.AimTopBar
import com.aim.app.core.ui.FilterChipRow
import com.aim.app.core.ui.SearchBox
import com.aim.app.model.AimRoute

private data class ContactUi(
    val id: Long,
    val name: String,
    val avatar: String,
    val subtitle: String,
    val group: String,
    val online: Boolean,
)

private val contactItems = listOf(
    ContactUi(1, "蓝莓", "蓝", "产品确认中 · 2 条未读", "好友", true),
    ContactUi(2, "王工", "王", "WebSocket 联调", "好友", true),
    ContactUi(3, "AIM 项目组", "项", "8 人 · 后端测试服", "群聊", false),
    ContactUi(4, "测试公告", "测", "只读通知频道", "群聊", false),
    ContactUi(5, "AIM 助手", "AI", "可在聊天内唤起", "AI", true),
)

@Composable
fun ContactsScreen(
    currentRoute: AimRoute,
    onNavigate: (AimRoute) -> Unit,
    onOpenChat: () -> Unit,
) {
    var selected by rememberSaveable { mutableStateOf("全部") }
    val contacts = contactItems.filter { selected == "全部" || it.group == selected }

    AimMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                AimTopBar(
                    title = "联系人",
                    subtitle = "5 个联系人 · 2 人在线",
                    trailing = {
                        IconButton(onClick = {}) {
                            Icon(Icons.Default.PersonAdd, contentDescription = "添加联系人")
                        }
                    },
                )
                SearchBox("搜索好友、群聊或 AIM 助手")
                Spacer(Modifier.height(14.dp))
                FilterChipRow(
                    items = listOf("全部", "好友", "群聊", "AI"),
                    selected = selected,
                    aiChip = "AI",
                    onSelected = { selected = it },
                )
            }
            items(contacts, key = { it.id }) { contact ->
                ContactRow(contact = contact, onClick = onOpenChat)
            }
            item { Spacer(Modifier.height(88.dp)) }
        }
    }
}

@Composable
private fun ContactRow(contact: ContactUi, onClick: () -> Unit) {
    Card(
        onClick = onClick,
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(8.dp),
        border = BorderStroke(1.dp, AimColors.Line),
        colors = CardDefaults.cardColors(containerColor = AimColors.Surface),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(72.dp)
                .padding(horizontal = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            AimAvatar(
                text = contact.avatar,
                online = contact.online,
                color = if (contact.group == "AI") AimColors.Ai else AimColors.Primary,
                background = if (contact.group == "AI") AimColors.AiSoft else AimColors.PrimarySoft,
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = contact.name,
                    color = AimColors.Text,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = contact.subtitle,
                    color = AimColors.Muted,
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Icon(
                imageVector = if (contact.group == "好友") Icons.Default.Chat else Icons.Default.ChevronRight,
                contentDescription = null,
                tint = AimColors.Subtle,
            )
        }
    }
}
