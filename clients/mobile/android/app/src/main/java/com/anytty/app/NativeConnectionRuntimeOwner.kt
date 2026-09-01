package com.anytty.app

import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Base64
import com.anytty.app.goclient.AndroidClientAccessCredentialStore
import com.anytty.app.goclient.AndroidEndpointRegistryStore
import com.anytty.app.goclient.AndroidGoClientEngine
import com.anytty.app.goclient.AndroidPlatformStorageCoordinator
import com.anytty.app.goclient.AndroidSSHCredentialStore
import com.anytty.app.goclient.GoClientNative
import anytty.client.binding.v1.ClientBinding
import java.security.SecureRandom

internal fun nativeSupervisorDemandSnapshot(
    snapshot: NativeRendererDemandSnapshot,
): ClientBinding.EndpointSupervisorDemandSnapshot {
    val demand = ClientBinding.EndpointSupervisorDemandSnapshot.newBuilder()
        .setAttachmentId(snapshot.attachmentId)
        .setDemandRevision(snapshot.demandRevision)
    snapshot.endpointIds.sorted().forEach { endpointId ->
        demand.addEndpoints(ClientBinding.EndpointSupervisorDemand.newBuilder()
            .setEndpointId(endpointId)
            .setMode(ClientBinding.EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_TAKEOVER))
    }
    return demand.build()
}

internal fun submitNativeDemandOrRecover(
    submit: () -> Unit,
    recoverRuntime: () -> Unit,
) {
    try {
        submit()
    } catch (failure: Exception) {
        recoverRuntime()
        throw failure
    }
}

internal fun maintainNativeDemandForegroundOwnership(
    maintainForegroundService: () -> Unit,
    recoverRuntime: () -> Unit,
) {
    try {
        maintainForegroundService()
    } catch (failure: Exception) {
        recoverRuntime()
        throw failure
    }
}

internal fun replayNativeSupervisorState(
    maintainForegroundOwnership: () -> Unit,
    signalHost: () -> Unit,
    replaceDemand: () -> Unit,
) {
    maintainForegroundOwnership()
    signalHost()
    replaceDemand()
}

internal fun handoffRendererDemand(
    rendererDemand: NativeRendererDemandState,
    replaceGoDemand: (NativeRendererDemandSnapshot) -> Unit,
    ensureRuntimeForDemand: () -> Unit,
    maintainForegroundService: () -> Unit,
): NativeRendererDemandSnapshot {
    val snapshot = rendererDemand.attachRenderer()
    if (snapshot.endpointIds.isNotEmpty()) {
        maintainForegroundService()
    }
    replaceGoDemand(snapshot)
    if (snapshot.endpointIds.isNotEmpty()) ensureRuntimeForDemand()
    return snapshot
}

internal fun reconcileRendererDemand(
    rendererDemand: NativeRendererDemandState,
    attachmentId: String,
    baseDemandRevision: Long,
    endpointIds: Set<String>,
    replaceGoDemand: (NativeRendererDemandSnapshot) -> Unit,
    ensureRuntimeForDemand: () -> Unit,
    maintainForegroundService: () -> Unit,
    stopForegroundService: () -> Unit,
): NativeRendererDemandSnapshot {
    val next = rendererDemand.replaceDemand(attachmentId, baseDemandRevision, endpointIds)
    // Desired intent is process-owned. A transient ABI or FGS failure must not
    // roll it back; the same canonical snapshot is replayed on the next attempt.
    if (next.endpointIds.isNotEmpty()) {
        maintainForegroundService()
    } else {
        stopForegroundService()
    }
    replaceGoDemand(next)
    if (next.endpointIds.isNotEmpty()) ensureRuntimeForDemand()
    return next
}

internal class NativeForegroundRefreshState {
    private var foreground = false
    private var refreshedRuntimeGeneration = 0L
    private var refreshingRuntimeGeneration = 0L

    fun markForeground() {
        if (!foreground) {
            refreshedRuntimeGeneration = 0L
            refreshingRuntimeGeneration = 0L
        }
        foreground = true
    }

    fun beginRefresh(runtimeGeneration: Long): Boolean {
        check(runtimeGeneration > 0L) { "native runtime generation is unavailable" }
        if (
            !foreground ||
            refreshedRuntimeGeneration == runtimeGeneration ||
            refreshingRuntimeGeneration == runtimeGeneration
        ) return false
        refreshingRuntimeGeneration = runtimeGeneration
        return true
    }

    fun completeRefresh(runtimeGeneration: Long) {
        if (refreshingRuntimeGeneration != runtimeGeneration) return
        refreshingRuntimeGeneration = 0L
        if (foreground) refreshedRuntimeGeneration = runtimeGeneration
    }

    fun failRefresh(runtimeGeneration: Long) {
        if (refreshingRuntimeGeneration == runtimeGeneration) refreshingRuntimeGeneration = 0L
    }

    fun markBackground() {
        foreground = false
        refreshingRuntimeGeneration = 0L
    }

    fun reset() {
        foreground = false
        refreshedRuntimeGeneration = 0L
        refreshingRuntimeGeneration = 0L
    }
}

internal data class NativeRuntimeRebuildTicket(
    val token: Long,
    val delayMillis: Long,
)

internal data class NativeRuntimeHealthTicket(
    val token: Long,
    val runtimeGeneration: Long,
    val delayMillis: Long,
)

internal class NativeRuntimeRecoveryState(
    rebuildBackoffMillis: LongArray,
    private val runtimeStableWindowMillis: Long = 30_000L,
) {
    private val backoffMillis = rebuildBackoffMillis.copyOf()
    private var rebuildToken = 0L
    private var scheduledRebuildToken: Long? = null
    private var rebuildBackoffIndex = 0
    private var healthToken = 0L
    private var scheduledHealthToken: Long? = null

    var runtimeGeneration = 0L
        private set

    init {
        require(backoffMillis.isNotEmpty()) { "runtime rebuild backoff is required" }
        require(backoffMillis.all { it >= 0L }) { "runtime rebuild backoff must not be negative" }
        require(runtimeStableWindowMillis >= 0L) { "runtime stable window must not be negative" }
    }

    fun beginRuntimeAttempt(): Long {
        check(runtimeGeneration < Long.MAX_VALUE) { "native runtime generation is exhausted" }
        invalidateScheduledHealth()
        runtimeGeneration += 1L
        return runtimeGeneration
    }

    fun scheduleRebuild(): NativeRuntimeRebuildTicket? {
        invalidateScheduledHealth()
        if (scheduledRebuildToken != null) return null
        val token = advanceRebuildToken()
        scheduledRebuildToken = token
        val delay = backoffMillis[rebuildBackoffIndex.coerceAtMost(backoffMillis.lastIndex)]
        rebuildBackoffIndex = (rebuildBackoffIndex + 1).coerceAtMost(backoffMillis.lastIndex)
        return NativeRuntimeRebuildTicket(token = token, delayMillis = delay)
    }

    fun consumeRebuild(token: Long): Boolean {
        if (scheduledRebuildToken != token) return false
        scheduledRebuildToken = null
        return true
    }

    fun scheduleRuntimeHealth(runtimeGeneration: Long): NativeRuntimeHealthTicket {
        check(runtimeGeneration == this.runtimeGeneration) { "runtime generation is stale" }
        invalidateScheduledRebuild()
        val token = advanceHealthToken()
        scheduledHealthToken = token
        return NativeRuntimeHealthTicket(
            token = token,
            runtimeGeneration = runtimeGeneration,
            delayMillis = runtimeStableWindowMillis,
        )
    }

    fun consumeRuntimeHealth(ticket: NativeRuntimeHealthTicket): Boolean {
        if (
            scheduledHealthToken != ticket.token ||
            runtimeGeneration != ticket.runtimeGeneration
        ) return false
        scheduledHealthToken = null
        rebuildBackoffIndex = 0
        return true
    }

    fun cancelRebuilds() {
        invalidateScheduledRebuild()
        invalidateScheduledHealth()
        rebuildBackoffIndex = 0
    }

    fun resetForTests() {
        cancelRebuilds()
        runtimeGeneration = 0L
    }

    private fun invalidateScheduledRebuild() {
        advanceRebuildToken()
        scheduledRebuildToken = null
    }

    private fun invalidateScheduledHealth() {
        advanceHealthToken()
        scheduledHealthToken = null
    }

    private fun advanceRebuildToken(): Long {
        check(rebuildToken < Long.MAX_VALUE) { "native runtime rebuild token is exhausted" }
        rebuildToken += 1L
        return rebuildToken
    }

    private fun advanceHealthToken(): Long {
        check(healthToken < Long.MAX_VALUE) { "native runtime health token is exhausted" }
        healthToken += 1L
        return healthToken
    }
}

internal object NativeConnectionRuntimeOwner {
    private const val BRIDGE_TOKEN_BYTES = 32
    private const val RUNTIME_STABLE_WINDOW_MILLIS = 30_000L
    private val RUNTIME_REBUILD_BACKOFF_MILLIS = longArrayOf(1_000L, 2_000L, 4_000L, 8_000L, 15_000L)

    internal data class BridgeEndpoint(val port: Int, val token: String)

    internal class DisconnectAllRequest internal constructor(
        internal val runtimeGeneration: Long,
        private val networkMonitor: NativeNetworkMonitor?,
        private val engine: AndroidGoClientEngine?,
    ) {
        fun closeDetachedRuntime() {
            runCatching { networkMonitor?.close() }
            runCatching { engine?.close() }
        }
    }

    private var bridgeEndpoint: BridgeEndpoint? = null
    private var goEngine: AndroidGoClientEngine? = null
    private val runtimeHandler = Handler(Looper.getMainLooper())
    private var runtimeContext: Context? = null
    private val runtimeRecoveryState = NativeRuntimeRecoveryState(RUNTIME_REBUILD_BACKOFF_MILLIS)
    private val rendererDemand = NativeRendererDemandState()
    private val disconnectAllEvents = NativeDisconnectAllEventState()
    private val processNetworkMonitor = NativeProcessNetworkMonitorSlot<NativeNetworkMonitor>()
    private val networkObservers = linkedSetOf<(NativeNetworkSnapshot) -> Unit>()
    private var networkSignature: NativeNetworkMonitor.Signature? = null
    private var latestNetworkSnapshot = NativeNetworkSnapshot(
        epoch = 0L,
        connected = true,
        reason = "path_changed",
    )
    private val foregroundRefreshState = NativeForegroundRefreshState()
    private var hostRevision = 0L
    private var networkConnected = true
    private var networkReason = "path_changed"

    @Synchronized
    fun isStarted(): Boolean = bridgeEndpoint != null && goEngine != null && processNetworkMonitor.current() != null

    @Synchronized
    fun ensureStarted(context: Context): BridgeEndpoint {
        val appContext = context.applicationContext
        runtimeContext = appContext
        bridgeEndpoint?.let { endpoint ->
            if (goEngine != null) {
                ensureNetworkMonitorLocked(appContext)
                maintainForegroundServiceForDemandLocked(appContext)
                return endpoint
            }
        }

        val token = generateBridgeToken()
        val nextRuntimeGeneration = runtimeRecoveryState.beginRuntimeAttempt()
        var engine: AndroidGoClientEngine? = null
        try {
            val createdEngine = AndroidGoClientEngine(appContext) { failure ->
                reportPlatformPumpFailure(nextRuntimeGeneration, failure)
            }
            engine = createdEngine
            val port = GoClientNative.startBridge(createdEngine.handle, token)
            val endpoint = BridgeEndpoint(port, token)
            goEngine = createdEngine
            bridgeEndpoint = endpoint
            ensureNetworkMonitorLocked(appContext)
            replayNativeSupervisorState(
                maintainForegroundOwnership = { maintainForegroundServiceForDemandLocked(appContext) },
                signalHost = ::replayHostSignalLocked,
                replaceDemand = { replaceGoDemandLocked(rendererDemand.canonicalSnapshot()) },
            )
            scheduleRuntimeHealthLocked(nextRuntimeGeneration)
            AnyTTYDebugLog.event(AnyTTYDebugEvent.BRIDGE_STARTED)
            return endpoint
        } catch (failure: Exception) {
            processNetworkMonitor.close()
            runCatching { engine?.close() }
            goEngine = null
            bridgeEndpoint = null
            scheduleRuntimeRebuildLocked()
            throw failure
        }
    }

    @Synchronized
    fun endpoint(): BridgeEndpoint = bridgeEndpoint
        ?: throw IllegalStateException("native bridge server is not ready")

    @Synchronized
    fun resetLocalPairings(context: Context): BridgeEndpoint {
        val storageCoordinator = AndroidPlatformStorageCoordinator.process
        val resetGeneration = storageCoordinator.beginGeneration()
        try {
            storageCoordinator.withGeneration(resetGeneration) {
                AndroidEndpointRegistryStore(context.applicationContext).clear()
                AndroidClientAccessCredentialStore(context.applicationContext).clearAll()
                AndroidSSHCredentialStore().clearAll()
            }
        } finally {
            cancelRuntimeRebuildLocked()
            stopRuntimeLocked()
        }
        return ensureStarted(context.applicationContext)
    }

    @Synchronized
    fun attachRenderer(context: Context): NativeRendererDemandSnapshot {
        val appContext = context.applicationContext
        return try {
            handoffRendererDemand(
                rendererDemand = rendererDemand,
                replaceGoDemand = ::replaceGoDemandWithRecoveryLocked,
                ensureRuntimeForDemand = { ensureStarted(appContext) },
                maintainForegroundService = { maintainForegroundServiceForDemandLocked(appContext) },
            )
        } catch (_: Exception) {
            // Attachment identity must remain callable even while its process runtime
            // is rebuilding; later full-snapshot reconciliation replays the intent.
            AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED)
            rendererDemand.canonicalSnapshot()
        }
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
    ): NativeRendererDemandSnapshot {
        val appContext = context.applicationContext
        val next = reconcileRendererDemand(
            rendererDemand = rendererDemand,
            attachmentId = attachmentId,
            baseDemandRevision = baseDemandRevision,
            endpointIds = endpointIds,
            replaceGoDemand = ::replaceGoDemandWithRecoveryLocked,
            ensureRuntimeForDemand = { ensureStarted(appContext) },
            maintainForegroundService = { maintainForegroundServiceForDemandLocked(appContext) },
            stopForegroundService = { AnyTTYConnectionService.stop(appContext) },
        )
        return next
    }

    @Synchronized
    fun rendererDemandSnapshot(attachmentId: String): NativeRendererDemandSnapshot =
        rendererDemand.currentSnapshot(attachmentId)

    @Synchronized
    fun resumeRendererDemand(
        attachmentId: String,
        intentId: String,
        baseStopEpoch: Long,
    ): NativeRendererDemandResumeResult = rendererDemand.resumeDemand(attachmentId, intentId, baseStopEpoch)

    @Synchronized
    fun acknowledgeDisconnectAll(stopEpoch: Long) {
        disconnectAllEvents.acknowledge(stopEpoch)
    }

    fun observeDisconnectAll(observer: (NativeRendererDemandSnapshot) -> Unit): AutoCloseable {
        val canonicalSnapshot = synchronized(this) { rendererDemand.canonicalSnapshot() }
        return disconnectAllEvents.observe(canonicalSnapshot, observer)
    }

    @Synchronized
    fun hasActiveEndpoints(): Boolean = rendererDemand.hasDemand()

    fun networkSnapshot(): NativeNetworkSnapshot = synchronized(this) {
        latestNetworkSnapshot
    }

    fun observeNetwork(observer: (NativeNetworkSnapshot) -> Unit): AutoCloseable {
        synchronized(this) { networkObservers += observer }
        return AutoCloseable {
            synchronized(this@NativeConnectionRuntimeOwner) { networkObservers -= observer }
        }
    }

    @JvmStatic
    fun handleActivityForeground(context: Context) {
        synchronized(this) { foregroundRefreshState.markForeground() }
        refreshForegroundIfNeeded(context)
    }

    fun refreshForegroundNetwork(context: Context) {
        refreshForegroundIfNeeded(context)
    }

    private fun refreshForegroundIfNeeded(context: Context) {
        val shouldRun = synchronized(this) {
            bridgeEndpoint != null || goEngine != null || rendererDemand.hasDemand()
        }
        if (!shouldRun) return
        ensureStarted(context.applicationContext)
        val (monitor, sampledAtEpoch, runtimeGeneration) = synchronized(this) {
            val generation = runtimeRecoveryState.runtimeGeneration
            if (!foregroundRefreshState.beginRefresh(generation)) {
                AnyTTYDebugLog.connection(
                    "foreground_ack revision=$hostRevision runtime_generation=$generation",
                )
                return
            }
            Triple(processNetworkMonitor.current(), latestNetworkSnapshot.epoch, generation)
        }
        val sampled = try {
            monitor?.resampleNow()
        } catch (failure: Exception) {
            synchronized(this) { foregroundRefreshState.failRefresh(runtimeGeneration) }
            throw failure
        }
        synchronized(this) {
            if (
                sampled != null &&
                processNetworkMonitor.current() === monitor &&
                latestNetworkSnapshot.epoch == sampledAtEpoch
            ) {
                networkSignature = sampled
                networkConnected = sampled.connected
                if (!sampled.connected) networkReason = "offline"
            }
            foregroundRefreshState.completeRefresh(runtimeGeneration)
            AnyTTYDebugLog.connection(
                "foreground_refresh revision=$hostRevision connected=$networkConnected" +
                    " reason=$networkReason demand_count=${rendererDemand.canonicalSnapshot().endpointIds.size}" +
                    " runtime_generation=$runtimeGeneration",
            )
        }
    }

    @JvmStatic
    fun handleActivityBackground() {
        synchronized(this) { foregroundRefreshState.markBackground() }
    }

    @Synchronized
    fun endpointSupervisorSnapshot(): ByteArray {
        val engine = goEngine ?: return ByteArray(0)
        return GoClientNative.supervisorSnapshot(engine.handle)
    }

    @Synchronized
    fun requestEndpointRecovery(context: Context, endpointId: String) {
        val normalized = endpointId.trim()
        require(normalized.isNotEmpty()) { "endpointId is required" }
        require(rendererDemand.canonicalSnapshot().endpointIds.contains(normalized)) {
            "endpoint is not demanded"
        }
        ensureStarted(context.applicationContext)
        val engine = goEngine ?: throw IllegalStateException("Go client engine is unavailable")
        try {
            GoClientNative.repairSupervisorEndpoint(engine.handle, normalized.toByteArray(Charsets.UTF_8))
        } catch (failure: Exception) {
            recoverBrokenRuntimeLocked()
            throw failure
        }
    }

    /** Commits the user Stop synchronously, then hands heavy old-runtime disposal to the caller. */
    @Synchronized
    fun acceptDisconnectAll(): DisconnectAllRequest {
        val stopped = rendererDemand.clearDemand() ?: rendererDemand.canonicalSnapshot()
        disconnectAllEvents.publish(stopped)
        runCatching { replaceGoDemandLocked(stopped) }
            .onFailure { AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED) }
        cancelRuntimeRebuildLocked()
        runtimeContext = null
        bridgeEndpoint = null
        val detachedEngine = goEngine
        goEngine = null
        return DisconnectAllRequest(
            runtimeGeneration = runtimeRecoveryState.runtimeGeneration,
            networkMonitor = processNetworkMonitor.take(),
            engine = detachedEngine,
        )
    }

    @Synchronized
    fun canFinishDisconnectAll(request: DisconnectAllRequest): Boolean =
        disconnectAllCompletionOwnsForeground(
            acceptedRuntimeGeneration = request.runtimeGeneration,
            currentRuntimeGeneration = runtimeRecoveryState.runtimeGeneration,
            hasActiveEndpoints = rendererDemand.hasDemand(),
        ) && goEngine == null

    @Synchronized
    fun finishDisconnectAll(request: DisconnectAllRequest, finishForeground: () -> Unit): Boolean {
        return finishDisconnectAllServiceRequest(
            canStopService = { canFinishDisconnectAll(request) },
            stopService = finishForeground,
        )
    }

    @Synchronized
    internal fun stopForTests() {
        stopRuntimeLocked()
        rendererDemand.resetForTests()
        disconnectAllEvents.resetForTests()
        networkObservers.clear()
        networkSignature = null
        latestNetworkSnapshot = NativeNetworkSnapshot(0L, true, "path_changed")
        foregroundRefreshState.reset()
        runtimeRecoveryState.resetForTests()
        runtimeContext = null
        hostRevision = 0L
        networkConnected = true
        networkReason = "path_changed"
    }

    private fun stopRuntimeLocked() {
        processNetworkMonitor.close()
        bridgeEndpoint = null
        val engine = goEngine
        goEngine = null
        engine?.close()
    }

    private fun reportPlatformPumpFailure(generation: Long, failure: Exception) {
        runtimeHandler.post {
            synchronized(this) {
                if (generation != runtimeRecoveryState.runtimeGeneration || goEngine == null) return@synchronized
                AnyTTYDebugLog.connection(
                    "platform_pump failed runtime_generation=$generation type=${failure.javaClass.simpleName}",
                )
                stopRuntimeLocked()
                scheduleRuntimeRebuildLocked()
            }
        }
    }

    private fun scheduleRuntimeRebuildLocked() {
        if (runtimeContext == null) return
        val ticket = runtimeRecoveryState.scheduleRebuild() ?: return
        runtimeHandler.postDelayed({
            synchronized(this) {
                if (!runtimeRecoveryState.consumeRebuild(ticket.token)) return@synchronized
                val currentContext = runtimeContext ?: return@synchronized
                try {
                    // Keep the ticket check and actual attempt under the owner lock. A user
                    // stop either invalidates the ticket first or stops this attempt afterward.
                    ensureStarted(currentContext)
                } catch (_: Exception) {
                    scheduleRuntimeRebuildLocked()
                }
            }
        }, ticket.delayMillis)
    }

    private fun scheduleRuntimeHealthLocked(runtimeGeneration: Long) {
        val ticket = runtimeRecoveryState.scheduleRuntimeHealth(runtimeGeneration)
        runtimeHandler.postDelayed({
            synchronized(this) {
                if (goEngine == null) return@synchronized
                runtimeRecoveryState.consumeRuntimeHealth(ticket)
            }
        }, ticket.delayMillis)
    }

    private fun cancelRuntimeRebuildLocked() {
        runtimeRecoveryState.cancelRebuilds()
    }

    private fun ensureNetworkMonitorLocked(context: Context) {
        processNetworkMonitor.ensure(
            create = {
                lateinit var created: NativeNetworkMonitor
                created = NativeNetworkMonitor(
                    context = context,
                    initialSignature = networkSignature,
                    initialEpoch = latestNetworkSnapshot.epoch,
                    onStableNetworkChanged = { change ->
                        publishStableNetworkChange(created, change)
                    },
                )
                if (networkSignature == null) {
                    val initial = created.stableSignature()
                    networkSignature = initial
                    networkConnected = initial.connected
                    networkReason = if (initial.connected) "path_changed" else "offline"
                    latestNetworkSnapshot = NativeNetworkSnapshot(
                        epoch = latestNetworkSnapshot.epoch,
                        connected = networkConnected,
                        reason = networkReason,
                    )
                    incrementHostRevisionLocked()
                }
                created
            },
            start = NativeNetworkMonitor::start,
        )
    }

    private fun publishStableNetworkChange(
        source: NativeNetworkMonitor,
        change: NativeStableNetworkChange,
    ) {
        val projection: NativeNetworkSnapshot
        val observers: List<(NativeNetworkSnapshot) -> Unit>
        synchronized(this) {
            if (
                processNetworkMonitor.current() !== source ||
                change.epoch <= latestNetworkSnapshot.epoch
            ) return
            networkSignature = change.current
            networkConnected = change.current.connected
            networkReason = change.reason
            latestNetworkSnapshot = NativeNetworkSnapshot(
                epoch = change.epoch,
                connected = networkConnected,
                reason = networkReason,
            )
            incrementHostRevisionLocked()
            AnyTTYDebugLog.connection(
                "host_signal revision=$hostRevision foreground=false" +
                    " connected=$networkConnected reason=$networkReason",
            )
            try {
                signalGoSupervisorLocked(foreground = false)
            } catch (_: Exception) {
                AnyTTYDebugLog.event(AnyTTYDebugEvent.GENERATION_CHANGE_FAILED)
                recoverBrokenRuntimeLocked()
            }
            AnyTTYDebugLog.event(AnyTTYDebugEvent.NETWORK_CHANGED, change.epoch)
            projection = latestNetworkSnapshot
            observers = networkObservers.toList()
        }
        observers.forEach { observer -> runCatching { observer(projection) } }
    }

    private fun replaceGoDemandLocked(snapshot: NativeRendererDemandSnapshot) {
        val engine = goEngine ?: return
        GoClientNative.replaceSupervisorDemand(
            engine.handle,
            nativeSupervisorDemandSnapshot(snapshot).toByteArray(),
        )
    }

    private fun replaceGoDemandWithRecoveryLocked(snapshot: NativeRendererDemandSnapshot) {
        submitNativeDemandOrRecover(
            submit = { replaceGoDemandLocked(snapshot) },
            recoverRuntime = ::recoverBrokenRuntimeLocked,
        )
    }

    private fun maintainForegroundServiceForDemandLocked(context: Context) {
        if (!rendererDemand.hasDemand()) return
        val appContext = context.applicationContext
        runtimeContext = appContext
        maintainNativeDemandForegroundOwnership(
            maintainForegroundService = { AnyTTYConnectionService.start(appContext) },
            recoverRuntime = ::recoverBrokenRuntimeLocked,
        )
    }

    private fun recoverBrokenRuntimeLocked() {
        stopRuntimeLocked()
        scheduleRuntimeRebuildLocked()
    }

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
