import 'package:anytty_native/src/native/android_terminal_input_platform.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('test.android-terminal-input');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'requests native input ownership through the platform channel',
    () async {
      final calls = <MethodCall>[];
      messenger.setMockMethodCallHandler(channel, (call) async {
        calls.add(call);
        return null;
      });
      final platform = AndroidTerminalInputPlatform(channel: channel);

      await platform.show('terminal-7');
      await platform.hide('terminal-7');
      await platform.release('terminal-7');

      expect(calls.map((call) => call.method), ['show', 'hide', 'release']);
      expect(calls.map((call) => call.arguments), [
        {'owner': 'terminal-7'},
        {'owner': 'terminal-7'},
        {'owner': 'terminal-7'},
      ]);
    },
  );

  test('decodes committed text and bounded Backspace repeats', () {
    final text = AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
      'owner': 'terminal-7',
      'type': 'text',
      'text': '微信输入',
    });
    final backspace =
        AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
          'owner': 'terminal-7',
          'type': 'backspace',
          'count': 100,
        });

    expect(text, isA<AndroidTerminalTextInput>());
    expect((text as AndroidTerminalTextInput).text, '微信输入');
    expect(backspace, isA<AndroidTerminalBackspaceInput>());
    expect((backspace as AndroidTerminalBackspaceInput).count, 64);
  });

  test('decodes active and completed IME composition', () {
    final active = AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent(
      {
        'owner': 'terminal-7',
        'type': 'composition',
        'text': '正在识别',
        'active': true,
      },
    );
    final completed =
        AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
          'owner': 'terminal-7',
          'type': 'composition',
          'text': '',
          'active': false,
        });

    expect(active, isA<AndroidTerminalCompositionInput>());
    expect((active as AndroidTerminalCompositionInput).text, '正在识别');
    expect(active.active, isTrue);
    expect(completed, isA<AndroidTerminalCompositionInput>());
    expect((completed as AndroidTerminalCompositionInput).active, isFalse);
  });

  test('maps native Enter and hardware keys to USB HID usages', () {
    final enter = AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
      'owner': 'terminal-7',
      'type': 'enter',
    });
    final controlA =
        AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
          'owner': 'terminal-7',
          'type': 'key',
          'keyCode': 29,
          'modifiers': 2,
          'unshiftedCodepoint': 97,
          'text': 'a',
        });

    expect((enter as AndroidTerminalKeyInput).hidUsage, 0x00070028);
    expect((controlA as AndroidTerminalKeyInput).hidUsage, 0x00070004);
    expect(controlA.modifiers, 2);
    expect(controlA.unshiftedCodepoint, 97);
    expect(androidKeyCodeToUsbHidUsage(67), 0x0007002a);
    expect(androidKeyCodeToUsbHidUsage(19), 0x00070052);
  });

  test('rejects malformed or empty native events', () {
    expect(
      AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent(null),
      isNull,
    );
    expect(
      AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
        'owner': '',
        'type': 'backspace',
        'count': 1,
      }),
      isNull,
    );
    expect(
      AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
        'owner': 'terminal-7',
        'type': 'text',
        'text': '',
      }),
      isNull,
    );
    expect(
      AndroidTerminalInputPlatform.decodeAndroidTerminalInputEvent({
        'owner': 'terminal-7',
        'type': 'composition',
        'text': 'partial',
      }),
      isNull,
    );
  });
}
