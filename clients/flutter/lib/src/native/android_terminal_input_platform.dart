import 'dart:async';

import 'package:flutter/services.dart';

sealed class AndroidTerminalInputEvent {
  const AndroidTerminalInputEvent(this.owner);

  final String owner;
}

final class AndroidTerminalTextInput extends AndroidTerminalInputEvent {
  const AndroidTerminalTextInput(super.owner, this.text);

  final String text;
}

final class AndroidTerminalCompositionInput extends AndroidTerminalInputEvent {
  const AndroidTerminalCompositionInput(
    super.owner, {
    required this.text,
    required this.active,
  });

  final String text;
  final bool active;
}

final class AndroidTerminalBackspaceInput extends AndroidTerminalInputEvent {
  const AndroidTerminalBackspaceInput(super.owner, this.count);

  final int count;
}

final class AndroidTerminalKeyInput extends AndroidTerminalInputEvent {
  const AndroidTerminalKeyInput(
    super.owner, {
    required this.hidUsage,
    required this.modifiers,
    required this.unshiftedCodepoint,
    required this.text,
  });

  final int? hidUsage;
  final int modifiers;
  final int unshiftedCodepoint;
  final String text;
}

final class AndroidTerminalInputPlatform {
  AndroidTerminalInputPlatform({MethodChannel? channel})
    : _channel = channel ?? const MethodChannel(_channelName) {
    _channel.setMethodCallHandler(_handleMethodCall);
  }

  static final instance = AndroidTerminalInputPlatform();
  static const _channelName = 'com.anytty.app/terminal-input';
  static const _usbKeyboardPage = 0x00070000;

  final MethodChannel _channel;
  final _events = StreamController<AndroidTerminalInputEvent>.broadcast(
    sync: true,
  );

  Stream<AndroidTerminalInputEvent> eventsFor(String owner) =>
      _events.stream.where((event) => event.owner == owner);

  Future<void> show(String owner) =>
      _channel.invokeMethod<void>('show', {'owner': owner});

  Future<void> hide(String owner) =>
      _channel.invokeMethod<void>('hide', {'owner': owner});

  Future<void> release(String owner) =>
      _channel.invokeMethod<void>('release', {'owner': owner});

  Future<void> _handleMethodCall(MethodCall call) async {
    if (call.method != 'event') return;
    final event = decodeAndroidTerminalInputEvent(call.arguments);
    if (event != null) _events.add(event);
  }

  static AndroidTerminalInputEvent? decodeAndroidTerminalInputEvent(
    Object? arguments,
  ) {
    if (arguments is! Map) return null;
    final values = Map<Object?, Object?>.from(arguments);
    final owner = values['owner'];
    final type = values['type'];
    if (owner is! String || owner.isEmpty || type is! String) return null;
    switch (type) {
      case 'text':
        final text = values['text'];
        return text is String && text.isNotEmpty
            ? AndroidTerminalTextInput(owner, text)
            : null;
      case 'composition':
        final text = values['text'];
        final active = values['active'];
        return text is String && active is bool
            ? AndroidTerminalCompositionInput(owner, text: text, active: active)
            : null;
      case 'backspace':
        final count = values['count'];
        return count is num && count.toInt() > 0
            ? AndroidTerminalBackspaceInput(
                owner,
                count.toInt().clamp(1, 64).toInt(),
              )
            : null;
      case 'enter':
        return AndroidTerminalKeyInput(
          owner,
          hidUsage: _usbKeyboardPage | 0x28,
          modifiers: 0,
          unshiftedCodepoint: 0,
          text: '',
        );
      case 'key':
        final keyCode = values['keyCode'];
        if (keyCode is! num) return null;
        final modifiers = values['modifiers'];
        final unshiftedCodepoint = values['unshiftedCodepoint'];
        final text = values['text'];
        return AndroidTerminalKeyInput(
          owner,
          hidUsage: androidKeyCodeToUsbHidUsage(keyCode.toInt()),
          modifiers: modifiers is num ? modifiers.toInt() : 0,
          unshiftedCodepoint: unshiftedCodepoint is num
              ? unshiftedCodepoint.toInt()
              : 0,
          text: text is String ? text : '',
        );
      default:
        return null;
    }
  }
}

int? androidKeyCodeToUsbHidUsage(int keyCode) {
  const page = 0x00070000;
  if (keyCode >= 29 && keyCode <= 54) return page | (0x04 + keyCode - 29);
  if (keyCode >= 8 && keyCode <= 16) return page | (0x1e + keyCode - 8);
  if (keyCode == 7) return page | 0x27;
  if (keyCode >= 131 && keyCode <= 142) {
    return page | (0x3a + keyCode - 131);
  }
  return switch (keyCode) {
    66 => page | 0x28,
    111 => page | 0x29,
    67 => page | 0x2a,
    61 => page | 0x2b,
    62 => page | 0x2c,
    69 => page | 0x2d,
    70 => page | 0x2e,
    71 => page | 0x2f,
    72 => page | 0x30,
    73 => page | 0x31,
    74 => page | 0x33,
    75 => page | 0x34,
    68 => page | 0x35,
    55 => page | 0x36,
    56 => page | 0x37,
    76 => page | 0x38,
    124 => page | 0x49,
    122 => page | 0x4a,
    92 => page | 0x4b,
    112 => page | 0x4c,
    123 => page | 0x4d,
    93 => page | 0x4e,
    22 => page | 0x4f,
    21 => page | 0x50,
    20 => page | 0x51,
    19 => page | 0x52,
    _ => null,
  };
}
