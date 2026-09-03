import Flutter
import Network
import UIKit
import UserNotifications
import WebKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate, UIDocumentPickerDelegate {
  private var fileChannel: FlutterMethodChannel?
  private var externalURIChannel: FlutterMethodChannel?
  private var backgroundChannel: FlutterMethodChannel?
  private var sshCredentialChannel: FlutterMethodChannel?
  private var localDiscoveryChannel: FlutterMethodChannel?
  private var browserProxyChannel: FlutterMethodChannel?
  private var browserProxyLease: String?
  private let sshCredentialBridge = SSHCredentialBridge()
  private let localDiscoveryBridge = LocalDiscoveryBridge()
  private var pendingExportResult: FlutterResult?
  private var pendingNotificationRoute: String?
  private var connectionBackgroundTask: UIBackgroundTaskIdentifier = .invalid

  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    UNUserNotificationCenter.current().delegate = self
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let channel = FlutterMethodChannel(
      name: "com.anytty.app/files",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    channel.setMethodCallHandler { [weak self] call, result in
      self?.handleFileMethod(call, result: result)
    }
    fileChannel = channel

    let externalChannel = FlutterMethodChannel(
      name: "com.anytty.app/external-uri",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    externalChannel.setMethodCallHandler { [weak self] call, result in
      self?.handleExternalURIMethod(call, result: result)
    }
    externalURIChannel = externalChannel

    let connectionChannel = FlutterMethodChannel(
      name: "com.anytty.app/background",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    connectionChannel.setMethodCallHandler { [weak self] call, result in
      self?.handleBackgroundMethod(call, result: result)
    }
    backgroundChannel = connectionChannel

    let sshChannel = FlutterMethodChannel(
      name: "com.anytty.app/ssh-credentials",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    sshChannel.setMethodCallHandler { [weak self] call, result in
      self?.sshCredentialBridge.handle(call, result: result)
    }
    sshCredentialChannel = sshChannel

    let discoveryChannel = FlutterMethodChannel(
      name: "com.anytty.app/local-discovery",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    discoveryChannel.setMethodCallHandler { [weak self] call, result in
      self?.localDiscoveryBridge.handle(call, result: result)
    }
    localDiscoveryChannel = discoveryChannel

    let browserChannel = FlutterMethodChannel(
      name: "com.anytty.app/browser-proxy",
      binaryMessenger: engineBridge.applicationRegistrar.messenger()
    )
    browserChannel.setMethodCallHandler { [weak self] call, result in
      self?.handleBrowserProxyMethod(call, result: result)
    }
    browserProxyChannel = browserChannel
  }

  private func handleBrowserProxyMethod(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "clearData":
      let store = WKWebsiteDataStore.default()
      if #available(iOS 17.0, *) {
        store.proxyConfigurations = []
      }
      store.removeData(
        ofTypes: WKWebsiteDataStore.allWebsiteDataTypes(),
        modifiedSince: Date.distantPast
      ) {
        DispatchQueue.main.async { result(nil) }
      }
    case "open":
      openBrowserProxy(call, result: result)
    case "close":
      closeBrowserProxy(call, result: result)
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private func openBrowserProxy(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard #available(iOS 17.0, *) else {
      result(FlutterError(code: "browser_proxy_unavailable", message: "Session-bound WebView proxy requires iOS 17 or later", details: nil))
      return
    }
    guard
      let arguments = call.arguments as? [String: Any],
      let sessionID = arguments["sessionId"] as? String,
      let endpointID = arguments["endpointId"] as? String,
      let proxyHost = arguments["proxyHost"] as? String,
      let proxyPort = arguments["proxyPort"] as? Int,
      let routeID = arguments["routeId"] as? String,
      let routeGeneration = arguments["routeGeneration"] as? Int,
      !sessionID.isEmpty,
      !endpointID.isEmpty,
      !proxyHost.isEmpty,
      !routeID.isEmpty,
      routeGeneration > 0,
      (1...65535).contains(proxyPort),
      browserProxyLease == nil
    else {
      result(FlutterError(code: "browser_proxy_invalid", message: "The browser proxy binding is invalid or busy", details: nil))
      return
    }
    let endpoint = NWEndpoint.hostPort(
      host: NWEndpoint.Host(proxyHost),
      port: NWEndpoint.Port(rawValue: UInt16(proxyPort))!
    )
    // The proxy receives the original authority, so remote localhost targets
    // are dialed by the daemon rather than by the iOS device.
    let configuration = ProxyConfiguration(httpCONNECTProxy: endpoint, tlsOptions: nil)
    WKWebsiteDataStore.default().proxyConfigurations = [configuration]
    let leaseID = "browser-\(sessionID)-\(UInt64(Date.timeIntervalSinceReferenceDate * 1_000_000))"
    browserProxyLease = leaseID
    result([
      "leaseId": leaseID,
      "sessionId": sessionID,
      "endpointId": endpointID,
      "routeId": routeID,
      "routeGeneration": routeGeneration,
      "dnsProxied": true,
    ])
  }

  private func closeBrowserProxy(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard #available(iOS 17.0, *) else {
      result(nil)
      return
    }
    guard
      let arguments = call.arguments as? [String: Any],
      let leaseID = arguments["leaseId"] as? String,
      browserProxyLease == leaseID
    else {
      result(nil)
      return
    }
    WKWebsiteDataStore.default().proxyConfigurations = []
    browserProxyLease = nil
    result(nil)
  }

  private func handleBackgroundMethod(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    switch call.method {
    case "takePendingRoute":
      let route = pendingNotificationRoute
      pendingNotificationRoute = nil
      result(route)
    case "syncState":
      syncBackgroundState(call, result: result)
    case "notificationsAuthorized":
      notificationAuthorization(result: result, request: false)
    case "requestNotificationAuthorization":
      notificationAuthorization(result: result, request: true)
    case "showNotification":
      showNotification(call, result: result)
    default:
      result(FlutterMethodNotImplemented)
    }
  }

  private func handleExternalURIMethod(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard call.method == "openExternalUri" else {
      result(FlutterMethodNotImplemented)
      return
    }
    guard
      let arguments = call.arguments as? [String: Any],
      let rawURI = arguments["uri"] as? String,
      let url = URL(string: rawURI),
      let scheme = url.scheme?.lowercased(),
      ["http", "https", "mailto", "tel"].contains(scheme)
    else {
      result(FlutterError(code: "invalid_external_uri", message: "External URI scheme is not allowed", details: nil))
      return
    }
    guard UIApplication.shared.canOpenURL(url) else {
      result(FlutterError(code: "open_external_failed", message: "No application can open the external URI", details: nil))
      return
    }
    UIApplication.shared.open(url, options: [:]) { opened in
      if opened {
        result(nil)
      } else {
        result(FlutterError(code: "open_external_failed", message: "Unable to open the external URI", details: nil))
      }
    }
  }

  private func syncBackgroundState(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    let arguments = call.arguments as? [String: Any]
    let enabled = arguments?["enabled"] as? Bool ?? false
    let endpoints = arguments?["endpoints"] as? [[String: Any]] ?? []
    if enabled && !endpoints.isEmpty {
      beginConnectionBackgroundTaskIfNeeded()
    } else {
      endConnectionBackgroundTask()
    }
    result(nil)
  }

  private func beginConnectionBackgroundTaskIfNeeded() {
    guard connectionBackgroundTask == .invalid else { return }
    connectionBackgroundTask = UIApplication.shared.beginBackgroundTask(
      withName: "AnyTTY active connections"
    ) { [weak self] in
      self?.endConnectionBackgroundTask()
    }
  }

  private func endConnectionBackgroundTask() {
    guard connectionBackgroundTask != .invalid else { return }
    let task = connectionBackgroundTask
    connectionBackgroundTask = .invalid
    UIApplication.shared.endBackgroundTask(task)
  }

  private func notificationAuthorization(result: @escaping FlutterResult, request: Bool) {
    let center = UNUserNotificationCenter.current()
    center.getNotificationSettings { settings in
      switch settings.authorizationStatus {
      case .authorized, .provisional, .ephemeral:
        DispatchQueue.main.async { result(true) }
      case .notDetermined where request:
        center.requestAuthorization(options: [.alert, .sound]) { allowed, _ in
          DispatchQueue.main.async { result(allowed) }
        }
      default:
        DispatchQueue.main.async { result(false) }
      }
    }
  }

  private func showNotification(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard
      let arguments = call.arguments as? [String: Any],
      let id = arguments["id"] as? String,
      !id.isEmpty,
      let title = arguments["title"] as? String,
      !title.isEmpty,
      let body = arguments["body"] as? String,
      let route = arguments["route"] as? String,
      route.hasPrefix("/terminal/")
    else {
      result(FlutterError(code: "invalid_notification", message: "Notification data is incomplete", details: nil))
      return
    }
    let content = UNMutableNotificationContent()
    content.title = title
    content.body = body
    content.sound = .default
    content.userInfo = ["route": route]
    let request = UNNotificationRequest(identifier: id, content: content, trigger: nil)
    UNUserNotificationCenter.current().add(request) { error in
      DispatchQueue.main.async {
        if let error {
          result(FlutterError(code: "notification_failed", message: "Unable to show notification", details: error.localizedDescription))
        } else {
          result(nil)
        }
      }
    }
  }

  override func userNotificationCenter(
    _ center: UNUserNotificationCenter,
    willPresent notification: UNNotification,
    withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
  ) {
    completionHandler([])
  }

  override func userNotificationCenter(
    _ center: UNUserNotificationCenter,
    didReceive response: UNNotificationResponse,
    withCompletionHandler completionHandler: @escaping () -> Void
  ) {
    if let route = response.notification.request.content.userInfo["route"] as? String,
       route.hasPrefix("/terminal/") {
      pendingNotificationRoute = route
      backgroundChannel?.invokeMethod("openRoute", arguments: route) { [weak self] _ in
        if self?.pendingNotificationRoute == route {
          self?.pendingNotificationRoute = nil
        }
      }
    }
    completionHandler()
  }

  private func handleFileMethod(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    if call.method == "openFile" {
      openFile(call, result: result)
      return
    }
    guard call.method == "exportFile" else {
      result(FlutterMethodNotImplemented)
      return
    }
    guard pendingExportResult == nil else {
      result(FlutterError(code: "export_busy", message: "Another file export is active", details: nil))
      return
    }
    guard
      let arguments = call.arguments as? [String: Any],
      let sourcePath = arguments["sourcePath"] as? String,
      !sourcePath.isEmpty,
      let sourceURL = allowedExportURL(path: sourcePath)
    else {
      result(FlutterError(code: "invalid_export", message: "Invalid export source", details: nil))
      return
    }
    guard let presenter = activeViewController() else {
      result(FlutterError(code: "export_failed", message: "No active window can present the save dialog", details: nil))
      return
    }

    pendingExportResult = result
    let picker = UIDocumentPickerViewController(forExporting: [sourceURL], asCopy: true)
    picker.delegate = self
    presenter.present(picker, animated: true)
  }

  private func openFile(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
    guard
      let arguments = call.arguments as? [String: Any],
      let rawURI = arguments["uri"] as? String,
      let url = URL(string: rawURI),
      url.isFileURL,
      let presenter = activeViewController()
    else {
      result(FlutterError(code: "invalid_file", message: "Download URI is invalid", details: nil))
      return
    }

    let accessed = url.startAccessingSecurityScopedResource()
    let controller = UIActivityViewController(activityItems: [url], applicationActivities: nil)
    if let popover = controller.popoverPresentationController {
      popover.sourceView = presenter.view
      popover.sourceRect = CGRect(
        x: presenter.view.bounds.midX,
        y: presenter.view.bounds.midY,
        width: 1,
        height: 1
      )
    }
    controller.completionWithItemsHandler = { _, _, _, _ in
      if accessed { url.stopAccessingSecurityScopedResource() }
    }
    presenter.present(controller, animated: true) {
      result(nil)
    }
  }

  private func allowedExportURL(path: String) -> URL? {
    let manager = FileManager.default
    let source = URL(fileURLWithPath: path).resolvingSymlinksInPath().standardizedFileURL
    var isDirectory: ObjCBool = false
    guard manager.fileExists(atPath: source.path, isDirectory: &isDirectory), !isDirectory.boolValue else {
      return nil
    }
    let roots = [
      URL(fileURLWithPath: NSTemporaryDirectory(), isDirectory: true),
      manager.urls(for: .cachesDirectory, in: .userDomainMask).first,
      manager.urls(for: .applicationSupportDirectory, in: .userDomainMask).first,
    ].compactMap { $0?.resolvingSymlinksInPath().standardizedFileURL }
    guard roots.contains(where: { source.path.hasPrefix($0.path + "/") }) else {
      return nil
    }
    return source
  }

  private func activeViewController() -> UIViewController? {
    let root = UIApplication.shared.connectedScenes
      .compactMap { $0 as? UIWindowScene }
      .flatMap(\.windows)
      .first(where: \.isKeyWindow)?
      .rootViewController
    return topViewController(root)
  }

  private func topViewController(_ controller: UIViewController?) -> UIViewController? {
    if let presented = controller?.presentedViewController {
      return topViewController(presented)
    }
    if let navigation = controller as? UINavigationController {
      return topViewController(navigation.visibleViewController)
    }
    if let tab = controller as? UITabBarController {
      return topViewController(tab.selectedViewController)
    }
    return controller
  }

  func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
    finishExport(urls.first?.absoluteString)
  }

  func documentPickerWasCancelled(_ controller: UIDocumentPickerViewController) {
    finishExport(nil)
  }

  private func finishExport(_ value: String?) {
    let result = pendingExportResult
    pendingExportResult = nil
    result?(value)
  }
}
