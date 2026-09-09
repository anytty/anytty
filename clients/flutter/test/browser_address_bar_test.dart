import 'package:anytty_native/src/features/browser/presentation/browser_address_bar.dart';
import 'package:anytty_native/src/features/browser/data/browser_history_store.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  late TextEditingController text;
  late FocusNode focus;
  late List<String> opened;
  setUp(() {
    text = TextEditingController();
    focus = FocusNode();
    opened = [];
  });
  tearDown(() {
    text.dispose();
    focus.dispose();
  });
  Future<void> show(WidgetTester tester) => tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: Column(
          children: [
            SizedBox(
              width: 280,
              child: BrowserAddressBar(
                controller: text,
                focusNode: focus,
                enabled: true,
                history: const [
                  BrowserHistoryEntry(
                    url: 'https://app.example/settings',
                    title: 'Application',
                  ),
                ],
                onNavigate: (value) async => opened.add(value),
                onDismiss: () {
                  focus.unfocus();
                  text.clear();
                },
              ),
            ),
            const Expanded(
              child: ColoredBox(color: Colors.white, child: SizedBox.expand()),
            ),
          ],
        ),
      ),
    ),
  );
  final field = find.byKey(const ValueKey('browser-address-field'));
  final options = find.byKey(const ValueKey('browser-address-suggestions'));

  testWidgets(
    'touch outside dismisses focus and suggestions; clear stays focused',
    (tester) async {
      await show(tester);
      await tester.enterText(field, 'app');
      await tester.pumpAndSettle();
      expect(focus.hasFocus, isTrue);
      expect(options, findsOneWidget);
      expect(
        tester.getSize(options).width,
        lessThanOrEqualTo(tester.getSize(field).width),
      );
      await tester.tap(find.byTooltip('Clear address'));
      await tester.pumpAndSettle();
      expect(focus.hasFocus, isTrue);
      await tester.tapAt(const Offset(350, 500));
      await tester.pumpAndSettle();
      expect(focus.hasFocus, isFalse);
      expect(options, findsNothing);
      expect(opened, isEmpty);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'Go navigates typed address exactly once, not the first history match',
    (tester) async {
      await show(tester);
      await tester.enterText(field, 'https://app.example');
      await tester.pumpAndSettle();
      await tester.testTextInput.receiveAction(TextInputAction.go);
      await tester.pumpAndSettle();
      expect(opened, ['https://app.example']);
      expect(focus.hasFocus, isFalse);
      expect(options, findsNothing);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'suggestion taps navigate once and Escape cancels without navigation',
    (tester) async {
      await show(tester);
      await tester.enterText(field, 'app');
      await tester.pumpAndSettle();
      await tester.tap(find.text('Application'));
      await tester.pumpAndSettle();
      expect(opened, ['https://app.example/settings']);
      expect(focus.hasFocus, isFalse);
      await tester.enterText(field, 'another');
      await tester.pumpAndSettle();
      await tester.sendKeyEvent(LogicalKeyboardKey.escape);
      await tester.pumpAndSettle();
      expect(focus.hasFocus, isFalse);
      expect(options, findsNothing);
      expect(opened, hasLength(1));
      await tester.pumpWidget(const SizedBox());
    },
  );
}
