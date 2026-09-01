import Darwin
import Foundation
import Network
import Security
import SwiftProtobuf
import UIKit

struct IOSNativeNetworkSnapshot {
    let epoch: UInt64
    let connected: Bool
    let reason: String
}

struct IOSNetworkPathDescriptor: Equatable {
    let connected: Bool
    let fingerprint: String
}

func iosNetworkTransitionReason(
    from previous: IOSNetworkPathDescriptor,
    to next: IOSNetworkPathDescriptor
) -> String? {
    guard previous != next else { return nil }
    if !next.connected { return "offline" }
    if !previous.connected { return "available" }
    return "network_replaced"
}

func replayIOSSupervisorState(
    signalHost: () throws -> Void,
    submitDemand: () throws -> Void
) throws {
    try signalHost()
    try submitDemand()
}

func runIOSPairingResetTransaction(
    clearPersistentState: () throws -> Void,
    replaceRuntime: () throws -> Void,
    stopRuntimeAfterFailure: () -> Void
) throws {
    var replacementCompleted = false
    defer {
        if !replacementCompleted { stopRuntimeAfterFailure() }
    }
    try clearPersistentState()
    try replaceRuntime()
    replacementCompleted = true
}

final class IOSForegroundResumeLease {}

final class IOSForegroundResumeFence {
    private let lock = NSLock()
    private var currentLease: IOSForegroundResumeLease?

    init(isForeground: Bool) {
        currentLease = isForeground ? IOSForegroundResumeLease() : nil
    }

    func enterForeground() -> IOSForegroundResumeLease {
        let lease = IOSForegroundResumeLease()
        lock.lock()
        currentLease = lease
        lock.unlock()
        return lease
    }

    func enterBackground() {
        lock.lock()
        currentLease = nil
        lock.unlock()
    }

    func capture() -> IOSForegroundResumeLease? {
        lock.lock()
        defer { lock.unlock() }
        return currentLease
    }

    func accepts(_ lease: IOSForegroundResumeLease) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return currentLease === lease
    }

    func performIfAccepted<T>(
        _ lease: IOSForegroundResumeLease,
        operation: () throws -> T
    ) rethrows -> T? {
        lock.lock()
        defer { lock.unlock() }
        guard currentLease === lease else { return nil }
        return try operation()
    }
}

struct IOSRendererDemandSnapshot {
    let attachmentID: String
    let demandRevision: UInt64
    let stopEpoch: UInt64
    let endpointIDs: Set<String>
    let stopped: Bool
}

enum IOSRendererDemandResumeOutcome: String {
    case resumed
    case stopped
}

struct IOSRendererDemandResumeResult {
    let snapshot: IOSRendererDemandSnapshot
    let outcome: IOSRendererDemandResumeOutcome
}

private struct IOSRendererDemandResumeRecord {
    let stopEpoch: UInt64
    let accepted: Bool
}

enum IOSConnectionRuntimeEvent {
    case generation(event: String, reason: String, epoch: UInt64)
    case network(IOSNativeNetworkSnapshot)
}

final class IOSRendererDemandState {
    private static let maxRetainedResumeIntents = 4096
    private let attachmentIDFactory: () -> String
    private var attachmentID: String?
    private var demandRevision: UInt64 = 0
    private var stopEpoch: UInt64 = 0
    private var endpointIDs = Set<String>()
    private var stopped = false
    private var resumeIntents = [String: IOSRendererDemandResumeRecord]()

    init(attachmentIDFactory: @escaping () -> String = { UUID().uuidString }) {
        self.attachmentIDFactory = attachmentIDFactory
    }

    func attachRenderer() -> IOSRendererDemandSnapshot {
        let nextAttachmentID = attachmentIDFactory().trimmingCharacters(in: .whitespacesAndNewlines)
        precondition(!nextAttachmentID.isEmpty, "renderer attachment ID is empty")
        precondition(nextAttachmentID != attachmentID, "renderer attachment ID was reused")
        attachmentID = nextAttachmentID
        resumeIntents.removeAll(keepingCapacity: true)
        return snapshot(attachmentID: nextAttachmentID)
    }

    func rotateRenderer(attachmentID candidate: String) throws -> IOSRendererDemandSnapshot {
        guard !candidate.isEmpty, attachmentID == candidate else {
            throw IOSConnectionRuntimeError.staleRenderer
        }
        return attachRenderer()
    }

    func detachRenderer(attachmentID candidate: String) {
        if attachmentID == candidate { attachmentID = nil }
    }

    func replaceDemand(
        attachmentID candidate: String,
        baseDemandRevision: UInt64,
        endpointIDs replacement: Set<String>
    ) throws -> IOSRendererDemandSnapshot {
        guard !candidate.isEmpty, attachmentID == candidate else {
            throw IOSConnectionRuntimeError.staleRenderer
        }
        guard baseDemandRevision == demandRevision else {
            throw IOSConnectionRuntimeError.staleDemandRevision
        }
        if stopped, !replacement.isEmpty {
            throw IOSConnectionRuntimeError.demandStopped
        }
        try incrementDemandRevision()
        endpointIDs = replacement
        return snapshot(attachmentID: candidate)
    }

    func resumeDemand(
        attachmentID candidate: String,
        intentID: String,
        baseStopEpoch: UInt64
    ) throws -> IOSRendererDemandResumeResult {
        guard !candidate.isEmpty, attachmentID == candidate else {
            throw IOSConnectionRuntimeError.staleRenderer
        }
        let normalizedIntentID = intentID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedIntentID.isEmpty, normalizedIntentID.count <= 128 else {
            throw IOSConnectionRuntimeError.invalidResumeIntent
        }
        if let existing = resumeIntents[normalizedIntentID] {
            return IOSRendererDemandResumeResult(
                snapshot: snapshot(attachmentID: candidate),
                outcome: existing.accepted && existing.stopEpoch == stopEpoch ? .resumed : .stopped
            )
        }
        guard resumeIntents.count < Self.maxRetainedResumeIntents else {
            throw IOSConnectionRuntimeError.resumeIntentCapacityExhausted
        }
        if baseStopEpoch != stopEpoch {
            resumeIntents[normalizedIntentID] = IOSRendererDemandResumeRecord(
                stopEpoch: baseStopEpoch,
                accepted: false
            )
            return IOSRendererDemandResumeResult(
                snapshot: snapshot(attachmentID: candidate),
                outcome: .stopped
            )
        }
        if stopped {
            try incrementDemandRevision()
            stopped = false
        }
        resumeIntents[normalizedIntentID] = IOSRendererDemandResumeRecord(
            stopEpoch: stopEpoch,
            accepted: true
        )
        return IOSRendererDemandResumeResult(
            snapshot: snapshot(attachmentID: candidate),
            outcome: .resumed
        )
    }

    @discardableResult
    func clearDemandForUserStop() throws -> IOSRendererDemandSnapshot? {
        try incrementDemandRevision()
        stopEpoch = demandRevision
        endpointIDs.removeAll()
        stopped = true
        return attachmentID.map(snapshot)
    }

    func currentSnapshot(attachmentID candidate: String) throws -> IOSRendererDemandSnapshot {
        guard !candidate.isEmpty, attachmentID == candidate else {
            throw IOSConnectionRuntimeError.staleRenderer
        }
        return snapshot(attachmentID: candidate)
    }

    func canonicalSnapshot() -> IOSRendererDemandSnapshot {
        snapshot(attachmentID: attachmentID ?? "process-retained-demand")
    }

    private func snapshot(attachmentID: String) -> IOSRendererDemandSnapshot {
        IOSRendererDemandSnapshot(
            attachmentID: attachmentID,
            demandRevision: demandRevision,
            stopEpoch: stopEpoch,
            endpointIDs: endpointIDs,
            stopped: stopped
        )
    }

    private func incrementDemandRevision() throws {
        guard demandRevision < UInt64.max else {
            throw IOSConnectionRuntimeError.revisionExhausted
        }
        demandRevision += 1
    }
}

enum IOSConnectionRuntimeError: Error, LocalizedError, Equatable {
    case staleRenderer
    case staleForegroundResume
    case staleDemandRevision
    case demandStopped
    case invalidResumeIntent
    case resumeIntentCapacityExhausted
    case endpointNotDemanded
    case revisionExhausted
    case unavailable

    var errorDescription: String? {
        switch self {
        case .staleRenderer: return "renderer attachment is stale"
        case .staleForegroundResume: return "foreground resume acknowledgement is stale"
        case .staleDemandRevision: return "renderer demand revision is stale"
        case .demandStopped: return "renderer demand is stopped until explicitly resumed"
        case .invalidResumeIntent: return "renderer resume intent ID is invalid"
        case .resumeIntentCapacityExhausted: return "renderer resume intent capacity is exhausted"
        case .endpointNotDemanded: return "endpoint is not demanded"
        case .revisionExhausted: return "native connection revision is exhausted"
        case .unavailable: return "native connection runtime is unavailable"
        }
    }
}

/// Process-owned iOS connection runtime. Renderer replacement only fences the old
/// attachment; it does not release endpoint demand or the Go physical sessions.
final class IOSConnectionRuntimeOwner {
    static let shared = IOSConnectionRuntimeOwner()

    private static let bridgeTokenBytes = 32
    private static let runtimeRebuildBackoffSeconds = [1, 2, 4, 8, 15]

    private let queue = DispatchQueue(label: "com.anytty.ios.connection-runtime")
    private let pathQueue = DispatchQueue(label: "com.anytty.ios.connection-path")
    private let accessCredentials = IOSClientAccessCredentialStore()
    private let sshCredentials = IOSSSHCredentialStore()
    private let endpointRegistry = IOSEndpointRegistryStore()
    private let rendererDemand = IOSRendererDemandState()
    private let pathMonitor = NWPathMonitor()
    private let foregroundFence: IOSForegroundResumeFence

    private var engine: IOSGoClientEngine?
    private var port: UInt16 = 0
    private var token = ""
    private var engineGeneration: UInt64 = 0
    private var runtimeHealthy = false
    private var rebuildScheduleEpoch: UInt64 = 0
    private var scheduledRebuildEpoch: UInt64?
    private var rebuildBackoffIndex = 0
    private var generationEventEpoch: UInt64 = 0

    private var hostRevision: UInt64 = 1
    private weak var lastSignaledForegroundLease: IOSForegroundResumeLease?
    private var networkEpoch: UInt64 = 1
    private var networkPath: IOSNetworkPathDescriptor
    private var networkReason: String
    private var pendingNetworkChange: DispatchWorkItem?
    private var foregroundObserver: NSObjectProtocol?
    private var backgroundObserver: NSObjectProtocol?

    private var eventSinkAttachmentID: String?
    private var eventSink: ((IOSConnectionRuntimeEvent) -> Void)?

    private init() {
        foregroundFence = IOSForegroundResumeFence(
            isForeground: UIApplication.shared.applicationState != .background
        )
        let initialPath = pathMonitor.currentPath
        let initialNetworkPath = Self.descriptor(for: initialPath)
        networkPath = initialNetworkPath
        networkReason = initialNetworkPath.connected ? "path_changed" : "offline"

        pathMonitor.pathUpdateHandler = { [weak self] path in
            self?.queue.async { self?.scheduleNetworkChange(path) }
        }
        pathMonitor.start(queue: pathQueue)
        foregroundObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.willEnterForegroundNotification,
            object: nil,
            queue: nil
        ) { [weak self] _ in
            guard let self else { return }
            let lease = self.foregroundFence.enterForeground()
            self.queue.async { [weak self] in self?.handleNativeForegroundResume(lease: lease) }
        }
        backgroundObserver = NotificationCenter.default.addObserver(
            forName: UIApplication.didEnterBackgroundNotification,
            object: nil,
            queue: nil
        ) { [weak self] _ in
            self?.foregroundFence.enterBackground()
        }
    }

    deinit {
        pathMonitor.cancel()
        if let foregroundObserver { NotificationCenter.default.removeObserver(foregroundObserver) }
        if let backgroundObserver { NotificationCenter.default.removeObserver(backgroundObserver) }
        pendingNetworkChange?.cancel()
        engine?.close()
    }

    func attachRenderer(
        eventSink: @escaping (IOSConnectionRuntimeEvent) -> Void
    ) -> IOSRendererDemandSnapshot {
        queue.sync {
            let snapshot = rendererDemand.attachRenderer()
            eventSinkAttachmentID = snapshot.attachmentID
            self.eventSink = eventSink
            do {
                try ensureRuntimeLocked(reason: "renderer_attached")
                try submitDemandLocked(snapshot)
            } catch {
                markRuntimeUnhealthyLocked(reason: "renderer_attach_failed")
            }
            return snapshot
        }
    }

    func detachRenderer(attachmentID: String) {
        queue.sync {
            rendererDemand.detachRenderer(attachmentID: attachmentID)
            if eventSinkAttachmentID == attachmentID {
                eventSinkAttachmentID = nil
                eventSink = nil
            }
        }
    }

    func rotateRenderer(attachmentID: String) throws -> IOSRendererDemandSnapshot {
        try queue.sync {
            let snapshot = try rendererDemand.rotateRenderer(attachmentID: attachmentID)
            if eventSinkAttachmentID == attachmentID {
                eventSinkAttachmentID = snapshot.attachmentID
            }
            return snapshot
        }
    }

    func currentDemand(attachmentID: String) throws -> IOSRendererDemandSnapshot {
        try queue.sync { try rendererDemand.currentSnapshot(attachmentID: attachmentID) }
    }

    func replaceDemand(
        attachmentID: String,
        baseDemandRevision: UInt64,
        endpointIDs: Set<String>
    ) throws -> IOSRendererDemandSnapshot {
        try queue.sync {
            let snapshot = try rendererDemand.replaceDemand(
                attachmentID: attachmentID,
                baseDemandRevision: baseDemandRevision,
                endpointIDs: endpointIDs
            )
            do {
                try ensureRuntimeLocked(reason: "demand_reconcile")
                try submitDemandLocked(snapshot)
            } catch {
                // The process-owned desired set remains authoritative and will be
                // replayed into the next healthy engine generation.
                markRuntimeUnhealthyLocked(reason: "demand_submit_failed")
                throw error
            }
            return snapshot
        }
    }

    func resumeDemand(
        attachmentID: String,
        intentID: String,
        baseStopEpoch: UInt64
    ) throws -> IOSRendererDemandResumeResult {
        try queue.sync {
            try rendererDemand.resumeDemand(
                attachmentID: attachmentID,
                intentID: intentID,
                baseStopEpoch: baseStopEpoch
            )
        }
    }

    func networkSnapshot() -> IOSNativeNetworkSnapshot {
        queue.sync {
            IOSNativeNetworkSnapshot(epoch: networkEpoch, connected: networkPath.connected, reason: networkReason)
        }
    }

    func ensureRuntime() throws {
        try queue.sync { try ensureRuntimeLocked(reason: "runtime_requested") }
    }

    func bridgeEndpoint() throws -> (port: UInt16, token: String) {
        try queue.sync {
            try ensureRuntimeLocked(reason: "bridge_requested")
            guard port > 0, !token.isEmpty else { throw IOSConnectionRuntimeError.unavailable }
            return (port, token)
        }
    }

    func captureForegroundResumeLease() -> IOSForegroundResumeLease? {
        foregroundFence.capture()
    }

    func handleForegroundResume(lease: IOSForegroundResumeLease?) throws {
        guard let lease else { return }
        try queue.sync {
            guard foregroundFence.accepts(lease) else { return }
            do {
                _ = try beginForegroundResumeLocked(lease: lease)
            } catch IOSConnectionRuntimeError.staleForegroundResume {
                return
            } catch {
                markRuntimeUnhealthyLocked(reason: "foreground_resume_failed")
                throw error
            }
        }
    }

    func requestEndpointRecovery(endpointID: String) throws {
        let endpointID = endpointID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !endpointID.isEmpty else { throw IOSConnectionRuntimeError.unavailable }
        try queue.sync {
            guard rendererDemand.canonicalSnapshot().endpointIDs.contains(endpointID) else {
                throw IOSConnectionRuntimeError.endpointNotDemanded
            }
            do {
                try ensureRuntimeLocked(reason: "endpoint_repair")
                guard let engine else { throw IOSConnectionRuntimeError.unavailable }
                try GoClientNative.repairSupervisorEndpoint(
                    engine: engine.handle,
                    endpointID: endpointID
                )
            } catch {
                if (error as? GoClientNativeError)?.invalidatesRuntimeGeneration != false {
                    markRuntimeUnhealthyLocked(reason: "endpoint_repair_failed")
                }
                throw error
            }
        }
    }

    func awaitDemandReady(timeoutMillis: UInt32) throws {
        let handle = try queue.sync { () throws -> UInt64 in
            try ensureRuntimeLocked(reason: "supervisor_wait")
            guard let engine else { throw IOSConnectionRuntimeError.unavailable }
            return engine.handle
        }
        try GoClientNative.awaitSupervisorReady(engine: handle, timeoutMillis: timeoutMillis)
    }

    func resetLocalPairings() throws {
        try queue.sync {
            try runIOSPairingResetTransaction(
                clearPersistentState: {
                    let storageCoordinator = IOSPlatformStorageCoordinator.process
                    let resetGeneration = storageCoordinator.beginGeneration()
                    try storageCoordinator.withGeneration(resetGeneration) {
                        endpointRegistry.clear()
                        try accessCredentials.clearAll()
                        try sshCredentials.clearAll()
                    }
                },
                replaceRuntime: { try replaceRuntimeLocked(reason: "pairings_reset") },
                stopRuntimeAfterFailure: { stopRuntimeLocked() }
            )
        }
    }

    func supervisorSnapshot() throws -> Anytty_Client_Binding_V1_EndpointSupervisorSnapshot {
        let handle = try queue.sync { () throws -> UInt64 in
            try ensureRuntimeLocked(reason: "supervisor_snapshot")
            guard let engine else { throw IOSConnectionRuntimeError.unavailable }
            return engine.handle
        }
        return try Anytty_Client_Binding_V1_EndpointSupervisorSnapshot(
            serializedBytes: GoClientNative.supervisorSnapshot(engine: handle)
        )
    }

    private func handleNativeForegroundResume(lease: IOSForegroundResumeLease) {
        guard foregroundFence.accepts(lease) else { return }
        do {
            _ = try beginForegroundResumeLocked(lease: lease)
        } catch IOSConnectionRuntimeError.staleForegroundResume {
            return
        } catch {
            markRuntimeUnhealthyLocked(reason: "foreground_resume_failed")
        }
    }

    private func beginForegroundResumeLocked(lease: IOSForegroundResumeLease) throws -> UInt64 {
        guard foregroundFence.accepts(lease) else { throw IOSConnectionRuntimeError.staleForegroundResume }
        try ensureRuntimeLocked(reason: "foreground_resume")
        guard let engine else { throw IOSConnectionRuntimeError.unavailable }
        guard foregroundFence.accepts(lease) else { throw IOSConnectionRuntimeError.staleForegroundResume }
        if lastSignaledForegroundLease === lease {
            return engine.handle
        }
        resampleNetworkLocked()
        try incrementHostRevisionLocked()
        let signaled = try foregroundFence.performIfAccepted(lease) {
            try signalSupervisorLocked(foreground: true)
            return true
        }
        guard signaled == true else { throw IOSConnectionRuntimeError.staleForegroundResume }
        lastSignaledForegroundLease = lease
        return engine.handle
    }

    private func resampleNetworkLocked() {
        let next = Self.descriptor(for: pathMonitor.currentPath)
        guard let reason = iosNetworkTransitionReason(from: networkPath, to: next) else { return }
        networkPath = next
        networkReason = reason
        guard networkEpoch < UInt64.max else { return }
        networkEpoch += 1
        emit(.network(IOSNativeNetworkSnapshot(
            epoch: networkEpoch,
            connected: networkPath.connected,
            reason: networkReason
        )))
    }

    private func scheduleNetworkChange(_ path: NWPath) {
        let next = Self.descriptor(for: path)
        pendingNetworkChange?.cancel()
        let work = DispatchWorkItem { [weak self] in
            self?.commitNetworkChange(next)
        }
        pendingNetworkChange = work
        queue.asyncAfter(
            deadline: .now() + .milliseconds(next.connected ? 200 : 750),
            execute: work
        )
    }

    private func commitNetworkChange(_ next: IOSNetworkPathDescriptor) {
        guard let reason = iosNetworkTransitionReason(from: networkPath, to: next) else { return }
        networkPath = next
        networkReason = reason
        guard networkEpoch < UInt64.max else {
            markRuntimeUnhealthyLocked(reason: "network_revision_exhausted")
            return
        }
        networkEpoch += 1

        do {
            try incrementHostRevisionLocked()
            if engine != nil {
                try signalSupervisorLocked(foreground: false)
            }
        } catch {
            markRuntimeUnhealthyLocked(reason: "network_signal_failed")
        }

        emit(.network(IOSNativeNetworkSnapshot(
            epoch: networkEpoch,
            connected: networkPath.connected,
            reason: networkReason
        )))
    }

    private func ensureRuntimeLocked(reason: String) throws {
        if engine != nil, port > 0, runtimeHealthy { return }
        if engineGeneration == 0, engine == nil {
            try startRuntimeLocked()
            return
        }
        try replaceRuntimeLocked(reason: reason)
    }

    private func replaceRuntimeLocked(reason: String) throws {
        guard generationEventEpoch < UInt64.max else {
            throw IOSConnectionRuntimeError.revisionExhausted
        }
        generationEventEpoch += 1
        let eventEpoch = generationEventEpoch
        emit(.generation(event: "generationChanging", reason: reason, epoch: eventEpoch))
        stopRuntimeLocked()
        do {
            try startRuntimeLocked()
            emit(.generation(event: "generationChanged", reason: reason, epoch: eventEpoch))
        } catch {
            emit(.generation(event: "generationChangeFailed", reason: reason, epoch: eventEpoch))
            throw error
        }
    }

    private func startRuntimeLocked() throws {
        guard engineGeneration < UInt64.max else { throw IOSConnectionRuntimeError.revisionExhausted }
        engineGeneration += 1
        let generation = engineGeneration
        let nextToken = try bridgeToken()
        let nextEngine = try IOSGoClientEngine(
            accessCredentials: accessCredentials,
            sshCredentials: sshCredentials,
            endpointRegistry: endpointRegistry
        ) { [weak self] error in
            self?.platformPumpFailed(generation: generation, error: error)
        }

        do {
            let nextPort = try GoClientNative.startBridge(engine: nextEngine.handle, token: nextToken)
            engine = nextEngine
            port = nextPort
            token = nextToken
            try replayIOSSupervisorState(
                signalHost: { try signalSupervisorLocked(foreground: false) },
                submitDemand: { try submitDemandLocked(rendererDemand.canonicalSnapshot()) }
            )
            runtimeHealthy = true
            scheduledRebuildEpoch = nil
            rebuildBackoffIndex = 0
        } catch {
            engine = nil
            port = 0
            token = ""
            runtimeHealthy = false
            nextEngine.close()
            throw error
        }
    }

    private func stopRuntimeLocked() {
        let current = engine
        engine = nil
        port = 0
        token = ""
        runtimeHealthy = false
        current?.close()
    }

    private func submitDemandLocked(_ snapshot: IOSRendererDemandSnapshot) throws {
        guard let engine else { throw IOSConnectionRuntimeError.unavailable }
        var demand = Anytty_Client_Binding_V1_EndpointSupervisorDemandSnapshot()
        demand.attachmentID = snapshot.attachmentID
        demand.demandRevision = snapshot.demandRevision
        demand.endpoints = snapshot.endpointIDs.sorted().map { endpointID in
            var endpoint = Anytty_Client_Binding_V1_EndpointSupervisorDemand()
            endpoint.endpointID = endpointID
            endpoint.mode = .takeover
            return endpoint
        }
        try GoClientNative.replaceSupervisorDemand(engine: engine.handle, payload: demand.serializedData())
    }

    private func signalSupervisorLocked(foreground: Bool) throws {
        guard let engine else { throw IOSConnectionRuntimeError.unavailable }
        var signal = Anytty_Client_Binding_V1_EndpointSupervisorHostSignal()
        signal.revision = hostRevision
        signal.connected = networkPath.connected
        signal.reason = networkReason
        signal.foreground = foreground
        try GoClientNative.signalSupervisor(engine: engine.handle, payload: signal.serializedData())
    }

    private func incrementHostRevisionLocked() throws {
        guard hostRevision < UInt64.max else { throw IOSConnectionRuntimeError.revisionExhausted }
        hostRevision += 1
    }

    private func platformPumpFailed(generation: UInt64, error _: Error) {
        queue.async { [weak self] in
            guard let self, self.engineGeneration == generation, self.engine != nil else { return }
            self.runtimeHealthy = false
            self.scheduleRuntimeRebuildLocked(reason: "platform_pump_failed")
        }
    }

    private func markRuntimeUnhealthyLocked(reason: String) {
        runtimeHealthy = false
        scheduleRuntimeRebuildLocked(reason: reason)
    }

    private func scheduleRuntimeRebuildLocked(reason: String) {
        guard scheduledRebuildEpoch == nil, rebuildScheduleEpoch < UInt64.max else { return }
        rebuildScheduleEpoch += 1
        let scheduleEpoch = rebuildScheduleEpoch
        scheduledRebuildEpoch = scheduleEpoch
        let delay = Self.runtimeRebuildBackoffSeconds[
            min(rebuildBackoffIndex, Self.runtimeRebuildBackoffSeconds.count - 1)
        ]
        rebuildBackoffIndex = min(rebuildBackoffIndex + 1, Self.runtimeRebuildBackoffSeconds.count - 1)
        queue.asyncAfter(deadline: .now() + .seconds(delay)) { [weak self] in
            guard let self else { return }
            guard self.scheduledRebuildEpoch == scheduleEpoch else { return }
            self.scheduledRebuildEpoch = nil
            guard !self.runtimeHealthy else { return }
            do {
                try self.replaceRuntimeLocked(reason: reason)
            } catch {
                self.scheduleRuntimeRebuildLocked(reason: reason)
            }
        }
    }

    private func emit(_ event: IOSConnectionRuntimeEvent) {
        eventSink?(event)
    }

    private func bridgeToken() throws -> String {
        var bytes = [UInt8](repeating: 0, count: Self.bridgeTokenBytes)
        guard SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes) == errSecSuccess else {
            throw IOSConnectionRuntimeError.unavailable
        }
        return Data(bytes).base64URLEncodedString()
    }

    private static func descriptor(for path: NWPath) -> IOSNetworkPathDescriptor {
        let activeInterfaces = [
            NWInterface.InterfaceType.wifi,
            .cellular,
            .wiredEthernet,
            .loopback,
            .other,
        ].filter(path.usesInterfaceType).map(interfaceTypeName).joined(separator: ",")
        let availableInterfaces = path.availableInterfaces.map { interface in
            "\(interfaceTypeName(interface.type)):\(interface.name):\(interface.index)"
        }.sorted().joined(separator: ",")
        let activeInterfaceNames = Set(path.availableInterfaces.filter {
            path.usesInterfaceType($0.type)
        }.map(\.name))
        let activeAddresses = interfaceAddresses(names: activeInterfaceNames).joined(separator: ",")
        let status: String
        switch path.status {
        case .satisfied: status = "satisfied"
        case .unsatisfied: status = "unsatisfied"
        case .requiresConnection: status = "requires_connection"
        @unknown default: status = "unknown"
        }
        return IOSNetworkPathDescriptor(
            connected: path.status == .satisfied,
            fingerprint: [
                "status=\(status)",
                "active=\(activeInterfaces)",
                "available=\(availableInterfaces)",
                "addresses=\(activeAddresses)",
                "expensive=\(path.isExpensive)",
                "constrained=\(path.isConstrained)",
                "dns=\(path.supportsDNS)",
                "ipv4=\(path.supportsIPv4)",
                "ipv6=\(path.supportsIPv6)",
            ].joined(separator: ";")
        )
    }

    private static func interfaceTypeName(_ type: NWInterface.InterfaceType) -> String {
        switch type {
        case .wifi: return "wifi"
        case .cellular: return "cellular"
        case .wiredEthernet: return "wired"
        case .loopback: return "loopback"
        case .other: return "other"
        @unknown default: return "unknown"
        }
    }

    private static func interfaceAddresses(names: Set<String>) -> [String] {
        guard !names.isEmpty else { return [] }
        var first: UnsafeMutablePointer<ifaddrs>?
        guard getifaddrs(&first) == 0, let first else { return [] }
        defer { freeifaddrs(first) }

        var values = Set<String>()
        var cursor: UnsafeMutablePointer<ifaddrs>? = first
        while let current = cursor {
            let interface = current.pointee
            defer { cursor = interface.ifa_next }
            guard let namePointer = interface.ifa_name,
                  names.contains(String(cString: namePointer)),
                  let address = interface.ifa_addr else { continue }
            let family = Int32(address.pointee.sa_family)
            guard family == AF_INET || family == AF_INET6 else { continue }

            var host = [CChar](repeating: 0, count: Int(NI_MAXHOST))
            let result = host.withUnsafeMutableBufferPointer { buffer in
                getnameinfo(
                    address,
                    socklen_t(address.pointee.sa_len),
                    buffer.baseAddress,
                    socklen_t(buffer.count),
                    nil,
                    0,
                    NI_NUMERICHOST
                )
            }
            if result == 0 { values.insert(String(cString: host)) }
        }
        return values.sorted()
    }
}
