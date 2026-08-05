import Capacitor
import CryptoKit
import Foundation
import UniformTypeIdentifiers
import UIKit

@objc(NativeFilePickerPlugin)
public final class NativeFilePickerPlugin: CAPPlugin, CAPBridgedPlugin, UIDocumentPickerDelegate {
    public let identifier = "NativeFilePickerPlugin"
    public let jsName = "NativeFilePicker"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "pickFiles", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "saveFile", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "getDownloadResumeOffset", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "appendDownloadPartial", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "commitDownloadPartial", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "discardDownloadPartial", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "openUploadSource", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "readUploadSource", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "finishUploadSource", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "closeUploadSource", returnType: CAPPluginReturnPromise),
    ]

    private let ioQueue = DispatchQueue(label: "com.anytty.ios.files")
    private let store = IOSDownloadStore()
    private var pickerCall: CAPPluginCall?
    private var uploads: [String: IOSUploadSource] = [:]

    @objc func pickFiles(_ call: CAPPluginCall) {
        DispatchQueue.main.async { [weak self] in
            guard let self, self.pickerCall == nil, let viewController = self.bridge?.viewController else {
                call.reject("file picker is unavailable")
                return
            }
            let picker = UIDocumentPickerViewController(forOpeningContentTypes: [.data], asCopy: true)
            picker.allowsMultipleSelection = call.options["multiple"] as? Bool ?? false
            picker.delegate = self
            self.pickerCall = call
            viewController.present(picker, animated: true)
        }
    }

    public func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
        guard let call = pickerCall else { return }
        pickerCall = nil
        let files: [[String: Any]] = urls.compactMap { url in
            let values = try? url.resourceValues(forKeys: [.fileSizeKey, .contentTypeKey, .nameKey])
            guard let size = values?.fileSize, size >= 0 else { return nil }
            return [
                "uri": url.absoluteString,
                "name": values?.name ?? url.lastPathComponent,
                "size": size,
                "mimeType": values?.contentType?.preferredMIMEType ?? "application/octet-stream",
            ]
        }
        finishPicker { call.resolve(["files": files]) }
    }

    public func documentPickerWasCancelled(_ controller: UIDocumentPickerViewController) {
        guard let call = pickerCall else { return }
        pickerCall = nil
        finishPicker { call.resolve(["files": []]) }
    }

    @objc func saveFile(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                guard let encoded = call.options["dataBase64"] as? String,
                      let bytes = Data(base64Encoded: encoded) else { throw FileStoreError.invalidArgument }
                call.resolve(try self.store.save(name: self.string(call, "name"), bytes: bytes))
            } catch { call.reject("failed to save download", nil, error) }
        }
    }

    @objc func getDownloadResumeOffset(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                let offset = try self.store.resumeOffset(
                    machineID: self.string(call, "machineId"),
                    remotePath: self.string(call, "remotePath"),
                    totalSize: self.int64(call, "totalSize")
                )
                call.resolve(["offset": NSNumber(value: offset)])
            } catch { call.reject("failed to inspect download partial", nil, error) }
        }
    }

    @objc func appendDownloadPartial(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                guard let encoded = call.options["dataBase64"] as? String,
                      let bytes = Data(base64Encoded: encoded),
                      !bytes.isEmpty, bytes.count <= 4 * 1024 * 1024 else { throw FileStoreError.invalidArgument }
                let offset = try self.store.append(
                    machineID: self.string(call, "machineId"),
                    remotePath: self.string(call, "remotePath"),
                    totalSize: self.int64(call, "totalSize"),
                    offset: self.int64(call, "offset"),
                    bytes: bytes
                )
                call.resolve(["offset": NSNumber(value: offset)])
            } catch { call.reject("failed to append download partial", nil, error) }
        }
    }

    @objc func commitDownloadPartial(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                guard let encoded = call.options["sha256Base64"] as? String,
                      let digest = Data(base64Encoded: encoded), digest.count == 32 else { throw FileStoreError.invalidArgument }
                let result = try self.store.commit(
                    name: self.string(call, "name"),
                    machineID: self.string(call, "machineId"),
                    remotePath: self.string(call, "remotePath"),
                    totalSize: self.int64(call, "totalSize"),
                    expectedDigest: digest
                )
                call.resolve(result)
            } catch { call.reject("failed to commit download partial", nil, error) }
        }
    }

    @objc func discardDownloadPartial(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                let discarded = try self.store.discard(
                    machineID: self.string(call, "machineId"),
                    remotePath: self.string(call, "remotePath"),
                    totalSize: self.int64(call, "totalSize")
                )
                call.resolve(["discarded": discarded])
            } catch { call.reject("failed to discard download partial", nil, error) }
        }
    }

    @objc func openUploadSource(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            do {
                let source = try IOSUploadSource(
                    uri: self.string(call, "contentUri"),
                    offset: self.int64(call, "offset"),
                    totalSize: self.int64(call, "totalSize")
                )
                self.uploads[source.id] = source
                call.resolve(["handle": source.id, "offset": NSNumber(value: source.offset)])
            } catch { call.reject("failed to open upload source", nil, error) }
        }
    }

    @objc func readUploadSource(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self, let source = self.uploads[self.string(call, "handle")] else {
                call.reject("upload source is unavailable"); return
            }
            do {
                let length = Int(self.int64(call, "length"))
                guard length > 0, length <= 4 * 1024 * 1024 else { throw FileStoreError.invalidArgument }
                let bytes = try source.read(length: length)
                call.resolve([
                    "dataBase64": bytes.base64EncodedString(),
                    "offset": NSNumber(value: source.offset),
                    "eof": source.offset == source.totalSize,
                ])
            } catch { call.reject("failed to read upload source", nil, error) }
        }
    }

    @objc func finishUploadSource(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            guard let self else { call.reject("file store is unavailable"); return }
            let handle = self.string(call, "handle")
            guard let source = self.uploads.removeValue(forKey: handle) else {
                call.reject("upload source is unavailable"); return
            }
            do {
                let digest = try source.finish()
                call.resolve(["sha256Base64": digest.base64EncodedString()])
            } catch { call.reject("failed to finish upload source", nil, error) }
        }
    }

    @objc func closeUploadSource(_ call: CAPPluginCall) {
        ioQueue.async { [weak self] in
            self?.uploads.removeValue(forKey: self?.string(call, "handle") ?? "")?.close()
            call.resolve()
        }
    }

    private func finishPicker(_ completion: @escaping () -> Void) {
        NativeConnectionPlugin.refreshAfterNativePicker(completion: completion)
    }

    private func string(_ call: CAPPluginCall, _ key: String) -> String {
        (call.options[key] as? String) ?? ""
    }

    private func int64(_ call: CAPPluginCall, _ key: String) -> Int64 {
        (call.options[key] as? NSNumber)?.int64Value ?? -1
    }
}

private enum FileStoreError: Error { case invalidArgument, invalidState, digestMismatch }

private final class IOSDownloadStore {
    private let manager = FileManager.default

    func save(name: String, bytes: Data) throws -> [String: Any] {
        let digest = Data(SHA256.hash(data: bytes))
        let destination = try downloadDestination(name: name)
        try bytes.write(to: destination, options: .atomic)
        return saved(destination, bytes: Int64(bytes.count), digest: digest)
    }

    func resumeOffset(machineID: String, remotePath: String, totalSize: Int64) throws -> Int64 {
        guard totalSize > 0 else { return 0 }
        let file = try partial(machineID: machineID, remotePath: remotePath, totalSize: totalSize)
        let length = fileSize(file)
        if length > totalSize {
            try? manager.removeItem(at: file)
            return 0
        }
        return length
    }

    func append(machineID: String, remotePath: String, totalSize: Int64, offset: Int64, bytes: Data) throws -> Int64 {
        guard totalSize >= 0, offset >= 0, offset <= totalSize,
              !bytes.isEmpty, offset + Int64(bytes.count) <= totalSize else { throw FileStoreError.invalidArgument }
        let file = try partial(machineID: machineID, remotePath: remotePath, totalSize: totalSize)
        if !manager.fileExists(atPath: file.path) { manager.createFile(atPath: file.path, contents: nil) }
        guard fileSize(file) == offset else { throw FileStoreError.invalidState }
        let handle = try FileHandle(forWritingTo: file)
        try handle.seekToEnd()
        try handle.write(contentsOf: bytes)
        try handle.synchronize()
        try handle.close()
        return fileSize(file)
    }

    func commit(
        name: String,
        machineID: String,
        remotePath: String,
        totalSize: Int64,
        expectedDigest: Data
    ) throws -> [String: Any] {
        let file = try partial(machineID: machineID, remotePath: remotePath, totalSize: totalSize)
        guard fileSize(file) == totalSize else { throw FileStoreError.invalidState }
        let digest = try sha256(file)
        guard digest == expectedDigest else { throw FileStoreError.digestMismatch }
        let destination = try downloadDestination(name: name)
        if manager.fileExists(atPath: destination.path) { try manager.removeItem(at: destination) }
        try manager.moveItem(at: file, to: destination)
        return saved(destination, bytes: totalSize, digest: digest)
    }

    func discard(machineID: String, remotePath: String, totalSize: Int64) throws -> Bool {
        let file = try partial(machineID: machineID, remotePath: remotePath, totalSize: totalSize)
        if manager.fileExists(atPath: file.path) { try manager.removeItem(at: file) }
        return true
    }

    private func partial(machineID: String, remotePath: String, totalSize: Int64) throws -> URL {
        guard !machineID.isEmpty, !remotePath.isEmpty, totalSize >= 0 else { throw FileStoreError.invalidArgument }
        let key = Data(SHA256.hash(data: Data("\(machineID)\0\(remotePath)\0\(totalSize)".utf8))).hex
        let directory = try manager.url(for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
            .appendingPathComponent("AnyTTY/FileTransfers", isDirectory: true)
        try manager.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory.appendingPathComponent(key + ".part")
    }

    private func downloadDestination(name: String) throws -> URL {
        let normalized = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalized.isEmpty, normalized != ".", normalized != "..",
              !normalized.contains("/"), !normalized.contains("\\"), !normalized.contains("\0") else {
            throw FileStoreError.invalidArgument
        }
        let documents = try manager.url(for: .documentDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        let directory = documents.appendingPathComponent("Downloads/AnyTTY", isDirectory: true)
        try manager.createDirectory(at: directory, withIntermediateDirectories: true)
        return directory.appendingPathComponent(normalized, isDirectory: false)
    }

    private func saved(_ url: URL, bytes: Int64, digest: Data) -> [String: Any] {
        ["uri": url.absoluteString, "path": url.path, "bytes": NSNumber(value: bytes), "sha256": digest.hex]
    }

    private func fileSize(_ url: URL) -> Int64 {
        ((try? manager.attributesOfItem(atPath: url.path)[.size]) as? NSNumber)?.int64Value ?? 0
    }

    private func sha256(_ url: URL) throws -> Data {
        let handle = try FileHandle(forReadingFrom: url)
        defer { try? handle.close() }
        var hasher = SHA256()
        while true {
            let data = try handle.read(upToCount: 256 * 1024) ?? Data()
            if data.isEmpty { break }
            hasher.update(data: data)
        }
        return Data(hasher.finalize())
    }
}

private final class IOSUploadSource {
    let id = UUID().uuidString
    let totalSize: Int64
    private(set) var offset: Int64
    private let handle: FileHandle
    private var hasher = SHA256()
    private var closed = false

    init(uri: String, offset: Int64, totalSize: Int64) throws {
        guard let url = URL(string: uri), url.isFileURL, offset >= 0, totalSize >= 0, offset <= totalSize else {
            throw FileStoreError.invalidArgument
        }
        let size = ((try FileManager.default.attributesOfItem(atPath: url.path)[.size]) as? NSNumber)?.int64Value ?? -1
        guard size == totalSize else { throw FileStoreError.invalidState }
        handle = try FileHandle(forReadingFrom: url)
        self.offset = 0
        self.totalSize = totalSize
        while self.offset < offset {
            let count = min(256 * 1024, Int(offset - self.offset))
            let prefix = try handle.read(upToCount: count) ?? Data()
            guard !prefix.isEmpty else { throw FileStoreError.invalidState }
            hasher.update(data: prefix)
            self.offset += Int64(prefix.count)
        }
    }

    func read(length: Int) throws -> Data {
        guard !closed, offset < totalSize else { throw FileStoreError.invalidState }
        let count = min(length, Int(totalSize - offset))
        let data = try handle.read(upToCount: count) ?? Data()
        guard !data.isEmpty else { throw FileStoreError.invalidState }
        hasher.update(data: data)
        offset += Int64(data.count)
        return data
    }

    func finish() throws -> Data {
        guard !closed, offset == totalSize else { throw FileStoreError.invalidState }
        closed = true
        try handle.close()
        return Data(hasher.finalize())
    }

    func close() {
        guard !closed else { return }
        closed = true
        try? handle.close()
    }

    deinit { close() }
}

private extension Data {
    var hex: String { map { String(format: "%02x", $0) }.joined() }
}
