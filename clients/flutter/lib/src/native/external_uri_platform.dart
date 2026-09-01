import 'package:flutter/services.dart';

abstract interface class ExternalUriLauncher {
  Future<void> open(Uri uri);
}

final class MethodChannelExternalUriLauncher implements ExternalUriLauncher {
  MethodChannelExternalUriLauncher({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel(_channelName);

  static const _channelName = 'com.anytty.app/external-uri';
  static const _allowedSchemes = {'http', 'https', 'mailto', 'tel'};
  static final instance = MethodChannelExternalUriLauncher();

  final MethodChannel _channel;

  @override
  Future<void> open(Uri uri) async {
    if (!_allowedSchemes.contains(uri.scheme.toLowerCase())) {
      throw const FormatException('External URI scheme is not allowed');
    }
    await _channel.invokeMethod<void>('openExternalUri', {
      'uri': uri.toString(),
    });
  }
}
