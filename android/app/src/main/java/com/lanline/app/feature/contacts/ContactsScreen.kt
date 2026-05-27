package com.lanline.app.feature.contacts

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
import com.lanline.app.core.ui.LanLineAvatar
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.core.ui.LanLineMainScaffold
import com.lanline.app.core.ui.LanLineTopBar
import com.lanline.app.core.ui.FilterChipRow
import com.lanline.app.core.ui.SearchBox
import com.lanline.app.model.ContactUi
import com.lanline.app.model.LanLineRoute

@Composable
fun ContactsScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    contacts: List<ContactUi>,
    backendStatus: String,
    onOpenChat: (ContactUi) -> Unit,
) {
    var selected by rememberSaveable { mutableStateOf("全部") }
    val filteredContacts = contacts.filter { selected == "全部" || it.group == selected }

    LanLineMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LanLineTopBar(
                    title = "联系人",
                    subtitle = backendStatus,
                    trailing = {
                        IconButton(onClick = {}) {
                            Icon(Icons.Default.PersonAdd, contentDescription = "添加联系人")
                        }
                    },
                )
                SearchBox("搜索好友、群聊或 LanLine 助手")
                Spacer(Modifier.height(14.dp))
                FilterChipRow(
                    items = listOf("全部", "好友", "群聊", "AI"),
                    selected = selected,
                    aiChip = "AI",
                    onSelected = { selected = it },
                )
            }
            if (filteredContacts.isEmpty()) {
                item {
                    Text(
                        text = "暂无后端联系人数据，请先在网页端或服务端创建好友、群组或 AI 助手。",
                        color = LanLineColors.Muted,
                        modifier = Modifier.padding(top = 18.dp),
                    )
                }
            }
            items(filteredContacts, key = { "${it.group}:${it.id}" }) { contact ->
                ContactRow(contact = contact, onClick = { onOpenChat(contact) })
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
        border = BorderStroke(1.dp, LanLineColors.Line),
        colors = CardDefaults.cardColors(containerColor = LanLineColors.Surface),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(72.dp)
                .padding(horizontal = 13.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            LanLineAvatar(
                text = contact.avatar,
                online = contact.online,
                color = if (contact.group == "AI") LanLineColors.Ai else LanLineColors.Primary,
                background = if (contact.group == "AI") LanLineColors.AiSoft else LanLineColors.PrimarySoft,
            )
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = contact.name,
                    color = LanLineColors.Text,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = contact.subtitle,
                    color = LanLineColors.Muted,
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Icon(
                imageVector = if (contact.group == "好友") Icons.Default.Chat else Icons.Default.ChevronRight,
                contentDescription = null,
                tint = LanLineColors.Subtle,
            )
        }
    }
}
