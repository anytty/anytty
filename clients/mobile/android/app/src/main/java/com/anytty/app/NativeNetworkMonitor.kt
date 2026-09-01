package com.anytty.app

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.os.Build
import android.os.Handler
import android.os.Looper

internal class NativeNetworkMonitor(
    context: Context,
    initialSignature: Signature? = null,
    initialEpoch: Long = 0L,
    private val onStableNetworkChanged: (NativeStableNetworkChange) -> Unit,
) : AutoCloseable {
    companion object {
        internal const val PATH_SETTLE_DELAY_MILLIS = 200L
        internal const val OFFLINE_GRACE_DELAY_MILLIS = 750L
        private val UNDERLYING_RESAMPLE_DELAYS_MILLIS = longArrayOf(400L, 1_000L, 2_000L)
    }

    internal data class Signature(
        val networkHandle: Long,
        val internet: Boolean,
        val validated: Boolean,
        val addresses: List<String>,
        val dnsServers: List<String>,
        val routes: List<String>,
        val transports: List<Int> = emptyList(),
        val underlyingNetworks: List<UnderlyingNetworkIdentity> = emptyList(),
    ) {
        // VALIDATED is an internet-quality hint, not an end-to-end verdict. A default
        // route may still carry a healthy LAN P2P session or reach a captive portal.
        val connected: Boolean get() = networkHandle != 0L
    }

    internal data class UnderlyingNetworkIdentity(
        val networkHandle: Long,
        val transports: List<Int>,
    )

    private data class NetworkSample(
        val signature: Signature,
        val ambiguousUnderlying: List<NativeUnderlyingNetworkCandidate>?,
    )

    private val connectivity = context.applicationContext
        .getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    private val handler = Handler(Looper.getMainLooper())
    private val sampleLock = Any()
    private val underlyingResampleState = NativeBoundedNetworkResampleState(
        UNDERLYING_RESAMPLE_DELAYS_MILLIS,
    )
    private val stableState = NativeStableNetworkState(
        initialSignature = initialSignature ?: readCurrentNetworkSample().signature,
        initialEpoch = initialEpoch,
    )
    private var started = false
    private var bearerCallbackRegistered = false
    @Volatile private var closed = false
    private var settleScheduled = false
    private val settle = Runnable {
        settleScheduled = false
        publishStableNetwork()
    }
    private var underlyingResampleScheduled = false
    private val underlyingResample = Runnable {
        underlyingResampleScheduled = false
        publishStableNetwork()
    }
    private val callback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) = scheduleCheck()
        override fun onLost(network: Network) = scheduleCheck()
        override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) = scheduleCheck()
        override fun onLinkPropertiesChanged(network: Network, linkProperties: LinkProperties) = scheduleCheck()
    }
    private val bearerCallback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) = scheduleCheck()
        override fun onLost(network: Network) = scheduleCheck()
        override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) = scheduleCheck()
        override fun onLinkPropertiesChanged(network: Network, linkProperties: LinkProperties) = scheduleCheck()
    }

    fun start() {
        synchronized(this) {
            check(!closed) { "native network monitor is closed" }
            if (started) return
            started = true
            connectivity.registerDefaultNetworkCallback(callback)
            bearerCallbackRegistered = runCatching {
                connectivity.registerNetworkCallback(
                    NetworkRequest.Builder()
                        .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                        .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
                        .build(),
                    bearerCallback,
                )
            }.isSuccess
        }
        resampleNow()
    }

    fun stableSignature(): Signature = stableState.currentSignature()

    fun resampleNow(): Signature {
        val (current, change, resampleDelay) = synchronized(sampleLock) {
            if (closed) return stableState.currentSignature()
            val preferredUnderlyingHandle = stableState.currentSignature()
                .underlyingNetworks
                .singleOrNull()
                ?.networkHandle
            val rawSample = readCurrentNetworkSample(preferredUnderlyingHandle)
            val decision = underlyingResampleState.observe(rawSample.ambiguousUnderlying)
            val sampled = if (decision.retainAmbiguousSelection) {
                rawSample.signature
            } else {
                rawSample.signature.copy(underlyingNetworks = emptyList())
            }
            Triple(sampled, stableState.observe(sampled), decision.retryDelayMillis)
        }
        scheduleUnderlyingResample(resampleDelay)
        if (change != null) publishStableNetwork(change)
        return current
    }

    private fun scheduleCheck() {
        handler.post {
            if (closed) return@post
            if (settleScheduled) handler.removeCallbacks(settle)
            settleScheduled = true
            val delay = if (runCatching { connectivity.activeNetwork }.getOrNull() == null) {
                OFFLINE_GRACE_DELAY_MILLIS
            } else {
                PATH_SETTLE_DELAY_MILLIS
            }
            handler.postDelayed(settle, delay)
        }
    }

    private fun publishStableNetwork() {
        resampleNow()
    }

    private fun scheduleUnderlyingResample(delayMillis: Long?) {
        handler.post {
            if (underlyingResampleScheduled) handler.removeCallbacks(underlyingResample)
            underlyingResampleScheduled = false
            if (closed || delayMillis == null) return@post
            underlyingResampleScheduled = true
            handler.postDelayed(underlyingResample, delayMillis)
        }
    }

    private fun publishStableNetwork(change: NativeStableNetworkChange) {
        val previous = change.previous
        val current = change.current
        AnyTTYDebugLog.connection(buildString {
            append("network_stable epoch=${change.epoch} reason=${change.reason}")
            append(" connected=${current.connected} internet=${current.internet} validated=${current.validated}")
            append(" handle_changed=${previous.networkHandle != current.networkHandle}")
            append(" transport_changed=${previous.transports != current.transports}")
            append(" underlying_changed=${previous.underlyingNetworks != current.underlyingNetworks}")
            append(" address_changed=${previous.addresses != current.addresses}")
            append(" dns_changed=${previous.dnsServers != current.dnsServers}")
            append(" routes_changed=${previous.routes != current.routes}")
            append(" address_count=${current.addresses.size}")
            append(" dns_count=${current.dnsServers.size}")
            append(" route_count=${current.routes.size}")
            append(" transport_count=${current.transports.size}")
            append(" underlying_count=${current.underlyingNetworks.size}")
        })
        onStableNetworkChanged(change)
    }

    private fun readCurrentNetworkSample(preferredUnderlyingHandle: Long? = null): NetworkSample {
        val network = runCatching { connectivity.activeNetwork }.getOrNull()
            ?: return NetworkSample(disconnectedSignature(), ambiguousUnderlying = null)
        val capabilities = runCatching { connectivity.getNetworkCapabilities(network) }.getOrNull()
        val linkProperties = runCatching { connectivity.getLinkProperties(network) }.getOrNull()
        val transports = readTransports(capabilities)
        val internet = capabilities.hasCapabilitySafely(NetworkCapabilities.NET_CAPABILITY_INTERNET)
        val validated = capabilities.hasCapabilitySafely(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
        val addresses = linkProperties?.linkAddresses
            ?.map { it.address.hostAddress.orEmpty() }
            ?.filter { it.isNotBlank() }
            ?.sorted()
            .orEmpty()
        val dnsServers = linkProperties?.dnsServers
            ?.map { it.hostAddress.orEmpty() }
            ?.filter { it.isNotBlank() }
            ?.sorted()
            .orEmpty()
        val routes = linkProperties?.routes
            ?.map { route -> route.toString() }
            ?.sorted()
            .orEmpty()
        val underlying = readUnderlyingNetworks(network, transports, preferredUnderlyingHandle)
        return NetworkSample(signature = Signature(
            networkHandle = network.networkHandle,
            internet = internet,
            validated = validated,
            addresses = addresses,
            dnsServers = dnsServers,
            routes = routes,
            transports = transports,
            underlyingNetworks = underlying.identities,
        ), ambiguousUnderlying = underlying.ambiguousCandidates)
    }

    private fun disconnectedSignature() = Signature(
        networkHandle = 0L,
        internet = false,
        validated = false,
        addresses = emptyList(),
        dnsServers = emptyList(),
        routes = emptyList(),
    )

    @Suppress("DEPRECATION")
    private fun readUnderlyingNetworks(
        defaultNetwork: Network,
        defaultTransports: List<Int>,
        preferredHandle: Long?,
    ): NativeUnderlyingNetworkSelection {
        if (NetworkCapabilities.TRANSPORT_VPN !in defaultTransports) {
            return NativeUnderlyingNetworkSelection(emptyList(), ambiguousCandidates = null)
        }
        val physicalTransports = defaultTransports.filterNot { it == NetworkCapabilities.TRANSPORT_VPN }.toSet()
        val candidates = runCatching { connectivity.allNetworks.toList() }.getOrDefault(emptyList())
        val matching = candidates.mapNotNull { candidate ->
            val handle = runCatching { candidate.networkHandle }.getOrNull() ?: return@mapNotNull null
            if (handle == defaultNetwork.networkHandle || handle == 0L) return@mapNotNull null
            val capabilities = runCatching { connectivity.getNetworkCapabilities(candidate) }.getOrNull()
                ?: return@mapNotNull null
            if (
                !capabilities.hasCapabilitySafely(NetworkCapabilities.NET_CAPABILITY_INTERNET) ||
                !capabilities.hasCapabilitySafely(NetworkCapabilities.NET_CAPABILITY_NOT_VPN) ||
                capabilities.hasTransportSafely(NetworkCapabilities.TRANSPORT_VPN)
            ) return@mapNotNull null
            val transports = readTransports(capabilities)
            if (physicalTransports.isNotEmpty() && transports.none(physicalTransports::contains)) {
                return@mapNotNull null
            }
            NativeUnderlyingNetworkCandidate(
                identity = UnderlyingNetworkIdentity(handle, transports),
                validated = capabilities.hasCapabilitySafely(NetworkCapabilities.NET_CAPABILITY_VALIDATED),
            )
        }
            .distinctBy { it.identity.networkHandle }
            .sortedBy { it.identity.networkHandle }
        return selectUnderlyingNetwork(matching, preferredHandle)
    }

    private fun readTransports(capabilities: NetworkCapabilities?): List<Int> =
        supportedTransportTypes().filter { capabilities.hasTransportSafely(it) }

    private fun supportedTransportTypes(): List<Int> = buildList {
        add(NetworkCapabilities.TRANSPORT_CELLULAR)
        add(NetworkCapabilities.TRANSPORT_WIFI)
        add(NetworkCapabilities.TRANSPORT_BLUETOOTH)
        add(NetworkCapabilities.TRANSPORT_ETHERNET)
        add(NetworkCapabilities.TRANSPORT_VPN)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            add(NetworkCapabilities.TRANSPORT_WIFI_AWARE)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O_MR1) {
            add(NetworkCapabilities.TRANSPORT_LOWPAN)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            add(NetworkCapabilities.TRANSPORT_USB)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
            add(NetworkCapabilities.TRANSPORT_THREAD)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.VANILLA_ICE_CREAM) {
            add(NetworkCapabilities.TRANSPORT_SATELLITE)
        }
    }

    override fun close() {
        val unregisterBearer: Boolean
        synchronized(this) {
            if (closed) return
            closed = true
            started = false
            unregisterBearer = bearerCallbackRegistered
            bearerCallbackRegistered = false
        }
        handler.removeCallbacks(settle)
        handler.removeCallbacks(underlyingResample)
        settleScheduled = false
        underlyingResampleScheduled = false
        runCatching { connectivity.unregisterNetworkCallback(callback) }
        if (unregisterBearer) {
            runCatching { connectivity.unregisterNetworkCallback(bearerCallback) }
        }
    }
}

internal data class NativeUnderlyingNetworkCandidate(
    val identity: NativeNetworkMonitor.UnderlyingNetworkIdentity,
    val validated: Boolean,
)

internal data class NativeUnderlyingNetworkSelection(
    val identities: List<NativeNetworkMonitor.UnderlyingNetworkIdentity>,
    val ambiguousCandidates: List<NativeUnderlyingNetworkCandidate>?,
)

/**
 * compileSdk 36 exposes neither NetworkCapabilities#getUnderlyingNetworks nor an equivalent
 * ConnectivityManager API to ordinary apps. Use public bearer candidates without folding every
 * physical network into the signature: a unique validated bearer wins, while an ambiguous old
 * preference is only retained during the bounded settle probes below.
 */
internal fun selectUnderlyingNetwork(
    candidates: List<NativeUnderlyingNetworkCandidate>,
    preferredHandle: Long?,
): NativeUnderlyingNetworkSelection {
    val ordered = candidates
        .distinctBy { it.identity.networkHandle }
        .sortedBy { it.identity.networkHandle }
    val validated = ordered.filter(NativeUnderlyingNetworkCandidate::validated)
    val preferred = ordered.firstOrNull { it.identity.networkHandle == preferredHandle }
    val selected = validated.singleOrNull()
        ?: ordered.singleOrNull()
        ?: preferred?.takeIf { it.validated || validated.isEmpty() }
    val ambiguous = ordered.takeIf { it.size > 1 && validated.size != 1 }
    return NativeUnderlyingNetworkSelection(
        identities = listOfNotNull(selected?.identity),
        ambiguousCandidates = ambiguous,
    )
}

internal fun selectUnderlyingNetworks(
    candidates: List<NativeUnderlyingNetworkCandidate>,
    preferredHandle: Long?,
): List<NativeNetworkMonitor.UnderlyingNetworkIdentity> =
    selectUnderlyingNetwork(candidates, preferredHandle).identities

internal data class NativeBoundedNetworkResampleDecision(
    val retainAmbiguousSelection: Boolean,
    val retryDelayMillis: Long?,
)

internal class NativeBoundedNetworkResampleState(resampleDelaysMillis: LongArray) {
    private val delaysMillis = resampleDelaysMillis.copyOf()
    private var fingerprint: List<NativeUnderlyingNetworkCandidate>? = null
    private var attempt = 0

    init {
        require(delaysMillis.all { it >= 0L }) { "network resample delay must not be negative" }
    }

    fun observe(ambiguousCandidates: List<NativeUnderlyingNetworkCandidate>?): NativeBoundedNetworkResampleDecision {
        if (ambiguousCandidates.isNullOrEmpty()) {
            fingerprint = null
            attempt = 0
            return NativeBoundedNetworkResampleDecision(
                retainAmbiguousSelection = true,
                retryDelayMillis = null,
            )
        }
        if (fingerprint != ambiguousCandidates) {
            fingerprint = ambiguousCandidates.toList()
            attempt = 0
        }
        if (attempt >= delaysMillis.size) {
            return NativeBoundedNetworkResampleDecision(
                retainAmbiguousSelection = false,
                retryDelayMillis = null,
            )
        }
        val delay = delaysMillis[attempt]
        attempt += 1
        return NativeBoundedNetworkResampleDecision(
            retainAmbiguousSelection = true,
            retryDelayMillis = delay,
        )
    }
}

private fun NetworkCapabilities?.hasCapabilitySafely(capability: Int): Boolean =
    this != null && runCatching { hasCapability(capability) }.getOrDefault(false)

private fun NetworkCapabilities?.hasTransportSafely(transport: Int): Boolean =
    this != null && runCatching { hasTransport(transport) }.getOrDefault(false)

internal data class NativeStableNetworkChange(
    val epoch: Long,
    val previous: NativeNetworkMonitor.Signature,
    val current: NativeNetworkMonitor.Signature,
    val reason: String,
)

internal class NativeStableNetworkState(
    initialSignature: NativeNetworkMonitor.Signature,
    initialEpoch: Long = 0L,
) {
    private var signature = initialSignature
    private var epoch = initialEpoch

    @Synchronized
    fun observe(current: NativeNetworkMonitor.Signature): NativeStableNetworkChange? {
        if (current == signature) return null
        val previous = signature
        signature = current
        if (!requiresSessionRecovery(previous, current)) return null
        check(epoch < Long.MAX_VALUE) { "native network epoch is exhausted" }
        epoch += 1
        return NativeStableNetworkChange(
            epoch = epoch,
            previous = previous,
            current = current,
            reason = networkChangeReason(previous, current),
        )
    }

    @Synchronized
    fun currentSignature(): NativeNetworkMonitor.Signature = signature

    @Synchronized
    fun currentEpoch(): Long = epoch
}

internal class NativeProcessNetworkMonitorSlot<T : AutoCloseable> : AutoCloseable {
    private var monitor: T? = null

    fun ensure(create: () -> T, start: (T) -> Unit = {}): T {
        monitor?.let { return it }
        val created = create()
        monitor = created
        try {
            start(created)
            return created
        } catch (failure: Exception) {
            if (monitor === created) monitor = null
            runCatching { created.close() }
            throw failure
        }
    }

    fun current(): T? = monitor

    fun take(): T? {
        val current = monitor
        monitor = null
        return current
    }

    override fun close() {
        take()?.close()
    }
}

internal data class NativeNetworkSnapshot(
    val epoch: Long,
    val connected: Boolean,
    val reason: String,
)

internal fun requiresSessionRecovery(
    previous: NativeNetworkMonitor.Signature,
    current: NativeNetworkMonitor.Signature,
): Boolean = previous.networkHandle != current.networkHandle ||
    previous.connected != current.connected ||
    previous.internet != current.internet ||
    previous.transports != current.transports ||
    previous.underlyingNetworks != current.underlyingNetworks ||
    previous.addresses != current.addresses ||
    previous.dnsServers != current.dnsServers ||
    previous.routes != current.routes

internal fun networkChangeReason(
    previous: NativeNetworkMonitor.Signature,
    current: NativeNetworkMonitor.Signature,
): String = when {
    !current.connected -> "offline"
    !previous.connected -> "available"
    previous.networkHandle != current.networkHandle && effectivePathChanged(previous, current) -> "network_replaced"
    previous.transports != current.transports -> "network_replaced"
    previous.underlyingNetworks != current.underlyingNetworks -> "network_replaced"
    else -> "path_changed"
}

private fun effectivePathChanged(
    previous: NativeNetworkMonitor.Signature,
    current: NativeNetworkMonitor.Signature,
): Boolean = previous.internet != current.internet ||
    previous.transports != current.transports ||
    previous.underlyingNetworks != current.underlyingNetworks ||
    previous.addresses != current.addresses ||
    previous.dnsServers != current.dnsServers ||
    previous.routes != current.routes
