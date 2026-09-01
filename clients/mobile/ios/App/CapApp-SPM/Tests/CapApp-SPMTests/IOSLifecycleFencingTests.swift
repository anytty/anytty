import WebKit
import XCTest
@testable import CapApp_SPM

final class IOSLifecycleFencingTests: XCTestCase {
    private enum ExpectedFailure: Error {
        case failed
    }

    func testRendererCallFenceRejectsQueuedWorkFromTerminatedContentProcess() {
        let fence = IOSRendererCallFence()
        let terminatedRenderer = fence.capture()

        let replacementRenderer = fence.rotate()

        XCTAssertFalse(fence.accepts(terminatedRenderer))
        XCTAssertTrue(fence.accepts(replacementRenderer))
    }

    func testBackgroundInvalidatesDelayedForegroundAcknowledgement() throws {
        let fence = IOSForegroundResumeFence(isForeground: true)
        let foregroundAcknowledgement = try XCTUnwrap(fence.capture())

        fence.enterBackground()

        XCTAssertFalse(fence.accepts(foregroundAcknowledgement))
        XCTAssertNil(fence.capture())
        var stalePulseCount = 0
        let staleResult = fence.performIfAccepted(foregroundAcknowledgement) {
            stalePulseCount += 1
            return true
        }
        XCTAssertNil(staleResult)
        XCTAssertEqual(stalePulseCount, 0)

        let nextForeground = fence.enterForeground()
        XCTAssertTrue(fence.accepts(nextForeground))
        XCTAssertFalse(fence.accepts(foregroundAcknowledgement))
        XCTAssertEqual(fence.performIfAccepted(nextForeground, operation: { true }), true)
    }

    func testPairingResetStopsOldRuntimeWhenPersistentCleanupFails() {
        var replacementAttempts = 0
        var stopCount = 0

        XCTAssertThrowsError(try runIOSPairingResetTransaction(
            clearPersistentState: { throw ExpectedFailure.failed },
            replaceRuntime: { replacementAttempts += 1 },
            stopRuntimeAfterFailure: { stopCount += 1 }
        ))

        XCTAssertEqual(replacementAttempts, 0)
        XCTAssertEqual(stopCount, 1)
    }

    func testPairingResetStopsOldRuntimeWhenReplacementFails() {
        var stopCount = 0

        XCTAssertThrowsError(try runIOSPairingResetTransaction(
            clearPersistentState: {},
            replaceRuntime: { throw ExpectedFailure.failed },
            stopRuntimeAfterFailure: { stopCount += 1 }
        ))

        XCTAssertEqual(stopCount, 1)
    }

    func testSuccessfulPairingResetDoesNotStopReplacementRuntime() throws {
        var stopCount = 0

        try runIOSPairingResetTransaction(
            clearPersistentState: {},
            replaceRuntime: {},
            stopRuntimeAfterFailure: { stopCount += 1 }
        )

        XCTAssertEqual(stopCount, 0)
    }

    @MainActor
    func testContentProcessTerminationHookRunsBeforeCapacitorDelegate() {
        var operations: [String] = []
        let downstream = ContentProcessTerminationRecorder {
            operations.append("capacitor")
        }
        let proxy = AnyTTYWebViewNavigationDelegateProxy(
            downstream: downstream,
            onContentProcessTermination: { operations.append("fence") }
        )

        proxy.webViewWebContentProcessDidTerminate(WKWebView())

        XCTAssertEqual(operations, ["fence", "capacitor"])
    }
}

private final class ContentProcessTerminationRecorder: NSObject, WKNavigationDelegate {
    private let onTermination: () -> Void

    init(onTermination: @escaping () -> Void) {
        self.onTermination = onTermination
    }

    func webViewWebContentProcessDidTerminate(_ webView: WKWebView) {
        onTermination()
    }
}
