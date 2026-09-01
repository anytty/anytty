import Flutter
import Foundation

final class LocalDiscoveryBridge: NSObject, NetServiceBrowserDelegate, NetServiceDelegate {
  private struct Candidate {
    let discoveryKey: String
    let address: String
    let port: Int
    let protocolVersion: Int
    let expiresAtUnixNano: Int64
  }

  private var browser: NetServiceBrowser?
  private var services: [String: NetService] = [:]
  private var candidates: [String: [Candidate]] = [:]
  private var resolving: Set<String> = []
  private var idleGeneration = 0

  func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard call.method == "lookup" else {
      result(FlutterMethodNotImplemented)
      return
    }
    guard
      let arguments = call.arguments as? [String: Any],
      let expectedKey = arguments["expectedKey"] as? String,
      expectedKey.range(of: "^[0-9a-f]{64}$", options: .regularExpression) != nil
    else {
      result(FlutterError(code: "protocol", message: "Local discovery identity is invalid", details: nil))
      return
    }
    ensureBrowsing()
    refreshKnownServices()
    scheduleIdleStop()
    let current = matching(expectedKey)
    if !current.isEmpty {
      NSLog("AnyTTYDiscovery stage=lookup source=cache candidates=%d", current.count)
      result(current)
      return
    }
    DispatchQueue.main.asyncAfter(deadline: .now() + 1.4) { [weak self] in
      let discovered = self?.matching(expectedKey) ?? []
      NSLog("AnyTTYDiscovery stage=lookup source=browse candidates=%d", discovered.count)
      result(discovered)
    }
  }

  func close() {
    idleGeneration += 1
    stopBrowsing()
    services.removeAll()
    candidates.removeAll()
    resolving.removeAll()
  }

  private func ensureBrowsing() {
    guard browser == nil else { return }
    let next = NetServiceBrowser()
    next.delegate = self
    browser = next
    next.searchForServices(ofType: "_anytty._tcp.", inDomain: "local.")
  }

  private func refreshKnownServices() {
    for service in services.values { resolve(service) }
  }

  private func resolve(_ service: NetService) {
    let key = serviceKey(service)
    guard resolving.insert(key).inserted else { return }
    service.delegate = self
    service.resolve(withTimeout: 1.2)
  }

  func netServiceBrowser(
    _ browser: NetServiceBrowser,
    didFind service: NetService,
    moreComing: Bool
  ) {
    services[serviceKey(service)] = service
    resolve(service)
  }

  func netServiceBrowser(
    _ browser: NetServiceBrowser,
    didRemove service: NetService,
    moreComing: Bool
  ) {
    let key = serviceKey(service)
    services.removeValue(forKey: key)
    candidates.removeValue(forKey: key)
    resolving.remove(key)
  }

  func netServiceDidResolveAddress(_ sender: NetService) {
    let key = serviceKey(sender)
    resolving.remove(key)
    services[key] = sender
    guard
      let txtData = sender.txtRecordData(),
      sender.port > 0,
      sender.port <= 65_535
    else {
      candidates.removeValue(forKey: key)
      return
    }
    let txt = NetService.dictionary(fromTXTRecord: txtData)
    guard
      txtString(txt, "v") == "1",
      let protocolVersion = Int(txtString(txt, "p") ?? ""),
      protocolVersion > 0,
      let discoveryKey = txtString(txt, "k")?.lowercased(),
      discoveryKey.range(of: "^[0-9a-f]{64}$", options: .regularExpression) != nil
    else {
      candidates.removeValue(forKey: key)
      return
    }
    let expires = unixNanoNow() + 30_000_000_000
    let addresses = (sender.addresses ?? [])
      .compactMap(numericAddress)
      .filter { !$0.isEmpty }
    candidates[key] = Array(Set(addresses)).sorted().prefix(8).map { address in
      Candidate(
        discoveryKey: discoveryKey,
        address: address,
        port: sender.port,
        protocolVersion: protocolVersion,
        expiresAtUnixNano: expires
      )
    }
  }

  func netService(
    _ sender: NetService,
    didNotResolve errorDict: [String: NSNumber]
  ) {
    resolving.remove(serviceKey(sender))
  }

  private func matching(_ expectedKey: String) -> [[String: Any]] {
    let now = unixNanoNow()
    var seen: Set<String> = []
    var values: [[String: Any]] = []
    for candidate in candidates.values.joined() {
      guard
        candidate.discoveryKey == expectedKey,
        candidate.expiresAtUnixNano > now
      else { continue }
      let unique = "\(candidate.address)\u{0}\(candidate.port)"
      guard seen.insert(unique).inserted else { continue }
      values.append([
        "address": candidate.address,
        "port": candidate.port,
        "protocolVersion": candidate.protocolVersion,
        "expiresAtUnixNano": candidate.expiresAtUnixNano,
        "networkHandle": 0,
      ])
      if values.count == 64 { break }
    }
    return values
  }

  private func scheduleIdleStop() {
    idleGeneration += 1
    let generation = idleGeneration
    DispatchQueue.main.asyncAfter(deadline: .now() + 30) { [weak self] in
      guard let self, generation == idleGeneration else { return }
      stopBrowsing()
    }
  }

  private func stopBrowsing() {
    browser?.stop()
    browser?.delegate = nil
    browser = nil
  }

  private func serviceKey(_ service: NetService) -> String {
    "\(service.domain)\u{0}\(service.type)\u{0}\(service.name)"
  }

  private func txtString(_ values: [String: Data], _ key: String) -> String? {
    guard let data = values[key] else { return nil }
    return String(data: data, encoding: .utf8)?.trimmingCharacters(in: .whitespacesAndNewlines)
  }

  private func numericAddress(_ data: Data) -> String? {
    data.withUnsafeBytes { bytes in
      guard let address = bytes.baseAddress?.assumingMemoryBound(to: sockaddr.self) else {
        return nil
      }
      var host = [CChar](repeating: 0, count: Int(NI_MAXHOST))
      let status = host.withUnsafeMutableBufferPointer { buffer in
        getnameinfo(
          address,
          socklen_t(data.count),
          buffer.baseAddress,
          socklen_t(buffer.count),
          nil,
          0,
          NI_NUMERICHOST
        )
      }
      return status == 0 ? String(cString: host) : nil
    }
  }

  private func unixNanoNow() -> Int64 {
    Int64(Date().timeIntervalSince1970 * 1_000_000_000)
  }
}
