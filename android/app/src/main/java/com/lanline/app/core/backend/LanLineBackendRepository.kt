package com.lanline.app.core.backend

import com.lanline.app.core.auth.AuthSession
import com.lanline.app.core.network.LanLineApiClient
import com.lanline.app.core.network.LanLineWebApi
import com.lanline.app.core.network.optArray
import com.lanline.app.core.ui.LanLineColors
import com.lanline.app.model.BackendSnapshot
import com.lanline.app.model.ChatMessageUi
import com.lanline.app.model.ContactUi
import com.lanline.app.model.ConversationType
import com.lanline.app.model.ConversationUi
import com.lanline.app.model.FriendRequestUi
import com.lanline.app.model.GroupUi
import com.lanline.app.model.MessageStatus
import org.json.JSONArray
import org.json.JSONObject

class LanLineBackendRepository(
    session: AuthSession,
) {
    private val api = LanLineWebApi(LanLineApiClient(session.apiBaseUrl, session.token))

    suspend fun loadSnapshot(): Result<BackendSnapshot> =
        runCatching {
            val failures = mutableListOf<String>()
            val onlineIds = runApi("在线好友", failures) { api.onlineFriends() }
                ?.firstArray("friends", "users", "data", "items")
                ?.toOnlineIdSet()
                .orEmpty()
            val friends = runApi("好友", failures) { api.friends() }?.firstArray("friends", "data", "items") ?: JSONArray()
            val groups = runApi("群组", failures) { api.groups() }?.firstArray("groups", "data", "items") ?: JSONArray()
            val bots = runApi("AI助手", failures) { api.aiBots() }?.firstArray("bots", "data", "items") ?: JSONArray()
            val pending = runApi("好友申请", failures) { api.pendingFriends() }?.firstArray("requests", "friends", "data", "items") ?: JSONArray()

            val friendContacts = friends.mapObjects { friend ->
                val id = friend.firstLong("friend_id", "user_id", "id")
                val name = friend.displayName(friend.optString("remark"))
                ContactUi(
                    id = id,
                    name = name,
                    avatar = name.initial("友"),
                    subtitle = friend.optString("note").ifBlank { friend.optString("username").ifBlank { "好友" } },
                    group = "好友",
                    online = onlineIds.contains(id),
                )
            }.filter { it.id > 0 }

            val groupItems = groups.mapObjects { group ->
                val id = group.firstLong("id", "group_id")
                val name = group.optString("name").ifBlank { "群组 $id" }
                GroupUi(
                    id = id,
                    name = name,
                    avatar = name.initial("群"),
                    description = group.optString("description").ifBlank { "暂无简介" },
                    memberCount = group.optInt("member_count", group.optInt("members_count", 0)),
                    doNotDisturb = group.optBoolean("dnd", false),
                )
            }.filter { it.id > 0 }

            val aiContacts = bots.mapObjects { bot ->
                val userId = bot.firstLong("user_id", "id")
                val name = bot.displayName()
                ContactUi(
                    id = userId,
                    name = name,
                    avatar = "AI",
                    subtitle = bot.optString("unavailable_reason").ifBlank {
                        if (bot.optBoolean("ready", false)) "可对话" else bot.optString("status").ifBlank { "AI助手" }
                    },
                    group = "AI",
                    online = bot.optBoolean("ready", false),
                )
            }.filter { it.id > 0 }

            val pendingRequests = pending.mapObjects { request ->
                val userId = request.firstLong("user_id", "from_user_id", "friend_id")
                val name = request.displayName()
                FriendRequestUi(
                    id = request.firstLong("id"),
                    userId = userId,
                    name = name,
                    message = request.optString("message"),
                )
            }.filter { it.id > 0 }

            val conversations = buildList {
                add(
                    ConversationUi(
                        id = 0,
                        type = ConversationType.Broadcast,
                        title = "广播消息",
                        avatarText = "广",
                        lastMessage = "系统广播与通知",
                        timeText = "",
                        accent = LanLineColors.Muted,
                        accentSoft = LanLineColors.SurfaceAlt,
                    ),
                )
                addAll(friendContacts.map { contact ->
                    ConversationUi(
                        id = contact.id,
                        type = ConversationType.User,
                        title = contact.name,
                        avatarText = contact.avatar,
                        lastMessage = contact.subtitle.ifBlank { "点击查看单聊历史" },
                        timeText = "",
                        online = contact.online,
                        accent = LanLineColors.Accent,
                        accentSoft = LanLineColors.AccentSoft,
                    )
                })
                addAll(aiContacts.map { contact ->
                    ConversationUi(
                        id = contact.id,
                        type = ConversationType.Ai,
                        title = contact.name,
                        avatarText = "AI",
                        lastMessage = contact.subtitle,
                        timeText = "",
                        online = contact.online,
                        accent = LanLineColors.Ai,
                        accentSoft = LanLineColors.AiSoft,
                    )
                })
                addAll(groupItems.map { group ->
                    ConversationUi(
                        id = group.id,
                        type = ConversationType.Group,
                        title = group.name,
                        avatarText = group.avatar,
                        lastMessage = group.description,
                        timeText = "",
                    )
                })
            }

            val status = if (failures.isEmpty()) {
                "已同步 · ${friendContacts.size} 好友 · ${groupItems.size} 群组 · ${aiContacts.size} AI"
            } else {
                "部分接口失败：${failures.distinct().joinToString("、")}"
            }
            BackendSnapshot(
                conversations = conversations,
                contacts = friendContacts + aiContacts,
                groups = groupItems,
                pendingFriends = pendingRequests,
                statusText = status,
            )
        }

    suspend fun loadConversationHistory(conversation: ConversationUi, currentUserId: Long): Result<List<ChatMessageUi>> =
        runCatching {
            val response = when (conversation.type) {
                ConversationType.User, ConversationType.Ai -> api.history(conversation.id)
                ConversationType.Group -> api.groupHistory(conversation.id)
                ConversationType.Broadcast -> api.broadcastHistory()
            }
            response.firstArray("messages", "data", "items")
                .mapObjects { message -> message.toChatMessage(currentUserId) }
                .asReversed()
        }

    suspend fun handleFriendRequest(requestId: Long, accept: Boolean): Result<Unit> =
        runCatching { api.handleFriendRequest(requestId, accept) }.map { }

    suspend fun logout(): Result<Unit> =
        runCatching { api.logout() }.map { }

    suspend fun deleteFriend(friendId: Long): Result<Unit> =
        runCatching { api.deleteFriend(friendId) }.map { }

    private suspend fun runApi(
        name: String,
        failures: MutableList<String>,
        block: suspend () -> JSONObject,
    ): JSONObject? =
        runCatching { block() }
            .onFailure { failures += name }
            .getOrNull()

    private fun JSONObject.firstArray(vararg names: String): JSONArray =
        names.firstNotNullOfOrNull { name -> optArray(name).takeIf { it.length() > 0 } } ?: JSONArray()

    private fun JSONArray.toOnlineIdSet(): Set<Long> =
        mapObjects { friend -> friend.firstLong("id", "friend_id", "user_id") }.filter { it > 0 }.toSet()

    private fun <T> JSONArray.mapObjects(transform: (JSONObject) -> T): List<T> {
        val result = mutableListOf<T>()
        for (index in 0 until length()) {
            optJSONObject(index)?.let { result += transform(it) }
        }
        return result
    }

    private fun JSONObject.displayName(preferred: String = ""): String =
        preferred.ifBlank {
            optString("nickname").ifBlank {
                optString("name").ifBlank {
                    optString("username").ifBlank {
                        val id = firstLong("id", "user_id", "friend_id")
                        if (id > 0) "用户$id" else "未命名"
                    }
                }
            }
        }

    private fun JSONObject.firstLong(vararg names: String): Long {
        for (name in names) {
            if (has(name) && !isNull(name)) {
                val value = optLong(name, 0L)
                if (value > 0) return value
            }
        }
        return 0L
    }

    private fun String.initial(fallback: String): String =
        trim().take(1).ifBlank { fallback }

    private fun JSONObject.toChatMessage(currentUserId: Long): ChatMessageUi {
        val type = optString("type", "text")
        val fileName = optString("file_name")
        val content = when {
            optBoolean("is_recalled", false) -> "消息已撤回"
            optString("content").isNotBlank() -> optString("content")
            fileName.isNotBlank() -> fileName
            else -> "收到一条${type}消息"
        }
        return ChatMessageUi(
            id = firstLong("id").takeIf { it > 0 } ?: System.currentTimeMillis(),
            content = content,
            fromMe = firstLong("from_user_id") == currentUserId,
            timeText = optString("created_at").toShortTime(),
            status = MessageStatus.Read,
            isAi = optJSONObject("from_user")?.optBoolean("is_ai", false) == true ||
                optJSONObject("from_user")?.optString("username").orEmpty().contains("ai", ignoreCase = true),
            isImage = type == "image",
        )
    }

    private fun String.toShortTime(): String =
        when {
            length >= 16 -> substring(11, 16)
            isNotBlank() -> this
            else -> ""
        }
}
