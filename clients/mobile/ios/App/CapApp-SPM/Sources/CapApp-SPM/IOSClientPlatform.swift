import Foundation
import SwiftProtobuf

final class IOSEndpointRegistryStore {
    private let defaults = UserDefaults.standard
    private let key = "anytty.ios.endpoint-registry.v1"
    private let lock = NSLock()

    func load() -> Data {
        lock.lock()
        defer { lock.unlock() }
        return defaults.data(forKey: key) ?? Data()
    }

    func clear() {
        lock.lock()
        defer { lock.unlock() }
        defaults.removeObject(forKey: key)
    }

    func store(
        _ data: Data,
        deleting refs: [String],
        accessCredentials: IOSClientAccessCredentialStore,
        sshCredentials: IOSSSHCredentialStore
    ) throws {
        guard !data.isEmpty, data.count <= 1 << 20 else {
            throw AnyTTYPlatformError.failure(code: "protocol", message: "endpoint registry payload size is invalid")
        }
        lock.lock()
        defer { lock.unlock() }
        let previous = defaults.data(forKey: key)
        defaults.set(data, forKey: key)
        do {
            try accessCredentials.deleteMany(refs.filter { !$0.hasPrefix(IOSSSHCredentialStore.refPrefix) })
            try sshCredentials.deleteMany(refs.filter { $0.hasPrefix(IOSSSHCredentialStore.refPrefix) })
        } catch {
            if let previous { defaults.set(previous, forKey: key) } else { defaults.removeObject(forKey: key) }
            throw error
        }
    }
}

final class IOSClientPlatform {
    private let engine: UInt64
    private let accessCredentials: IOSClientAccessCredentialStore
    private let sshCredentials: IOSSSHCredentialStore
    private let endpointRegistry: IOSEndpointRegistryStore
    private let queue = DispatchQueue(label: "com.anytty.ios.platform")
    private let activeLock = NSLock()
    private var active = true

    init(
        engine: UInt64,
        accessCredentials: IOSClientAccessCredentialStore,
        sshCredentials: IOSSSHCredentialStore,
        endpointRegistry: IOSEndpointRegistryStore
    ) {
        self.engine = engine
        self.accessCredentials = accessCredentials
        self.sshCredentials = sshCredentials
        self.endpointRegistry = endpointRegistry
        queue.async { [weak self] in self?.run() }
    }

    func close() {
        activeLock.lock()
        active = false
        activeLock.unlock()
    }

    private func isActive() -> Bool {
        activeLock.lock()
        defer { activeLock.unlock() }
        return active
    }

    private func run() {
        while isActive() {
            do {
                let payload = try GoClientNative.nextPlatformRequest(engine: engine)
                let request = try Anytty_Client_Binding_V1_PlatformRequest(serializedBytes: payload)
                let response = dispatch(request)
                try GoClientNative.completePlatformRequest(engine: engine, payload: response.serializedData())
            } catch {
                return
            }
        }
    }

    private func dispatch(
        _ request: Anytty_Client_Binding_V1_PlatformRequest
    ) -> Anytty_Client_Binding_V1_PlatformResponse {
        var response = Anytty_Client_Binding_V1_PlatformResponse()
        response.requestID = request.requestID
        do {
            switch request.request {
            case .credentialPrepare(let value):
                response.credential = try accessCredentials.prepareRecord(ref: value.credentialRef, endpointID: value.endpointID)
            case .credentialResolve(let value):
                response.credential = try accessCredentials.resolveRecord(ref: value.credentialRef, endpointID: value.endpointID)
            case .credentialDelete(let value):
                try accessCredentials.delete(value.credentialRef)
            case .credentialSign(let value):
                var signed = Anytty_Client_Binding_V1_CredentialSignResponse()
                signed.signature = try accessCredentials.sign(ref: value.credentialRef, payload: value.payload)
                response.credentialSign = signed
            case .credentialBind(let value):
                response.credential = try accessCredentials.bindRecord(
                    ref: value.credentialRef,
                    endpointID: value.endpointID,
                    grant: value.capabilityGrant,
                    cloudRouteGrant: value.cloudRouteGrant,
                    cloudEdgeLocator: value.cloudEdgeLocator
                )
            case .endpointRegistryLoad:
                var loaded = Anytty_Client_Binding_V1_EndpointRegistryLoaded()
                loaded.registryProto = endpointRegistry.load()
                response.endpointRegistry = loaded
            case .endpointRegistryStore(let value):
                try endpointRegistry.store(
                    value.registryProto,
                    deleting: value.deleteCredentialRefs,
                    accessCredentials: accessCredentials,
                    sshCredentials: sshCredentials
                )
            case .sshCredentialLookup(let value):
                response.sshCredential = try sshCredentials.lookup(ref: value.credentialRef, createIfMissing: value.createIfMissing)
            case .sshCredentialSign(let value):
                var signed = Anytty_Client_Binding_V1_SSHCredentialSignResponse()
                signed.signature = try sshCredentials.sign(ref: value.credentialRef, digest: value.digest, hash: value.hash)
                response.sshCredentialSign = signed
            case .sshCredentialDelete(let value):
                try sshCredentials.delete(value.credentialRef)
            case .cloudProfileResolve(let value):
                response.cloudProfile = try cloudProfile(value.accountProfileRef)
            case nil:
                throw AnyTTYPlatformError.failure(code: "protocol", message: "platform request payload is missing")
            }
        } catch let error as AnyTTYPlatformError {
            response.error = apiError(code: error.code, message: error.message)
        } catch {
            response.error = apiError(code: "temporary", message: "iOS platform request failed")
        }
        return response
    }

    private func cloudProfile(_ reference: String) throws -> Anytty_Client_Binding_V1_CloudProfileRecord {
        let normalized = reference.trimmingCharacters(in: .whitespacesAndNewlines)
        let info = Bundle.main.infoDictionary ?? [:]
        let address = (info["AnyTTYCloudControllerAddress"] as? String ?? "cloud.anytty.com:443").trimmingCharacters(in: .whitespacesAndNewlines)
        let serverName = (info["AnyTTYCloudControllerServerName"] as? String ?? "cloud.anytty.com").trimmingCharacters(in: .whitespacesAndNewlines)
        guard normalized == "default", !address.isEmpty, !serverName.isEmpty else {
            throw AnyTTYPlatformError.failure(code: "route_unavailable", message: "AnyTTY Cloud profile is unavailable")
        }
        var profile = Anytty_Client_Binding_V1_CloudProfileRecord()
        profile.accountProfileRef = normalized
        profile.controllerAddress = address
        profile.controllerServerName = serverName
        if let encoded = info["AnyTTYCloudControllerCAPEMBase64"] as? String,
           !encoded.isEmpty,
           let data = Data(base64Encoded: encoded) {
            profile.controllerCaPem = data
        }
        return profile
    }

    private func apiError(code: String, message: String) -> Anytty_Api_V1_ApiError {
        var error = Anytty_Api_V1_ApiError()
        switch code {
        case "protocol": error.code = .invalidRequest
        case "unauthenticated", "login_required", "capability_invalid", "capability_expired", "identity_conflict": error.code = .unauthorized
        case "quota_exhausted": error.code = .conflict
        case "entitlement_denied": error.code = .entitlementDenied
        case "cancelled": error.code = .cancelled
        case "route_unavailable", "temporary", "companion_missing", "backpressure": error.code = .unavailable
        default: error.code = .internal
        }
        error.message = message
        error.retryable = error.code == .unavailable || code == "quota_exhausted"
        error.attempted = true
        return error
    }
}

final class IOSGoClientEngine {
    let handle: UInt64
    private let platform: IOSClientPlatform
    private let lock = NSLock()
    private var closed = false

    init(
        accessCredentials: IOSClientAccessCredentialStore,
        sshCredentials: IOSSSHCredentialStore,
        endpointRegistry: IOSEndpointRegistryStore
    ) throws {
        handle = try GoClientNative.create()
        platform = IOSClientPlatform(
            engine: handle,
            accessCredentials: accessCredentials,
            sshCredentials: sshCredentials,
            endpointRegistry: endpointRegistry
        )
    }

    func close() {
        lock.lock()
        guard !closed else { lock.unlock(); return }
        closed = true
        lock.unlock()
        try? GoClientNative.close(engine: handle)
        platform.close()
    }

    deinit { close() }
}
