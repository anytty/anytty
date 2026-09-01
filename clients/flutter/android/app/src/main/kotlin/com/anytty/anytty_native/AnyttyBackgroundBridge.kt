package com.anytty.app

import android.Manifest
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import io.flutter.plugin.common.BinaryMessenger
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.lang.ref.WeakReference

object AnyttyBackgroundBridge {
    private const val CHANNEL_NAME = "com.anytty.app/background"
    private const val EVENT_CHANNEL_ID = "anytty_terminal_events"
    private const val NOTIFICATION_PERMISSION_REQUEST = 7301
    const val EXTRA_ROUTE = "com.anytty.app.extra.ROUTE"

    private var channel: MethodChannel? = null
    private var context: Context? = null
    private var activity = WeakReference<MainActivity>(null)
    private var pendingPermissionResult: MethodChannel.Result? = null
    private var pendingRoute: String? = null

    fun attach(context: Context, messenger: BinaryMessenger) {
        this.context = context.applicationContext
        if (channel != null) return
        channel = MethodChannel(messenger, CHANNEL_NAME).also {
            it.setMethodCallHandler(::handleMethodCall)
        }
    }

    fun attachActivity(activity: MainActivity) {
        this.activity = WeakReference(activity)
    }

    fun detachActivity(activity: MainActivity) {
        if (this.activity.get() === activity) this.activity.clear()
        pendingPermissionResult?.error(
            "activity_unavailable",
            "The notification permission request was interrupted",
            null,
        )
        pendingPermissionResult = null
    }

    fun deliverRoute(route: String?) {
        val normalized = route?.trim().orEmpty()
        if (!normalized.startsWith("/terminal/")) return
        pendingRoute = normalized
        channel?.invokeMethod("openRoute", normalized, object : MethodChannel.Result {
            override fun success(result: Any?) {
                if (pendingRoute == normalized) pendingRoute = null
            }

            override fun error(code: String, message: String?, details: Any?) = Unit

            override fun notImplemented() = Unit
        })
    }

    fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray,
    ): Boolean {
        if (requestCode != NOTIFICATION_PERMISSION_REQUEST) return false
        val result = pendingPermissionResult
        pendingPermissionResult = null
        val index = permissions.indexOf(Manifest.permission.POST_NOTIFICATIONS)
        result?.success(index >= 0 && grantResults.getOrNull(index) == PackageManager.PERMISSION_GRANTED)
        return true
    }

    private fun handleMethodCall(call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            "takePendingRoute" -> {
                val route = pendingRoute
                pendingRoute = null
                result.success(route)
            }

            "syncState" -> syncState(call, result)
            "notificationsAuthorized" -> result.success(notificationsAuthorized())
            "requestNotificationAuthorization" -> requestNotificationAuthorization(result)
            "showNotification" -> showNotification(call, result)
            else -> result.notImplemented()
        }
    }

    private fun syncState(call: MethodCall, result: MethodChannel.Result) {
        val appContext = context
        if (appContext == null) {
            result.error("background_unavailable", "Background host is not initialized", null)
            return
        }
        val foreground = call.argument<Boolean>("foreground") ?: true
        val enabled = call.argument<Boolean>("enabled") ?: false
        val rawEndpoints = call.argument<List<Map<String, Any?>>>("endpoints").orEmpty()
        val endpoints = rawEndpoints.mapNotNull { value ->
            val endpointId = value["endpointId"] as? String ?: return@mapNotNull null
            if (endpointId.isBlank()) return@mapNotNull null
            ConnectionForegroundService.EndpointState(
                endpointId = endpointId,
                phase = (value["phase"] as? String).orEmpty(),
            )
        }
        try {
            if (enabled && !foreground && endpoints.isNotEmpty()) {
                ConnectionForegroundService.update(appContext, endpoints)
            } else {
                ConnectionForegroundService.stop(appContext)
            }
            result.success(null)
        } catch (error: Exception) {
            result.error("background_failed", "Unable to update background connections", error.message)
        }
    }

    private fun notificationsAuthorized(): Boolean {
        val appContext = context ?: return false
        return Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU ||
            appContext.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) ==
            PackageManager.PERMISSION_GRANTED
    }

    private fun requestNotificationAuthorization(result: MethodChannel.Result) {
        if (notificationsAuthorized()) {
            result.success(true)
            return
        }
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
            result.success(true)
            return
        }
        val activeActivity = activity.get()
        if (activeActivity == null) {
            result.error("activity_unavailable", "Open AnyTTY to allow notifications", null)
            return
        }
        if (pendingPermissionResult != null) {
            result.error("permission_busy", "A notification permission request is active", null)
            return
        }
        pendingPermissionResult = result
        activeActivity.requestPermissions(
            arrayOf(Manifest.permission.POST_NOTIFICATIONS),
            NOTIFICATION_PERMISSION_REQUEST,
        )
    }

    private fun showNotification(call: MethodCall, result: MethodChannel.Result) {
        val appContext = context
        if (appContext == null) {
            result.error("notification_unavailable", "Notification host is not initialized", null)
            return
        }
        if (!notificationsAuthorized()) {
            result.success(null)
            return
        }
        val id = call.argument<String>("id")?.trim().orEmpty()
        val title = call.argument<String>("title")?.trim().orEmpty()
        val body = call.argument<String>("body")?.trim().orEmpty()
        val route = call.argument<String>("route")?.trim().orEmpty()
        if (id.isEmpty() || title.isEmpty() || !route.startsWith("/terminal/")) {
            result.error("invalid_notification", "Notification data is incomplete", null)
            return
        }
        val manager = appContext.getSystemService(NotificationManager::class.java)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            manager.createNotificationChannel(
                NotificationChannel(
                    EVENT_CHANNEL_ID,
                    "Terminal activity",
                    NotificationManager.IMPORTANCE_DEFAULT,
                ),
            )
        }
        val openIntent = Intent(appContext, MainActivity::class.java).apply {
            action = Intent.ACTION_VIEW
            putExtra(EXTRA_ROUTE, route)
            flags = Intent.FLAG_ACTIVITY_CLEAR_TOP or Intent.FLAG_ACTIVITY_SINGLE_TOP
        }
        val pendingIntent = PendingIntent.getActivity(
            appContext,
            id.hashCode(),
            openIntent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(appContext, EVENT_CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(appContext)
        }
        val notification = builder
            .setSmallIcon(R.drawable.ic_stat_terminal)
            .setContentTitle(title)
            .setContentText(body)
            .setStyle(Notification.BigTextStyle().bigText(body))
            .setContentIntent(pendingIntent)
            .setAutoCancel(true)
            .setCategory(Notification.CATEGORY_EVENT)
            .build()
        manager.notify(id.hashCode() and Int.MAX_VALUE, notification)
        result.success(null)
    }
}
