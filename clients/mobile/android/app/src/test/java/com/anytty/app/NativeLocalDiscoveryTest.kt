package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.After
import org.junit.Test

class NativeLocalDiscoveryTest {
	@After
	fun clearDiscoveryCache() {
		NativeLocalDiscoveryCache.clear()
	}

    @Test
    fun discoveryKeyMatchesDaemonContractAndTrimsIdentity() {
        assertEquals(
            "f53482982aaa564cffa0b7a6964bf7694745e2f379d6a85f871b60d2585dde8f",
            nativeLocalDiscoveryKey(" device-1234567890 ", " ed25519-sha256:test "),
        )
    }

    @Test
	fun multicastLockIsOnlyRequiredBeforeAndroid13Extension7() {
        assertEquals(true, nativeLocalDiscoveryNeedsMulticastLock(24, 0))
        assertEquals(true, nativeLocalDiscoveryNeedsMulticastLock(33, 6))
        assertEquals(false, nativeLocalDiscoveryNeedsMulticastLock(33, 7))
        assertEquals(false, nativeLocalDiscoveryNeedsMulticastLock(34, 0))
		assertEquals(false, nativeLocalDiscoverySupportsServiceInfoCallback(24, 0))
		assertEquals(false, nativeLocalDiscoverySupportsServiceInfoCallback(33, 6))
		assertEquals(true, nativeLocalDiscoverySupportsServiceInfoCallback(33, 7))
		assertEquals(true, nativeLocalDiscoverySupportsServiceInfoCallback(34, 0))
	}

	@Test
	fun discoveredServiceLeaseRenewsUntilServiceIsLost() {
		val deviceID = "device-1234567890"
		val fingerprint = "ed25519-sha256:test"
		NativeLocalDiscoveryCache.update("service", listOf(NativeLocalDiscoveryCandidate(
			discoveryKey = nativeLocalDiscoveryKey(deviceID, fingerprint),
			address = "192.168.1.8",
			port = 41120,
			protocolVersion = 7,
			expiresAtUnixNano = 1L,
			networkHandle = 42L,
		)))

		val renewed = NativeLocalDiscoveryCache.snapshot(deviceID, fingerprint)

		assertEquals(1, renewed.size)
		assertTrue(renewed.single().expiresAtUnixNano > System.currentTimeMillis() * 1_000_000L)
		NativeLocalDiscoveryCache.remove("service")
		assertTrue(NativeLocalDiscoveryCache.snapshot(deviceID, fingerprint).isEmpty())
	}
}
