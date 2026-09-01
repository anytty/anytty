import 'package:characters/characters.dart';

List<String> terminalGraphemeClusters(String text) {
  if (text.isEmpty) return const [];
  return text.characters.toList(growable: false);
}

int terminalGraphemeCellWidth(String grapheme) {
  if (grapheme.isEmpty) return 0;
  final runes = grapheme.runes.toList(growable: false);
  if (runes.every(_isZeroWidthCodepoint)) return 0;
  return runes.any(_isWideCodepoint) ? 2 : 1;
}

bool _isZeroWidthCodepoint(int rune) =>
    (rune >= 0x0300 && rune <= 0x036f) ||
    (rune >= 0x1ab0 && rune <= 0x1aff) ||
    (rune >= 0x1dc0 && rune <= 0x1dff) ||
    (rune >= 0x20d0 && rune <= 0x20ff) ||
    (rune >= 0xfe00 && rune <= 0xfe0f) ||
    (rune >= 0xfe20 && rune <= 0xfe2f) ||
    (rune >= 0xe0100 && rune <= 0xe01ef) ||
    rune == 0x200b ||
    rune == 0x200c ||
    rune == 0x200d;

bool _isWideCodepoint(int rune) =>
    (rune >= 0x1100 && rune <= 0x115f) ||
    rune == 0x2329 ||
    rune == 0x232a ||
    (rune >= 0x2e80 && rune <= 0xa4cf && rune != 0x303f) ||
    (rune >= 0xac00 && rune <= 0xd7a3) ||
    (rune >= 0xf900 && rune <= 0xfaff) ||
    (rune >= 0xfe10 && rune <= 0xfe19) ||
    (rune >= 0xfe30 && rune <= 0xfe6f) ||
    (rune >= 0xff00 && rune <= 0xff60) ||
    (rune >= 0xffe0 && rune <= 0xffe6) ||
    (rune >= 0x1f000 && rune <= 0x1faff) ||
    (rune >= 0x20000 && rune <= 0x3fffd);
