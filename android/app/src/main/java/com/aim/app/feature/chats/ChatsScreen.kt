package com.aim.app.feature.chats

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.aim.app.core.ui.AimColors
import com.aim.app.core.ui.AimMainScaffold
import com.aim.app.core.ui.AimTopBar
import com.aim.app.core.ui.ConversationRow
import com.aim.app.core.ui.FilterChipRow
import com.aim.app.core.ui.SearchBox
import com.aim.app.model.AimRoute
import com.aim.app.model.AimSampleData
import com.aim.app.model.ConversationType

@Composable
fun ChatsScreen(
    currentRoute: AimRoute,
    onNavigate: (AimRoute) -> Unit,
    onOpenChat: () -> Unit,
    onOpenAi: () -> Unit,
) {
    var selected by rememberSaveable { mutableStateOf("全部") }
    val conversations = AimSampleData.conversations.filter {
        when (selected) {
            "未读" -> it.unreadCount > 0
            "群聊" -> it.type == ConversationType.Group
            "AI" -> it.type == ConversationType.Ai
            else -> true
        }
    }

    AimMainScaffold(
        currentRoute = currentRoute,
        onNavigate = onNavigate,
        floatingActionButton = {
            FloatingActionButton(
                onClick = {},
                containerColor = AimColors.Primary,
                contentColor = AimColors.Surface,
            ) {
                Icon(Icons.Default.Add, contentDescription = "新建会话")
            }
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier.padding(padding),
            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            item {
                AimTopBar(title = "消息", subtitle = "5 个会话 · 2 人在线")
                SearchBox("搜索联系人、群聊或消息")
                Spacer(Modifier.height(14.dp))
                FilterChipRow(
                    items = listOf("全部", "未读", "群聊", "AI"),
                    selected = selected,
                    aiChip = "AI",
                    onSelected = { selected = it },
                )
            }
            items(conversations, key = { it.id }) { conversation ->
                ConversationRow(
                    conversation = conversation,
                    onClick = {
                        if (conversation.type == ConversationType.Ai) onOpenAi() else onOpenChat()
                    },
                )
            }
            item { Spacer(Modifier.height(88.dp)) }
        }
    }
}
