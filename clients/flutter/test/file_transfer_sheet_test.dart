import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/files/presentation/file_transfer_controller.dart';
import 'package:anytty_native/src/features/files/presentation/file_transfer_sheet.dart';
import 'package:anytty_native/src/native/native_file_export.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('matches the full-screen transfer center and selection flow', (
    tester,
  ) async {
    final active = _item('active', FileTransferStatus.transferring);
    final paused = _item('paused', FileTransferStatus.paused);
    final failed = _item('failed', FileTransferStatus.failed);
    final completed = _item(
      'completed',
      FileTransferStatus.completed,
      savedUri: Uri.parse('content://downloads/completed.txt'),
    );
    final opener = _RecordingFileOpener();
    final controller = FileTransferController(
      session: (_) async => throw StateError('offline'),
      fileOpener: opener,
      initialItems: [active, paused, failed, completed],
    );
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => showFileTransferCenter(context, controller),
              child: const Text('Open transfers'),
            ),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Open transfers'));
    await tester.pumpAndSettle();

    expect(find.text('Transfer Center'), findsOneWidget);
    expect(find.byType(BottomSheet), findsNothing);
    expect(find.text('Select'), findsOneWidget);
    expect(find.text('Done'), findsOneWidget);
    expect(find.text('Failed'), findsOneWidget);
    expect(
      tester.getSize(find.byKey(const ValueKey('transfer-completed'))).height,
      112,
    );

    await tester.tap(find.byTooltip('Open completed.txt'));
    await tester.pump();
    expect(opener.opened, [Uri.parse('content://downloads/completed.txt')]);

    await tester.tap(find.byTooltip('Select transfers'));
    await tester.pump();
    expect(find.text('0 selected'), findsOneWidget);
    expect(find.text('Pause'), findsOneWidget);
    expect(find.text('Start'), findsOneWidget);

    await tester.tap(find.text('Select all'));
    await tester.pump();
    expect(find.text('4 selected'), findsOneWidget);

    await tester.tap(find.byTooltip('Pause selected transfers'));
    await tester.pump();
    expect(active.status, FileTransferStatus.paused);

    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(find.text('Transfer Center'), findsOneWidget);
    expect(find.text('Select'), findsOneWidget);

    await tester.binding.handlePopRoute();
    await tester.pump();
    await tester.pumpAndSettle();
    expect(find.text('Transfer Center'), findsNothing);
  });

  testWidgets('keeps a completed download when system open fails', (
    tester,
  ) async {
    final completed = _item(
      'completed',
      FileTransferStatus.completed,
      savedUri: Uri.parse('content://downloads/completed.txt'),
    );
    final controller = FileTransferController(
      session: (_) async => throw StateError('offline'),
      fileOpener: _ThrowingFileOpener(),
      initialItems: [completed],
    );
    addTearDown(controller.dispose);

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Builder(
          builder: (context) => Scaffold(
            body: TextButton(
              onPressed: () => showFileTransferCenter(context, controller),
              child: const Text('Open transfers'),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('Open transfers'));
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Open completed.txt'));
    await tester.pump();

    expect(find.text('This file could not be opened'), findsOneWidget);
    expect(find.text('completed.txt'), findsOneWidget);
    expect(controller.items, hasLength(1));
  });
}

FileTransferViewItem _item(
  String id,
  FileTransferStatus status, {
  Uri? savedUri,
}) =>
    FileTransferViewItem(
        id: id,
        endpointId: 'endpoint',
        endpointLabel: 'Test device',
        name: '$id.txt',
        remotePath: '/tmp/$id.txt',
        direction: FileTransferDirection.download,
        totalBytes: 10,
      )
      ..transferredBytes = status == FileTransferStatus.completed ? 10 : 4
      ..status = status
      ..savedUri = savedUri;

final class _RecordingFileOpener implements FileOpener {
  final List<Uri> opened = [];

  @override
  Future<void> openFile({
    required Uri uri,
    required String fileName,
    required String mimeType,
  }) async {
    opened.add(uri);
  }
}

final class _ThrowingFileOpener implements FileOpener {
  @override
  Future<void> openFile({
    required Uri uri,
    required String fileName,
    required String mimeType,
  }) => throw StateError('no compatible application');
}
