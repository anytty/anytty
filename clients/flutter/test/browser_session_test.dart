import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:anytty_native/src/features/browser/data/browser_session_store.dart';
import 'package:anytty_native/src/features/browser/domain/browser_session.dart';

BrowserSessionSnapshot _snapshot({
  String sessionId = 'device-a',
  String url = 'http://127.0.0.1:9000/app',
}) => BrowserSessionSnapshot(
  sessionId: sessionId,
  endpointId: sessionId,
  endpointLabel: sessionId,
  url: url,
  title: 'Remote app',
  scrollX: 4,
  scrollY: 120,
  snapshotPath: '/tmp/$sessionId.png',
  routeId: 'route-$sessionId',
  routeGeneration: 7,
  parkedAt: DateTime.utc(2026, 9, 3, 8, 0),
);

void main() {
  group('BrowserSessionSnapshot', () {
    test('round trips only serializable browser state', () {
      final original = _snapshot();

      expect(BrowserSessionSnapshot.decode(original.encode()), original);
      expect(original.restorableUri.toString(), 'http://127.0.0.1:9000/app');
    });

    test('rejects non-web schemes before restore', () {
      expect(_snapshot(url: 'javascript:alert(1)').restorableUri, isNull);
      expect(_snapshot(url: 'file:///private/data').restorableUri, isNull);
    });
  });

  group('BrowserSessionStateMachine', () {
    test('ignores stale lifecycle events after a switch', () {
      final machine = BrowserSessionStateMachine();
      final first = machine.begin('device-a');
      final second = machine.begin('device-b');

      machine.markActive(first, 'device-a', _snapshot());
      expect(machine.state.phase, BrowserSessionPhase.parking);
      expect(machine.state.sessionId, 'device-b');

      machine.markRestoring(
        second,
        'device-b',
        _snapshot(sessionId: 'device-b'),
      );
      machine.markActive(second, 'device-b', _snapshot(sessionId: 'device-b'));
      expect(machine.state.phase, BrowserSessionPhase.active);
      expect(machine.state.sessionId, 'device-b');
    });
  });

  group('SharedPreferencesBrowserSessionStore', () {
    setUp(() => SharedPreferences.setMockInitialValues({}));

    test('keeps one bounded record per session', () async {
      const store = SharedPreferencesBrowserSessionStore();
      await store.save(_snapshot());
      await store.save(_snapshot(url: 'https://example.test/next'));
      await store.save(_snapshot(sessionId: 'device-b'));

      expect((await store.load('device-a'))!.url, 'https://example.test/next');
      expect((await store.load('device-b'))!.sessionId, 'device-b');

      await store.remove('device-a');
      expect(await store.load('device-a'), isNull);
    });
  });
}
