import AnyTTYClient
import Foundation

enum GoClientNativeError: Error, LocalizedError {
    case status(Int32)
    case invalidBuffer

    var errorDescription: String? {
        switch self {
        case .status(let value): return "AnyTTY native status \(value)"
        case .invalidBuffer: return "AnyTTY native returned an invalid buffer"
        }
    }
}

enum GoClientNative {
    static func validateABI() throws {
        guard anytty_client_abi_version() == ANYTTY_CLIENT_ABI_VERSION else {
            throw GoClientNativeError.status(-1)
        }
    }

    static func create() throws -> UInt64 {
        try validateABI()
        var handle: UInt64 = 0
        try check(anytty_engine_create(&handle))
        return handle
    }

    static func startBridge(engine: UInt64, token: String) throws -> UInt16 {
        var port: UInt16 = 0
        try withPayload(Data(token.utf8)) { bytes, count in
            try check(anytty_bridge_start(engine, bytes, count, &port))
        }
        return port
    }

    static func localProbe(_ payload: Data) throws -> Bool {
        var reachable: UInt8 = 0
        try withPayload(payload) { bytes, count in
            try check(anytty_local_probe(bytes, count, &reachable))
        }
        return reachable != 0
    }

    static func directProbe(_ payload: Data) throws -> Bool {
        var reachable: UInt8 = 0
        try withPayload(payload) { bytes, count in
            try check(anytty_direct_probe(bytes, count, &reachable))
        }
        return reachable != 0
    }

    static func openSession(engine: UInt64, payload: Data) throws -> UInt64 {
        try operation(payload) { bytes, count, output in
            anytty_engine_open_session(engine, bytes, count, output)
        }
    }

    static func execute(engine: UInt64, session: UInt64, payload: Data) throws -> UInt64 {
        try operation(payload) { bytes, count, output in
            anytty_engine_execute(engine, session, bytes, count, output)
        }
    }

    static func openResourceStream(engine: UInt64, session: UInt64, payload: Data) throws -> UInt64 {
        try operation(payload) { bytes, count, output in
            anytty_engine_open_resource_stream(engine, session, bytes, count, output)
        }
    }

    static func sendResourceStreamFrame(engine: UInt64, stream: UInt64, payload: Data) throws {
        try withPayload(payload) { bytes, count in
            try check(anytty_engine_send_resource_stream_frame(engine, stream, bytes, count))
        }
    }

    static func closeResourceStream(engine: UInt64, stream: UInt64) throws {
        try check(anytty_engine_close_resource_stream(engine, stream))
    }

    static func engineCommand(engine: UInt64, payload: Data) throws -> UInt64 {
        try operation(payload) { bytes, count, output in
            anytty_engine_command(engine, bytes, count, output)
        }
    }

    static func nextEvent(engine: UInt64) throws -> Data {
        try readBuffer { anytty_engine_next_event(engine, 0, $0) }
    }

    static func nextPlatformRequest(engine: UInt64) throws -> Data {
        try readBuffer { anytty_platform_next_request(engine, 0, $0) }
    }

    static func completePlatformRequest(engine: UInt64, payload: Data) throws {
        try withPayload(payload) { bytes, count in
            try check(anytty_platform_complete(engine, bytes, count))
        }
    }

    static func cancel(engine: UInt64, operation: UInt64) throws {
        try check(anytty_engine_cancel(engine, operation))
    }

    static func closeSession(engine: UInt64, session: UInt64) throws {
        try check(anytty_engine_close_session(engine, session))
    }

    static func release(engine: UInt64, handle: UInt64) throws {
        try check(anytty_engine_release(engine, handle))
    }

    static func close(engine: UInt64) throws {
        try check(anytty_engine_close(engine))
    }

    private static func operation(
        _ payload: Data,
        invoke: (UnsafePointer<UInt8>?, Int, UnsafeMutablePointer<UInt64>) -> anytty_status_v1
    ) throws -> UInt64 {
        var output: UInt64 = 0
        try withPayload(payload) { bytes, count in
            try check(invoke(bytes, count, &output))
        }
        return output
    }

    private static func withPayload<T>(
        _ payload: Data,
        body: (UnsafePointer<UInt8>?, Int) throws -> T
    ) rethrows -> T {
        try payload.withUnsafeBytes { raw in
            try body(raw.bindMemory(to: UInt8.self).baseAddress, payload.count)
        }
    }

    private static func readBuffer(
        invoke: (UnsafeMutablePointer<anytty_buffer_v1>) -> anytty_status_v1
    ) throws -> Data {
        var buffer = anytty_buffer_v1()
        try check(invoke(&buffer))
        guard buffer.length == 0 || buffer.data != nil else {
            if buffer.buffer_handle != 0 { _ = anytty_buffer_free(buffer.buffer_handle) }
            throw GoClientNativeError.invalidBuffer
        }
        let data = buffer.length == 0 ? Data() : Data(bytes: buffer.data!, count: Int(buffer.length))
        try check(anytty_buffer_free(buffer.buffer_handle))
        return data
    }

    private static func check(_ status: anytty_status_v1) throws {
        guard status == ANYTTY_STATUS_OK else {
            throw GoClientNativeError.status(Int32(status.rawValue))
        }
    }
}
