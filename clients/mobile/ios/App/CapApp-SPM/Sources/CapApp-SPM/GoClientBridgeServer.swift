import CryptoKit
import Foundation
import Network

final class GoClientBridgeServer {
    static let protocolName = "anytty.binding.v1"
    static let maxMessageBytes = 4 * 1024 * 1024

    private let engine: IOSGoClientEngine
    private let token: Data
    private let queue = DispatchQueue(label: "com.anytty.ios.bridge")
    private let requestQueue = DispatchQueue(label: "com.anytty.ios.bridge.requests")
    private let eventQueue = DispatchQueue(label: "com.anytty.ios.bridge.events")
    private let pendingCondition = NSCondition()
    private var listener: NWListener?
    private var client: BridgeConnection?
    private var pendingEvents: [Data] = []
    private var active = false
    private var readyResult: Result<UInt16, Error>?
    private let ready = DispatchSemaphore(value: 0)

    init(engine: IOSGoClientEngine, token: String) throws {
        guard let data = token.data(using: .utf8), data.count == 43,
              token.range(of: "^[A-Za-z0-9_-]{43}$", options: .regularExpression) != nil else {
            throw BridgeError.invalidToken
        }
        self.engine = engine
        self.token = data
    }

    func start() throws -> UInt16 {
        let parameters = NWParameters.tcp
        parameters.allowLocalEndpointReuse = false
        parameters.includePeerToPeer = false
        parameters.requiredLocalEndpoint = .hostPort(host: "127.0.0.1", port: .any)
        let listener = try NWListener(using: parameters, on: .any)
        self.listener = listener
        active = true
        listener.stateUpdateHandler = { [weak self] state in self?.listenerStateChanged(state) }
        listener.newConnectionHandler = { [weak self] connection in self?.accept(connection) }
        listener.start(queue: queue)
        guard ready.wait(timeout: .now() + 5) == .success else {
            stop()
            throw BridgeError.startTimeout
        }
        switch readyResult {
        case .success(let port):
            eventQueue.async { [weak self] in self?.pumpEvents() }
            return port
        case .failure(let error): throw error
        case nil: throw BridgeError.startFailed
        }
    }

    func stop() {
        queue.sync {
            guard active else { return }
            active = false
            client?.close(code: 1001, reason: "bridge stopped")
            client = nil
            listener?.cancel()
            listener = nil
        }
        pendingCondition.lock()
        pendingCondition.broadcast()
        pendingCondition.unlock()
    }

    private func listenerStateChanged(_ state: NWListener.State) {
        switch state {
        case .ready:
            guard let port = listener?.port?.rawValue else {
                readyResult = .failure(BridgeError.startFailed)
                ready.signal()
                return
            }
            readyResult = .success(port)
            ready.signal()
        case .failed(let error):
            if readyResult == nil {
                readyResult = .failure(error)
                ready.signal()
            }
        default: break
        }
    }

    private func accept(_ connection: NWConnection) {
        guard active, client == nil else {
            connection.cancel()
            return
        }
        let accepted = BridgeConnection(connection: connection, server: self, queue: queue)
        client = accepted
        accepted.start()
    }

    fileprivate func connectionClosed(_ connection: BridgeConnection) {
        if client === connection { client = nil }
    }

    fileprivate func authenticated(_ connection: BridgeConnection, frame: Data) -> Bool {
        guard client === connection, frame.count == 44, frame.first == 0x01 else { return false }
        let candidate = frame.dropFirst()
        guard constantTimeEqual(Data(candidate), token) else { return false }
        sendResponse(to: connection, operation: 0x21, requestID: 0, handle: 0, payload: Data())
        pendingCondition.lock()
        let events = pendingEvents
        pendingEvents.removeAll(keepingCapacity: true)
        pendingCondition.broadcast()
        pendingCondition.unlock()
        for event in events {
            sendResponse(to: connection, operation: 0x30, requestID: 0, handle: 0, payload: event)
        }
        return true
    }

    fileprivate func received(_ connection: BridgeConnection, frame: Data) {
        guard frame.count >= 9, frame.count <= Self.maxMessageBytes else {
            connection.close(code: 1009, reason: "binding message exceeds limit")
            return
        }
        let operation = frame[0]
        let requestID = frame.uint64(at: 1)
        requestQueue.async { [weak self, weak connection] in
            guard let self, let connection else { return }
            do {
                switch operation {
                case 0x10:
                    let handle = try GoClientNative.openSession(engine: self.engine.handle, payload: frame.subdata(in: 9..<frame.count))
                    self.accepted(connection, requestID: requestID, handle: handle)
                case 0x11:
                    try self.requireHandleFrame(frame)
                    let handle = try GoClientNative.execute(
                        engine: self.engine.handle,
                        session: frame.uint64(at: 9),
                        payload: frame.subdata(in: 17..<frame.count)
                    )
                    self.accepted(connection, requestID: requestID, handle: handle)
                case 0x12:
                    let handle = try GoClientNative.engineCommand(engine: self.engine.handle, payload: frame.subdata(in: 9..<frame.count))
                    self.accepted(connection, requestID: requestID, handle: handle)
                case 0x14:
                    try self.requireHandleFrame(frame)
                    try GoClientNative.cancel(engine: self.engine.handle, operation: frame.uint64(at: 9))
                    self.acknowledged(connection, requestID: requestID)
                case 0x15:
                    try self.requireHandleFrame(frame)
                    try GoClientNative.closeSession(engine: self.engine.handle, session: frame.uint64(at: 9))
                    self.acknowledged(connection, requestID: requestID)
                case 0x16:
                    try self.requireHandleFrame(frame)
                    try GoClientNative.release(engine: self.engine.handle, handle: frame.uint64(at: 9))
                    self.acknowledged(connection, requestID: requestID)
                case 0x17:
                    try self.requireHandleFrame(frame)
                    let handle = try GoClientNative.openResourceStream(
                        engine: self.engine.handle,
                        session: frame.uint64(at: 9),
                        payload: frame.subdata(in: 17..<frame.count)
                    )
                    self.accepted(connection, requestID: requestID, handle: handle)
                case 0x18:
                    try self.requireHandleFrame(frame)
                    try GoClientNative.sendResourceStreamFrame(
                        engine: self.engine.handle,
                        stream: frame.uint64(at: 9),
                        payload: frame.subdata(in: 17..<frame.count)
                    )
                    self.acknowledged(connection, requestID: requestID)
                case 0x19:
                    try self.requireHandleFrame(frame)
                    try GoClientNative.closeResourceStream(engine: self.engine.handle, stream: frame.uint64(at: 9))
                    self.acknowledged(connection, requestID: requestID)
                default:
                    self.failed(connection, requestID: requestID, message: "unsupported binding operation")
                }
            } catch {
                self.failed(connection, requestID: requestID, message: error.localizedDescription)
            }
        }
    }

    private func requireHandleFrame(_ frame: Data) throws {
        guard frame.count >= 17 else { throw BridgeError.truncatedRequest }
    }

    private func pumpEvents() {
        while isActive {
            do {
                let event = try GoClientNative.nextEvent(engine: engine.handle)
                queue.async { [weak self] in self?.deliver(event) }
            } catch {
                return
            }
        }
    }

    private var isActive: Bool {
        queue.sync { active }
    }

    private func deliver(_ event: Data) {
        guard active else { return }
        if let client, client.isAuthenticated {
            sendResponse(to: client, operation: 0x30, requestID: 0, handle: 0, payload: event)
            return
        }
        pendingCondition.lock()
        if pendingEvents.count < 256 { pendingEvents.append(event) }
        pendingCondition.unlock()
    }

    private func accepted(_ connection: BridgeConnection, requestID: UInt64, handle: UInt64) {
        sendResponse(to: connection, operation: 0x20, requestID: requestID, handle: handle, payload: Data())
    }

    private func acknowledged(_ connection: BridgeConnection, requestID: UInt64) {
        sendResponse(to: connection, operation: 0x21, requestID: requestID, handle: 0, payload: Data())
    }

    private func failed(_ connection: BridgeConnection, requestID: UInt64, message: String) {
        let payload = Data(message.utf8)
        guard payload.count <= Self.maxMessageBytes - 21 else {
            connection.close(code: 1009, reason: "binding message exceeds limit")
            return
        }
        sendResponse(to: connection, operation: 0x22, requestID: requestID, handle: 0, payload: payload)
    }

    private func sendResponse(
        to connection: BridgeConnection,
        operation: UInt8,
        requestID: UInt64,
        handle: UInt64,
        payload: Data
    ) {
        guard payload.count <= Self.maxMessageBytes - 21 else {
            connection.close(code: 1009, reason: "binding message exceeds limit")
            return
        }
        var data = Data([operation])
        data.appendBigEndian(requestID)
        data.appendBigEndian(handle)
        data.appendBigEndian(UInt32(payload.count))
        data.append(payload)
        connection.sendBinary(data)
    }
}

private enum BridgeError: Error {
    case invalidToken
    case startTimeout
    case startFailed
    case truncatedRequest
}

private func constantTimeEqual(_ left: Data, _ right: Data) -> Bool {
    guard left.count == right.count else { return false }
    var difference: UInt8 = 0
    for index in left.indices { difference |= left[index] ^ right[index] }
    return difference == 0
}

final class BridgeConnection {
    private let connection: NWConnection
    private unowned let server: GoClientBridgeServer
    private let queue: DispatchQueue
    private var buffer = Data()
    private var handshakeComplete = false
    private var authenticated = false
    private var fragmented = Data()
    private var fragmentOpcode: UInt8?
    private var closed = false
    private var authDeadline: DispatchWorkItem?

    var isAuthenticated: Bool { authenticated && !closed }

    init(connection: NWConnection, server: GoClientBridgeServer, queue: DispatchQueue) {
        self.connection = connection
        self.server = server
        self.queue = queue
    }

    func start() {
        connection.stateUpdateHandler = { [weak self] state in
            guard let self else { return }
            if case .failed = state { self.finish() }
            if case .cancelled = state { self.finish() }
        }
        connection.start(queue: queue)
        receive()
        let deadline = DispatchWorkItem { [weak self] in
            guard let self, !self.authenticated else { return }
            self.close(code: 1008, reason: "authentication timed out")
        }
        authDeadline = deadline
        queue.asyncAfter(deadline: .now() + 2, execute: deadline)
    }

    func sendBinary(_ payload: Data) {
        guard !closed else { return }
        sendFrame(opcode: 0x02, payload: payload)
    }

    func close(code: UInt16, reason: String) {
        guard !closed else { return }
        var payload = Data()
        payload.appendBigEndian(code)
        payload.append(Data(reason.utf8.prefix(120)))
        sendFrame(opcode: 0x08, payload: payload)
        finish()
    }

    private func receive() {
        connection.receive(minimumIncompleteLength: 1, maximumLength: 64 * 1024) { [weak self] data, _, complete, error in
            guard let self, !self.closed else { return }
            if let data, !data.isEmpty {
                self.buffer.append(data)
                do { try self.consume() } catch { self.close(code: 1002, reason: "invalid binding transport") }
            }
            if complete || error != nil { self.finish(); return }
            self.receive()
        }
    }

    private func consume() throws {
        if !handshakeComplete {
            guard let delimiter = buffer.range(of: Data("\r\n\r\n".utf8)) else {
                if buffer.count > 16 * 1024 { throw BridgeError.startFailed }
                return
            }
            let header = buffer.subdata(in: 0..<delimiter.upperBound)
            buffer.removeSubrange(0..<delimiter.upperBound)
            try handshake(header)
            handshakeComplete = true
        }
        while let frame = try nextFrame() { try handle(frame) }
    }

    private func handshake(_ data: Data) throws {
        guard let value = String(data: data, encoding: .utf8) else { throw BridgeError.startFailed }
        let lines = value.components(separatedBy: "\r\n")
        guard lines.first == "GET / HTTP/1.1" else { throw BridgeError.startFailed }
        var headers: [String: String] = [:]
        for line in lines.dropFirst() {
            guard let separator = line.firstIndex(of: ":") else { continue }
            let name = line[..<separator].lowercased()
            let value = line[line.index(after: separator)...].trimmingCharacters(in: .whitespaces)
            headers[name] = value
        }
        let allowedOrigins = ["capacitor://localhost", "http://localhost", "https://localhost"]
        guard headers["upgrade"]?.lowercased() == "websocket",
              headers["connection"]?.lowercased().contains("upgrade") == true,
              headers["sec-websocket-version"] == "13",
              headers["sec-websocket-protocol"]?.split(separator: ",").map({ $0.trimmingCharacters(in: .whitespaces) }).contains(GoClientBridgeServer.protocolName) == true,
              allowedOrigins.contains(headers["origin"] ?? ""),
              let key = headers["sec-websocket-key"],
              Data(base64Encoded: key)?.count == 16 else {
            throw BridgeError.startFailed
        }
        let accept = Data(Insecure.SHA1.hash(data: Data((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").utf8))).base64EncodedString()
        let response = "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: \(accept)\r\nSec-WebSocket-Protocol: \(GoClientBridgeServer.protocolName)\r\n\r\n"
        connection.send(content: Data(response.utf8), completion: .contentProcessed { _ in })
    }

    private struct Frame { let final: Bool; let opcode: UInt8; let payload: Data }

    private func nextFrame() throws -> Frame? {
        guard buffer.count >= 2 else { return nil }
        let first = buffer[0]
        let second = buffer[1]
        guard first & 0x70 == 0, second & 0x80 != 0 else { throw BridgeError.startFailed }
        var length = UInt64(second & 0x7f)
        var offset = 2
        if length == 126 {
            guard buffer.count >= 4 else { return nil }
            length = UInt64(buffer.uint16(at: 2)); offset = 4
        } else if length == 127 {
            guard buffer.count >= 10 else { return nil }
            length = buffer.uint64(at: 2); offset = 10
        }
        guard length <= UInt64(GoClientBridgeServer.maxMessageBytes), length <= UInt64(Int.max) else { throw BridgeError.startFailed }
        guard buffer.count >= offset + 4 + Int(length) else { return nil }
        let mask = Array(buffer[offset..<(offset + 4)])
        offset += 4
        var payload = Data(buffer[offset..<(offset + Int(length))])
        for index in payload.indices { payload[index] ^= mask[index % 4] }
        buffer.removeSubrange(0..<(offset + Int(length)))
        return Frame(final: first & 0x80 != 0, opcode: first & 0x0f, payload: payload)
    }

    private func handle(_ frame: Frame) throws {
        switch frame.opcode {
        case 0x00:
            guard fragmentOpcode != nil else { throw BridgeError.startFailed }
            fragmented.append(frame.payload)
            guard fragmented.count <= GoClientBridgeServer.maxMessageBytes else { throw BridgeError.startFailed }
            if frame.final {
                let opcode = fragmentOpcode!
                let payload = fragmented
                fragmentOpcode = nil
                fragmented.removeAll(keepingCapacity: true)
                try handleMessage(opcode: opcode, payload: payload)
            }
        case 0x01, 0x02:
            guard fragmentOpcode == nil else { throw BridgeError.startFailed }
            if frame.final { try handleMessage(opcode: frame.opcode, payload: frame.payload) }
            else { fragmentOpcode = frame.opcode; fragmented = frame.payload }
        case 0x08: finish()
        case 0x09: sendFrame(opcode: 0x0a, payload: frame.payload)
        case 0x0a: break
        default: throw BridgeError.startFailed
        }
    }

    private func handleMessage(opcode: UInt8, payload: Data) throws {
        guard opcode == 0x02 else { throw BridgeError.startFailed }
        if !authenticated {
            authenticated = server.authenticated(self, frame: payload)
            guard authenticated else { close(code: 1008, reason: "authentication failed"); return }
            authDeadline?.cancel()
            authDeadline = nil
            return
        }
        server.received(self, frame: payload)
    }

    private func sendFrame(opcode: UInt8, payload: Data) {
        var frame = Data([0x80 | opcode])
        if payload.count < 126 {
            frame.append(UInt8(payload.count))
        } else if payload.count <= Int(UInt16.max) {
            frame.append(126); frame.appendBigEndian(UInt16(payload.count))
        } else {
            frame.append(127); frame.appendBigEndian(UInt64(payload.count))
        }
        frame.append(payload)
        connection.send(content: frame, completion: .contentProcessed { _ in })
    }

    private func finish() {
        guard !closed else { return }
        closed = true
        authDeadline?.cancel()
        connection.cancel()
        server.connectionClosed(self)
    }
}

private extension Data {
    func uint16(at offset: Int) -> UInt16 {
        self[offset..<(offset + 2)].reduce(0) { ($0 << 8) | UInt16($1) }
    }

    func uint64(at offset: Int) -> UInt64 {
        self[offset..<(offset + 8)].reduce(0) { ($0 << 8) | UInt64($1) }
    }

    mutating func appendBigEndian<T: FixedWidthInteger>(_ value: T) {
        var big = value.bigEndian
        Swift.withUnsafeBytes(of: &big) { append(contentsOf: $0) }
    }
}
