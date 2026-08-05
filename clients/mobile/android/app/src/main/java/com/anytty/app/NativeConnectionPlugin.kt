package com.anytty.app

import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import java.util.concurrent.CancellationException
import java.util.concurrent.atomic.AtomicBoolean
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

@CapacitorPlugin(name = "NativeConnection")
class NativeConnectionPlugin : Plugin() {
    private val runtimeScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var networkMonitor: NativeNetworkMonitor? = null
    private val runtimeCoordinator = NativeConnectionRuntimeCoordinator(
        startRuntime = { NativeConnectionRuntimeOwner.ensureStarted(context.applicationContext) },
        isRuntimeStarted = NativeConnectionRuntimeOwner::isStarted,
        resetRuntime = { NativeConnectionRuntimeOwner.resetLocalPairings(context.applicationContext) },
    )

    override fun load() {
        runtimeCoordinator.load()
        networkMonitor = NativeNetworkMonitor(context.applicationContext) { epoch, connected ->
            AnyTTYDebugLog.event(AnyTTYDebugEvent.NETWORK_CHANGED, epoch)
            notifyListeners(
                "networkChanged",
                JSObject().put("epoch", epoch).put("connected", connected),
                true,
            )
        }.also { it.start() }
        AnyTTYDebugLog.event(AnyTTYDebugEvent.CONNECTION_PLUGIN_LOADED)
    }

    override fun handleOnDestroy() {
        // The process owner keeps the Go engine alive across Activity and WebView recreation.
        networkMonitor?.close()
        networkMonitor = null
        runtimeCoordinator.destroy()
        runtimeScope.cancel("NativeConnectionPlugin destroyed")
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
}
