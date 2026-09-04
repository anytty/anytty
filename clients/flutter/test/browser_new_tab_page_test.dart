import 'package:anytty_native/src/features/browser/data/browser_bookmark_store.dart';
import 'package:anytty_native/src/features/browser/data/browser_history_store.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_new_tab_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('provides search, saved links, and recent pages', (tester) async {
    final searchController = TextEditingController();
    final searchFocusNode = FocusNode();
    addTearDown(searchController.dispose);
    addTearDown(searchFocusNode.dispose);
    String? searched;
    String? removed;
    var historyOpened = false;

    await tester.pumpWidget(
      MaterialApp(
        home: BrowserNewTabPage(
          searchController: searchController,
          searchFocusNode: searchFocusNode,
          onSearch: (value) async => searched = value,
          bookmarks: const [
            BrowserBookmark(url: 'https://anytty.dev', title: 'AnyTTY'),
          ],
          history: const [
            BrowserHistoryEntry(url: 'https://example.com', title: 'Example'),
          ],
          onRemoveBookmark: (url) async => removed = url,
          onOpenHistory: () => historyOpened = true,
        ),
      ),
    );

    expect(find.text('AnyTTY browser'), findsOneWidget);
    expect(find.text('Saved links'), findsOneWidget);
    expect(find.text('AnyTTY'), findsOneWidget);
    expect(find.text('Recent pages'), findsOneWidget);
    expect(find.text('Example'), findsOneWidget);

    await tester.enterText(
      find.byKey(const ValueKey('browser-new-tab-search')),
      'flutter webview',
    );
    await tester.tap(find.byTooltip('Search'));
    await tester.pump();
    expect(searched, 'flutter webview');

    await tester.tap(find.byTooltip('Remove saved link'));
    await tester.pump();
    expect(removed, 'https://anytty.dev');

    await tester.tap(find.text('View all'));
    expect(historyOpened, isTrue);
  });
}
