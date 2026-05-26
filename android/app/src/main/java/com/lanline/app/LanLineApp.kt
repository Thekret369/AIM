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
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.platform.LocalContext
import com.lanline.app.core.app.LanLineBuildInfo
import com.lanline.app.core.auth.AuthSession
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
import com.lanline.app.feature.profile.ProfileScreen
import com.lanline.app.feature.setup.SetupScreen
import com.lanline.app.model.LanLineRoute

@Composable
fun LanLineApp() {
    val context = LocalContext.current
    val mainHandler = remember { Handler(Looper.getMainLooper()) }
    val notifier = remember(context) { LocalMessageNotifier(context.applicationContext) }
    var routeName by rememberSaveable { mutableStateOf(LanLineRoute.Login.name) }
    var apiBaseUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultApiBaseUrl) }
    var websocketUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultWebSocketUrl) }
    var session by remember { mutableStateOf<AuthSession?>(null) }
    var realtimeState by remember { mutableStateOf(RealtimeConnectionState.Idle) }
    var realtimeMessages by remember { mutableStateOf<List<RealtimeChatMessage>>(emptyList()) }
    var updateInfo by remember { mutableStateOf<AppUpdateInfo?>(null) }
    val route = LanLineRoute.valueOf(routeName)
    val serverConfig = ServerConfig(apiBaseUrl = apiBaseUrl, websocketUrl = websocketUrl)
    val navigate: (LanLineRoute) -> Unit = { next -> routeName = next.name }

    DisposableEffect(session?.token, session?.websocketUrl) {
        val activeSession = session
        if (activeSession == null) {
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
                    notifier.notifyMessage(message, activeSession.user.id)
                    mainHandler.post {
                        realtimeMessages = (realtimeMessages + message).takeLast(MaxRealtimeMessages)
                    }
                },
            )
            realtimeMessages = emptyList()
            client.connect()
            onDispose { client.disconnect() }
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
                LanLineRoute.Setup -> SetupScreen(
                    config = serverConfig,
                    onSaved = { nextConfig ->
                        apiBaseUrl = nextConfig.apiBaseUrl
                        websocketUrl = nextConfig.websocketUrl
                        navigate(LanLineRoute.Login)
                    },
                )
                LanLineRoute.Login -> LoginScreen(
                    config = serverConfig,
                    onLogin = { nextSession ->
                        session = nextSession
                        navigate(LanLineRoute.Chats)
                    },
                    onChangeServer = { navigate(LanLineRoute.Setup) },
                )
                LanLineRoute.Chats -> ChatsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    realtimeState = realtimeState,
                    realtimeMessages = realtimeMessages,
                    onOpenChat = { navigate(LanLineRoute.Chat) },
                    onOpenAi = { navigate(LanLineRoute.Ai) },
                )
                LanLineRoute.Chat -> ChatScreen(
                    onBack = { navigate(LanLineRoute.Chats) },
                    realtimeMessages = realtimeMessages,
                    currentUserId = session?.user?.id,
                )
                LanLineRoute.Ai -> AiScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    onBack = { navigate(LanLineRoute.Chats) },
                )
                LanLineRoute.Contacts -> ContactsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    onOpenChat = { navigate(LanLineRoute.Chat) },
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
