package com.lanline.app.core.network

import org.json.JSONArray
import org.json.JSONObject

class LanLineWebApi(
    private val client: LanLineApiClient,
) {
    suspend fun register(username: String, password: String, nickname: String): JSONObject =
        client.post("register", jsonBody("username" to username, "password" to password, "nickname" to nickname))

    suspend fun logout(): JSONObject = client.post("logout")

    suspend fun profile(userId: Long? = null): JSONObject =
        if (userId == null) client.get("profile") else client.get("profile/$userId")

    suspend fun updateProfile(nickname: String, avatar: String, bio: String): JSONObject =
        client.put("profile", jsonBody("nickname" to nickname, "avatar" to avatar, "bio" to bio))

    suspend fun changePassword(oldPassword: String, newPassword: String): JSONObject =
        client.put("profile/password", jsonBody("old_password" to oldPassword, "new_password" to newPassword))

    suspend fun friends(): JSONObject = client.get("friends")

    suspend fun onlineFriends(): JSONObject = client.get("friends/online")

    suspend fun pendingFriends(): JSONObject = client.get("friends/pending")

    suspend fun addFriend(friendId: Long, message: String): JSONObject =
        client.post("friends", jsonBody("friend_id" to friendId, "message" to message))

    suspend fun handleFriendRequest(requestId: Long, accept: Boolean): JSONObject =
        client.put("friends/$requestId", jsonBody("accept" to accept))

    suspend fun deleteFriend(friendId: Long): JSONObject = client.delete("friends/$friendId")

    suspend fun updateFriendRemark(friendId: Long, remark: String, note: String, groupId: Long?): JSONObject =
        client.put("friends/$friendId/remark", jsonBody("remark" to remark, "note" to note, "group_id" to groupId))

    suspend fun contactGroups(): JSONObject = client.get("contact-groups")

    suspend fun createContactGroup(name: String): JSONObject =
        client.post("contact-groups", jsonBody("name" to name))

    suspend fun updateContactGroup(id: Long, name: String): JSONObject =
        client.put("contact-groups/$id", jsonBody("name" to name))

    suspend fun deleteContactGroup(id: Long): JSONObject = client.delete("contact-groups/$id")

    suspend fun groups(): JSONObject = client.get("groups")

    suspend fun createGroup(name: String, description: String): JSONObject =
        client.post("groups", jsonBody("name" to name, "description" to description))

    suspend fun groupDetail(groupId: Long): JSONObject = client.get("groups/$groupId")

    suspend fun updateGroup(groupId: Long, name: String, avatar: String, description: String, announce: String): JSONObject =
        client.put("groups/$groupId", jsonBody("name" to name, "avatar" to avatar, "description" to description, "announce" to announce))

    suspend fun joinGroup(groupId: Long): JSONObject = client.post("groups/$groupId/join")

    suspend fun addGroupMember(groupId: Long, userId: Long): JSONObject =
        client.post("groups/$groupId/add-member", jsonBody("user_id" to userId))

    suspend fun leaveGroup(groupId: Long): JSONObject = client.post("groups/$groupId/leave")

    suspend fun kickGroupMember(groupId: Long, userId: Long): JSONObject =
        client.post("groups/$groupId/kick", jsonBody("user_id" to userId))

    suspend fun transferGroupOwner(groupId: Long, newOwnerId: Long): JSONObject =
        client.put("groups/$groupId/transfer", jsonBody("new_owner_id" to newOwnerId))

    suspend fun toggleGroupAdmin(groupId: Long, userId: Long): JSONObject =
        client.put("groups/$groupId/admin/$userId")

    suspend fun muteGroupMember(groupId: Long, userId: Long, minutes: Int): JSONObject =
        client.post("groups/$groupId/mute", jsonBody("user_id" to userId, "minutes" to minutes))

    suspend fun unmuteGroupMember(groupId: Long, userId: Long): JSONObject =
        client.post("groups/$groupId/unmute", jsonBody("user_id" to userId))

    suspend fun toggleGroupDnd(groupId: Long): JSONObject = client.post("groups/$groupId/dnd")

    suspend fun groupMembers(groupId: Long): JSONObject = client.get("groups/$groupId/members")

    suspend fun groupReads(groupId: Long): JSONObject = client.get("groups/$groupId/reads")

    suspend fun createGroupAnnouncement(groupId: Long, content: String): JSONObject =
        client.post("groups/$groupId/announcements", jsonBody("content" to content))

    suspend fun groupAnnouncements(groupId: Long): JSONObject = client.get("groups/$groupId/announcements")

    suspend fun history(peerId: Long, page: Int = 1, pageSize: Int = 30, afterId: Long? = null): JSONObject =
        client.get("history", mapOf("peer_id" to "$peerId", "page" to "$page", "page_size" to "$pageSize", "after_id" to afterId?.toString()))

    suspend fun groupHistory(groupId: Long, page: Int = 1, pageSize: Int = 30, afterId: Long? = null): JSONObject =
        client.get("history/group/$groupId", mapOf("page" to "$page", "page_size" to "$pageSize", "after_id" to afterId?.toString()))

    suspend fun broadcastHistory(page: Int = 1, pageSize: Int = 30): JSONObject =
        client.get("history/broadcast", mapOf("page" to "$page", "page_size" to "$pageSize"))

    suspend fun searchMessages(
        type: String,
        query: String,
        targetId: Long? = null,
        startTime: String? = null,
        endTime: String? = null,
        groupByConversation: Boolean = false,
    ): JSONObject =
        client.get(
            "search/messages",
            mapOf(
                "type" to type,
                "q" to query,
                "target_id" to targetId?.toString(),
                "start_time" to startTime,
                "end_time" to endTime,
                "group_by" to if (groupByConversation) "conversation" else null,
                "page" to "1",
                "page_size" to "50",
            ),
        )

    suspend fun recallMessage(messageId: Long): JSONObject = client.post("messages/$messageId/recall")

    suspend fun deleteMessage(messageId: Long): JSONObject = client.delete("messages/$messageId")

    suspend fun aiBots(): JSONObject = client.get("ai/bots")

    suspend fun createAiBot(body: JSONObject): JSONObject = client.post("ai/bots", body)

    suspend fun updateAiBot(id: Long, body: JSONObject): JSONObject = client.put("ai/bots/$id", body)

    suspend fun deleteAiBot(id: Long): JSONObject = client.delete("ai/bots/$id")

    suspend fun tokenUsages(pageSize: Int = 8): JSONObject =
        client.get("ai/token-usages", mapOf("page_size" to "$pageSize"))

    suspend fun knowledgeBases(): JSONObject = client.get("ai/knowledge-bases")

    suspend fun createKnowledgeBase(name: String, description: String): JSONObject =
        client.post("ai/knowledge-bases", jsonBody("name" to name, "description" to description))

    suspend fun updateKnowledgeBase(id: Long, name: String, description: String): JSONObject =
        client.put("ai/knowledge-bases/$id", jsonBody("name" to name, "description" to description))

    suspend fun deleteKnowledgeBase(id: Long): JSONObject = client.delete("ai/knowledge-bases/$id")

    suspend fun addKnowledgeDocument(baseId: Long, title: String, content: String): JSONObject =
        client.post("ai/knowledge-bases/$baseId/documents", jsonBody("title" to title, "content" to content))

    suspend fun updateKnowledgeDocument(id: Long, title: String, content: String): JSONObject =
        client.put("ai/knowledge-documents/$id", jsonBody("title" to title, "content" to content))

    suspend fun deleteKnowledgeDocument(id: Long): JSONObject = client.delete("ai/knowledge-documents/$id")

    suspend fun settings(): JSONObject = client.get("settings")

    suspend fun updateSettings(settings: JSONObject): JSONObject = client.put("settings", settings)

    suspend fun uploadBytes(fileName: String, contentType: String, bytes: ByteArray): JSONObject =
        client.uploadBytes("upload", fileName, contentType, bytes)
}

fun JSONObject.optArray(name: String): JSONArray =
    optJSONArray(name) ?: JSONArray()
