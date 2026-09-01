package com.anytty.app

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import android.net.wifi.WifiManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.ext.SdkExtensions
import android.util.Log
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.nio.charset.StandardCharsets

class AndroidLocalDiscoveryBridge(context: Context) {
    private data class Candidate(
        val serviceName: String,
        val discoveryKey: String,
        val address: String,
        val port: Int,
        val protocolVersion: Int,
        val expiresAtUnixNano: Long,
        val networkHandle: Long,
    )

    private val applicationContext = context.applicationContext
    private val manager = applicationContext.getSystemService(NsdManager::class.java)
    private val handler = Handler(Looper.getMainLooper())
    private val candidates = mutableMapOf<String, List<Candidate>>()
    private val services = mutableMapOf<String, NsdServiceInfo>()
    private val resolving = mutableSetOf<String>()
    private var discoveryListener: NsdManager.DiscoveryListener? = null
    private var multicastLock: WifiManager.MulticastLock? = null
    private var idleGeneration = 0

    fun handle(call: MethodCall, result: MethodChannel.Result) {
        if (call.method != "lookup") {
            result.notImplemented()
            return
        }
        val expectedKey = call.argument<String>("expectedKey")?.trim().orEmpty()
        if (!expectedKey.matches(Regex("^[0-9a-f]{64}$"))) {
            result.error("protocol", "Local discovery identity is invalid", null)
            return
        }
        ensureDiscovery()
        refreshKnownServices()
        scheduleIdleStop()
        val cached = matching(expectedKey)
        if (cached.isNotEmpty()) {
            Log.i(LOG_TAG, "stage=lookup source=cache candidates=${cached.size}")
            result.success(cached)
            return
        }
        handler.postDelayed({
            val discovered = matching(expectedKey)
            Log.i(LOG_TAG, "stage=lookup source=browse candidates=${discovered.size}")
            result.success(discovered)
        }, LOOKUP_WAIT_MS)
    }

    fun close() {
        idleGeneration += 1
        stopDiscovery()
        candidates.clear()
        services.clear()
        resolving.clear()
    }

    private fun ensureDiscovery() {
        if (discoveryListener != null) return
        acquireMulticastLockIfNeeded()
        val listener = object : NsdManager.DiscoveryListener {
            override fun onDiscoveryStarted(serviceType: String) = Unit

            override fun onServiceFound(serviceInfo: NsdServiceInfo) {
                services[serviceInfo.serviceName] = serviceInfo
                resolve(serviceInfo)
            }

            override fun onServiceLost(serviceInfo: NsdServiceInfo) {
                val name = serviceInfo.serviceName
                services.remove(name)
                candidates.remove(name)
                resolving.remove(name)
            }

            override fun onDiscoveryStopped(serviceType: String) = Unit

            override fun onStartDiscoveryFailed(serviceType: String, errorCode: Int) {
                if (discoveryListener === this) discoveryListener = null
                releaseMulticastLock()
            }

            override fun onStopDiscoveryFailed(serviceType: String, errorCode: Int) = Unit
        }
        discoveryListener = listener
        try {
            manager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, listener)
        } catch (_: Exception) {
            discoveryListener = null
            releaseMulticastLock()
        }
    }

    @Suppress("DEPRECATION")
    private fun resolve(serviceInfo: NsdServiceInfo) {
        val name = serviceInfo.serviceName
        if (!resolving.add(name)) return
        try {
            manager.resolveService(
                serviceInfo,
                object : NsdManager.ResolveListener {
                    override fun onResolveFailed(value: NsdServiceInfo, errorCode: Int) {
                        resolving.remove(name)
                    }

                    override fun onServiceResolved(value: NsdServiceInfo) {
                        resolving.remove(name)
                        services[name] = value
                        cacheResolved(value)
                    }
                },
            )
        } catch (_: Exception) {
            resolving.remove(name)
        }
    }

    @Suppress("DEPRECATION")
    private fun cacheResolved(serviceInfo: NsdServiceInfo) {
        val attributes = serviceInfo.attributes
        val schema = attributes.text("v")?.toIntOrNull()
        val protocolVersion = attributes.text("p")?.toIntOrNull()
        val discoveryKey = attributes.text("k")?.lowercase()
        if (schema != 1 ||
            protocolVersion == null ||
            protocolVersion <= 0 ||
            discoveryKey?.matches(Regex("^[0-9a-f]{64}$")) != true ||
            serviceInfo.port !in 1..65535
        ) {
            candidates.remove(serviceInfo.serviceName)
            return
        }
        val addresses = if (Build.VERSION.SDK_INT >= 34) {
            serviceInfo.hostAddresses
        } else {
            listOfNotNull(serviceInfo.host)
        }
        val networkHandle = if (Build.VERSION.SDK_INT >= 33) {
            serviceInfo.network?.networkHandle ?: 0L
        } else {
            0L
        }
        val expires = unixNanoNow() + CANDIDATE_TTL_NANOS
        candidates[serviceInfo.serviceName] = addresses
            .mapNotNull { it.hostAddress?.trim() }
            .filter { it.isNotEmpty() }
            .distinct()
            .take(MAX_ADDRESSES_PER_SERVICE)
            .map { address ->
                Candidate(
                    serviceName = serviceInfo.serviceName,
                    discoveryKey = discoveryKey,
                    address = address,
                    port = serviceInfo.port,
                    protocolVersion = protocolVersion,
                    expiresAtUnixNano = expires,
                    networkHandle = networkHandle,
                )
            }
    }

    private fun matching(expectedKey: String): List<Map<String, Any>> {
        val now = unixNanoNow()
        return candidates.values
            .flatten()
            .asSequence()
            .filter { it.discoveryKey == expectedKey && it.expiresAtUnixNano > now }
            .distinctBy { Triple(it.address, it.port, it.networkHandle) }
            .take(MAX_CANDIDATES)
            .map { candidate ->
                mapOf(
                    "address" to candidate.address,
                    "port" to candidate.port,
                    "protocolVersion" to candidate.protocolVersion,
                    "expiresAtUnixNano" to candidate.expiresAtUnixNano,
                    "networkHandle" to candidate.networkHandle,
                )
            }
            .toList()
    }

    private fun refreshKnownServices() {
        for (service in services.values.toList()) resolve(service)
    }

    private fun scheduleIdleStop() {
        val generation = ++idleGeneration
        handler.postDelayed({
            if (generation == idleGeneration) stopDiscovery()
        }, IDLE_STOP_MS)
    }

    private fun stopDiscovery() {
        val listener = discoveryListener
        discoveryListener = null
        if (listener != null) {
            try {
                manager.stopServiceDiscovery(listener)
            } catch (_: Exception) {
                // The platform may already have stopped a failed discovery.
            }
        }
        releaseMulticastLock()
    }

    private fun acquireMulticastLockIfNeeded() {
        if (!requiresManualMulticastLock() || multicastLock != null) return
        val wifi = applicationContext.getSystemService(WifiManager::class.java)
        multicastLock = wifi.createMulticastLock("anytty-local-discovery").apply {
            setReferenceCounted(false)
            acquire()
        }
    }

    private fun requiresManualMulticastLock(): Boolean {
        if (Build.VERSION.SDK_INT < 30) return true
        if (Build.VERSION.SDK_INT < 33) return true
        return SdkExtensions.getExtensionVersion(Build.VERSION_CODES.TIRAMISU) < 7
    }

    private fun releaseMulticastLock() {
        multicastLock?.let { lock ->
            if (lock.isHeld) lock.release()
        }
        multicastLock = null
    }

    private fun Map<String, ByteArray>.text(key: String): String? =
        this[key]?.toString(StandardCharsets.UTF_8)?.trim()

    private fun unixNanoNow(): Long = System.currentTimeMillis() * 1_000_000L

    companion object {
        private const val SERVICE_TYPE = "_anytty._tcp"
        private const val LOG_TAG = "AnyTTYDiscovery"
        private const val LOOKUP_WAIT_MS = 1_400L
        private const val IDLE_STOP_MS = 30_000L
        private const val CANDIDATE_TTL_NANOS = 30_000_000_000L
        private const val MAX_ADDRESSES_PER_SERVICE = 8
        private const val MAX_CANDIDATES = 64
    }
}
