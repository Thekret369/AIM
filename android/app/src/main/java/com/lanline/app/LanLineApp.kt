package com.lanline.app

import android.os.Build
import android.os.Handler
import android.os.Looper
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.lanline.app.core.app.LanLineBuildInfo
import com.lanline.app.core.auth.AuthSession
import com.lanline.app.core.backend.LanLineBackendRepository
import com.lanline.app.core.config.ServerConfig
import com.lanline.app.core.notification.LocalMessageNotifier
import com.lanline.app.core.realtime.LanLineRealtimeClient
import com.lanline.app.core.realtime.RealtimeChatMessage
import com.lanline.app.core.realtime.RealtimeConnectionState
import com.lanline.app.core.ui.LanLineTheme
import com.lanline.app.core.update.AppUpdateCheckRequest
import com.lanline.app.core.update.AppUpdateInfo
import com.lanline.app.core.update.HttpAppUpdateGateway
import com.lanline.app.core.update.UpdateChannel
import com.lanline.app.feature.ai.AiScreen
import com.lanline.app.feature.auth.LoginScreen
import com.lanline.app.feature.chat.ChatScreen
import com.lanline.app.feature.chats.ChatsScreen
import com.lanline.app.feature.contacts.ContactsScreen
import com.lanline.app.feature.groups.GroupsScreen
import com.lanline.app.feature.profile.ProfileScreen
import com.lanline.app.model.BackendSnapshot
import com.lanline.app.model.ChatMessageUi
import com.lanline.app.model.ContactUi
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.LanLineRoute
import kotlinx.coroutines.launch
import org.json.JSONObject

@Composable
fun LanLineApp() {
    val context = LocalContext.current
    val mainHandler = remember { Handler(Looper.getMainLooper()) }
    val scope = rememberCoroutineScope()
    val notifier = remember(context) { LocalMessageNotifier(context.applicationContext) }
    val serverConfig = remember { ServerConfig() }
    var routeName by rememberSaveable { mutableStateOf(LanLineRoute.Login.name) }
    var session by remember { mutableStateOf<AuthSession?>(null) }
    var realtimeClient by remember { mutableStateOf<LanLineRealtimeClient?>(null) }
    var realtimeState by remember { mutableStateOf(RealtimeConnectionState.Idle) }
    var realtimeMessages by remember { mutableStateOf<List<RealtimeChatMessage>>(emptyList()) }
    var updateInfo by remember { mutableStateOf<AppUpdateInfo?>(null) }
    var backendSnapshot by remember { mutableStateOf(BackendSnapshot()) }
    var selectedConversation by remember { mutableStateOf<ConversationUi?>(null) }
    var historyMessages by remember { mutableStateOf<List<ChatMessageUi>>(emptyList()) }
    var historyStatus by remember { mutableStateOf("请选择会话") }
    val route = LanLineRoute.valueOf(routeName)
    val navigate: (LanLineRoute) -> Unit = { next -> routeName = next.name }

    suspend fun reloadSnapshot(activeSession: AuthSession) {
        backendSnapshot = BackendSnapshot(statusText = "正在同步")
        LanLineBackendRepository(activeSession).loadSnapshot()
            .onSuccess { snapshot -> backendSnapshot = snapshot }
            .onFailure { error ->
                backendSnapshot = BackendSnapshot(statusText = error.message ?: "接口连接失败")
            }
    }

    DisposableEffect(session?.token, session?.websocketUrl) {
        val activeSession = session
        if (activeSession == null) {
            realtimeClient = null
            realtimeState = RealtimeConnectionState.Idle
            realtimeMessages = emptyList()
            onDispose { }
        } else {
            val client = LanLineRealtimeClient(
                session = activeSession,
                onStateChange = { state ->
                    mainHandler.post { realtimeState = state }
                },
                onChatMessage = { message ->
                    if (message.fromUserId != activeSession.user.id) {
                        notifier.notifyMessage(message, activeSession.user.id)
                    }
                    mainHandler.post {
                        realtimeMessages = (realtimeMessages + message)
                            .distinctBy { it.id.takeIf { id -> id > 0 } ?: it.receivedAtEpochMillis }
                            .takeLast(MaxRealtimeMessages)
                    }
                },
            )
            realtimeClient = client
            realtimeMessages = emptyList()
            client.connect()
            onDispose {
                if (realtimeClient === client) realtimeClient = null
                client.disconnect()
            }
        }
    }

    LaunchedEffect(session?.apiBaseUrl) {
        val activeSession = session ?: return@LaunchedEffect
        val gateway = HttpAppUpdateGateway(
            ServerConfig(apiBaseUrl = activeSession.apiBaseUrl, websocketUrl = activeSession.websocketUrl),
        )
        gateway.checkLatest(
            AppUpdateCheckRequest(
                versionCode = LanLineBuildInfo.VersionCode,
                versionName = LanLineBuildInfo.VersionName,
                channel = UpdateChannel.Internal,
                deviceAbi = Build.SUPPORTED_ABIS.firstOrNull().orEmpty(),
            ),
        ).onSuccess { info ->
            updateInfo = info
        }
    }

    LaunchedEffect(session?.token, session?.apiBaseUrl) {
        val activeSession = session
        if (activeSession == null) {
            backendSnapshot = BackendSnapshot()
            selectedConversation = null
            return@LaunchedEffect
        }
        reloadSnapshot(activeSession)
    }

    LaunchedEffect(session?.token, selectedConversation?.type, selectedConversation?.id) {
        val activeSession = session
        val activeConversation = selectedConversation
        if (activeSession == null || activeConversation == null) {
            historyMessages = emptyList()
            historyStatus = "请选择会话"
            return@LaunchedEffect
        }
        historyMessages = emptyList()
        historyStatus = "正在加载历史消息"
        LanLineBackendRepository(activeSession).loadConversationHistory(activeConversation, activeSession.user.id)
            .onSuccess { messages ->
                historyMessages = messages
                historyStatus = if (messages.isEmpty()) "暂无历史消息" else "已加载 ${messages.size} 条历史消息"
            }
            .onFailure { error ->
                historyStatus = error.message ?: "历史消息加载失败"
            }
    }

    LanLineTheme {
        AnimatedContent(
            targetState = route,
            transitionSpec = {
                (fadeIn() + slideInHorizontally { it / 8 }) togetherWith
                    (fadeOut() + slideOutHorizontally { -it / 10 })
            },
            label = "lanline-route-transition",
        ) { current ->
            when (current) {
                LanLineRoute.Login -> LoginScreen(
                    config = serverConfig,
                    onLogin = { nextSession ->
                        session = nextSession
                        navigate(LanLineRoute.Chats)
                    },
                )
                LanLineRoute.Chats -> ChatsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    realtimeState = realtimeState,
                    realtimeMessages = realtimeMessages,
                    conversations = backendSnapshot.conversations,
                    backendStatus = backendSnapshot.statusText,
                    onOpenChat = { conversation ->
                        selectedConversation = conversation
                        navigate(LanLineRoute.Chat)
                    },
                )
                LanLineRoute.Chat -> ChatScreen(
                    onBack = { navigate(LanLineRoute.Chats) },
                    conversation = selectedConversation,
                    historyMessages = historyMessages,
                    historyStatus = historyStatus,
                    realtimeMessages = realtimeMessages,
                    currentUserId = session?.user?.id,
                    onSendText = { conversation, text ->
                        conversation.type != ConversationType.Broadcast &&
                            realtimeClient?.sendChat(conversation.toMessagePayload(text)) == true
                    },
                )
                LanLineRoute.Contacts -> ContactsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    contacts = backendSnapshot.contacts,
                    pendingFriends = backendSnapshot.pendingFriends,
                    backendStatus = backendSnapshot.statusText,
                    onOpenChat = { contact ->
                        selectedConversation = contact.toConversation()
                        navigate(LanLineRoute.Chat)
                    },
                    onHandleRequest = { request, accept ->
                        val activeSession = session
                        if (activeSession != null) {
                            scope.launch {
                                LanLineBackendRepository(activeSession).handleFriendRequest(request.id, accept)
                                reloadSnapshot(activeSession)
                            }
                        }
                    },
                )
                LanLineRoute.Groups -> GroupsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    groups = backendSnapshot.groups,
                    backendStatus = backendSnapshot.statusText,
                    onOpenGroup = { conversation ->
                        selectedConversation = conversation
                        navigate(LanLineRoute.Chat)
                    },
                )
                LanLineRoute.Ai -> AiScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    aiContacts = backendSnapshot.contacts.filter { it.group == "AI" },
                    backendStatus = backendSnapshot.statusText,
                    onOpenChat = { conversation ->
                        selectedConversation = conversation
                        navigate(LanLineRoute.Chat)
                    },
                )
                LanLineRoute.Profile -> ProfileScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    session = session,
                    realtimeState = realtimeState,
                    updateInfo = updateInfo,
                )
            }
        }
    }
}

private const val MaxRealtimeMessages = 50

private fun ConversationUi.toMessagePayload(text: String): JSONObject =
    JSONObject()
        .put("type", "text")
        .put("content", text)
        .apply {
            when (type) {
                ConversationType.User, ConversationType.Ai -> put("to_user_id", id)
                ConversationType.Group -> put("group_id", id)
                ConversationType.Broadcast -> Unit
            }
        }

private fun ContactUi.toConversation(): ConversationUi =
    ConversationUi(
        id = id,
        type = if (group == "AI") ConversationType.Ai else ConversationType.User,
        title = name,
        avatarText = avatar,
        lastMessage = subtitle,
        timeText = "",
        online = online,
        accent = if (group == "AI") com.lanline.app.core.ui.LanLineColors.Ai else com.lanline.app.core.ui.LanLineColors.Accent,
        accentSoft = if (group == "AI") com.lanline.app.core.ui.LanLineColors.AiSoft else com.lanline.app.core.ui.LanLineColors.AccentSoft,
    )
