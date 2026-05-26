package com.aim.app.core.auth

data class AuthSession(
    val token: String,
    val user: AuthUser,
    val apiBaseUrl: String,
    val websocketUrl: String,
)

data class AuthUser(
    val id: Long,
    val username: String,
    val nickname: String,
    val avatar: String,
    val bio: String,
    val isAi: Boolean,
) {
    val displayName: String
        get() = nickname.ifBlank { username }
}

class AimApiException(
    val statusCode: Int,
    message: String,
) : Exception(message)
