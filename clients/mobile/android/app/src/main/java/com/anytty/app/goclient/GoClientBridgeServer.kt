package com.anytty.app.goclient

import org.java_websocket.WebSocket
import org.java_websocket.drafts.Draft
import org.java_websocket.exceptions.InvalidHandshakeException
import org.java_websocket.framing.CloseFrame
import org.java_websocket.handshake.ClientHandshake
import org.java_websocket.handshake.ServerHandshakeBuilder
import org.java_websocket.server.WebSocketServer
import java.net.InetSocketAddress
import java.nio.ByteBuffer
import java.nio.ByteOrder
import java.nio.CharBuffer
import java.nio.charset.CodingErrorAction
import java.security.MessageDigest
import java.util.concurrent.ArrayBlockingQueue
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

internal const val BRIDGE_RESPONSE_HEADER_BYTES = 21
private const val UTF8_SIZE_SCRATCH_BYTES = 256

internal fun bridgeResponseFrameBytes(payloadBytes: Int): Int? {
    if (payloadBytes < 0) return null
    val frameBytes = BRIDGE_RESPONSE_HEADER_BYTES.toLong() + payloadBytes.toLong()
    return if (frameBytes <= BRIDGE_MAX_MESSAGE_BYTES) frameBytes.toInt() else null
}

internal fun protobufUtf8PayloadBytes(value: String): Int? {
    val maxPayloadBytes = BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES
    if (value.length > maxPayloadBytes) return null

    val encoder = Charsets.UTF_8.newEncoder()
        .onMalformedInput(CodingErrorAction.REPLACE)
        .onUnmappableCharacter(CodingErrorAction.REPLACE)
    val input = CharBuffer.wrap(value)
    val scratch = ByteBuffer.allocate(UTF8_SIZE_SCRATCH_BYTES)
    var encodedBytes = 0

    while (true) {
        scratch.clear()
        val result = encoder.encode(input, scratch, true)
        encodedBytes += scratch.position()
        if (encodedBytes > maxPayloadBytes || (encodedBytes == maxPayloadBytes && input.hasRemaining())) {
            return null
        }
        if (result.isOverflow) continue
        check(result.isUnderflow)
        break
    }

    while (true) {
        scratch.clear()
        val result = encoder.flush(scratch)
        encodedBytes += scratch.position()
        if (encodedBytes > maxPayloadBytes) return null
        if (result.isOverflow) continue
        check(result.isUnderflow)
        return encodedBytes
    }
}

internal interface GoClientBridgeEngine : AutoCloseable {
    fun openSession(payload: ByteArray): Long
    fun execute(session: Long, payload: ByteArray): Long
    fun openResourceStream(session: Long, payload: ByteArray): Long
    fun sendResourceStreamFrame(stream: Long, payload: ByteArray)
    fun closeResourceStream(stream: Long)
    fun engineCommand(payload: ByteArray): Long
    fun cancel(operation: Long)
    fun closeSession(session: Long)
    fun release(handle: Long)
    fun nextEvent(): ByteArray
}

private class NativeGoClientBridgeEngine(
    private val engine: AndroidGoClientEngine,
) : GoClientBridgeEngine {
    override fun openSession(payload: ByteArray) = GoClientNative.openSession(engine.handle, payload)
    override fun execute(session: Long, payload: ByteArray) = GoClientNative.execute(engine.handle, session, payload)
    override fun openResourceStream(session: Long, payload: ByteArray) =
        GoClientNative.openResourceStream(engine.handle, session, payload)
    override fun sendResourceStreamFrame(stream: Long, payload: ByteArray) =
        GoClientNative.sendResourceStreamFrame(engine.handle, stream, payload)
    override fun closeResourceStream(stream: Long) = GoClientNative.closeResourceStream(engine.handle, stream)
    override fun engineCommand(payload: ByteArray) = GoClientNative.engineCommand(engine.handle, payload)
    override fun cancel(operation: Long) = GoClientNative.cancel(engine.handle, operation)
    override fun closeSession(session: Long) = GoClientNative.closeSession(engine.handle, session)
    override fun release(handle: Long) = GoClientNative.release(engine.handle, handle)
    override fun nextEvent() = GoClientNative.nextEvent(engine.handle, 0)
    override fun close() = engine.close()
}

/**
 * GoClientBridgeServer 是 WebView 到 Go binding 的唯一二进制数据面。
 * 它只解释固定 operation header 和 opaque Proto bytes，不解析任何 terminal/history/file/storage 字段。
 */
class GoClientBridgeServer internal constructor(
    private val engine: GoClientBridgeEngine,
    authToken: String,
) : WebSocketServer(InetSocketAddress("127.0.0.1", 0), 1, listOf(BridgeDraft6455())), AutoCloseable {
    constructor(engine: AndroidGoClientEngine, authToken: String) : this(NativeGoClientBridgeEngine(engine), authToken)

    private val active = AtomicBoolean(true)
    private val token = authToken.toByteArray(Charsets.UTF_8)
    private val connections = BridgeConnectionRegistry()
    private val pendingEvents = ArrayBlockingQueue<ByteArray>(EVENT_CAPACITY)
    private val eventThread = Thread(::pumpEvents, "anytty-go-events").apply { isDaemon = true }
    private val started = CountDownLatch(1)
    private val engineLifecycle = Object()
    private var engineStopping = false
    private var requestsInFlight = 0

    init {
        require(token.size == AUTH_TOKEN_BYTES && authToken.all { it.isLetterOrDigit() || it == '-' || it == '_' }) {
            "bridge token must be 43-byte base64url"
        }
        setWebSocketFactory(BridgeWebSocketFactory(connections))
    }

    override fun onStart() {
        started.countDown()
        eventThread.start()
    }

    fun awaitStarted(timeoutMillis: Long): Boolean = started.await(timeoutMillis, TimeUnit.MILLISECONDS)

    override fun onOpen(conn: WebSocket, handshake: ClientHandshake) = Unit

    override fun onWebsocketHandshakeReceivedAsServer(
        conn: WebSocket,
        draft: Draft,
        request: ClientHandshake,
    ): ServerHandshakeBuilder {
        val bridge = conn as? BridgeWebSocketImpl ?: throw InvalidHandshakeException("invalid binding transport")
        if (!connections.acquireUpgrade(bridge)) throw InvalidHandshakeException("binding upgrade limit")
        return super.onWebsocketHandshakeReceivedAsServer(conn, draft, request)
    }

    override fun onMessage(conn: WebSocket, message: String) {
        conn.close(CLOSE_PROTOCOL, "text frames are not supported")
    }

    override fun onMessage(conn: WebSocket, message: ByteBuffer) {
        val bridge = conn as? BridgeWebSocketImpl ?: run {
            conn.close(CLOSE_PROTOCOL, "invalid binding transport")
            return
        }
        if (!connections.isAuthenticated(bridge)) {
            if (message.remaining() != AUTH_FRAME_BYTES) {
                rejectAuthentication(bridge)
                return
            }
            val bytes = ByteArray(AUTH_FRAME_BYTES).also(message::get)
            authenticate(bridge, bytes)
            return
        }
        if (message.remaining() == 0) {
            conn.close(CLOSE_PROTOCOL, "empty binding frame")
            return
        }
        if (!connections.admitRequest(bridge)) return
        val bytes = ByteArray(message.remaining()).also(message::get)
        withEngineRequest {
            try {
                handleRequest(bridge, bytes)
            } catch (error: Throwable) {
                val requestId = if (bytes.size >= 9) ByteBuffer.wrap(bytes, 1, 8).order(ByteOrder.BIG_ENDIAN).long else 0L
                sendError(bridge, requestId, error.message ?: "native binding request failed")
            }
        }
    }

    override fun onClose(conn: WebSocket, code: Int, reason: String?, remote: Boolean) = Unit

    override fun onError(conn: WebSocket?, ex: Exception?) = Unit

    override fun close() {
        if (!active.compareAndSet(true, false)) return
        synchronized(engineLifecycle) { engineStopping = true }
        val openConnections = connections.stop()
        openConnections.forEach { connection ->
            runCatching { connection.closeConnection(CloseFrame.GOING_AWAY, "bridge stopped") }
        }
        runCatching { stop(1_000) }
        synchronized(engineLifecycle) {
            while (requestsInFlight != 0) engineLifecycle.wait()
        }
        runCatching { engine.close() }
        eventThread.interrupt()
    }

    private fun authenticate(conn: BridgeWebSocketImpl, frame: ByteArray) {
        if (frame.size != AUTH_FRAME_BYTES) {
            rejectAuthentication(conn)
            return
        }
        val candidate = frame.copyOfRange(1, AUTH_FRAME_BYTES)
        val tokenMatches = MessageDigest.isEqual(candidate, token)
        val valid = frame[0] == OP_AUTH && tokenMatches
        if (!valid || !connections.mayAuthenticate(conn)) {
            rejectAuthentication(conn)
            return
        }
        try {
            sendAck(conn, 0)
        } catch (_: RuntimeException) {
            rejectAuthentication(conn)
            return
        }
        val commit = connections.commitAuthentication(conn)
        if (!commit.accepted) {
            rejectAuthentication(conn)
            return
        }
        commit.replaced?.let { replaced ->
            runCatching { replaced.close(CLOSE_REPLACED, "binding client replaced") }
        }
        while (true) {
            val event = pendingEvents.poll() ?: break
            runCatching { sendFrame(conn, OP_EVENT, 0, 0, event) }
        }
    }

    private fun rejectAuthentication(conn: BridgeWebSocketImpl) {
        connections.beginClosing(conn)
        conn.closeForPolicy()
    }

    private fun handleRequest(conn: WebSocket, frame: ByteArray) {
        val input = ByteBuffer.wrap(frame).order(ByteOrder.BIG_ENDIAN)
        val operation = input.get()
        val requestId = input.long
        when (operation) {
            OP_OPEN_SESSION -> accept(conn, requestId, engine.openSession(remaining(input)))
            OP_EXECUTE -> {
                val session = input.long
                accept(conn, requestId, engine.execute(session, remaining(input)))
            }
            OP_OPEN_RESOURCE_STREAM -> {
                val session = input.long
                accept(conn, requestId, engine.openResourceStream(session, remaining(input)))
            }
            OP_SEND_RESOURCE_STREAM_FRAME -> {
                val stream = input.long
                engine.sendResourceStreamFrame(stream, remaining(input))
                sendAck(conn, requestId)
            }
            OP_CLOSE_RESOURCE_STREAM -> {
                engine.closeResourceStream(input.long)
                sendAck(conn, requestId)
            }
            OP_ENGINE_COMMAND -> accept(conn, requestId, engine.engineCommand(remaining(input)))
            OP_CANCEL -> {
                engine.cancel(input.long)
                sendAck(conn, requestId)
            }
            OP_CLOSE_SESSION -> {
                engine.closeSession(input.long)
                sendAck(conn, requestId)
            }
            OP_RELEASE -> {
                engine.release(input.long)
                sendAck(conn, requestId)
            }
            else -> sendError(conn, requestId, "unsupported binding operation")
        }
    }

    private fun pumpEvents() {
        while (active.get()) {
            val event = try {
                engine.nextEvent()
            } catch (_: InterruptedException) {
                return
            } catch (_: IllegalStateException) {
                return
            }
            val client = connections.currentConnection()
            val delivered = client != null && client.isOpen &&
                runCatching { sendFrame(client, OP_EVENT, 0, 0, event) }.isSuccess
            if (!delivered) {
                try {
                    pendingEvents.put(event)
                } catch (_: InterruptedException) {
                    return
                }
            }
        }
    }

    private fun accept(conn: WebSocket, requestId: Long, operationHandle: Long) =
        sendFrame(conn, OP_ACCEPTED, requestId, operationHandle, ByteArray(0))

    private fun sendAck(conn: WebSocket, requestId: Long) =
        sendFrame(conn, OP_ACK, requestId, 0, ByteArray(0))

    private fun sendError(conn: WebSocket, requestId: Long, message: String) {
        val encodedBytes = protobufUtf8PayloadBytes(message)
        if (encodedBytes == null || bridgeResponseFrameBytes(encodedBytes) == null) {
            conn.close(CLOSE_TOO_BIG, "binding message exceeds limit")
            return
        }
        val encoded = message.toByteArray(Charsets.UTF_8)
        check(encoded.size == encodedBytes)
        sendFrame(conn, OP_ERROR, requestId, 0, encoded)
    }

    private fun sendFrame(conn: WebSocket, operation: Byte, requestId: Long, handle: Long, payload: ByteArray) {
        val frameBytes = bridgeResponseFrameBytes(payload.size)
        if (frameBytes == null) {
            conn.close(CLOSE_TOO_BIG, "binding message exceeds limit")
            return
        }
        val output = ByteBuffer.allocate(frameBytes).order(ByteOrder.BIG_ENDIAN)
        output.put(operation).putLong(requestId).putLong(handle).putInt(payload.size).put(payload).flip()
        conn.send(output)
    }

    private fun remaining(input: ByteBuffer): ByteArray = ByteArray(input.remaining()).also(input::get)

    private inline fun withEngineRequest(action: () -> Unit) {
        synchronized(engineLifecycle) {
            if (engineStopping) return
            requestsInFlight += 1
        }
        try {
            action()
        } finally {
            synchronized(engineLifecycle) {
                requestsInFlight -= 1
                if (requestsInFlight == 0) engineLifecycle.notifyAll()
            }
        }
    }

    companion object {
        const val OP_AUTH: Byte = 0x01
        const val OP_OPEN_SESSION: Byte = 0x10
        const val OP_EXECUTE: Byte = 0x11
        const val OP_ENGINE_COMMAND: Byte = 0x12
        const val OP_CANCEL: Byte = 0x14
        const val OP_CLOSE_SESSION: Byte = 0x15
        const val OP_RELEASE: Byte = 0x16
        const val OP_OPEN_RESOURCE_STREAM: Byte = 0x17
        const val OP_SEND_RESOURCE_STREAM_FRAME: Byte = 0x18
        const val OP_CLOSE_RESOURCE_STREAM: Byte = 0x19
        const val OP_ACCEPTED: Byte = 0x20
        const val OP_ACK: Byte = 0x21
        const val OP_ERROR: Byte = 0x22
        const val OP_EVENT: Byte = 0x30
        private const val EVENT_CAPACITY = 256
        private const val AUTH_TOKEN_BYTES = 43
        private const val AUTH_FRAME_BYTES = 44
        private const val CLOSE_PROTOCOL = 1002
        private const val CLOSE_TOO_BIG = 1009
        private const val CLOSE_REPLACED = 4001
    }
}
