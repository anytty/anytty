import Capacitor
import Foundation
import Network
import Security
import UIKit

@objc(NativeConnectionPlugin)
public final class NativeConnectionPlugin: CAPPlugin, CAPBridgedPlugin {
    private static weak var current: NativeConnectionPlugin?
    public let identifier = "NativeConnectionPlugin"
    public let jsName = "NativeConnection"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "handleForegroundResume", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "resetLocalPairings", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getBridgeEndpoint", returnType: CAPPluginReturnPromise),
    ]

    private let runtimeQueue = DispatchQueue(label: "com.anytty.ios.runtime")
    private let accessCredentials = IOSClientAccessCredentialStore()
    private let sshCredentials = IOSSSHCredentialStore()
    private let endpointRegistry = IOSEndpointRegistryStore()
    private let pathMonitor = NWPathMonitor()
    private var engine: IOSGoClientEngine?
    private var server: GoClientBridgeServer?
    private var port: UInt16 = 0
    private var token = ""
    private var epoch: UInt64 = 0
    private var receivedInitialPath = false
    private var backgrounded = false
    private var observers: [NSObjectProtocol] = []

    override public func load() {
        Self.current = self
        observers.append(NotificationCenter.default.addObserver(
            forName: UIApplication.didEnterBackgroundNotification,
            object: nil,
            queue: nil
        ) { [weak self] _ in
            self?.runtimeQueue.async {
                self?.backgrounded = true
                self?.stopRuntime()
            }
        })
        observers.append(NotificationCenter.default.addObserver(
            forName: UIApplication.willEnterForegroundNotification,
            object: nil,
            queue: nil
        ) { [weak self] _ in
            self?.runtimeQueue.async { self?.backgrounded = false }
        })
        pathMonitor.pathUpdateHandler = { [weak self] _ in
            self?.runtimeQueue.async { self?.networkChanged() }
        }
        pathMonitor.start(queue: DispatchQueue(label: "com.anytty.ios.path-monitor"))
        runtimeQueue.async { [weak self] in _ = try? self?.ensureRuntime() }
    }

    deinit {
        if Self.current === self { Self.current = nil }
        observers.forEach(NotificationCenter.default.removeObserver)
        pathMonitor.cancel()
        runtimeQueue.sync { stopRuntime() }
    }

    static func refreshAfterNativePicker(completion: @escaping () -> Void) {
        guard let plugin = current else {
            DispatchQueue.main.async(execute: completion)
            return
        }
        plugin.runtimeQueue.async {
            plugin.backgrounded = false
            try? plugin.replaceRuntime(reason: "foreground")
            DispatchQueue.main.async(execute: completion)
        }
    }

    @objc func handleForegroundResume(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            self.backgrounded = false
            do {
                try self.replaceRuntime(reason: "foreground")
                call.resolve()
            } catch {
                call.reject("Go client engine could not resume", nil, error)
            }
        }
    }

    @objc func resetLocalPairings(_ call: CAPPluginCall) {
        runtimeQueue.async { [weak self] in
            guard let self else { call.reject("native runtime is unavailable"); return }
            do {
                self.stopRuntime()
                self.endpointRegistry.clear()
                try self.accessCredentials.clearAll()
                try self.sshCredentials.clearAll()
                try self.replaceRuntime(reason: "pairings_reset", alreadyStopped: true)
                call.resolve()
            } catch {
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

    private func networkChanged() {
        guard receivedInitialPath else {
            receivedInitialPath = true
            return
        }
        guard !backgrounded, engine != nil else { return }
        do {
            try replaceRuntime(reason: "network_available")
        } catch {
            notifyGeneration("generationChangeFailed", reason: "network_available", epoch: epoch)
        }
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
        if engine == nil || server == nil || port == 0 { try startRuntime() }
    }

    private func startRuntime() throws {
        let nextToken = try bridgeToken()
        let nextEngine = try IOSGoClientEngine(
            accessCredentials: accessCredentials,
            sshCredentials: sshCredentials,
            endpointRegistry: endpointRegistry
        )
        do {
            let nextServer = try GoClientBridgeServer(engine: nextEngine, token: nextToken)
            let nextPort = try nextServer.start()
            engine = nextEngine
            server = nextServer
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
        server?.stop()
        server = nil
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
}
