package com.anytty.app

import android.content.ClipData
import android.content.ContentResolver
import android.content.Intent
import android.net.ConnectivityManager
import android.os.SystemClock
import android.util.Base64
import com.anytty.app.goclient.GoClientNative
import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import java.util.concurrent.CancellationException
import java.util.concurrent.atomic.AtomicBoolean
import java.lang.ref.WeakReference
import java.io.File
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@CapacitorPlugin(name = "NativeConnection")
class NativeConnectionPlugin : Plugin() {
    companion object {
        // The supervisor keeps recovering after this observation budget. Foreground UI
        // readiness must not wait on a slow or offline endpoint.
        private const val FOREGROUND_SUPERVISOR_TIMEOUT_MILLIS = 3_000
        @Volatile private var loadedPlugin = WeakReference<NativeConnectionPlugin>(null)

        internal fun notifyDisconnectAllRequested(snapshot: NativeRendererDemandSnapshot?) {
            val plugin = loadedPlugin.get() ?: return
            plugin.activity.runOnUiThread {
                if (snapshot != null) plugin.adoptDemandSnapshot(snapshot, emptySet())
                plugin.notifyListeners("disconnectAllRequested", JSObject(), true)
            }
        }
    }

    private val runtimeScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var networkMonitor: NativeNetworkMonitor? = null
    private var localDiscovery: NativeLocalDiscovery? = null
    private var rendererDemand: NativeRendererDemandSnapshot? = null
    private var goManagedEndpointIds: Set<String> = emptySet()
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
        try {
            runtimeCoordinator.load()
            adoptDemandResult(NativeConnectionRuntimeOwner.attachRenderer(context.applicationContext))
            val connectivity = context.applicationContext
                .getSystemService(android.content.Context.CONNECTIVITY_SERVICE) as ConnectivityManager
            latestNetworkSnapshot = NativeNetworkSnapshot(
                epoch = 0L,
                connected = connectivity.activeNetwork != null,
                reason = if (connectivity.activeNetwork == null) "offline" else "path_changed",
            )
            NativeConnectionRuntimeOwner.signalNetworkChanged(
                latestNetworkSnapshot.connected,
                latestNetworkSnapshot.reason,
            )
            localDiscovery = NativeLocalDiscovery(context.applicationContext) {
                notifyListeners("localDiscoveryChanged", JSObject(), false)
            }.also {
                it.restart(latestNetworkSnapshot.connected)
            }
            networkMonitor = NativeNetworkMonitor(context.applicationContext) { epoch, connected, reason ->
                latestNetworkSnapshot = NativeNetworkSnapshot(epoch, connected, reason)
                runCatching { NativeConnectionRuntimeOwner.signalNetworkChanged(connected, reason) }
                    .onFailure { AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED) }
                if (!connected || reason == "available" || reason == "network_replaced") {
                    localDiscovery?.restart(connected)
                }
                AnyTTYDebugLog.event(AnyTTYDebugEvent.NETWORK_CHANGED, epoch)
                notifyListeners(
                    "networkChanged",
                    nativeNetworkChangedPayload(epoch, connected, reason),
                    false,
                )
            }.also { it.start() }
            loadedPlugin = WeakReference(this)
            AnyTTYDebugLog.event(AnyTTYDebugEvent.CONNECTION_PLUGIN_LOADED)
        } catch (failure: Exception) {
            networkMonitor?.close()
            networkMonitor = null
            localDiscovery?.close()
            localDiscovery = null
            detachRendererDemand()
            runtimeCoordinator.destroy()
            if (loadedPlugin.get() === this) loadedPlugin.clear()
            throw failure
        }
    }

    override fun handleOnDestroy() {
        // The process owner keeps the Go engine alive across Activity and WebView recreation.
        networkMonitor?.close()
        networkMonitor = null
        localDiscovery?.close()
        localDiscovery = null
        detachRendererDemand()
        runtimeCoordinator.destroy()
        runtimeScope.cancel("NativeConnectionPlugin destroyed")
        if (loadedPlugin.get() === this) loadedPlugin.clear()
        super.handleOnDestroy()
    }

    @PluginMethod
    fun writeDebugDiagnostic(call: PluginCall) {
        val value = call.getString("value").orEmpty().trim()
        if (value.isNotEmpty()) AnyTTYDebugLog.connection("web $value")
        call.resolve()
    }

    @PluginMethod
    fun shareDiagnosticBundle(call: PluginCall) {
        val settled = AtomicBoolean(false)
        val sharing = runtimeScope.launch {
            var bundle: AnyTTYDiagnosticBundle? = null
            try {
                val createdBundle = AnyTTYDiagnosticStore.createBundle(context.applicationContext)
                bundle = createdBundle
                val saved = AndroidDownloadStore(context.applicationContext).save(
                    createdBundle.name,
                    "application/zip",
                    createdBundle.file,
                )
                val shareUri = if (saved.uri.scheme == ContentResolver.SCHEME_CONTENT) {
                    saved.uri
                } else {
                    AnyTTYDownloadProvider.uriForFile(
                        context.applicationContext,
                        File(requireNotNull(saved.uri.path) { "diagnostic download path is missing" }),
                    )
                }
                val result = JSObject()
                    .put("name", createdBundle.name)
                    .put("path", saved.path)
                    .put("bytes", saved.bytes)
                    .put("sha256", saved.sha256)
                val shareIntent = Intent(Intent.ACTION_SEND).apply {
                    type = "application/zip"
                    putExtra(Intent.EXTRA_STREAM, shareUri)
                    clipData = ClipData.newUri(context.contentResolver, createdBundle.name, shareUri)
                    addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                }
                try {
                    withContext(Dispatchers.Main) {
                        activity.startActivity(Intent.createChooser(shareIntent, null))
                    }
                } catch (failure: Exception) {
                    AnyTTYDebugLog.connection("diagnostic_bundle share_failed type=${failure.javaClass.simpleName}")
                    if (settled.compareAndSet(false, true)) {
                        call.reject("failed to open the Android share sheet", failure)
                    }
                    return@launch
                }
                AnyTTYDebugLog.connection("diagnostic_bundle shared entries=${createdBundle.entryCount}")
                if (settled.compareAndSet(false, true)) call.resolve(result)
            } catch (failure: Exception) {
                AnyTTYDebugLog.connection("diagnostic_bundle prepare_failed type=${failure.javaClass.simpleName}")
                if (settled.compareAndSet(false, true)) call.reject("failed to prepare diagnostic logs", failure)
            } finally {
                bundle?.file?.delete()
            }
        }
        sharing.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("diagnostic log sharing was cancelled", failure)
            }
        }
    }

    @PluginMethod
    fun handleForegroundResume(call: PluginCall) {
        if (!runtimeCoordinator.isReady()) {
            call.reject("native runtime is not available")
            return
        }
        val settled = AtomicBoolean(false)
        val recovery = runtimeScope.launch {
            try {
                runtimeCoordinator.ensureForForeground()
                NativeConnectionRuntimeOwner.handleForegroundResume(
                    context.applicationContext,
                    FOREGROUND_SUPERVISOR_TIMEOUT_MILLIS,
                )
                if (settled.compareAndSet(false, true)) call.resolve()
            } catch (failure: Exception) {
                if (settled.compareAndSet(false, true)) call.reject("Go client engine could not resume", failure)
            }
        }
        recovery.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("native runtime was destroyed during foreground recovery", failure)
            }
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
    @Synchronized
    fun replaceSessionDemand(call: PluginCall) {
        val values = call.getArray("endpointIds")
        if (values == null) {
            call.reject("endpointIds is required")
            return
        }
        val endpointIds = linkedSetOf<String>()
        try {
            for (index in 0 until values.length()) {
                val endpointId = (values.get(index) as? String)?.trim().orEmpty()
                if (endpointId.isEmpty()) throw IllegalArgumentException("endpointIds contains an invalid value")
                endpointIds += endpointId
            }
            val current = rendererDemand
                ?: throw IllegalStateException("renderer demand attachment is unavailable")
            val next = NativeConnectionRuntimeOwner.replaceRendererDemand(
                context.applicationContext,
                current.attachmentId,
                current.demandRevision,
                endpointIds,
            )
            adoptDemandResult(next)
            call.resolve(JSObject().put("goManagedEndpointIds", next.goManagedEndpointIds.toList()))
        } catch (failure: Exception) {
            rendererDemand?.let { current ->
                runCatching { NativeConnectionRuntimeOwner.rendererDemandSnapshot(current.attachmentId) }
                    .getOrNull()
                    ?.let(::adoptDemandResult)
            }
            call.reject("failed to replace renderer connection demand", failure)
        }
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
            try {
                val startedAt = SystemClock.elapsedRealtime()
                val reachable = GoClientNative.localProbe(nativeLocalDiscoveryResult(candidates).toByteArray())
                AnyTTYDebugLog.connection(
                    "local_discovery probe candidates=${candidates.size} reachable=$reachable probe_ms=${SystemClock.elapsedRealtime() - startedAt}",
                )
                if (settled.compareAndSet(false, true)) {
                    call.resolve(JSObject().put("discovered", reachable))
                }
            } catch (failure: Exception) {
                if (settled.compareAndSet(false, true)) {
                    call.reject("local discovery probe failed", failure)
                }
            }
        }
        probe.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("local discovery probe was cancelled", failure)
            }
        }
    }

    @PluginMethod
    fun isDirectRouteReachable(call: PluginCall) {
        val encoded = call.getString("routeProtoBase64").orEmpty().trim()
        val routeProto = try {
            Base64.decode(encoded, Base64.DEFAULT)
        } catch (failure: IllegalArgumentException) {
            call.reject("routeProtoBase64 is invalid", failure)
            return
        }
        if (routeProto.isEmpty()) {
            call.reject("routeProtoBase64 is required")
            return
        }
        val settled = AtomicBoolean(false)
        val probe = runtimeScope.launch {
            try {
                val startedAt = SystemClock.elapsedRealtime()
                val reachable = GoClientNative.directProbe(routeProto)
                AnyTTYDebugLog.connection(
                    "direct_probe reachable=$reachable probe_ms=${SystemClock.elapsedRealtime() - startedAt}",
                )
                if (settled.compareAndSet(false, true)) {
                    call.resolve(JSObject().put("reachable", reachable))
                }
            } catch (failure: Exception) {
                AnyTTYDebugLog.connection("direct_probe failed type=${failure.javaClass.simpleName}")
                if (settled.compareAndSet(false, true)) {
                    call.reject("Direct TCP probe failed", failure)
                }
            }
        }
        probe.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("Direct TCP probe was cancelled", failure)
            }
        }
    }

    @Synchronized
    private fun adoptDemandResult(result: NativeSupervisorDemandResult) {
        adoptDemandSnapshot(result.snapshot, result.goManagedEndpointIds)
    }

    @Synchronized
    private fun adoptDemandSnapshot(snapshot: NativeRendererDemandSnapshot, managedEndpointIds: Set<String>) {
        val current = rendererDemand
        if (current == null || current.attachmentId == snapshot.attachmentId) {
            rendererDemand = snapshot
            goManagedEndpointIds = managedEndpointIds.toSet()
        }
    }

    @Synchronized
    private fun detachRendererDemand() {
        val current = rendererDemand ?: return
        rendererDemand = null
        goManagedEndpointIds = emptySet()
        NativeConnectionRuntimeOwner.detachRenderer(current.attachmentId)
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
