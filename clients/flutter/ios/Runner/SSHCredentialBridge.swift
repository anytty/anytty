import CryptoKit
import Flutter
import Security

final class SSHCredentialBridge {
  private let lock = NSLock()
  private let refPrefix = "ssh-platform-"
  private let tagPrefix = "com.anytty.app.ssh.v1."

  func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    do {
      switch call.method {
      case "lookup":
        let arguments = try requiredArguments(call)
        let ref = try requiredString(arguments, "credentialRef")
        let createIfMissing = arguments["createIfMissing"] as? Bool ?? false
        result(try lookup(ref: ref, createIfMissing: createIfMissing))
      case "sign":
        let arguments = try requiredArguments(call)
        let ref = try requiredString(arguments, "credentialRef")
        guard let digest = (arguments["digest"] as? FlutterStandardTypedData)?.data else {
          throw SSHCredentialBridgeError(code: "protocol", message: "SSH digest is missing")
        }
        let hash = try requiredString(arguments, "hash")
        result(FlutterStandardTypedData(bytes: try sign(ref: ref, digest: digest, hash: hash)))
      case "delete":
        let arguments = try requiredArguments(call)
        try delete(try requiredString(arguments, "credentialRef"))
        result(nil)
      default:
        result(FlutterMethodNotImplemented)
      }
    } catch let error as SSHCredentialBridgeError {
      result(FlutterError(code: error.code, message: error.message, details: nil))
    } catch {
      result(FlutterError(code: "temporary", message: "SSH secure signer operation failed", details: nil))
    }
  }

  private func lookup(ref: String, createIfMissing: Bool) throws -> [String: Any] {
    try withLock {
      let normalized = try validateRef(ref)
      var key = try privateKey(normalized)
      let newlyCreated = key == nil && createIfMissing
      if newlyCreated { key = try createPrivateKey(normalized) }
      guard
        let key,
        let publicKey = SecKeyCopyPublicKey(key),
        let external = SecKeyCopyExternalRepresentation(publicKey, nil) as Data?
      else {
        throw SSHCredentialBridgeError(code: "unauthenticated", message: "SSH credential is missing")
      }
      return [
        "credentialRef": normalized,
        "publicKeyPkix": FlutterStandardTypedData(bytes: p256SubjectPublicKeyInfo(external)),
        "newlyCreated": newlyCreated,
      ]
    }
  }

  private func sign(ref: String, digest: Data, hash: String) throws -> Data {
    try withLock {
      guard hash == "SHA-256", digest.count == 32 else {
        throw SSHCredentialBridgeError(
          code: "protocol",
          message: "SSH signer only accepts SHA-256 digests"
        )
      }
      let normalized = try validateRef(ref)
      guard let key = try privateKey(normalized) else {
        throw SSHCredentialBridgeError(code: "unauthenticated", message: "SSH credential is missing")
      }
      var error: Unmanaged<CFError>?
      guard
        let signature = SecKeyCreateSignature(
          key,
          .ecdsaSignatureDigestX962SHA256,
          digest as CFData,
          &error
        ) as Data?
      else {
        throw SSHCredentialBridgeError(code: "temporary", message: "SSH signature failed")
      }
      return signature
    }
  }

  private func delete(_ ref: String) throws {
    try withLock {
      let status = SecItemDelete(keyQuery(try validateRef(ref)) as CFDictionary)
      guard status == errSecSuccess || status == errSecItemNotFound else {
        throw SSHCredentialBridgeError(code: "temporary", message: "SSH credential delete failed")
      }
    }
  }

  private func privateKey(_ ref: String) throws -> SecKey? {
    var result: CFTypeRef?
    var query = keyQuery(ref)
    query[kSecReturnRef as String] = true
    let status = SecItemCopyMatching(query as CFDictionary, &result)
    if status == errSecItemNotFound { return nil }
    guard status == errSecSuccess, let key = result else {
      throw SSHCredentialBridgeError(code: "temporary", message: "SSH credential lookup failed")
    }
    return (key as! SecKey)
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
    privateAttributes.removeValue(forKey: kSecAttrAccessible as String)
    attributes[kSecPrivateKeyAttrs as String] = privateAttributes
#endif
    var error: Unmanaged<CFError>?
    guard let key = SecKeyCreateRandomKey(attributes as CFDictionary, &error) else {
      throw SSHCredentialBridgeError(code: "temporary", message: "SSH credential creation failed")
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
    let digest = Data(SHA256.hash(data: Data(ref.utf8)))
      .base64EncodedString()
      .replacingOccurrences(of: "+", with: "-")
      .replacingOccurrences(of: "/", with: "_")
      .replacingOccurrences(of: "=", with: "")
    return Data((tagPrefix + digest).utf8)
  }

  private func p256SubjectPublicKeyInfo(_ x963: Data) -> Data {
    let header = Data([
      0x30, 0x59, 0x30, 0x13, 0x06, 0x07, 0x2a, 0x86, 0x48, 0xce, 0x3d,
      0x02, 0x01, 0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01,
      0x07, 0x03, 0x42, 0x00,
    ])
    return header + x963
  }

  private func validateRef(_ value: String) throws -> String {
    let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines)
    let valid = normalized.range(
      of: "^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$",
      options: .regularExpression
    ) != nil
    guard normalized.hasPrefix(refPrefix), valid else {
      throw SSHCredentialBridgeError(code: "protocol", message: "SSH credential ref is invalid")
    }
    return normalized
  }

  private func requiredArguments(_ call: FlutterMethodCall) throws -> [String: Any] {
    guard let arguments = call.arguments as? [String: Any] else {
      throw SSHCredentialBridgeError(code: "protocol", message: "SSH credential request is incomplete")
    }
    return arguments
  }

  private func requiredString(_ arguments: [String: Any], _ key: String) throws -> String {
    guard let value = arguments[key] as? String, !value.isEmpty else {
      throw SSHCredentialBridgeError(code: "protocol", message: "SSH credential request is incomplete")
    }
    return value
  }

  private func withLock<T>(_ body: () throws -> T) rethrows -> T {
    lock.lock()
    defer { lock.unlock() }
    return try body()
  }
}

private struct SSHCredentialBridgeError: Error {
  let code: String
  let message: String
}
