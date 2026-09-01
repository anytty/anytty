import 'dart:io';

import 'package:anytty_native/src/native/native_file_export.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('com.anytty.app/files.test');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() {
    messenger.setMockMethodCallHandler(channel, null);
  });

  test('exports a file through the platform save surface', () async {
    MethodCall? invocation;
    messenger.setMockMethodCallHandler(channel, (call) async {
      invocation = call;
      return 'content://downloads/hello.txt';
    });

    final uri = await const NativeFileExporter(channel: channel).exportFile(
      source: File('/tmp/hello.txt'),
      fileName: 'hello.txt',
      mimeType: 'text/plain',
    );

    expect(uri, Uri.parse('content://downloads/hello.txt'));
    expect(invocation?.method, 'exportFile');
    expect(invocation?.arguments, {
      'sourcePath': '/tmp/hello.txt',
      'fileName': 'hello.txt',
      'mimeType': 'text/plain',
    });
  });

  test('returns null when the platform save surface is cancelled', () async {
    messenger.setMockMethodCallHandler(channel, (_) async => null);

    final uri = await const NativeFileExporter(channel: channel).exportFile(
      source: File('/tmp/hello.txt'),
      fileName: 'hello.txt',
      mimeType: 'text/plain',
    );

    expect(uri, isNull);
  });

  test('propagates platform export failures', () async {
    messenger.setMockMethodCallHandler(channel, (_) async {
      throw PlatformException(code: 'export_failed', message: 'disk full');
    });

    await expectLater(
      const NativeFileExporter(channel: channel).exportFile(
        source: File('/tmp/hello.txt'),
        fileName: 'hello.txt',
        mimeType: 'text/plain',
      ),
      throwsA(
        isA<PlatformException>().having(
          (error) => error.code,
          'code',
          'export_failed',
        ),
      ),
    );
  });

  test('opens an exported file through the platform surface', () async {
    MethodCall? invocation;
    messenger.setMockMethodCallHandler(channel, (call) async {
      invocation = call;
      return null;
    });

    await const NativeFileOpener(channel: channel).openFile(
      uri: Uri.parse('content://downloads/report.pdf'),
      fileName: 'report.pdf',
      mimeType: 'application/pdf',
    );

    expect(invocation?.method, 'openFile');
    expect(invocation?.arguments, {
      'uri': 'content://downloads/report.pdf',
      'fileName': 'report.pdf',
      'mimeType': 'application/pdf',
    });
  });

  test('projects maintained MIME types and preserves an opaque fallback', () {
    expect(fileMimeType('report.pdf'), 'application/pdf');
    expect(fileMimeType('photo.png'), 'image/png');
    expect(fileMimeType('archive.unknown-anytty'), 'application/octet-stream');
  });
}
