package com.aim.app

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.runtime.mutableStateOf
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
    var routeName by rememberSaveable { mutableStateOf(AimRoute.Setup.name) }
    val route = AimRoute.valueOf(routeName)
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
                AimRoute.Setup -> SetupScreen(onSaved = { navigate(AimRoute.Login) })
                AimRoute.Login -> LoginScreen(
                    onLogin = { navigate(AimRoute.Chats) },
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
                )
            }
        }
    }
}
