package com.anytty.app.goclient

import android.content.Context
import android.util.Base64
import com.anytty.app.BuildConfig
import com.google.protobuf.ByteString
import anytty.api.v1.Common
import anytty.client.binding.v1.ClientBinding
import java.util.concurrent.Executors
import java.util.concurrent.Future
import java.util.concurrent.atomic.AtomicBoolean

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
) : AutoCloseable {
    private val active = AtomicBoolean(true)
    private val executor = Executors.newSingleThreadExecutor { runnable ->
        Thread(runnable, "anytty-go-platform").apply { isDaemon = true }
    }
    private val pump: Future<*> = executor.submit(::runPump)

    override fun close() {
        if (!active.compareAndSet(true, false)) return
        executor.shutdownNow()
        runCatching { pump.get() }
    }

    private fun runPump() {
        while (active.get()) {
            val payload = try {
                GoClientNative.nextPlatformRequest(engineHandle, 0)
            } catch (_: IllegalStateException) {
                return
            }
            val request = try {
                ClientBinding.PlatformRequest.parseFrom(payload)
            } catch (_: Exception) {
                return
            }
            val response = dispatch(request)
            try {
                GoClientNative.completePlatformRequest(engineHandle, response.toByteArray())
            } catch (_: IllegalStateException) {
                return
            }
        }
    }

    private fun dispatch(request: ClientBinding.PlatformRequest): ClientBinding.PlatformResponse {
        val response = ClientBinding.PlatformResponse.newBuilder().setRequestId(request.requestId)
        return try {
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
                    response.setCloudProfile(resolveCloudProfile(request.cloudProfileResolve.accountProfileRef))
                ClientBinding.PlatformRequest.RequestCase.REQUEST_NOT_SET ->
                    throw ClientPlatformFailure("protocol", "platform request payload is missing")
                null -> throw ClientPlatformFailure("protocol", "platform request case is invalid")
            }
            response.build()
        } catch (failure: ClientPlatformFailure) {
            response.setError(platformError(failure.code, failure.message ?: failure.code)).build()
        } catch (_: Throwable) {
            response.setError(platformError("temporary", "Android platform request failed")).build()
        }
    }

    /** resolveCloudProfile 只暴露构建时注入的 TLS locator；所有 Cloud 协议仍由 Go 执行。 */
    private fun resolveCloudProfile(reference: String): ClientBinding.CloudProfileRecord {
        val normalized = reference.trim()
        val address = BuildConfig.ANYTTY_CLOUD_CONTROLLER_ADDRESS.trim()
        val serverName = BuildConfig.ANYTTY_CLOUD_CONTROLLER_SERVER_NAME.trim()
        if (normalized != "default" || address.isBlank() || serverName.isBlank()) {
            throw ClientPlatformFailure("route_unavailable", "AnyTTY Cloud profile is unavailable")
        }
        val caPEM = BuildConfig.ANYTTY_CLOUD_CONTROLLER_CA_PEM_BASE64.trim().let {
            if (it.isBlank()) byteArrayOf() else Base64.decode(it, Base64.DEFAULT)
        }
        return ClientBinding.CloudProfileRecord.newBuilder()
            .setAccountProfileRef(normalized)
            .setControllerAddress(address)
            .setControllerServerName(serverName)
            .setControllerCaPem(ByteString.copyFrom(caPEM))
            .build()
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
class AndroidGoClientEngine(context: Context) : AutoCloseable {
    val handle: Long = GoClientNative.create()
    private val platform = AndroidClientPlatform(context, handle)
    private val closed = AtomicBoolean(false)

    override fun close() {
        if (!closed.compareAndSet(false, true)) return
        GoClientNative.close(handle)
        platform.close()
    }
}
