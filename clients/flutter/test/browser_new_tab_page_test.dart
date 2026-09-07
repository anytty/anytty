import 'dart:io';
import 'dart:ui' as ui;

import 'package:anytty_native/src/app/anytty_localizations.dart';
import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/browser/data/browser_bookmark_store.dart';
import 'package:anytty_native/src/features/browser/data/browser_history_store.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_new_tab_page.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_perched_mascot.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
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
    var searchFocused = false;

    await tester.pumpWidget(
      MaterialApp(
        home: BrowserNewTabPage(
          searchController: searchController,
          searchFocusNode: searchFocusNode,
          onSearch: (value) async => searched = value,
          onFocusSearch: () => searchFocused = true,
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

    expect(find.byType(BrowserPerchedMascot), findsOneWidget);
    expect(find.text('Saved links'), findsOneWidget);
    expect(find.text('AnyTTY'), findsNWidgets(2));
    expect(find.text('Recent pages'), findsOneWidget);
    expect(find.text('Example'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('browser-new-tab-search')));
    expect(searchFocused, isTrue);
    await tester.enterText(
      find.byKey(const ValueKey('browser-new-tab-search')),
      'flutter webview',
    );
    await tester.tap(find.byTooltip('Search'));
    await tester.pump();
    expect(searched, 'flutter webview');

    await tester.ensureVisible(find.byTooltip('https://anytty.dev'));
    await tester.longPress(find.byTooltip('https://anytty.dev'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Remove saved link'));
    await tester.pump();
    expect(removed, 'https://anytty.dev');

    await tester.ensureVisible(find.text('View all'));
    await tester.tap(find.text('View all'));
    expect(historyOpened, isTrue);
  });

  testWidgets(
    'opens saved and recent pages, switches device, ignores empty search',
    (tester) async {
      final controller = TextEditingController();
      final focus = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focus.dispose);
      final opened = <String>[];
      var switched = false;
      await tester.pumpWidget(
        MaterialApp(
          theme: anyttyTheme(Brightness.light),
          home: Scaffold(
            body: BrowserNewTabPage(
              searchController: controller,
              searchFocusNode: focus,
              onSearch: (value) async => opened.add(value),
              onFocusSearch: () {},
              bookmarks: const [
                BrowserBookmark(
                  url: 'https://example.com/saved',
                  title: 'Saved page',
                ),
              ],
              history: const [
                BrowserHistoryEntry(
                  url: 'https://example.com/recent',
                  title: 'Recent page',
                ),
              ],
              onRemoveBookmark: (_) async {},
              onOpenHistory: () {},
              endpointLabel: 'My workstation',
              onSwitchEndpoint: () => switched = true,
            ),
          ),
        ),
      );
      await tester.tap(find.byTooltip('Search'));
      expect(opened, isEmpty);
      await tester.tap(find.byTooltip('Switch device'));
      expect(switched, isTrue);
      await tester.ensureVisible(find.text('Saved page'));
      await tester.tap(find.text('Saved page'));
      await tester.ensureVisible(find.text('Recent page'));
      await tester.tap(find.text('Recent page'));
      expect(opened, [
        'https://example.com/saved',
        'https://example.com/recent',
      ]);
      expect(tester.takeException(), isNull);
    },
  );

  for (final brightness in Brightness.values) {
    for (final scenario in [
      (name: 'phone', size: const Size(375, 812), scale: 1.0),
      (name: 'small', size: const Size(320, 640), scale: 1.0),
      (name: 'landscape', size: const Size(844, 390), scale: 1.0),
      (name: 'tablet', size: const Size(768, 1024), scale: 1.0),
      (name: 'large-text', size: const Size(375, 812), scale: 3.0),
    ]) {
      for (final language in ['en', 'zh']) {
        testWidgets(
          'brand layout ${brightness.name} ${scenario.name} $language',
          (tester) async {
            await tester.binding.setSurfaceSize(scenario.size);
            addTearDown(() => tester.binding.setSurfaceSize(null));
            final controller = TextEditingController();
            final focus = FocusNode();
            addTearDown(controller.dispose);
            addTearDown(focus.dispose);
            final previewDirectory =
                Platform.environment['ANYTTY_BRAND_PREVIEW_DIR'];
            final previewFont =
                Platform.environment['ANYTTY_BRAND_PREVIEW_FONT'];
            if (previewDirectory != null && previewFont != null) {
              await tester.runAsync(() async {
                final loader = FontLoader('BrandPreview');
                loader.addFont(
                  Future.value(
                    ByteData.sublistView(await File(previewFont).readAsBytes()),
                  ),
                );
                await loader.load();
                final wordmark = FontLoader('JetBrainsMonoNerd');
                wordmark.addFont(
                  rootBundle.load(
                    'assets/fonts/JetBrainsMonoNerdFont-Bold.ttf',
                  ),
                );
                await wordmark.load();
                final icons = FontLoader('MaterialIcons');
                icons.addFont(
                  rootBundle.load('fonts/MaterialIcons-Regular.otf'),
                );
                await icons.load();
              });
            }
            final boundaryKey = GlobalKey();
            final baseTheme = anyttyTheme(brightness);
            await tester.pumpWidget(
              MaterialApp(
                theme: previewFont == null
                    ? baseTheme
                    : baseTheme.copyWith(
                        textTheme: baseTheme.textTheme.apply(
                          fontFamily: 'BrandPreview',
                        ),
                      ),
                locale: Locale(language),
                supportedLocales: AnyttyLocalizations.supportedLocales,
                localizationsDelegates: const [
                  AnyttyLocalizations.delegate,
                  ...GlobalMaterialLocalizations.delegates,
                ],
                builder: (context, child) => MediaQuery(
                  data: MediaQuery.of(context).copyWith(
                    textScaler: TextScaler.linear(scenario.scale),
                    disableAnimations: true,
                  ),
                  child: child!,
                ),
                home: RepaintBoundary(
                  key: boundaryKey,
                  child: Scaffold(
                    body: SafeArea(
                      child: BrowserNewTabPage(
                        searchController: controller,
                        searchFocusNode: focus,
                        onSearch: (_) async {},
                        onFocusSearch: () {},
                        bookmarks: const [
                          BrowserBookmark(
                            url: 'http://localhost:3000',
                            title: 'localhost:3000',
                          ),
                          BrowserBookmark(
                            url: 'https://anytty.com',
                            title: 'AnyTTY',
                          ),
                        ],
                        history: const [
                          BrowserHistoryEntry(
                            url: 'http://localhost:3000/settings',
                            title: 'localhost:3000/settings',
                          ),
                        ],
                        onRemoveBookmark: (_) async {},
                        onOpenHistory: () {},
                        endpointLabel: 'MacBook Pro',
                        onSwitchEndpoint: () {},
                      ),
                    ),
                  ),
                ),
              ),
            );
            await tester.pumpAndSettle();
            expect(tester.takeException(), isNull);
            expect(find.byType(BrowserPerchedMascot), findsOneWidget);
            if (previewDirectory != null && scenario.scale == 1) {
              await tester.runAsync(() async {
                final boundary =
                    boundaryKey.currentContext!.findRenderObject()!
                        as RenderRepaintBoundary;
                final image = await boundary.toImage(pixelRatio: 2);
                final png = await image.toByteData(
                  format: ui.ImageByteFormat.png,
                );
                await Directory(previewDirectory).create(recursive: true);
                await File(
                  '$previewDirectory/browser-${brightness.name}-${scenario.name}-$language.png',
                ).writeAsBytes(png!.buffer.asUint8List());
                image.dispose();
              });
            }
            await tester.ensureVisible(
              find.byTooltip('http://localhost:3000').first,
            );
            await tester.pumpAndSettle();
            expect(tester.takeException(), isNull);
            await tester.ensureVisible(
              find.text('localhost:3000/settings').first,
            );
            await tester.pumpAndSettle();
            expect(tester.takeException(), isNull);
          },
        );
      }
    }
  }
}
