package com.anytty.app

import android.content.Context
import android.net.ConnectivityManager
import android.net.LinkProperties
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.util.Log
import androidx.annotation.Keep
import org.json.JSONArray
import org.json.JSONObject

// A process-scoped observer publishes metadata only; Go owns all dialing.
@Keep
object AndroidNetworkBridge {
    private var started = false
    private external fun updateNetworks(snapshot: String)

    @Synchronized
    fun start(context: Context) {
        if (started) return
        System.loadLibrary("anytty_client")
        val manager = context.applicationContext.getSystemService(ConnectivityManager::class.java)
        val callback = object : ConnectivityManager.NetworkCallback() {
            private val links = mutableMapOf<Network, LinkProperties>()

            override fun onLinkPropertiesChanged(network: Network, properties: LinkProperties) {
                synchronized(this) { links[network] = properties; publish() }
            }

            override fun onLost(network: Network) {
                synchronized(this) { links.remove(network); publish() }
            }

            private fun publish() {
                val paths = JSONArray()
                for ((network, properties) in links.entries.sortedBy { it.key.networkHandle }) {
                    val addresses = properties.linkAddresses.filter {
                        !it.address.isLoopbackAddress && !it.address.isAnyLocalAddress && !it.address.isMulticastAddress
                    }
                    if (addresses.isEmpty()) continue
                    paths.put(JSONObject().apply {
                        put("handle", network.networkHandle)
                        put("name", properties.interfaceName ?: "network-${network.networkHandle}")
                        put("addresses", JSONArray(addresses.map { "${it.address.hostAddress?.substringBefore('%')}/${it.prefixLength}" }))
                        put("dns", JSONArray(properties.dnsServers.mapNotNull { it.hostAddress }))
                    })
                }
                updateNetworks(paths.toString())
                Log.i("AnyTTYNetwork", "stage=snapshot networks=${paths.length()}")
            }
        }
        try {
            manager.registerNetworkCallback(NetworkRequest.Builder()
                .clearCapabilities()
                .addCapability(NetworkCapabilities.NET_CAPABILITY_NOT_RESTRICTED)
                .build(), callback)
            started = true
        } catch (error: Exception) {
            Log.w("AnyTTYNetwork", "network enumeration unavailable", error)
        }
    }
}
