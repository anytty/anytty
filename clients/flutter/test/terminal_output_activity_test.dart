import 'package:anytty_native/src/features/terminal/domain/terminal_output_activity.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final now = DateTime.utc(2026, 9, 2, 12);

  TerminalInfo terminal(Duration? quiet) {
    final lastOutputAt = quiet == null
        ? null
        : Int64((now.microsecondsSinceEpoch - quiet.inMicroseconds) * 1000);
    return TerminalInfo(lastOutputAtUnixNano: lastOutputAt);
  }

  test('converts the last output timestamp to quiet duration', () {
    expect(
      terminalOutputQuietDuration(
        terminal(const Duration(seconds: 12)),
        now: now,
      ),
      const Duration(seconds: 12),
    );
    expect(terminalOutputQuietDuration(terminal(null), now: now), isNull);
  });

  test('assigns activity tones without treating quiet as an error', () {
    expect(
      terminalOutputActivityTone(
        terminal(const Duration(seconds: 4)),
        now: now,
      ),
      TerminalOutputActivityTone.fresh,
    );
    expect(
      terminalOutputActivityTone(
        terminal(const Duration(seconds: 30)),
        now: now,
      ),
      TerminalOutputActivityTone.recent,
    );
    expect(
      terminalOutputActivityTone(
        terminal(const Duration(minutes: 3)),
        now: now,
      ),
      TerminalOutputActivityTone.idle,
    );
    expect(
      terminalOutputActivityTone(terminal(const Duration(hours: 2)), now: now),
      TerminalOutputActivityTone.stale,
    );
    expect(
      terminalOutputActivityTone(terminal(null), now: now),
      TerminalOutputActivityTone.none,
    );
  });
}
