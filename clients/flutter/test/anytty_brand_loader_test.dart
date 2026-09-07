import 'dart:io';
import 'dart:ui' as ui;

import 'package:anytty_native/src/shared/presentation/anytty_brand_mark.dart';
import 'package:anytty_native/src/shared/presentation/anytty_rive.dart';
import 'package:anytty_native/src/features/browser/presentation/browser_perched_mascot.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:rive/rive.dart' as rive;

Future<void> loadMotion(WidgetTester tester, int count) async {
  for (var i = 0; i < 30; i++) {
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
    if (find
            .descendant(
              of: find.byType(AnyttyRive),
              matching: find.byType(CustomPaint),
            )
            .evaluate()
            .length ==
        count) {
      return;
    }
  }
  fail('Rive must render, not silently pass on its static fallback');
}

Future<ui.Image> capture(WidgetTester tester, GlobalKey key) async =>
    (key.currentContext!.findRenderObject()! as RenderRepaintBoundary).toImage(
      pixelRatio: 2,
    );

void main() {
  setUpAll(() async {
    await rive.RiveNative.init();
  });

  testWidgets('all nine scenes render native Rive inside stable bounds', (
    tester,
  ) async {
    final key = GlobalKey();
    final hashes = <int>{};
    for (final scene in AnyttyMascotScene.values) {
      await tester.pumpWidget(
        MaterialApp(
          home: Center(
            child: RepaintBoundary(
              key: key,
              child: AnyttyBrandLoader(scene: scene, height: 96),
            ),
          ),
        ),
      );
      await loadMotion(tester, 1);
      final bounds = tester.getRect(find.byType(AnyttyBrandLoader));
      await tester.pump(const Duration(milliseconds: 400));
      await tester.runAsync(() async {
        final image = await capture(tester, key);
        final data = (await image.toByteData())!.buffer.asUint8List();
        expect(data.where((value) => value != 0).length, greaterThan(1000));
        hashes.add(Object.hashAll(data));
        image.dispose();
      });
      expect(tester.getRect(find.byType(AnyttyBrandLoader)), bounds);
    }
    expect(hashes.length, greaterThanOrEqualTo(7));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('motion pauses in background, offstage and reduced motion', (
    tester,
  ) async {
    var reduced = false, tickers = true;
    late StateSetter update;
    await tester.pumpWidget(
      MaterialApp(
        home: StatefulBuilder(
          builder: (context, setState) {
            update = setState;
            return MediaQuery(
              data: MediaQuery.of(context).copyWith(disableAnimations: reduced),
              child: TickerMode(
                enabled: tickers,
                child: const Center(child: AnyttyBrandLoader()),
              ),
            );
          },
        ),
      ),
    );
    await loadMotion(tester, 1);
    await tester.pump(const Duration(milliseconds: 100));
    expect(tester.binding.hasScheduledFrame, isTrue);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    await tester.pumpAndSettle();
    expect(tester.binding.hasScheduledFrame, isFalse);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();
    expect(tester.binding.hasScheduledFrame, isTrue);
    update(() => tickers = false);
    await tester.pumpAndSettle();
    expect(tester.binding.hasScheduledFrame, isFalse);
    update(() {
      tickers = true;
      reduced = true;
    });
    await tester.pumpAndSettle();
    expect(tester.binding.hasScheduledFrame, isFalse);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets(
    'browser keeps opaque black body and animated tail behind field',
    (tester) async {
      final key = GlobalKey();
      for (final dark in [false, true]) {
        await tester.pumpWidget(
          MaterialApp(
            home: Center(
              child: RepaintBoundary(
                key: key,
                child: ColoredBox(
                  color: dark ? const Color(0xff17191c) : Colors.white,
                  child: SizedBox(
                    width: 360,
                    child: BrowserPerchedMascot(
                      visible: true,
                      child: Container(
                        height: 58,
                        decoration: BoxDecoration(
                          color: dark
                              ? const Color(0xff292d32)
                              : const Color(0xffeeeeee),
                          border: Border.all(color: Colors.grey),
                          borderRadius: BorderRadius.circular(8),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
        await loadMotion(tester, 3);
        final hashes = <int>{};
        for (var i = 0; i < 3; i++) {
          await tester.pump(const Duration(milliseconds: 700));
          await tester.runAsync(() async {
            final image = await capture(tester, key);
            final data = (await image.toByteData())!;
            var ink = 0;
            for (var y = 128 * 2; y < 145 * 2; y++) {
              for (var x = 230 * 2; x < 335 * 2; x++) {
                final offset = (y * image.width + x) * 4;
                if (data.getUint8(offset) < 15 &&
                    data.getUint8(offset + 1) < 15 &&
                    data.getUint8(offset + 3) > 250) {
                  ink++;
                }
              }
            }
            expect(
              ink,
              greaterThan(30),
              reason: 'Black belly must remain opaque below the field',
            );
            hashes.add(Object.hashAll(data.buffer.asUint8List()));
            final output = Platform.environment['ANYTTY_BRAND_PREVIEW_DIR'];
            if (output != null) {
              await Directory(output).create(recursive: true);
              await File(
                '$output/rive-browser-${dark ? 'dark' : 'light'}-$i.png',
              ).writeAsBytes(
                (await image.toByteData(format: ui.ImageByteFormat.png))!.buffer
                    .asUint8List(),
              );
            }
            image.dispose();
          });
        }
        expect(hashes.length, greaterThan(1));
      }
      await tester.pump(const Duration(seconds: 15));
      await tester.pumpAndSettle();
      expect(tester.binding.hasScheduledFrame, isFalse);
      await tester.pumpWidget(const SizedBox());
    },
  );
}
