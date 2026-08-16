package com.anytty.app

import android.content.Context
import android.util.Base64
import com.anytty.app.goclient.AndroidClientAccessCredentialStore
import com.anytty.app.goclient.AndroidEndpointRegistryStore
import com.anytty.app.goclient.AndroidGoClientEngine
import com.anytty.app.goclient.AndroidSSHCredentialStore
import com.anytty.app.goclient.GoClientNative
import java.security.SecureRandom

internal object NativeConnectionRuntimeOwner {
    private const val BRIDGE_TOKEN_BYTES = 32

    internal data class BridgeEndpoint(val port: Int, val token: String)

    private var bridgeEndpoint: BridgeEndpoint? = null
    private var goEngine: AndroidGoClientEngine? = null
    private val activeEndpoints = linkedSetOf<String>()

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
    fun hasActiveEndpoints(): Boolean = activeEndpoints.isNotEmpty()

    fun requestDisconnectAll() {
        synchronized(this) {
            activeEndpoints.clear()
        }
        NativeConnectionPlugin.notifyDisconnectAllRequested()
    }

    @Synchronized
    internal fun stopForTests() {
        activeEndpoints.clear()
        stopRuntimeLocked()
    }

    private fun stopRuntimeLocked() {
        bridgeEndpoint = null
        goEngine?.close()
        goEngine = null
    }

    private fun generateBridgeToken(): String {
        val bytes = ByteArray(BRIDGE_TOKEN_BYTES)
        SecureRandom().nextBytes(bytes)
        return Base64.encodeToString(bytes, Base64.URL_SAFE or Base64.NO_PADDING or Base64.NO_WRAP)
    }
}
