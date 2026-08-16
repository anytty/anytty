package com.anytty.app

internal class NativeConnectionRuntimeCoordinator(
    private val startRuntime: () -> Unit,
    private val isRuntimeStarted: () -> Boolean,
    private val resetRuntime: () -> Unit,
) {
    private enum class State { NEW, READY, DESTROYED }

    private val monitor = Any()
    private var state = State.NEW

    fun load() = synchronized(monitor) {
        check(state == State.NEW) { "native runtime cannot be loaded after destruction" }
        if (!isRuntimeStarted()) startRuntime()
        state = State.READY
    }

    fun isReady(): Boolean = synchronized(monitor) { state == State.READY }

    fun ensureForForeground() = synchronized(monitor) {
        requireReadyLocked()
        if (!isRuntimeStarted()) startRuntime()
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
}
