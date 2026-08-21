package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class NativeEndpointSupervisorRolloutTest {
    @Test
    fun percentageBoundariesAreExact() {
        assertFalse(NativeEndpointSupervisorRollout.isTakeover("studio", 0))
        assertTrue(NativeEndpointSupervisorRollout.isTakeover("studio", 100))
    }

    @Test
    fun endpointBucketIsStable() {
        val first = NativeEndpointSupervisorRollout.isTakeover("studio", 37)
        repeat(500) {
            assertEquals(first, NativeEndpointSupervisorRollout.isTakeover("studio", 37))
        }
    }

    @Test(expected = IllegalArgumentException::class)
    fun rejectsEmptyEndpoint() {
        NativeEndpointSupervisorRollout.isTakeover("  ", 50)
    }
}
