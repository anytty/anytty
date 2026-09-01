import 'package:anytty_native/src/features/terminal/domain/terminal_soft_input.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('keeps an untouched soft keyboard buffer idle', () {
    expect(decodeTerminalSoftInput(terminalSoftInputSentinel).isIdle, isTrue);
  });

  test('turns one-character sentinel deletion into Backspace', () {
    final edit = decodeTerminalSoftInput('.');

    expect(edit.backspace, isTrue);
    expect(edit.text, isEmpty);
  });

  test('turns whole sentinel deletion into Backspace', () {
    expect(decodeTerminalSoftInput('').backspace, isTrue);
  });

  test('returns committed text without the sentinel', () {
    final edit = decodeTerminalSoftInput('${terminalSoftInputSentinel}hello\n');

    expect(edit.backspace, isFalse);
    expect(edit.text, 'hello\n');
  });

  test('preserves committed text when an IME replaces the whole buffer', () {
    final edit = decodeTerminalSoftInput('hello');

    expect(edit.backspace, isFalse);
    expect(edit.text, 'hello');
  });
}
