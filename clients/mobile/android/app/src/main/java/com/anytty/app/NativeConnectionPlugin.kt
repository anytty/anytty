package com.anytty.app

import android.content.ClipData
import android.content.ContentResolver
import android.content.Intent
import android.os.SystemClock
import android.util.Base64
import com.anytty.app.goclient.GoClientNative
import com.getcapacitor.JSObject
import com.getcapacitor.JSArray
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.CapacitorPlugin
import java.util.concurrent.CancellationException
import java.util.concurrent.atomic.AtomicBoolean
import java.io.File
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@CapacitorPlugin(name = "NativeConnection")
class NativeConnectionPlugin : Plugin() {
    private val runtimeScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var networkProjectionSubscription: AutoCloseable? = null
    private var disconnectAllSubscription: AutoCloseable? = null
    private var localDiscovery: NativeLocalDiscovery? = null
    private var rendererDemand: NativeRendererDemandSnapshot? = null
    private var projectedNetworkEpoch = -1L
    @Volatile private var acceptingEvents = true
    private val runtimeCoordinator = NativeConnectionRuntimeCoordinator(
        startRuntime = { NativeConnectionRuntimeOwner.ensureStarted(context.applicationContext) },
        isRuntimeStarted = NativeConnectionRuntimeOwner::isStarted,
        resetRuntime = { NativeConnectionRuntimeOwner.resetLocalPairings(context.applicationContext) },
    )

    override fun load() {
        try {
            acceptingEvents = true
            runtimeCoordinator.load()?.let {
                AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED)
            }
            adoptDemandSnapshot(NativeConnectionRuntimeOwner.attachRenderer(context.applicationContext))
            val initialNetwork = NativeConnectionRuntimeOwner.networkSnapshot()
            projectedNetworkEpoch = initialNetwork.epoch
            localDiscovery = NativeLocalDiscovery(context.applicationContext) {
                notifyListeners("localDiscoveryChanged", JSObject(), false)
            }.also {
                it.restart(initialNetwork.connected)
            }
            networkProjectionSubscription = NativeConnectionRuntimeOwner.observeNetwork { projection ->
                activity.runOnUiThread {
                    if (networkProjectionSubscription == null) return@runOnUiThread
                    val (epoch, connected, reason) = projection
                    if (epoch <= projectedNetworkEpoch) return@runOnUiThread
                    projectedNetworkEpoch = epoch
                    if (!connected || reason == "available" || reason == "network_replaced") {
                        localDiscovery?.restart(connected)
                    }
                    notifyListeners(
                        "networkChanged",
                        nativeNetworkChangedPayload(epoch, connected, reason),
                        false,
                    )
                }
            }
            // Close the snapshot-to-subscription window. A transition delivered before
            // the subscription field was assigned is recovered by its monotonic epoch.
            val reconciledNetwork = NativeConnectionRuntimeOwner.networkSnapshot()
            if (reconciledNetwork.epoch > projectedNetworkEpoch) {
                projectedNetworkEpoch = reconciledNetwork.epoch
                if (
                    !reconciledNetwork.connected ||
                    reconciledNetwork.reason == "available" ||
                    reconciledNetwork.reason == "network_replaced"
                ) {
                    localDiscovery?.restart(reconciledNetwork.connected)
                }
            }
            disconnectAllSubscription = NativeConnectionRuntimeOwner.observeDisconnectAll(
                ::notifyDisconnectAllRequested,
            )
            AnyTTYDebugLog.event(AnyTTYDebugEvent.CONNECTION_PLUGIN_LOADED)
        } catch (failure: Exception) {
            disconnectAllSubscription?.close()
            disconnectAllSubscription = null
            networkProjectionSubscription?.close()
            networkProjectionSubscription = null
            localDiscovery?.close()
            localDiscovery = null
            detachRendererDemand()
            runtimeCoordinator.destroy()
            throw failure
        }
    }

    override fun handleOnDestroy() {
        // The process owner keeps the Go engine alive across Activity and WebView recreation.
        acceptingEvents = false
        disconnectAllSubscription?.close()
        disconnectAllSubscription = null
        networkProjectionSubscription?.close()
        networkProjectionSubscription = null
        localDiscovery?.close()
        localDiscovery = null
        detachRendererDemand()
        runtimeCoordinator.destroy()
        runtimeScope.cancel("NativeConnectionPlugin destroyed")
        super.handleOnDestroy()
    }

    private fun notifyDisconnectAllRequested(snapshot: NativeRendererDemandSnapshot) {
        activity.runOnUiThread {
            if (!acceptingEvents) return@runOnUiThread
            notifyListeners(
                "disconnectAllRequested",
                nativeDisconnectAllRequestedPayload(snapshot),
                true,
            )
        }
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
                NativeConnectionRuntimeOwner.refreshForegroundNetwork(context.applicationContext)
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
    fun requestEndpointRecovery(call: PluginCall) {
        val endpointId = call.getString("endpointId").orEmpty().trim()
        if (endpointId.isEmpty()) {
            call.reject("endpointId is required")
            return
        }
        val settled = AtomicBoolean(false)
        val repair = runtimeScope.launch {
            try {
                runtimeCoordinator.ensureForForeground()
                NativeConnectionRuntimeOwner.requestEndpointRecovery(context.applicationContext, endpointId)
                if (settled.compareAndSet(false, true)) call.resolve()
            } catch (failure: Exception) {
                if (settled.compareAndSet(false, true)) call.reject("endpoint recovery could not be requested", failure)
            }
        }
        repair.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("native runtime was destroyed during endpoint recovery", failure)
            }
        }
    }

    @PluginMethod
    fun getNetworkSnapshot(call: PluginCall) {
        val snapshot = NativeConnectionRuntimeOwner.networkSnapshot()
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
        val settled = AtomicBoolean(false)
        val lookup = runtimeScope.launch {
            try {
                runtimeCoordinator.ensureForBridgeEndpoint()
                val endpoint = NativeConnectionRuntimeOwner.endpoint()
                if (settled.compareAndSet(false, true)) {
                    call.resolve(JSObject().put("port", endpoint.port).put("token", endpoint.token))
                }
            } catch (failure: Exception) {
                if (settled.compareAndSet(false, true)) {
                    call.reject("native bridge server is not ready", failure)
                }
            }
        }
        lookup.invokeOnCompletion { failure ->
            if (failure is CancellationException && settled.compareAndSet(false, true)) {
                call.reject("native runtime was destroyed while resolving the bridge endpoint", failure)
            }
        }
    }

    @PluginMethod
    @Synchronized
    fun getSessionDemandLease(call: PluginCall) {
        try {
            val current = rendererDemand
                ?: throw IllegalStateException("renderer demand attachment is unavailable")
            val snapshot = NativeConnectionRuntimeOwner.rendererDemandSnapshot(current.attachmentId)
            rendererDemand = snapshot
            call.resolve(nativeDemandLeasePayload(snapshot))
        } catch (failure: Exception) {
            call.reject("renderer demand attachment is unavailable", failure)
        }
    }

    @PluginMethod
    fun acknowledgeDisconnectAll(call: PluginCall) {
        val stopEpoch = call.getString("stopEpoch")?.toLongOrNull()
        if (stopEpoch == null || stopEpoch < 0L) {
            call.reject("a valid disconnect-all Stop epoch is required")
            return
        }
        try {
            NativeConnectionRuntimeOwner.acknowledgeDisconnectAll(stopEpoch)
            call.resolve()
        } catch (failure: Exception) {
            call.reject("disconnect-all cleanup acknowledgement was rejected", failure)
        }
    }

    @PluginMethod
    @Synchronized
    fun resumeSessionDemand(call: PluginCall) {
        val intentId = call.getString("intentId").orEmpty().trim()
        val baseStopEpoch = call.getString("baseStopEpoch")?.toLongOrNull()
        if (intentId.isEmpty() || intentId.length > 128 || baseStopEpoch == null || baseStopEpoch < 0L) {
            call.reject("a valid renderer resume intent ID is required")
            return
        }
        try {
            val current = rendererDemand
                ?: throw IllegalStateException("renderer demand attachment is unavailable")
            val resumed = NativeConnectionRuntimeOwner.resumeRendererDemand(
                current.attachmentId,
                intentId,
                baseStopEpoch,
            )
            adoptDemandSnapshot(resumed.snapshot)
            call.resolve(nativeDemandLeasePayload(resumed.snapshot).put("outcome", resumed.outcome.wireValue))
        } catch (failure: Exception) {
            rendererDemand?.let { current ->
                runCatching { NativeConnectionRuntimeOwner.rendererDemandSnapshot(current.attachmentId) }
                    .getOrNull()
                    ?.let(::adoptDemandSnapshot)
            }
            call.reject("failed to resume renderer connection demand", failure)
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
        val attachmentId = call.getString("attachmentId").orEmpty().trim()
        val baseDemandRevision = call.getString("baseDemandRevision")?.toLongOrNull()
        if (attachmentId.isEmpty() || baseDemandRevision == null || baseDemandRevision < 0L) {
            call.reject("a valid native demand lease is required")
            return
        }
        val endpointIds = linkedSetOf<String>()
        try {
            for (index in 0 until values.length()) {
                val endpointId = (values.get(index) as? String)?.trim().orEmpty()
                if (endpointId.isEmpty()) throw IllegalArgumentException("endpointIds contains an invalid value")
                endpointIds += endpointId
            }
            val next = NativeConnectionRuntimeOwner.replaceRendererDemand(
                context.applicationContext,
                attachmentId,
                baseDemandRevision,
                endpointIds,
            )
            adoptDemandSnapshot(next)
            call.resolve(nativeDemandLeasePayload(next))
        } catch (failure: Exception) {
            rendererDemand?.let { current ->
                runCatching { NativeConnectionRuntimeOwner.rendererDemandSnapshot(current.attachmentId) }
                    .getOrNull()
                    ?.let(::adoptDemandSnapshot)
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
    private fun adoptDemandSnapshot(snapshot: NativeRendererDemandSnapshot) {
        val current = rendererDemand
        if (current == null || current.attachmentId == snapshot.attachmentId) {
            rendererDemand = snapshot
        }
    }

    @Synchronized
    private fun detachRendererDemand() {
        val current = rendererDemand ?: return
        rendererDemand = null
        NativeConnectionRuntimeOwner.detachRenderer(current.attachmentId)
    }
}

internal fun nativeNetworkChangedPayload(epoch: Long, connected: Boolean, reason: String): JSObject = JSObject()
    .put("epoch", epoch)
    .put("connected", connected)
    .put("reason", reason)
    .put("scope", "session")

internal fun nativeDemandLeasePayload(snapshot: NativeRendererDemandSnapshot): JSObject {
    val endpointIds = JSArray()
    snapshot.endpointIds.sorted().forEach(endpointIds::put)
    return JSObject()
        .put("attachmentId", snapshot.attachmentId)
        .put("demandRevision", snapshot.demandRevision.toString())
        .put("stopEpoch", snapshot.stopEpoch.toString())
        .put("endpointIds", endpointIds)
        .put("stopped", snapshot.stopped)
}

internal fun nativeDisconnectAllRequestedPayload(snapshot: NativeRendererDemandSnapshot): JSObject = JSObject()
    .put("stopEpoch", snapshot.stopEpoch.toString())
    .put("stopped", snapshot.stopped)
