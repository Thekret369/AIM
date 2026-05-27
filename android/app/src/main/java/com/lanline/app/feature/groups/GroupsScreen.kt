package com.lanline.app.feature.groups

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
import androidx.compose.material.icons.filled.ChevronRight
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
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
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.GroupUi
import com.lanline.app.model.LanLineRoute

@Composable
fun GroupsScreen(
    currentRoute: LanLineRoute,
    onNavigate: (LanLineRoute) -> Unit,
    groups: List<GroupUi>,
    backendStatus: String,
    onOpenGroup: (ConversationUi) -> Unit,
) {
    LanLineMainScaffold(currentRoute = currentRoute, onNavigate = onNavigate) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                LanLineTopBar(title = "群组", subtitle = backendStatus)
            }
            if (groups.isEmpty()) {
                item {
                    Text(
                        text = "暂无群组",
                        color = LanLineColors.Muted,
                        modifier = Modifier.padding(top = 18.dp),
                    )
                }
            }
            items(groups, key = { it.id }) { group ->
                GroupRow(group = group, onClick = { onOpenGroup(group.toConversation()) })
            }
            item { Spacer(Modifier.height(88.dp)) }
        }
    }
}

@Composable
private fun GroupRow(group: GroupUi, onClick: () -> Unit) {
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
            LanLineAvatar(text = group.avatar)
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = group.name,
                    color = LanLineColors.Text,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = buildString {
                        if (group.memberCount > 0) append("${group.memberCount} 人 · ")
                        append(group.description)
                        if (group.doNotDisturb) append(" · 免打扰")
                    },
                    color = LanLineColors.Muted,
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Icon(Icons.Default.ChevronRight, contentDescription = null, tint = LanLineColors.Subtle)
        }
    }
}

private fun GroupUi.toConversation(): ConversationUi =
    ConversationUi(
        id = id,
        type = ConversationType.Group,
        title = name,
        avatarText = avatar,
        lastMessage = description,
        timeText = "",
    )
