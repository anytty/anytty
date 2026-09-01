package com.anytty.app

import org.junit.Assert.assertFalse
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertSame
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
    fun `handle only replacement verifies the existing path before reconnecting`() {
        val vpnBefore = signature(42L, validated = true, address = "192.0.2.10")
        val vpnAfter = vpnBefore.copy(networkHandle = 77L)

        assertTrue(requiresSessionRecovery(vpnBefore, vpnAfter))
        assertEquals("path_changed", networkChangeReason(vpnBefore, vpnAfter))
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
    fun `transport change on the same vpn requires session recovery`() {
        val vpnOnWifi = signature(
            handle = 42L,
            validated = true,
            transports = listOf(1, 4),
        )
        val vpnOnCellular = vpnOnWifi.copy(transports = listOf(0, 4))

        assertTrue(requiresSessionRecovery(vpnOnWifi, vpnOnCellular))
        assertEquals("network_replaced", networkChangeReason(vpnOnWifi, vpnOnCellular))
    }

    @Test
    fun `cellular bearer replacement under the same vpn advances exactly once`() {
        val vpn = signature(
            handle = 42L,
            validated = true,
            address = "10.8.0.2",
            transports = listOf(0, 4),
            underlyingNetworks = listOf(underlying(handle = 100L, transports = listOf(0))),
        )
        val replacement = vpn.copy(
            underlyingNetworks = listOf(underlying(handle = 101L, transports = listOf(0))),
        )
        val state = NativeStableNetworkState(vpn, initialEpoch = 11L)

        val changed = state.observe(replacement)

        assertNotNull(changed)
        assertEquals(12L, changed?.epoch)
        assertEquals("network_replaced", changed?.reason)
        repeat(20) {
            assertNull(state.observe(replacement.copy()))
        }
        assertEquals(12L, state.currentEpoch())
    }

    @Test
    fun `ambiguous vpn fallback retains one bearer without fingerprinting every network`() {
        val cellular = underlying(handle = 100L, transports = listOf(0))
        val wifi = underlying(handle = 200L, transports = listOf(1))
        val candidates = listOf(
            NativeUnderlyingNetworkCandidate(cellular, validated = true),
            NativeUnderlyingNetworkCandidate(wifi, validated = true),
        )

        assertEquals(listOf(cellular), selectUnderlyingNetworks(candidates, preferredHandle = 100L))
        assertEquals(emptyList<NativeNetworkMonitor.UnderlyingNetworkIdentity>(), selectUnderlyingNetworks(candidates, null))
        assertEquals(
            listOf(wifi),
            selectUnderlyingNetworks(candidates.drop(1), preferredHandle = 100L),
        )
    }

    @Test
    fun `a unique validated bearer replaces an unvalidated preferred bearer exactly once`() {
        val stale = underlying(handle = 100L, transports = listOf(0))
        val replacement = underlying(handle = 101L, transports = listOf(0))
        val candidates = listOf(
            NativeUnderlyingNetworkCandidate(stale, validated = false),
            NativeUnderlyingNetworkCandidate(replacement, validated = true),
        )

        val selected = selectUnderlyingNetwork(candidates, preferredHandle = stale.networkHandle)
        assertEquals(listOf(replacement), selected.identities)
        assertNull(selected.ambiguousCandidates)

        val before = signature(
            handle = 42L,
            validated = true,
            address = "10.8.0.2",
            transports = listOf(4),
            underlyingNetworks = listOf(stale),
        )
        val after = before.copy(underlyingNetworks = selected.identities)
        val state = NativeStableNetworkState(before, initialEpoch = 21L)
        assertEquals(22L, state.observe(after)?.epoch)
        repeat(20) { assertNull(state.observe(after.copy())) }
        assertEquals(22L, state.currentEpoch())
    }

    @Test
    fun `ambiguous bearer preference expires after bounded resampling`() {
        val cellular = NativeUnderlyingNetworkCandidate(
            underlying(handle = 100L, transports = listOf(0)),
            validated = true,
        )
        val wifi = NativeUnderlyingNetworkCandidate(
            underlying(handle = 200L, transports = listOf(1)),
            validated = true,
        )
        val ambiguity = listOf(cellular, wifi)
        val state = NativeBoundedNetworkResampleState(longArrayOf(10L, 20L))

        assertEquals(NativeBoundedNetworkResampleDecision(true, 10L), state.observe(ambiguity))
        assertEquals(NativeBoundedNetworkResampleDecision(true, 20L), state.observe(ambiguity))
        assertEquals(NativeBoundedNetworkResampleDecision(false, null), state.observe(ambiguity))
        assertEquals(NativeBoundedNetworkResampleDecision(false, null), state.observe(ambiguity))

        assertEquals(NativeBoundedNetworkResampleDecision(true, null), state.observe(null))
        assertEquals(NativeBoundedNetworkResampleDecision(true, 10L), state.observe(ambiguity))
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

    @Test
    fun `cached offline state publishes available when foreground resamples active network`() {
        val offline = signature(0L, internet = false)
        val wifi = signature(42L, validated = true, address = "192.0.2.10")
        val state = NativeStableNetworkState(offline)

        val recovered = state.observe(wifi)

        assertNotNull(recovered)
        assertEquals(1L, recovered?.epoch)
        assertEquals(true, recovered?.current?.connected)
        assertEquals("available", recovered?.reason)
        assertEquals(wifi, state.currentSignature())
    }

    @Test
    fun `repeated callbacks for the same stable path do not advance the epoch`() {
        val wifi = signature(
            handle = 42L,
            validated = true,
            address = "192.0.2.10",
            transports = listOf(1),
        )
        val state = NativeStableNetworkState(wifi, initialEpoch = 7L)

        assertNull(state.observe(wifi))
        assertNull(state.observe(wifi.copy()))
        assertEquals(7L, state.currentEpoch())

        val changed = state.observe(wifi.copy(addresses = listOf("192.0.2.11")))
        assertEquals(8L, changed?.epoch)
        assertNull(state.observe(wifi.copy(addresses = listOf("192.0.2.11"))))
        assertEquals(8L, state.currentEpoch())
    }

    @Test
    fun `process monitor remains active when renderer detaches with retained demand`() {
        val monitorSlot = NativeProcessNetworkMonitorSlot<FakeMonitor>()
        var starts = 0
        val monitor = monitorSlot.ensure(create = {
            starts += 1
            FakeMonitor()
        })
        val demand = NativeRendererDemandState { "renderer-a" }
        val renderer = demand.attachRenderer()
        demand.replaceDemand(renderer.attachmentId, renderer.demandRevision, setOf("machine-a"))

        demand.detachRenderer(renderer.attachmentId)
        val retained = monitorSlot.ensure(create = {
            starts += 1
            FakeMonitor()
        })

        assertTrue(demand.hasDemand())
        assertSame(monitor, retained)
        assertEquals(1, starts)
        assertFalse(monitor.closed)

        monitorSlot.close()
        assertTrue(monitor.closed)
    }

    private fun signature(
        handle: Long,
        internet: Boolean = true,
        validated: Boolean = false,
        address: String? = null,
        transports: List<Int> = emptyList(),
        underlyingNetworks: List<NativeNetworkMonitor.UnderlyingNetworkIdentity> = emptyList(),
    ) = NativeNetworkMonitor.Signature(
        networkHandle = handle,
        internet = internet,
        validated = validated,
        addresses = listOfNotNull(address),
        dnsServers = emptyList(),
        routes = emptyList(),
        transports = transports,
        underlyingNetworks = underlyingNetworks,
    )

    private fun underlying(
        handle: Long,
        transports: List<Int>,
    ) = NativeNetworkMonitor.UnderlyingNetworkIdentity(handle, transports)

    private class FakeMonitor : AutoCloseable {
        var closed = false

        override fun close() {
            closed = true
        }
    }
}
