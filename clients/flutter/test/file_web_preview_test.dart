import 'dart:convert';

import 'package:anytty_native/src/features/files/domain/file_web_preview.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'allows complete text formats but rejects binary and truncated files',
    () {
      for (final extension in [
        'html',
        'htm',
        'md',
        'json',
        'csv',
        'tsv',
        'yaml',
      ]) {
        expect(
          supportsWebFilePreview(
            '/preview.$extension',
            FilePreviewResult(mimeType: 'text/plain', content: [65]),
          ),
          isTrue,
        );
      }
      expect(
        supportsWebFilePreview(
          '/index.html',
          FilePreviewResult(mimeType: 'text/html', truncated: true),
        ),
        isFalse,
      );
      expect(
        supportsWebFilePreview(
          '/index.html',
          FilePreviewResult(mimeType: 'application/octet-stream'),
        ),
        isFalse,
      );
    },
  );

  test('encodes filename and content as data, never executable markup', () {
    const attack = '</script><script>alert(1)</script>';
    final html = buildFileWebPreview(
      template: 'nonce=__ANYTTY_NONCE__;data=__ANYTTY_PREVIEW_DATA__',
      path: '/index.html',
      preview: FilePreviewResult(
        mimeType: 'text/html',
        content: utf8.encode(attack),
      ),
      dark: true,
      chinese: true,
    );
    expect(html, isNot(contains(attack)));
    expect(html, isNot(contains('__ANYTTY')));
    final payload = jsonDecode(
      utf8.decode(base64Decode(html.split(';data=').last)),
    ) as Map<String, dynamic>;
    expect(utf8.decode(base64Decode(payload['content'] as String)), attack);
    expect(payload['html'], isTrue);
  });
}
