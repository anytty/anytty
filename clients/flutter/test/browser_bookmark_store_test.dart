import 'dart:convert';

import 'package:anytty_native/src/features/browser/data/browser_bookmark_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  setUp(() => SharedPreferences.setMockInitialValues({}));

  group('SharedPreferencesBrowserBookmarkStore', () {
    test('keeps saved links first, deduplicated, and bounded', () async {
      const store = SharedPreferencesBrowserBookmarkStore(limit: 2);

      await store.add(
        const BrowserBookmark(url: 'https://one.example', title: 'One'),
      );
      await store.add(
        const BrowserBookmark(url: 'https://two.example', title: 'Two'),
      );
      await store.add(
        const BrowserBookmark(url: 'https://one.example', title: 'Updated'),
      );

      expect(await store.load(), const [
        BrowserBookmark(url: 'https://one.example', title: 'Updated'),
        BrowserBookmark(url: 'https://two.example', title: 'Two'),
      ]);
    });

    test(
      'serializes concurrent writes and supports remove and clear',
      () async {
        const store = SharedPreferencesBrowserBookmarkStore();

        await Future.wait([
          store.add(
            const BrowserBookmark(url: 'https://a.example', title: 'A'),
          ),
          store.add(
            const BrowserBookmark(url: 'https://b.example', title: 'B'),
          ),
        ]);
        expect(await store.load(), hasLength(2));

        await store.remove('https://a.example');
        expect(await store.load(), const [
          BrowserBookmark(url: 'https://b.example', title: 'B'),
        ]);

        await store.clear();
        expect(await store.load(), isEmpty);
      },
    );

    test('skips malformed records while keeping compatible links', () async {
      SharedPreferences.setMockInitialValues({
        'browser.bookmarks.v1': jsonEncode([
          {'url': 'https://valid.example', 'title': 'Valid'},
          {'title': 'Missing URL'},
          'invalid',
        ]),
      });

      const store = SharedPreferencesBrowserBookmarkStore();
      expect(await store.load(), const [
        BrowserBookmark(url: 'https://valid.example', title: 'Valid'),
      ]);
    });
  });
}
