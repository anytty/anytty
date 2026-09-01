import 'dart:convert';

import 'package:anytty_native/src/features/files/data/path_bookmark_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'stores bookmarks per endpoint using the Web storage contract',
    () async {
      SharedPreferences.setMockInitialValues({});
      const store = PathBookmarkStore();

      final saved = await store.add('endpoint-a', '/srv/app/', label: 'Prod');

      expect(saved.path, '/srv/app');
      expect(saved.label, 'Prod');
      expect(await store.load('endpoint-b'), isEmpty);
      final preferences = await SharedPreferences.getInstance();
      final raw = preferences.getString('anytty.path-bookmarks.v1:endpoint-a');
      expect(
        (jsonDecode(raw!) as List).single,
        containsPair('path', '/srv/app'),
      );
    },
  );

  test('adding an existing path updates instead of duplicating it', () async {
    SharedPreferences.setMockInitialValues({});
    const store = PathBookmarkStore();

    final original = await store.add('endpoint-a', '/srv/app');
    final updated = await store.add(
      'endpoint-a',
      '/srv/app/',
      label: 'Production',
    );

    expect(updated.id, original.id);
    expect(updated.version, 2);
    expect(await store.load('endpoint-a'), hasLength(1));
    expect((await store.load('endpoint-a')).single.label, 'Production');
  });

  test('renames and removes a bookmark while preserving its path', () async {
    SharedPreferences.setMockInitialValues({});
    const store = PathBookmarkStore();
    final bookmark = await store.add('endpoint-a', '/srv/app');

    final renamed = await store.rename('endpoint-a', bookmark.id, 'Production');
    expect(renamed.path, '/srv/app');
    expect(renamed.label, 'Production');

    await store.remove('endpoint-a', bookmark.id);
    expect(await store.load('endpoint-a'), isEmpty);
  });

  test('loads compatible records and derives missing labels', () async {
    SharedPreferences.setMockInitialValues({
      'anytty.path-bookmarks.v1:endpoint-a': jsonEncode([
        {'id': 'one', 'path': '/srv/app/', 'version': 4},
      ]),
    });

    final bookmark = (await const PathBookmarkStore().load('endpoint-a'))
        .single;
    expect(bookmark.path, '/srv/app');
    expect(bookmark.label, 'app');
    expect(bookmark.version, 4);
  });
}
