package com.anytty.app

import java.util.UUID

internal data class NativeRendererDemandSnapshot(
    val attachmentId: String,
    val demandRevision: Long,
    val stopEpoch: Long,
    val endpointIds: Set<String>,
    val stopped: Boolean,
)

internal enum class NativeRendererDemandResumeOutcome(val wireValue: String) {
    RESUMED("resumed"),
    STOPPED("stopped"),
}

internal data class NativeRendererDemandResumeResult(
    val snapshot: NativeRendererDemandSnapshot,
    val outcome: NativeRendererDemandResumeOutcome,
)

/** Retains the latest process-owned Stop until JS cleanup explicitly acknowledges its epoch. */
internal class NativeDisconnectAllEventState {
    private val observers = linkedSetOf<(NativeRendererDemandSnapshot) -> Unit>()
    private var retained: NativeRendererDemandSnapshot? = null
    private var acknowledgedStopEpoch: Long? = null

    fun observe(
        canonicalSnapshot: NativeRendererDemandSnapshot,
        observer: (NativeRendererDemandSnapshot) -> Unit,
    ): AutoCloseable {
        val replay = synchronized(this) {
            retainLocked(canonicalSnapshot)
            observers += observer
            retained
        }
        try {
            replay?.let(observer)
        } catch (failure: Exception) {
            synchronized(this) { observers -= observer }
            throw failure
        }
        return AutoCloseable { synchronized(this) { observers -= observer } }
    }

    fun publish(snapshot: NativeRendererDemandSnapshot) {
        check(snapshot.stopped) { "disconnect-all event must carry a stopped demand snapshot" }
        val delivery = synchronized(this) {
            if (!retainLocked(snapshot)) return
            observers.toList()
        }
        for (observer in delivery) runCatching { observer(snapshot) }
    }

    fun acknowledge(stopEpoch: Long) {
        check(stopEpoch >= 0L) { "disconnect-all acknowledgement epoch is invalid" }
        synchronized(this) {
            val current = retained
            if (current == null) {
                check(acknowledgedStopEpoch == stopEpoch) {
                    "disconnect-all acknowledgement has no matching retained Stop"
                }
                return
            }
            check(current.stopEpoch == stopEpoch) {
                "disconnect-all acknowledgement does not match the retained Stop"
            }
            retained = null
            acknowledgedStopEpoch = stopEpoch
        }
    }

    fun resetForTests() {
        synchronized(this) {
            observers.clear()
            retained = null
            acknowledgedStopEpoch = null
        }
    }

    private fun retainLocked(snapshot: NativeRendererDemandSnapshot): Boolean {
        if (!snapshot.stopped) return false
        val acknowledged = acknowledgedStopEpoch
        if (acknowledged != null && snapshot.stopEpoch <= acknowledged) return false
        val current = retained
        if (current != null && snapshot.stopEpoch < current.stopEpoch) return false
        retained = snapshot
        return true
    }
}

private data class NativeRendererDemandResumeRecord(
    val stopEpoch: Long,
    val accepted: Boolean,
)

/**
 * Process-owned canonical renderer demand. Replacing a renderer fences the old
 * attachment without changing demand; the new renderer's first full replacement
 * reconciles the retained endpoints with its visible work.
 */
internal class NativeRendererDemandState(
    private val attachmentIdFactory: () -> String = { UUID.randomUUID().toString() },
) {
    private companion object {
        const val MAX_RETAINED_RESUME_INTENTS = 4096
    }

    private var attachmentId: String? = null
    private var demandRevision = 0L
    private var stopEpoch = 0L
    private val endpointIds = linkedSetOf<String>()
    private var userStopGate = false
    private val resumeIntents = linkedMapOf<String, NativeRendererDemandResumeRecord>()

    fun attachRenderer(): NativeRendererDemandSnapshot {
        val nextAttachmentId = nextAttachmentId()
        attachmentId = nextAttachmentId
        resumeIntents.clear()
        return snapshot(nextAttachmentId)
    }

    fun detachRenderer(candidateAttachmentId: String) {
        if (attachmentId == candidateAttachmentId) attachmentId = null
    }

    fun replaceDemand(
        candidateAttachmentId: String,
        baseDemandRevision: Long,
        replacementEndpointIds: Set<String>,
    ): NativeRendererDemandSnapshot {
        requireCurrentAttachment(candidateAttachmentId)
        check(baseDemandRevision == demandRevision) {
            "renderer demand revision is stale"
        }
        check(replacementEndpointIds.none { it.isBlank() }) {
            "renderer demand contains an empty endpoint ID"
        }
        check(replacementEndpointIds.isEmpty() || !userStopGate) {
            "renderer demand is stopped; an explicit resume is required"
        }
        incrementRevision()
        endpointIds.clear()
        endpointIds.addAll(replacementEndpointIds)
        return snapshot(candidateAttachmentId)
    }

    fun resumeDemand(
        candidateAttachmentId: String,
        intentId: String,
        baseStopEpoch: Long,
    ): NativeRendererDemandResumeResult {
        requireCurrentAttachment(candidateAttachmentId)
        val normalizedIntentId = intentId.trim()
        check(normalizedIntentId.isNotEmpty() && normalizedIntentId.length <= 128) {
            "renderer resume intent ID is invalid"
        }
        check(baseStopEpoch >= 0L) { "renderer resume base stop epoch is invalid" }
        val existing = resumeIntents[normalizedIntentId]
        if (existing != null) {
            val outcome = if (existing.accepted && existing.stopEpoch == stopEpoch) {
                NativeRendererDemandResumeOutcome.RESUMED
            } else {
                NativeRendererDemandResumeOutcome.STOPPED
            }
            return NativeRendererDemandResumeResult(snapshot(candidateAttachmentId), outcome)
        }
        check(resumeIntents.size < MAX_RETAINED_RESUME_INTENTS) {
            "renderer resume intent capacity is exhausted"
        }
        if (baseStopEpoch != stopEpoch) {
            resumeIntents[normalizedIntentId] = NativeRendererDemandResumeRecord(baseStopEpoch, accepted = false)
            return NativeRendererDemandResumeResult(
                snapshot(candidateAttachmentId),
                NativeRendererDemandResumeOutcome.STOPPED,
            )
        }
        if (userStopGate) {
            incrementRevision()
            userStopGate = false
        }
        resumeIntents[normalizedIntentId] = NativeRendererDemandResumeRecord(stopEpoch, accepted = true)
        return NativeRendererDemandResumeResult(
            snapshot(candidateAttachmentId),
            NativeRendererDemandResumeOutcome.RESUMED,
        )
    }

    fun currentSnapshot(candidateAttachmentId: String): NativeRendererDemandSnapshot {
        requireCurrentAttachment(candidateAttachmentId)
        return snapshot(candidateAttachmentId)
    }

    /** User stop stays latched until an exact current lease explicitly resumes demand. */
    fun clearDemand(): NativeRendererDemandSnapshot? {
        incrementRevision()
        stopEpoch = demandRevision
        endpointIds.clear()
        userStopGate = true
        return attachmentId?.let(::snapshot)
    }

    fun hasDemand(): Boolean = endpointIds.isNotEmpty()

    fun canonicalSnapshot(): NativeRendererDemandSnapshot = snapshot(attachmentId ?: "process-retained-demand")

    fun resetForTests() {
        attachmentId = null
        demandRevision = 0L
        stopEpoch = 0L
        endpointIds.clear()
        userStopGate = false
        resumeIntents.clear()
    }

    private fun requireCurrentAttachment(candidateAttachmentId: String) {
        check(candidateAttachmentId.isNotEmpty() && attachmentId == candidateAttachmentId) {
            "renderer attachment is stale"
        }
    }

    private fun nextAttachmentId(): String {
        val nextAttachmentId = attachmentIdFactory().trim()
        check(nextAttachmentId.isNotEmpty()) { "renderer attachment ID is empty" }
        check(nextAttachmentId != attachmentId) { "renderer attachment ID was reused" }
        return nextAttachmentId
    }

    private fun incrementRevision() {
        check(demandRevision < Long.MAX_VALUE) { "renderer demand revision is exhausted" }
        demandRevision += 1
    }

    private fun snapshot(currentAttachmentId: String) = NativeRendererDemandSnapshot(
        attachmentId = currentAttachmentId,
        demandRevision = demandRevision,
        stopEpoch = stopEpoch,
        endpointIds = endpointIds.toSet(),
        stopped = userStopGate,
    )
}
