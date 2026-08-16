import Capacitor
import Foundation
import Network
import Security
import SwiftProtobuf
import UIKit

@objc(NativeConnectionPlugin)
public final class NativeConnectionPlugin: CAPPlugin, CAPBridgedPlugin {
    private static weak var current: NativeConnectionPlugin?
    public let identifier = "NativeConnectionPlugin"
    public let jsName = "NativeConnection"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "handleForegroundResume", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getNetworkSnapshot", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "resetLocalPairings", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getBridgeEndpoint", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "setSessionActive", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "isLocalEndpointDiscovered", returnType: CAPPluginReturnPromise),
    ]

    private let runtimeQueue = DispatchQueue(label: "com.anytty.ios.runtime")
    private let accessCredentials = IOSClientAccessCredentialStore()
    private let sshCredentials = IOSSSHCredentialStore()
    private let endpointRegistry = IOSEndpointRegistryStore()
    private let pathMonitor = NWPathMonitor()
    private lazy var localDiscovery = NativeLocalDiscovery { [weak self] in
        self?.notifyListeners("localDiscoveryChanged", data: [:], retainUntilConsumed: true)
    }
    private var engine: IOSGoClientEngine?
    private var port: UInt16 = 0
    private var token = ""
    private var epoch: UInt64 = 0
    private var networkEpoch: UInt64 = 0
    private var receivedInitialPath = false
    private var lastNetworkSignature: String?
    private var pendingNetworkChange: DispatchWorkItem?
    private var latestNetworkConnected = true
    private var latestNetworkReason = "path_changed"

    override public func load() {
        Self.current = self
        pathMonitor.pathUpdateHandler = { [weak self] path in
            self?.runtimeQueue.async { self?.networkChanged(path) }
        }
        pathMonitor.start(queue: DispatchQueue(label: "com.anytty.ios.path-monitor"))
        runtimeQueue.async { [weak self] in _ = try? self?.ensureRuntime() }
    }

    deinit {
        if Self.current === self { Self.current = nil }
        pathMonitor.cancel()
        if Thread.isMainThread {
            localDiscovery.stop()
        } else {
            DispatchQueue.main.sync { localDiscovery.stop() }
        }
        NativeLocalDiscoveryCache.shared.clear()
        runtimeQueue.sync {
            pendingNetworkChange?.cancel()
            stopRuntime()
        }
    }

    static func refreshAfterNativePicker(completion: @escaping () -> Void) {
        guard let plugin = current else {
            DispatchQueue.main.async(execute: completion)
            return
        }
        plugin.runtimeQueue.async {
            try? plugin.ensureRuntime()
            DispatchQueue.main.async(execute: completion)
        }
    }

    @objc func handleForegroundResume(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            do {
                try self.ensureRuntime()
                call.resolve()
            } catch {
                call.reject("Go client engine could not resume", nil, error)
            }
        }
    }

    @objc func getNetworkSnapshot(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            call.resolve([
                "epoch": NSNumber(value: self.networkEpoch),
                "connected": self.latestNetworkConnected,
                "reason": self.latestNetworkReason,
                "scope": "session",
            ])
        }
    }

    @objc func resetLocalPairings(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            do {
                self.endpointRegistry.clear()
                try self.accessCredentials.clearAll()
                try self.sshCredentials.clearAll()
                try self.replaceRuntime(reason: "pairings_reset")
                call.resolve()
            } catch {
                try? self.ensureRuntime()
                call.reject("failed to reset local pairings", nil, error)
            }
        }
    }

    @objc func getBridgeEndpoint(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            do {
                try self.ensureRuntime()
                guard self.port > 0, !self.token.isEmpty else {
                    throw AnyTTYPlatformError.failure(code: "temporary", message: "native bridge server is not ready")
                }
                call.resolve(["port": Int(self.port), "token": self.token])
            } catch {
                call.reject("native bridge server is not ready", nil, error)
            }
        }
    }

    @objc func setSessionActive(_ call: CAPPluginCall) {
        let machineID = call.getString("machineId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !machineID.isEmpty else {
            call.reject("machineId is required")
            return
        }
        // iOS does not need Android's foreground-service ownership signal.
        call.resolve()
    }

    @objc func isLocalEndpointDiscovered(_ call: CAPPluginCall) {
        let deviceID = call.getString("deviceId")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let fingerprint = call.getString("fingerprint")?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !deviceID.isEmpty, !fingerprint.isEmpty else {
            call.reject("deviceId and fingerprint are required")
            return
        }
        runtimeQueue.async {
            do {
                let result = nativeLocalDiscoveryResult(
                    NativeLocalDiscoveryCache.shared.snapshot(deviceID: deviceID, fingerprint: fingerprint)
                )
                call.resolve(["discovered": try GoClientNative.localProbe(result.serializedData())])
            } catch {
                call.reject("local discovery probe failed", nil, error)
            }
        }
    }

    private func networkChanged(_ path: NWPath) {
        let signature = networkSignature(path)
        guard receivedInitialPath else {
            receivedInitialPath = true
            lastNetworkSignature = signature
            latestNetworkConnected = path.status == .satisfied
            latestNetworkReason = latestNetworkConnected ? "path_changed" : "offline"
            DispatchQueue.main.async { [weak self] in
                self?.localDiscovery.restart(connected: path.status == .satisfied)
            }
            return
        }
        let connected = path.status == .satisfied
        pendingNetworkChange?.cancel()
        let work = DispatchWorkItem { [weak self] in
            guard let self else { return }
            let previousSignature = self.lastNetworkSignature
            self.lastNetworkSignature = signature
            self.networkEpoch &+= 1
            let reason = connected
                ? (previousSignature?.hasPrefix("offline:") == true
                    ? "available"
                    : signature == previousSignature ? "path_changed" : "network_replaced")
                : "offline"
            self.latestNetworkConnected = connected
            self.latestNetworkReason = reason
            if !connected || reason == "available" || reason == "network_replaced" {
                DispatchQueue.main.async { [weak self] in self?.localDiscovery.restart(connected: connected) }
            }
            self.notifyNetworkChanged(connected: connected, reason: reason, epoch: self.networkEpoch)
        }
        pendingNetworkChange = work
        runtimeQueue.asyncAfter(
            deadline: .now() + .milliseconds(connected ? 200 : 750),
            execute: work
        )
    }

    private func networkSignature(_ path: NWPath) -> String {
        let interfaces = [
            path.usesInterfaceType(.wifi) ? "wifi" : "",
            path.usesInterfaceType(.cellular) ? "cellular" : "",
            path.usesInterfaceType(.wiredEthernet) ? "wired" : "",
            path.usesInterfaceType(.loopback) ? "loopback" : "",
            path.usesInterfaceType(.other) ? "other" : "",
        ].filter { !$0.isEmpty }.joined(separator: ",")
        return "\(path.status == .satisfied ? "online" : "offline"):\(interfaces)"
    }

    private func replaceRuntime(reason: String, alreadyStopped: Bool = false) throws {
        epoch &+= 1
        let currentEpoch = epoch
        notifyGeneration("generationChanging", reason: reason, epoch: currentEpoch)
        if !alreadyStopped { stopRuntime() }
        do {
            try startRuntime()
            notifyGeneration("generationChanged", reason: reason, epoch: currentEpoch)
        } catch {
            notifyGeneration("generationChangeFailed", reason: reason, epoch: currentEpoch)
            throw error
        }
    }

    private func ensureRuntime() throws {
        if engine == nil || port == 0 { try startRuntime() }
    }

    private func startRuntime() throws {
        let nextToken = try bridgeToken()
        let nextEngine = try IOSGoClientEngine(
            accessCredentials: accessCredentials,
            sshCredentials: sshCredentials,
            endpointRegistry: endpointRegistry
        )
        do {
            let nextPort = try GoClientNative.startBridge(engine: nextEngine.handle, token: nextToken)
            engine = nextEngine
            port = nextPort
            token = nextToken
        } catch {
            nextEngine.close()
            throw error
        }
    }

    private func stopRuntime() {
        port = 0
        token = ""
        engine?.close()
        engine = nil
    }

    private func bridgeToken() throws -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        guard SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes) == errSecSuccess else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "bridge token generation failed")
        }
        return Data(bytes).base64URLEncodedString()
    }

    private func notifyGeneration(_ event: String, reason: String, epoch: UInt64) {
        DispatchQueue.main.async { [weak self] in
            self?.notifyListeners(event, data: ["reason": reason, "epoch": NSNumber(value: epoch)])
        }
    }

    private func notifyNetworkChanged(connected: Bool, reason: String, epoch: UInt64) {
        DispatchQueue.main.async { [weak self] in
            self?.notifyListeners(
                "networkChanged",
                data: [
                    "epoch": NSNumber(value: epoch),
                    "connected": connected,
                    "reason": reason,
                    "scope": "session",
                ],
                retainUntilConsumed: true
            )
        }
    }
}
