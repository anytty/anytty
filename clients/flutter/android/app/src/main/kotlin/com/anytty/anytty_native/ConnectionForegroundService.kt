package com.anytty.app

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.IBinder

class ConnectionForegroundService : Service() {
    data class EndpointState(val endpointId: String, val phase: String)

    override fun onCreate() {
        super.onCreate()
        createChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return START_NOT_STICKY
        }
        val endpoints = intent?.getStringArrayListExtra(EXTRA_ENDPOINTS)
            .orEmpty()
            .mapNotNull(::decodeEndpoint)
        if (endpoints.isEmpty()) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
            return START_NOT_STICKY
        }
        val notification = buildNotification(endpoints)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_CONNECTED_DEVICE,
            )
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
        return START_NOT_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                "Active connections",
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = "Ongoing AnyTTY device connections"
                setShowBadge(false)
            },
        )
    }

    private fun buildNotification(endpoints: List<EndpointState>): Notification {
        val readyCount = endpoints.count { it.phase.equals("ready", ignoreCase = true) }
        val count = endpoints.size
        val title = if (readyCount > 0) {
            "$readyCount device ${if (readyCount == 1) "connection" else "connections"} active"
        } else {
            "Keeping $count ${if (count == 1) "device" else "devices"} available"
        }
        val openIntent = Intent(this, MainActivity::class.java).apply {
            action = Intent.ACTION_MAIN
            flags = Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            openIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }
        return builder
            .setSmallIcon(R.drawable.ic_stat_terminal)
            .setContentTitle(title)
            .setContentText("Tap to return to AnyTTY")
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .setOnlyAlertOnce(true)
            .setCategory(Notification.CATEGORY_SERVICE)
            .setVisibility(Notification.VISIBILITY_PRIVATE)
            .build()
    }

    companion object {
        private const val CHANNEL_ID = "anytty_active_connections"
        private const val NOTIFICATION_ID = 4101
        private const val ACTION_SYNC = "com.anytty.app.background.SYNC"
        private const val ACTION_STOP = "com.anytty.app.background.STOP"
        private const val EXTRA_ENDPOINTS = "com.anytty.app.extra.ENDPOINTS"
        private const val SEPARATOR = '\u001f'

        fun update(context: Context, endpoints: List<EndpointState>) {
            val intent = Intent(context, ConnectionForegroundService::class.java).apply {
                action = ACTION_SYNC
                putStringArrayListExtra(
                    EXTRA_ENDPOINTS,
                    ArrayList(endpoints.map(::encodeEndpoint)),
                )
            }
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, ConnectionForegroundService::class.java))
        }

        private fun encodeEndpoint(endpoint: EndpointState): String =
            endpoint.endpointId + SEPARATOR + endpoint.phase

        private fun decodeEndpoint(value: String): EndpointState? {
            val separator = value.indexOf(SEPARATOR)
            if (separator <= 0) return null
            return EndpointState(
                endpointId = value.substring(0, separator),
                phase = value.substring(separator + 1),
            )
        }
    }
}
