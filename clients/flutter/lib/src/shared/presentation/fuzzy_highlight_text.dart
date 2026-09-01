import 'package:flutter/material.dart';

import '../../app/anytty_theme.dart';
import '../domain/fuzzy_search.dart';

final class FuzzyHighlightText extends StatelessWidget {
  const FuzzyHighlightText(
    this.text, {
    super.key,
    required this.query,
    this.style,
    this.maxLines = 1,
    this.overflow = TextOverflow.ellipsis,
    this.textAlign,
  });

  final String text;
  final String query;
  final TextStyle? style;
  final int? maxLines;
  final TextOverflow overflow;
  final TextAlign? textAlign;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final baseStyle = DefaultTextStyle.of(context).style.merge(style);
    final match = fuzzySubsequenceMatch(text, query);
    if (match == null || match.indices.isEmpty) {
      return Text(
        text,
        maxLines: maxLines,
        overflow: overflow,
        textAlign: textAlign,
        style: baseStyle,
      );
    }

    final characters = text.characters.toList(growable: false);
    final matched = match.indices.toSet();
    final spans = <InlineSpan>[];
    var start = 0;
    var highlighted = matched.contains(0);
    for (var index = 1; index <= characters.length; index += 1) {
      final nextHighlighted =
          index < characters.length && matched.contains(index);
      if (index < characters.length && nextHighlighted == highlighted) continue;
      final value = characters.sublist(start, index).join();
      spans.add(
        TextSpan(
          text: value,
          style: highlighted
              ? baseStyle.copyWith(
                  color: palette.accent,
                  fontWeight: FontWeight.w800,
                  backgroundColor: palette.accent.withValues(alpha: 0.12),
                )
              : baseStyle,
        ),
      );
      start = index;
      highlighted = nextHighlighted;
    }

    return Semantics(
      label: text,
      excludeSemantics: true,
      child: Text.rich(
        TextSpan(children: spans),
        maxLines: maxLines,
        overflow: overflow,
        textAlign: textAlign,
      ),
    );
  }
}
