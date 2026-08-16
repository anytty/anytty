import Capacitor
import UIKit
import WebKit

@objc(AnyTTYBridgeViewController)
public final class AnyTTYBridgeViewController: CAPBridgeViewController {
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
    }
}
