import 'dart:convert';
import 'dart:math';

import '../../../generated/proto/apipb/file.pb.dart';
import 'file_preview_safety.dart';

bool isHtmlPreview(String path, String mime) =>
    normalizedPreviewMimeType(mime) == 'text/html' ||
    const {'html', 'htm'}.contains(path.split('.').last.toLowerCase());

bool supportsWebFilePreview(String path, FilePreviewResult preview) {
  if (preview.truncated || preview.content.length > maximumFilePreviewBytes) {
    return false;
  }
  final mime = normalizedPreviewMimeType(preview.mimeType);
  if (!mime.startsWith('text/') &&
      !const {
        'application/json',
        'application/xml',
        'application/yaml',
        'application/x-yaml',
      }.contains(mime)) {
    return false;
  }
  return isHtmlPreview(path, mime) ||
      const {
        'md',
        'markdown',
        'json',
        'csv',
        'tsv',
        'xml',
        'yaml',
        'yml',
        'diff',
        'patch',
        'txt',
      }.contains(path.split('.').last.toLowerCase());
}

String buildFileWebPreview({
  required String template,
  required String path,
  required FilePreviewResult preview,
  required bool dark,
  required bool chinese,
}) {
  if (!supportsWebFilePreview(path, preview)) {
    throw const FilePreviewSafetyException(
      'This file cannot be rendered safely.',
    );
  }
  final payload = base64Encode(
    utf8.encode(
      jsonEncode({
        'name': path.split('/').last,
        'mime': normalizedPreviewMimeType(preview.mimeType),
        'content': base64Encode(preview.content),
        'dark': dark,
        'chinese': chinese,
        'html': isHtmlPreview(path, preview.mimeType),
      }),
    ),
  );
  final random = Random.secure();
  final nonce = base64UrlEncode(List.generate(24, (_) => random.nextInt(256)));
  return template
      .replaceAll('__ANYTTY_PREVIEW_DATA__', payload)
      .replaceAll('__ANYTTY_NONCE__', nonce);
}
