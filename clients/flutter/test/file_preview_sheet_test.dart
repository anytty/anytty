import 'dart:async';
import 'dart:convert';

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/features/files/presentation/file_manager_screen.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';

void main() {
  testWidgets('keeps close available while the remote preview is offline', (
    tester,
  ) async {
    final pending = Completer<FilePreviewResult>();
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: Builder(
            builder: (context) => Scaffold(
              body: TextButton(
                onPressed: () => showModalBottomSheet<void>(
                  context: context,
                  isScrollControlled: true,
                  builder: (context) => FilePreviewSheet(
                    endpointId: 'offline-endpoint',
                    path: '/remote/waiting.txt',
                    entry: FileEntry(
                      path: '/remote/waiting.txt',
                      name: 'waiting.txt',
                      type: FileEntryType.FILE_ENTRY_TYPE_FILE,
                      size: Int64(12),
                    ),
                    preview: pending.future,
                  ),
                ),
                child: const Text('Open preview'),
              ),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open preview'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(
      tester.getSize(find.byTooltip('Close preview')).height,
      greaterThanOrEqualTo(44),
    );

    await tester.tap(find.byTooltip('Close preview'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.byTooltip('Close preview'), findsNothing);
  });

  testWidgets('decodes raster previews through the bounded resize provider', (
    tester,
  ) async {
    final png = base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
    );
    await tester.pumpWidget(
      ProviderScope(
        child: MaterialApp(
          home: FilePreviewSheet(
            endpointId: 'endpoint',
            path: '/remote/pixel.png',
            entry: FileEntry(
              path: '/remote/pixel.png',
              name: 'pixel.png',
              type: FileEntryType.FILE_ENTRY_TYPE_FILE,
              size: Int64(png.length),
            ),
            preview: Future.value(
              FilePreviewResult(
                entry: FileEntry(
                  path: '/remote/pixel.png',
                  name: 'pixel.png',
                  type: FileEntryType.FILE_ENTRY_TYPE_FILE,
                  size: Int64(png.length),
                ),
                mimeType: 'image/png',
                content: png,
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    final image = tester.widget<Image>(find.byType(Image));
    final provider = image.image as ResizeImage;
    expect(provider.width, 2048);
    expect(provider.height, 2048);
    expect(provider.policy, ResizeImagePolicy.fit);
  });
}
