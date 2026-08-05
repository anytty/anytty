import Capacitor
import UIKit

@objc(NativeHapticPlugin)
public final class NativeHapticPlugin: CAPPlugin, CAPBridgedPlugin {
    public let identifier = "NativeHapticPlugin"
    public let jsName = "NativeHaptic"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "impact", returnType: CAPPluginReturnPromise),
    ]

    @objc func impact(_ call: CAPPluginCall) {
        DispatchQueue.main.async {
            if call.options["pattern"] is [Any] {
                UINotificationFeedbackGenerator().notificationOccurred(.error)
            } else if let duration = call.options["pattern"] as? NSNumber, duration.intValue >= 20 {
                UIImpactFeedbackGenerator(style: .medium).impactOccurred()
            } else {
                UISelectionFeedbackGenerator().selectionChanged()
            }
            call.resolve()
        }
    }
}
