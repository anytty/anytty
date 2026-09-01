import 'package:anytty_native/src/features/terminal/domain/terminal_form.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('round trips command arguments with spaces and quotes', () {
    final command = ['/opt/My Shell/bin/zsh', '-lc', "printf '%s' ready"];
    expect(parseTerminalCommand(formatTerminalCommand(command)), command);
  });

  test('rejects unfinished command quoting', () {
    expect(
      () => parseTerminalCommand("echo 'unfinished"),
      throwsA(isA<FormatException>()),
    );
  });

  test('parses environment blocks and preserves equals signs in values', () {
    expect(parseTerminalEnvironment('A=1\nexport TOKEN=a=b\n# ignored\n'), [
      'A=1',
      'TOKEN=a=b',
    ]);
  });

  test('rejects invalid and duplicate environment names', () {
    expect(
      () => parseTerminalEnvironment('BAD-NAME=x'),
      throwsA(isA<FormatException>()),
    );
    expect(
      () => parseTerminalEnvironment('A=1\nA=2'),
      throwsA(isA<FormatException>()),
    );
  });
}
