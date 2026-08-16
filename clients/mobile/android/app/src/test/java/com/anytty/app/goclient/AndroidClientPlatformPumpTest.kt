package com.anytty.app.goclient

import org.junit.Assert.assertEquals
import org.junit.Test
import java.util.concurrent.atomic.AtomicBoolean

class AndroidClientPlatformPumpTest {
    @Test
    fun `late cancelled response does not stop later platform requests`() {
        val active = AtomicBoolean(true)
        val requests = ArrayDeque(listOf(byteArrayOf(1), byteArrayOf(2)))
        val handled = mutableListOf<Byte>()
        val completed = mutableListOf<Byte>()

        runAndroidClientPlatformPump(
            active = active,
            nextRequest = { requests.removeFirst() },
            handleRequest = { payload ->
                handled += payload.single()
                payload
            },
            completeRequest = { response ->
                completed += response.single()
                if (response.single() == 1.toByte()) {
                    throw IllegalStateException("request was cancelled")
                }
                active.set(false)
            },
        )

        assertEquals(listOf<Byte>(1, 2), handled)
        assertEquals(listOf<Byte>(1, 2), completed)
    }
}
