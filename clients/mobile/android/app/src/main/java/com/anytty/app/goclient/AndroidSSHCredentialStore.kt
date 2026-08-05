package com.anytty.app.goclient

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import com.google.protobuf.ByteString
import anytty.client.binding.v1.ClientBinding
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.MessageDigest
import java.security.Signature
import java.security.spec.ECGenParameterSpec

/**
 * AndroidSSHCredentialStore 是 Android Keystore 中不可导出 SSH signer 的唯一平台 owner。
 * Go 只通过 credential ref 查询公钥和请求签名；Endpoint/Route、SSH handshake 和重连状态不进入 Kotlin。
 */
class AndroidSSHCredentialStore {
    private val lock = Any()

    /** lookup 返回公开 PKIX key；只有显式 provision 请求可以创建新 signer。 */
    fun lookup(credentialRef: String, createIfMissing: Boolean): ClientBinding.SSHCredentialRecord = synchronized(lock) {
        val ref = validateRef(credentialRef)
        val alias = alias(ref)
        var publicKey = keyStore().getCertificate(alias)?.publicKey
        val newlyCreated = publicKey == null && createIfMissing
        if (newlyCreated) {
            val generator = KeyPairGenerator.getInstance(KeyProperties.KEY_ALGORITHM_EC, KEYSTORE)
            generator.initialize(
                KeyGenParameterSpec.Builder(alias, KeyProperties.PURPOSE_SIGN)
                    .setAlgorithmParameterSpec(ECGenParameterSpec("secp256r1"))
                    .setDigests(KeyProperties.DIGEST_NONE, KeyProperties.DIGEST_SHA256)
                    .setUserAuthenticationRequired(false)
                    .build(),
            )
            publicKey = generator.generateKeyPair().public
        }
        val encoded = publicKey?.encoded
            ?: throw ClientPlatformFailure("unauthenticated", "SSH credential is missing")
        ClientBinding.SSHCredentialRecord.newBuilder()
            .setCredentialRef(ref)
            .setPublicKeyPkix(ByteString.copyFrom(encoded))
            .setNewlyCreated(newlyCreated)
            .build()
    }

    /** sign 对 Go SSH 已计算的 SHA-256 digest 执行 ECDSA 签名，私钥 handle 不离开 Keystore。 */
    fun sign(credentialRef: String, digest: ByteArray, hash: String): ByteArray = synchronized(lock) {
        if (hash != "SHA-256" || digest.size != 32) {
            throw ClientPlatformFailure("protocol", "SSH signer only accepts SHA-256 digests")
        }
        val ref = validateRef(credentialRef)
        val privateKey = keyStore().getKey(alias(ref), null)
            ?: throw ClientPlatformFailure("unauthenticated", "SSH credential is missing")
        Signature.getInstance("NONEwithECDSA").run {
            initSign(privateKey as java.security.PrivateKey)
            update(digest)
            sign()
        }
    }

    /** delete 删除 credential ref 对应的 Keystore alias；不存在时保持幂等。 */
    fun delete(credentialRef: String) = synchronized(lock) {
        val ref = validateRef(credentialRef)
        keyStore().deleteEntry(alias(ref))
    }

    /** deleteMany 只处理 Go registry 已确认不再引用的 SSH credential refs。 */
    fun deleteMany(credentialRefs: List<String>) = synchronized(lock) {
        val store = keyStore()
        credentialRefs.map(::validateRef).distinct().forEach { store.deleteEntry(alias(it)) }
    }

    fun clearAll() = synchronized(lock) {
        val store = keyStore()
        store.aliases().toList().filter { it.startsWith(ALIAS_PREFIX) }.forEach(store::deleteEntry)
    }

    private fun keyStore(): KeyStore = KeyStore.getInstance(KEYSTORE).apply { load(null) }

    private fun validateRef(value: String): String {
        val normalized = value.trim()
        if (!normalized.startsWith(REF_PREFIX) || !REF_PATTERN.matches(normalized)) {
            throw ClientPlatformFailure("protocol", "SSH credential ref is invalid")
        }
        return normalized
    }

    private fun alias(credentialRef: String): String {
        val digest = MessageDigest.getInstance("SHA-256").digest(credentialRef.toByteArray(Charsets.UTF_8))
        return ALIAS_PREFIX + Base64.encodeToString(digest, Base64.NO_WRAP or Base64.URL_SAFE or Base64.NO_PADDING)
    }

    companion object {
        const val REF_PREFIX = "ssh-platform-"
        private const val KEYSTORE = "AndroidKeyStore"
        private const val ALIAS_PREFIX = "anytty.ssh.v1."
        private val REF_PATTERN = Regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
    }
}
