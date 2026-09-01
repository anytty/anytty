import 'package:anytty_native/src/features/terminal/data/terminal_petal_menu_preferences_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_petal_menu_preferences.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('defaults expose a three-level action tree', () {
    final defaults = TerminalPetalMenuPreferences.defaults;

    expect(defaults.visibleActionCount, terminalPetalActionIds.length);
    expect(defaults.visibleRootActionIds, [
      'history',
      'search',
      'selection',
      'paste',
      'quick-keys',
      'keyboard',
      'resources',
      'more',
    ]);
    expect(defaults.layout.firstWhere((item) => item.id == 'enter').depth, 2);
    expect(
      defaults.layout.firstWhere((item) => item.id == 'page-down').depth,
      2,
    );
    expect(defaults.layout.firstWhere((item) => item.id == 'split').depth, 1);
    expect(
      defaults.layout.firstWhere((item) => item.id == 'split-rows').depth,
      2,
    );
    expect(
      defaults.layout.firstWhere((item) => item.id == 'split-columns').depth,
      2,
    );
    expect(defaults.enabled, isTrue);
    expect(defaults.hapticsEnabled, isTrue);
  });

  test('migrates legacy root and More ordering into the action tree', () {
    final preferences = TerminalPetalMenuPreferences.fromJson({
      'enabled': false,
      'hapticsEnabled': false,
      'rootOrder': ['paste', 'unknown', 'paste'],
      'moreOrder': ['settings', 'unknown'],
      'hiddenActionIds': ['paste', 'unknown'],
    });

    expect(preferences.visibleRootActionIds, [
      'history',
      'search',
      'more',
      'selection',
      'resources',
    ]);
    expect(
      preferences.layout
          .where((item) => item.depth == 0)
          .map((item) => item.id),
      ['paste', 'history', 'search', 'more', 'selection', 'resources'],
    );
    final moreIndex = preferences.layout.indexWhere(
      (item) => item.id == 'more',
    );
    expect(
      preferences.layout[moreIndex + 1],
      const TerminalPetalMenuPlacement(id: 'settings', depth: 1),
    );
    expect(preferences.hiddenActionIds, {'paste'});
    expect(
      preferences.layout.map((item) => item.id).toSet(),
      terminalPetalActionIds.toSet(),
    );
  });

  test('upgrades the untouched version two default to eight main petals', () {
    final preferences = TerminalPetalMenuPreferences.fromJson({
      'version': 2,
      'layout': [
        for (final placement in const [
          TerminalPetalMenuPlacement(id: 'history', depth: 0),
          TerminalPetalMenuPlacement(id: 'search', depth: 0),
          TerminalPetalMenuPlacement(id: 'more', depth: 0),
          TerminalPetalMenuPlacement(id: 'command-bar', depth: 1),
          TerminalPetalMenuPlacement(id: 'resize', depth: 1),
          TerminalPetalMenuPlacement(id: 'settings', depth: 1),
          TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
          TerminalPetalMenuPlacement(id: 'quick-keys', depth: 2),
          TerminalPetalMenuPlacement(id: 'keyboard', depth: 2),
          TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
          TerminalPetalMenuPlacement(id: 'clear', depth: 2),
          TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
          TerminalPetalMenuPlacement(id: 'split', depth: 2),
          TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
          TerminalPetalMenuPlacement(id: 'selection', depth: 0),
          TerminalPetalMenuPlacement(id: 'paste', depth: 0),
        ])
          placement.toJson(),
      ],
    });

    expect(preferences.layout, terminalPetalDefaultLayout);
    expect(preferences.visibleRootActionIds, hasLength(8));
  });

  test('upgrades the untouched version three default layout', () {
    final oldVersionThreeLayout = _versionFourDefaultFixture();
    oldVersionThreeLayout.removeWhere(
      (placement) => placement.id == 'enter' || placement.id == 'escape',
    );
    final resourceIndex = oldVersionThreeLayout.indexWhere(
      (placement) => placement.id == 'resources',
    );
    oldVersionThreeLayout.insertAll(resourceIndex, const [
      TerminalPetalMenuPlacement(id: 'enter', depth: 0),
      TerminalPetalMenuPlacement(id: 'escape', depth: 0),
    ]);

    final preferences = TerminalPetalMenuPreferences.fromJson({
      'version': 3,
      'layout': [
        for (final placement in oldVersionThreeLayout) placement.toJson(),
      ],
    });

    expect(preferences.layout, terminalPetalDefaultLayout);
    expect(preferences.visibleRootActionIds, hasLength(8));
    expect(preferences.toJson()['version'], 5);
  });

  test('upgrades the untouched version four default layout', () {
    final preferences = TerminalPetalMenuPreferences.fromJson({
      'version': 4,
      'layout': [
        for (final placement in _versionFourDefaultFixture())
          placement.toJson(),
      ],
    });

    expect(preferences.layout, terminalPetalDefaultLayout);
    expect(preferences.toJson()['version'], 5);
  });

  test('does not cap the number of petals in one ring', () {
    final preferences = TerminalPetalMenuPreferences.defaults.copyWith(
      layout: [
        for (final id in terminalPetalActionIds)
          TerminalPetalMenuPlacement(id: id, depth: 0),
      ],
    );

    expect(preferences.visibleRootActionIds, terminalPetalActionIds);
  });

  test(
    'normalizes malformed version two layouts and appends missing actions',
    () {
      final preferences = TerminalPetalMenuPreferences.fromJson({
        'version': 2,
        'layout': [
          {'id': 'paste', 'depth': 2},
          {'id': 'search', 'depth': 8},
          {'id': 'paste', 'depth': 0},
          {'id': 'unknown', 'depth': 0},
        ],
      });

      expect(preferences.layout.take(2), const [
        TerminalPetalMenuPlacement(id: 'paste', depth: 0),
        TerminalPetalMenuPlacement(id: 'search', depth: 1),
      ]);
      expect(
        preferences.layout.map((item) => item.id).toSet(),
        terminalPetalActionIds.toSet(),
      );
    },
  );

  test('moves a parent together with its complete subtree', () {
    final moved = moveTerminalPetalSubtree(
      terminalPetalDefaultLayout,
      actionId: 'input-tools',
      offset: -1,
    );
    final filesIndex = moved.indexWhere((item) => item.id == 'files');
    final inputIndex = moved.indexWhere((item) => item.id == 'input-tools');

    expect(inputIndex, lessThan(filesIndex));
    expect(moved.sublist(inputIndex, inputIndex + 10).map((item) => item.id), [
      'input-tools',
      'enter',
      'escape',
      'tab',
      'backspace',
      'delete',
      'interrupt',
      'eof',
      'suspend',
      'clear',
    ]);
  });

  test('indents and promotes arbitrary action subtrees up to level three', () {
    const source = <TerminalPetalMenuPlacement>[
      TerminalPetalMenuPlacement(id: 'history', depth: 0),
      TerminalPetalMenuPlacement(id: 'search', depth: 0),
      TerminalPetalMenuPlacement(id: 'paste', depth: 0),
    ];

    final levelTwo = indentTerminalPetalSubtree(source, actionId: 'search');
    final pasteAtLevelTwo = indentTerminalPetalSubtree(
      levelTwo,
      actionId: 'paste',
    );
    final levelThree = indentTerminalPetalSubtree(
      pasteAtLevelTwo,
      actionId: 'paste',
    );
    expect(levelThree.map((item) => item.depth), [0, 1, 2]);
    expect(
      outdentTerminalPetalSubtree(
        levelThree,
        actionId: 'search',
      ).map((item) => item.depth),
      [0, 0, 1],
    );
  });

  test('promotes visible descendants when their parent is hidden', () {
    final preferences = TerminalPetalMenuPreferences.defaults.copyWith(
      hiddenActionIds: const {'input-tools'},
    );

    expect(
      preferences.visibleLayout.firstWhere((item) => item.id == 'tab').depth,
      1,
    );
  });

  test('outdenting a branch does not capture its former siblings', () {
    final promoted = outdentTerminalPetalSubtree(
      terminalPetalDefaultLayout,
      actionId: 'input-tools',
    );
    final session = promoted.firstWhere((item) => item.id == 'session-tools');
    final input = promoted.firstWhere((item) => item.id == 'input-tools');

    expect(session.depth, 1);
    expect(input.depth, 0);
    expect(promoted.indexOf(session), lessThan(promoted.indexOf(input)));
  });

  test(
    'moves complete petals inside another petal without creating cycles',
    () {
      final secondRing = reparentTerminalPetalSubtree(
        terminalPetalDefaultLayout,
        actionId: 'search',
        parentActionId: 'selection',
      );
      final thirdRing = reparentTerminalPetalSubtree(
        secondRing,
        actionId: 'paste',
        parentActionId: 'search',
      );

      expect(thirdRing.firstWhere((item) => item.id == 'selection').depth, 0);
      expect(thirdRing.firstWhere((item) => item.id == 'search').depth, 1);
      expect(thirdRing.firstWhere((item) => item.id == 'paste').depth, 2);
      expect(
        reparentTerminalPetalSubtree(
          thirdRing,
          actionId: 'selection',
          parentActionId: 'paste',
        ),
        thirdRing,
      );
    },
  );

  test('rejects a petal branch that would extend beyond the third ring', () {
    expect(
      reparentTerminalPetalSubtree(
        terminalPetalDefaultLayout,
        actionId: 'more',
        parentActionId: 'selection',
      ),
      terminalPetalDefaultLayout,
    );
  });

  test('persists and reloads a normalized petal menu configuration', () async {
    final storage = _MemoryStorage();
    final store = TerminalPetalMenuPreferencesStore(storage: storage);
    final value = TerminalPetalMenuPreferences.defaults.copyWith(
      enabled: false,
      hapticsEnabled: false,
      layout: moveTerminalPetalSubtree(
        terminalPetalDefaultLayout,
        actionId: 'paste',
        offset: -1,
      ),
      hiddenActionIds: const {'search', 'resize'},
    );

    expect(await store.save(value), value);
    expect(await store.load(), value);
    expect(
      storage.values,
      contains(TerminalPetalMenuPreferencesStore.storageKey),
    );
  });

  test('invalid persisted preferences fall back without throwing', () async {
    final storage = _MemoryStorage()
      ..values[TerminalPetalMenuPreferencesStore.storageKey] = '{invalid';
    final store = TerminalPetalMenuPreferencesStore(storage: storage);

    expect(await store.load(), TerminalPetalMenuPreferences.defaults);
  });

  test('failed petal preference writes surface an error', () async {
    final store = TerminalPetalMenuPreferencesStore(
      storage: _MemoryStorage(writeSucceeds: false),
    );

    await expectLater(
      store.save(TerminalPetalMenuPreferences.defaults),
      throwsStateError,
    );
  });
}

List<TerminalPetalMenuPlacement> _versionFourDefaultFixture() {
  final layout = terminalPetalDefaultLayout
      .where(
        (placement) =>
            placement.id != 'split' &&
            placement.id != 'split-rows' &&
            placement.id != 'split-columns',
      )
      .toList();
  final sessionIndex = layout.indexWhere(
    (placement) => placement.id == 'session-tools',
  );
  layout.insert(
    sessionIndex + 1,
    const TerminalPetalMenuPlacement(id: 'split', depth: 2),
  );
  return layout;
}

final class _MemoryStorage implements TerminalPetalMenuPreferencesStorage {
  _MemoryStorage({this.writeSucceeds = true});

  final bool writeSucceeds;
  final Map<String, String> values = <String, String>{};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<bool> write(String key, String value) async {
    if (writeSucceeds) values[key] = value;
    return writeSucceeds;
  }
}
