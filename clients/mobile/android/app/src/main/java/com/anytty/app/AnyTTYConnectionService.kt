package com.anytty.app

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

internal fun finishDisconnectAllServiceRequest(
    canStopService: () -> Boolean,
    stopService: () -> Unit,
): Boolean {
    if (!canStopService()) return false
    stopService()
    return true
}

internal fun disconnectAllCompletionOwnsForeground(
    acceptedRuntimeGeneration: Long,
    currentRuntimeGeneration: Long,
    hasActiveEndpoints: Boolean,
): Boolean = !hasActiveEndpoints && acceptedRuntimeGeneration == currentRuntimeGeneration

class AnyTTYConnectionService : Service() {
    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    companion object {
        private const val CHANNEL_ID = "anytty_connection"
        private const val NOTIFICATION_ID = 1001
        private const val ACTION_DISCONNECT_ALL = "com.anytty.app.action.DISCONNECT_ALL"

        fun start(context: Context) {
            ContextCompat.startForegroundService(context, Intent(context, AnyTTYConnectionService::class.java))
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, AnyTTYConnectionService::class.java))
        }
    }

    override fun onCreate() {
        super.onCreate()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(NotificationChannel(
                CHANNEL_ID,
                getString(R.string.connection_notification_channel),
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = getString(R.string.connection_notification_channel_description)
                setShowBadge(false)
            })
        }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_DISCONNECT_ALL) {
            val request = NativeConnectionRuntimeOwner.acceptDisconnectAll()
            serviceScope.launch {
                request.closeDetachedRuntime()
                withContext(Dispatchers.Main.immediate) {
                    NativeConnectionRuntimeOwner.finishDisconnectAll(request) {
                        stopForeground(STOP_FOREGROUND_REMOVE)
                        stopSelf(startId)
                    }
                }
            }
            return START_NOT_STICKY
        }
        val launchIntent = Intent(this, MainActivity::class.java).apply {
            addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP)
        }
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            launchIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val disconnectIntent = PendingIntent.getService(
            this,
            1,
            Intent(this, AnyTTYConnectionService::class.java).setAction(ACTION_DISCONNECT_ALL),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(getString(R.string.app_name))
            .setContentText(getString(R.string.connection_notification_text))
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .setSilent(true)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .addAction(R.drawable.ic_stop_connection, getString(R.string.connection_notification_stop), disconnectIntent)
            .build()
        startForeground(NOTIFICATION_ID, notification)
        if (!NativeConnectionRuntimeOwner.hasActiveEndpoints()) {
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf(startId)
        }
        return START_NOT_STICKY
    }

    override fun onDestroy() {
        serviceScope.cancel()
        super.onDestroy()
    }

    override fun onBind(intent: Intent?): IBinder? = null
}
