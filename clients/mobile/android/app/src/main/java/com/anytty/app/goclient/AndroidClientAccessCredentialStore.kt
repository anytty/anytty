package com.anytty.app.goclient

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import org.bouncycastle.crypto.params.Ed25519PrivateKeyParameters
import org.bouncycastle.crypto.signers.Ed25519Signer
import org.json.JSONObject
import anytty.client.binding.v1.ClientBinding
import java.security.KeyStore
import java.security.MessageDigest
import java.security.SecureRandom
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/**
 * AndroidClientAccessIdentity 是单个 Endpoint 专用的 Ed25519 访问身份。
 * privateKeySeed 只存在于 Android Keystore AES-GCM 保护的 native credential payload；不同 Endpoint 不得复用 key。
 */
data class AndroidClientAccessIdentity(
    val endpointId: String,
    val privateKeySeed: ByteArray,
    val publicKey: ByteArray,
    val fingerprint: String,
) {
    /** sign 仅用于 PairingExchange 或 capability channel-bound proof，不允许签 route 或 UI payload。 */
    fun sign(message: ByteArray): ByteArray = Ed25519Signer().run {
        init(true, Ed25519PrivateKeyParameters(privateKeySeed, 0))
        update(message, 0, message.size)
        generateSignature()
    }
}

/** AndroidClientAccessCredential 是 native secure store 返回给公开 authorizer 的完整 client-bound credential。 */
data class AndroidClientAccessCredential(
    val identity: AndroidClientAccessIdentity,
    val capabilityGrant: String,
    val cloudRouteGrant: ByteArray,
    val cloudEdgeLocator: ByteArray,
) {
    /** ready 只有在 PairingExchange 已返回并持久化非空 CapabilityGrant v2 后为 true。 */
    fun ready(): Boolean = capabilityGrant.isNotBlank()
}

/**
 * AndroidClientAccessCredentialStore 使用 Android Keystore AES-GCM key 原子保存 per-endpoint ClientAccessIdentity 与 bound grant。
 * 普通 endpoint registry/WebView 只持有 credential ref；解密结果只交给公开 PairingExchange/authorizer。
 */
class AndroidClientAccessCredentialStore(context: Context) {
    private val preferences = context.applicationContext.getSharedPreferences(PREFERENCES_NAME, Context.MODE_PRIVATE)
    private val random = SecureRandom()
    private val lock = Any()

    /**
     * loadOrCreateIdentity 在首次 pairing 前生成并持久化 Endpoint 专用 key。
     * 已存在 ref 必须绑定同一 endpointId；先写 key 使响应丢失后的重试仍能证明同一 subject possession。
     */
    fun loadOrCreateIdentity(credentialRef: String, endpointId: String): AndroidClientAccessCredential = synchronized(lock) {
        val normalizedRef = validateRef(credentialRef)
        val normalizedEndpoint = endpointId.trim().ifBlank {
            throw ClientPlatformFailure("protocol", "endpoint_id is required for ClientAccessIdentity")
        }
        readCredential(normalizedRef, requireGrant = false)?.let { existing ->
            if (existing.identity.endpointId != normalizedEndpoint) {
                throw ClientPlatformFailure("identity_conflict", "credential ref belongs to another endpoint")
            }
            return@synchronized existing
        }
        val privateKey = Ed25519PrivateKeyParameters(random)
        val publicKey = privateKey.generatePublicKey().encoded
        val identity = AndroidClientAccessIdentity(
            endpointId = normalizedEndpoint,
            privateKeySeed = privateKey.encoded,
            publicKey = publicKey,
            fingerprint = keyFingerprint(publicKey),
        )
        AndroidClientAccessCredential(identity, "", byteArrayOf(), byteArrayOf()).also { persist(normalizedRef, it) }
    }

    /** delete 删除本地 ClientAccessIdentity/grant 密文；它不撤销 daemon-local grant。 */
    fun delete(credentialRef: String) = synchronized(lock) {
        if (!preferences.edit().remove(preferenceKey(validateRef(credentialRef))).commit()) {
            throw ClientPlatformFailure("temporary", "failed to delete client access credential")
        }
    }

    /** deleteMany 使用一次 preferences commit 清理 Go registry 已确认不再引用的 credential refs。 */
    fun deleteMany(credentialRefs: List<String>) = synchronized(lock) {
        val normalized = credentialRefs.map(::validateRef).distinct()
        if (normalized.isEmpty()) return@synchronized
        val editor = preferences.edit()
        normalized.forEach { editor.remove(preferenceKey(it)) }
        if (!editor.commit()) {
            throw ClientPlatformFailure("temporary", "failed to delete client access credentials")
        }
    }

    fun clearAll() = synchronized(lock) {
        if (!preferences.edit().clear().commit()) {
            throw ClientPlatformFailure("temporary", "failed to clear client access credentials")
        }
        KeyStore.getInstance(KEYSTORE).apply { load(null) }.deleteEntry(KEY_ALIAS)
    }

    /** prepareRecord 为 Go Client Engine 准备 public identity projection；private seed 不离开本 secure-store owner。 */
    fun prepareRecord(credentialRef: String, endpointId: String): ClientBinding.CredentialRecord = synchronized(lock) {
        val normalizedRef = validateRef(credentialRef)
        val newlyCreated = readCredential(normalizedRef, requireGrant = false) == null
        loadOrCreateIdentity(normalizedRef, endpointId).toPlatformRecord(normalizedRef, newlyCreated)
    }

    /** resolveRecord 返回 public identity 与 bound grant；private seed 不进入 JNI/Go。 */
    fun resolveRecord(credentialRef: String, endpointId: String): ClientBinding.CredentialRecord = synchronized(lock) {
        val credential = readCredential(validateRef(credentialRef), requireGrant = true)
            ?: throw ClientPlatformFailure("unauthenticated", "client access credential is missing")
        if (credential.identity.endpointId != endpointId.trim()) {
            throw ClientPlatformFailure("identity_conflict", "credential ref belongs to another endpoint")
        }
        credential.toPlatformRecord(credentialRef)
    }

    /** bindRecord 只持久化 Go PairingExchange 已验签的 grant；Kotlin 不解析或复制 remote-auth 规则。 */
    fun bindRecord(credentialRef: String, endpointId: String, capabilityGrant: String, cloudRouteGrant: ByteArray, cloudEdgeLocator: ByteArray): ClientBinding.CredentialRecord = synchronized(lock) {
        val normalizedRef = validateRef(credentialRef)
        val normalizedEndpoint = endpointId.trim()
        val grant = capabilityGrant.trim()
        if (normalizedEndpoint.isBlank() || grant.isBlank()) {
            throw ClientPlatformFailure("protocol", "credential bind request is incomplete")
        }
        val current = readCredential(normalizedRef, requireGrant = false)
            ?: throw ClientPlatformFailure("unauthenticated", "client access identity is missing")
        if (current.identity.endpointId != normalizedEndpoint) {
            throw ClientPlatformFailure("identity_conflict", "credential ref belongs to another endpoint")
        }
        current.copy(
            capabilityGrant = grant,
            cloudRouteGrant = cloudRouteGrant.copyOf(),
            cloudEdgeLocator = cloudEdgeLocator.copyOf(),
        ).also { persist(normalizedRef, it) }.toPlatformRecord(normalizedRef)
    }

    /** sign 只对 Go remote-auth 提供的 canonical bytes 签名；调用方只能按 credential ref 访问，不取得 key seed。 */
    fun sign(credentialRef: String, payload: ByteArray): ByteArray = synchronized(lock) {
        val credential = readCredential(validateRef(credentialRef), requireGrant = false)
            ?: throw ClientPlatformFailure("unauthenticated", "client access credential is missing")
        credential.identity.sign(payload)
    }

    private fun persist(credentialRef: String, credential: AndroidClientAccessCredential) {
        validateCredential(credential)
        val payload = JSONObject()
            .put("version", CREDENTIAL_VERSION)
            .put("endpoint_id", credential.identity.endpointId)
            .put("private_key_seed", encodeUrl(credential.identity.privateKeySeed))
            .put("capability_grant", credential.capabilityGrant.trim())
            .put("cloud_route_grant", encodeUrl(credential.cloudRouteGrant))
            .put("cloud_edge_locator", encodeUrl(credential.cloudEdgeLocator))
            .toString()
            .toByteArray(Charsets.UTF_8)
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, secretKey())
        cipher.updateAAD(credentialRef.toByteArray(Charsets.UTF_8))
        val ciphertext = cipher.doFinal(payload)
        val encoded = Base64.encodeToString(cipher.iv, Base64.NO_WRAP) + "." +
            Base64.encodeToString(ciphertext, Base64.NO_WRAP)
        if (!preferences.edit().putString(preferenceKey(credentialRef), encoded).commit()) {
            throw ClientPlatformFailure("temporary", "failed to persist client access credential")
        }
    }

    private fun readCredential(credentialRef: String, requireGrant: Boolean): AndroidClientAccessCredential? {
        val encoded = preferences.getString(preferenceKey(credentialRef), null) ?: return null
        val parts = encoded.split('.', limit = 2)
        if (parts.size != 2) throw ClientPlatformFailure("unauthenticated", "client access credential ciphertext is malformed")
        return try {
            val iv = Base64.decode(parts[0], Base64.NO_WRAP)
            val ciphertext = Base64.decode(parts[1], Base64.NO_WRAP)
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, secretKey(), GCMParameterSpec(128, iv))
            cipher.updateAAD(credentialRef.toByteArray(Charsets.UTF_8))
            val value = JSONObject(String(cipher.doFinal(ciphertext), Charsets.UTF_8))
            if (value.keys().asSequence().toSet() != setOf("version", "endpoint_id", "private_key_seed", "capability_grant", "cloud_route_grant", "cloud_edge_locator") ||
                value.getInt("version") != CREDENTIAL_VERSION) {
                throw ClientPlatformFailure("unauthenticated", "client access credential schema is invalid")
            }
            val endpointId = value.getString("endpoint_id").trim()
            val seed = decodeUrl(value.getString("private_key_seed"))
            val privateKey = Ed25519PrivateKeyParameters(seed, 0)
            val publicKey = privateKey.generatePublicKey().encoded
            val credential = AndroidClientAccessCredential(
                identity = AndroidClientAccessIdentity(endpointId, seed, publicKey, keyFingerprint(publicKey)),
                capabilityGrant = value.getString("capability_grant").trim(),
                cloudRouteGrant = decodeUrl(value.getString("cloud_route_grant")),
                cloudEdgeLocator = decodeUrl(value.getString("cloud_edge_locator")),
            )
            validateCredential(credential)
            if (requireGrant && !credential.ready()) {
                throw ClientPlatformFailure("unauthenticated", "client access credential is awaiting pairing")
            }
            credential
        } catch (failure: ClientPlatformFailure) {
            throw failure
        } catch (_: Exception) {
            throw ClientPlatformFailure("unauthenticated", "client access credential could not be decrypted")
        }
    }

    private fun validateCredential(credential: AndroidClientAccessCredential) {
        if (credential.identity.endpointId.isBlank() || credential.identity.privateKeySeed.size != 32 || credential.identity.publicKey.size != 32 ||
            credential.identity.fingerprint != keyFingerprint(credential.identity.publicKey)) {
            throw ClientPlatformFailure("unauthenticated", "ClientAccessIdentity is invalid")
        }
    }

    private fun AndroidClientAccessCredential.toPlatformRecord(credentialRef: String, newlyCreated: Boolean = false): ClientBinding.CredentialRecord =
        ClientBinding.CredentialRecord.newBuilder()
            .setEndpointId(identity.endpointId)
            .setCredentialRef(credentialRef)
            .setPublicKey(com.google.protobuf.ByteString.copyFrom(identity.publicKey))
            .setKeyFingerprint(identity.fingerprint)
            .setCapabilityGrant(capabilityGrant)
            .setCloudRouteGrant(com.google.protobuf.ByteString.copyFrom(cloudRouteGrant))
            .setCloudEdgeLocator(com.google.protobuf.ByteString.copyFrom(cloudEdgeLocator))
            .setNewlyCreated(newlyCreated)
            .build()

    private fun secretKey(): SecretKey {
        val keyStore = KeyStore.getInstance(KEYSTORE).apply { load(null) }
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, KEYSTORE)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            ).setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setRandomizedEncryptionRequired(true)
                .build(),
        )
        return generator.generateKey()
    }

    private fun validateRef(value: String): String {
        val normalized = value.trim()
        if (!CREDENTIAL_REF.matches(normalized)) throw ClientPlatformFailure("unauthenticated", "credential ref is invalid")
        return normalized
    }

    private fun preferenceKey(credentialRef: String): String = "access." + encodeUrl(credentialRef.toByteArray(Charsets.UTF_8))

    private fun encodeUrl(value: ByteArray): String = Base64.encodeToString(value, Base64.NO_WRAP or Base64.URL_SAFE or Base64.NO_PADDING)
    private fun decodeUrl(value: String): ByteArray = Base64.decode(value, Base64.NO_WRAP or Base64.URL_SAFE or Base64.NO_PADDING)

    private fun keyFingerprint(publicKey: ByteArray): String =
        "ed25519-sha256:" + encodeUrl(MessageDigest.getInstance("SHA-256").digest(publicKey))

    companion object {
        private const val CREDENTIAL_VERSION = 3
        private const val KEYSTORE = "AndroidKeyStore"
        private const val KEY_ALIAS = "anytty.client-access.credentials.v1"
        private const val PREFERENCES_NAME = "anytty_client_access_credentials_v1"
        private const val TRANSFORMATION = "AES/GCM/NoPadding"
        private val CREDENTIAL_REF = Regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
    }
}
