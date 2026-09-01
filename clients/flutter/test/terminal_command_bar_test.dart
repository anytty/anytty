import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_modifiers.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_quick_action.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_command_bar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('runs any visible command with one tap', (tester) async {
    final executed = <String>[];
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
        id: 'interrupt',
        kind: TerminalQuickActionKind.chord,
        keyId: 'c',
        modifiers: terminalModifierControlBit,
      ),
      TerminalQuickAction(
        id: 'status',
        kind: TerminalQuickActionKind.text,
        text: 'git status',
      ),
    ];
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: actions,
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(),
          onAction: (action) => executed.add(action.id),
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );

    await tester.tap(find.text('Tab').first);
    await tester.pump();
    expect(executed, ['tab']);
  });

  testWidgets('handles a fling without leaving its bounded range', (
    tester,
  ) async {
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: defaultTerminalQuickActions,
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );

    final scrollable = find.descendant(
      of: find.byKey(const ValueKey('command-reel')),
      matching: find.byType(Scrollable),
    );
    final position = tester.state<ScrollableState>(scrollable).position;
    await tester.fling(
      find.byKey(const ValueKey('command-reel')),
      const Offset(-240, 0),
      1800,
    );
    await tester.pump(const Duration(milliseconds: 100));

    expect(position.pixels, greaterThan(0));
    expect(position.pixels, lessThanOrEqualTo(position.maxScrollExtent));
  });

  testWidgets('keeps the command reel bounded at both ends', (tester) async {
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: defaultTerminalQuickActions,
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );

    final reel = find.byKey(const ValueKey('command-reel'));
    final scrollable = find.descendant(
      of: reel,
      matching: find.byType(Scrollable),
    );
    final position = tester.state<ScrollableState>(scrollable).position;
    expect(position.pixels, 0);
    expect(
      position.maxScrollExtent,
      lessThanOrEqualTo(defaultTerminalQuickActions.length * 52),
    );

    await tester.drag(reel, const Offset(300, 0));
    await tester.pumpAndSettle();
    expect(position.pixels, 0);

    position.jumpTo(position.maxScrollExtent);
    await tester.drag(reel, const Offset(-300, 0));
    await tester.pumpAndSettle();
    expect(position.pixels, position.maxScrollExtent);
  });

  testWidgets('places Fn left and the keyboard control right', (tester) async {
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: defaultTerminalQuickActions,
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(
            key: ValueKey('keyboard-anchor'),
          ),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );

    expect(
      tester.getCenter(find.text('Fn')).dx,
      lessThan(tester.getCenter(find.byKey(const ValueKey('command-reel'))).dx),
    );
    expect(
      tester.getCenter(find.byKey(const ValueKey('keyboard-anchor'))).dx,
      greaterThan(
        tester.getCenter(find.byKey(const ValueKey('command-reel'))).dx,
      ),
    );
  });

  testWidgets('keeps function keys on the compact fixed edge', (tester) async {
    var fnPressed = false;
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: defaultTerminalQuickActions,
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () => fnPressed = true,
        ),
      ),
    );

    expect(tester.getSize(find.byType(TerminalCommandBar)).height, 39);
    await tester.tap(find.text('Fn'));
    await tester.pump();
    expect(fnPressed, isTrue);
  });

  testWidgets('shows an active modifier without changing the bar height', (
    tester,
  ) async {
    const action = TerminalQuickAction(
      id: 'control',
      kind: TerminalQuickActionKind.key,
      keyId: 'control',
    );
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: const [action],
          inputEnabled: true,
          modifiers: const TerminalModifierState(
            control: TerminalModifierLatch.locked,
          ),
          keyboardControl: const SizedBox.expand(),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );

    final inactiveHeight = tester
        .getSize(find.byType(TerminalCommandBar))
        .height;
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandBar(
          actions: const [action],
          inputEnabled: true,
          modifiers: const TerminalModifierState(),
          keyboardControl: const SizedBox.expand(),
          onAction: (_) {},
          onConfigure: () {},
          functionKeysActive: false,
          onFunctionKeys: () {},
        ),
      ),
    );
    expect(
      tester.getSize(find.byType(TerminalCommandBar)).height,
      inactiveHeight,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('creates a text command that submits Enter', (tester) async {
    List<TerminalQuickAction>? saved;
    await tester.pumpWidget(
      _Harness(
        child: TerminalCommandEditorTray(
          actions: const [],
          onSave: (actions) async => saved = actions,
        ),
      ),
    );

    await tester.tap(find.text('Add command'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Text').last);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byKey(const ValueKey('command-text-field')),
      'git status',
    );
    await tester.pump();
    expect(
      tester
          .widget<TextField>(find.byKey(const ValueKey('command-text-field')))
          .controller!
          .text,
      'git status',
    );
    await tester.tap(find.text('Send Enter after text'));
    final applyButton = tester.widget<TextButton>(
      find.ancestor(of: find.text('Apply'), matching: find.byType(TextButton)),
    );
    expect(applyButton.onPressed, isNotNull);
    await tester.tap(find.text('Apply'));
    await tester.pumpAndSettle();
    expect(find.byKey(const ValueKey('command-editor-list')), findsOneWidget);
    await tester.tap(find.text('Done'));
    await tester.pumpAndSettle();

    expect(saved, hasLength(1));
    expect(saved!.single.kind, TerminalQuickActionKind.text);
    expect(saved!.single.text, 'git status');
    expect(saved!.single.sendEnter, isTrue);
  });
}

final class _Harness extends StatelessWidget {
  const _Harness({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) => MaterialApp(
    theme: anyttyTheme(Brightness.light),
    darkTheme: anyttyTheme(Brightness.dark),
    themeMode: ThemeMode.dark,
    home: Scaffold(body: child),
  );
}
