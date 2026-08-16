import CryptoKit
import Darwin
import Foundation

struct NativeLocalDiscoveryCandidate {
    let discoveryKey: String
    let address: String
    let port: UInt32
    let protocolVersion: UInt32
    let expiresAtUnixNano: Int64
}

func nativeLocalDiscoveryResult(
    _ candidates: [NativeLocalDiscoveryCandidate]
) -> Anytty_Client_Binding_V1_LocalDiscoveryLookupResult {
    var result = Anytty_Client_Binding_V1_LocalDiscoveryLookupResult()
    result.candidates = candidates.map { candidate in
        var value = Anytty_Client_Binding_V1_LocalDiscoveryCandidate()
        value.address = candidate.address
        value.port = candidate.port
        value.protocolVersion = candidate.protocolVersion
        value.expiresAtUnixNano = candidate.expiresAtUnixNano
        return value
    }
    return result
}

final class NativeLocalDiscoveryCache {
    static let shared = NativeLocalDiscoveryCache()
    private let lock = NSLock()
    private var candidates: [String: [NativeLocalDiscoveryCandidate]] = [:]

    func update(service: NetService) {
        guard let data = service.txtRecordData() else { remove(serviceName: service.name); return }
        let text = NetService.dictionary(fromTXTRecord: data)
        let version = textValue(text["v"]).flatMap(Int.init)
        let protocolVersion = textValue(text["p"]).flatMap(UInt32.init)
        let discoveryKey = textValue(text["k"])?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        let addresses = (service.addresses ?? []).compactMap(numericHost).uniqued()
        guard version == 1, let protocolVersion, discoveryKey.range(of: "^[0-9a-f]{64}$", options: .regularExpression) != nil,
              service.port > 0, service.port <= 65_535, !addresses.isEmpty else {
            remove(serviceName: service.name)
            return
        }
        let expires = Int64(Date().timeIntervalSince1970 * 1_000_000_000) + 45_000_000_000
        let values = addresses.map {
            NativeLocalDiscoveryCandidate(
                discoveryKey: discoveryKey, address: $0,
                port: UInt32(service.port), protocolVersion: protocolVersion, expiresAtUnixNano: expires
            )
        }
        lock.lock()
        candidates[service.name] = values
        lock.unlock()
    }

    func remove(serviceName: String) {
        lock.lock()
        candidates.removeValue(forKey: serviceName)
        lock.unlock()
    }

    func clear() {
        lock.lock()
        candidates.removeAll()
        lock.unlock()
    }

    func snapshot(deviceID: String, fingerprint: String) -> [NativeLocalDiscoveryCandidate] {
        let expires = Int64(Date().timeIntervalSince1970 * 1_000_000_000) + 60_000_000_000
        let expectedKey = nativeLocalDiscoveryKey(deviceID: deviceID, fingerprint: fingerprint)
        lock.lock()
        let result = candidates.values.flatMap { $0 }.filter {
            $0.discoveryKey == expectedKey
        }.map {
            NativeLocalDiscoveryCandidate(
                discoveryKey: $0.discoveryKey,
                address: $0.address,
                port: $0.port,
                protocolVersion: $0.protocolVersion,
                expiresAtUnixNano: expires
            )
        }
        lock.unlock()
        var seen = Set<String>()
        return result.filter { seen.insert("\($0.address):\($0.port)").inserted }
    }

    private func textValue(_ data: Data?) -> String? {
        guard let data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    private func numericHost(_ data: Data) -> String? {
        data.withUnsafeBytes { buffer in
            guard let base = buffer.baseAddress?.assumingMemoryBound(to: sockaddr.self) else { return nil }
            var host = [CChar](repeating: 0, count: Int(NI_MAXHOST))
            guard getnameinfo(base, socklen_t(data.count), &host, socklen_t(host.count), nil, 0, NI_NUMERICHOST) == 0 else {
                return nil
            }
            return String(cString: host)
        }
    }
}

func nativeLocalDiscoveryKey(deviceID: String, fingerprint: String) -> String {
    let identity = "anytty-lan-discovery-v1\0\(deviceID.trimmingCharacters(in: .whitespacesAndNewlines))\0\(fingerprint.trimmingCharacters(in: .whitespacesAndNewlines))"
    return SHA256.hash(data: Data(identity.utf8)).map { String(format: "%02x", $0) }.joined()
}

final class NativeLocalDiscovery: NSObject, NetServiceBrowserDelegate, NetServiceDelegate {
    private let browser = NetServiceBrowser()
    private let onChanged: () -> Void
    private var services: [String: NetService] = [:]
    private var running = false

    init(onChanged: @escaping () -> Void = {}) {
        self.onChanged = onChanged
        super.init()
        browser.delegate = self
    }

    func restart(connected: Bool) {
        stop()
        NativeLocalDiscoveryCache.shared.clear()
        onChanged()
        guard connected else { return }
        running = true
        browser.searchForServices(ofType: "_anytty._tcp.", inDomain: "local.")
    }

    func stop() {
        guard running else { return }
        running = false
        browser.stop()
        for service in services.values { service.stop() }
        services.removeAll()
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didFind service: NetService, moreComing: Bool) {
        guard running else { return }
        services[service.name] = service
        service.delegate = self
        service.resolve(withTimeout: 5)
    }

    func netServiceBrowser(_ browser: NetServiceBrowser, didRemove service: NetService, moreComing: Bool) {
        services.removeValue(forKey: service.name)?.stop()
        NativeLocalDiscoveryCache.shared.remove(serviceName: service.name)
        onChanged()
    }

    func netServiceDidResolveAddress(_ sender: NetService) {
        NativeLocalDiscoveryCache.shared.update(service: sender)
        onChanged()
    }

    func netService(_ sender: NetService, didUpdateTXTRecord data: Data) {
        NativeLocalDiscoveryCache.shared.update(service: sender)
        onChanged()
    }

    func netService(_ sender: NetService, didNotResolve errorDict: [String: NSNumber]) {
        NativeLocalDiscoveryCache.shared.remove(serviceName: sender.name)
        onChanged()
    }

    deinit { stop() }
}

private extension Array where Element: Hashable {
    func uniqued() -> [Element] {
        var seen = Set<Element>()
        return filter { seen.insert($0).inserted }
    }
}
