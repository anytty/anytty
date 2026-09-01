package com.anytty.app.goclient

import android.content.Context
import com.anytty.app.AnyTTYDiagnosticStore
import com.anytty.app.BuildConfig
import com.anytty.app.NativeLocalDiscoveryCache
import com.anytty.app.nativeLocalDiscoveryResult
import com.google.protobuf.ByteString
import anytty.api.v1.Common
import anytty.client.binding.v1.ClientBinding
import java.util.concurrent.CancellationException
import java.util.concurrent.ExecutionException
import java.util.concurrent.Executors
import java.util.concurrent.Future
import java.util.concurrent.TimeUnit
import java.util.concurrent.TimeoutException
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicReference
import java.util.concurrent.locks.ReentrantLock

internal const val ANDROID_CLIENT_PLATFORM_CLOSE_TIMEOUT_MILLIS = 500L

internal class AndroidPlatformStorageGeneration internal constructor()

/**
 * Process-wide fence for platform stores shared by successively-created Go engines.
 * Generation changes never wait for a platform primitive. The operation lock covers
 * the generation check and the actual persistence call, preserving commit order when
 * a retired primitive was already inside an uninterruptible KeyStore/store operation.
 */
internal class AndroidPlatformStorageCoordinator {
    private val currentGeneration = AtomicReference<AndroidPlatformStorageGeneration?>(null)
    private val operationLock = ReentrantLock(true)

    fun beginGeneration(): AndroidPlatformStorageGeneration =
        AndroidPlatformStorageGeneration().also(currentGeneration::set)

    fun retire(generation: AndroidPlatformStorageGeneration) {
        currentGeneration.compareAndSet(generation, null)
    }

    fun <T> withGeneration(
        generation: AndroidPlatformStorageGeneration,
        operation: () -> T,
    ): T {
        try {
            operationLock.lockInterruptibly()
        } catch (_: InterruptedException) {
            Thread.currentThread().interrupt()
            throw ClientPlatformFailure("cancelled", "platform storage operation was cancelled")
        }
        try {
            if (currentGeneration.get() !== generation) {
                throw ClientPlatformFailure("cancelled", "platform storage generation is retired")
            }
            return operation()
        } finally {
            operationLock.unlock()
        }
    }

    companion object {
        val process = AndroidPlatformStorageCoordinator()
    }
}

internal fun awaitAndroidClientPlatformPumpClose(
    pump: Future<*>,
    timeoutMillis: Long = ANDROID_CLIENT_PLATFORM_CLOSE_TIMEOUT_MILLIS,
): Boolean {
    require(timeoutMillis > 0L) { "platform pump close timeout must be positive" }
    return try {
        pump.get(timeoutMillis, TimeUnit.MILLISECONDS)
        true
    } catch (_: TimeoutException) {
        false
    } catch (_: ExecutionException) {
        true
    } catch (_: CancellationException) {
        true
    } catch (_: InterruptedException) {
        Thread.currentThread().interrupt()
        false
    }
}

internal fun runAndroidClientPlatformPump(
    active: AtomicBoolean,
    nextRequest: () -> ByteArray,
    handleRequest: (ByteArray) -> ByteArray,
    completeRequest: (ByteArray) -> Unit,
    onFatalError: (Exception) -> Unit = {},
) {
    while (active.get()) {
        val payload = try {
            nextRequest()
        } catch (failure: Exception) {
            if (active.get()) onFatalError(failure)
            return
        }
        if (!active.get()) return
        val response = try {
            handleRequest(payload)
        } catch (failure: Exception) {
            if (active.get()) onFatalError(failure)
            return
        }
        try {
            completeRequest(response)
        } catch (failure: Exception) {
            // The C ABI accepts a response for a legitimately retired request. Any
            // exception here means the engine/pump contract itself is no longer usable.
            if (active.get()) onFatalError(failure)
            return
        }
    }
}

/**
 * AndroidClientPlatform 是 Go Client Engine 的 Android primitive adapter。
 * 它只处理 bindingpb PlatformRequest；连接、认证、Hello、API、generation 和重连真值全部留在 Go。
 */
class AndroidClientPlatform(
    context: Context,
    private val engineHandle: Long,
    private val credentials: AndroidClientAccessCredentialStore = AndroidClientAccessCredentialStore(context.applicationContext),
    private val sshCredentials: AndroidSSHCredentialStore = AndroidSSHCredentialStore(),
    private val endpointRegistry: AndroidEndpointRegistryStore = AndroidEndpointRegistryStore(context.applicationContext),
    private val onFatalError: (Exception) -> Unit,
) : AutoCloseable {
    private val active = AtomicBoolean(true)
    private val storageCoordinator = AndroidPlatformStorageCoordinator.process
    private val storageGeneration = storageCoordinator.beginGeneration()
    private val executor = Executors.newSingleThreadExecutor { runnable ->
        Thread(runnable, "anytty-go-platform").apply { isDaemon = true }
    }
    private val pump: Future<*> = executor.submit(::runPump)

    fun deactivate() {
        if (active.compareAndSet(true, false)) {
            storageCoordinator.retire(storageGeneration)
            executor.shutdownNow()
        }
    }

    override fun close() {
        deactivate()
        // A platform primitive such as AndroidKeyStore may ignore interruption.
        // Never let that worker hold the process runtime owner's monitor forever.
        awaitAndroidClientPlatformPumpClose(pump)
    }

    private fun runPump() {
        runAndroidClientPlatformPump(
            active = active,
            nextRequest = { GoClientNative.nextPlatformRequest(engineHandle, 0) },
            handleRequest = { payload ->
                dispatch(ClientBinding.PlatformRequest.parseFrom(payload)).toByteArray()
            },
            completeRequest = { response ->
                GoClientNative.completePlatformRequest(engineHandle, response)
            },
            onFatalError = onFatalError,
        )
    }

    private fun dispatch(request: ClientBinding.PlatformRequest): ClientBinding.PlatformResponse {
        val response = ClientBinding.PlatformResponse.newBuilder().setRequestId(request.requestId)
        return try {
            storageCoordinator.withGeneration(storageGeneration) {
                when (request.requestCase) {
                    ClientBinding.PlatformRequest.RequestCase.CREDENTIAL_PREPARE ->
                        response.setCredential(credentials.prepareRecord(
                            request.credentialPrepare.credentialRef,
                            request.credentialPrepare.endpointId,
                        ))
                    ClientBinding.PlatformRequest.RequestCase.CREDENTIAL_RESOLVE ->
                        response.setCredential(credentials.resolveRecord(
                            request.credentialResolve.credentialRef,
                            request.credentialResolve.endpointId,
                        ))
                    ClientBinding.PlatformRequest.RequestCase.CREDENTIAL_DELETE -> {
                        credentials.delete(request.credentialDelete.credentialRef)
                    }
                    ClientBinding.PlatformRequest.RequestCase.CREDENTIAL_SIGN ->
                        response.setCredentialSign(ClientBinding.CredentialSignResponse.newBuilder()
                            .setSignature(ByteString.copyFrom(credentials.sign(
                                request.credentialSign.credentialRef,
                                request.credentialSign.payload.toByteArray(),
                            ))))
                    ClientBinding.PlatformRequest.RequestCase.CREDENTIAL_BIND ->
                        response.setCredential(credentials.bindRecord(
                            request.credentialBind.credentialRef,
                            request.credentialBind.endpointId,
                            request.credentialBind.capabilityGrant,
                            request.credentialBind.cloudRouteGrant.toByteArray(),
                            request.credentialBind.cloudEdgeLocator.toByteArray(),
                        ))
                    ClientBinding.PlatformRequest.RequestCase.ENDPOINT_REGISTRY_LOAD ->
                        response.setEndpointRegistry(ClientBinding.EndpointRegistryLoaded.newBuilder()
                            .setRegistryProto(ByteString.copyFrom(endpointRegistry.load())))
                    ClientBinding.PlatformRequest.RequestCase.ENDPOINT_REGISTRY_STORE -> {
                        endpointRegistry.store(
                            request.endpointRegistryStore.registryProto.toByteArray(),
                            request.endpointRegistryStore.deleteCredentialRefsList,
                            credentials,
                            sshCredentials,
                        )
                    }
                    ClientBinding.PlatformRequest.RequestCase.SSH_CREDENTIAL_LOOKUP ->
                        response.setSshCredential(sshCredentials.lookup(
                            request.sshCredentialLookup.credentialRef,
                            request.sshCredentialLookup.createIfMissing,
                        ))
                    ClientBinding.PlatformRequest.RequestCase.SSH_CREDENTIAL_SIGN ->
                        response.setSshCredentialSign(ClientBinding.SSHCredentialSignResponse.newBuilder()
                            .setSignature(ByteString.copyFrom(sshCredentials.sign(
                                request.sshCredentialSign.credentialRef,
                                request.sshCredentialSign.digest.toByteArray(),
                                request.sshCredentialSign.hash,
                            ))))
                    ClientBinding.PlatformRequest.RequestCase.SSH_CREDENTIAL_DELETE ->
                        sshCredentials.delete(request.sshCredentialDelete.credentialRef)
                    ClientBinding.PlatformRequest.RequestCase.CLOUD_PROFILE_RESOLVE ->
                        throw ClientPlatformFailure("protocol", "Cloud profile resolution is owned by Go")
                    ClientBinding.PlatformRequest.RequestCase.LOCAL_DISCOVERY_LOOKUP -> {
                        val lookup = request.localDiscoveryLookup
                        response.setLocalDiscovery(nativeLocalDiscoveryResult(
                            NativeLocalDiscoveryCache.snapshot(lookup.deviceId, lookup.deviceFingerprint),
                        ))
                    }
                    ClientBinding.PlatformRequest.RequestCase.REQUEST_NOT_SET ->
                        throw ClientPlatformFailure("protocol", "platform request payload is missing")
                    null -> throw ClientPlatformFailure("protocol", "platform request case is invalid")
                }
            }
            response.build()
        } catch (failure: ClientPlatformFailure) {
            response.setError(platformError(failure.code, failure.message ?: failure.code)).build()
        } catch (_: Throwable) {
            response.setError(platformError("temporary", "Android platform request failed")).build()
        }
    }

    private fun platformError(code: String, message: String): Common.ApiError {
        val apiCode = when (code) {
            "protocol" -> Common.ApiErrorCode.API_ERROR_CODE_INVALID_REQUEST
            "unauthenticated", "login_required", "capability_invalid", "capability_expired" ->
                Common.ApiErrorCode.API_ERROR_CODE_UNAUTHORIZED
            // quota_exhausted 表示 Hub 明确拒绝且没有创建 signaling session。Go Client Engine
            // 只允许对这一类可证明未产生服务端副作用的冲突执行有界重试。
            "quota_exhausted" -> Common.ApiErrorCode.API_ERROR_CODE_CONFLICT
            "entitlement_denied" -> Common.ApiErrorCode.API_ERROR_CODE_ENTITLEMENT_DENIED
            "cancelled" -> Common.ApiErrorCode.API_ERROR_CODE_CANCELLED
            "route_unavailable", "temporary", "companion_missing", "backpressure" -> Common.ApiErrorCode.API_ERROR_CODE_UNAVAILABLE
            else -> Common.ApiErrorCode.API_ERROR_CODE_INTERNAL
        }
        return Common.ApiError.newBuilder()
            .setCode(apiCode)
            .setMessage(message)
            .setRetryable(apiCode == Common.ApiErrorCode.API_ERROR_CODE_UNAVAILABLE || code == "quota_exhausted")
            .setAttempted(true)
            .build()
    }
}

/** AndroidGoClientEngine owns one production Go engine and its platform pump. */
class AndroidGoClientEngine(
    context: Context,
    onPlatformFailure: (Exception) -> Unit,
) : AutoCloseable {
    val handle: Long
    init {
        GoClientNative.setDebugLogPath(AnyTTYDiagnosticStore.nativeLogPath(context))
        check(GoClientNative.abiVersion() == GoClientNative.ABI_VERSION) { "AnyTTY native client ABI mismatch" }
        handle = GoClientNative.create()
    }
    private val platform = AndroidClientPlatform(context, handle, onFatalError = onPlatformFailure)
    private val closed = AtomicBoolean(false)

    override fun close() {
        if (!closed.compareAndSet(false, true)) return
        platform.deactivate()
        try {
            GoClientNative.close(handle)
        } finally {
            platform.close()
        }
    }
}
