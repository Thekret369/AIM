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
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Done
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
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
import com.lanline.app.model.ContactUi
import com.lanline.app.model.FriendRequestUi
import com.lanline.app.model.LanLineRoute

@Composable
fun ContactsScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    contacts: List<ContactUi>,
    pendingFriends: List<FriendRequestUi>,
    backendStatus: String,
    onOpenChat: (ContactUi) -> Unit,
    onHandleRequest: (FriendRequestUi, Boolean) -> Unit,
) {
    val friendContacts = contacts.filter { it.group == "好友" }

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
                )
            }
            if (pendingFriends.isNotEmpty()) {
                item {
                    Text("待处理申请", color = LanLineColors.Text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                }
                items(pendingFriends, key = { it.id }) { request ->
                    PendingRequestRow(
                        request = request,
                        onAccept = { onHandleRequest(request, true) },
                        onReject = { onHandleRequest(request, false) },
                    )
                }
            }
            item {
                Text("好友列表", color = LanLineColors.Text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
            }
            if (friendContacts.isEmpty()) {
                item {
                    Text(
                        text = "暂无好友",
                        color = LanLineColors.Muted,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
            items(friendContacts, key = { it.id }) { contact ->
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
                color = LanLineColors.Primary,
                background = LanLineColors.PrimarySoft,
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
            Icon(Icons.AutoMirrored.Filled.Chat, contentDescription = null, tint = LanLineColors.Subtle)
        }
    }
}

@Composable
private fun PendingRequestRow(
    request: FriendRequestUi,
    onAccept: () -> Unit,
    onReject: () -> Unit,
) {
    Card(
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
            LanLineAvatar(text = request.name.take(1).ifBlank { "友" })
            Column(modifier = Modifier.weight(1f)) {
                Text(request.name, color = LanLineColors.Text, fontSize = 15.sp, fontWeight = FontWeight.Bold)
                Text(request.message.ifBlank { "请求添加好友" }, color = LanLineColors.Muted, fontSize = 12.sp)
            }
            IconButton(onClick = onAccept) {
                Icon(Icons.Default.Done, contentDescription = "同意", tint = LanLineColors.Primary)
            }
            IconButton(onClick = onReject) {
                Icon(Icons.Default.Close, contentDescription = "拒绝", tint = LanLineColors.Danger)
            }
        }
    }
}
