import 'dart:convert';
import 'dart:ffi';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';

import '../generated/proto/apipb/history.pb.dart';
import 'generated/anytty_terminal_input_bindings.dart';
import 'native_library_loader.dart';

final class TerminalInputRejected implements Exception {
  const TerminalInputRejected(this.operation);

  final String operation;

  @override
  String toString() => 'Terminal $operation requires confirmation';
}

final class TerminalInputException implements Exception {
  const TerminalInputException(this.operation, this.status);

  final String operation;
  final int status;

  @override
  String toString() => 'Terminal $operation failed with status $status';
}

final class TerminalInputGeometry {
  const TerminalInputGeometry({
    required this.screenWidth,
    required this.screenHeight,
    required this.cellWidth,
    required this.cellHeight,
    this.paddingTop = 0,
    this.paddingBottom = 0,
    this.paddingRight = 0,
    this.paddingLeft = 0,
  });

  final double screenWidth;
  final double screenHeight;
  final double cellWidth;
  final double cellHeight;
  final double paddingTop;
  final double paddingBottom;
  final double paddingRight;
  final double paddingLeft;
}

final class TerminalInputEncoder {
  TerminalInputEncoder._(this._native, this._input);

  final AnyttyTerminalInputNative _native;
  final Pointer<anytty_terminal_input_v1> _input;
  bool _closed = false;

  static const _geometryScale = 100.0;

  static TerminalInputEncoder open() {
    final native = AnyttyTerminalInputNative(
      loadAnyttyLibrary('libanytty_terminal_input.so'),
    );
    final version = native.anytty_terminal_input_abi_version();
    if (version != ANYTTY_TERMINAL_INPUT_ABI_VERSION) {
      throw StateError(
        'AnyTTY terminal input ABI mismatch: native=$version dart=$ANYTTY_TERMINAL_INPUT_ABI_VERSION',
      );
    }
    final out = calloc<Pointer<anytty_terminal_input_v1>>();
    try {
      _check('create', native.anytty_terminal_input_new(out));
      return TerminalInputEncoder._(native, out.value);
    } finally {
      calloc.free(out);
    }
  }

  void applyModes(TerminalModes modes) {
    _ensureOpen();
    final legacyNormalMouse =
        modes.mouseTracking &&
        !modes.mouseX10 &&
        !modes.mouseNormal &&
        !modes.mouseButtonEvent &&
        !modes.mouseAnyEvent;
    final value = calloc<anytty_terminal_modes_v1>();
    try {
      value.ref
        ..size = sizeOf<anytty_terminal_modes_v1>()
        ..application_cursor = modes.applicationCursor
        ..application_keypad = false
        ..alt_esc_prefix = true
        ..modify_other_keys_state_2 = false
        ..kitty_keyboard_flags = 0
        ..backarrow_key = false
        ..mouse_x10 = modes.mouseX10
        ..mouse_normal = modes.mouseNormal || legacyNormalMouse
        ..mouse_button_event = modes.mouseButtonEvent
        ..mouse_any_event = modes.mouseAnyEvent
        ..mouse_sgr = modes.mouseSgr
        ..bracketed_paste = modes.bracketedPaste;
      _check(
        'set_modes',
        _native.anytty_terminal_input_set_modes(_input, value),
      );
    } finally {
      calloc.free(value);
    }
  }

  void applyGeometry(TerminalInputGeometry geometry) {
    _ensureOpen();
    final value = calloc<anytty_terminal_geometry_v1>();
    try {
      value.ref
        ..size = sizeOf<anytty_terminal_geometry_v1>()
        ..screen_width = _scaledGeometry(geometry.screenWidth)
        ..screen_height = _scaledGeometry(geometry.screenHeight)
        ..cell_width = _scaledGeometry(geometry.cellWidth)
        ..cell_height = _scaledGeometry(geometry.cellHeight)
        ..padding_top = _scaledGeometry(geometry.paddingTop)
        ..padding_bottom = _scaledGeometry(geometry.paddingBottom)
        ..padding_right = _scaledGeometry(geometry.paddingRight)
        ..padding_left = _scaledGeometry(geometry.paddingLeft);
      _check(
        'set_geometry',
        _native.anytty_terminal_input_set_geometry(_input, value),
      );
    } finally {
      calloc.free(value);
    }
  }

  Uint8List encodeKey({
    required int hidUsage,
    int action = ANYTTY_TERMINAL_KEY_PRESS,
    int modifiers = 0,
    int unshiftedCodepoint = 0,
    bool composing = false,
    String text = '',
  }) {
    _ensureOpen();
    return _withUtf8(text, (data, length, out) {
      return _native.anytty_terminal_input_encode_key(
        _input,
        hidUsage,
        action,
        modifiers,
        unshiftedCodepoint,
        composing,
        data,
        length,
        out,
      );
    }, 'encode_key');
  }

  Uint8List encodeMouse({
    required int action,
    required int button,
    required int modifiers,
    required double x,
    required double y,
  }) {
    _ensureOpen();
    final out = calloc<anytty_terminal_buffer_v1>();
    try {
      final status = _native.anytty_terminal_input_encode_mouse(
        _input,
        action,
        button,
        modifiers,
        x * _geometryScale,
        y * _geometryScale,
        out,
      );
      _check('encode_mouse', status);
      return _copyAndFree(out.ref);
    } finally {
      calloc.free(out);
    }
  }

  Uint8List encodeScroll({
    required bool up,
    required int modifiers,
    required double x,
    required double y,
  }) {
    return encodeMouse(
      action: ANYTTY_TERMINAL_MOUSE_PRESS,
      button: up
          ? ANYTTY_TERMINAL_MOUSE_BUTTON_SCROLL_UP
          : ANYTTY_TERMINAL_MOUSE_BUTTON_SCROLL_DOWN,
      modifiers: modifiers,
      x: x,
      y: y,
    );
  }

  Uint8List encodePrimaryClick({
    required int modifiers,
    required double x,
    required double y,
  }) {
    final output = BytesBuilder(copy: false)
      ..add(
        encodeMouse(
          action: ANYTTY_TERMINAL_MOUSE_PRESS,
          button: ANYTTY_TERMINAL_MOUSE_BUTTON_LEFT,
          modifiers: modifiers,
          x: x,
          y: y,
        ),
      )
      ..add(
        encodeMouse(
          action: ANYTTY_TERMINAL_MOUSE_RELEASE,
          button: ANYTTY_TERMINAL_MOUSE_BUTTON_LEFT,
          modifiers: modifiers,
          x: x,
          y: y,
        ),
      );
    return output.takeBytes();
  }

  Uint8List encodePaste(String text, {bool allowUnsafe = false}) {
    _ensureOpen();
    return _withUtf8(text, (data, length, out) {
      return _native.anytty_terminal_input_encode_paste(
        _input,
        data,
        length,
        allowUnsafe,
        out,
      );
    }, 'paste');
  }

  void close() {
    if (_closed) return;
    _closed = true;
    _native.anytty_terminal_input_free(_input);
  }

  Uint8List _withUtf8(
    String text,
    int Function(
      Pointer<Uint8> data,
      int length,
      Pointer<anytty_terminal_buffer_v1> out,
    )
    invoke,
    String operation,
  ) {
    final bytes = utf8.encode(text);
    final data = bytes.isEmpty
        ? nullptr.cast<Uint8>()
        : calloc<Uint8>(bytes.length);
    final out = calloc<anytty_terminal_buffer_v1>();
    try {
      if (bytes.isNotEmpty) data.asTypedList(bytes.length).setAll(0, bytes);
      final status = invoke(data, bytes.length, out);
      _check(operation, status);
      return _copyAndFree(out.ref);
    } finally {
      calloc.free(out);
      if (bytes.isNotEmpty) calloc.free(data);
    }
  }

  Uint8List _copyAndFree(anytty_terminal_buffer_v1 buffer) {
    try {
      if (buffer.length == 0) return Uint8List(0);
      return Uint8List.fromList(buffer.data.asTypedList(buffer.length));
    } finally {
      if (buffer.data.address != 0) {
        _native.anytty_terminal_buffer_free(buffer);
      }
    }
  }

  void _ensureOpen() {
    if (_closed) throw StateError('Terminal input encoder is closed');
  }

  static void _check(String operation, int status) {
    if (status == ANYTTY_TERMINAL_STATUS_OK) return;
    if (status == ANYTTY_TERMINAL_STATUS_REJECTED) {
      throw TerminalInputRejected(operation);
    }
    throw TerminalInputException(operation, status);
  }

  static int _scaledGeometry(double value) {
    return (value * _geometryScale).round();
  }
}
