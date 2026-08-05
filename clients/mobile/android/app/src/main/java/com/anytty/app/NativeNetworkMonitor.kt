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
    private val onStableNetworkChanged: (epoch: Long, connected: Boolean) -> Unit,
) : AutoCloseable {
    companion object {
        internal const val STABILITY_DELAY_MILLIS = 3_000L
    }

    private data class Signature(
        val networkHandle: Long,
        val validated: Boolean,
        val addresses: List<String>,
    ) {
        val connected: Boolean get() = networkHandle != 0L && validated
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
            if (closed || settleScheduled) return@post
            settleScheduled = true
            handler.postDelayed(settle, STABILITY_DELAY_MILLIS)
        }
    }

    private fun publishStableNetwork() {
        if (closed) return
        val current = currentSignature()
        if (current == lastStable) return
        lastStable = current
        epoch += 1
        onStableNetworkChanged(epoch, current.connected)
    }

    private fun currentSignature(): Signature {
        val network = connectivity.activeNetwork ?: return Signature(0L, false, emptyList())
        val capabilities = connectivity.getNetworkCapabilities(network)
        val linkProperties = connectivity.getLinkProperties(network)
        val validated = capabilities?.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED) == true
        val addresses = linkProperties?.linkAddresses
            ?.map { it.address.hostAddress.orEmpty() }
            ?.filter { it.isNotBlank() }
            ?.sorted()
            .orEmpty()
        return Signature(network.networkHandle, validated, addresses)
    }

    override fun close() {
        if (closed) return
        closed = true
        handler.removeCallbacks(settle)
        settleScheduled = false
        runCatching { connectivity.unregisterNetworkCallback(callback) }
    }
}
