package com.anytty.app.goclient

import android.content.Context
import android.util.Base64

/**
 * AndroidEndpointRegistryStore 只保存 Go Client Engine 生成的 opaque EndpointRegistryV1 bytes。
 * Kotlin 不解析、索引或合并 payload；所有 identity、Route 和 default 语义都留在 Go。
 */
class AndroidEndpointRegistryStore(context: Context) {
    private val preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
    private val lock = Any()

    /** load 返回独立 bytes；空数组表示当前平台尚未保存 registry。 */
    fun load(): ByteArray = synchronized(lock) {
        val encoded = preferences.getString(REGISTRY_KEY, null) ?: return@synchronized ByteArray(0)
        try {
            Base64.decode(encoded, Base64.NO_WRAP)
        } catch (_: IllegalArgumentException) {
            throw ClientPlatformFailure("protocol", "endpoint registry payload is malformed")
        }
    }

    fun clear() = synchronized(lock) {
        if (!preferences.edit().remove(REGISTRY_KEY).commit()) {
            throw ClientPlatformFailure("temporary", "failed to clear endpoint registry")
        }
    }

    /**
     * store 先提交 registry bytes，再用一次 preferences commit 清理 Go 已判定不再引用的 credential refs。
     * credential commit 失败时恢复旧 registry，使 Go 只在整组平台写入成功后发布新 snapshot。
     */
    fun store(
        registryProto: ByteArray,
        deleteCredentialRefs: List<String>,
        credentials: AndroidClientAccessCredentialStore,
        sshCredentials: AndroidSSHCredentialStore,
    ) = synchronized(lock) {
        if (registryProto.isEmpty() || registryProto.size > MAX_REGISTRY_BYTES) {
            throw ClientPlatformFailure("protocol", "endpoint registry payload size is invalid")
        }
        val previous = preferences.getString(REGISTRY_KEY, null)
        val encoded = Base64.encodeToString(registryProto, Base64.NO_WRAP)
        if (!preferences.edit().putString(REGISTRY_KEY, encoded).commit()) {
            throw ClientPlatformFailure("temporary", "failed to persist endpoint registry")
        }
        try {
            credentials.deleteMany(deleteCredentialRefs.filterNot { it.startsWith(AndroidSSHCredentialStore.REF_PREFIX) })
            sshCredentials.deleteMany(deleteCredentialRefs.filter { it.startsWith(AndroidSSHCredentialStore.REF_PREFIX) })
        } catch (failure: Throwable) {
            val rollback = preferences.edit()
            if (previous == null) rollback.remove(REGISTRY_KEY) else rollback.putString(REGISTRY_KEY, previous)
            if (!rollback.commit()) {
                throw ClientPlatformFailure("temporary", "failed to roll back endpoint registry")
            }
            throw failure
        }
    }

    companion object {
        private const val PREFERENCES_NAME = "anytty_go_client_engine_v1"
        private const val REGISTRY_KEY = "endpoint_registry_proto"
        private const val MAX_REGISTRY_BYTES = 1 shl 20
    }
}
