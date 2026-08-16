package com.anytty.app

import android.net.ConnectivityManager
import android.os.SystemClock
import com.anytty.app.goclient.GoClientNative
import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import java.util.concurrent.CancellationException
import java.util.concurrent.atomic.AtomicBoolean
import java.lang.ref.WeakReference
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

@CapacitorPlugin(name = "NativeConnection")
class NativeConnectionPlugin : Plugin() {
    companion object {
        @Volatile private var loadedPlugin = WeakReference<NativeConnectionPlugin>(null)

        internal fun notifyDisconnectAllRequested() {
            val plugin = loadedPlugin.get() ?: return
            plugin.activity.runOnUiThread {
                plugin.notifyListeners("disconnectAllRequested", JSObject(), true)
            }
        }
    }

    private val runtimeScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var networkMonitor: NativeNetworkMonitor? = null
    private var localDiscovery: NativeLocalDiscovery? = null
    @Volatile private var latestNetworkSnapshot = NativeNetworkSnapshot(
        epoch = 0L,
        connected = true,
        reason = "path_changed",
    )
    private val runtimeCoordinator = NativeConnectionRuntimeCoordinator(
        startRuntime = { NativeConnectionRuntimeOwner.ensureStarted(context.applicationContext) },
        isRuntimeStarted = NativeConnectionRuntimeOwner::isStarted,
        resetRuntime = { NativeConnectionRuntimeOwner.resetLocalPairings(context.applicationContext) },
    )

    override fun load() {
        loadedPlugin = WeakReference(this)
        runtimeCoordinator.load()
        val connectivity = context.applicationContext
            .getSystemService(android.content.Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        latestNetworkSnapshot = NativeNetworkSnapshot(
            epoch = 0L,
            connected = connectivity.activeNetwork != null,
            reason = if (connectivity.activeNetwork == null) "offline" else "path_changed",
        )
        localDiscovery = NativeLocalDiscovery(context.applicationContext) {
            notifyListeners("localDiscoveryChanged", JSObject(), true)
        }.also {
            it.restart(latestNetworkSnapshot.connected)
        }
        networkMonitor = NativeNetworkMonitor(context.applicationContext) { epoch, connected, reason ->
            latestNetworkSnapshot = NativeNetworkSnapshot(epoch, connected, reason)
            if (!connected || reason == "available" || reason == "network_replaced") {
                localDiscovery?.restart(connected)
            }
            AnyTTYDebugLog.event(AnyTTYDebugEvent.NETWORK_CHANGED, epoch)
            notifyListeners(
                "networkChanged",
                nativeNetworkChangedPayload(epoch, connected, reason),
                true,
            )
        }.also { it.start() }
        AnyTTYDebugLog.event(AnyTTYDebugEvent.CONNECTION_PLUGIN_LOADED)
    }

    override fun handleOnDestroy() {
        // The process owner keeps the Go engine alive across Activity and WebView recreation.
        networkMonitor?.close()
        networkMonitor = null
        localDiscovery?.close()
        localDiscovery = null
        runtimeCoordinator.destroy()
        runtimeScope.cancel("NativeConnectionPlugin destroyed")
        if (loadedPlugin.get() === this) loadedPlugin.clear()
        super.handleOnDestroy()
    }

    @PluginMethod
    fun handleForegroundResume(call: PluginCall) {
        try {
            runtimeCoordinator.ensureForForeground()
            call.resolve()
        } catch (failure: Exception) {
            call.reject("Go client engine could not resume", failure)
        }
    }

    @PluginMethod
    fun getNetworkSnapshot(call: PluginCall) {
        val snapshot = latestNetworkSnapshot
        call.resolve(nativeNetworkChangedPayload(snapshot.epoch, snapshot.connected, snapshot.reason))
    }

    @PluginMethod
    fun resetLocalPairings(call: PluginCall) {
        if (!runtimeCoordinator.isReady()) {
            call.reject("native runtime is not available")
            return
        }
        val settled = AtomicBoolean(false)
        val reset = runtimeScope.launch {
            try {
                runtimeCoordinator.resetLocalPairings()
                if (settled.compareAndSet(false, true)) call.resolve()
            } catch (failure: Exception) {
                AnyTTYDebugLog.event(AnyTTYDebugEvent.RESET_PAIRINGS_FAILED)
                runCatching { runtimeCoordinator.ensureForForeground() }
                if (settled.compareAndSet(false, true)) call.reject("failed to reset local pairings", failure)
            }
        }
        reset.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("native runtime was destroyed during reset", failure)
            }
        }
    }

    @PluginMethod
    fun getBridgeEndpoint(call: PluginCall) {
        try {
            val endpoint = NativeConnectionRuntimeOwner.endpoint()
            call.resolve(JSObject().put("port", endpoint.port).put("token", endpoint.token))
        } catch (failure: Exception) {
            call.reject("native bridge server is not ready", failure)
        }
    }

    @PluginMethod
    fun setSessionActive(call: PluginCall) {
        val endpointId = call.getString("machineId").orEmpty().trim()
        if (endpointId.isEmpty()) {
            call.reject("machineId is required")
            return
        }
        val active = call.getBoolean("active", false) ?: false
        NativeConnectionRuntimeOwner.setEndpointActive(context.applicationContext, endpointId, active)
        call.resolve()
    }

    @PluginMethod
    fun isLocalEndpointDiscovered(call: PluginCall) {
        val deviceId = call.getString("deviceId").orEmpty().trim()
        val fingerprint = call.getString("fingerprint").orEmpty().trim()
        if (deviceId.isEmpty() || fingerprint.isEmpty()) {
            call.reject("deviceId and fingerprint are required")
            return
        }
        val candidates = NativeLocalDiscoveryCache.snapshot(deviceId, fingerprint)
        if (candidates.isEmpty()) {
            call.resolve(JSObject().put("discovered", false))
            return
        }
        val settled = AtomicBoolean(false)
        val probe = runtimeScope.launch {
            val startedAt = SystemClock.elapsedRealtime()
            val reachable = GoClientNative.localProbe(nativeLocalDiscoveryResult(candidates).toByteArray())
            AnyTTYDebugLog.connection(
                "local_discovery probe candidates=${candidates.size} reachable=$reachable probe_ms=${SystemClock.elapsedRealtime() - startedAt}",
            )
            if (settled.compareAndSet(false, true)) {
                call.resolve(JSObject().put("discovered", reachable))
            }
        }
        probe.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("local discovery probe was cancelled", failure)
            }
        }
    }
}

internal data class NativeNetworkSnapshot(
    val epoch: Long,
    val connected: Boolean,
    val reason: String,
)

internal fun nativeNetworkChangedPayload(epoch: Long, connected: Boolean, reason: String): JSObject = JSObject()
    .put("epoch", epoch)
    .put("connected", connected)
    .put("reason", reason)
    .put("scope", "session")
