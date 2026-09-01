import 'package:anytty_native/src/features/terminal/presentation/terminal_recovery_notice.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('keeps short terminal recovery quiet', (tester) async {
    final changes = <bool>[];
    final gate = TerminalRecoveryNoticeGate(onVisibilityChanged: changes.add);
    addTearDown(gate.dispose);

    gate.setRecovering(true);
    await tester.pump(const Duration(milliseconds: 1199));

    expect(gate.visible, isFalse);
    expect(changes, isEmpty);

    await tester.pump(const Duration(milliseconds: 1));

    expect(gate.visible, isTrue);
    expect(changes, [true]);
  });

  testWidgets('hides and cancels a terminal recovery notice', (tester) async {
    final changes = <bool>[];
    final gate = TerminalRecoveryNoticeGate(onVisibilityChanged: changes.add);
    addTearDown(gate.dispose);

    gate.setRecovering(true);
    await tester.pump(const Duration(milliseconds: 1200));
    gate.setRecovering(false);

    expect(gate.visible, isFalse);
    expect(changes, [true, false]);

    gate.setRecovering(true);
    await tester.pump(const Duration(milliseconds: 600));
    gate.setRecovering(false);
    await tester.pump(const Duration(seconds: 1));

    expect(gate.visible, isFalse);
    expect(changes, [true, false]);
  });
}
