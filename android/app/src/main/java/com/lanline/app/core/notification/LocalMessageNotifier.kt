package com.lanline.app.core.notification

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import com.lanline.app.MainActivity
import com.lanline.app.R
import com.lanline.app.core.realtime.RealtimeChatMessage

class LocalMessageNotifier(
    private val context: Context,
) {
    private val manager = context.getSystemService(NotificationManager::class.java)

    init {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                MessageChannelId,
                "LanLine 消息",
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply {
                description = "WebSocket 在线消息提醒"
            }
            manager.createNotificationChannel(channel)
        }
    }

    fun notifyMessage(message: RealtimeChatMessage, currentUserId: Long) {
        if (message.fromUserId == currentUserId || !canNotify()) return
        val intent = Intent(context, MainActivity::class.java)
        val pendingIntent = PendingIntent.getActivity(
            context,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = Notification.Builder(context, MessageChannelId)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(message.conversationTitle)
            .setContentText(message.preview.take(80))
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .setShowWhen(true)
            .build()
        manager.notify(message.notificationId(), notification)
    }

    private fun canNotify(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
        return context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
    }

    private fun RealtimeChatMessage.notificationId(): Int {
        val source = if (id > 0) id else receivedAtEpochMillis
        return (source % Int.MAX_VALUE).toInt()
    }

    companion object {
        private const val MessageChannelId = "lanline_messages"
    }
}
