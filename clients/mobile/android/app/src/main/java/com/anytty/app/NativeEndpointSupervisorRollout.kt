package com.anytty.app

import java.nio.ByteBuffer
import java.security.MessageDigest

internal object NativeEndpointSupervisorRollout {
    fun isTakeover(endpointId: String, percent: Int = BuildConfig.ENDPOINT_SUPERVISOR_PERCENT): Boolean {
        require(percent in 0..100) { "endpoint supervisor percentage must be between 0 and 100" }
        val normalized = endpointId.trim()
        require(normalized.isNotEmpty()) { "endpoint ID is empty" }
        if (percent == 0) return false
        if (percent == 100) return true
        val digest = MessageDigest.getInstance("SHA-256").digest(normalized.toByteArray(Charsets.UTF_8))
        val bucket = java.lang.Long.remainderUnsigned(ByteBuffer.wrap(digest, 0, 8).long, 100L).toInt()
        return bucket < percent
    }
}
