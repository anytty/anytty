import 'package:anytty_native/src/features/terminal/domain/terminal_modifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('cycles each modifier through off once locked and off', () {
    const initial = TerminalModifierState();
    final once = initial.cycle(TerminalModifier.control);
    final locked = once.cycle(TerminalModifier.control);
    final off = locked.cycle(TerminalModifier.control);

    expect(once.control, TerminalModifierLatch.once);
    expect(locked.control, TerminalModifierLatch.locked);
    expect(off.control, TerminalModifierLatch.off);
  });

  test('consumes one-shot modifiers and keeps locked modifiers', () {
    const state = TerminalModifierState(
      shift: TerminalModifierLatch.once,
      control: TerminalModifierLatch.locked,
      alt: TerminalModifierLatch.once,
    );

    expect(
      state.bits,
      terminalModifierShiftBit |
          terminalModifierControlBit |
          terminalModifierAltBit,
    );
    expect(
      state.consumeOnce(),
      const TerminalModifierState(control: TerminalModifierLatch.locked),
    );
  });

  test('restores a rejected once state without overwriting later input', () {
    const sent = TerminalModifierState(control: TerminalModifierLatch.once);

    expect(sent.consumeOnce().restoreRejected(sent), sent);
    expect(
      const TerminalModifierState(control: TerminalModifierLatch.locked)
          .restoreRejected(sent),
      const TerminalModifierState(control: TerminalModifierLatch.locked),
    );
  });
}
