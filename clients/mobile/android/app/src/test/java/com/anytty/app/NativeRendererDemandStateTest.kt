package com.anytty.app

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class NativeRendererDemandStateTest {
    @Test
    fun newAttachmentClearsStaleDemandAndFencesThePreviousRenderer() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        val rendererA = state.attachRenderer()
        val activeA = state.replaceDemand(
            rendererA.attachmentId,
            rendererA.demandRevision,
            setOf("machine-a"),
        )

        val rendererB = state.attachRenderer()

        assertEquals(activeA.demandRevision + 1, rendererB.demandRevision)
        assertEquals(emptySet<String>(), rendererB.endpointIds)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(
                rendererA.attachmentId,
                activeA.demandRevision,
                emptySet(),
            )
        }
        state.detachRenderer(rendererA.attachmentId)
        assertEquals(rendererB, state.currentSnapshot(rendererB.attachmentId))
    }

    @Test
    fun replacementRequiresTheExactBaseRevisionAndLeavesStateUntouchedOnFailure() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        val active = state.replaceDemand(
            attached.attachmentId,
            attached.demandRevision,
            setOf("machine-a", "machine-b"),
        )

        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(attached.attachmentId, attached.demandRevision, emptySet())
        }

        assertEquals(active, state.currentSnapshot(attached.attachmentId))
    }

    @Test
    fun userStopClearsDemandAndFencesAlreadyQueuedReplacements() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        val active = state.replaceDemand(
            attached.attachmentId,
            attached.demandRevision,
            setOf("machine-a"),
        )

        val stopped = state.clearDemand()

        assertEquals(emptySet<String>(), stopped?.endpointIds)
        assertEquals(active.demandRevision + 1, stopped?.demandRevision)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(attached.attachmentId, active.demandRevision, setOf("machine-a"))
        }
    }

    @Test
    fun repeatedRendererReplacementRetainsOnlyTheCurrentAttachment() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        var current = state.attachRenderer()

        repeat(500) {
            val stale = current
            current = state.attachRenderer()
            assertThrows(IllegalStateException::class.java) {
                state.currentSnapshot(stale.attachmentId)
            }
        }

        assertEquals("renderer-501", current.attachmentId)
        assertEquals(current, state.currentSnapshot(current.attachmentId))
    }
}
