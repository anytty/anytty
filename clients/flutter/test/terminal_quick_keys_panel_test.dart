import 'package:anytty_native/src/features/terminal/data/terminal_quick_keys_layout_store.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_modifiers.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_quick_action.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_keyboard_inset.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_quick_keys_panel.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('groups Quick Keys and persists drag order only in its layout', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    const actions = [
      TerminalQuickAction(
        id: 'escape',
        kind: TerminalQuickActionKind.key,
        keyId: 'escape',
      ),
      TerminalQuickAction(
        id: 'tab',
        kind: TerminalQuickActionKind.key,
        keyId: 'tab',
      ),
      TerminalQuickAction(
        id: 'delete',
        kind: TerminalQuickActionKind.key,
        keyId: 'delete',
      ),
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
        label: 'Status',
      ),
    ];

    await tester.pumpWidget(
      MaterialApp(
        home: Align(
          alignment: Alignment.topLeft,
          child: SizedBox(
            width: 360,
            child: TerminalQuickKeysPanel(
              actions: actions,
              inputEnabled: true,
              onClose: () {},
              onAction: (_) {},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('KEY'), findsOneWidget);
    expect(find.text('CHORD'), findsOneWidget);
    expect(find.text('TEXT'), findsOneWidget);

    final delete = find.byKey(const ValueKey('terminal-quick-key-delete'));
    final escape = find.byKey(const ValueKey('terminal-quick-key-escape'));
    final gesture = await tester.startGesture(tester.getCenter(escape));
    await tester.pump(const Duration(milliseconds: 320));
    await gesture.moveTo(tester.getCenter(delete));
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();

    expect(await const TerminalQuickKeysLayoutStore().load(), [
      'delete',
      'tab',
      'escape',
      'interrupt',
      'status',
    ]);
  });

  testWidgets(
    'keeps attached Quick Keys fixed while the terminal workspace shifts',
    (tester) async {
      tester.view.devicePixelRatio = 1;
      tester.view.physicalSize = const Size(390, 800);
      addTearDown(tester.view.resetDevicePixelRatio);
      addTearDown(tester.view.resetPhysicalSize);
      final visualInset = ValueNotifier<double>(0);
      addTearDown(visualInset.dispose);

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Stack(
              fit: StackFit.expand,
              children: [
                TerminalKeyboardWorkspace(
                  visualInset: visualInset,
                  child: const SizedBox.expand(
                    key: ValueKey('terminal-workspace'),
                  ),
                ),
                const TerminalHeaderQuickKeysLayer(
                  child: SizedBox(
                    key: ValueKey('quick-keys-header-layer'),
                    height: 120,
                  ),
                ),
              ],
            ),
          ),
        ),
      );

      final terminalBefore = tester.getTopLeft(
        find.byKey(const ValueKey('terminal-workspace')),
      );
      final headerLayerBefore = tester.getRect(
        find.byKey(const ValueKey('quick-keys-header-layer')),
      );
      expect(headerLayerBefore.left, 0);
      expect(headerLayerBefore.right, 390);

      visualInset.value = 300;
      await tester.pump();

      final terminalAfter = tester.getTopLeft(
        find.byKey(const ValueKey('terminal-workspace')),
      );
      final headerLayerAfter = tester.getRect(
        find.byKey(const ValueKey('quick-keys-header-layer')),
      );

      expect(terminalAfter.dy, terminalBefore.dy - 300);
      expect(headerLayerAfter, headerLayerBefore);
    },
  );
}
