import 'package:anytty_native/src/features/terminal/domain/live_screen_store.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_swipe_switcher.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

final class _FakeBridge implements TerminalSwipeBridge {
  _FakeBridge({
    List<String>? terminals,
    this.currentIndex = 0,
    List<String>? devices,
  }) : terminals = terminals ?? const ['a', 'b', 'c'],
       devices = devices ?? const ['host-a', 'host-b'];

  final List<String> terminals;
  final List<String> devices;
  final int currentIndex;

  final List<String> committedTerminals = [];
  final List<String> committedDevices = [];
  final List<String> preparedTerminals = [];
  final List<String> heldPreviews = [];
  final List<String> releasedPreviews = [];
  int switcherOpens = 0;
  final List<TerminalSwipeMode> snapshotModes = [];
  CanonicalLiveScreen? preparedScreen;
  final List<String> events = [];

  @override
  Future<TerminalSwipeSnapshot> snapshot(TerminalSwipeMode mode) async {
    snapshotModes.add(mode);
    if (mode == TerminalSwipeMode.terminals) {
      return TerminalSwipeSnapshot(
        targets: [
          for (final terminal in terminals)
            TerminalSwipeTarget(id: terminal, label: terminal),
        ],
        currentIndex: currentIndex,
      );
    }
    return TerminalSwipeSnapshot(
      targets: [
        for (final device in devices)
          TerminalSwipeTarget(id: device, label: device, subtitle: device),
      ],
      currentIndex: 0,
    );
  }

  @override
  void commitTerminal(String terminalId) {
    committedTerminals.add(terminalId);
    events.add('commit:$terminalId');
  }

  @override
  void commitDevice(String endpointId, String label) =>
      committedDevices.add(endpointId);

  @override
  Future<CanonicalLiveScreen?> prepareTerminal(
    String terminalId, {
    required int cols,
    required int rows,
  }) async {
    preparedTerminals.add(terminalId);
    events.add('prepare:$terminalId');
    return preparedScreen;
  }

  @override
  void holdPreview(CanonicalLiveScreen screen, String terminalId) {
    heldPreviews.add(terminalId);
    events.add('hold:$terminalId');
  }

  @override
  void releasePreview(String terminalId) => releasedPreviews.add(terminalId);

  @override
  void openSwitcher() => switcherOpens++;
}

Future<void> _pump(
  WidgetTester tester,
  _FakeBridge bridge, {
  bool enabled = true,
  TerminalSettings settings = defaultTerminalSettings,
  double width = 400,
  int targetCols = 0,
  int targetRows = 0,
}) async {
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: Center(
          child: SizedBox(
            width: width,
            height: 600,
            child: TerminalSwipeSwitcher(
              bridge: bridge,
              enabled: enabled,
              settings: settings,
              hapticsEnabled: false,
              currentScreenTerminalId: 'a',
              targetCols: targetCols,
              targetRows: targetRows,
              child: const ColoredBox(color: Color(0xFF101010)),
            ),
          ),
        ),
      ),
    ),
  );
  await tester.pump();
}

Future<void> _oneFingerSwipe(
  WidgetTester tester, {
  required double dx,
  double dy = 0,
}) async {
  final gesture = await tester.startGesture(const Offset(300, 300));
  for (var step = 0; step < 6; step++) {
    await gesture.moveBy(Offset(dx / 6, dy / 6));
    await tester.pump(const Duration(milliseconds: 16));
  }
  await gesture.up();
  await tester.pumpAndSettle();
}

Future<void> _twoFingerSwipe(WidgetTester tester, {required double dx}) async {
  final first = await tester.startGesture(const Offset(260, 280));
  final second = await tester.startGesture(const Offset(340, 280));
  for (var step = 0; step < 6; step++) {
    await first.moveBy(Offset(dx / 6, 0));
    await second.moveBy(Offset(dx / 6, 0));
    await tester.pump(const Duration(milliseconds: 16));
  }
  await first.up();
  await second.up();
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('单指左滑切换到下一个终端', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge);

    await _oneFingerSwipe(tester, dx: -120);

    expect(bridge.snapshotModes, [TerminalSwipeMode.terminals]);
    expect(bridge.committedTerminals, ['b']);
    expect(bridge.committedDevices, isEmpty);
  });

  testWidgets('单指右滑切换到上一个终端', (tester) async {
    final bridge = _FakeBridge(currentIndex: 2);
    await _pump(tester, bridge);

    await _oneFingerSwipe(tester, dx: 120);

    expect(bridge.committedTerminals, ['b']);
  });

  testWidgets('切换前先预热目标终端，再用预热帧垫住切换窗口', (tester) async {
    final bridge = _FakeBridge();
    bridge.preparedScreen = CanonicalLiveScreen(
      terminal: TerminalRef(
        endpointId: 'endpoint',
        terminalId: 'b',
      ),
      connectionGeneration: Int64(1),
      revision: Int64(2),
      cols: 80,
      rows: 24,
      screenRows: const [],
      alternateScreen: false,
      cursor: null,
      modes: null,
      timestampUnixNano: Int64(2),
    );
    await _pump(tester, bridge, targetCols: 80, targetRows: 24);

    await _oneFingerSwipe(tester, dx: -120);

    expect(bridge.events, ['prepare:b', 'hold:b', 'commit:b']);
    expect(bridge.committedTerminals, ['b']);
  });

  testWidgets('预热失败时仍然切换（不允许手势卡死）', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge, targetCols: 80, targetRows: 24);

    await _oneFingerSwipe(tester, dx: -120);

    expect(bridge.preparedTerminals, ['b']);
    expect(bridge.heldPreviews, isEmpty);
    expect(bridge.committedTerminals, ['b']);
  });

  testWidgets('双指左滑切换到下一个设备', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge);

    await _twoFingerSwipe(tester, dx: -120);

    expect(bridge.snapshotModes, [TerminalSwipeMode.devices]);
    expect(bridge.committedDevices, ['host-b']);
    expect(bridge.committedTerminals, isEmpty);
  });

  testWidgets('到头继续甩出打开切换面板', (tester) async {
    final bridge = _FakeBridge(currentIndex: 2);
    await _pump(tester, bridge);

    await _oneFingerSwipe(tester, dx: -160);

    expect(bridge.switcherOpens, 1);
    expect(bridge.committedTerminals, isEmpty);
  });

  testWidgets('短距离拖动只弹回，不触发任何动作', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge);

    await _oneFingerSwipe(tester, dx: -12);

    expect(bridge.committedTerminals, isEmpty);
    expect(bridge.committedDevices, isEmpty);
    expect(bridge.switcherOpens, 0);
  });

  testWidgets('竖向拖动不接管（交给历史滚动）', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge);

    await _oneFingerSwipe(tester, dx: -140, dy: 140);

    expect(bridge.committedTerminals, isEmpty);
    expect(bridge.snapshotModes, isEmpty);
  });

  testWidgets('enabled=false 时完全不响应', (tester) async {
    final bridge = _FakeBridge();
    await _pump(tester, bridge, enabled: false);

    await _oneFingerSwipe(tester, dx: -120);

    expect(bridge.snapshotModes, isEmpty);
    expect(bridge.committedTerminals, isEmpty);
  });

  testWidgets('关闭终端手势开关后单指不再切换，双指仍可切设备', (tester) async {
    final bridge = _FakeBridge();
    await _pump(
      tester,
      bridge,
      settings: defaultTerminalSettings.copyWith(gestureTerminalSwitch: false),
    );

    await _oneFingerSwipe(tester, dx: -120);
    expect(bridge.snapshotModes, isEmpty);

    await _twoFingerSwipe(tester, dx: -120);
    expect(bridge.committedDevices, ['host-b']);
  });
}
