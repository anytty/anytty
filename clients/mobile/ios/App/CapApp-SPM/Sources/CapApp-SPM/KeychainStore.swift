import Foundation
import Security

enum AnyTTYPlatformError: Error {
    case failure(code: String, message: String)

    var code: String {
        if case .failure(let code, _) = self { return code }
        return "internal"
    }

    var message: String {
        if case .failure(_, let message) = self { return message }
        return "iOS platform operation failed"
    }
}

final class KeychainStore {
    private let service: String

    init(service: String) {
        self.service = service
    }

    func read(_ account: String) throws -> Data? {
        var query = baseQuery(account)
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess, let data = result as? Data else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "Keychain read failed")
        }
        return data
    }

    func write(_ data: Data, account: String) throws {
        let query = baseQuery(account)
        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
        if updateStatus == errSecSuccess { return }
        guard updateStatus == errSecItemNotFound else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "Keychain update failed")
        }
        var insert = query
        attributes.forEach { insert[$0.key] = $0.value }
        guard SecItemAdd(insert as CFDictionary, nil) == errSecSuccess else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "Keychain write failed")
        }
    }

    func delete(_ account: String) throws {
        let status = SecItemDelete(baseQuery(account) as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "Keychain delete failed")
        }
    }

    func deleteAll() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
        ]
        let status = SecItemDelete(query as CFDictionary)
        guard status == errSecSuccess || status == errSecItemNotFound else {
            throw AnyTTYPlatformError.failure(code: "temporary", message: "Keychain clear failed")
        }
    }

    private func baseQuery(_ account: String) -> [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
    }
}
