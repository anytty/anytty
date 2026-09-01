package com.anytty.app

import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.security.KeyPairGenerator
import java.security.KeyStore
import java.security.MessageDigest
import java.security.PrivateKey
import java.security.Signature
import java.security.spec.ECGenParameterSpec

internal class AndroidSSHCredentialStore {
    private val lock = Any()

    fun handle(call: MethodCall, result: MethodChannel.Result) {
        try {
            when (call.method) {
                "lookup" -> result.success(
                    lookup(
                        requiredString(call, "credentialRef"),
                        call.argument<Boolean>("createIfMissing") == true,
                    ),
                )
                "sign" -> result.success(
                    sign(
                        requiredString(call, "credentialRef"),
                        call.argument<ByteArray>("digest")
                            ?: throw SSHCredentialFailure("protocol", "SSH digest is missing"),
                        requiredString(call, "hash"),
                    ),
                )
                "delete" -> {
                    delete(requiredString(call, "credentialRef"))
                    result.success(null)
                }
                else -> result.notImplemented()
            }
        } catch (failure: SSHCredentialFailure) {
            result.error(failure.code, failure.message, null)
        } catch (_: Exception) {
            result.error("temporary", "SSH secure signer operation failed", null)
        }
    }

    private fun lookup(credentialRef: String, createIfMissing: Boolean): Map<String, Any> =
        synchronized(lock) {
            val ref = validateRef(credentialRef)
            val alias = alias(ref)
            var publicKey = keyStore().getCertificate(alias)?.publicKey
            val newlyCreated = publicKey == null && createIfMissing
            if (newlyCreated) {
                val generator = KeyPairGenerator.getInstance(
                    KeyProperties.KEY_ALGORITHM_EC,
                    KEYSTORE,
                )
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
                ?: throw SSHCredentialFailure("unauthenticated", "SSH credential is missing")
            mapOf(
                "credentialRef" to ref,
                "publicKeyPkix" to encoded,
                "newlyCreated" to newlyCreated,
            )
        }

    private fun sign(credentialRef: String, digest: ByteArray, hash: String): ByteArray =
        synchronized(lock) {
            if (hash != "SHA-256" || digest.size != 32) {
                throw SSHCredentialFailure(
                    "protocol",
                    "SSH signer only accepts SHA-256 digests",
                )
            }
            val ref = validateRef(credentialRef)
            val privateKey = keyStore().getKey(alias(ref), null) as? PrivateKey
                ?: throw SSHCredentialFailure("unauthenticated", "SSH credential is missing")
            Signature.getInstance("NONEwithECDSA").run {
                initSign(privateKey)
                update(digest)
                sign()
            }
        }

    private fun delete(credentialRef: String) = synchronized(lock) {
        keyStore().deleteEntry(alias(validateRef(credentialRef)))
    }

    private fun keyStore(): KeyStore = KeyStore.getInstance(KEYSTORE).apply { load(null) }

    private fun validateRef(value: String): String {
        val normalized = value.trim()
        if (!normalized.startsWith(REF_PREFIX) || !REF_PATTERN.matches(normalized)) {
            throw SSHCredentialFailure("protocol", "SSH credential ref is invalid")
        }
        return normalized
    }

    private fun alias(credentialRef: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
            .digest(credentialRef.toByteArray(Charsets.UTF_8))
        return ALIAS_PREFIX + Base64.encodeToString(
            digest,
            Base64.NO_WRAP or Base64.URL_SAFE or Base64.NO_PADDING,
        )
    }

    private fun requiredString(call: MethodCall, name: String): String =
        call.argument<String>(name)?.takeIf { it.isNotBlank() }
            ?: throw SSHCredentialFailure("protocol", "SSH credential request is incomplete")

    private class SSHCredentialFailure(val code: String, override val message: String) :
        Exception(message)

    companion object {
        private const val REF_PREFIX = "ssh-platform-"
        private const val KEYSTORE = "AndroidKeyStore"
        private const val ALIAS_PREFIX = "anytty.ssh.v1."
        private val REF_PATTERN = Regex("^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$")
    }
}
