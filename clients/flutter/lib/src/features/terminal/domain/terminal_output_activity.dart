import '../../../generated/proto/apipb/terminal.pb.dart';

enum TerminalOutputActivityTone { fresh, recent, idle, stale, none }

DateTime? terminalLastOutputAt(TerminalInfo terminal) {
  final unixNano = terminal.lastOutputAtUnixNano.toInt();
  if (unixNano <= 0) return null;
  return DateTime.fromMicrosecondsSinceEpoch(unixNano ~/ 1000, isUtc: true);
}

Duration? terminalOutputQuietDuration(TerminalInfo terminal, {DateTime? now}) {
  final lastOutputAt = terminalLastOutputAt(terminal);
  if (lastOutputAt == null) return null;
  final quiet = (now ?? DateTime.now()).toUtc().difference(lastOutputAt);
  return quiet.isNegative ? Duration.zero : quiet;
}

TerminalOutputActivityTone terminalOutputActivityTone(
  TerminalInfo terminal, {
  DateTime? now,
}) {
  final quiet = terminalOutputQuietDuration(terminal, now: now);
  if (quiet == null) return TerminalOutputActivityTone.none;
  if (quiet < const Duration(seconds: 5)) {
    return TerminalOutputActivityTone.fresh;
  }
  if (quiet < const Duration(minutes: 1)) {
    return TerminalOutputActivityTone.recent;
  }
  if (quiet < const Duration(hours: 1)) {
    return TerminalOutputActivityTone.idle;
  }
  return TerminalOutputActivityTone.stale;
}
