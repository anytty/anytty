package com.anytty.app.goclient

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicReference

class AndroidClientPlatformPumpTest {
    @Test
    fun `platform completion failure is fatal and stops later requests`() {
        val active = AtomicBoolean(true)
        val requests = ArrayDeque(listOf(byteArrayOf(1), byteArrayOf(2)))
        val handled = mutableListOf<Byte>()
        val completed = mutableListOf<Byte>()
        val failures = mutableListOf<String>()

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
                    throw IllegalStateException("platform completion failed")
                }
                active.set(false)
            },
            onFatalError = { failures += it.message.orEmpty() },
        )

        assertEquals(listOf<Byte>(1), handled)
        assertEquals(listOf<Byte>(1), completed)
        assertEquals(listOf("platform completion failed"), failures)
    }

    @Test
    fun `fatal platform dequeue failure is reported while the engine is active`() {
        val active = AtomicBoolean(true)
        val failures = mutableListOf<String>()

        runAndroidClientPlatformPump(
            active = active,
            nextRequest = { throw IllegalStateException("native platform queue closed") },
            handleRequest = { it },
            completeRequest = {},
            onFatalError = { failures += it.message.orEmpty() },
        )

        assertEquals(listOf("native platform queue closed"), failures)
    }

    @Test
    fun `intentional platform shutdown does not report a fatal failure`() {
        val active = AtomicBoolean(false)
        val failures = mutableListOf<Exception>()

        runAndroidClientPlatformPump(
            active = active,
            nextRequest = { error("must not dequeue") },
            handleRequest = { it },
            completeRequest = {},
            onFatalError = failures::add,
        )

        assertEquals(emptyList<Exception>(), failures)
    }

    @Test
    fun `shutdown after dequeue skips platform dispatch`() {
        val active = AtomicBoolean(true)
        var handled = false
        var completed = false

        runAndroidClientPlatformPump(
            active = active,
            nextRequest = {
                active.set(false)
                byteArrayOf(1)
            },
            handleRequest = {
                handled = true
                it
            },
            completeRequest = { completed = true },
        )

        assertFalse(handled)
        assertFalse(completed)
    }

    @Test
    fun `retired generation cannot enter the persistence commit lock`() {
        val coordinator = AndroidPlatformStorageCoordinator()
        val oldGeneration = coordinator.beginGeneration()
        val freshGeneration = coordinator.beginGeneration()
        val persisted = AtomicReference("initial")

        coordinator.withGeneration(freshGeneration) { persisted.set("fresh") }
        try {
            coordinator.withGeneration(oldGeneration) { persisted.set("stale") }
            fail("retired generation unexpectedly entered the persistence commit lock")
        } catch (failure: ClientPlatformFailure) {
            assertEquals("cancelled", failure.code)
        }

        assertEquals("fresh", persisted.get())
    }

    @Test
    fun `new generation commits after an old uninterruptible mutator times out`() {
        val coordinator = AndroidPlatformStorageCoordinator()
        val oldGeneration = coordinator.beginGeneration()
        val oldEnteredCommit = CountDownLatch(1)
        val releaseOldCommit = CountDownLatch(1)
        val freshFinished = CountDownLatch(1)
        val persisted = AtomicReference("initial")
        val oldExecutor = Executors.newSingleThreadExecutor()
        val freshExecutor = Executors.newSingleThreadExecutor()
        val oldPump = oldExecutor.submit {
            coordinator.withGeneration(oldGeneration) {
                oldEnteredCommit.countDown()
                while (true) {
                    try {
                        releaseOldCommit.await()
                        break
                    } catch (_: InterruptedException) {
                        // Model a KeyStore/store primitive that ignores cancellation.
                    }
                }
                persisted.set("old")
            }
        }
        try {
            assertTrue(oldEnteredCommit.await(1, TimeUnit.SECONDS))
            coordinator.retire(oldGeneration)
            oldExecutor.shutdownNow()
            assertFalse(awaitAndroidClientPlatformPumpClose(oldPump))

            val freshGeneration = coordinator.beginGeneration()
            val freshCommit = freshExecutor.submit {
                coordinator.withGeneration(freshGeneration) { persisted.set("fresh") }
                freshFinished.countDown()
            }
            assertFalse(freshFinished.await(50, TimeUnit.MILLISECONDS))

            releaseOldCommit.countDown()
            assertTrue(awaitAndroidClientPlatformPumpClose(oldPump, timeoutMillis = 1_000L))
            freshCommit.get(1, TimeUnit.SECONDS)
            assertEquals("fresh", persisted.get())
        } finally {
            releaseOldCommit.countDown()
            oldExecutor.shutdownNow()
            freshExecutor.shutdownNow()
        }
    }

    @Test
    fun `platform close does not wait forever for a task that ignores interruption`() {
        val executor = Executors.newSingleThreadExecutor { runnable ->
            Thread(runnable, "uninterruptible-platform-test").apply { isDaemon = true }
        }
        val started = CountDownLatch(1)
        val release = CountDownLatch(1)
        val pump = executor.submit {
            started.countDown()
            while (true) {
                try {
                    release.await()
                    return@submit
                } catch (_: InterruptedException) {
                    // Model a platform primitive that does not respond to shutdownNow().
                }
            }
        }
        try {
            assertTrue(started.await(1, TimeUnit.SECONDS))

            executor.shutdownNow()
            val closeStartedAt = System.nanoTime()
            assertFalse(awaitAndroidClientPlatformPumpClose(pump, timeoutMillis = 25L))
            val closeElapsedMillis = TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - closeStartedAt)
            assertTrue("bounded close took ${closeElapsedMillis}ms", closeElapsedMillis < 500L)

            release.countDown()
            assertTrue(awaitAndroidClientPlatformPumpClose(pump, timeoutMillis = 1_000L))
            assertTrue(executor.awaitTermination(1, TimeUnit.SECONDS))
        } finally {
            release.countDown()
            executor.shutdownNow()
        }
    }
}
