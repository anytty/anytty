package com.anytty.app

import android.content.Context
import android.util.Base64
import com.anytty.app.goclient.AndroidClientAccessCredentialStore
import com.anytty.app.goclient.AndroidEndpointRegistryStore
import com.anytty.app.goclient.AndroidGoClientEngine
import com.anytty.app.goclient.AndroidSSHCredentialStore
import com.anytty.app.goclient.GoClientBridgeServer
import java.security.SecureRandom

internal object NativeConnectionRuntimeOwner {
    private const val BRIDGE_TOKEN_BYTES = 32

    internal data class BridgeEndpoint(val port: Int, val token: String)

    private var bridgeEndpoint: BridgeEndpoint? = null
    private var goBridgeServer: GoClientBridgeServer? = null
    private val activeEndpoints = linkedSetOf<String>()

    @Synchronized
    fun isStarted(): Boolean = bridgeEndpoint != null && goBridgeServer != null

    @Synchronized
    fun ensureStarted(context: Context): BridgeEndpoint {
        bridgeEndpoint?.let { endpoint ->
            if (goBridgeServer != null) return endpoint
        }

        val token = generateBridgeToken()
        val engine = AndroidGoClientEngine(context.applicationContext)
        val bridge = GoClientBridgeServer(engine, token)
        try {
            bridge.start()
            if (!bridge.awaitStarted(5_000)) {
                throw IllegalStateException("Go binding bridge start timed out")
            }
            val endpoint = BridgeEndpoint(bridge.port, token)
            goBridgeServer = bridge
            bridgeEndpoint = endpoint
            AnyTTYDebugLog.event(AnyTTYDebugEvent.BRIDGE_STARTED)
            return endpoint
        } catch (failure: Exception) {
            runCatching { bridge.close() }
            goBridgeServer = null
            bridgeEndpoint = null
            throw failure
        }
    }

    @Synchronized
    fun endpoint(): BridgeEndpoint = bridgeEndpoint
        ?: throw IllegalStateException("native bridge server is not ready")

    @Synchronized
    fun resetLocalPairings(context: Context): BridgeEndpoint {
        stopRuntimeLocked()
        AndroidEndpointRegistryStore(context.applicationContext).clear()
        AndroidClientAccessCredentialStore(context.applicationContext).clearAll()
        AndroidSSHCredentialStore().clearAll()
        return ensureStarted(context.applicationContext)
    }

    fun setEndpointActive(context: Context, endpointId: String, active: Boolean) {
        val shouldStart: Boolean
        val shouldStop: Boolean
        synchronized(this) {
            val wasActive = activeEndpoints.isNotEmpty()
            if (active) activeEndpoints += endpointId else activeEndpoints -= endpointId
            val isActive = activeEndpoints.isNotEmpty()
            shouldStart = !wasActive && isActive
            shouldStop = wasActive && !isActive
        }
        if (shouldStart) AnyTTYConnectionService.start(context.applicationContext)
        if (shouldStop) AnyTTYConnectionService.stop(context.applicationContext)
    }

    @Synchronized
    internal fun stopForTests() {
        activeEndpoints.clear()
        stopRuntimeLocked()
    }

    private fun stopRuntimeLocked() {
        bridgeEndpoint = null
        goBridgeServer?.close()
        goBridgeServer = null
    }

    private fun generateBridgeToken(): String {
        val bytes = ByteArray(BRIDGE_TOKEN_BYTES)
        SecureRandom().nextBytes(bytes)
        return Base64.encodeToString(bytes, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
    }
}
