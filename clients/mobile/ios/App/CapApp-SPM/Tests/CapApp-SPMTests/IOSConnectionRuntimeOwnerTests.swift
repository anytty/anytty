import XCTest
@testable import CapApp_SPM

final class IOSConnectionRuntimeOwnerTests: XCTestCase {
    func testRuntimeReplaySignalsHostStateBeforeRestoringDemand() throws {
        var operations: [String] = []

        try replayIOSSupervisorState(
            signalHost: { operations.append("host") },
            submitDemand: { operations.append("demand") }
        )

        XCTAssertEqual(operations, ["host", "demand"])
    }

    func testOrdinaryEmptyDemandDoesNotEnterUserStoppedState() throws {
        let state = IOSRendererDemandState(attachmentIDFactory: { "renderer-a" })
        let attached = state.attachRenderer()
        let active = try state.replaceDemand(
            attachmentID: attached.attachmentID,
            baseDemandRevision: attached.demandRevision,
            endpointIDs: ["machine-a"]
        )
        let empty = try state.replaceDemand(
            attachmentID: active.attachmentID,
            baseDemandRevision: active.demandRevision,
            endpointIDs: []
        )

        let reopened = try state.replaceDemand(
            attachmentID: empty.attachmentID,
            baseDemandRevision: empty.demandRevision,
            endpointIDs: ["machine-b"]
        )

        XCTAssertEqual(reopened.endpointIDs, ["machine-b"])
        XCTAssertEqual(reopened.demandRevision, empty.demandRevision + 1)
        let idempotentResume = try state.resumeDemand(
            attachmentID: reopened.attachmentID,
            intentID: "intent-open",
            baseStopEpoch: reopened.stopEpoch
        )
        XCTAssertEqual(idempotentResume.outcome, .resumed)
        XCTAssertEqual(idempotentResume.snapshot.demandRevision, reopened.demandRevision)
        XCTAssertEqual(idempotentResume.snapshot.endpointIDs, reopened.endpointIDs)
    }

    func testUserStopRejectsNonEmptyDemandUntilExactResume() throws {
        let state = IOSRendererDemandState(attachmentIDFactory: { "renderer-a" })
        let attached = state.attachRenderer()
        let active = try state.replaceDemand(
            attachmentID: attached.attachmentID,
            baseDemandRevision: attached.demandRevision,
            endpointIDs: ["machine-a"]
        )
        let stopped = try XCTUnwrap(state.clearDemandForUserStop())

        XCTAssertEqual(stopped.endpointIDs, [])
        XCTAssertEqual(stopped.demandRevision, active.demandRevision + 1)
        XCTAssertEqual(stopped.stopEpoch, stopped.demandRevision)
        XCTAssertThrowsError(try state.replaceDemand(
            attachmentID: stopped.attachmentID,
            baseDemandRevision: stopped.demandRevision,
            endpointIDs: ["machine-a"]
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .demandStopped)
        }

        let stillStopped = try state.replaceDemand(
            attachmentID: stopped.attachmentID,
            baseDemandRevision: stopped.demandRevision,
            endpointIDs: []
        )
        XCTAssertThrowsError(try state.replaceDemand(
            attachmentID: stillStopped.attachmentID,
            baseDemandRevision: stillStopped.demandRevision,
            endpointIDs: ["machine-a"]
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .demandStopped)
        }

        let resumed = try state.resumeDemand(
            attachmentID: stillStopped.attachmentID,
            intentID: "intent-fresh",
            baseStopEpoch: stillStopped.stopEpoch
        )
        let reopened = try state.replaceDemand(
            attachmentID: resumed.snapshot.attachmentID,
            baseDemandRevision: resumed.snapshot.demandRevision,
            endpointIDs: ["machine-a"]
        )
        XCTAssertEqual(resumed.outcome, .resumed)
        XCTAssertEqual(resumed.snapshot.demandRevision, stillStopped.demandRevision + 1)
        XCTAssertEqual(resumed.snapshot.stopEpoch, stillStopped.stopEpoch)
        XCTAssertEqual(reopened.endpointIDs, ["machine-a"])
    }

    func testResumeIntentIsIdempotentWithinAnEpochAndFailsClosedAcrossUserStop() throws {
        var attachment = 0
        let state = IOSRendererDemandState(attachmentIDFactory: {
            attachment += 1
            return "renderer-\(attachment)"
        })
        let rendererA = state.attachRenderer()
        let accepted = try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "intent-old",
            baseStopEpoch: rendererA.stopEpoch
        )
        XCTAssertEqual(accepted.outcome, .resumed)
        let stopped = try XCTUnwrap(state.clearDemandForUserStop())

        let rejectedRetry = try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "intent-old",
            baseStopEpoch: stopped.stopEpoch
        )
        XCTAssertEqual(rejectedRetry.outcome, .stopped)
        XCTAssertEqual(rejectedRetry.snapshot.demandRevision, stopped.demandRevision)
        let delayed = try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "intent-delayed",
            baseStopEpoch: rendererA.stopEpoch
        )
        XCTAssertEqual(delayed.outcome, .stopped)
        let delayedRetry = try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "intent-delayed",
            baseStopEpoch: stopped.stopEpoch
        )
        XCTAssertEqual(delayedRetry.outcome, .stopped)
        let rendererB = state.attachRenderer()

        XCTAssertThrowsError(try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "intent-old",
            baseStopEpoch: stopped.stopEpoch
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .staleRenderer)
        }
        XCTAssertThrowsError(try state.replaceDemand(
            attachmentID: rendererB.attachmentID,
            baseDemandRevision: rendererB.demandRevision,
            endpointIDs: ["machine-a"]
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .demandStopped)
        }

        let resumed = try state.resumeDemand(
            attachmentID: rendererB.attachmentID,
            intentID: "intent-fresh",
            baseStopEpoch: rendererB.stopEpoch
        )
        XCTAssertEqual(resumed.outcome, .resumed)
        XCTAssertEqual(resumed.snapshot.attachmentID, rendererB.attachmentID)
        XCTAssertEqual(resumed.snapshot.demandRevision, rendererB.demandRevision + 1)
        let duplicate = try state.resumeDemand(
            attachmentID: rendererB.attachmentID,
            intentID: "intent-fresh",
            baseStopEpoch: rendererB.stopEpoch
        )
        XCTAssertEqual(duplicate.outcome, .resumed)
        XCTAssertEqual(duplicate.snapshot.demandRevision, resumed.snapshot.demandRevision)
    }

    func testResumeIntentValidationAndCapacityFailClosed() throws {
        var attachment = 0
        let state = IOSRendererDemandState(attachmentIDFactory: {
            attachment += 1
            return "renderer-\(attachment)"
        })
        let attached = state.attachRenderer()

        XCTAssertThrowsError(try state.resumeDemand(
            attachmentID: attached.attachmentID,
            intentID: "   ",
            baseStopEpoch: attached.stopEpoch
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .invalidResumeIntent)
        }
        XCTAssertThrowsError(try state.resumeDemand(
            attachmentID: attached.attachmentID,
            intentID: String(repeating: "x", count: 129),
            baseStopEpoch: attached.stopEpoch
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .invalidResumeIntent)
        }
        for index in 0..<4096 {
            let result = try state.resumeDemand(
                attachmentID: attached.attachmentID,
                intentID: "intent-\(index)",
                baseStopEpoch: attached.stopEpoch
            )
            XCTAssertEqual(result.outcome, .resumed)
        }
        let stopped = try XCTUnwrap(state.clearDemandForUserStop())
        XCTAssertThrowsError(try state.resumeDemand(
            attachmentID: attached.attachmentID,
            intentID: "intent-over-capacity",
            baseStopEpoch: stopped.stopEpoch
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .resumeIntentCapacityExhausted)
        }
        XCTAssertThrowsError(try state.replaceDemand(
            attachmentID: attached.attachmentID,
            baseDemandRevision: stopped.demandRevision,
            endpointIDs: ["machine-a"]
        )) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .demandStopped)
        }
        let replacement = state.attachRenderer()
        let resumedAfterRotation = try state.resumeDemand(
            attachmentID: replacement.attachmentID,
            intentID: "intent-after-rotation",
            baseStopEpoch: replacement.stopEpoch
        )
        XCTAssertEqual(resumedAfterRotation.outcome, .resumed)
    }

    func testContentProcessRotationPreservesDemandAndInvalidatesOldAttachmentAndResumeIntent() throws {
        var attachment = 0
        let state = IOSRendererDemandState(attachmentIDFactory: {
            attachment += 1
            return "renderer-\(attachment)"
        })
        let rendererA = state.attachRenderer()
        let demanded = try state.replaceDemand(
            attachmentID: rendererA.attachmentID,
            baseDemandRevision: rendererA.demandRevision,
            endpointIDs: ["machine-a"]
        )
        let firstStop = try XCTUnwrap(state.clearDemandForUserStop())
        let firstResume = try state.resumeDemand(
            attachmentID: rendererA.attachmentID,
            intentID: "retry-same-id",
            baseStopEpoch: firstStop.stopEpoch
        )
        let restored = try state.replaceDemand(
            attachmentID: rendererA.attachmentID,
            baseDemandRevision: firstResume.snapshot.demandRevision,
            endpointIDs: demanded.endpointIDs
        )

        let rendererB = try state.rotateRenderer(attachmentID: rendererA.attachmentID)

        XCTAssertEqual(rendererB.endpointIDs, ["machine-a"])
        XCTAssertEqual(rendererB.demandRevision, restored.demandRevision)
        XCTAssertThrowsError(try state.currentSnapshot(attachmentID: rendererA.attachmentID)) { error in
            XCTAssertEqual(error as? IOSConnectionRuntimeError, .staleRenderer)
        }

        let secondStop = try XCTUnwrap(state.clearDemandForUserStop())
        let reusedIntentID = try state.resumeDemand(
            attachmentID: rendererB.attachmentID,
            intentID: "retry-same-id",
            baseStopEpoch: secondStop.stopEpoch
        )
        XCTAssertEqual(reusedIntentID.outcome, .resumed)
    }

    func testDuplicatePathSnapshotDoesNotProduceTransition() {
        let path = IOSNetworkPathDescriptor(
            connected: true,
            fingerprint: "status=satisfied;active=wifi;available=wifi:en0:4;dns=true"
        )

        XCTAssertNil(iosNetworkTransitionReason(from: path, to: path))
    }

    func testOfflineToOnlinePathBecomesAvailable() {
        let offline = IOSNetworkPathDescriptor(connected: false, fingerprint: "status=unsatisfied")
        let online = IOSNetworkPathDescriptor(connected: true, fingerprint: "status=satisfied;active=wifi")

        XCTAssertEqual(iosNetworkTransitionReason(from: offline, to: online), "available")
    }

    func testMaterialOnlinePathChangeReplacesNetwork() {
        let wifi = IOSNetworkPathDescriptor(connected: true, fingerprint: "status=satisfied;active=wifi")
        let cellular = IOSNetworkPathDescriptor(connected: true, fingerprint: "status=satisfied;active=cellular")

        XCTAssertEqual(iosNetworkTransitionReason(from: wifi, to: cellular), "network_replaced")
    }
}
