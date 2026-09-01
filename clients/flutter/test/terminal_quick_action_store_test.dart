import 'dart:convert';

import 'package:anytty_native/src/features/terminal/data/terminal_quick_action_store.dart';
import 'package:anytty_native/src/features/terminal/data/terminal_quick_keys_layout_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_modifiers.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_quick_action.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('loads the default command bar when no configuration exists', () async {
    SharedPreferences.setMockInitialValues({});

    final actions = await const TerminalQuickActionStore().load();

    expect(actions, defaultTerminalQuickActions);
    expect(actions.every((action) => action.isValid), isTrue);
    expect(actions.map((action) => action.displayLabel), contains('Ctrl'));
  });

  test('persists key chords and text submission behavior', () async {
    SharedPreferences.setMockInitialValues({});
    const store = TerminalQuickActionStore();
    const actions = [
      TerminalQuickAction(
        id: 'interrupt',
        kind: TerminalQuickActionKind.chord,
        keyId: 'c',
        modifiers: terminalModifierControlBit,
      ),
      TerminalQuickAction(
        id: 'status',
        kind: TerminalQuickActionKind.text,
        text: 'git status',
        sendEnter: true,
        label: 'Status',
      ),
    ];

    await store.save(actions);
    final loaded = await store.load();

    expect(loaded, hasLength(2));
    expect(loaded.first.displayLabel, 'Ctrl+C');
    expect(loaded.last.displayLabel, 'Status');
    expect(loaded.last.sendEnter, isTrue);
  });

  test(
    'keeps large command bars lazy by not imposing a storage limit',
    () async {
      SharedPreferences.setMockInitialValues({});
      const store = TerminalQuickActionStore();
      final actions = List.generate(
        300,
        (index) => TerminalQuickAction(
          id: 'command-$index',
          kind: TerminalQuickActionKind.text,
          text: 'echo $index',
        ),
      );

      await store.save(actions);

      expect(await store.load(), hasLength(300));
    },
  );

  test('falls back when the stored payload is malformed', () async {
    SharedPreferences.setMockInitialValues({
      TerminalQuickActionStore.storageKey: jsonEncode({
        'version': 1,
        'actions': 'not-a-list',
      }),
    });

    expect(
      await const TerminalQuickActionStore().load(),
      defaultTerminalQuickActions,
    );
  });

  test(
    'persists a Quick Keys order independently from command bar data',
    () async {
      SharedPreferences.setMockInitialValues({});
      const layoutStore = TerminalQuickKeysLayoutStore();

      await layoutStore.save(['second', 'first', 'second', '']);

      expect(await layoutStore.load(), ['second', 'first']);
      expect(
        (await const TerminalQuickActionStore().load()).map(
          (action) => action.id,
        ),
        defaultTerminalQuickActions.map((action) => action.id),
      );
    },
  );

  test('reconciles stored Quick Keys with added and removed commands', () {
    const first = TerminalQuickAction(
      id: 'first',
      kind: TerminalQuickActionKind.key,
      keyId: 'escape',
    );
    const second = TerminalQuickAction(
      id: 'second',
      kind: TerminalQuickActionKind.key,
      keyId: 'tab',
    );
    const added = TerminalQuickAction(
      id: 'added',
      kind: TerminalQuickActionKind.text,
      text: 'pwd',
    );

    expect(
      terminalQuickKeysInLayoutOrder(
        actions: const [first, second, added],
        layoutIds: const ['removed', 'second', 'first'],
      ).map((action) => action.id),
      ['second', 'first', 'added'],
    );
    expect(
      swapTerminalQuickKeyLayout(
        layoutIds: const ['first', 'second', 'added'],
        draggedId: 'first',
        targetId: 'added',
      ),
      ['added', 'second', 'first'],
    );
  });

  test('shows direction keys as arrows with descriptive semantics', () {
    expect(terminalQuickKeyById('left')?.label, '←');
    expect(terminalQuickKeyById('down')?.label, '↓');
    expect(terminalQuickKeyById('up')?.label, '↑');
    expect(terminalQuickKeyById('right')?.label, '→');
    expect(terminalQuickKeyById('left')?.accessibilityLabel, 'Left arrow');
  });
}
