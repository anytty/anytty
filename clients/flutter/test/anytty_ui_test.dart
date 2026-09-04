import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('shared mobile UI tokens match the product direction', () {
    final theme = anyttyTheme(Brightness.dark);
    final title = theme.textTheme.titleLarge!;
    final section = theme.textTheme.titleMedium!;
    final body = theme.textTheme.bodyMedium!;
    final iconButton = theme.iconButtonTheme.style!;
    final cardShape = theme.cardTheme.shape! as RoundedRectangleBorder;

    expect(theme.textTheme.bodyMedium!.fontFamily, anyttyUiFontFamily);
    expect(title.fontSize, 22);
    expect(title.height! * title.fontSize!, closeTo(28, 0.01));
    expect(title.fontWeight, FontWeight.w600);
    expect(title.letterSpacing, -0.7);
    expect(section.fontSize, 19);
    expect(section.height! * section.fontSize!, closeTo(24, 0.01));
    expect(section.fontWeight, FontWeight.w600);
    expect(section.letterSpacing, 0.6);
    expect(body.fontSize, 14.5);
    expect(body.height! * body.fontSize!, closeTo(18, 0.01));
    expect(body.letterSpacing, 0.3);
    expect(iconButton.minimumSize!.resolve({}), Size.square(44));
    expect(iconButton.shape!.resolve({}), const CircleBorder());
    expect(cardShape.side, BorderSide.none);
    expect(theme.dividerTheme.color, AnyttyPalette.dark.track);
  });
}
