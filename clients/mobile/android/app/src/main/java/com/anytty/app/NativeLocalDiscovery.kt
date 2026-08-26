package com.anytty.app

import android.annotation.SuppressLint
import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.net.wifi.WifiManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.os.ext.SdkExtensions
import anytty.client.binding.v1.ClientBinding
import java.security.MessageDigest
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.Executor

internal data class NativeLocalDiscoveryCandidate(
    val discoveryKey: String,
    val address: String,
    val port: Int,
    val protocolVersion: Int,
    val expiresAtUnixNano: Long,
	val networkHandle: Long,
)

internal fun nativeLocalDiscoveryResult(
    candidates: List<NativeLocalDiscoveryCandidate>,
): ClientBinding.LocalDiscoveryLookupResult = ClientBinding.LocalDiscoveryLookupResult.newBuilder().also { result ->
    candidates.forEach { candidate ->
        result.addCandidates(ClientBinding.LocalDiscoveryCandidate.newBuilder()
            .setAddress(candidate.address)
            .setPort(candidate.port)
            .setProtocolVersion(candidate.protocolVersion)
            .setExpiresAtUnixNano(candidate.expiresAtUnixNano)
            .setNetworkHandle(candidate.networkHandle))
    }
}.build()

internal object NativeLocalDiscoveryCache {
	private const val CANDIDATE_LEASE_MILLIS = 60_000L
    private val candidates = ConcurrentHashMap<String, List<NativeLocalDiscoveryCandidate>>()

    fun update(serviceName: String, info: NsdServiceInfo) {
        val attributes = info.attributes
        val version = attributes["v"]?.decodeToString()?.toIntOrNull()
        val protocol = attributes["p"]?.decodeToString()?.toIntOrNull()
        val discoveryKey = attributes["k"]?.decodeToString()?.trim().orEmpty()
        if (version != 1 || protocol == null || protocol <= 0 || !discoveryKey.matches(Regex("[0-9a-f]{64}")) || info.port !in 1..65535) {
            candidates.remove(serviceName)
            return
        }
        val addresses = if (android.os.Build.VERSION.SDK_INT >= 34) {
            info.hostAddresses.mapNotNull { it.hostAddress }
        } else {
            @Suppress("DEPRECATION")
            listOfNotNull(info.host?.hostAddress)
        }.distinct()
		val expiresAt = System.currentTimeMillis() * 1_000_000L
        candidates[serviceName] = addresses.map { address ->
            NativeLocalDiscoveryCandidate(discoveryKey, address, info.port, protocol, expiresAt, 0L)
        }
    }

    fun update(serviceName: String, values: List<NativeLocalDiscoveryCandidate>) {
        if (values.isEmpty()) candidates.remove(serviceName) else candidates[serviceName] = values
    }

    fun remove(serviceName: String) { candidates.remove(serviceName) }
    fun clear() { candidates.clear() }

	fun snapshot(deviceId: String, fingerprint: String): List<NativeLocalDiscoveryCandidate> {
		val expectedKey = nativeLocalDiscoveryKey(deviceId, fingerprint)
		val expiresAt = (System.currentTimeMillis() + CANDIDATE_LEASE_MILLIS) * 1_000_000L
		return candidates.values.flatten().filter {
			it.discoveryKey == expectedKey
		}.map {
			it.copy(expiresAtUnixNano = expiresAt)
		}.distinctBy { "${it.address}:${it.port}:${it.networkHandle}" }
	}
}

internal fun nativeLocalDiscoveryKey(deviceId: String, fingerprint: String): String =
    MessageDigest.getInstance("SHA-256")
        .digest("anytty-lan-discovery-v1\u0000${deviceId.trim()}\u0000${fingerprint.trim()}".toByteArray(Charsets.UTF_8))
        .joinToString("") { "%02x".format(it.toInt() and 0xff) }

internal fun nativeLocalDiscoveryNeedsMulticastLock(apiLevel: Int, tiramisuExtension: Int): Boolean =
    !nativeLocalDiscoverySupportsServiceInfoCallback(apiLevel, tiramisuExtension)

internal fun nativeLocalDiscoverySupportsServiceInfoCallback(apiLevel: Int, tiramisuExtension: Int): Boolean =
    apiLevel > Build.VERSION_CODES.TIRAMISU ||
        (apiLevel == Build.VERSION_CODES.TIRAMISU && tiramisuExtension >= 7)

internal class NativeLocalDiscovery(
	context: Context,
	private val onChanged: () -> Unit = {},
) : AutoCloseable {
    companion object { private const val SERVICE_TYPE = "_anytty._tcp." }

    private val manager = context.applicationContext.getSystemService(Context.NSD_SERVICE) as NsdManager
    private val connectivity = context.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    private val wifiManager = context.applicationContext.getSystemService(Context.WIFI_SERVICE) as? WifiManager
    private val mainHandler = Handler(Looper.getMainLooper())
    private val callbackExecutor = Executor { command -> mainHandler.post(command) }
    private var listener: NsdManager.DiscoveryListener? = null
    private var multicastLock: WifiManager.MulticastLock? = null
    private val known = mutableSetOf<String>()
    private val pending = linkedMapOf<String, NsdServiceInfo>()
    private val serviceInfoSubscriptions = mutableMapOf<String, NativeLocalDiscoverySubscription>()
    private val foundAtElapsedMillis = mutableMapOf<String, Long>()
    private var resolvingKey: String? = null

    @Synchronized
    fun restart(connected: Boolean) {
        stopLocked()
        NativeLocalDiscoveryCache.clear()
		onChanged()
        if (!connected) return
		acquireMulticastLockIfNeeded()
		val next = object : NsdManager.DiscoveryListener {
			override fun onDiscoveryStarted(serviceType: String) {
				AnyTTYDebugLog.connection("local_discovery started")
			}
			override fun onDiscoveryStopped(serviceType: String) {
				AnyTTYDebugLog.connection("local_discovery stopped")
			}
			override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
				AnyTTYDebugLog.connection("local_discovery start_failed code=$errorCode")
				stopIfCurrent(this)
			}
			override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) {
				AnyTTYDebugLog.connection("local_discovery stop_failed code=$errorCode")
				stopIfCurrent(this)
			}
            override fun onServiceLost(serviceInfo: NsdServiceInfo) {
                val key = serviceKey(serviceInfo)
                synchronized(this@NativeLocalDiscovery) {
					known.remove(key)
					pending.remove(key)
					foundAtElapsedMillis.remove(key)
					unregisterServiceInfoCallbackLocked(key)
				}
				NativeLocalDiscoveryCache.remove(key)
					onChanged()
				AnyTTYDebugLog.connection("local_discovery lost")
            }
			override fun onServiceFound(serviceInfo: NsdServiceInfo) {
				if (!serviceInfo.serviceType.equals(SERVICE_TYPE, ignoreCase = true)) return
				AnyTTYDebugLog.connection("local_discovery found")
                synchronized(this@NativeLocalDiscovery) {
                    if (listener !== this) return
                    val key = serviceKey(serviceInfo)
                    known.add(key)
                    foundAtElapsedMillis.putIfAbsent(key, SystemClock.elapsedRealtime())
                    if (supportsServiceInfoCallback()) {
                        registerServiceInfoSubscriptionLocked(this, key, serviceInfo)
                    } else {
                        pending[key] = serviceInfo
                        resolveNextLocked(this)
                    }
                }
            }
        }
        listener = next
        runCatching {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                val request = NetworkRequest.Builder()
                    .addTransportType(NetworkCapabilities.TRANSPORT_WIFI)
                    .addTransportType(NetworkCapabilities.TRANSPORT_ETHERNET)
                    .build()
                manager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, request, callbackExecutor, next)
            } else {
                @Suppress("DEPRECATION")
                manager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, next)
            }
        }
            .onFailure { stopIfCurrent(next) }
    }

    @Synchronized
    private fun stopIfCurrent(value: NsdManager.DiscoveryListener) {
        if (listener === value) stopLocked()
    }

    @Synchronized
    private fun resolveNextLocked(owner: NsdManager.DiscoveryListener) {
        if (listener !== owner || resolvingKey != null) return
        val entry = pending.entries.firstOrNull() ?: return
        val key = entry.key
        val serviceInfo = entry.value
        pending.remove(key)
        resolvingKey = key
		val resolver = object : NsdManager.ResolveListener {
			override fun onResolveFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                AnyTTYDebugLog.connection("local_discovery resolve_failed code=$errorCode")
                synchronized(this@NativeLocalDiscovery) {
                    if (resolvingKey == key) resolvingKey = null
                    resolveNextLocked(owner)
                }
            }
            override fun onServiceResolved(resolved: NsdServiceInfo) {
                synchronized(this@NativeLocalDiscovery) {
                    if (listener === owner && known.contains(key)) updateResolvedService(key, resolved)
                    if (resolvingKey == key) resolvingKey = null
                    resolveNextLocked(owner)
                }
            }
        }
        runCatching {
            @Suppress("DEPRECATION")
            manager.resolveService(serviceInfo, resolver)
        }.onFailure {
            resolvingKey = null
            resolveNextLocked(owner)
        }
    }

    private fun registerServiceInfoSubscriptionLocked(
        owner: NsdManager.DiscoveryListener,
        key: String,
        serviceInfo: NsdServiceInfo,
    ) {
        if (serviceInfoSubscriptions.containsKey(key) || !supportsServiceInfoCallback()) return
        val subscription = NativeLocalDiscoveryModernSubscription(
            manager = manager,
            onFailure = { errorCode ->
                AnyTTYDebugLog.connection("local_discovery callback_failed code=$errorCode")
                synchronized(this@NativeLocalDiscovery) {
                    serviceInfoSubscriptions.remove(key)
                    if (listener === owner && known.contains(key)) {
                        pending[key] = serviceInfo
                        resolveNextLocked(owner)
                    }
                }
            },
            onUnregistered = {
                synchronized(this@NativeLocalDiscovery) {
                    serviceInfoSubscriptions.remove(key)
                }
            },
            onLost = {
                synchronized(this@NativeLocalDiscovery) {
                    known.remove(key)
                    foundAtElapsedMillis.remove(key)
                    serviceInfoSubscriptions.remove(key)
                }
                NativeLocalDiscoveryCache.remove(key)
                onChanged()
            },
            onUpdated = { info ->
                synchronized(this@NativeLocalDiscovery) {
                    if (listener === owner && serviceInfoSubscriptions.containsKey(key) && known.contains(key)) {
                        updateResolvedService(key, info)
                    }
                }
            },
        )
        serviceInfoSubscriptions[key] = subscription
        runCatching { subscription.start(serviceInfo, callbackExecutor) }.onFailure {
            if (serviceInfoSubscriptions[key] === subscription) {
                serviceInfoSubscriptions.remove(key)
                pending[key] = serviceInfo
                resolveNextLocked(owner)
            }
        }
    }

    @SuppressLint("NewApi") // Guarded by supportsServiceInfoCallback(), including Android 13 extension 7.
    private fun updateResolvedService(name: String, info: NsdServiceInfo) {
        val attributes = info.attributes
        val version = attributes["v"]?.decodeToString()?.toIntOrNull()
        val protocol = attributes["p"]?.decodeToString()?.toIntOrNull()
		val discoveryKey = attributes["k"]?.decodeToString()?.trim().orEmpty()
		if (version != 1 || protocol == null || protocol <= 0 || !discoveryKey.matches(Regex("[0-9a-f]{64}")) || info.port !in 1..65535) {
			NativeLocalDiscoveryCache.remove(name)
			onChanged()
			return
		}
        val addresses = if (supportsServiceInfoCallback()) {
            info.hostAddresses
        } else {
            @Suppress("DEPRECATION")
            listOfNotNull(info.host)
        }.distinctBy { it.hostAddress }
        val source = if (Build.VERSION.SDK_INT >= 33) info.network else null
		val expiresAt = System.currentTimeMillis() * 1_000_000L
		val discovered = addresses.map { address ->
			NativeLocalDiscoveryCandidate(
                discoveryKey = discoveryKey,
                address = address.hostAddress.orEmpty(),
                port = info.port,
                protocolVersion = protocol,
                expiresAtUnixNano = expiresAt,
				networkHandle = (source ?: bestNetworkFor(address))?.networkHandle ?: 0L,
			)
		}.filter { it.address.isNotBlank() }
			NativeLocalDiscoveryCache.update(name, discovered)
			onChanged()
		val networkCount = discovered.map { it.networkHandle }.distinct().size
		val foundAt = foundAtElapsedMillis.remove(name)
		val resolvedIn = foundAt?.let { SystemClock.elapsedRealtime() - it } ?: -1L
		AnyTTYDebugLog.connection("local_discovery candidates=${discovered.size} network_count=$networkCount resolve_ms=$resolvedIn")
	}

    private fun bestNetworkFor(address: java.net.InetAddress): Network? = connectivity.allNetworks
        .mapNotNull { network ->
            val prefix = connectivity.getLinkProperties(network)?.routes
                ?.asSequence()
                ?.map { it.destination }
                ?.filter { it.contains(address) }
                ?.maxOfOrNull { it.prefixLength }
                ?: return@mapNotNull null
            network to prefix
        }
        .maxByOrNull { it.second }
        ?.first

    private fun serviceKey(info: NsdServiceInfo): String {
        val networkHandle = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            info.network?.networkHandle ?: 0L
        } else {
            0L
        }
        return "${info.serviceName}\u0000$networkHandle"
    }

    private fun supportsServiceInfoCallback(): Boolean {
        val extension = if (Build.VERSION.SDK_INT == Build.VERSION_CODES.TIRAMISU) {
            SdkExtensions.getExtensionVersion(Build.VERSION_CODES.TIRAMISU)
        } else {
            0
        }
        return nativeLocalDiscoverySupportsServiceInfoCallback(Build.VERSION.SDK_INT, extension)
    }

    private fun unregisterServiceInfoCallbackLocked(key: String) {
        serviceInfoSubscriptions.remove(key)?.close()
    }

    @Synchronized
    private fun stopLocked() {
        val current = listener
        listener = null
        known.clear()
        pending.clear()
        foundAtElapsedMillis.clear()
        resolvingKey = null
        val subscriptions = serviceInfoSubscriptions.values.toList()
        serviceInfoSubscriptions.clear()
        subscriptions.forEach(NativeLocalDiscoverySubscription::close)
        if (current != null) runCatching { manager.stopServiceDiscovery(current) }
        multicastLock?.let { lock -> runCatching { if (lock.isHeld) lock.release() } }
        multicastLock = null
    }

    private fun acquireMulticastLockIfNeeded() {
        val extension = if (Build.VERSION.SDK_INT == Build.VERSION_CODES.TIRAMISU) {
            SdkExtensions.getExtensionVersion(Build.VERSION_CODES.TIRAMISU)
        } else {
            0
        }
        if (!nativeLocalDiscoveryNeedsMulticastLock(Build.VERSION.SDK_INT, extension)) return
        multicastLock = wifiManager?.createMulticastLock("AnyTTYLocalDiscovery")?.apply {
            setReferenceCounted(false)
            acquire()
        }
    }

    override fun close() {
        synchronized(this) { stopLocked() }
        NativeLocalDiscoveryCache.clear()
		onChanged()
    }
}

private interface NativeLocalDiscoverySubscription {
    fun close()
}

// Kept outside NativeLocalDiscovery so Android versions without the modular
// ServiceInfoCallback API never resolve this platform type while loading the plugin.
@SuppressLint("NewApi")
private class NativeLocalDiscoveryModernSubscription(
    private val manager: NsdManager,
    onFailure: (Int) -> Unit,
    onUnregistered: () -> Unit,
    onLost: () -> Unit,
    onUpdated: (NsdServiceInfo) -> Unit,
) : NativeLocalDiscoverySubscription {
    private var registered = false
    private val callback = object : NsdManager.ServiceInfoCallback {
        override fun onServiceInfoCallbackRegistrationFailed(errorCode: Int) = onFailure(errorCode)
        override fun onServiceInfoCallbackUnregistered() = onUnregistered()
        override fun onServiceLost() = onLost()
        override fun onServiceUpdated(serviceInfo: NsdServiceInfo) = onUpdated(serviceInfo)
    }

    fun start(serviceInfo: NsdServiceInfo, executor: Executor) {
        manager.registerServiceInfoCallback(serviceInfo, executor, callback)
        registered = true
    }

    override fun close() {
        if (registered) runCatching { manager.unregisterServiceInfoCallback(callback) }
        registered = false
    }
}
