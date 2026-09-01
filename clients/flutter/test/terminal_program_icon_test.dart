import 'package:anytty_native/src/features/terminal/presentation/terminal_program_icon.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TerminalInfo terminal(String process, {List<String> command = const []}) =>
      TerminalInfo(foregroundProcess: process, command: command);

  test('classifies common foreground terminal programs', () {
    expect(
      terminalProgramKind(terminal('/bin/zsh')),
      TerminalProgramKind.shell,
    );
    expect(terminalProgramKind(terminal('nvim')), TerminalProgramKind.editor);
    expect(terminalProgramKind(terminal('codex')), TerminalProgramKind.agent);
    expect(
      terminalProgramKind(terminal('python3 app.py')),
      TerminalProgramKind.runtime,
    );
    expect(
      terminalProgramKind(terminal('docker')),
      TerminalProgramKind.container,
    );
    expect(terminalProgramKind(terminal('psql')), TerminalProgramKind.database);
    expect(terminalProgramKind(terminal('ssh')), TerminalProgramKind.remote);
    expect(terminalProgramKind(terminal('lazygit')), TerminalProgramKind.git);
    expect(terminalProgramKind(terminal('btop')), TerminalProgramKind.monitor);
  });

  test('falls back to the command executable and a neutral terminal icon', () {
    final value = terminal('', command: const ['/usr/local/bin/custom-tool']);

    expect(terminalProgramName(value), 'custom-tool');
    expect(terminalProgramKind(value), TerminalProgramKind.terminal);
  });
}
