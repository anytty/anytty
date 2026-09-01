package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class NativeConnectionRuntimeCoordinatorTest {
    @Test
    fun userStopInvalidatesOldRebuildWithoutConsumingANewerSchedule() {
        val recovery = NativeRuntimeRecoveryState(longArrayOf(10L, 20L))
        val stoppedRuntimeTicket = requireNotNull(recovery.scheduleRebuild())

        recovery.cancelRebuilds()
        val nextRuntimeTicket = requireNotNull(recovery.scheduleRebuild())

        assertEquals(false, recovery.consumeRebuild(stoppedRuntimeTicket.token))
        assertEquals(true, recovery.consumeRebuild(nextRuntimeTicket.token))
    }

    @Test
    fun immediateRuntimeFatalsKeepAdvancingBackoffUntilTheStableWindowCompletes() {
        val recovery = NativeRuntimeRecoveryState(
            rebuildBackoffMillis = longArrayOf(1L, 2L, 4L, 8L, 15L),
            runtimeStableWindowMillis = 30L,
        )

        listOf(1L, 2L, 4L, 8L, 15L, 15L).forEach { expectedDelay ->
            val generation = recovery.beginRuntimeAttempt()
            val health = recovery.scheduleRuntimeHealth(generation)
            val rebuild = requireNotNull(recovery.scheduleRebuild())

            assertEquals(expectedDelay, rebuild.delayMillis)
            assertEquals(false, recovery.consumeRuntimeHealth(health))
            assertEquals(true, recovery.consumeRebuild(rebuild.token))
        }

        val stableGeneration = recovery.beginRuntimeAttempt()
        val stableHealth = recovery.scheduleRuntimeHealth(stableGeneration)
        assertEquals(30L, stableHealth.delayMillis)
        assertEquals(true, recovery.consumeRuntimeHealth(stableHealth))
        assertEquals(1L, requireNotNull(recovery.scheduleRebuild()).delayMillis)
    }

    @Test
    fun userStopCancelsBothRuntimeRebuildAndStableHealthTickets() {
        val recovery = NativeRuntimeRecoveryState(longArrayOf(10L, 20L), runtimeStableWindowMillis = 30L)
        val generation = recovery.beginRuntimeAttempt()
        val health = recovery.scheduleRuntimeHealth(generation)
        val first = requireNotNull(recovery.scheduleRebuild())

        recovery.cancelRebuilds()

        assertEquals(false, recovery.consumeRuntimeHealth(health))
        assertEquals(false, recovery.consumeRebuild(first.token))
        assertEquals(10L, requireNotNull(recovery.scheduleRebuild()).delayMillis)
    }

    @Test
    fun failedRuntimeAttemptsStillAdvanceTheCallbackFence() {
        val recovery = NativeRuntimeRecoveryState(longArrayOf(0L))

        val failedAttempt = recovery.beginRuntimeAttempt()
        val healthyAttempt = recovery.beginRuntimeAttempt()

        assertEquals(1L, failedAttempt)
        assertEquals(2L, healthyAttempt)
        assertEquals(healthyAttempt, recovery.runtimeGeneration)
    }

    @Test
    fun oneForegroundGenerationRefreshesOnlyOnceAcrossActivityAndPluginCallbacks() {
        val foreground = NativeForegroundRefreshState()

        foreground.markForeground()
        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 1L)) // Activity.onStart
        foreground.completeRefresh(runtimeGeneration = 1L)
        foreground.markForeground()
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L)) // Activity.onResume
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L)) // plugin ack

        foreground.markBackground()
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L)) // delayed plugin ack
        foreground.markForeground()
        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 1L))
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L))
        foreground.completeRefresh(runtimeGeneration = 1L)
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L))
    }

    @Test
    fun foregroundGenerationRefreshesAgainWhenTheProcessRuntimeChanges() {
        val foreground = NativeForegroundRefreshState()

        foreground.markForeground()
        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 1L))
        foreground.completeRefresh(runtimeGeneration = 1L)
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L))
        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 2L))
        foreground.completeRefresh(runtimeGeneration = 2L)
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 2L))
    }

    @Test
    fun failedForegroundRefreshCanRetryInTheSameForegroundGeneration() {
        val foreground = NativeForegroundRefreshState()
        foreground.markForeground()

        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 1L))
        foreground.failRefresh(runtimeGeneration = 1L)
        assertEquals(true, foreground.beginRefresh(runtimeGeneration = 1L))
        foreground.completeRefresh(runtimeGeneration = 1L)
        assertEquals(false, foreground.beginRefresh(runtimeGeneration = 1L))
    }

    @Test
    fun loadReusesAnAlreadyRunningProcessRuntime() {
        val harness = Harness()
        harness.started = true

        harness.coordinator.load()

        assertEquals(emptyList<String>(), harness.operations)
    }

    @Test
    fun transientInitialStartFailureKeepsThePluginRuntimeCallable() {
        var starts = 0
        var started = false
        val coordinator = NativeConnectionRuntimeCoordinator(
            startRuntime = {
                starts += 1
                if (starts == 1) throw IllegalStateException("bridge start failed")
                started = true
            },
            isRuntimeStarted = { started },
            resetRuntime = {},
        )

        assertEquals("bridge start failed", coordinator.load()?.message)
        assertEquals(true, coordinator.isReady())

        coordinator.ensureForForeground()
        assertEquals(true, started)
        assertEquals(2, starts)
    }

    @Test
    fun bridgeEndpointLookupRetriesATransientInitialStartFailure() {
        var starts = 0
        var started = false
        val coordinator = NativeConnectionRuntimeCoordinator(
            startRuntime = {
                starts += 1
                if (starts == 1) throw IllegalStateException("bridge start failed")
                started = true
            },
            isRuntimeStarted = { started },
            resetRuntime = {},
        )

        assertEquals("bridge start failed", coordinator.load()?.message)

        coordinator.ensureForBridgeEndpoint()
        assertEquals(true, started)
        assertEquals(2, starts)

        coordinator.ensureForBridgeEndpoint()
        assertEquals(2, starts)

        started = false
        coordinator.ensureForBridgeEndpoint()
        assertEquals(true, started)
        assertEquals(3, starts)
    }

    @Test
    fun foregroundResumeOnlyStartsAStoppedRuntime() {
        val harness = Harness()
        harness.coordinator.load()
        harness.operations.clear()

        harness.coordinator.ensureForForeground()
        assertEquals(emptyList<String>(), harness.operations)

        harness.started = false
        harness.coordinator.ensureForForeground()
        assertEquals(listOf("start"), harness.operations)
    }

    @Test
    fun resetAndDestroyRemainFailClosed() {
        val harness = Harness()
        harness.coordinator.load()
        harness.operations.clear()

        harness.coordinator.resetLocalPairings()
        assertEquals(listOf("reset"), harness.operations)

        harness.coordinator.destroy()

        assertThrows(IllegalStateException::class.java) { harness.coordinator.ensureForForeground() }
        assertThrows(IllegalStateException::class.java) { harness.coordinator.ensureForBridgeEndpoint() }
        assertThrows(IllegalStateException::class.java) { harness.coordinator.resetLocalPairings() }
    }

    private class Harness {
        var started = false
        val operations = mutableListOf<String>()
        val coordinator = NativeConnectionRuntimeCoordinator(
            startRuntime = {
                operations += "start"
                started = true
            },
            isRuntimeStarted = { started },
            resetRuntime = { operations += "reset" },
        )
    }
}
