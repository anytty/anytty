import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_petal_menu.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

const _actions = <TerminalPetalMenuItem>[
  TerminalPetalMenuItem(
    id: 'history',
    label: 'History',
    icon: LucideIcons.history,
  ),
  TerminalPetalMenuItem(
    id: 'search',
    label: 'Search',
    icon: LucideIcons.search,
  ),
  TerminalPetalMenuItem(
    id: 'more',
    label: 'More',
    icon: LucideIcons.ellipsis,
    children: [
      TerminalPetalMenuItem(
        id: 'shortcut',
        label: 'Shortcut',
        icon: LucideIcons.slidersHorizontal,
      ),
      TerminalPetalMenuItem(
        id: 'resize',
        label: 'Resize',
        icon: LucideIcons.maximize2,
      ),
      TerminalPetalMenuItem(
        id: 'settings',
        label: 'Settings',
        icon: LucideIcons.settings,
        children: [
          TerminalPetalMenuItem(
            id: 'keyboard',
            label: 'Keyboard',
            icon: LucideIcons.keyboard,
          ),
          TerminalPetalMenuItem(
            id: 'interrupt',
            label: 'Ctrl+C',
            icon: LucideIcons.circleStop,
          ),
        ],
      ),
    ],
  ),
  TerminalPetalMenuItem(
    id: 'selection',
    label: 'Select',
    icon: LucideIcons.scanText,
  ),
  TerminalPetalMenuItem(
    id: 'paste',
    label: 'Paste',
    icon: LucideIcons.clipboardPaste,
  ),
];

final _tenActions = List<TerminalPetalMenuItem>.unmodifiable([
  for (var index = 0; index < 10; index += 1)
    TerminalPetalMenuItem(
      id: 'action-$index',
      label: 'Action $index',
      icon: LucideIcons.zap,
    ),
]);

void main() {
  test('selects a root action and cancels again in the center', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: _actions,
    );

    final selected = session.move(session.rootPositions[4]);
    expect(selected.selectedAction?.id, 'paste');

    final cancelled = selected.move(session.origin);
    expect(cancelled.selection, isNull);
    expect(cancelled.selectedAction, isNull);
  });

  test('adapts the main ring radius for ten petals without overlap', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: _tenActions,
    );

    expect(session.rootRadius, greaterThan(78));
    for (var first = 0; first < session.rootPositions.length; first += 1) {
      for (
        var second = first + 1;
        second < session.rootPositions.length;
        second += 1
      ) {
        expect(
          (session.rootPositions[first] - session.rootPositions[second])
              .distance,
          greaterThanOrEqualTo(63.9),
        );
      }
    }
  });

  testWidgets('keeps guide lines and does not underline petal labels', (
    tester,
  ) async {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: _actions,
    );

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.dark),
        home: SizedBox(
          width: 375,
          height: 600,
          child: TerminalPetalMenu(session: session),
        ),
      ),
    );

    expect(find.byKey(const ValueKey('terminal-petal-guides')), findsOneWidget);
    final label = tester.widget<Text>(find.text('History'));
    expect(label.style?.decoration, TextDecoration.none);
  });

  test('lays out ten children on an adaptive fan', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: [
        TerminalPetalMenuItem(
          id: 'parent',
          label: 'Parent',
          icon: LucideIcons.ellipsis,
          children: _tenActions,
        ),
      ],
    );
    final positions = session.positionsForParent(const [0]);

    expect(positions, hasLength(10));
    for (var first = 0; first < positions.length; first += 1) {
      for (var second = first + 1; second < positions.length; second += 1) {
        expect(
          (positions[first] - positions[second]).distance,
          greaterThanOrEqualTo(63.9),
        );
      }
    }
  });

  test('expands a branch and resolves its second-level action', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: _actions,
    );

    final expanded = session.move(session.rootPositions[2]);
    expect(expanded.expandedPath, [2]);
    expect(expanded.selectedAction?.id, 'more');

    final child = expanded.move(expanded.childPositions(2)[1]);
    expect(child.selection?.path, [2, 1]);
    expect(child.selectedAction?.id, 'resize');
  });

  test('expands and resolves a third-level action', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(375, 600),
      origin: const Offset(187.5, 300),
      actions: _actions,
    );

    final root = session.move(session.rootPositions[2]);
    final parent = root.move(root.childPositions(2)[2]);
    expect(parent.expandedPath, [2, 2]);
    expect(parent.selectedAction?.id, 'settings');

    final leaf = parent.move(parent.positionsForParent([2, 2])[1]);
    expect(leaf.selection?.path, [2, 2, 1]);
    expect(leaf.selectedAction?.id, 'interrupt');
  });

  test('keeps edge-triggered petals inside the viewport', () {
    final session = TerminalPetalMenuSession.start(
      viewport: const Size(320, 240),
      origin: const Offset(4, 4),
      actions: _actions,
    );

    for (final point in session.rootPositions) {
      expect(point.dx, inInclusiveRange(28, 292));
      expect(point.dy, inInclusiveRange(28, 212));
    }
    for (final point in session.childPositions(2)) {
      expect(point.dx, inInclusiveRange(34, 286));
      expect(point.dy, inInclusiveRange(34, 206));
    }
    final parent = session
        .move(session.rootPositions[2])
        .move(session.childPositions(2)[2]);
    for (final point in parent.positionsForParent([2, 2])) {
      expect(point.dx, inInclusiveRange(34, 286));
      expect(point.dy, inInclusiveRange(34, 206));
    }
  });

  testWidgets('crosses a split boundary but commits to the origin pane', (
    tester,
  ) async {
    final opened = <String>[];
    final selected = <String>[];
    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.dark),
        home: Scaffold(
          body: Center(
            child: SizedBox(
              width: 700,
              height: 500,
              child: TerminalPetalMenuOverlay(
                child: Row(
                  children: [
                    Expanded(
                      child: TerminalPetalMenuRegion(
                        key: const ValueKey('left-pane'),
                        actions: _actions,
                        hapticsEnabled: false,
                        onOpened: () => opened.add('left'),
                        onSelected: (action) =>
                            selected.add('left:${action.id}'),
                        child: const ColoredBox(color: Color(0xff09090b)),
                      ),
                    ),
                    Expanded(
                      child: TerminalPetalMenuRegion(
                        key: const ValueKey('right-pane'),
                        actions: _actions,
                        hapticsEnabled: false,
                        onOpened: () => opened.add('right'),
                        onSelected: (action) =>
                            selected.add('right:${action.id}'),
                        child: const ColoredBox(color: Color(0xff18181b)),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );

    final leftPane = tester.getRect(find.byKey(const ValueKey('left-pane')));
    final rightPane = tester.getRect(find.byKey(const ValueKey('right-pane')));
    final gesture = await tester.startGesture(
      Offset(leftPane.right - 18, leftPane.center.dy),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);

    final search = find.byKey(const ValueKey('terminal-petal-action-search'));
    expect(search, findsOneWidget);
    expect(tester.getCenter(search).dx, greaterThan(rightPane.left));

    await gesture.moveTo(tester.getCenter(search));
    await tester.pump();
    await gesture.up();
    await tester.pump();

    expect(opened, ['left']);
    expect(selected, ['left:search']);
  });

  testWidgets('long press, petal change, and commit all play haptics', (
    tester,
  ) async {
    final platformCalls = <MethodCall>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        platformCalls.add(call);
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    final selected = <String>[];
    await tester.pumpWidget(
      _Harness(
        child: TerminalPetalMenuRegion(
          actions: _actions,
          onSelected: (action) => selected.add(action.id),
          child: const ColoredBox(color: Color(0xff09090b)),
        ),
      ),
    );
    platformCalls.clear();

    final region = find.byType(TerminalPetalMenuRegion);
    final gesture = await tester.startGesture(tester.getCenter(region));
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);

    expect(find.byKey(terminalPetalMenuKey), findsOneWidget);
    await gesture.moveTo(
      tester.getCenter(
        find.byKey(const ValueKey('terminal-petal-action-paste')),
      ),
    );
    await tester.pump();
    await gesture.up();
    await tester.pump();

    expect(selected, ['paste']);
    expect(find.byKey(terminalPetalMenuKey), findsNothing);
    final haptics = platformCalls
        .where((call) => call.method == 'HapticFeedback.vibrate')
        .map((call) => call.arguments)
        .toList(growable: false);
    expect(
      haptics,
      containsAllInOrder([
        'HapticFeedbackType.mediumImpact',
        'HapticFeedbackType.lightImpact',
        'HapticFeedbackType.mediumImpact',
      ]),
    );
  });

  testWidgets('vibrates again when the pointer re-enters the same petal', (
    tester,
  ) async {
    final platformCalls = <MethodCall>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        platformCalls.add(call);
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    await tester.pumpWidget(
      _Harness(
        child: TerminalPetalMenuRegion(
          actions: _actions,
          onSelected: (_) {},
          child: const ColoredBox(color: Color(0xff09090b)),
        ),
      ),
    );
    platformCalls.clear();

    final region = find.byType(TerminalPetalMenuRegion);
    final center = tester.getCenter(region);
    final gesture = await tester.startGesture(center);
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);
    final paste = tester.getCenter(
      find.byKey(const ValueKey('terminal-petal-action-paste')),
    );
    await gesture.moveTo(paste);
    await tester.pump();
    await gesture.moveTo(center);
    await tester.pump();
    await gesture.moveTo(paste);
    await tester.pump();
    await gesture.cancel();
    await tester.pump();

    final lightImpacts = platformCalls.where(
      (call) =>
          call.method == 'HapticFeedback.vibrate' &&
          call.arguments == 'HapticFeedbackType.lightImpact',
    );
    expect(lightImpacts, hasLength(2));
  });

  testWidgets('can disable all petal-menu haptics without disabling actions', (
    tester,
  ) async {
    final platformCalls = <MethodCall>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        platformCalls.add(call);
        return null;
      },
    );
    addTearDown(
      () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
        SystemChannels.platform,
        null,
      ),
    );
    final selected = <String>[];
    await tester.pumpWidget(
      _Harness(
        child: TerminalPetalMenuRegion(
          actions: _actions,
          hapticsEnabled: false,
          onSelected: (action) => selected.add(action.id),
          child: const ColoredBox(color: Color(0xff09090b)),
        ),
      ),
    );
    platformCalls.clear();

    final region = find.byType(TerminalPetalMenuRegion);
    final gesture = await tester.startGesture(tester.getCenter(region));
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);
    await gesture.moveTo(
      tester.getCenter(
        find.byKey(const ValueKey('terminal-petal-action-paste')),
      ),
    );
    await tester.pump();
    await gesture.up();
    await tester.pump();

    expect(selected, ['paste']);
    expect(
      platformCalls.where((call) => call.method == 'HapticFeedback.vibrate'),
      isEmpty,
    );
  });

  testWidgets('keeps a secondary menu in the same held gesture', (
    tester,
  ) async {
    final selected = <String>[];
    await tester.pumpWidget(
      _Harness(
        child: TerminalPetalMenuRegion(
          actions: _actions,
          onSelected: (action) => selected.add(action.id),
          child: const ColoredBox(color: Color(0xff09090b)),
        ),
      ),
    );

    final region = find.byType(TerminalPetalMenuRegion);
    final gesture = await tester.startGesture(tester.getCenter(region));
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);
    await gesture.moveTo(
      tester.getCenter(
        find.byKey(const ValueKey('terminal-petal-action-more')),
      ),
    );
    await tester.pump();
    await tester.pump(AnyttyMotion.quick);

    final settings = find.byKey(
      const ValueKey('terminal-petal-action-settings'),
    );
    expect(settings, findsOneWidget);
    await gesture.moveTo(tester.getCenter(settings));
    await tester.pump();
    await gesture.up();
    await tester.pump();

    expect(selected, ['settings']);
  });

  testWidgets('keeps a third-level menu in the same held gesture', (
    tester,
  ) async {
    final selected = <String>[];
    await tester.pumpWidget(
      _Harness(
        child: TerminalPetalMenuRegion(
          actions: _actions,
          onSelected: (action) => selected.add(action.id),
          child: const ColoredBox(color: Color(0xff09090b)),
        ),
      ),
    );

    final gesture = await tester.startGesture(
      tester.getCenter(find.byType(TerminalPetalMenuRegion)),
    );
    await tester.pump(kLongPressTimeout + const Duration(milliseconds: 20));
    await tester.pump(AnyttyMotion.standard);
    await gesture.moveTo(
      tester.getCenter(
        find.byKey(const ValueKey('terminal-petal-action-more')),
      ),
    );
    await tester.pump();
    await tester.pump(AnyttyMotion.quick);
    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const ValueKey('terminal-petal-opacity-more')),
          )
          .opacity,
      0.62,
    );
    await gesture.moveTo(
      tester.getCenter(
        find.byKey(const ValueKey('terminal-petal-action-settings')),
      ),
    );
    await tester.pump();
    await tester.pump(AnyttyMotion.quick);
    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const ValueKey('terminal-petal-opacity-more')),
          )
          .opacity,
      0.42,
    );
    expect(
      tester
          .widget<AnimatedOpacity>(
            find.byKey(const ValueKey('terminal-petal-opacity-settings')),
          )
          .opacity,
      0.62,
    );
    final interrupt = find.byKey(
      const ValueKey('terminal-petal-action-interrupt'),
    );
    expect(interrupt, findsOneWidget);
    await gesture.moveTo(tester.getCenter(interrupt));
    await tester.pump();
    await gesture.up();
    await tester.pump();

    expect(selected, ['interrupt']);
  });
}

final class _Harness extends StatelessWidget {
  const _Harness({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) => MaterialApp(
    theme: anyttyTheme(Brightness.dark),
    home: Scaffold(
      body: Center(child: SizedBox(width: 375, height: 600, child: child)),
    ),
  );
}
