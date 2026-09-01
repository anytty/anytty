package com.anytty.app

internal class NativeConnectionRuntimeCoordinator(
    private val startRuntime: () -> Unit,
    private val isRuntimeStarted: () -> Boolean,
    private val resetRuntime: () -> Unit,
) {
    private enum class State { NEW, READY, DESTROYED }

    private val monitor = Any()
    private var state = State.NEW

    /** Plugin registration must survive a transient native start failure; the owner retries it. */
    fun load(): Exception? = synchronized(monitor) {
        check(state == State.NEW) { "native runtime cannot be loaded after destruction" }
        state = State.READY
        if (isRuntimeStarted()) return@synchronized null
        try {
            startRuntime()
            null
        } catch (failure: Exception) {
            failure
        }
    }

    fun isReady(): Boolean = synchronized(monitor) { state == State.READY }

    fun ensureForForeground() = synchronized(monitor) {
        ensureRuntimeStartedLocked()
    }

    fun ensureForBridgeEndpoint() = synchronized(monitor) {
        ensureRuntimeStartedLocked()
    }

    fun resetLocalPairings() = synchronized(monitor) {
        requireReadyLocked()
        resetRuntime()
    }

    fun destroy() = synchronized(monitor) {
        if (state == State.DESTROYED) return@synchronized
        state = State.DESTROYED
    }

    private fun requireReadyLocked() {
        check(state == State.READY) { "native runtime is not available" }
    }

    private fun ensureRuntimeStartedLocked() {
        requireReadyLocked()
        if (!isRuntimeStarted()) startRuntime()
    }
}
