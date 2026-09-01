import 'package:anytty_native/src/features/terminal/presentation/terminal_workspace_screen.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('keeps the terminal history search field transparent', () {
    expect(terminalHistorySearchInputDecoration.filled, isFalse);
    expect(terminalHistorySearchInputDecoration.fillColor, isNull);
    expect(terminalHistorySearchInputDecoration.border, InputBorder.none);
  });
}
