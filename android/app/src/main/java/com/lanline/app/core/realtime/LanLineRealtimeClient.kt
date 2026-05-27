package com.lanline.app.core.realtime

import com.lanline.app.core.auth.AuthSession
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import java.util.concurrent.TimeUnit
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONArray
import org.json.JSONObject

class LanLineRealtimeClient(
    private val session: AuthSession,
    private val onStateChange: (RealtimeConnectionState) -> Unit,
    private val onChatMessage: (RealtimeChatMessage) -> Unit,
) {
    private val client = OkHttpClient.Builder()
        .pingInterval(25, TimeUnit.SECONDS)
        .retryOnConnectionFailure(true)
        .build()
    private var webSocket: WebSocket? = null
    private var manuallyClosed = false
    private var reconnectAttempt = 0

    fun connect() {
        manuallyClosed = false
        reconnectAttempt = 0
        openSocket(RealtimeConnectionState.Connecting)
    }

    fun disconnect() {
        manuallyClosed = true
        webSocket?.close(NormalClosureCode, "LanLine logout")
        webSocket = null
        onStateChange(RealtimeConnectionState.Closed)
    }

    fun sendChat(payload: JSONObject, requestId: String = nextRequestId()): Boolean =
        sendWrapped("chat", requestId, payload)

    fun sendTyping(toUserId: Long = 0, groupId: Long = 0, isTyping: Boolean): Boolean =
        sendWrapped(
            "typing",
            nextRequestId(),
            JSONObject()
                .put("to_user_id", toUserId)
                .put("group_id", groupId)
                .put("is_typing", isTyping),
        )

    fun sendReadReceipt(peerUserId: Long = 0, groupId: Long = 0, messageIds: List<Long> = emptyList(), lastReadMsgId: Long = 0): Boolean =
        sendWrapped(
            "read_receipt",
            nextRequestId(),
            JSONObject()
                .put("peer_user_id", peerUserId)
                .put("group_id", groupId)
                .put("message_ids", JSONArray(messageIds))
                .put("last_read_msg_id", lastReadMsgId),
        )

    private fun openSocket(state: RealtimeConnectionState) {
        onStateChange(state)
        val request = Request.Builder()
            .url(session.websocketUrl.withToken(session.token))
            .build()
        webSocket = client.newWebSocket(request, listener())
    }

    private fun sendWrapped(type: String, requestId: String, payload: JSONObject): Boolean {
        val message = JSONObject()
            .put("type", type)
            .put("request_id", requestId)
            .put("payload", payload)
            .toString()
        return webSocket?.send(message) == true
    }

    private fun nextRequestId(): String =
        "${System.currentTimeMillis().toString(36)}-${++requestSeq}"

    private fun listener(): WebSocketListener =
        object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                reconnectAttempt = 0
                onStateChange(RealtimeConnectionState.Connected)
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                parseChatMessage(text)?.let(onChatMessage)
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                onStateChange(RealtimeConnectionState.Closed)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                if (manuallyClosed) return
                onStateChange(RealtimeConnectionState.Failed)
                scheduleReconnect()
            }
        }

    private fun scheduleReconnect() {
        reconnectAttempt += 1
        val delayMillis = minOf(30_000L, 1_500L * reconnectAttempt)
        Thread {
            Thread.sleep(delayMillis)
            if (!manuallyClosed) {
                openSocket(RealtimeConnectionState.Reconnecting)
            }
        }.start()
    }

    private fun parseChatMessage(text: String): RealtimeChatMessage? =
        runCatching {
            val root = JSONObject(text)
            if (root.optString("type") != "chat") return@runCatching null
            val payload = root.optJSONObject("payload") ?: return@runCatching null
            val fromUser = payload.optJSONObject("from_user")
            RealtimeChatMessage(
                id = payload.optLong("id"),
                type = payload.optString("type", "text"),
                fromUserId = payload.optLong("from_user_id"),
                fromName = fromUser?.optString("nickname")
                    ?.ifBlank { fromUser.optString("username") }
                    .orEmpty(),
                toUserId = payload.optNullableLong("to_user_id"),
                groupId = payload.optNullableLong("group_id"),
                content = payload.optString("content"),
                fileName = payload.optString("file_name"),
                createdAt = payload.optString("created_at"),
                receivedAtEpochMillis = System.currentTimeMillis(),
            )
        }.getOrNull()

    private fun String.withToken(token: String): String {
        val encoded = URLEncoder.encode(token, StandardCharsets.UTF_8.name())
        val separator = if (contains("?")) "&" else "?"
        return "$this${separator}token=$encoded"
    }

    private fun JSONObject.optNullableLong(name: String): Long? =
        if (isNull(name) || !has(name)) null else optLong(name)

    companion object {
        private const val NormalClosureCode = 1000
        private var requestSeq = 0
    }
}
