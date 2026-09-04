import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:anytty_native/src/features/browser/data/browser_history_store.dart';

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  group('SharedPreferencesBrowserHistoryStore', () {
    test('keeps recent entries first, deduplicated, and bounded', () async {
      const store = SharedPreferencesBrowserHistoryStore(limit: 3);

      await store.add(
        const BrowserHistoryEntry(url: 'https://one.example', title: 'One'),
      );
      await store.add(
        const BrowserHistoryEntry(url: 'https://two.example', title: 'Two'),
      );
      await store.add(
        const BrowserHistoryEntry(url: 'https://three.example', title: 'Three'),
      );
      await store.add(
        const BrowserHistoryEntry(
          url: 'https://one.example',
          title: 'One updated',
        ),
      );

      expect(await store.load(), const [
        BrowserHistoryEntry(url: 'https://one.example', title: 'One updated'),
        BrowserHistoryEntry(url: 'https://three.example', title: 'Three'),
        BrowserHistoryEntry(url: 'https://two.example', title: 'Two'),
      ]);
    });

    test('serializes concurrent writes and supports clear', () async {
      const store = SharedPreferencesBrowserHistoryStore();

      await Future.wait([
        store.add(
          const BrowserHistoryEntry(url: 'https://a.example', title: 'A'),
        ),
        store.add(
          const BrowserHistoryEntry(url: 'https://b.example', title: 'B'),
        ),
      ]);
      expect(await store.load(), hasLength(2));

      await store.clear();
      expect(await store.load(), isEmpty);
    });

    test('isolates history by device scope', () async {
      const first = SharedPreferencesBrowserHistoryStore(scope: 'device-a');
      const second = SharedPreferencesBrowserHistoryStore(scope: 'device-b');

      await first.add(
        const BrowserHistoryEntry(url: 'https://a.example', title: 'A'),
      );
      await second.add(
        const BrowserHistoryEntry(url: 'https://b.example', title: 'B'),
      );

      expect(await first.load(), const [
        BrowserHistoryEntry(url: 'https://a.example', title: 'A'),
      ]);
      expect(await second.load(), const [
        BrowserHistoryEntry(url: 'https://b.example', title: 'B'),
      ]);
    });
  });
}
