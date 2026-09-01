import 'terminal_modifiers.dart';

const terminalUsbKeyboardPage = 0x00070000;

enum TerminalQuickActionKind { key, chord, text }

final class TerminalQuickKey {
  const TerminalQuickKey({
    required this.id,
    required this.label,
    this.hidUsage,
    this.text,
    this.modifier,
    this.unshiftedCodepoint = 0,
  });

  final String id;
  final String label;
  final int? hidUsage;
  final String? text;
  final TerminalModifier? modifier;
  final int unshiftedCodepoint;

  bool get supportsChord => hidUsage != null && modifier == null;

  String get accessibilityLabel => switch (id) {
    'left' => 'Left arrow',
    'down' => 'Down arrow',
    'up' => 'Up arrow',
    'right' => 'Right arrow',
    _ => label,
  };
}

final terminalQuickKeys = <TerminalQuickKey>[
  const TerminalQuickKey(
    id: 'escape',
    label: 'Esc',
    hidUsage: terminalUsbKeyboardPage | 0x29,
  ),
  const TerminalQuickKey(
    id: 'tab',
    label: 'Tab',
    hidUsage: terminalUsbKeyboardPage | 0x2b,
  ),
  const TerminalQuickKey(
    id: 'enter',
    label: 'Enter',
    hidUsage: terminalUsbKeyboardPage | 0x28,
  ),
  const TerminalQuickKey(
    id: 'backspace',
    label: 'Backspace',
    hidUsage: terminalUsbKeyboardPage | 0x2a,
  ),
  const TerminalQuickKey(
    id: 'delete',
    label: 'Del',
    hidUsage: terminalUsbKeyboardPage | 0x4c,
  ),
  const TerminalQuickKey(
    id: 'left',
    label: '←',
    hidUsage: terminalUsbKeyboardPage | 0x50,
  ),
  const TerminalQuickKey(
    id: 'down',
    label: '↓',
    hidUsage: terminalUsbKeyboardPage | 0x51,
  ),
  const TerminalQuickKey(
    id: 'up',
    label: '↑',
    hidUsage: terminalUsbKeyboardPage | 0x52,
  ),
  const TerminalQuickKey(
    id: 'right',
    label: '→',
    hidUsage: terminalUsbKeyboardPage | 0x4f,
  ),
  const TerminalQuickKey(
    id: 'home',
    label: 'Home',
    hidUsage: terminalUsbKeyboardPage | 0x4a,
  ),
  const TerminalQuickKey(
    id: 'end',
    label: 'End',
    hidUsage: terminalUsbKeyboardPage | 0x4d,
  ),
  const TerminalQuickKey(
    id: 'page-up',
    label: 'PgUp',
    hidUsage: terminalUsbKeyboardPage | 0x4b,
  ),
  const TerminalQuickKey(
    id: 'page-down',
    label: 'PgDn',
    hidUsage: terminalUsbKeyboardPage | 0x4e,
  ),
  const TerminalQuickKey(
    id: 'insert',
    label: 'Insert',
    hidUsage: terminalUsbKeyboardPage | 0x49,
  ),
  const TerminalQuickKey(
    id: 'space',
    label: 'Space',
    hidUsage: terminalUsbKeyboardPage | 0x2c,
    text: ' ',
    unshiftedCodepoint: 32,
  ),
  const TerminalQuickKey(
    id: 'control',
    label: 'Ctrl',
    modifier: TerminalModifier.control,
  ),
  const TerminalQuickKey(
    id: 'alt',
    label: 'Alt',
    modifier: TerminalModifier.alt,
  ),
  const TerminalQuickKey(
    id: 'shift',
    label: 'Shift',
    modifier: TerminalModifier.shift,
  ),
  const TerminalQuickKey(id: 'slash', label: '/', text: '/'),
  const TerminalQuickKey(id: 'pipe', label: '|', text: '|'),
  const TerminalQuickKey(id: 'minus', label: '-', text: '-'),
  const TerminalQuickKey(id: 'tilde', label: '~', text: '~'),
  const TerminalQuickKey(id: 'backslash', label: r'\', text: r'\'),
  ..._letterKeys,
  ..._digitKeys,
  ..._functionKeys,
];

const _letterKeys = <TerminalQuickKey>[
  TerminalQuickKey(
    id: 'a',
    label: 'A',
    hidUsage: terminalUsbKeyboardPage | 0x04,
    text: 'a',
    unshiftedCodepoint: 97,
  ),
  TerminalQuickKey(
    id: 'b',
    label: 'B',
    hidUsage: terminalUsbKeyboardPage | 0x05,
    text: 'b',
    unshiftedCodepoint: 98,
  ),
  TerminalQuickKey(
    id: 'c',
    label: 'C',
    hidUsage: terminalUsbKeyboardPage | 0x06,
    text: 'c',
    unshiftedCodepoint: 99,
  ),
  TerminalQuickKey(
    id: 'd',
    label: 'D',
    hidUsage: terminalUsbKeyboardPage | 0x07,
    text: 'd',
    unshiftedCodepoint: 100,
  ),
  TerminalQuickKey(
    id: 'e',
    label: 'E',
    hidUsage: terminalUsbKeyboardPage | 0x08,
    text: 'e',
    unshiftedCodepoint: 101,
  ),
  TerminalQuickKey(
    id: 'f',
    label: 'F',
    hidUsage: terminalUsbKeyboardPage | 0x09,
    text: 'f',
    unshiftedCodepoint: 102,
  ),
  TerminalQuickKey(
    id: 'g',
    label: 'G',
    hidUsage: terminalUsbKeyboardPage | 0x0a,
    text: 'g',
    unshiftedCodepoint: 103,
  ),
  TerminalQuickKey(
    id: 'h',
    label: 'H',
    hidUsage: terminalUsbKeyboardPage | 0x0b,
    text: 'h',
    unshiftedCodepoint: 104,
  ),
  TerminalQuickKey(
    id: 'i',
    label: 'I',
    hidUsage: terminalUsbKeyboardPage | 0x0c,
    text: 'i',
    unshiftedCodepoint: 105,
  ),
  TerminalQuickKey(
    id: 'j',
    label: 'J',
    hidUsage: terminalUsbKeyboardPage | 0x0d,
    text: 'j',
    unshiftedCodepoint: 106,
  ),
  TerminalQuickKey(
    id: 'k',
    label: 'K',
    hidUsage: terminalUsbKeyboardPage | 0x0e,
    text: 'k',
    unshiftedCodepoint: 107,
  ),
  TerminalQuickKey(
    id: 'l',
    label: 'L',
    hidUsage: terminalUsbKeyboardPage | 0x0f,
    text: 'l',
    unshiftedCodepoint: 108,
  ),
  TerminalQuickKey(
    id: 'm',
    label: 'M',
    hidUsage: terminalUsbKeyboardPage | 0x10,
    text: 'm',
    unshiftedCodepoint: 109,
  ),
  TerminalQuickKey(
    id: 'n',
    label: 'N',
    hidUsage: terminalUsbKeyboardPage | 0x11,
    text: 'n',
    unshiftedCodepoint: 110,
  ),
  TerminalQuickKey(
    id: 'o',
    label: 'O',
    hidUsage: terminalUsbKeyboardPage | 0x12,
    text: 'o',
    unshiftedCodepoint: 111,
  ),
  TerminalQuickKey(
    id: 'p',
    label: 'P',
    hidUsage: terminalUsbKeyboardPage | 0x13,
    text: 'p',
    unshiftedCodepoint: 112,
  ),
  TerminalQuickKey(
    id: 'q',
    label: 'Q',
    hidUsage: terminalUsbKeyboardPage | 0x14,
    text: 'q',
    unshiftedCodepoint: 113,
  ),
  TerminalQuickKey(
    id: 'r',
    label: 'R',
    hidUsage: terminalUsbKeyboardPage | 0x15,
    text: 'r',
    unshiftedCodepoint: 114,
  ),
  TerminalQuickKey(
    id: 's',
    label: 'S',
    hidUsage: terminalUsbKeyboardPage | 0x16,
    text: 's',
    unshiftedCodepoint: 115,
  ),
  TerminalQuickKey(
    id: 't',
    label: 'T',
    hidUsage: terminalUsbKeyboardPage | 0x17,
    text: 't',
    unshiftedCodepoint: 116,
  ),
  TerminalQuickKey(
    id: 'u',
    label: 'U',
    hidUsage: terminalUsbKeyboardPage | 0x18,
    text: 'u',
    unshiftedCodepoint: 117,
  ),
  TerminalQuickKey(
    id: 'v',
    label: 'V',
    hidUsage: terminalUsbKeyboardPage | 0x19,
    text: 'v',
    unshiftedCodepoint: 118,
  ),
  TerminalQuickKey(
    id: 'w',
    label: 'W',
    hidUsage: terminalUsbKeyboardPage | 0x1a,
    text: 'w',
    unshiftedCodepoint: 119,
  ),
  TerminalQuickKey(
    id: 'x',
    label: 'X',
    hidUsage: terminalUsbKeyboardPage | 0x1b,
    text: 'x',
    unshiftedCodepoint: 120,
  ),
  TerminalQuickKey(
    id: 'y',
    label: 'Y',
    hidUsage: terminalUsbKeyboardPage | 0x1c,
    text: 'y',
    unshiftedCodepoint: 121,
  ),
  TerminalQuickKey(
    id: 'z',
    label: 'Z',
    hidUsage: terminalUsbKeyboardPage | 0x1d,
    text: 'z',
    unshiftedCodepoint: 122,
  ),
];

const _digitKeys = <TerminalQuickKey>[
  TerminalQuickKey(
    id: '1',
    label: '1',
    hidUsage: terminalUsbKeyboardPage | 0x1e,
    text: '1',
    unshiftedCodepoint: 49,
  ),
  TerminalQuickKey(
    id: '2',
    label: '2',
    hidUsage: terminalUsbKeyboardPage | 0x1f,
    text: '2',
    unshiftedCodepoint: 50,
  ),
  TerminalQuickKey(
    id: '3',
    label: '3',
    hidUsage: terminalUsbKeyboardPage | 0x20,
    text: '3',
    unshiftedCodepoint: 51,
  ),
  TerminalQuickKey(
    id: '4',
    label: '4',
    hidUsage: terminalUsbKeyboardPage | 0x21,
    text: '4',
    unshiftedCodepoint: 52,
  ),
  TerminalQuickKey(
    id: '5',
    label: '5',
    hidUsage: terminalUsbKeyboardPage | 0x22,
    text: '5',
    unshiftedCodepoint: 53,
  ),
  TerminalQuickKey(
    id: '6',
    label: '6',
    hidUsage: terminalUsbKeyboardPage | 0x23,
    text: '6',
    unshiftedCodepoint: 54,
  ),
  TerminalQuickKey(
    id: '7',
    label: '7',
    hidUsage: terminalUsbKeyboardPage | 0x24,
    text: '7',
    unshiftedCodepoint: 55,
  ),
  TerminalQuickKey(
    id: '8',
    label: '8',
    hidUsage: terminalUsbKeyboardPage | 0x25,
    text: '8',
    unshiftedCodepoint: 56,
  ),
  TerminalQuickKey(
    id: '9',
    label: '9',
    hidUsage: terminalUsbKeyboardPage | 0x26,
    text: '9',
    unshiftedCodepoint: 57,
  ),
  TerminalQuickKey(
    id: '0',
    label: '0',
    hidUsage: terminalUsbKeyboardPage | 0x27,
    text: '0',
    unshiftedCodepoint: 48,
  ),
];

const _functionKeys = <TerminalQuickKey>[
  TerminalQuickKey(
    id: 'f1',
    label: 'F1',
    hidUsage: terminalUsbKeyboardPage | 0x3a,
  ),
  TerminalQuickKey(
    id: 'f2',
    label: 'F2',
    hidUsage: terminalUsbKeyboardPage | 0x3b,
  ),
  TerminalQuickKey(
    id: 'f3',
    label: 'F3',
    hidUsage: terminalUsbKeyboardPage | 0x3c,
  ),
  TerminalQuickKey(
    id: 'f4',
    label: 'F4',
    hidUsage: terminalUsbKeyboardPage | 0x3d,
  ),
  TerminalQuickKey(
    id: 'f5',
    label: 'F5',
    hidUsage: terminalUsbKeyboardPage | 0x3e,
  ),
  TerminalQuickKey(
    id: 'f6',
    label: 'F6',
    hidUsage: terminalUsbKeyboardPage | 0x3f,
  ),
  TerminalQuickKey(
    id: 'f7',
    label: 'F7',
    hidUsage: terminalUsbKeyboardPage | 0x40,
  ),
  TerminalQuickKey(
    id: 'f8',
    label: 'F8',
    hidUsage: terminalUsbKeyboardPage | 0x41,
  ),
  TerminalQuickKey(
    id: 'f9',
    label: 'F9',
    hidUsage: terminalUsbKeyboardPage | 0x42,
  ),
  TerminalQuickKey(
    id: 'f10',
    label: 'F10',
    hidUsage: terminalUsbKeyboardPage | 0x43,
  ),
  TerminalQuickKey(
    id: 'f11',
    label: 'F11',
    hidUsage: terminalUsbKeyboardPage | 0x44,
  ),
  TerminalQuickKey(
    id: 'f12',
    label: 'F12',
    hidUsage: terminalUsbKeyboardPage | 0x45,
  ),
];

TerminalQuickKey? terminalQuickKeyById(String? id) {
  if (id == null) return null;
  for (final key in terminalQuickKeys) {
    if (key.id == id) return key;
  }
  return null;
}

final class TerminalQuickAction {
  const TerminalQuickAction({
    required this.id,
    required this.kind,
    this.keyId,
    this.modifiers = 0,
    this.text = '',
    this.sendEnter = false,
    this.label = '',
  });

  final String id;
  final TerminalQuickActionKind kind;
  final String? keyId;
  final int modifiers;
  final String text;
  final bool sendEnter;
  final String label;

  TerminalQuickKey? get key => terminalQuickKeyById(keyId);

  String get displayLabel {
    final custom = label.trim();
    if (custom.isNotEmpty) return custom;
    if (kind == TerminalQuickActionKind.text) {
      return '${text.trim().isEmpty ? 'Text' : text.trim()}${sendEnter ? '  ↵' : ''}';
    }
    final base = key?.label ?? 'Key';
    if (kind == TerminalQuickActionKind.key) return base;
    final parts = <String>[
      if (modifiers & terminalModifierControlBit != 0) 'Ctrl',
      if (modifiers & terminalModifierAltBit != 0) 'Alt',
      if (modifiers & terminalModifierShiftBit != 0) 'Shift',
      if (modifiers & terminalModifierSuperBit != 0) 'Super',
      base,
    ];
    return parts.join('+');
  }

  String get accessibilityLabel {
    final custom = label.trim();
    if (custom.isNotEmpty) return custom;
    if (kind == TerminalQuickActionKind.text) return displayLabel;
    final base = key?.accessibilityLabel ?? 'Key';
    if (kind == TerminalQuickActionKind.key) return base;
    final parts = <String>[
      if (modifiers & terminalModifierControlBit != 0) 'Control',
      if (modifiers & terminalModifierAltBit != 0) 'Alt',
      if (modifiers & terminalModifierShiftBit != 0) 'Shift',
      if (modifiers & terminalModifierSuperBit != 0) 'Super',
      base,
    ];
    return parts.join(' + ');
  }

  bool get isValid {
    return switch (kind) {
      TerminalQuickActionKind.key => key != null,
      TerminalQuickActionKind.chord =>
        key?.supportsChord == true && modifiers != 0,
      TerminalQuickActionKind.text => text.isNotEmpty,
    };
  }

  TerminalQuickAction copyWith({
    String? id,
    TerminalQuickActionKind? kind,
    String? keyId,
    int? modifiers,
    String? text,
    bool? sendEnter,
    String? label,
  }) => TerminalQuickAction(
    id: id ?? this.id,
    kind: kind ?? this.kind,
    keyId: keyId ?? this.keyId,
    modifiers: modifiers ?? this.modifiers,
    text: text ?? this.text,
    sendEnter: sendEnter ?? this.sendEnter,
    label: label ?? this.label,
  );

  Map<String, Object?> toJson() => {
    'id': id,
    'kind': kind.name,
    'key': ?keyId,
    if (modifiers != 0) 'modifiers': modifiers,
    if (text.isNotEmpty) 'text': text,
    if (sendEnter) 'sendEnter': true,
    if (label.trim().isNotEmpty) 'label': label.trim(),
  };

  static TerminalQuickAction? fromJson(Map<String, dynamic> json) {
    final id = json['id'];
    final kindName = json['kind'];
    if (id is! String || id.isEmpty || kindName is! String) return null;
    final kind = TerminalQuickActionKind.values
        .where((value) => value.name == kindName)
        .firstOrNull;
    if (kind == null) return null;
    final action = TerminalQuickAction(
      id: id,
      kind: kind,
      keyId: json['key'] is String ? json['key'] as String : null,
      modifiers: json['modifiers'] is int ? json['modifiers'] as int : 0,
      text: json['text'] is String ? json['text'] as String : '',
      sendEnter: json['sendEnter'] == true,
      label: json['label'] is String ? json['label'] as String : '',
    );
    return action.isValid ? action : null;
  }
}

List<TerminalQuickAction> terminalQuickKeysInLayoutOrder({
  required List<TerminalQuickAction> actions,
  required List<String> layoutIds,
}) {
  final byId = {for (final action in actions) action.id: action};
  final seen = <String>{};
  return List.unmodifiable([
    for (final id in layoutIds)
      if (byId[id] case final action? when seen.add(id)) action,
    for (final action in actions)
      if (seen.add(action.id)) action,
  ]);
}

List<String> swapTerminalQuickKeyLayout({
  required List<String> layoutIds,
  required String draggedId,
  required String targetId,
}) {
  if (draggedId == targetId ||
      !layoutIds.contains(draggedId) ||
      !layoutIds.contains(targetId)) {
    return List.unmodifiable(layoutIds);
  }
  final swapped = [...layoutIds];
  final draggedIndex = swapped.indexOf(draggedId);
  final targetIndex = swapped.indexOf(targetId);
  swapped[draggedIndex] = targetId;
  swapped[targetIndex] = draggedId;
  return List.unmodifiable(swapped);
}

const defaultTerminalQuickActions = <TerminalQuickAction>[
  TerminalQuickAction(
    id: 'default-escape',
    kind: TerminalQuickActionKind.key,
    keyId: 'escape',
  ),
  TerminalQuickAction(
    id: 'default-tab',
    kind: TerminalQuickActionKind.key,
    keyId: 'tab',
  ),
  TerminalQuickAction(
    id: 'default-control',
    kind: TerminalQuickActionKind.key,
    keyId: 'control',
  ),
  TerminalQuickAction(
    id: 'default-alt',
    kind: TerminalQuickActionKind.key,
    keyId: 'alt',
  ),
  TerminalQuickAction(
    id: 'default-slash',
    kind: TerminalQuickActionKind.key,
    keyId: 'slash',
  ),
  TerminalQuickAction(
    id: 'default-pipe',
    kind: TerminalQuickActionKind.key,
    keyId: 'pipe',
  ),
  TerminalQuickAction(
    id: 'default-minus',
    kind: TerminalQuickActionKind.key,
    keyId: 'minus',
  ),
  TerminalQuickAction(
    id: 'default-tilde',
    kind: TerminalQuickActionKind.key,
    keyId: 'tilde',
  ),
  TerminalQuickAction(
    id: 'default-backslash',
    kind: TerminalQuickActionKind.key,
    keyId: 'backslash',
  ),
  TerminalQuickAction(
    id: 'default-left',
    kind: TerminalQuickActionKind.key,
    keyId: 'left',
  ),
  TerminalQuickAction(
    id: 'default-down',
    kind: TerminalQuickActionKind.key,
    keyId: 'down',
  ),
  TerminalQuickAction(
    id: 'default-up',
    kind: TerminalQuickActionKind.key,
    keyId: 'up',
  ),
  TerminalQuickAction(
    id: 'default-right',
    kind: TerminalQuickActionKind.key,
    keyId: 'right',
  ),
  TerminalQuickAction(
    id: 'default-home',
    kind: TerminalQuickActionKind.key,
    keyId: 'home',
  ),
  TerminalQuickAction(
    id: 'default-end',
    kind: TerminalQuickActionKind.key,
    keyId: 'end',
  ),
  TerminalQuickAction(
    id: 'default-page-up',
    kind: TerminalQuickActionKind.key,
    keyId: 'page-up',
  ),
  TerminalQuickAction(
    id: 'default-page-down',
    kind: TerminalQuickActionKind.key,
    keyId: 'page-down',
  ),
  TerminalQuickAction(
    id: 'default-delete',
    kind: TerminalQuickActionKind.key,
    keyId: 'delete',
  ),
];
