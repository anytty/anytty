package com.anytty.app

import anytty.client.binding.v1.ClientBinding
import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test

class NativeRendererDemandStateTest {
    @Test
    fun disconnectAllPublishedWithoutAPluginReplaysWhenRendererAttaches() {
        val events = NativeDisconnectAllEventState()
        val stopped = demandSnapshot(demandRevision = 2L, stopEpoch = 2L, stopped = true)
        val delivered = mutableListOf<NativeRendererDemandSnapshot>()

        events.publish(stopped)
        events.observe(stopped, delivered::add)

        assertEquals(listOf(stopped), delivered)
    }

    @Test
    fun disconnectAllPublishedBetweenActivitiesReplaysOnlyToTheReplacement() {
        val events = NativeDisconnectAllEventState()
        val running = demandSnapshot(demandRevision = 1L, stopEpoch = 0L, stopped = false)
        val stopped = demandSnapshot(demandRevision = 2L, stopEpoch = 2L, stopped = true)
        val firstActivity = mutableListOf<NativeRendererDemandSnapshot>()
        val replacementActivity = mutableListOf<NativeRendererDemandSnapshot>()
        val firstSubscription = events.observe(running, firstActivity::add)

        firstSubscription.close()
        events.publish(stopped)
        events.observe(stopped, replacementActivity::add)

        assertEquals(emptyList<NativeRendererDemandSnapshot>(), firstActivity)
        assertEquals(listOf(stopped), replacementActivity)
    }

    @Test
    fun resumedDemandStillReplaysAnUnacknowledgedStopAfterRendererCrash() {
        val events = NativeDisconnectAllEventState()
        val stopped = demandSnapshot(demandRevision = 2L, stopEpoch = 2L, stopped = true)
        val resumed = demandSnapshot(demandRevision = 3L, stopEpoch = 2L, stopped = false)
        val delivered = mutableListOf<NativeRendererDemandSnapshot>()

        events.publish(stopped)
        events.observe(resumed, delivered::add).close()
        assertEquals(listOf(stopped), delivered)

        events.acknowledge(stopped.stopEpoch)
        events.observe(stopped, delivered::add).close()
        events.publish(stopped)
        events.observe(resumed, delivered::add).close()

        assertEquals(listOf(stopped), delivered)
    }

    @Test
    fun oldAcknowledgementCannotClearANewerRetainedStop() {
        val events = NativeDisconnectAllEventState()
        val firstStop = demandSnapshot(demandRevision = 2L, stopEpoch = 2L, stopped = true)
        val newerStop = demandSnapshot(demandRevision = 4L, stopEpoch = 4L, stopped = true)
        val resumed = demandSnapshot(demandRevision = 5L, stopEpoch = 4L, stopped = false)
        val delivered = mutableListOf<NativeRendererDemandSnapshot>()

        events.publish(firstStop)
        events.publish(newerStop)

        assertThrows(IllegalStateException::class.java) {
            events.acknowledge(firstStop.stopEpoch)
        }
        events.observe(resumed, delivered::add).close()
        assertEquals(listOf(newerStop), delivered)

        events.acknowledge(newerStop.stopEpoch)
        events.acknowledge(newerStop.stopEpoch)
        events.observe(resumed, delivered::add).close()
        assertEquals(listOf(newerStop), delivered)
    }

    @Test
    fun runtimeReplayAcquiresForegroundOwnershipBeforeHostAndDemand() {
        val operations = mutableListOf<String>()

        replayNativeSupervisorState(
            maintainForegroundOwnership = { operations += "foreground" },
            signalHost = { operations += "host" },
            replaceDemand = { operations += "demand" },
        )

        assertEquals(listOf("foreground", "host", "demand"), operations)
    }

    @Test
    fun newAttachmentInheritsCanonicalDemandAndFencesThePreviousRenderer() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        val rendererA = state.attachRenderer()
        val activeA = state.replaceDemand(
            rendererA.attachmentId,
            rendererA.demandRevision,
            setOf("machine-a"),
        )
        val goProjections = mutableListOf<NativeRendererDemandSnapshot>()
        val operations = mutableListOf<String>()

        val rendererB = handoffRendererDemand(
            rendererDemand = state,
            replaceGoDemand = {
                operations += "go"
                goProjections += it
            },
            ensureRuntimeForDemand = { operations += "runtime" },
            maintainForegroundService = { operations += "fgs" },
        )

        assertEquals(activeA.demandRevision, rendererB.demandRevision)
        assertEquals(setOf("machine-a"), rendererB.endpointIds)
        assertEquals(setOf("machine-a"), state.canonicalSnapshot().endpointIds)
        assertEquals(true, state.hasDemand())
        assertEquals(listOf(rendererB), goProjections)
        assertEquals(listOf("fgs", "go", "runtime"), operations)
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
    fun newAttachmentFirstFullSnapshotAtomicallyReplacesInheritedDemand() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        val rendererA = state.attachRenderer()
        val activeA = state.replaceDemand(
            rendererA.attachmentId,
            rendererA.demandRevision,
            setOf("machine-a", "machine-b"),
        )
        val rendererB = state.attachRenderer()

        val activeB = state.replaceDemand(
            rendererB.attachmentId,
            rendererB.demandRevision,
            emptySet(),
        )

        assertEquals(activeA.demandRevision + 1, activeB.demandRevision)
        assertEquals(emptySet<String>(), activeB.endpointIds)
        assertEquals(false, state.hasDemand())
    }

    @Test
    fun allCanonicalDemandUsesGoTakeoverWithoutARolloutSplit() {
        val snapshot = NativeRendererDemandSnapshot(
            attachmentId = "renderer",
            demandRevision = 7L,
            stopEpoch = 0L,
            endpointIds = linkedSetOf("machine-b", "machine-a"),
            stopped = false,
        )

        val wire = nativeSupervisorDemandSnapshot(snapshot)

        assertEquals(listOf("machine-a", "machine-b"), wire.endpointsList.map { it.endpointId })
        assertEquals(
            setOf(ClientBinding.EndpointSupervisorMode.ENDPOINT_SUPERVISOR_MODE_TAKEOVER),
            wire.endpointsList.mapTo(linkedSetOf()) { it.mode },
        )
    }

    @Test
    fun transientNativeFailureDoesNotRollBackCanonicalDesiredDemand() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        var runtimeRecoveries = 0
        var foregroundServiceMaintains = 0

        assertThrows(IllegalStateException::class.java) {
            reconcileRendererDemand(
                rendererDemand = state,
                attachmentId = attached.attachmentId,
                baseDemandRevision = attached.demandRevision,
                endpointIds = setOf("machine-a"),
                replaceGoDemand = {
                    submitNativeDemandOrRecover(
                        submit = { throw IllegalStateException("temporary ABI failure") },
                        recoverRuntime = { runtimeRecoveries += 1 },
                    )
                },
                ensureRuntimeForDemand = { error("must not ensure after Go rejects demand") },
                maintainForegroundService = { foregroundServiceMaintains += 1 },
                stopForegroundService = { error("must not stop") },
            )
        }

        val retained = state.currentSnapshot(attached.attachmentId)
        assertEquals(setOf("machine-a"), retained.endpointIds)
        assertEquals(attached.demandRevision + 1, retained.demandRevision)
        assertEquals(1, runtimeRecoveries)
        assertEquals(1, foregroundServiceMaintains)
    }

    @Test
    fun foregroundServiceFailureRetainsIntentAndRecoversBeforeGoMutation() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        val submitted = mutableListOf<NativeRendererDemandSnapshot>()
        var runtimeRecoveries = 0

        assertThrows(IllegalStateException::class.java) {
            reconcileRendererDemand(
                rendererDemand = state,
                attachmentId = attached.attachmentId,
                baseDemandRevision = attached.demandRevision,
                endpointIds = setOf("machine-a"),
                replaceGoDemand = submitted::add,
                ensureRuntimeForDemand = { error("must not start without FGS ownership") },
                maintainForegroundService = {
                    maintainNativeDemandForegroundOwnership(
                        maintainForegroundService = {
                            throw IllegalStateException("FGS start rejected")
                        },
                        recoverRuntime = { runtimeRecoveries += 1 },
                    )
                },
                stopForegroundService = { error("must not stop") },
            )
        }

        assertEquals(setOf("machine-a"), state.canonicalSnapshot().endpointIds)
        assertEquals(emptyList<NativeRendererDemandSnapshot>(), submitted)
        assertEquals(1, runtimeRecoveries)
    }

    @Test
    fun nonEmptyDemandEnsuresTheProcessRuntimeWithoutLosingIntentOnStartupFailure() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        var foregroundServiceStarts = 0

        assertThrows(IllegalStateException::class.java) {
            reconcileRendererDemand(
                rendererDemand = state,
                attachmentId = attached.attachmentId,
                baseDemandRevision = attached.demandRevision,
                endpointIds = setOf("machine-a"),
                replaceGoDemand = {},
                ensureRuntimeForDemand = { throw IllegalStateException("runtime is rebuilding") },
                maintainForegroundService = { foregroundServiceStarts += 1 },
                stopForegroundService = { error("must not stop") },
            )
        }

        assertEquals(setOf("machine-a"), state.currentSnapshot(attached.attachmentId).endpointIds)
        assertEquals(1, foregroundServiceStarts)
    }

    @Test
    fun emptyDemandStopsForegroundOwnershipBeforeGoMutationFailure() {
        val state = NativeRendererDemandState { "renderer" }
        val attached = state.attachRenderer()
        val active = state.replaceDemand(
            attached.attachmentId,
            attached.demandRevision,
            setOf("machine-a"),
        )
        val operations = mutableListOf<String>()

        assertThrows(IllegalStateException::class.java) {
            reconcileRendererDemand(
                rendererDemand = state,
                attachmentId = active.attachmentId,
                baseDemandRevision = active.demandRevision,
                endpointIds = emptySet(),
                replaceGoDemand = {
                    operations += "go"
                    throw IllegalStateException("temporary ABI failure")
                },
                ensureRuntimeForDemand = { error("must not ensure empty demand") },
                maintainForegroundService = { error("must not maintain empty demand") },
                stopForegroundService = { operations += "stop" },
            )
        }

        assertEquals(emptySet<String>(), state.canonicalSnapshot().endpointIds)
        assertEquals(listOf("stop", "go"), operations)
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
    fun userStopGateSurvivesLeaseRefreshEmptyReplacementAndRendererHandoff() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        val attached = state.attachRenderer()
        val active = state.replaceDemand(
            attached.attachmentId,
            attached.demandRevision,
            setOf("machine-a"),
        )

        val stopped = state.clearDemand()

        assertEquals(emptySet<String>(), stopped?.endpointIds)
        assertEquals(active.demandRevision + 1, stopped?.demandRevision)
        assertEquals(stopped?.demandRevision, stopped?.stopEpoch)
        assertEquals(attached.attachmentId, stopped?.attachmentId)
        val refreshed = state.currentSnapshot(attached.attachmentId)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(attached.attachmentId, refreshed.demandRevision, setOf("machine-a"))
        }

        val emptyReplacement = state.replaceDemand(
            attached.attachmentId,
            refreshed.demandRevision,
            emptySet(),
        )
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(
                attached.attachmentId,
                emptyReplacement.demandRevision,
                setOf("machine-a"),
            )
        }

        val replacementRenderer = state.attachRenderer()
        assertEquals(emptyReplacement.demandRevision, replacementRenderer.demandRevision)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(
                replacementRenderer.attachmentId,
                replacementRenderer.demandRevision,
                setOf("machine-a"),
            )
        }
    }

    @Test
    fun resumeIntentIsIdempotentWithinAnEpochAndFailsClosedAcrossUserStop() {
        var nextAttachment = 0
        val state = NativeRendererDemandState { "renderer-${++nextAttachment}" }
        val rendererA = state.attachRenderer()
        val active = state.replaceDemand(
            rendererA.attachmentId,
            rendererA.demandRevision,
            setOf("machine-a"),
        )
        val accepted = state.resumeDemand(rendererA.attachmentId, "intent-old", rendererA.stopEpoch)
        assertEquals(NativeRendererDemandResumeOutcome.RESUMED, accepted.outcome)
        assertEquals(active, accepted.snapshot)
        val stoppedA = requireNotNull(state.clearDemand())

        val rejectedRetry = state.resumeDemand(rendererA.attachmentId, "intent-old", stoppedA.stopEpoch)
        assertEquals(NativeRendererDemandResumeOutcome.STOPPED, rejectedRetry.outcome)
        assertEquals(stoppedA, rejectedRetry.snapshot)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(rendererA.attachmentId, stoppedA.demandRevision, setOf("machine-a"))
        }
        val delayed = state.resumeDemand(rendererA.attachmentId, "intent-delayed", active.stopEpoch)
        assertEquals(NativeRendererDemandResumeOutcome.STOPPED, delayed.outcome)
        val delayedRetry = state.resumeDemand(rendererA.attachmentId, "intent-delayed", stoppedA.stopEpoch)
        assertEquals(NativeRendererDemandResumeOutcome.STOPPED, delayedRetry.outcome)

        val stoppedB = state.attachRenderer()
        assertThrows(IllegalStateException::class.java) {
            state.resumeDemand(rendererA.attachmentId, "intent-old", stoppedA.stopEpoch)
        }

        val resumed = state.resumeDemand(stoppedB.attachmentId, "intent-fresh", stoppedB.stopEpoch)

        assertEquals(NativeRendererDemandResumeOutcome.RESUMED, resumed.outcome)
        assertEquals(stoppedB.demandRevision + 1, resumed.snapshot.demandRevision)
        assertEquals(stoppedB.stopEpoch, resumed.snapshot.stopEpoch)
        assertEquals(emptySet<String>(), resumed.snapshot.endpointIds)
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(
                stoppedB.attachmentId,
                stoppedB.demandRevision,
                setOf("machine-a"),
            )
        }
        val restored = state.replaceDemand(
            resumed.snapshot.attachmentId,
            resumed.snapshot.demandRevision,
            setOf("machine-a"),
        )
        assertEquals(setOf("machine-a"), restored.endpointIds)
        assertEquals(
            NativeRendererDemandResumeResult(restored, NativeRendererDemandResumeOutcome.RESUMED),
            state.resumeDemand(restored.attachmentId, "intent-fresh", restored.stopEpoch),
        )
    }

    @Test
    fun resumeIntentValidationAndCapacityFailClosed() {
        var attachment = 0
        val state = NativeRendererDemandState { "renderer-${++attachment}" }
        val attached = state.attachRenderer()

        assertThrows(IllegalStateException::class.java) {
            state.resumeDemand(attached.attachmentId, "   ", attached.stopEpoch)
        }
        assertThrows(IllegalStateException::class.java) {
            state.resumeDemand(attached.attachmentId, "x".repeat(129), attached.stopEpoch)
        }
        assertThrows(IllegalStateException::class.java) {
            state.resumeDemand(attached.attachmentId, "intent-negative-epoch", -1L)
        }
        repeat(4096) { index ->
            assertEquals(
                NativeRendererDemandResumeOutcome.RESUMED,
                state.resumeDemand(attached.attachmentId, "intent-$index", attached.stopEpoch).outcome,
            )
        }
        val stopped = requireNotNull(state.clearDemand())
        assertThrows(IllegalStateException::class.java) {
            state.resumeDemand(attached.attachmentId, "intent-over-capacity", stopped.stopEpoch)
        }
        assertThrows(IllegalStateException::class.java) {
            state.replaceDemand(attached.attachmentId, stopped.demandRevision, setOf("machine-a"))
        }
        val replacement = state.attachRenderer()
        assertEquals(
            NativeRendererDemandResumeOutcome.RESUMED,
            state.resumeDemand(replacement.attachmentId, "intent-after-rotation", replacement.stopEpoch).outcome,
        )
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

    private fun demandSnapshot(
        demandRevision: Long,
        stopEpoch: Long,
        stopped: Boolean,
    ) = NativeRendererDemandSnapshot(
        attachmentId = "renderer",
        demandRevision = demandRevision,
        stopEpoch = stopEpoch,
        endpointIds = emptySet(),
        stopped = stopped,
    )
}
