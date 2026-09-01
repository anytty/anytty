import Capacitor
import Foundation
import SwiftProtobuf
import UIKit

final class IOSRendererCallGeneration {}

final class IOSRendererCallFence {
    private let lock = NSLock()
    private var currentGeneration = IOSRendererCallGeneration()

    func capture() -> IOSRendererCallGeneration {
        lock.lock()
        defer { lock.unlock() }
        return currentGeneration
    }

    func rotate() -> IOSRendererCallGeneration {
        let generation = IOSRendererCallGeneration()
        lock.lock()
        currentGeneration = generation
        lock.unlock()
        return generation
    }

    func accepts(_ generation: IOSRendererCallGeneration) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return currentGeneration === generation
    }
}

@objc(NativeConnectionPlugin)
public final class NativeConnectionPlugin: CAPPlugin, CAPBridgedPlugin {
    private static weak var current: NativeConnectionPlugin?
    public let identifier = "NativeConnectionPlugin"
    public let jsName = "NativeConnection"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "handleForegroundResume", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "requestEndpointRecovery", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getNetworkSnapshot", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "resetLocalPairings", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getBridgeEndpoint", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getSessionDemandLease", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "resumeSessionDemand", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "replaceSessionDemand", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "isLocalEndpointDiscovered", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "isDirectRouteReachable", returnType: CAPPluginReturnPromise),
    ]

    private let runtimeQueue = DispatchQueue(label: "com.anytty.ios.connection-plugin")
    private let runtimeOwner = IOSConnectionRuntimeOwner.shared
    private let rendererCallFence = IOSRendererCallFence()
    private lazy var localDiscovery = NativeLocalDiscovery { [weak self] in
        self?.notifyListeners("localDiscoveryChanged", data: [:], retainUntilConsumed: true)
    }
    private var rendererDemand: IOSRendererDemandSnapshot?

    override public func load() {
        Self.current = self
        rendererDemand = runtimeOwner.attachRenderer { [weak self] event in
            self?.handleRuntimeEvent(event)
        }
        let network = runtimeOwner.networkSnapshot()
        DispatchQueue.main.async { [weak self] in
            self?.localDiscovery.restart(connected: network.connected)
        }
    }

    deinit {
        if Self.current === self { Self.current = nil }
        if let rendererDemand { runtimeOwner.detachRenderer(attachmentID: rendererDemand.attachmentID) }
        if Thread.isMainThread {
            localDiscovery.stop()
        } else {
            DispatchQueue.main.sync { localDiscovery.stop() }
        }
        NativeLocalDiscoveryCache.shared.clear()
    }

    static func refreshAfterNativePicker(completion: @escaping () -> Void) {
        guard let plugin = current else {
            DispatchQueue.main.async(execute: completion)
            return
        }
        plugin.runtimeQueue.async {
            try? plugin.runtimeOwner.ensureRuntime()
            DispatchQueue.main.async(execute: completion)
        }
    }

    func rendererContentProcessDidTerminate() {
        let generation = rendererCallFence.rotate()
        runtimeQueue.async { [weak self] in
            guard let self, self.rendererCallFence.accepts(generation) else { return }
            guard let current = self.rendererDemand else { return }
            do {
                self.rendererDemand = try self.runtimeOwner.rotateRenderer(
                    attachmentID: current.attachmentID
                )
            } catch {
                if self.rendererCallFence.accepts(generation) {
                    self.rendererDemand = nil
                }
            }
        }
    }

    @objc func handleForegroundResume(_ call: CAPPluginCall) {
        let rendererGeneration = rendererCallFence.capture()
        let foregroundLease = runtimeOwner.captureForegroundResumeLease()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                try self.runtimeOwner.handleForegroundResume(lease: foregroundLease)
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve()
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("Go client engine could not resume", nil, error)
            }
        }
    }

    @objc func requestEndpointRecovery(_ call: CAPPluginCall) {
        let endpointID = call.getString("endpointId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !endpointID.isEmpty else {
            call.reject("endpointId is required")
            return
        }
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                try self.runtimeOwner.requestEndpointRecovery(endpointID: endpointID)
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve()
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("endpoint recovery could not be requested", nil, error)
            }
        }
    }

    @objc func getNetworkSnapshot(_ call: CAPPluginCall) {
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            let snapshot = self.runtimeOwner.networkSnapshot()
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            call.resolve([
                "epoch": NSNumber(value: snapshot.epoch),
                "connected": snapshot.connected,
                "reason": snapshot.reason,
                "scope": "session",
            ])
        }
    }

    @objc func resetLocalPairings(_ call: CAPPluginCall) {
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                try self.runtimeOwner.resetLocalPairings()
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve()
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                try? self.runtimeOwner.ensureRuntime()
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("failed to reset local pairings", nil, error)
            }
        }
    }

    @objc func getBridgeEndpoint(_ call: CAPPluginCall) {
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                let endpoint = try self.runtimeOwner.bridgeEndpoint()
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(["port": Int(endpoint.port), "token": endpoint.token])
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("native bridge server is not ready", nil, error)
            }
        }
    }

    @objc func getSessionDemandLease(_ call: CAPPluginCall) {
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                guard let current = self.rendererDemand else {
                    throw NSError(domain: "AnyTTY", code: 1, userInfo: [
                        NSLocalizedDescriptionKey: "renderer demand attachment is unavailable",
                    ])
                }
                let snapshot = try self.runtimeOwner.currentDemand(attachmentID: current.attachmentID)
                self.rendererDemand = snapshot
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(self.demandLeasePayload(snapshot))
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("renderer demand attachment is unavailable", nil, error)
            }
        }
    }

    @objc func replaceSessionDemand(_ call: CAPPluginCall) {
        guard let requestedEndpointIDs = call.getArray("endpointIds", String.self) else {
            call.reject("endpointIds is required")
            return
        }
        guard requestedEndpointIDs.allSatisfy({ !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }) else {
            call.reject("endpointIds contains an invalid value")
            return
        }
        let attachmentID = call.getString("attachmentId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let baseDemandRevision = call.getString("baseDemandRevision").flatMap(UInt64.init)
        guard !attachmentID.isEmpty, let baseDemandRevision else {
            call.reject("a valid native demand lease is required")
            return
        }
        let endpointIDs = Set(requestedEndpointIDs.map { $0.trimmingCharacters(in: .whitespacesAndNewlines) })
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                let next = try self.runtimeOwner.replaceDemand(
                    attachmentID: attachmentID,
                    baseDemandRevision: baseDemandRevision,
                    endpointIDs: endpointIDs
                )
                self.rendererDemand = next
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(self.demandLeasePayload(next))
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                if let current = self.rendererDemand {
                    self.rendererDemand = try? self.runtimeOwner.currentDemand(attachmentID: current.attachmentID)
                }
                call.reject("failed to replace renderer connection demand", nil, error)
            }
        }
    }

    @objc func resumeSessionDemand(_ call: CAPPluginCall) {
        let intentID = call.getString("intentId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let baseStopEpoch = call.getString("baseStopEpoch").flatMap(UInt64.init)
        guard !intentID.isEmpty, intentID.count <= 128, let baseStopEpoch else {
            call.reject("a valid renderer resume intent ID is required")
            return
        }
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                guard let current = self.rendererDemand else {
                    throw NSError(domain: "AnyTTY", code: 1, userInfo: [
                        NSLocalizedDescriptionKey: "renderer demand attachment is unavailable",
                    ])
                }
                let resumed = try self.runtimeOwner.resumeDemand(
                    attachmentID: current.attachmentID,
                    intentID: intentID,
                    baseStopEpoch: baseStopEpoch
                )
                self.rendererDemand = resumed.snapshot
                var payload = self.demandLeasePayload(resumed.snapshot)
                payload["outcome"] = resumed.outcome.rawValue
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(payload)
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                if let current = self.rendererDemand {
                    self.rendererDemand = try? self.runtimeOwner.currentDemand(attachmentID: current.attachmentID)
                }
                call.reject("failed to resume renderer connection demand", nil, error)
            }
        }
    }

    @objc func isLocalEndpointDiscovered(_ call: CAPPluginCall) {
        let deviceID = call.getString("deviceId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let fingerprint = call.getString("fingerprint")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !deviceID.isEmpty, !fingerprint.isEmpty else {
            call.reject("deviceId and fingerprint are required")
            return
        }
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                let result = nativeLocalDiscoveryResult(
                    NativeLocalDiscoveryCache.shared.snapshot(deviceID: deviceID, fingerprint: fingerprint)
                )
                let discovered = try GoClientNative.localProbe(result.serializedData())
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(["discovered": discovered])
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("local discovery probe failed", nil, error)
            }
        }
    }

    @objc func isDirectRouteReachable(_ call: CAPPluginCall) {
        guard let encoded = call.getString("routeProtoBase64")?.trimmingCharacters(in: .whitespacesAndNewlines),
              let routeProto = Data(base64Encoded: encoded), !routeProto.isEmpty else {
            call.reject("routeProtoBase64 is required")
            return
        }
        let rendererGeneration = rendererCallFence.capture()
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
            do {
                let reachable = try GoClientNative.directProbe(routeProto)
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.resolve(["reachable": reachable])
            } catch {
                guard self.requireCurrentRenderer(rendererGeneration, call: call) else { return }
                call.reject("Direct TCP probe failed", nil, error)
            }
        }
    }

    private func handleRuntimeEvent(_ event: IOSConnectionRuntimeEvent) {
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            switch event {
            case .generation(let event, let reason, let epoch):
                self.notifyListeners(event, data: ["reason": reason, "epoch": NSNumber(value: epoch)])
            case .network(let snapshot):
                self.localDiscovery.restart(connected: snapshot.connected)
                self.notifyListeners(
                    "networkChanged",
                    data: [
                        "epoch": NSNumber(value: snapshot.epoch),
                        "connected": snapshot.connected,
                        "reason": snapshot.reason,
                        "scope": "session",
                    ],
                    retainUntilConsumed: true
                )
            }
        }
    }

    private func requireCurrentRenderer(
        _ generation: IOSRendererCallGeneration,
        call: CAPPluginCall
    ) -> Bool {
        guard rendererCallFence.accepts(generation) else {
            call.reject("renderer attachment was replaced")
            return false
        }
        return true
    }

    private func demandLeasePayload(_ snapshot: IOSRendererDemandSnapshot) -> [String: Any] {
        [
            "attachmentId": snapshot.attachmentID,
            "demandRevision": String(snapshot.demandRevision),
            "stopEpoch": String(snapshot.stopEpoch),
            "endpointIds": snapshot.endpointIDs.sorted(),
            "stopped": snapshot.stopped,
        ]
    }
}
