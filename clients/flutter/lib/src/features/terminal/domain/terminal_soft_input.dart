// Keep real, editable text before the IME cursor. Some Android keyboards treat
// zero-width characters as no surrounding text and suppress Backspace.
const terminalSoftInputSentinel = '. ';

final class TerminalSoftInputEdit {
  const TerminalSoftInputEdit({this.text = '', this.backspace = false});

  final String text;
  final bool backspace;

  bool get isIdle => text.isEmpty && !backspace;
}

TerminalSoftInputEdit decodeTerminalSoftInput(String value) {
  if (value == terminalSoftInputSentinel) {
    return const TerminalSoftInputEdit();
  }
  if (value.isEmpty || value == '.') {
    return const TerminalSoftInputEdit(backspace: true);
  }
  return TerminalSoftInputEdit(
    text: value.startsWith(terminalSoftInputSentinel)
        ? value.substring(terminalSoftInputSentinel.length)
        : value,
  );
}
