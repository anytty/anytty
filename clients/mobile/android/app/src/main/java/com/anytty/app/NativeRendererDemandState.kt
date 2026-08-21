package com.anytty.app

import java.util.UUID

internal data class NativeRendererDemandSnapshot(
    val attachmentId: String,
    val demandRevision: Long,
    val endpointIds: Set<String>,
)

/**
 * Process-owned canonical renderer demand. A new attachment inherits the last
 * snapshot until it submits its own full replacement, so WebView recreation
 * cannot briefly tear down a healthy native session.
 */
internal class NativeRendererDemandState(
    private val attachmentIdFactory: () -> String = { UUID.randomUUID().toString() },
) {
    private var attachmentId: String? = null
    private var demandRevision = 0L
    private val endpointIds = linkedSetOf<String>()

    fun attachRenderer(): NativeRendererDemandSnapshot {
        val nextAttachmentId = attachmentIdFactory().trim()
        check(nextAttachmentId.isNotEmpty()) { "renderer attachment ID is empty" }
        attachmentId = nextAttachmentId
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
        incrementRevision()
        endpointIds.clear()
        endpointIds.addAll(replacementEndpointIds)
        return snapshot(candidateAttachmentId)
    }

    fun currentSnapshot(candidateAttachmentId: String): NativeRendererDemandSnapshot {
        requireCurrentAttachment(candidateAttachmentId)
        return snapshot(candidateAttachmentId)
    }

    fun clearDemand(): NativeRendererDemandSnapshot? {
        incrementRevision()
        endpointIds.clear()
        return attachmentId?.let(::snapshot)
    }

    fun hasDemand(): Boolean = endpointIds.isNotEmpty()

    fun canonicalSnapshot(): NativeRendererDemandSnapshot = snapshot(attachmentId ?: "process-retained-demand")

    fun resetForTests() {
        attachmentId = null
        demandRevision = 0L
        endpointIds.clear()
    }

    private fun requireCurrentAttachment(candidateAttachmentId: String) {
        check(candidateAttachmentId.isNotEmpty() && attachmentId == candidateAttachmentId) {
            "renderer attachment is stale"
        }
    }

    private fun incrementRevision() {
        check(demandRevision < Long.MAX_VALUE) { "renderer demand revision is exhausted" }
        demandRevision += 1
    }

    private fun snapshot(currentAttachmentId: String) = NativeRendererDemandSnapshot(
        attachmentId = currentAttachmentId,
        demandRevision = demandRevision,
        endpointIds = endpointIds.toSet(),
    )
}
