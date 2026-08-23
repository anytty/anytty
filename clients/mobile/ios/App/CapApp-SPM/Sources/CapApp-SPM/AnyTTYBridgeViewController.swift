import Capacitor
import UIKit
import WebKit

@objc(AnyTTYBridgeViewController)
public final class AnyTTYBridgeViewController: CAPBridgeViewController {
    private var keyboardObservers: [NSObjectProtocol] = []

    public override func capacitorDidLoad() {
        let fontFallback = WKUserScript(
            source: """
            (() => {
              const style = document.createElement("style");
              style.textContent = `
                html, body, #root, .font-sans {
                  font-family: "PingFang SC", -apple-system, BlinkMacSystemFont,
                    "Helvetica Neue", Arial, sans-serif !important;
                }
              `;
              (document.head || document.documentElement).appendChild(style);
            })();
            """,
            injectionTime: .atDocumentEnd,
            forMainFrameOnly: true
        )
        webView?.configuration.userContentController.addUserScript(fontFallback)
        bridge?.registerPluginInstance(NativeConnectionPlugin())
        bridge?.registerPluginInstance(NativeFilePickerPlugin())
        bridge?.registerPluginInstance(NativeHapticPlugin())
        webView?.scrollView.bounces = false
        webView?.scrollView.showsVerticalScrollIndicator = false
        webView?.scrollView.showsHorizontalScrollIndicator = false
        observeKeyboardGeometry()
    }

    deinit {
        keyboardObservers.forEach(NotificationCenter.default.removeObserver)
    }

    private func observeKeyboardGeometry() {
        keyboardObservers.forEach(NotificationCenter.default.removeObserver)
        keyboardObservers.removeAll()
        let center = NotificationCenter.default
        keyboardObservers.append(center.addObserver(
            forName: UIResponder.keyboardWillChangeFrameNotification,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            self?.publishKeyboardGeometry(notification)
        })
        keyboardObservers.append(center.addObserver(
            forName: UIResponder.keyboardDidChangeFrameNotification,
            object: nil,
            queue: .main
        ) { [weak self] notification in
            self?.publishKeyboardGeometry(notification)
        })
        keyboardObservers.append(center.addObserver(
            forName: UIResponder.keyboardWillHideNotification,
            object: nil,
            queue: .main
        ) { [weak self] _ in
            self?.dispatchKeyboardGeometry(visible: false, occludedHeight: 0)
        })
    }

    private func publishKeyboardGeometry(_ notification: Notification) {
        guard
            let webView,
            let window = webView.window,
            let endFrame = notification.userInfo?[UIResponder.keyboardFrameEndUserInfoKey] as? CGRect
        else { return }

        let frameInWindow = window.convert(endFrame, from: window.screen.coordinateSpace)
        let frameInWebView = webView.convert(frameInWindow, from: window)
        let intersection = webView.bounds.intersection(frameInWebView)
        let visible = !intersection.isNull && !intersection.isEmpty
        let reachesBottom = visible && abs(intersection.maxY - webView.bounds.maxY) <= 1
        let coversWidth = visible && intersection.width >= webView.bounds.width * 0.9
        let occludedHeight = reachesBottom && coversWidth ? intersection.height : 0
        dispatchKeyboardGeometry(visible: visible, occludedHeight: occludedHeight)
    }

    private func dispatchKeyboardGeometry(visible: Bool, occludedHeight: CGFloat) {
        let script = """
        window.dispatchEvent(new CustomEvent("anytty:native-keyboard", {
          detail: { visible: \(visible ? "true" : "false"), occludedHeight: \(max(0, Int(occludedHeight.rounded()))) }
        }));
        """
        webView?.evaluateJavaScript(script)
    }
}
