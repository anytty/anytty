package com.anytty.app.goclient

import org.java_websocket.WebSocket
import org.java_websocket.WebSocketAdapter
import org.java_websocket.WebSocketImpl
import org.java_websocket.WebSocketServerFactory
import org.java_websocket.drafts.Draft
import org.java_websocket.enums.Opcode
import org.java_websocket.exceptions.InvalidDataException
import org.java_websocket.exceptions.InvalidHandshakeException
import org.java_websocket.framing.BinaryFrame
import org.java_websocket.framing.CloseFrame
import org.java_websocket.framing.ContinuousFrame
import org.java_websocket.framing.PongFrame
import org.java_websocket.framing.TextFrame
import org.java_websocket.handshake.Handshakedata
import org.java_websocket.handshake.ClientHandshake
import org.java_websocket.handshake.ServerHandshakeBuilder
import org.java_websocket.server.WebSocketServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotSame
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.io.ByteArrayOutputStream
import java.io.Closeable
import java.io.EOFException
import java.net.InetSocketAddress
import java.net.Socket
import java.net.SocketTimeoutException
import java.nio.ByteBuffer
import java.nio.channels.ByteChannel
import java.nio.charset.StandardCharsets
import java.nio.channels.SelectableChannel
import java.nio.channels.SelectionKey
import java.nio.channels.Selector
import java.nio.channels.SocketChannel
import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger

class BridgeTransportTest {
    private val servers = mutableListOf<GoClientBridgeServer>()
    private val sockets = mutableListOf<RawSocket>()

    @After
    fun tearDown() {
        sockets.forEach { runCatching { it.close() } }
        servers.forEach { runCatching { it.close() } }
    }

    @Test
    fun `eight slow accepts reserve physical slots and ninth is closed`() {
        val server = startServer()
        List(BRIDGE_PHYSICAL_LIMIT) { raw(server) }

        val rejected = raw(server)
        assertEquals(-1, rejected.readByte())
    }

    @Test
    fun `eight closing connections retain physical slots until closeConnection returns`() {
        val registry = BridgeConnectionRegistry()
        val factory = BridgeWebSocketFactory(registry)
        val closeEntered = CountDownLatch(BRIDGE_PHYSICAL_LIMIT)
        val closeRelease = CountDownLatch(1)
        val closingListener = testWebSocketListener(onClose = {
            closeEntered.countDown()
            closeRelease.await(5, TimeUnit.SECONDS)
        })
        val closing = List(BRIDGE_PHYSICAL_LIMIT) {
            factory.createWebSocket(closingListener, listOf(BridgeDraft6455())) as BridgeWebSocketImpl
        }
        val channels = closing.map { connection ->
            SocketChannel.open().also { channel ->
                val key = AttachedSelectionKey(channel, connection)
                connection.setSelectionKey(key)
                connection.setChannel(factory.wrapChannel(channel, key))
            }
        }
        val closeThreads = closing.map { connection ->
            Thread {
                connection.closeConnection(CloseFrame.GOING_AWAY, "test close", false)
            }.apply { start() }
        }

        try {
            assertTrue(closeEntered.await(1, TimeUnit.SECONDS))

            val rejected = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
            SocketChannel.open().use { channel ->
                factory.wrapChannel(channel, AttachedSelectionKey(channel, rejected))
                assertFalse(channel.isOpen)
            }

            closeRelease.countDown()
            closeThreads.forEach { thread ->
                thread.join(1_000)
                assertFalse(thread.isAlive)
            }
            channels.forEach { assertFalse(it.isOpen) }

            val replacement = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
            SocketChannel.open().use { channel ->
                factory.wrapChannel(channel, AttachedSelectionKey(channel, replacement))
                assertTrue(channel.isOpen)
            }
            replacement.closeConnection(CloseFrame.NORMAL, "test complete", false)
        } finally {
            closeRelease.countDown()
            closeThreads.forEach { it.join(1_000) }
            factory.close()
        }
    }

    @Test
    fun `channel bound while close callback is blocked closes without releasing slot early`() {
        val registry = BridgeConnectionRegistry()
        val factory = BridgeWebSocketFactory(registry)
        val closeEntered = CountDownLatch(1)
        val closeRelease = CountDownLatch(1)
        val target = factory.createWebSocket(
            testWebSocketListener(onClose = {
                closeEntered.countDown()
                closeRelease.await(5, TimeUnit.SECONDS)
            }),
            listOf(BridgeDraft6455()),
        ) as BridgeWebSocketImpl
        val fillers = List(BRIDGE_PHYSICAL_LIMIT - 1) {
            factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
        }
        val fillerChannels = fillers.map { bind(factory, it) }
        val closeThread = Thread {
            target.closeConnection(CloseFrame.GOING_AWAY, "test close", false)
        }.apply { start() }

        try {
            assertTrue(closeEntered.await(1, TimeUnit.SECONDS))
            SocketChannel.open().use { channel ->
                val key = AttachedSelectionKey(channel, target)
                target.setSelectionKey(key)
                factory.wrapChannel(channel, key)
                assertFalse(channel.isOpen)
            }
            assertRejected(factory)

            closeRelease.countDown()
            closeThread.join(1_000)
            assertFalse(closeThread.isAlive)

            val replacement = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
            bind(factory, replacement).use { assertTrue(it.isOpen) }
            replacement.closeConnection(CloseFrame.NORMAL, "test complete", false)
        } finally {
            closeRelease.countDown()
            closeThread.join(1_000)
            fillers.forEach { it.closeConnection(CloseFrame.NORMAL, "test complete", false) }
            fillerChannels.forEach { runCatching { it.close() } }
            factory.close()
        }
    }

    @Test
    fun `close completed before channel binding retains slot until late channel closes`() {
        val registry = BridgeConnectionRegistry()
        val factory = BridgeWebSocketFactory(registry)
        val target = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
        val fillers = List(BRIDGE_PHYSICAL_LIMIT - 1) {
            factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
        }
        val fillerChannels = fillers.map { bind(factory, it) }

        try {
            target.closeConnection(CloseFrame.GOING_AWAY, "test close", false)
            assertRejected(factory)

            SocketChannel.open().use { channel ->
                val key = AttachedSelectionKey(channel, target)
                target.setSelectionKey(key)
                factory.wrapChannel(channel, key)
                assertFalse(channel.isOpen)
            }

            val replacement = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
            bind(factory, replacement).use { assertTrue(it.isOpen) }
            replacement.closeConnection(CloseFrame.NORMAL, "test complete", false)
        } finally {
            fillers.forEach { it.closeConnection(CloseFrame.NORMAL, "test complete", false) }
            fillerChannels.forEach { runCatching { it.close() } }
            factory.close()
        }
    }

    @Test
    fun `partial header consumes only physical slot and fifth complete upgrade is rejected`() {
        val server = startServer()
        raw(server).writeAscii("GET / HTTP/1.1\r\n")

        repeat(BRIDGE_UPGRADE_LIMIT) {
            assertTrue(upgraded(server).lastHttpResponse.startsWith("HTTP/1.1 101"))
        }

        val rejected = raw(server)
        rejected.writeAscii(handshakeBytes())
        assertFalse(rejected.readUntilClose().contains("HTTP/1.1 101"))
    }

    @Test
    fun `replaced sockets release capacity only after closeConnection reaches peer EOF`() {
        val server = startServer()
        val upgraded = mutableListOf<RawSocket>()
        repeat(BRIDGE_UPGRADE_LIMIT) {
            val socket = authenticated(server)
            upgraded += socket
            if (it > 0) assertEquals(8, upgraded[it - 1].readFrame().opcode)
        }
        upgraded.dropLast(1).forEach { assertEquals(-1, it.readByte()) }

        repeat(BRIDGE_UPGRADE_LIMIT - 1) { assertTrue(upgraded(server).lastHttpResponse.startsWith("HTTP/1.1 101")) }
        val rejected = raw(server)
        rejected.writeAscii(handshakeBytes())
        assertFalse(rejected.readUntilClose().contains("HTTP/1.1 101"))
    }

    @Test
    fun `complete header accepts exactly 16 KiB and rejects one byte more`() {
        val server = startServer()
        val accepted = raw(server)
        accepted.writeAscii(handshakeBytes(totalBytes = BRIDGE_MAX_HEADER_BYTES))
        assertTrue(accepted.readHttpResponse().startsWith("HTTP/1.1 101"))

        val rejected = raw(server)
        rejected.writeAscii(handshakeBytes(totalBytes = BRIDGE_MAX_HEADER_BYTES + 1))
        assertFalse(rejected.readUntilClose().contains("HTTP/1.1 101"))
    }

    @Test
    fun `upgrade acquisition rejects elapsed deadline while scheduler is blocked`() {
        val registry = BridgeConnectionRegistry()
        val factory = BridgeWebSocketFactory(registry)
        val deadlineEntered = CountDownLatch(1)
        val deadlineRelease = CountDownLatch(1)
        val deadlineReturned = CountDownLatch(1)
        val blocker = factory.createWebSocket(
            testWebSocketListener(onWriteDemand = {
                deadlineEntered.countDown()
                deadlineRelease.await(5, TimeUnit.SECONDS)
                deadlineReturned.countDown()
            }),
            listOf(BridgeDraft6455()),
        ) as BridgeWebSocketImpl
        val targetCreated = CountDownLatch(1)
        val server = DeadlineHandshakeServer(registry, factory, targetCreated)
        server.start()
        assertTrue(server.awaitStarted())
        val target = RawSocket(server.port, 2_500).also(sockets::add)
        assertTrue(targetCreated.await(1, TimeUnit.SECONDS))
        val elapsed = CountDownLatch(1)
        val testClock = Executors.newSingleThreadScheduledExecutor()
        testClock.schedule(
            elapsed::countDown,
            BRIDGE_AUTH_DEADLINE_NANOS + TimeUnit.MILLISECONDS.toNanos(1),
            TimeUnit.NANOSECONDS,
        )

        try {
            assertTrue(deadlineEntered.await(3, TimeUnit.SECONDS))
            assertTrue(elapsed.await(1, TimeUnit.SECONDS))
            target.writeAscii(handshakeBytes())
            assertFalse(target.readUntilClose().contains("HTTP/1.1 101"))
        } finally {
            testClock.shutdownNow()
            deadlineRelease.countDown()
            assertTrue(deadlineReturned.await(1, TimeUnit.SECONDS))
            server.stop(1_000)
            blocker.closeConnection(CloseFrame.POLICY_VALIDATION, "test complete", false)
            bind(factory, blocker).close()
            factory.close()
        }
    }

    @Test
    fun `blocked engine request does not delay another sockets accept-to-auth deadline`() {
        val engine = FakeEngine(blockExecute = true)
        val server = startServer(engine)
        val active = authenticated(server)
        val unauthenticated = upgraded(server)
        active.sendFrame(Opcode.BINARY, executeFrame(7), true)
        assertTrue(engine.executeEntered.await(1, TimeUnit.SECONDS))

        try {
            val close = unauthenticated.readFrame()
            assertEquals(8, close.opcode)
            assertEquals(CloseFrame.POLICY_VALIDATION, close.closeCode())
        } finally {
            engine.executeRelease.countDown()
        }
        assertEquals(GoClientBridgeServer.OP_ACCEPTED, active.readFrame().payload[0])
    }

    @Test
    fun `strict path origin and protocol matrix`() {
        val invalid = listOf(
            handshakeBytes(path = "/other"),
            handshakeBytes(origin = null),
            handshakeBytes(origin = "http://localhost:80"),
            handshakeBytes(origin = " http://localhost"),
            handshakeBytes(origin = "http://localhost "),
            handshakeBytes(origin = null, extraHeaders = listOf("Origin : http://localhost")),
            handshakeBytes(origin = null, extraHeaders = listOf("Origin:\thttp://localhost")),
            handshakeBytes(protocol = null),
            handshakeBytes(protocol = "$BRIDGE_PROTOCOL,other"),
            handshakeBytes(protocol = " $BRIDGE_PROTOCOL"),
            handshakeBytes(protocol = "$BRIDGE_PROTOCOL "),
            handshakeBytes(protocol = null, extraHeaders = listOf("Sec-WebSocket-Protocol : $BRIDGE_PROTOCOL")),
            handshakeBytes(protocol = null, extraHeaders = listOf("Sec-WebSocket-Protocol:\t$BRIDGE_PROTOCOL")),
            handshakeBytes(extraHeaders = listOf("Origin: http://localhost")),
            handshakeBytes(extraHeaders = listOf("Sec-WebSocket-Protocol: $BRIDGE_PROTOCOL")),
        )
        for (request in invalid) {
            val socket = raw(startServer(), 250)
            socket.writeAscii(request)
            assertFalse("invalid request upgraded: $request", socket.readUntilClose().contains("HTTP/1.1 101"))
        }

        val valid = upgraded(startServer())
        assertTrue(valid.lastHttpResponse.contains("Sec-WebSocket-Protocol: $BRIDGE_PROTOCOL"))
    }

    @Test
    fun `only exact auth message is acknowledged and failures close 1008`() {
        val valid = authenticated(startServer())
        assertTrue(valid.lastHttpResponse.contains("Sec-WebSocket-Protocol: $BRIDGE_PROTOCOL"))

        val invalidFrames = listOf(
            ByteArray(0),
            authFrame(TOKEN).copyOf(43),
            authFrame(TOKEN).copyOf(45),
            authFrame(TOKEN).also { it[0] = 2 },
            authFrame("B".repeat(43)),
            ByteArray(256 * 1024),
        )
        for (invalid in invalidFrames) {
            val socket = upgraded(startServer())
            socket.sendFrame(Opcode.BINARY, invalid, true)
            val close = socket.readFrame()
            assertEquals(8, close.opcode)
            assertEquals(CloseFrame.POLICY_VALIDATION, close.closeCode())
            assertFalse(String(close.payload, StandardCharsets.UTF_8).contains("B".repeat(8)))
        }
    }

    @Test
    fun `oversized unauthenticated direct buffer is rejected without moving position`() {
        val server = startServer()
        val registry = BridgeConnectionRegistry()
        try {
            val connection = BridgeWebSocketImpl(testWebSocketListener(), listOf(BridgeDraft6455()), registry)
            val message = ByteBuffer.allocateDirect(256 * 1024).apply {
                position(17)
                limit(capacity() - 11)
            }
            val position = message.position()
            server.onMessage(connection, message)
            assertEquals(position, message.position())
        } finally {
            registry.stop()
        }
    }

    @Test
    fun `continuations aggregate to 1009 controls do not reset and text is 1002`() {
        val overLimit = upgraded(startServer())
        overLimit.sendFrame(Opcode.BINARY, ByteArray(BRIDGE_MAX_MESSAGE_BYTES / 2), false)
        overLimit.sendFrame(Opcode.PING, byteArrayOf(1), true)
        assertEquals(10, overLimit.readFrame().opcode)
        overLimit.sendFrame(Opcode.CONTINUOUS, ByteArray(BRIDGE_MAX_MESSAGE_BYTES / 2 + 1), true)
        assertEquals(CloseFrame.TOOBIG, overLimit.readFrame().closeCode())

        val text = upgraded(startServer())
        text.sendFrame(Opcode.TEXT, "x".toByteArray(), true)
        assertEquals(CloseFrame.PROTOCOL_ERROR, text.readFrame().closeCode())
    }

    @Test
    fun `1009 releases capacity only after closeConnection reaches peer EOF`() {
        val server = startServer()
        val socketsAtLimit = List(BRIDGE_UPGRADE_LIMIT) { upgraded(server) }
        socketsAtLimit.first().sendFrame(Opcode.BINARY, ByteArray(BRIDGE_MAX_MESSAGE_BYTES), false)
        socketsAtLimit.first().sendFrame(Opcode.CONTINUOUS, byteArrayOf(1), true)
        assertEquals(CloseFrame.TOOBIG, socketsAtLimit.first().readFrame().closeCode())
        assertEquals(-1, socketsAtLimit.first().readByte())
        assertTrue(upgraded(server).lastHttpResponse.startsWith("HTTP/1.1 101"))
    }

    @Test
    fun `Kotlin response header is counted before allocation`() {
        assertEquals(
            BRIDGE_MAX_MESSAGE_BYTES,
            bridgeResponseFrameBytes(BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES),
        )
        assertNull(bridgeResponseFrameBytes(BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES + 1))
    }

    @Test
    fun `error UTF-8 size is bounded before response allocation`() {
        val maxPayloadBytes = BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES
        assertEquals(0, protobufUtf8PayloadBytes(""))
        assertEquals(2, protobufUtf8PayloadBytes("\u00e9"))
        assertEquals(maxPayloadBytes, protobufUtf8PayloadBytes("x".repeat(maxPayloadBytes)))
        assertNull(protobufUtf8PayloadBytes("x".repeat(maxPayloadBytes + 1)))
    }

    @Test
    fun `isolated surrogate replacement matches response UTF-8 encoding`() {
        val malformed = "\uD800x\uDC00"
        assertEquals(
            malformed.toByteArray(Charsets.UTF_8).size,
            protobufUtf8PayloadBytes(malformed),
        )
    }

    @Test
    fun `multibyte error is rejected at the response allocation boundary`() {
        val maxPayloadBytes = BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES
        val completeThreeByteCharacters = maxPayloadBytes / 3
        assertEquals(
            completeThreeByteCharacters * 3,
            protobufUtf8PayloadBytes("\u20ac".repeat(completeThreeByteCharacters)),
        )
        assertNull(protobufUtf8PayloadBytes("\u20ac".repeat(completeThreeByteCharacters + 1)))
    }

    @Test
    fun `oversized engine error closes 1009 before response encoding`() {
        val message = "x".repeat(BRIDGE_MAX_MESSAGE_BYTES - BRIDGE_RESPONSE_HEADER_BYTES + 1)
        val server = startServer(FakeEngine(openSessionError = message))
        val socket = authenticated(server)
        socket.sendFrame(Opcode.BINARY, openSessionFrame(9), true)

        val close = socket.readFrame()
        assertEquals(8, close.opcode)
        assertEquals(CloseFrame.TOOBIG, close.closeCode())
    }

    @Test
    fun `draft copy reset and close isolate aggregate counters`() {
        val draft = BridgeDraft6455()
        assertNotSame(draft, draft.copyInstance())
        val socket = WebSocketImpl(testWebSocketListener(), BridgeDraft6455())
        draft.processFrame(
            socket,
            BinaryFrame().apply {
                setPayload(ByteBuffer.wrap(ByteArray(BRIDGE_MAX_MESSAGE_BYTES / 2 + 1)))
                isFin = false
            },
        )
        draft.processFrame(socket, PongFrame().apply { setPayload(ByteBuffer.wrap(byteArrayOf(1))) })
        try {
            draft.processFrame(
                socket,
                ContinuousFrame().apply {
                    setPayload(ByteBuffer.wrap(ByteArray(BRIDGE_MAX_MESSAGE_BYTES / 2)))
                    isFin = true
                },
            )
            fail("aggregate payload unexpectedly accepted")
        } catch (error: InvalidDataException) {
            assertEquals(CloseFrame.TOOBIG, error.closeCode)
        }

        draft.reset()
        val resetDraft = draft.copyInstance() as BridgeDraft6455
        resetDraft.processFrame(
            WebSocketImpl(testWebSocketListener(), BridgeDraft6455()),
            BinaryFrame().apply {
                setPayload(ByteBuffer.wrap(ByteArray(BRIDGE_MAX_MESSAGE_BYTES)))
                isFin = true
            },
        )
        try {
            resetDraft.processFrame(
                WebSocketImpl(testWebSocketListener(), BridgeDraft6455()),
                TextFrame().apply { setPayload(ByteBuffer.wrap(byteArrayOf(1))) },
            )
            fail("text frame unexpectedly accepted")
        } catch (error: InvalidDataException) {
            assertEquals(CloseFrame.PROTOCOL_ERROR, error.closeCode)
        }
    }

    @Test
    fun `server stop drains admitted request before engine close and rejects queued request`() {
        val engine = FakeEngine(blockExecute = true)
        val server = startServer(engine)
        val socket = authenticated(server)
        socket.sendFrame(Opcode.BINARY, executeFrame(1), true)
        assertTrue(engine.executeEntered.await(1, TimeUnit.SECONDS))
        socket.sendFrame(Opcode.BINARY, executeFrame(2), true)

        val stopped = CountDownLatch(1)
        val closer = Thread {
            server.close()
            stopped.countDown()
        }.apply { start() }
        try {
            assertFalse(engine.closeCalled.await(200, TimeUnit.MILLISECONDS))
        } finally {
            engine.executeRelease.countDown()
        }
        assertTrue(stopped.await(3, TimeUnit.SECONDS))
        closer.join(1_000)
        assertEquals(1, engine.executeCalls.get())
        assertTrue(engine.closeCalled.await(0, TimeUnit.MILLISECONDS))
        assertFalse(engine.calledAfterClose.get())
    }

    @Test
    fun `server stop closes partial and upgraded sockets`() {
        val server = startServer()
        val partial = raw(server).also { it.writeAscii("GET / HTTP/1.1\r\n") }
        val upgraded = upgraded(server)
        server.close()
        assertEquals(-1, partial.readByte())
        assertTrue(upgraded.readFrameOrNull()?.opcode == 8 || upgraded.readByte() == -1)
    }

    private fun startServer(engine: FakeEngine = FakeEngine()): GoClientBridgeServer {
        val server = GoClientBridgeServer(engine, TOKEN)
        servers += server
        server.start()
        assertTrue(server.awaitStarted(2_000))
        return server
    }

    private fun raw(server: GoClientBridgeServer, timeoutMillis: Int = 2_500): RawSocket {
        val socket = RawSocket(server.port, timeoutMillis)
        sockets += socket
        return socket
    }

    private fun upgraded(server: GoClientBridgeServer): RawSocket = raw(server).also {
        it.writeAscii(handshakeBytes())
        assertTrue(it.readHttpResponse().startsWith("HTTP/1.1 101"))
    }

    private fun authenticated(server: GoClientBridgeServer): RawSocket = upgraded(server).also {
        it.sendFrame(Opcode.BINARY, authFrame(TOKEN), true)
        val ack = it.readFrame()
        assertEquals(2, ack.opcode)
        assertEquals(GoClientBridgeServer.OP_ACK, ack.payload[0])
    }

    private fun testWebSocketListener(
        onClose: () -> Unit = {},
        onWriteDemand: () -> Unit = {},
    ): WebSocketAdapter = object : WebSocketAdapter() {
        override fun onWebsocketMessage(conn: WebSocket, message: String) = Unit
        override fun onWebsocketMessage(conn: WebSocket, blob: ByteBuffer) = Unit
        override fun onWebsocketOpen(conn: WebSocket, handshake: Handshakedata) = Unit
        override fun onWebsocketClose(conn: WebSocket, code: Int, reason: String, remote: Boolean) = onClose()
        override fun onWebsocketClosing(conn: WebSocket, code: Int, reason: String, remote: Boolean) = Unit
        override fun onWebsocketCloseInitiated(conn: WebSocket, code: Int, reason: String) = Unit
        override fun onWebsocketError(conn: WebSocket, ex: Exception) = Unit
        override fun onWriteDemand(conn: WebSocket) = onWriteDemand()
        override fun getLocalSocketAddress(conn: WebSocket) = null
        override fun getRemoteSocketAddress(conn: WebSocket) = null
    }

    private fun bind(factory: BridgeWebSocketFactory, connection: BridgeWebSocketImpl): SocketChannel =
        SocketChannel.open().also { channel ->
            val key = AttachedSelectionKey(channel, connection)
            connection.setSelectionKey(key)
            connection.setChannel(factory.wrapChannel(channel, key))
        }

    private fun assertRejected(factory: BridgeWebSocketFactory) {
        val rejected = factory.createWebSocket(testWebSocketListener(), listOf(BridgeDraft6455())) as BridgeWebSocketImpl
        SocketChannel.open().use { channel ->
            val key = AttachedSelectionKey(channel, rejected)
            rejected.setSelectionKey(key)
            factory.wrapChannel(channel, key)
            assertFalse(channel.isOpen)
        }
    }

    private class AttachedSelectionKey(
        private val socketChannel: SocketChannel,
        connection: BridgeWebSocketImpl,
    ) : SelectionKey() {
        private var valid = true

        init {
            attach(connection)
        }

        override fun channel(): SelectableChannel = socketChannel
        override fun selector(): Selector = throw UnsupportedOperationException()
        override fun isValid(): Boolean = valid
        override fun cancel() {
            valid = false
        }
        override fun interestOps(): Int = 0
        override fun interestOps(ops: Int): SelectionKey = this
        override fun readyOps(): Int = 0
    }

    private class DeadlineHandshakeServer(
        private val registry: BridgeConnectionRegistry,
        factory: BridgeWebSocketFactory,
        targetCreated: CountDownLatch,
    ) : WebSocketServer(InetSocketAddress("127.0.0.1", 0), 1, listOf(BridgeDraft6455())) {
        private val started = CountDownLatch(1)

        init {
            setWebSocketFactory(object : WebSocketServerFactory {
                override fun createWebSocket(listener: WebSocketAdapter, draft: Draft): WebSocketImpl =
                    factory.createWebSocket(listener, draft).also { targetCreated.countDown() }

                override fun createWebSocket(listener: WebSocketAdapter, drafts: List<Draft>): WebSocketImpl =
                    factory.createWebSocket(listener, drafts).also { targetCreated.countDown() }

                override fun wrapChannel(channel: SocketChannel, key: SelectionKey): ByteChannel =
                    factory.wrapChannel(channel, key)

                override fun close() = factory.close()
            })
        }

        fun awaitStarted(): Boolean = started.await(2, TimeUnit.SECONDS)

        override fun onStart() = started.countDown()
        override fun onOpen(conn: WebSocket, handshake: ClientHandshake) = Unit
        override fun onMessage(conn: WebSocket, message: String) = Unit
        override fun onMessage(conn: WebSocket, message: ByteBuffer) = Unit
        override fun onClose(conn: WebSocket, code: Int, reason: String, remote: Boolean) = Unit
        override fun onError(conn: WebSocket?, ex: Exception?) = Unit

        override fun onWebsocketHandshakeReceivedAsServer(
            conn: WebSocket,
            draft: Draft,
            request: ClientHandshake,
        ): ServerHandshakeBuilder {
            val bridge = conn as? BridgeWebSocketImpl ?: throw InvalidHandshakeException("invalid binding transport")
            if (!registry.acquireUpgrade(bridge)) throw InvalidHandshakeException("binding upgrade limit")
            return super.onWebsocketHandshakeReceivedAsServer(conn, draft, request)
        }
    }

    private class FakeEngine(
        private val blockExecute: Boolean = false,
        private val openSessionError: String? = null,
    ) : GoClientBridgeEngine {
        private val closed = AtomicBoolean(false)
        private val closedSignal = CountDownLatch(1)
        val executeEntered = CountDownLatch(1)
        val executeRelease = CountDownLatch(if (blockExecute) 1 else 0)
        val closeCalled = CountDownLatch(1)
        val executeCalls = AtomicInteger(0)
        val calledAfterClose = AtomicBoolean(false)

        private fun enter() {
            if (closed.get()) calledAfterClose.set(true)
        }

        override fun openSession(payload: ByteArray): Long {
            enter()
            openSessionError?.let { throw IllegalStateException(it) }
            return 1L
        }

        override fun execute(session: Long, payload: ByteArray): Long {
            enter()
            executeCalls.incrementAndGet()
            executeEntered.countDown()
            executeRelease.await()
            return 2L
        }

        override fun openResourceStream(session: Long, payload: ByteArray): Long {
            enter()
            return 3L
        }

        override fun sendResourceStreamFrame(stream: Long, payload: ByteArray) = enter()
        override fun closeResourceStream(stream: Long) = enter()
        override fun engineCommand(payload: ByteArray): Long {
            enter()
            return 4L
        }

        override fun cancel(operation: Long) = enter()
        override fun closeSession(session: Long) = enter()
        override fun release(handle: Long) = enter()

        override fun nextEvent(): ByteArray {
            closedSignal.await()
            throw IllegalStateException("closed")
        }

        override fun close() {
            closed.set(true)
            closedSignal.countDown()
            closeCalled.countDown()
        }
    }

    private class RawSocket(port: Int, timeoutMillis: Int) : Closeable {
        private val socket = Socket("127.0.0.1", port).apply { soTimeout = timeoutMillis }
        private val input = socket.getInputStream()
        private val output = socket.getOutputStream()
        var lastHttpResponse: String = ""
            private set

        fun writeAscii(value: String) {
            output.write(value.toByteArray(StandardCharsets.US_ASCII))
            output.flush()
        }

        fun readByte(): Int = try {
            input.read()
        } catch (_: SocketTimeoutException) {
            Int.MIN_VALUE
        }

        fun readHttpResponse(): String {
            val bytes = ByteArrayOutputStream()
            var delimiter = 0
            while (delimiter != 4) {
                val byte = input.read()
                if (byte < 0) throw EOFException("connection closed before HTTP response")
                bytes.write(byte)
                delimiter = when (delimiter) {
                    0 -> if (byte == 13) 1 else 0
                    1 -> if (byte == 10) 2 else if (byte == 13) 1 else 0
                    2 -> if (byte == 13) 3 else 0
                    3 -> if (byte == 10) 4 else if (byte == 13) 1 else 0
                    else -> 4
                }
            }
            lastHttpResponse = bytes.toString(StandardCharsets.US_ASCII.name())
            return lastHttpResponse
        }

        fun readUntilClose(): String {
            val bytes = ByteArrayOutputStream()
            return try {
                while (true) {
                    val byte = input.read()
                    if (byte < 0) break
                    bytes.write(byte)
                }
                bytes.toString(StandardCharsets.US_ASCII.name())
            } catch (_: SocketTimeoutException) {
                bytes.toString(StandardCharsets.US_ASCII.name())
            }
        }

        fun sendFrame(opcode: Opcode, payload: ByteArray, fin: Boolean) {
            val opcodeValue = when (opcode) {
                Opcode.CONTINUOUS -> 0
                Opcode.TEXT -> 1
                Opcode.BINARY -> 2
                Opcode.PING -> 9
                else -> error("unsupported test opcode")
            }
            output.write((if (fin) 0x80 else 0) or opcodeValue)
            when {
                payload.size <= 125 -> output.write(0x80 or payload.size)
                payload.size <= 0xffff -> {
                    output.write(0x80 or 126)
                    output.write(payload.size ushr 8)
                    output.write(payload.size)
                }
                else -> {
                    output.write(0x80 or 127)
                    repeat(4) { output.write(0) }
                    output.write(payload.size ushr 24)
                    output.write(payload.size ushr 16)
                    output.write(payload.size ushr 8)
                    output.write(payload.size)
                }
            }
            val mask = byteArrayOf(0x13, 0x37, 0x42, 0x55)
            output.write(mask)
            val masked = ByteArray(payload.size) { index -> (payload[index].toInt() xor mask[index % 4].toInt()).toByte() }
            output.write(masked)
            output.flush()
        }

        fun readFrameOrNull(): Frame? = try {
            readFrame()
        } catch (_: EOFException) {
            null
        } catch (_: SocketTimeoutException) {
            null
        }

        fun readFrame(): Frame {
            val first = input.read()
            if (first < 0) throw EOFException()
            val second = input.read()
            if (second < 0) throw EOFException()
            var length = second and 0x7f
            if (length == 126) length = (readRequired() shl 8) or readRequired()
            if (length == 127) {
                repeat(4) { if (readRequired() != 0) fail("unexpected large server frame") }
                length = (readRequired() shl 24) or (readRequired() shl 16) or (readRequired() shl 8) or readRequired()
            }
            val payload = ByteArray(length)
            var offset = 0
            while (offset < payload.size) {
                val read = input.read(payload, offset, payload.size - offset)
                if (read < 0) throw EOFException()
                offset += read
            }
            return Frame(first and 0x0f, payload)
        }

        private fun readRequired(): Int = input.read().also { if (it < 0) throw EOFException() }

        override fun close() = socket.close()
    }

    private data class Frame(val opcode: Int, val payload: ByteArray) {
        fun closeCode(): Int = if (payload.size >= 2) {
            ((payload[0].toInt() and 0xff) shl 8) or (payload[1].toInt() and 0xff)
        } else {
            CloseFrame.NOCODE
        }
    }

    private companion object {
        const val TOKEN = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

        fun authFrame(token: String): ByteArray = byteArrayOf(GoClientBridgeServer.OP_AUTH) + token.toByteArray()

        fun executeFrame(requestId: Long): ByteArray = ByteBuffer.allocate(17).apply {
            put(GoClientBridgeServer.OP_EXECUTE)
            putLong(requestId)
            putLong(1)
        }.array()

        fun openSessionFrame(requestId: Long): ByteArray = ByteBuffer.allocate(9).apply {
            put(GoClientBridgeServer.OP_OPEN_SESSION)
            putLong(requestId)
        }.array()

        fun handshakeBytes(
            path: String = "/",
            origin: String? = "http://localhost",
            protocol: String? = BRIDGE_PROTOCOL,
            extraHeaders: List<String> = emptyList(),
            totalBytes: Int? = null,
        ): String {
            val lines = mutableListOf(
                "GET $path HTTP/1.1",
                "Host: 127.0.0.1",
                "Upgrade: websocket",
                "Connection: Upgrade",
                "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
                "Sec-WebSocket-Version: 13",
            )
            if (origin != null) lines += "Origin: $origin"
            if (protocol != null) lines += "Sec-WebSocket-Protocol: $protocol"
            lines += extraHeaders
            var request = lines.joinToString("\r\n", postfix = "\r\n")
            if (totalBytes != null) {
                val overhead = "X-Pad: \r\n\r\n".length
                val padding = totalBytes - request.length - overhead
                require(padding >= 0)
                request += "X-Pad: ${"x".repeat(padding)}\r\n"
            }
            return request + "\r\n"
        }
    }
}
