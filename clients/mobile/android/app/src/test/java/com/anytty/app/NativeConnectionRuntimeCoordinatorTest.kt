package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class NativeConnectionRuntimeCoordinatorTest {
    @Test
    fun loadReusesAnAlreadyRunningProcessRuntime() {
        val harness = Harness()
        harness.started = true

        harness.coordinator.load()

        assertEquals(emptyList<String>(), harness.operations)
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
