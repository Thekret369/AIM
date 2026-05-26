package com.aim.app.core.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AutoAwesome
import androidx.compose.material.icons.filled.Chat
import androidx.compose.material.icons.filled.People
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.NavigationBarItemDefaults
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.aim.app.model.AimRoute
import com.aim.app.model.ConversationUi

private data class NavSpec(
    val route: AimRoute,
    val label: String,
    val icon: ImageVector,
)

private val mainNav = listOf(
    NavSpec(AimRoute.Chats, "聊天", Icons.Default.Chat),
    NavSpec(AimRoute.Contacts, "联系人", Icons.Default.People),
    NavSpec(AimRoute.Ai, "AI", Icons.Default.AutoAwesome),
    NavSpec(AimRoute.Profile, "我的", Icons.Default.Person),
)

@Composable
fun AimMainScaffold(
    currentRoute: AimRoute,
    onNavigate: (AimRoute) -> Unit,
    floatingActionButton: @Composable () -> Unit = {},
    content: @Composable (PaddingValues) -> Unit,
) {
    Scaffold(
        containerColor = AimColors.Background,
        floatingActionButton = floatingActionButton,
        bottomBar = {
            NavigationBar(
                containerColor = AimColors.Surface,
                tonalElevation = 6.dp,
            ) {
                mainNav.forEach { item ->
                    val selected = item.route == currentRoute
                    NavigationBarItem(
                        selected = selected,
                        onClick = { onNavigate(item.route) },
                        icon = {
                            Icon(
                                imageVector = item.icon,
                                contentDescription = item.label,
                            )
                        },
                        label = { Text(item.label) },
                        colors = NavigationBarItemDefaults.colors(
                            selectedIconColor = if (item.route == AimRoute.Ai) AimColors.Ai else AimColors.Primary,
                            selectedTextColor = AimColors.Text,
                            indicatorColor = if (item.route == AimRoute.Ai) AimColors.AiSoft else AimColors.PrimarySoft,
                            unselectedIconColor = AimColors.Subtle,
                            unselectedTextColor = AimColors.Muted,
                        ),
                    )
                }
            }
        },
        content = content,
    )
}

@Composable
fun AimTopBar(
    title: String,
    subtitle: String? = null,
    leading: @Composable (() -> Unit)? = null,
    trailing: @Composable (() -> Unit)? = null,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .statusBarsPadding()
            .height(72.dp)
            .padding(horizontal = 18.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(10.dp),
    ) {
        if (leading != null) {
            leading()
        }
        Column(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.Center,
        ) {
            Text(
                text = title,
                color = AimColors.Text,
                fontSize = 21.sp,
                lineHeight = 28.sp,
                fontWeight = FontWeight.Bold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (!subtitle.isNullOrBlank()) {
                Text(
                    text = subtitle,
                    color = AimColors.Muted,
                    fontSize = 12.sp,
                    lineHeight = 16.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        if (trailing != null) {
            trailing()
        }
    }
}

@Composable
fun AimPrimaryButton(
    text: String,
    modifier: Modifier = Modifier,
    dark: Boolean = false,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Button(
        onClick = onClick,
        enabled = enabled,
        modifier = modifier
            .fillMaxWidth()
            .height(48.dp),
        shape = RoundedCornerShape(8.dp),
        colors = ButtonDefaults.buttonColors(
            containerColor = if (dark) AimColors.Dark else AimColors.Primary,
            contentColor = Color.White,
        ),
    ) {
        Text(text = text, fontWeight = FontWeight.Bold)
    }
}

@Composable
fun AimTextField(
    label: String,
    value: String,
    onValueChange: (String) -> Unit,
    modifier: Modifier = Modifier,
    singleLine: Boolean = true,
) {
    OutlinedTextField(
        value = value,
        onValueChange = onValueChange,
        modifier = modifier.fillMaxWidth(),
        label = { Text(label) },
        singleLine = singleLine,
        shape = RoundedCornerShape(8.dp),
        colors = OutlinedTextFieldDefaults.colors(
            focusedBorderColor = AimColors.Primary,
            unfocusedBorderColor = AimColors.Line,
            focusedContainerColor = AimColors.Surface,
            unfocusedContainerColor = AimColors.Surface,
        ),
    )
}

@Composable
fun AimAvatar(
    text: String,
    modifier: Modifier = Modifier,
    color: Color = AimColors.Primary,
    background: Color = AimColors.PrimarySoft,
    online: Boolean = false,
) {
    Box(modifier = modifier.size(44.dp)) {
        Box(
            modifier = Modifier
                .matchParentSize()
                .clip(CircleShape)
                .background(background),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = text,
                color = color,
                fontWeight = FontWeight.Bold,
                fontSize = 13.sp,
                textAlign = TextAlign.Center,
            )
        }
        if (online) {
            Box(
                modifier = Modifier
                    .size(11.dp)
                    .align(Alignment.BottomEnd)
                    .clip(CircleShape)
                    .background(AimColors.Primary)
                    .border(2.dp, AimColors.Surface, CircleShape),
            )
        }
    }
}

@Composable
fun SearchBox(text: String, modifier: Modifier = Modifier) {
    Row(
        modifier = modifier
            .fillMaxWidth()
            .height(42.dp)
            .clip(RoundedCornerShape(8.dp))
            .background(AimColors.Surface)
            .border(1.dp, AimColors.Line, RoundedCornerShape(8.dp))
            .padding(horizontal = 14.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text("⌕", color = AimColors.Subtle, fontWeight = FontWeight.Bold)
        Spacer(Modifier.width(10.dp))
        Text(text = text, color = AimColors.Subtle, fontSize = 13.sp)
    }
}

@Composable
fun FilterChipRow(
    items: List<String>,
    selected: String,
    modifier: Modifier = Modifier,
    aiChip: String? = null,
    onSelected: (String) -> Unit,
) {
    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items.forEach { item ->
            val isSelected = item == selected
            val activeColor = if (item == aiChip) AimColors.Ai else AimColors.Primary
            Box(
                modifier = Modifier
                    .height(32.dp)
                    .clip(RoundedCornerShape(8.dp))
                    .background(if (isSelected) activeColor.copy(alpha = 0.12f) else AimColors.Surface)
                    .border(
                        width = if (isSelected) 0.dp else 1.dp,
                        color = if (isSelected) Color.Transparent else AimColors.Line,
                        shape = RoundedCornerShape(8.dp),
                    )
                    .clickable { onSelected(item) }
                    .padding(horizontal = 14.dp),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = item,
                    color = if (isSelected) activeColor else AimColors.Muted,
                    fontSize = 12.sp,
                    fontWeight = FontWeight.SemiBold,
                )
            }
        }
    }
}

@Composable
fun ConversationRow(
    conversation: ConversationUi,
    onClick: () -> Unit,
) {
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
        ) {
            AimAvatar(
                text = conversation.avatarText,
                color = conversation.accent,
                background = conversation.accentSoft,
                online = conversation.online,
            )
            Spacer(Modifier.width(12.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = conversation.title,
                    color = AimColors.Text,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Bold,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                Text(
                    text = conversation.lastMessage,
                    color = AimColors.Muted,
                    fontSize = 12.sp,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
            Column(
                horizontalAlignment = Alignment.End,
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text(text = conversation.timeText, color = AimColors.Subtle, fontSize = 11.sp)
                if (conversation.unreadCount > 0) {
                    BadgedBox(
                        badge = {
                            Badge(containerColor = AimColors.Danger) {
                                Text("${conversation.unreadCount}", color = Color.White, fontSize = 10.sp)
                            }
                        },
                    ) {}
                }
            }
        }
    }
}
