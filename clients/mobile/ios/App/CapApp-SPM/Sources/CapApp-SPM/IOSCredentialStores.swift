import CryptoKit
import Foundation
import Security

private struct StoredAccessCredential: Codable {
    let version: Int
    let endpointID: String
    let privateKey: Data
    var capabilityGrant: String
    var cloudRouteGrant: Data
    var cloudEdgeLocator: Data
}

final class IOSClientAccessCredentialStore {
    private let keychain = KeychainStore(service: "com.anytty.app.client-access.v1")
    private let lock = NSLock()

    func prepareRecord(ref: String, endpointID: String) throws -> Anytty_Client_Binding_V1_CredentialRecord {
        try lock.withLock {
            let normalizedRef = try validateCredentialRef(ref)
            let normalizedEndpoint = endpointID.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !normalizedEndpoint.isEmpty else {
                throw AnyTTYPlatformError.failure(code: "protocol", message: "endpoint_id is required")
            }
            let existing = try load(normalizedRef, requireGrant: false)
            if let existing {
                guard existing.endpointID == normalizedEndpoint else {
                    throw AnyTTYPlatformError.failure(code: "identity_conflict", message: "credential ref belongs to another endpoint")
                }
                return try record(existing, ref: normalizedRef, newlyCreated: false)
            }
            let privateKey = Curve25519.Signing.PrivateKey()
            let stored = StoredAccessCredential(
                version: 3,
                endpointID: normalizedEndpoint,
                privateKey: privateKey.rawRepresentation,
                capabilityGrant: "",
                cloudRouteGrant: Data(),
                cloudEdgeLocator: Data()
            )
            try persist(stored, ref: normalizedRef)
            return try record(stored, ref: normalizedRef, newlyCreated: true)
        }
    }

    func resolveRecord(ref: String, endpointID: String) throws -> Anytty_Client_Binding_V1_CredentialRecord {
        try lock.withLock {
            let normalizedRef = try validateCredentialRef(ref)
            guard let stored = try load(normalizedRef, requireGrant: true) else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "client access credential is missing")
            }
            guard stored.endpointID == endpointID.trimmingCharacters(in: .whitespacesAndNewlines) else {
                throw AnyTTYPlatformError.failure(code: "identity_conflict", message: "credential ref belongs to another endpoint")
            }
            return try record(stored, ref: normalizedRef, newlyCreated: false)
        }
    }

    func bindRecord(
        ref: String,
        endpointID: String,
        grant: String,
        cloudRouteGrant: Data,
        cloudEdgeLocator: Data
    ) throws -> Anytty_Client_Binding_V1_CredentialRecord {
        try lock.withLock {
            let normalizedRef = try validateCredentialRef(ref)
            let normalizedEndpoint = endpointID.trimmingCharacters(in: .whitespacesAndNewlines)
            let normalizedGrant = grant.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !normalizedEndpoint.isEmpty, !normalizedGrant.isEmpty else {
                throw AnyTTYPlatformError.failure(code: "protocol", message: "credential bind request is incomplete")
            }
            guard var stored = try load(normalizedRef, requireGrant: false) else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "client access identity is missing")
            }
            guard stored.endpointID == normalizedEndpoint else {
                throw AnyTTYPlatformError.failure(code: "identity_conflict", message: "credential ref belongs to another endpoint")
            }
            stored.capabilityGrant = normalizedGrant
            stored.cloudRouteGrant = cloudRouteGrant
            stored.cloudEdgeLocator = cloudEdgeLocator
            try persist(stored, ref: normalizedRef)
            return try record(stored, ref: normalizedRef, newlyCreated: false)
        }
    }

    func sign(ref: String, payload: Data) throws -> Data {
        try lock.withLock {
            let normalizedRef = try validateCredentialRef(ref)
            guard let stored = try load(normalizedRef, requireGrant: false) else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "client access credential is missing")
            }
            return try Curve25519.Signing.PrivateKey(rawRepresentation: stored.privateKey).signature(for: payload)
        }
    }

    func delete(_ ref: String) throws {
        try lock.withLock { try keychain.delete(validateCredentialRef(ref)) }
    }

    func deleteMany(_ refs: [String]) throws {
        try lock.withLock {
            for ref in Set(refs) { try keychain.delete(validateCredentialRef(ref)) }
        }
    }

    func clearAll() throws {
        try lock.withLock { try keychain.deleteAll() }
    }

    private func load(_ ref: String, requireGrant: Bool) throws -> StoredAccessCredential? {
        guard let data = try keychain.read(ref) else { return nil }
        do {
            let stored = try JSONDecoder().decode(StoredAccessCredential.self, from: data)
            guard stored.version == 3,
                  !stored.endpointID.isEmpty,
                  stored.privateKey.count == 32,
                  !requireGrant || !stored.capabilityGrant.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "client access credential is invalid")
            }
            return stored
        } catch let error as AnyTTYPlatformError {
            throw error
        } catch {
            throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "client access credential could not be decoded")
        }
    }

    private func persist(_ stored: StoredAccessCredential, ref: String) throws {
        try keychain.write(JSONEncoder().encode(stored), account: ref)
    }

    private func record(
        _ stored: StoredAccessCredential,
        ref: String,
        newlyCreated: Bool
    ) throws -> Anytty_Client_Binding_V1_CredentialRecord {
        let publicKey = try Curve25519.Signing.PrivateKey(rawRepresentation: stored.privateKey).publicKey.rawRepresentation
        var record = Anytty_Client_Binding_V1_CredentialRecord()
        record.endpointID = stored.endpointID
        record.credentialRef = ref
        record.publicKey = publicKey
        record.keyFingerprint = "ed25519-sha256:" + Data(SHA256.hash(data: publicKey)).base64URLEncodedString()
        record.capabilityGrant = stored.capabilityGrant
        record.newlyCreated = newlyCreated
        record.cloudRouteGrant = stored.cloudRouteGrant
        record.cloudEdgeLocator = stored.cloudEdgeLocator
        return record
    }
}

final class IOSSSHCredentialStore {
    static let refPrefix = "ssh-platform-"
    private let lock = NSLock()
    private let tagPrefix = "com.anytty.app.ssh.v1."

    func lookup(ref: String, createIfMissing: Bool) throws -> Anytty_Client_Binding_V1_SSHCredentialRecord {
        try lock.withLock {
            let normalized = try validateSSHRef(ref)
            var key = try privateKey(normalized)
            let newlyCreated = key == nil && createIfMissing
            if newlyCreated {
                key = try createPrivateKey(normalized)
            }
            guard let key, let publicKey = SecKeyCopyPublicKey(key),
                  let external = SecKeyCopyExternalRepresentation(publicKey, nil) as Data? else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "SSH credential is missing")
            }
            var record = Anytty_Client_Binding_V1_SSHCredentialRecord()
            record.credentialRef = normalized
            record.publicKeyPkix = p256SubjectPublicKeyInfo(external)
            record.newlyCreated = newlyCreated
            return record
        }
    }

    func sign(ref: String, digest: Data, hash: String) throws -> Data {
        try lock.withLock {
            guard hash == "SHA-256", digest.count == 32 else {
                throw AnyTTYPlatformError.failure(code: "protocol", message: "SSH signer only accepts SHA-256 digests")
            }
            let normalized = try validateSSHRef(ref)
            guard let key = try privateKey(normalized) else {
                throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "SSH credential is missing")
            }
            var error: Unmanaged<CFError>?
            guard let signature = SecKeyCreateSignature(key, .ecdsaSignatureDigestX962SHA256, digest as CFData, &error) as Data? else {
                throw AnyTTYPlatformError.failure(code: "temporary", message: "SSH signature failed")
            }
            return signature
        }
    }

    func delete(_ ref: String) throws {
        try lock.withLock {
            let normalized = try validateSSHRef(ref)
            let status = SecItemDelete(keyQuery(normalized) as CFDictionary)
            guard status == errSecSuccess || status == errSecItemNotFound else {
                throw AnyTTYPlatformError.failure(code: "temporary", message: "SSH credential delete failed")
            }
        }
    }

    func deleteMany(_ refs: [String]) throws {
        for ref in Set(refs) { try delete(ref) }
    }

    func clearAll() throws {
        try lock.withLock {
            let query: [String: Any] = [
                kSecClass as String: kSecClassKey,
                kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            ]
            var result: CFTypeRef?
            var listQuery = query
            listQuery[kSecReturnAttributes as String] = true
            listQuery[kSecMatchLimit as String] = kSecMatchLimitAll
            let status = SecItemCopyMatching(listQuery as CFDictionary, &result)
            guard status == errSecSuccess || status == errSecItemNotFound else {
                throw AnyTTYPlatformError.failure(code: "temporary", message: "SSH credential enumeration failed")
            }
            for attributes in (result as? [[String: Any]]) ?? [] {
                guard let tag = attributes[kSecAttrApplicationTag as String] as? Data,
                      String(data: tag, encoding: .utf8)?.hasPrefix(tagPrefix) == true else { continue }
                var delete = query
                delete[kSecAttrApplicationTag as String] = tag
                SecItemDelete(delete as CFDictionary)
            }
        }
    }

    private func privateKey(_ ref: String) throws -> SecKey? {
        var result: CFTypeRef?
        var query = keyQuery(ref)
        query[kSecReturnRef as String] = true
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "SSH credential lookup failed")
        }
        return (result as! SecKey)
    }

    private func createPrivateKey(_ ref: String) throws -> SecKey {
        let tag = applicationTag(ref)
        var privateAttributes: [String: Any] = [
            kSecAttrIsPermanent as String: true,
            kSecAttrApplicationTag as String: tag,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        var attributes: [String: Any] = [
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrKeySizeInBits as String: 256,
            kSecPrivateKeyAttrs as String: privateAttributes,
        ]
#if !targetEnvironment(simulator)
        attributes[kSecAttrTokenID as String] = kSecAttrTokenIDSecureEnclave
        privateAttributes[kSecAttrAccessControl as String] = SecAccessControlCreateWithFlags(
            nil,
            kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
            .privateKeyUsage,
            nil
        )
        attributes[kSecPrivateKeyAttrs as String] = privateAttributes
#endif
        var error: Unmanaged<CFError>?
        guard let key = SecKeyCreateRandomKey(attributes as CFDictionary, &error) else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "SSH credential creation failed")
        }
        return key
    }

    private func keyQuery(_ ref: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassKey,
            kSecAttrKeyType as String: kSecAttrKeyTypeECSECPrimeRandom,
            kSecAttrApplicationTag as String: applicationTag(ref),
        ]
    }

    private func applicationTag(_ ref: String) -> Data {
        Data((tagPrefix + Data(SHA256.hash(data: Data(ref.utf8))).base64URLEncodedString()).utf8)
    }

    private func p256SubjectPublicKeyInfo(_ x963: Data) -> Data {
        let header = Data([0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01,
                           0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07, 0x03, 0x42, 0x00])
        return header + x963
    }
}

private func validateCredentialRef(_ value: String) throws -> String {
    let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines)
    guard normalized.range(of: "^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$", options: .regularExpression) != nil else {
        throw AnyTTYPlatformError.failure(code: "unauthenticated", message: "credential ref is invalid")
    }
    return normalized
}

private func validateSSHRef(_ value: String) throws -> String {
    let normalized = try validateCredentialRef(value)
    guard normalized.hasPrefix(IOSSSHCredentialStore.refPrefix) else {
        throw AnyTTYPlatformError.failure(code: "protocol", message: "SSH credential ref is invalid")
    }
    return normalized
}

extension NSLock {
    fileprivate func withLock<T>(_ body: () throws -> T) rethrows -> T {
        lock()
        defer { unlock() }
        return try body()
    }
}

extension Data {
    func base64URLEncodedString() -> String {
        base64EncodedString().replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}
