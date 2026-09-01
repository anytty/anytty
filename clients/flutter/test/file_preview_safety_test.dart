import 'package:cryptography/dart.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/features/files/domain/file_preview_safety.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';

void main() {
  test(
    'accepts a bounded allowlisted preview with a matching digest',
    () async {
      final content = '{"ready":true}'.codeUnits;
      final digest = await const DartSha256().hash(content);
      final preview = _preview(
        content: content,
        digest: digest.bytes,
        mimeType: 'application/json; charset=utf-8',
      );

      await validateFilePreview(preview, requestedBytes: 1024);
      expect(normalizedPreviewMimeType(preview.mimeType), 'application/json');
    },
  );

  test('rejects a preview whose digest does not match its content', () async {
    final preview = _preview(
      content: 'hello'.codeUnits,
      digest: List<int>.filled(32, 1),
      mimeType: 'text/plain',
    );

    await expectLater(
      validateFilePreview(preview, requestedBytes: 1024),
      throwsA(
        isA<FilePreviewSafetyException>().having(
          (error) => error.message,
          'message',
          contains('digest verification failed'),
        ),
      ),
    );
  });

  test(
    'rejects oversized, unsupported, and truncated image previews',
    () async {
      final digest = await const DartSha256().hash([1, 2, 3, 4]);
      await expectLater(
        validateFilePreview(
          _preview(
            content: [1, 2, 3, 4],
            digest: digest.bytes,
            mimeType: 'text/plain',
          ),
          requestedBytes: 3,
        ),
        throwsA(isA<FilePreviewSafetyException>()),
      );
      await expectLater(
        validateFilePreview(
          _preview(
            content: [1, 2, 3, 4],
            digest: digest.bytes,
            mimeType: 'application/octet-stream',
          ),
          requestedBytes: 4,
        ),
        throwsA(isA<FilePreviewSafetyException>()),
      );
      await expectLater(
        validateFilePreview(
          _preview(
            content: [1, 2, 3, 4],
            digest: digest.bytes,
            mimeType: 'image/png',
            truncated: true,
          ),
          requestedBytes: 4,
        ),
        throwsA(isA<FilePreviewSafetyException>()),
      );
    },
  );
}

FilePreviewResult _preview({
  required List<int> content,
  required List<int> digest,
  required String mimeType,
  bool truncated = false,
}) {
  return FilePreviewResult(
    entry: FileEntry(
      path: '/tmp/preview',
      name: 'preview',
      type: FileEntryType.FILE_ENTRY_TYPE_FILE,
      size: Int64(content.length),
    ),
    mimeType: mimeType,
    content: content,
    truncated: truncated,
    sha256: digest,
  );
}
