package com.anytty.app

import android.content.Context
import android.util.Base64
import com.anytty.app.goclient.AndroidClientAccessCredentialStore
import com.anytty.app.goclient.AndroidEndpointRegistryStore
import com.anytty.app.goclient.AndroidGoClientEngine
import com.anytty.app.goclient.AndroidSSHCredentialStore
import com.anytty.app.goclient.GoClientNative
import anytty.client.binding.v1.ClientBinding
import java.security.SecureRandom

internal data class NativeSupervisorDemandResult(
    val snapshot: NativeRendererDemandSnapshot,
    val goManagedEndpointIds: Set<String>,
)

internal object NativeConnectionRuntimeOwner {
    private const val BRIDGE_TOKEN_BYTES = 32

    internal data class BridgeEndpoint(val port: Int, val token: String)

    private var bridgeEndpoint: BridgeEndpoint? = null
    private var goEngine: AndroidGoClientEngine? = null
    private val rendererDemand = NativeRendererDemandState()
    private var hostRevision = 0L
    private var networkConnected = true
    private var networkReason = "path_changed"

    @Synchronized
    fun isStarted(): Boolean = bridgeEndpoint != null && goEngine != null

    @Synchronized
    fun ensureStarted(context: Context): BridgeEndpoint {
        bridgeEndpoint?.let { endpoint ->
            if (goEngine != null) return endpoint
        }

        val token = generateBridgeToken()
        val engine = AndroidGoClientEngine(context.applicationContext)
        try {
            val port = GoClientNative.startBridge(engine.handle, token)
            val endpoint = BridgeEndpoint(port, token)
            goEngine = engine
            bridgeEndpoint = endpoint
            replaceGoDemandLocked(rendererDemand.canonicalSnapshot())
            replayHostSignalLocked()
            AnyTTYDebugLog.event(AnyTTYDebugEvent.BRIDGE_STARTED)
            return endpoint
        } catch (failure: Exception) {
            runCatching { engine.close() }
            goEngine = null
            bridgeEndpoint = null
            throw failure
        }
    }

    @Synchronized
    fun endpoint(): BridgeEndpoint = bridgeEndpoint
        ?: throw IllegalStateException("native bridge server is not ready")

    @Synchronized
    fun resetLocalPairings(context: Context): BridgeEndpoint {
        AndroidEndpointRegistryStore(context.applicationContext).clear()
        AndroidClientAccessCredentialStore(context.applicationContext).clearAll()
        AndroidSSHCredentialStore().clearAll()
        stopRuntimeLocked()
        return ensureStarted(context.applicationContext)
    }

    @Synchronized
    fun attachRenderer(): NativeSupervisorDemandResult {
        val snapshot = rendererDemand.attachRenderer()
        replaceGoDemandLocked(snapshot)
        return supervisorDemandResult(snapshot)
    }

    @Synchronized
    fun detachRenderer(attachmentId: String) {
        rendererDemand.detachRenderer(attachmentId)
    }

    @Synchronized
    fun replaceRendererDemand(
        context: Context,
        attachmentId: String,
        baseDemandRevision: Long,
        endpointIds: Set<String>,
    ): NativeSupervisorDemandResult {
        val previous = rendererDemand.currentSnapshot(attachmentId)
        val next = rendererDemand.replaceDemand(attachmentId, baseDemandRevision, endpointIds)
        try {
            replaceGoDemandLocked(next)
            if (next.endpointIds.isNotEmpty()) {
                AnyTTYConnectionService.start(context.applicationContext)
            } else if (previous.endpointIds.isNotEmpty()) {
                AnyTTYConnectionService.stop(context.applicationContext)
            }
            return supervisorDemandResult(next)
        } catch (failure: Exception) {
            val restored = rendererDemand.replaceDemand(attachmentId, next.demandRevision, previous.endpointIds)
            runCatching { replaceGoDemandLocked(restored) }
            throw failure
        }
    }

    @Synchronized
    fun rendererDemandSnapshot(attachmentId: String): NativeSupervisorDemandResult =
        supervisorDemandResult(rendererDemand.currentSnapshot(attachmentId))

    @Synchronized
    fun hasActiveEndpoints(): Boolean = rendererDemand.hasDemand()

    @Synchronized
    fun signalNetworkChanged(connected: Boolean, reason: String) {
        networkConnected = connected
        networkReason = reason.trim().ifEmpty { if (connected) "path_changed" else "offline" }
        incrementHostRevisionLocked()
        signalGoSupervisorLocked(foreground = false)
    }

    fun handleForegroundResume(context: Context, timeoutMillis: Int) {
        val engineHandle = synchronized(this) {
            ensureStarted(context.applicationContext)
            incrementHostRevisionLocked()
            signalGoSupervisorLocked(foreground = true)
            goEngine?.handle ?: throw IllegalStateException("Go client engine is unavailable")
        }
        GoClientNative.awaitSupervisorReady(engineHandle, timeoutMillis)
    }

    @Synchronized
    fun endpointSupervisorSnapshot(): ByteArray {
        val engine = goEngine ?: return ByteArray(0)
        return GoClientNative.supervisorSnapshot(engine.handle)
    }

    fun requestDisconnectAll() {
        var snapshot: NativeRendererDemandSnapshot? = null
        try {
            synchronized(this) {
                snapshot = rendererDemand.clearDemand()
                stopRuntimeLocked()
            }
        } catch (_: Exception) {
            AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED)
        } finally {
            NativeConnectionPlugin.notifyDisconnectAllRequested(snapshot)
        }
    }

    @Synchronized
    internal fun stopForTests() {
        rendererDemand.resetForTests()
        hostRevision = 0L
        networkConnected = true
        networkReason = "path_changed"
        stopRuntimeLocked()
    }

    private fun stopRuntimeLocked() {
        bridgeEndpoint = null
        val engine = goEngine
        goEngine = null
        engine?.close()
    }

    private fun replaceGoDemandLocked(snapshot: NativeRendererDemandSnapshot) {
        val engine = goEngine ?: return
        val demand = ClientBinding.EndpointSupervisorDemandSnapshot.newBuilder()
            .setAttachmentId(snapshot.attachmentId)
            .setDemandRevision(snapshot.demandRevision)
        snapshot.endpointIds.sorted().forEach { endpointId ->
            val mode = if (NativeEndpointSupervisorRollout.isTakeover(endpointId)) {
                ClientBinding.EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_TAKEOVER
            } else {
                ClientBinding.EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_SHADOW
            }
            demand.addEndpoints(ClientBinding.EndpointSupervisorDemand.newBuilder()
                .setEndpointId(endpointId)
                .setMode(mode))
        }
        GoClientNative.replaceSupervisorDemand(engine.handle, demand.build().toByteArray())
    }

    private fun supervisorDemandResult(snapshot: NativeRendererDemandSnapshot): NativeSupervisorDemandResult =
        NativeSupervisorDemandResult(
            snapshot = snapshot,
            goManagedEndpointIds = snapshot.endpointIds.filterTo(linkedSetOf()) {
                NativeEndpointSupervisorRollout.isTakeover(it)
            },
        )

    private fun replayHostSignalLocked() {
        if (hostRevision == 0L) return
        signalGoSupervisorLocked(foreground = false)
    }

    private fun signalGoSupervisorLocked(foreground: Boolean) {
        val engine = goEngine ?: return
        val signal = ClientBinding.EndpointSupervisorHostSignal.newBuilder()
            .setRevision(hostRevision)
            .setConnected(networkConnected)
            .setReason(networkReason)
            .setForeground(foreground)
            .build()
        GoClientNative.signalSupervisor(engine.handle, signal.toByteArray())
    }

    private fun incrementHostRevisionLocked() {
        check(hostRevision < Long.MAX_VALUE) { "endpoint supervisor host revision is exhausted" }
        hostRevision += 1
    }

    private fun generateBridgeToken(): String {
        val bytes = ByteArray(BRIDGE_TOKEN_BYTES)
        SecureRandom().nextBytes(bytes)
        return Base64.encodeToString(bytes, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
    }
}
