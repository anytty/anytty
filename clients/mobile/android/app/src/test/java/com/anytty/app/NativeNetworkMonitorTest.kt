package com.anytty.app

import org.junit.Assert.assertFalse
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class NativeNetworkMonitorTest {
    @Test
    fun `same default network reports connectivity and address changes`() {
        val previous = NativeNetworkMonitor.Signature(
            networkHandle = 42L,
            internet = true,
            validated = true,
            addresses = listOf("192.0.2.10", "2001:db8::10"),
            dnsServers = listOf("192.0.2.53"),
            routes = listOf("0.0.0.0/0 -> 192.0.2.1"),
        )
        val current = NativeNetworkMonitor.Signature(
            networkHandle = 42L,
            internet = true,
            validated = false,
            addresses = listOf("192.0.2.10", "2001:db8::11"),
            dnsServers = listOf("192.0.2.54"),
            routes = listOf("0.0.0.0/0 -> 192.0.2.1"),
        )

        assertTrue(requiresSessionRecovery(previous, current))

        assertTrue(requiresSessionRecovery(
            previous,
            previous.copy(addresses = listOf("192.0.2.11", "2001:db8::11")),
        ))
        assertTrue(requiresSessionRecovery(previous, previous.copy(dnsServers = listOf("198.51.100.53"))))
        assertTrue(requiresSessionRecovery(previous, previous.copy(routes = listOf("0.0.0.0/0 -> 192.0.2.254"))))
        assertEquals("path_changed", networkChangeReason(previous, current))
    }

    @Test
    fun `default network replacement and loss require session recovery`() {
        val wifi = signature(42L, validated = true, address = "192.0.2.10")
        val cellular = signature(77L, validated = true, address = "198.51.100.10")
        val offline = signature(0L, internet = false)

        assertTrue(requiresSessionRecovery(wifi, cellular))
        assertTrue(requiresSessionRecovery(cellular, offline))
        assertTrue(requiresSessionRecovery(offline, wifi))
        assertEquals("network_replaced", networkChangeReason(wifi, cellular))
        assertEquals("offline", networkChangeReason(cellular, offline))
        assertEquals("available", networkChangeReason(offline, wifi))
    }

    @Test
    fun `a replacement network publishes once while validation settles`() {
        val wifi = signature(42L, validated = true, address = "192.0.2.10")
        val settlingCellular = signature(77L, address = "198.51.100.10")
        val readyCellular = settlingCellular.copy(validated = true)

        assertTrue(requiresSessionRecovery(wifi, settlingCellular))
        assertFalse(requiresSessionRecovery(settlingCellular, readyCellular))
        assertFalse(requiresSessionRecovery(readyCellular, settlingCellular))
    }

    @Test
    fun `plugin payload is session scoped and preserves connectivity`() {
        val offline = nativeNetworkChangedPayload(epoch = 9L, connected = false, reason = "offline")

        assertEquals(9L, offline.getLong("epoch"))
        assertFalse(offline.getBoolean("connected"))
        assertEquals("offline", offline.getString("reason"))
        assertEquals("session", offline.getString("scope"))
    }

    @Test
    fun `network snapshot keeps the latest epoch and recovery reason`() {
        val snapshot = NativeNetworkSnapshot(
            epoch = 12L,
            connected = true,
            reason = "network_replaced",
        )
        val payload = nativeNetworkChangedPayload(snapshot.epoch, snapshot.connected, snapshot.reason)

        assertEquals(12L, payload.getLong("epoch"))
        assertTrue(payload.getBoolean("connected"))
        assertEquals("network_replaced", payload.getString("reason"))
    }

    @Test
    fun `validation is a path hint and does not block local connectivity`() {
        val captive = signature(42L, address = "192.0.2.10")
        assertTrue(captive.connected)
        assertFalse(requiresSessionRecovery(captive, captive.copy(validated = true)))
    }

    private fun signature(
        handle: Long,
        internet: Boolean = true,
        validated: Boolean = false,
        address: String? = null,
    ) = NativeNetworkMonitor.Signature(
        networkHandle = handle,
        internet = internet,
        validated = validated,
        addresses = listOfNotNull(address),
        dnsServers = emptyList(),
        routes = emptyList(),
    )
}
