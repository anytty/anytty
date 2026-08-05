package com.anytty.app.goclient

import org.java_websocket.WebSocketAdapter
import org.java_websocket.WebSocketImpl
import org.java_websocket.WebSocketServerFactory
import org.java_websocket.drafts.Draft
import org.java_websocket.drafts.Draft_6455
import org.java_websocket.enums.HandshakeState
import org.java_websocket.enums.Opcode
import org.java_websocket.exceptions.InvalidDataException
import org.java_websocket.exceptions.InvalidHandshakeException
import org.java_websocket.exceptions.LimitExceededException
import org.java_websocket.extensions.DefaultExtension
import org.java_websocket.framing.CloseFrame
import org.java_websocket.framing.Framedata
import org.java_websocket.handshake.ClientHandshake
import org.java_websocket.handshake.Handshakedata
import org.java_websocket.protocols.Protocol
import java.io.IOException
import java.nio.ByteBuffer
import java.nio.channels.ByteChannel
import java.nio.channels.SelectionKey
import java.nio.channels.SocketChannel
import java.nio.charset.StandardCharsets
import java.util.IdentityHashMap
import java.util.concurrent.Executors
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit

internal const val BRIDGE_PROTOCOL = "anytty.binding.v1"
internal const val BRIDGE_MAX_MESSAGE_BYTES = 4 * 1024 * 1024
internal const val BRIDGE_MAX_HEADER_BYTES = 16 * 1024
internal const val BRIDGE_AUTH_DEADLINE_NANOS = 2_000_000_000L
internal const val BRIDGE_PHYSICAL_LIMIT = 8
internal const val BRIDGE_UPGRADE_LIMIT = 4

/** Owns the exact physical/upgrade/auth lifecycle under one synchronization boundary. */
internal class BridgeConnectionRegistry {
    private enum class Phase {
        ACCEPTED,
        UPGRADED,
        CLOSING,
        CLOSED,
    }

    private data class State(
        val acceptedAtNanos: Long,
        var phase: Phase = Phase.ACCEPTED,
        var holdsUpgrade: Boolean = false,
        var authenticated: Boolean = false,
        var timeout: ScheduledFuture<*>? = null,
    )

    internal data class AuthenticationCommit(
        val accepted: Boolean,
        val replaced: BridgeWebSocketImpl? = null,
    )

    private val lock = Any()
    private val states = IdentityHashMap<BridgeWebSocketImpl, State>()
    private var upgradeCount = 0
    private var current: BridgeWebSocketImpl? = null
    private var stopped = false
    private val deadlines = Executors.newSingleThreadScheduledExecutor { task ->
        Thread(task, "anytty-bridge-deadline").apply { isDaemon = true }
    }

    fun registerPhysical(connection: BridgeWebSocketImpl): Boolean {
        val acceptedAtNanos = System.nanoTime()
        val state = State(acceptedAtNanos = acceptedAtNanos)
        return synchronized(lock) {
            if (stopped || states.size >= BRIDGE_PHYSICAL_LIMIT) {
                false
            } else {
                states[connection] = state
                true
            }
        }
    }

    fun armDeadline(connection: BridgeWebSocketImpl): Boolean {
        val state = synchronized(lock) {
            val registered = states[connection] ?: return@synchronized null
            if (stopped || registered.phase == Phase.CLOSING || registered.phase == Phase.CLOSED) null else registered
        } ?: return false
        val elapsedNanos = System.nanoTime() - state.acceptedAtNanos
        val delayNanos = (BRIDGE_AUTH_DEADLINE_NANOS - elapsedNanos).coerceAtLeast(0L)
        val timeout = try {
            deadlines.schedule(
                { expire(connection) },
                delayNanos,
                TimeUnit.NANOSECONDS,
            )
        } catch (_: RuntimeException) {
            beginClosing(connection)
            return false
        }

        val armed = synchronized(lock) {
            if (states[connection] !== state || stopped ||
                state.phase == Phase.CLOSING || state.phase == Phase.CLOSED
            ) {
                false
            } else {
                state.timeout = timeout
                true
            }
        }
        if (!armed) timeout.cancel(false)
        return armed
    }

    fun acquireUpgrade(connection: BridgeWebSocketImpl): Boolean {
        var timeout: ScheduledFuture<*>? = null
        val acquired = synchronized(lock) {
            val state = states[connection] ?: return@synchronized false
            if (stopped || state.phase == Phase.CLOSING || state.phase == Phase.CLOSED) return@synchronized false
            if (deadlineElapsed(state, System.nanoTime())) {
                timeout = beginClosingLocked(state)
                return@synchronized false
            }
            if (state.phase == Phase.UPGRADED) return@synchronized true
            if (upgradeCount >= BRIDGE_UPGRADE_LIMIT) return@synchronized false
            state.phase = Phase.UPGRADED
            state.holdsUpgrade = true
            upgradeCount += 1
            true
        }
        timeout?.cancel(false)
        return acquired
    }

    fun mayAuthenticate(connection: BridgeWebSocketImpl): Boolean = synchronized(lock) {
        val state = states[connection] ?: return@synchronized false
        !stopped &&
            state.phase == Phase.UPGRADED &&
            !state.authenticated &&
            !deadlineElapsed(state, System.nanoTime())
    }

    fun commitAuthentication(connection: BridgeWebSocketImpl): AuthenticationCommit {
        var timeout: ScheduledFuture<*>? = null
        val result = synchronized(lock) {
            val state = states[connection]
            if (state == null ||
                stopped ||
                state.phase != Phase.UPGRADED ||
                state.authenticated ||
                deadlineElapsed(state, System.nanoTime())
            ) {
                AuthenticationCommit(false)
            } else {
                val replaced = current?.takeIf { it !== connection }
                if (replaced != null) states[replaced]?.let(::beginClosingLocked)
                state.authenticated = true
                current = connection
                timeout = state.timeout
                state.timeout = null
                AuthenticationCommit(true, replaced)
            }
        }
        timeout?.cancel(false)
        return result
    }

    fun admitRequest(connection: BridgeWebSocketImpl): Boolean = synchronized(lock) {
        val state = states[connection]
        !stopped && state?.phase == Phase.UPGRADED && state.authenticated
    }

    fun isAuthenticated(connection: BridgeWebSocketImpl): Boolean = synchronized(lock) {
        val state = states[connection]
        state?.phase == Phase.UPGRADED && state.authenticated
    }

    fun currentConnection(): BridgeWebSocketImpl? = synchronized(lock) {
        val connection = current ?: return@synchronized null
        val state = states[connection]
        if (stopped || state?.phase != Phase.UPGRADED || !state.authenticated) null else connection
    }

    fun beginClosing(connection: BridgeWebSocketImpl) {
        val timeout = synchronized(lock) {
            val state = states[connection] ?: return@synchronized null
            beginClosingLocked(state)
        }
        timeout?.cancel(false)
    }

    fun closed(connection: BridgeWebSocketImpl) {
        val timeout = synchronized(lock) {
            val state = states[connection] ?: return@synchronized null
            if (state.phase == Phase.CLOSED) return@synchronized null
            state.phase = Phase.CLOSED
            if (state.holdsUpgrade) {
                state.holdsUpgrade = false
                upgradeCount -= 1
            }
            if (current === connection) current = null
            state.authenticated = false
            states.remove(connection)
            state.timeout.also { state.timeout = null }
        }
        timeout?.cancel(false)
    }

    fun stop(): List<BridgeWebSocketImpl> {
        val timeouts = mutableListOf<ScheduledFuture<*>>()
        val connections = synchronized(lock) {
            if (stopped) return@synchronized emptyList()
            stopped = true
            states.entries.forEach { (_, state) ->
                beginClosingLocked(state)?.let(timeouts::add)
            }
            states.keys.toList()
        }
        timeouts.forEach { it.cancel(false) }
        deadlines.shutdownNow()
        return connections
    }

    private fun expire(connection: BridgeWebSocketImpl) {
        val shouldClose = synchronized(lock) {
            val state = states[connection] ?: return@synchronized false
            if (state.authenticated || state.phase == Phase.CLOSING || state.phase == Phase.CLOSED ||
                !deadlineElapsed(state, System.nanoTime())
            ) {
                false
            } else {
                beginClosingLocked(state)
                true
            }
        }
        if (shouldClose) connection.closeForPolicy()
    }

    private fun deadlineElapsed(state: State, nowNanos: Long): Boolean =
        nowNanos - state.acceptedAtNanos >= BRIDGE_AUTH_DEADLINE_NANOS

    private fun beginClosingLocked(state: State): ScheduledFuture<*>? {
        if (state.phase == Phase.CLOSING || state.phase == Phase.CLOSED) return null
        state.phase = Phase.CLOSING
        val timeout = state.timeout
        state.timeout = null
        if (current != null && states[current] === state) current = null
        state.authenticated = false
        return timeout
    }
}

/** Registers accepted sockets before the selector can read and rejects overflow in wrapChannel. */
internal class BridgeWebSocketFactory(
    private val registry: BridgeConnectionRegistry,
) : WebSocketServerFactory {
    override fun createWebSocket(listener: WebSocketAdapter, draft: Draft): WebSocketImpl =
        create(listener, listOf(draft))

    override fun createWebSocket(listener: WebSocketAdapter, drafts: List<Draft>): WebSocketImpl =
        create(listener, drafts)

    private fun create(listener: WebSocketAdapter, drafts: List<Draft>): BridgeWebSocketImpl {
        val connection = BridgeWebSocketImpl(listener, drafts, registry)
        connection.registerPhysical()
        return connection
    }

    @Throws(IOException::class)
    override fun wrapChannel(channel: SocketChannel, key: SelectionKey): ByteChannel {
        val connection = key.attachment() as? BridgeWebSocketImpl
        if (connection != null && !connection.bindPhysicalChannel(channel)) {
            key.cancel()
            connection.closeConnection(CloseFrame.POLICY_VALIDATION, "bridge connection limit")
        }
        return channel
    }

    override fun close() {
        registry.stop().forEach { connection ->
            runCatching { connection.closeConnection(CloseFrame.GOING_AWAY, "bridge stopped") }
        }
    }
}

/** Adds only pre-handshake byte accounting; RFC 6455 parsing remains in WebSocketImpl. */
internal class BridgeWebSocketImpl(
    listener: WebSocketAdapter,
    drafts: List<Draft>,
    private val registry: BridgeConnectionRegistry,
) : WebSocketImpl(listener, drafts) {
    private val transportLock = Any()
    private var physicalRegistered = false
    private var physicalChannel: SocketChannel? = null
    private var transportClosing = false
    private var closeConnectionStarted = false
    private var closeConnectionComplete = false
    private var slotReleased = false
    private var headerComplete = false
    private var headerBytes = 0
    private var delimiterState = 0

    internal fun registerPhysical(): Boolean {
        val accepted = synchronized(transportLock) {
            registry.registerPhysical(this).also { physicalRegistered = it }
        }
        if (accepted && !registry.armDeadline(this)) closeForPolicy()
        return accepted
    }

    internal fun bindPhysicalChannel(channel: SocketChannel): Boolean {
        val shouldClose = synchronized(transportLock) {
            if (physicalChannel == null) physicalChannel = channel
            !physicalRegistered || transportClosing || closeConnectionComplete || physicalChannel !== channel
        }
        if (shouldClose) {
            runCatching { channel.close() }
            releaseSlotAfterClose()
            return false
        }
        return true
    }

    override fun setSelectionKey(key: SelectionKey) {
        (key.channel() as? SocketChannel)?.let(::bindPhysicalChannel)
        super.setSelectionKey(key)
    }

    override fun decode(socketBuffer: ByteBuffer) {
        val acceptsInput = synchronized(transportLock) { physicalRegistered && !transportClosing }
        if (!acceptsInput || isClosed) return
        if (!headerComplete) {
            val input = socketBuffer.duplicate()
            while (input.hasRemaining() && !headerComplete) {
                val byte = input.get()
                headerBytes += 1
                delimiterState = nextDelimiterState(delimiterState, byte)
                if (headerBytes > BRIDGE_MAX_HEADER_BYTES) {
                    registry.beginClosing(this)
                    closeForPolicy()
                    return
                }
                headerComplete = delimiterState == 4
            }
        }
        super.decode(socketBuffer)
    }

    override fun close(code: Int, message: String, remote: Boolean) {
        markTransportClosing()
        registry.beginClosing(this)
        super.close(code, message, remote)
    }

    override fun flushAndClose(code: Int, message: String, remote: Boolean) {
        markTransportClosing()
        registry.beginClosing(this)
        super.flushAndClose(code, message, remote)
    }

    override fun closeConnection(code: Int, message: String, remote: Boolean) {
        val shouldClose = synchronized(transportLock) {
            transportClosing = true
            if (closeConnectionStarted) {
                false
            } else {
                closeConnectionStarted = true
                true
            }
        }
        registry.beginClosing(this)
        if (!shouldClose) return
        closeOwnedChannel()
        try {
            super.closeConnection(code, message, remote)
        } finally {
            synchronized(transportLock) { closeConnectionComplete = true }
            closeOwnedChannel()
            releaseSlotAfterClose()
        }
    }

    internal fun closeForPolicy() = close(CloseFrame.POLICY_VALIDATION, "binding authentication failed")

    private fun markTransportClosing() {
        synchronized(transportLock) { transportClosing = true }
    }

    private fun closeOwnedChannel() {
        val keyChannel = runCatching { selectionKey?.channel() as? SocketChannel }.getOrNull()
        val channel = synchronized(transportLock) {
            if (physicalChannel == null && keyChannel != null) physicalChannel = keyChannel
            physicalChannel
        }
        if (channel != null && channel.isOpen) runCatching { channel.close() }
    }

    private fun releaseSlotAfterClose() {
        val release = synchronized(transportLock) {
            val channel = physicalChannel
            if (physicalRegistered && closeConnectionComplete && channel != null && !channel.isOpen && !slotReleased) {
                slotReleased = true
                physicalRegistered = false
                true
            } else {
                false
            }
        }
        if (release) registry.closed(this)
    }

    private fun nextDelimiterState(state: Int, byte: Byte): Int = when (state) {
        0 -> if (byte == CR) 1 else 0
        1 -> if (byte == LF) 2 else if (byte == CR) 1 else 0
        2 -> if (byte == CR) 3 else 0
        3 -> if (byte == LF) 4 else if (byte == CR) 1 else 0
        else -> 4
    }

    private companion object {
        const val CR: Byte = 13
        const val LF: Byte = 10
    }
}

/** Enforces the bridge handshake and aggregate message limit around Draft_6455. */
internal class BridgeDraft6455 : Draft_6455(
    listOf(DefaultExtension()),
    listOf(Protocol(BRIDGE_PROTOCOL)),
    BRIDGE_MAX_MESSAGE_BYTES,
) {
    private var messageBytes = 0L

    override fun translateHandshake(buffer: ByteBuffer): Handshakedata {
        val raw = buffer.duplicate()
        val handshake = super.translateHandshake(buffer)
        validateExactRequest(raw)
        return handshake
    }

    override fun acceptHandshakeAsServer(handshakedata: ClientHandshake): HandshakeState {
        if (handshakedata.resourceDescriptor != "/") return HandshakeState.NOT_MATCHED
        if (!handshakedata.hasFieldValue("Origin") || handshakedata.getFieldValue("Origin") != "http://localhost") {
            return HandshakeState.NOT_MATCHED
        }
        if (!handshakedata.hasFieldValue("Sec-WebSocket-Protocol") ||
            handshakedata.getFieldValue("Sec-WebSocket-Protocol") != BRIDGE_PROTOCOL
        ) {
            return HandshakeState.NOT_MATCHED
        }
        return super.acceptHandshakeAsServer(handshakedata)
    }

    override fun processFrame(webSocketImpl: WebSocketImpl, frame: Framedata) {
        val opcode = frame.opcode
        if (opcode == Opcode.TEXT) {
            messageBytes = 0
            throw InvalidDataException(CloseFrame.PROTOCOL_ERROR, "text frames are not supported")
        }
        val data = opcode == Opcode.BINARY || opcode == Opcode.CONTINUOUS
        if (data) {
            val payloadBytes = frame.payloadData.remaining().toLong()
            if (messageBytes > BRIDGE_MAX_MESSAGE_BYTES.toLong() - payloadBytes) {
                messageBytes = 0
                throw LimitExceededException("binding message exceeds limit", BRIDGE_MAX_MESSAGE_BYTES)
            }
            messageBytes += payloadBytes
        }
        try {
            super.processFrame(webSocketImpl, frame)
        } catch (error: InvalidDataException) {
            messageBytes = 0
            throw error
        } catch (error: RuntimeException) {
            messageBytes = 0
            throw error
        } catch (error: Error) {
            messageBytes = 0
            throw error
        } finally {
            if ((data && frame.isFin) || opcode == Opcode.CLOSING) messageBytes = 0
        }
    }

    override fun reset() {
        messageBytes = 0
        super.reset()
    }

    override fun copyInstance(): Draft = BridgeDraft6455()

    private fun validateExactRequest(raw: ByteBuffer) {
        var delimiter = 0
        var length = 0
        while (raw.hasRemaining() && delimiter != 4) {
            val byte = raw.get()
            length += 1
            delimiter = when (delimiter) {
                0 -> if (byte == 13.toByte()) 1 else 0
                1 -> if (byte == 10.toByte()) 2 else if (byte == 13.toByte()) 1 else 0
                2 -> if (byte == 13.toByte()) 3 else 0
                3 -> if (byte == 10.toByte()) 4 else if (byte == 13.toByte()) 1 else 0
                else -> 4
            }
        }
        if (delimiter != 4 || length > BRIDGE_MAX_HEADER_BYTES) throw InvalidHandshakeException("invalid bridge header")
        raw.position(raw.position() - length)
        val bytes = ByteArray(length)
        raw.get(bytes)
        val lines = String(bytes, StandardCharsets.US_ASCII).split("\r\n")
        if (lines.firstOrNull() != "GET / HTTP/1.1") throw InvalidHandshakeException("invalid bridge path")
        requireExactHeader(lines, "Origin", "http://localhost")
        requireExactHeader(lines, "Sec-WebSocket-Protocol", BRIDGE_PROTOCOL)
    }

    private fun requireExactHeader(lines: List<String>, name: String, value: String) {
        val matching = lines.filter { line ->
            val separator = line.indexOf(':')
            separator >= 0 && line.substring(0, separator).equals(name, ignoreCase = true)
        }
        if (matching.size != 1 || matching.single() != "${matching.single().substringBefore(':')}: $value") {
            throw InvalidHandshakeException("invalid bridge header")
        }
    }
}
