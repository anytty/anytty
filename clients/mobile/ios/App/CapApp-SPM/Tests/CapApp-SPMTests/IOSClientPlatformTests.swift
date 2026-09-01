import Foundation
import XCTest
@testable import CapApp_SPM

final class IOSClientPlatformTests: XCTestCase {
    func testShutdownAfterDequeueSkipsPlatformDispatch() {
        var active = true
        var handled = false
        var completed = false

        runIOSClientPlatformPump(
            isActive: { active },
            nextRequest: {
                active = false
                return Data([1])
            },
            handleRequest: { payload in
                handled = true
                return payload
            },
            completeRequest: { _ in completed = true }
        )

        XCTAssertFalse(handled)
        XCTAssertFalse(completed)
    }

    func testRetiredGenerationCannotEnterPersistenceCommitLock() throws {
        let coordinator = IOSPlatformStorageCoordinator()
        let oldGeneration = coordinator.beginGeneration()
        let freshGeneration = coordinator.beginGeneration()
        var persisted = "initial"

        try coordinator.withGeneration(freshGeneration) { persisted = "fresh" }
        XCTAssertThrowsError(try coordinator.withGeneration(oldGeneration) { persisted = "stale" }) { error in
            XCTAssertEqual((error as? AnyTTYPlatformError)?.code, "cancelled")
        }
        XCTAssertEqual(persisted, "fresh")
    }

    func testNewGenerationCommitsAfterOldMutatorOutlivesCloseTimeout() throws {
        let coordinator = IOSPlatformStorageCoordinator()
        let oldGeneration = coordinator.beginGeneration()
        let oldEnteredCommit = DispatchSemaphore(value: 0)
        let releaseOldCommit = DispatchSemaphore(value: 0)
        let oldFinished = DispatchSemaphore(value: 0)
        let freshFinished = DispatchSemaphore(value: 0)
        let valueLock = NSLock()
        var persisted = "initial"

        DispatchQueue.global().async {
            try? coordinator.withGeneration(oldGeneration) {
                oldEnteredCommit.signal()
                releaseOldCommit.wait()
                valueLock.lock()
                persisted = "old"
                valueLock.unlock()
            }
            oldFinished.signal()
        }
        XCTAssertEqual(oldEnteredCommit.wait(timeout: .now() + 1), .success)
        coordinator.retire(oldGeneration)
        XCTAssertEqual(oldFinished.wait(timeout: .now() + .milliseconds(500)), .timedOut)

        let freshGeneration = coordinator.beginGeneration()
        DispatchQueue.global().async {
            try? coordinator.withGeneration(freshGeneration) {
                valueLock.lock()
                persisted = "fresh"
                valueLock.unlock()
            }
            freshFinished.signal()
        }
        XCTAssertEqual(freshFinished.wait(timeout: .now() + .milliseconds(50)), .timedOut)

        releaseOldCommit.signal()
        XCTAssertEqual(oldFinished.wait(timeout: .now() + 1), .success)
        XCTAssertEqual(freshFinished.wait(timeout: .now() + 1), .success)
        valueLock.lock()
        let finalValue = persisted
        valueLock.unlock()
        XCTAssertEqual(finalValue, "fresh")
    }
}
