import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/shared/domain/fuzzy_search.dart';
import 'package:anytty_native/src/shared/presentation/fuzzy_highlight_text.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('matches case-insensitive ordered subsequences', () {
    expect(fuzzySubsequenceMatch('ABCD', 'acd')?.indices, [0, 2, 3]);
    expect(fuzzySubsequenceMatch('开发设备一号', '开设号')?.indices, [0, 2, 5]);
    expect(fuzzySubsequenceMatch('ABCD', 'adc'), isNull);
  });

  testWidgets('highlights only the matched character runs', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: const Scaffold(
          body: FuzzyHighlightText(
            'ABCD',
            query: 'acd',
            style: TextStyle(fontSize: 14),
          ),
        ),
      ),
    );

    final text = tester.widget<Text>(find.byType(Text));
    final spans = (text.textSpan! as TextSpan).children!.cast<TextSpan>();
    expect(spans.map((span) => span.text), ['A', 'B', 'CD']);
    expect(spans[0].style?.backgroundColor, isNotNull);
    expect(spans[1].style?.backgroundColor, isNull);
    expect(spans[2].style?.fontWeight, FontWeight.w800);
  });
}
