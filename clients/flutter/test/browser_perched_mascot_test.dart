import 'package:anytty_native/src/features/browser/presentation/browser_new_tab_page.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_perched_mascot.dart';
import 'package:anytty_native/src/shared/presentation/anytty_rive.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets(
    'mascot remains visible while typing and follows touch without blocking input',
    (tester) async {
      final search = TextEditingController();
      final address = TextEditingController(text: 'about:blank');
      final focus = FocusNode();
      final addressFocus = FocusNode();
      addTearDown(search.dispose);
      addTearDown(address.dispose);
      addTearDown(focus.dispose);
      addTearDown(addressFocus.dispose);
      var taps = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: Column(
              children: [
                Focus(focusNode: addressFocus, child: const SizedBox()),
                Expanded(
                  child: BrowserNewTabPage(
                    searchController: search,
                    searchFocusNode: focus,
                    addressController: address,
                    addressFocusNode: addressFocus,
                    onSearch: (_) async {},
                    onFocusSearch: () => taps++,
                    bookmarks: const [],
                    history: const [],
                    onRemoveBookmark: (_) async {},
                    onOpenHistory: () {},
                  ),
                ),
              ],
            ),
          ),
        ),
      );
      final field = find.byKey(const ValueKey('browser-new-tab-search'));
      final original = tester.getRect(field);
      bool visible() => tester
          .widget<BrowserPerchedMascot>(find.byType(BrowserPerchedMascot))
          .visible;
      expect(visible(), isTrue);
      // Click through the overlapping paws, not just the middle of the field.
      await tester.tapAt(Offset(original.right - 64, original.top + 4));
      await tester.pump(const Duration(milliseconds: 200));
      expect(taps, 1);
      expect(visible(), isTrue);
      expect(tester.getRect(field), original);
      focus.unfocus();
      search.text = 'query';
      await tester.pump();
      expect(visible(), isTrue);
      search.clear();
      await tester.pump();
      expect(visible(), isTrue);
      addressFocus.requestFocus();
      await tester.pump();
      expect(visible(), isTrue);
      address.text = 'a query in the toolbar';
      addressFocus.unfocus();
      await tester.pump();
      expect(visible(), isTrue);
      address.clear();
      await tester.pump();
      expect(visible(), isTrue);
      expect(tester.getRect(field), original);
      await tester.tapAt(const Offset(20, 30));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 200));
      final layers = tester.widgetList<AnyttyRive>(find.byType(AnyttyRive));
      expect(layers, hasLength(3));
      for (final layer in layers) {
        expect(layer.gaze.dx, lessThan(0));
        expect(layer.gaze.dy, lessThan(0));
        expect(layer.loop, isTrue);
      }
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
}
