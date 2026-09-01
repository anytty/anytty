package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class AnyTTYConnectionServiceTest {
    @Test
    fun staleDisconnectCompletionPreservesANewerForegroundDemand() {
        val operations = mutableListOf<String>()

        val stopped = finishDisconnectAllServiceRequest(
            canStopService = { false },
            stopService = { operations += "stop" },
        )

        assertFalse(stopped)
        assertEquals(emptyList<String>(), operations)
    }

    @Test
    fun disconnectCompletionStopsTheServiceWhileDemandRemainsEmpty() {
        val operations = mutableListOf<String>()

        val stopped = finishDisconnectAllServiceRequest(
            canStopService = { true },
            stopService = { operations += "stop" },
        )

        assertTrue(stopped)
        assertEquals(listOf("stop"), operations)
    }

    @Test
    fun acceptedStopCompletionCannotRemoveFreshResumeDemandOrRuntime() {
        var runtimeGeneration = 7L
        var demand = emptySet<String>() // Stop was synchronously accepted.
        val acceptedRuntimeGeneration = runtimeGeneration
        val operations = mutableListOf<String>()

        runtimeGeneration = 8L
        demand = setOf("machine-fresh")
        val stopped = finishDisconnectAllServiceRequest(
            canStopService = {
                disconnectAllCompletionOwnsForeground(
                    acceptedRuntimeGeneration = acceptedRuntimeGeneration,
                    currentRuntimeGeneration = runtimeGeneration,
                    hasActiveEndpoints = demand.isNotEmpty(),
                )
            },
            stopService = {
                demand = emptySet()
                operations += "stop"
            },
        )

        assertFalse(stopped)
        assertEquals(setOf("machine-fresh"), demand)
        assertEquals(emptyList<String>(), operations)
    }

    @Test
    fun acceptedStopCompletionCannotRemoveANewerRuntimeBeforeDemandArrives() {
        assertFalse(disconnectAllCompletionOwnsForeground(
            acceptedRuntimeGeneration = 7L,
            currentRuntimeGeneration = 8L,
            hasActiveEndpoints = false,
        ))
    }
}
