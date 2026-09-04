import 'dart:convert';

String browserDeviceStorageKey(String legacyKey, String endpointId) {
  final id = endpointId.trim();
  if (id.isEmpty) {
    throw ArgumentError.value(endpointId, 'endpointId', 'must not be empty');
  }
  final encoded = base64Url.encode(utf8.encode(id)).replaceAll('=', '');
  return '$legacyKey.device.$encoded';
}
