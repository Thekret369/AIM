package com.aim.app

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
import com.aim.app.core.auth.AuthSession
import com.aim.app.core.config.ServerConfig
import com.aim.app.core.ui.AimTheme
import com.aim.app.feature.ai.AiScreen
import com.aim.app.feature.auth.LoginScreen
import com.aim.app.feature.chat.ChatScreen
import com.aim.app.feature.chats.ChatsScreen
import com.aim.app.feature.contacts.ContactsScreen
import com.aim.app.feature.profile.ProfileScreen
import com.aim.app.feature.setup.SetupScreen
import com.aim.app.model.AimRoute

@Composable
fun AimApp() {
    var routeName by rememberSaveable { mutableStateOf(AimRoute.Login.name) }
    var apiBaseUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultApiBaseUrl) }
    var websocketUrl by rememberSaveable { mutableStateOf(ServerConfig.DefaultWebSocketUrl) }
    var session by remember { mutableStateOf<AuthSession?>(null) }
    val route = AimRoute.valueOf(routeName)
    val serverConfig = ServerConfig(apiBaseUrl = apiBaseUrl, websocketUrl = websocketUrl)
    val navigate: (AimRoute) -> Unit = { next -> routeName = next.name }

    AimTheme {
        AnimatedContent(
            targetState = route,
            transitionSpec = {
                (fadeIn() + slideInHorizontally { it / 8 }) togetherWith
                    (fadeOut() + slideOutHorizontally { -it / 10 })
            },
            label = "aim-route-transition",
        ) { current ->
            when (current) {
                AimRoute.Setup -> SetupScreen(
                    config = serverConfig,
                    onSaved = { nextConfig ->
                        apiBaseUrl = nextConfig.apiBaseUrl
                        websocketUrl = nextConfig.websocketUrl
                        navigate(AimRoute.Login)
                    },
                )
                AimRoute.Login -> LoginScreen(
                    config = serverConfig,
                    onLogin = { nextSession ->
                        session = nextSession
                        navigate(AimRoute.Chats)
                    },
                    onChangeServer = { navigate(AimRoute.Setup) },
                )
                AimRoute.Chats -> ChatsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    onOpenChat = { navigate(AimRoute.Chat) },
                    onOpenAi = { navigate(AimRoute.Ai) },
                )
                AimRoute.Chat -> ChatScreen(
                    onBack = { navigate(AimRoute.Chats) },
                )
                AimRoute.Ai -> AiScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    onBack = { navigate(AimRoute.Chats) },
                )
                AimRoute.Contacts -> ContactsScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    onOpenChat = { navigate(AimRoute.Chat) },
                )
                AimRoute.Profile -> ProfileScreen(
                    currentRoute = current,
                    onNavigate = navigate,
                    session = session,
                )
            }
        }
    }
}
