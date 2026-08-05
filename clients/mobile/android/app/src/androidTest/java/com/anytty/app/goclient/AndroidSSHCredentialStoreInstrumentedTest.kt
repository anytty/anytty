package com.anytty.app.goclient

import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertThrows
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import java.security.KeyFactory
import java.security.MessageDigest
import java.security.Signature
import java.security.spec.X509EncodedKeySpec

/** AndroidSSHCredentialStoreInstrumentedTest 在真实 Android Keystore provider 上验证 SSH signer primitive。 */
@RunWith(AndroidJUnit4::class)
class AndroidSSHCredentialStoreInstrumentedTest {
    @Test
    fun createsReusesSignsAndDeletesNonExportableSigner() {
        val store = AndroidSSHCredentialStore()
        val ref = AndroidSSHCredentialStore.REF_PREFIX + "instrumented"
        store.delete(ref)

        val created = store.lookup(ref, true)
        assertTrue(created.newlyCreated)
        assertTrue(created.publicKeyPkix.size() > 0)
        val reused = store.lookup(ref, false)
        assertFalse(reused.newlyCreated)
        assertArrayEquals(created.publicKeyPkix.toByteArray(), reused.publicKeyPkix.toByteArray())

        val digest = MessageDigest.getInstance("SHA-256").digest("anytty-ssh-keystore".toByteArray())
        val signature = store.sign(ref, digest, "SHA-256")
        val publicKey = KeyFactory.getInstance("EC").generatePublic(X509EncodedKeySpec(created.publicKeyPkix.toByteArray()))
        assertTrue(Signature.getInstance("NONEwithECDSA").run {
            initVerify(publicKey)
            update(digest)
            verify(signature)
        })

        store.delete(ref)
        assertThrows(ClientPlatformFailure::class.java) { store.lookup(ref, false) }
    }
}
