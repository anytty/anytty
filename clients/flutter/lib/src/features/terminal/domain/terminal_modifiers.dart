enum TerminalModifier { shift, control, alt }

enum TerminalModifierLatch { off, once, locked }

const terminalModifierShiftBit = 1 << 0;
const terminalModifierControlBit = 1 << 1;
const terminalModifierAltBit = 1 << 2;
const terminalModifierSuperBit = 1 << 3;

final class TerminalModifierState {
  const TerminalModifierState({
    this.shift = TerminalModifierLatch.off,
    this.control = TerminalModifierLatch.off,
    this.alt = TerminalModifierLatch.off,
  });

  final TerminalModifierLatch shift;
  final TerminalModifierLatch control;
  final TerminalModifierLatch alt;

  int get bits =>
      (shift == TerminalModifierLatch.off ? 0 : terminalModifierShiftBit) |
      (control == TerminalModifierLatch.off ? 0 : terminalModifierControlBit) |
      (alt == TerminalModifierLatch.off ? 0 : terminalModifierAltBit);

  bool get hasOnce =>
      shift == TerminalModifierLatch.once ||
      control == TerminalModifierLatch.once ||
      alt == TerminalModifierLatch.once;

  TerminalModifierLatch stateOf(TerminalModifier modifier) {
    return switch (modifier) {
      TerminalModifier.shift => shift,
      TerminalModifier.control => control,
      TerminalModifier.alt => alt,
    };
  }

  TerminalModifierState cycle(TerminalModifier modifier) {
    final next = _nextLatch(stateOf(modifier));
    return switch (modifier) {
      TerminalModifier.shift => copyWith(shift: next),
      TerminalModifier.control => copyWith(control: next),
      TerminalModifier.alt => copyWith(alt: next),
    };
  }

  TerminalModifierState consumeOnce() {
    return TerminalModifierState(
      shift: _consume(shift),
      control: _consume(control),
      alt: _consume(alt),
    );
  }

  TerminalModifierState restoreRejected(TerminalModifierState sent) {
    return TerminalModifierState(
      shift: _restoreLatch(current: shift, sent: sent.shift),
      control: _restoreLatch(current: control, sent: sent.control),
      alt: _restoreLatch(current: alt, sent: sent.alt),
    );
  }

  TerminalModifierState copyWith({
    TerminalModifierLatch? shift,
    TerminalModifierLatch? control,
    TerminalModifierLatch? alt,
  }) {
    return TerminalModifierState(
      shift: shift ?? this.shift,
      control: control ?? this.control,
      alt: alt ?? this.alt,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is TerminalModifierState &&
        other.shift == shift &&
        other.control == control &&
        other.alt == alt;
  }

  @override
  int get hashCode => Object.hash(shift, control, alt);
}

TerminalModifierLatch _nextLatch(TerminalModifierLatch current) {
  return switch (current) {
    TerminalModifierLatch.off => TerminalModifierLatch.once,
    TerminalModifierLatch.once => TerminalModifierLatch.locked,
    TerminalModifierLatch.locked => TerminalModifierLatch.off,
  };
}

TerminalModifierLatch _consume(TerminalModifierLatch state) {
  return state == TerminalModifierLatch.once
      ? TerminalModifierLatch.off
      : state;
}

TerminalModifierLatch _restoreLatch({
  required TerminalModifierLatch current,
  required TerminalModifierLatch sent,
}) {
  return sent == TerminalModifierLatch.once &&
          current == TerminalModifierLatch.off
      ? TerminalModifierLatch.once
      : current;
}
