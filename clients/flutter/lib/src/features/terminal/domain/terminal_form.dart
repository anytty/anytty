String formatTerminalCommand(List<String> arguments) {
  final plain = RegExp(r'^[A-Za-z0-9_./:@%+=,-]+$');
  return arguments
      .map((argument) {
        if (plain.hasMatch(argument)) return argument;
        return "'${argument.replaceAll("'", "'\\''")}'";
      })
      .join(' ');
}

List<String> parseTerminalCommand(String input) {
  final arguments = <String>[];
  final value = StringBuffer();
  String? quote;
  var escaped = false;
  var active = false;

  for (final rune in input.trim().runes) {
    final character = String.fromCharCode(rune);
    if (escaped) {
      value.write(character);
      escaped = false;
      active = true;
      continue;
    }
    if (character == r'\' && quote != "'") {
      escaped = true;
      active = true;
      continue;
    }
    if (quote != null) {
      if (character == quote) {
        quote = null;
      } else {
        value.write(character);
      }
      active = true;
      continue;
    }
    if (character == "'" || character == '"') {
      quote = character;
      active = true;
      continue;
    }
    if (character.trim().isEmpty) {
      if (active) {
        arguments.add(value.toString());
        value.clear();
        active = false;
      }
      continue;
    }
    value.write(character);
    active = true;
  }
  if (escaped || quote != null) {
    throw const FormatException('Command has an unfinished quote or escape');
  }
  if (active) arguments.add(value.toString());
  return arguments;
}

List<String> parseTerminalEnvironment(String input) {
  final result = <String>[];
  final keys = <String>{};
  final namePattern = RegExp(r'^[A-Za-z_][A-Za-z0-9_]*$');
  for (final rawLine in input.split(RegExp(r'\r?\n'))) {
    var line = rawLine.trim();
    if (line.isEmpty || line.startsWith('#')) continue;
    if (line.startsWith('export ')) line = line.substring(7).trim();
    final separator = line.indexOf('=');
    final key = (separator < 0 ? line : line.substring(0, separator)).trim();
    final value = separator < 0 ? '' : line.substring(separator + 1);
    if (!namePattern.hasMatch(key)) {
      throw FormatException(
        'Invalid environment variable name: ${key.isEmpty ? '(empty)' : key}',
      );
    }
    if (!keys.add(key)) {
      throw FormatException('Duplicate environment variable: $key');
    }
    result.add('$key=$value');
  }
  return result;
}
