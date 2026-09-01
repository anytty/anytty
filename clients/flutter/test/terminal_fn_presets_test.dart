import 'package:anytty_native/src/features/terminal/domain/terminal_fn_presets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('matches program presets without case sensitivity', () {
    expect(matchTerminalFnPreset('/usr/local/bin/CLAUDE')!.id, 'claude');
    expect(matchTerminalFnPreset('opencode --continue')!.name, 'OpenCode');
    expect(matchTerminalFnPreset('/bin/zsh'), isNull);
    expect(matchTerminalFnPreset(null), isNull);
  });

  test('keeps system control bytes aligned with the web key panel', () {
    final items = systemTerminalFnGroups.expand((group) => group.items);

    expect(items.singleWhere((item) => item.label == 'Ctrl+D').data, '\x04');
    expect(items.singleWhere((item) => item.label == r'Ctrl+\').data, '\x1c');
    expect(items.singleWhere((item) => item.label == 'Tab').data, '\t');
    expect(items.singleWhere((item) => item.label == 'Delete').data, '\x1b[3~');
  });

  test('program commands include submit while system shortcuts do not', () {
    final claude = matchTerminalFnPreset('claude')!;
    final clear = claude.groups
        .expand((group) => group.items)
        .singleWhere((item) => item.label == '/clear');

    expect(clear.data, '/clear\n');
    expect(systemTerminalFnGroups.first.items.first.data.endsWith('\n'), false);
  });
}
