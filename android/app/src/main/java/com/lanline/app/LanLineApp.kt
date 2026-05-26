package com.lanline.app

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import com.lanline.app.core.auth.AuthSession
import com.lanline.app.core.config.ServerConfig
import com.lanline.app.core.ui.LanLineTheme
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
    var routeName by rememberSaveable { mutableStateOf(LanLineRoute.Login.name) }
    var apiBaseUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultApiBaseUrl) }
    var websocketUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultWebSocketUrl) }
    var session by remember { mutableStateOf<AuthSession?>(null) }
    val route = LanLineRoute.valueOf(routeName)
    val serverConfig = ServerConfig(apiBaseUrl = apiBaseUrl, websocketUrl = websocketUrl)
    val navigate: (LanLineRoute) -> Unit = { next -> routeName = next.name }

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
                    onOpenChat = { navigate(LanLineRoute.Chat) },
                    onOpenAi = { navigate(LanLineRoute.Ai) },
                )
                LanLineRoute.Chat -> ChatScreen(
                    onBack = { navigate(LanLineRoute.Chats) },
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
                )
            }
        }
    }
}
