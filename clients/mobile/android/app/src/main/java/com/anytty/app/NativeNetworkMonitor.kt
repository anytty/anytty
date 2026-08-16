package com.anytty.app

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.os.Handler
import android.os.Looper

internal class NativeNetworkMonitor(
    context: Context,
    private val onStableNetworkChanged: (epoch: Long, connected: Boolean, reason: String) -> Unit,
) : AutoCloseable {
    companion object {
        internal const val PATH_SETTLE_DELAY_MILLIS = 200L
        internal const val OFFLINE_GRACE_DELAY_MILLIS = 750L
    }

    internal data class Signature(
        val networkHandle: Long,
        val internet: Boolean,
        val validated: Boolean,
        val addresses: List<String>,
        val dnsServers: List<String>,
        val routes: List<String>,
    ) {
        // VALIDATED is an internet-quality hint, not an end-to-end verdict. A default
        // route may still carry a healthy LAN P2P session or reach a captive portal.
        val connected: Boolean get() = networkHandle != 0L
    }

    private val connectivity = context.applicationContext
        .getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    private val handler = Handler(Looper.getMainLooper())
    private var lastStable = currentSignature()
    private var epoch = 0L
    private var closed = false
    private var settleScheduled = false
    private val settle = Runnable {
        settleScheduled = false
        publishStableNetwork()
    }
    private val callback = object : ConnectivityManager.NetworkCallback() {
        override fun onAvailable(network: Network) = scheduleCheck()
        override fun onLost(network: Network) = scheduleCheck()
        override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) = scheduleCheck()
        override fun onLinkPropertiesChanged(network: Network, linkProperties: LinkProperties) = scheduleCheck()
    }

    fun start() {
        connectivity.registerDefaultNetworkCallback(callback)
    }

    private fun scheduleCheck() {
        handler.post {
            if (closed) return@post
            if (settleScheduled) handler.removeCallbacks(settle)
            settleScheduled = true
            val delay = if (connectivity.activeNetwork == null) {
                OFFLINE_GRACE_DELAY_MILLIS
            } else {
                PATH_SETTLE_DELAY_MILLIS
            }
            handler.postDelayed(settle, delay)
        }
    }

    private fun publishStableNetwork() {
        if (closed) return
        val current = currentSignature()
        if (current == lastStable) return
        val previous = lastStable
        lastStable = current
        if (!requiresSessionRecovery(previous, current)) return
        epoch += 1
        onStableNetworkChanged(epoch, current.connected, networkChangeReason(previous, current))
    }

    private fun currentSignature(): Signature {
        val network = connectivity.activeNetwork ?: return Signature(
            0L,
            false,
            false,
            emptyList(),
            emptyList(),
            emptyList(),
        )
        val capabilities = connectivity.getNetworkCapabilities(network)
        val linkProperties = connectivity.getLinkProperties(network)
        val internet = capabilities?.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) == true
        val validated = capabilities?.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED) == true
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
        return Signature(network.networkHandle, internet, validated, addresses, dnsServers, routes)
    }

    override fun close() {
        if (closed) return
        closed = true
        handler.removeCallbacks(settle)
        settleScheduled = false
        runCatching { connectivity.unregisterNetworkCallback(callback) }
    }
}

internal fun requiresSessionRecovery(
    previous: NativeNetworkMonitor.Signature,
    current: NativeNetworkMonitor.Signature,
): Boolean = previous.networkHandle != current.networkHandle ||
    previous.connected != current.connected ||
    previous.internet != current.internet ||
    previous.addresses != current.addresses ||
    previous.dnsServers != current.dnsServers ||
    previous.routes != current.routes

internal fun networkChangeReason(
    previous: NativeNetworkMonitor.Signature,
    current: NativeNetworkMonitor.Signature,
): String = when {
    !current.connected -> "offline"
    !previous.connected -> "available"
    previous.networkHandle != current.networkHandle -> "network_replaced"
    else -> "path_changed"
}
