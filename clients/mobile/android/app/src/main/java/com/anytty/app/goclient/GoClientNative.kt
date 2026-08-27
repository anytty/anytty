package com.anytty.app.goclient

/** GoClientNative 是 Android 到稳定 C ABI 的薄 JNI 门面，不持有网络、认证、协议或重连状态。 */
object GoClientNative {
    const val ABI_VERSION = 6

    init {
        System.loadLibrary("anytty_client_jni")
    }

    external fun abiVersion(): Int
    external fun create(): Long
    external fun startBridge(engine: Long, token: String): Int
    external fun localProbe(localDiscoveryProto: ByteArray): Boolean
    external fun directProbe(directRouteProto: ByteArray): Boolean
    external fun replaceSupervisorDemand(engine: Long, demandProto: ByteArray)
    external fun signalSupervisor(engine: Long, signalProto: ByteArray)
    external fun awaitSupervisorReady(engine: Long, timeoutMillis: Int)
    external fun supervisorSnapshot(engine: Long): ByteArray
    external fun setDebugLogPath(path: String)
    external fun openSession(engine: Long, requestProto: ByteArray): Long
    external fun execute(engine: Long, session: Long, commandProto: ByteArray): Long
    external fun openResourceStream(engine: Long, session: Long, requestProto: ByteArray): Long
    external fun sendResourceStreamFrame(engine: Long, stream: Long, frameProto: ByteArray)
    external fun closeResourceStream(engine: Long, stream: Long)
    external fun engineCommand(engine: Long, commandProto: ByteArray): Long
    external fun nextEvent(engine: Long, timeoutMillis: Int): ByteArray
    external fun nextPlatformRequest(engine: Long, timeoutMillis: Int): ByteArray
    external fun completePlatformRequest(engine: Long, responseProto: ByteArray)
    external fun cancel(engine: Long, operation: Long)
    external fun closeSession(engine: Long, session: Long)
    external fun release(engine: Long, handle: Long)
    external fun close(engine: Long)
}
