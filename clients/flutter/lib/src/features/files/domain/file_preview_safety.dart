import 'package:cryptography/dart.dart';

import '../../../generated/proto/apipb/file.pb.dart';

const maximumFilePreviewBytes = 4 * 1024 * 1024;
const maximumFilePreviewSourceBytes = 64 * 1024 * 1024;
const maximumPreviewImageDimension = 2048;

final class FilePreviewSafetyException implements Exception {
  const FilePreviewSafetyException(this.message);

  final String message;

  @override
  String toString() => message;
}

Future<void> validateFilePreview(
  FilePreviewResult preview, {
  required int requestedBytes,
}) async {
  final limit = requestedBytes.clamp(1, maximumFilePreviewBytes);
  if (!preview.hasEntry() ||
      preview.entry.type != FileEntryType.FILE_ENTRY_TYPE_FILE ||
      preview.entry.size.toInt() < 0 ||
      preview.entry.size.toInt() > maximumFilePreviewSourceBytes) {
    throw const FilePreviewSafetyException(
      'File preview metadata is outside the supported limits',
    );
  }
  if (preview.content.length > limit) {
    throw const FilePreviewSafetyException(
      'File preview exceeded the requested byte limit',
    );
  }
  final mimeType = normalizedPreviewMimeType(preview.mimeType);
  if (!supportedPreviewMimeType(mimeType)) {
    throw FilePreviewSafetyException('Preview is not available for $mimeType');
  }
  if (mimeType.startsWith('image/') && preview.truncated) {
    throw const FilePreviewSafetyException(
      'An incomplete image cannot be previewed',
    );
  }
  if (preview.sha256.length != 32) {
    throw const FilePreviewSafetyException(
      'File preview did not include a valid digest',
    );
  }
  final digest = await const DartSha256().hash(preview.content);
  if (!_sameBytes(preview.sha256, digest.bytes)) {
    throw const FilePreviewSafetyException(
      'File preview digest verification failed',
    );
  }
}

String normalizedPreviewMimeType(String value) =>
    value.split(';').first.trim().toLowerCase();

bool supportedPreviewMimeType(String value) {
  final normalized = normalizedPreviewMimeType(value);
  if (normalized.startsWith('text/')) return true;
  return const {
    'application/json',
    'application/xml',
    'application/yaml',
    'application/x-yaml',
    'image/png',
    'image/jpeg',
    'image/gif',
    'image/webp',
    'image/bmp',
  }.contains(normalized);
}

bool _sameBytes(List<int> left, List<int> right) {
  if (left.length != right.length) return false;
  var difference = 0;
  for (var index = 0; index < left.length; index += 1) {
    difference |= left[index] ^ right[index];
  }
  return difference == 0;
}
