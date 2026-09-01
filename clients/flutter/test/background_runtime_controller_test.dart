import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart'
    as binding;
import 'package:anytty_native/src/native/background_runtime_controller.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('projects an exited terminal into an exact terminal route', () {
    final notification = projectBackgroundNotification(
      EventEnvelope(
        eventId: 'event-exit-1',
        terminalLifecycle: TerminalLifecycleEvent(
          terminal: TerminalInfo(
            ref: TerminalRef(endpointId: 'office mac', terminalId: 'build/42'),
            name: 'Release build',
            state: TerminalState.TERMINAL_STATE_EXITED,
            exitCode: 7,
          ),
        ),
      ),
    );

    expect(notification, isNotNull);
    expect(notification!.title, 'Terminal finished');
    expect(notification.body, 'Release build exited with code 7');
    expect(notification.route, '/terminal/office%20mac/build%2F42');
  });

  test('does not notify for running or removed terminal projections', () {
    for (final state in <TerminalState>[
      TerminalState.TERMINAL_STATE_RUNNING,
      TerminalState.TERMINAL_STATE_REMOVED,
    ]) {
      expect(
        projectBackgroundNotification(
          EventEnvelope(
            eventId: 'event-${state.value}',
            terminalLifecycle: TerminalLifecycleEvent(
              terminal: TerminalInfo(
                ref: TerminalRef(endpointId: 'a', terminalId: 'b'),
                state: state,
              ),
            ),
          ),
        ),
        isNull,
      );
    }
  });

  test('projects transfer completion to the owning endpoint', () {
    final notification = projectBackgroundNotification(
      EventEnvelope(
        eventId: 'transfer-1',
        originSession: EndpointSessionStamp(endpointId: 'studio'),
        fileTransferCompleted: FileTransferCompletedEvent(
          transfer: FileTransferHandle(path: '/tmp/result.tar'),
        ),
      ),
    );

    expect(notification, isNotNull);
    expect(notification!.title, 'Transfer complete');
    expect(notification.body, '/tmp/result.tar');
    expect(notification.route, '/terminal/studio');
  });

  test('rejects replay-unsafe events without an event id', () {
    expect(
      projectBackgroundNotification(
        EventEnvelope(
          terminalLifecycle: TerminalLifecycleEvent(
            terminal: TerminalInfo(
              ref: TerminalRef(endpointId: 'a', terminalId: 'b'),
              state: TerminalState.TERMINAL_STATE_EXITED,
            ),
          ),
        ),
      ),
      isNull,
    );
  });

  test('fences background events to the newest opened session generation', () {
    final fence = BackgroundSessionGenerationFence();
    final first = EndpointSessionStamp(
      endpointId: 'studio',
      routeId: 'direct',
      generation: Int64(7),
    );
    final replacement = EndpointSessionStamp(
      endpointId: 'studio',
      routeId: 'cloud',
      generation: Int64(8),
    );

    fence.applicationEventForCurrentSession(
      binding.EventEnvelope(
        openSession: binding.OpenSessionResult(
          sessionHandle: Int64(21),
          session: first,
        ),
      ),
    );
    expect(
      fence
          .applicationEventForCurrentSession(
            _bindingApplicationEvent('first-current', 21, first),
          )
          ?.eventId,
      'first-current',
    );

    fence.applicationEventForCurrentSession(
      binding.EventEnvelope(
        openSession: binding.OpenSessionResult(
          sessionHandle: Int64(22),
          session: replacement,
        ),
      ),
    );

    expect(
      fence.applicationEventForCurrentSession(
        _bindingApplicationEvent('late-first', 21, first),
      ),
      isNull,
    );
    expect(
      fence
          .applicationEventForCurrentSession(
            _bindingApplicationEvent('replacement-current', 22, replacement),
          )
          ?.eventId,
      'replacement-current',
    );
  });
}

binding.EventEnvelope _bindingApplicationEvent(
  String eventId,
  int sessionHandle,
  EndpointSessionStamp stamp,
) {
  return binding.EventEnvelope(
    application: binding.ApplicationEvent(
      sessionHandle: Int64(sessionHandle),
      event: EventEnvelope(eventId: eventId, originSession: stamp),
    ),
  );
}
