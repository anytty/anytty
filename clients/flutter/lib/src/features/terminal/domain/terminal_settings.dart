import 'terminal_metrics.dart';

enum TerminalKeyboardMode { automatic, resize, shift }

TerminalKeyboardMode resolveTerminalKeyboardMode(
  TerminalKeyboardMode configured, {
  required bool alternateScreen,
}) => switch (configured) {
  TerminalKeyboardMode.automatic =>
    alternateScreen ? TerminalKeyboardMode.resize : TerminalKeyboardMode.shift,
  TerminalKeyboardMode.resize => TerminalKeyboardMode.resize,
  TerminalKeyboardMode.shift => TerminalKeyboardMode.shift,
};

final class TerminalTheme {
  const TerminalTheme({
    required this.id,
    required this.label,
    required this.dark,
    required this.background,
    required this.foreground,
    required this.cursor,
    required this.selection,
    required this.ansi,
  });

  final String id;
  final String label;
  final bool dark;
  final int background;
  final int foreground;
  final int cursor;
  final int selection;
  final List<int> ansi;
}

const terminalThemes = <TerminalTheme>[
  TerminalTheme(
    id: 'anytty-dark',
    label: 'AnyTTY Dark',
    dark: true,
    background: 0x0c0c0c,
    foreground: 0xf4f4f5,
    cursor: 0xd4d4d8,
    selection: 0x3f3f46,
    ansi: [
      0x18181b,
      0xef4444,
      0x22c55e,
      0xeab308,
      0x3b82f6,
      0xd946ef,
      0x06b6d4,
      0xf4f4f5,
      0x71717a,
      0xf87171,
      0x4ade80,
      0xfde047,
      0x60a5fa,
      0xe879f9,
      0x22d3ee,
      0xfafafa,
    ],
  ),
  TerminalTheme(
    id: 'tokyo-night',
    label: 'Tokyo Night',
    dark: true,
    background: 0x1a1b26,
    foreground: 0xa9b1d6,
    cursor: 0xc0caf5,
    selection: 0x33467c,
    ansi: [
      0x15161e,
      0xf7768e,
      0x9ece6a,
      0xe0af68,
      0x7aa2f7,
      0xbb9af7,
      0x7dcfff,
      0xa9b1d6,
      0x414868,
      0xf7768e,
      0x9ece6a,
      0xe0af68,
      0x7aa2f7,
      0xbb9af7,
      0x7dcfff,
      0xc0caf5,
    ],
  ),
  TerminalTheme(
    id: 'dracula',
    label: 'Dracula',
    dark: true,
    background: 0x282a36,
    foreground: 0xf8f8f2,
    cursor: 0xf8f8f2,
    selection: 0x44475a,
    ansi: [
      0x21222c,
      0xff5555,
      0x50fa7b,
      0xf1fa8c,
      0xbd93f9,
      0xff79c6,
      0x8be9fd,
      0xf8f8f2,
      0x6272a4,
      0xff6e6e,
      0x69ff94,
      0xffffa5,
      0xd6acff,
      0xff92df,
      0xa4ffff,
      0xffffff,
    ],
  ),
  TerminalTheme(
    id: 'one-dark',
    label: 'One Dark',
    dark: true,
    background: 0x282c34,
    foreground: 0xabb2bf,
    cursor: 0x528bff,
    selection: 0x3e4451,
    ansi: [
      0x1e2127,
      0xe06c75,
      0x98c379,
      0xd19a66,
      0x61afef,
      0xc678dd,
      0x56b6c2,
      0xabb2bf,
      0x5c6370,
      0xe06c75,
      0x98c379,
      0xd19a66,
      0x61afef,
      0xc678dd,
      0x56b6c2,
      0xffffff,
    ],
  ),
  TerminalTheme(
    id: 'catppuccin-mocha',
    label: 'Catppuccin Mocha',
    dark: true,
    background: 0x1e1e2e,
    foreground: 0xcdd6f4,
    cursor: 0xf5e0dc,
    selection: 0x45475a,
    ansi: [
      0x45475a,
      0xf38ba8,
      0xa6e3a1,
      0xf9e2af,
      0x89b4fa,
      0xf5c2e7,
      0x94e2d5,
      0xbac2de,
      0x585b70,
      0xf38ba8,
      0xa6e3a1,
      0xf9e2af,
      0x89b4fa,
      0xf5c2e7,
      0x94e2d5,
      0xa6adc8,
    ],
  ),
  TerminalTheme(
    id: 'solarized-dark',
    label: 'Solarized Dark',
    dark: true,
    background: 0x002b36,
    foreground: 0x839496,
    cursor: 0x839496,
    selection: 0x073642,
    ansi: [
      0x073642,
      0xdc322f,
      0x859900,
      0xb58900,
      0x268bd2,
      0xd33682,
      0x2aa198,
      0xeee8d5,
      0x586e75,
      0xcb4b16,
      0x586e75,
      0x657b83,
      0x839496,
      0x6c71c4,
      0x93a1a1,
      0xfdf6e3,
    ],
  ),
  TerminalTheme(
    id: 'nord',
    label: 'Nord',
    dark: true,
    background: 0x2e3440,
    foreground: 0xd8dee9,
    cursor: 0xd8dee9,
    selection: 0x434c5e,
    ansi: [
      0x3b4252,
      0xbf616a,
      0xa3be8c,
      0xebcb8b,
      0x81a1c1,
      0xb48ead,
      0x88c0d0,
      0xe5e9f0,
      0x4c566a,
      0xbf616a,
      0xa3be8c,
      0xebcb8b,
      0x81a1c1,
      0xb48ead,
      0x8fbcbb,
      0xeceff4,
    ],
  ),
  TerminalTheme(
    id: 'gruvbox-dark',
    label: 'Gruvbox Dark',
    dark: true,
    background: 0x282828,
    foreground: 0xebdbb2,
    cursor: 0xebdbb2,
    selection: 0x504945,
    ansi: [
      0x282828,
      0xcc241d,
      0x98971a,
      0xd79921,
      0x458588,
      0xb16286,
      0x689d6a,
      0xa89984,
      0x928374,
      0xfb4934,
      0xb8bb26,
      0xfabd2f,
      0x83a598,
      0xd3869b,
      0x8ec07c,
      0xebdbb2,
    ],
  ),
  TerminalTheme(
    id: 'github-dark',
    label: 'GitHub Dark',
    dark: true,
    background: 0x0d1117,
    foreground: 0xc9d1d9,
    cursor: 0x58a6ff,
    selection: 0x264f78,
    ansi: [
      0x484f58,
      0xff7b72,
      0x3fb950,
      0xd29922,
      0x58a6ff,
      0xbc8cff,
      0x39c5cf,
      0xb1bac4,
      0x6e7681,
      0xffa198,
      0x56d364,
      0xe3b341,
      0x79c0ff,
      0xd2a8ff,
      0x56d4dd,
      0xf0f6fc,
    ],
  ),
  TerminalTheme(
    id: 'github-light',
    label: 'GitHub Light',
    dark: false,
    background: 0xffffff,
    foreground: 0x24292f,
    cursor: 0x044289,
    selection: 0xb6d4fe,
    ansi: [
      0x24292f,
      0xcf222e,
      0x116329,
      0x4d2d00,
      0x0969da,
      0x8250df,
      0x1b7c83,
      0x6e7781,
      0x57606a,
      0xa40e26,
      0x1a7f37,
      0x633c01,
      0x218bff,
      0xa475f9,
      0x3192aa,
      0x8c959f,
    ],
  ),
  TerminalTheme(
    id: 'solarized-light',
    label: 'Solarized Light',
    dark: false,
    background: 0xfdf6e3,
    foreground: 0x657b83,
    cursor: 0x586e75,
    selection: 0xeee8d5,
    ansi: [
      0x073642,
      0xdc322f,
      0x859900,
      0xb58900,
      0x268bd2,
      0xd33682,
      0x2aa198,
      0xeee8d5,
      0x002b36,
      0xcb4b16,
      0x586e75,
      0x657b83,
      0x839496,
      0x6c71c4,
      0x93a1a1,
      0xfdf6e3,
    ],
  ),
  TerminalTheme(
    id: 'catppuccin-latte',
    label: 'Catppuccin Latte',
    dark: false,
    background: 0xeff1f5,
    foreground: 0x4c4f69,
    cursor: 0xdc8a78,
    selection: 0xccd0da,
    ansi: [
      0x5c5f77,
      0xd20f39,
      0x40a02b,
      0xdf8e1d,
      0x1e66f5,
      0xea76cb,
      0x179299,
      0xacb0be,
      0x6c6f85,
      0xd20f39,
      0x40a02b,
      0xdf8e1d,
      0x1e66f5,
      0xea76cb,
      0x179299,
      0xbcc0cc,
    ],
  ),
  TerminalTheme(
    id: 'one-light',
    label: 'One Light',
    dark: false,
    background: 0xfafafa,
    foreground: 0x383a42,
    cursor: 0x526fff,
    selection: 0xbfceff,
    ansi: [
      0x383a42,
      0xe45649,
      0x50a14f,
      0xc18401,
      0x4078f2,
      0xa626a4,
      0x0184bc,
      0xa0a1a7,
      0x696c77,
      0xe45649,
      0x50a14f,
      0xc18401,
      0x4078f2,
      0xa626a4,
      0x0184bc,
      0xfafafa,
    ],
  ),
  TerminalTheme(
    id: 'nord-light',
    label: 'Nord Light',
    dark: false,
    background: 0xeceff4,
    foreground: 0x2e3440,
    cursor: 0x2e3440,
    selection: 0xd8dee9,
    ansi: [
      0x2e3440,
      0xbf616a,
      0xa3be8c,
      0xebcb8b,
      0x81a1c1,
      0xb48ead,
      0x88c0d0,
      0xd8dee9,
      0x4c566a,
      0xbf616a,
      0xa3be8c,
      0xebcb8b,
      0x81a1c1,
      0xb48ead,
      0x8fbcbb,
      0xeceff4,
    ],
  ),
  TerminalTheme(
    id: 'gruvbox-light',
    label: 'Gruvbox Light',
    dark: false,
    background: 0xfbf1c7,
    foreground: 0x3c3836,
    cursor: 0x3c3836,
    selection: 0xd5c4a1,
    ansi: [
      0x3c3836,
      0xcc241d,
      0x98971a,
      0xd79921,
      0x458588,
      0xb16286,
      0x689d6a,
      0x7c6f64,
      0x928374,
      0x9d0006,
      0x79740e,
      0xb57614,
      0x076678,
      0x8f3f71,
      0x427b58,
      0x3c3836,
    ],
  ),
  TerminalTheme(
    id: 'rose-pine-dawn',
    label: 'Rose Pine Dawn',
    dark: false,
    background: 0xfaf4ed,
    foreground: 0x575279,
    cursor: 0x575279,
    selection: 0xdfdad9,
    ansi: [
      0x575279,
      0xb4637a,
      0x286983,
      0xea9d34,
      0x56949f,
      0x907aa9,
      0xd7827e,
      0xf2e9e1,
      0x797593,
      0xb4637a,
      0x286983,
      0xea9d34,
      0x56949f,
      0x907aa9,
      0xd7827e,
      0xfaf4ed,
    ],
  ),
  TerminalTheme(
    id: 'everforest-light',
    label: 'Everforest Light',
    dark: false,
    background: 0xf3ead3,
    foreground: 0x5c6a72,
    cursor: 0x5c6a72,
    selection: 0xe0dcc7,
    ansi: [
      0x5c6a72,
      0xf85552,
      0x8da101,
      0xdfa000,
      0x3a94c5,
      0xdf69ba,
      0x35a77c,
      0xdfddc8,
      0x829181,
      0xf85552,
      0x8da101,
      0xdfa000,
      0x3a94c5,
      0xdf69ba,
      0x35a77c,
      0xf3ead3,
    ],
  ),
];

final defaultTerminalTheme = terminalThemes.first;

TerminalTheme resolveTerminalTheme(String id) => terminalThemes.firstWhere(
  (theme) => theme.id == id,
  orElse: () => defaultTerminalTheme,
);

final class TerminalMomentumProfile {
  const TerminalMomentumProfile({
    required this.enabled,
    required this.deceleration,
    required this.minimumVelocity,
  });

  final bool enabled;
  final double deceleration;
  final double minimumVelocity;
}

final class TerminalSettings {
  const TerminalSettings({
    this.fontSize = 14,
    this.fontFamily = 'JetBrainsMonoNerd',
    this.themeId = 'anytty-dark',
    this.keyboardMode = TerminalKeyboardMode.automatic,
    this.scrollInertia = 60,
    this.historyPrefetchThresholdRows = 30,
    this.cursorBlink = true,
    this.autoAcquireResizeOwner = false,
  });

  final int fontSize;
  final String fontFamily;
  final String themeId;
  final TerminalKeyboardMode keyboardMode;
  final int scrollInertia;
  final int historyPrefetchThresholdRows;
  final bool cursorBlink;
  final bool autoAcquireResizeOwner;

  TerminalCellMetrics get metrics =>
      TerminalCellMetrics.fromFontSize(fontSize.toDouble());
  TerminalTheme get theme => resolveTerminalTheme(themeId);

  TerminalSettings copyWith({
    int? fontSize,
    String? fontFamily,
    String? themeId,
    TerminalKeyboardMode? keyboardMode,
    int? scrollInertia,
    int? historyPrefetchThresholdRows,
    bool? cursorBlink,
    bool? autoAcquireResizeOwner,
  }) => TerminalSettings(
    fontSize: fontSize ?? this.fontSize,
    fontFamily: fontFamily ?? this.fontFamily,
    themeId: themeId ?? this.themeId,
    keyboardMode: keyboardMode ?? this.keyboardMode,
    scrollInertia: scrollInertia ?? this.scrollInertia,
    historyPrefetchThresholdRows:
        historyPrefetchThresholdRows ?? this.historyPrefetchThresholdRows,
    cursorBlink: cursorBlink ?? this.cursorBlink,
    autoAcquireResizeOwner:
        autoAcquireResizeOwner ?? this.autoAcquireResizeOwner,
  ).normalized();

  TerminalSettings normalized() => TerminalSettings(
    fontSize: fontSize.clamp(8, 32),
    fontFamily: terminalFontFamilies.contains(fontFamily)
        ? fontFamily
        : 'JetBrainsMonoNerd',
    themeId: terminalThemes.any((theme) => theme.id == themeId)
        ? themeId
        : 'anytty-dark',
    keyboardMode: keyboardMode,
    scrollInertia: scrollInertia.clamp(0, 100),
    historyPrefetchThresholdRows: historyPrefetchThresholdRows.clamp(0, 1000),
    cursorBlink: cursorBlink,
    autoAcquireResizeOwner: autoAcquireResizeOwner,
  );

  Map<String, Object> toJson() => {
    'fontSize': fontSize,
    'fontFamily': fontFamily,
    'themeId': themeId,
    'keyboardMode': keyboardMode.name,
    'scrollInertia': scrollInertia,
    'historyPrefetchThresholdRows': historyPrefetchThresholdRows,
    'cursorBlink': cursorBlink,
    'autoAcquireResizeOwner': autoAcquireResizeOwner,
  };

  factory TerminalSettings.fromJson(Map<String, Object?> json) {
    final inertia = switch (json['scrollInertia']) {
      'off' => 0,
      'short' => 25,
      'medium' => 60,
      'long' => 100,
      final num value => value.round(),
      _ => 60,
    };
    final keyboardMode = switch (json['keyboardMode']) {
      'resize' => TerminalKeyboardMode.resize,
      'shift' || 'cover' || 'overlay' => TerminalKeyboardMode.shift,
      _ => TerminalKeyboardMode.automatic,
    };
    return TerminalSettings(
      fontSize: (json['fontSize'] as num?)?.round() ?? 14,
      fontFamily: json['fontFamily'] as String? ?? 'JetBrainsMonoNerd',
      themeId: json['themeId'] as String? ?? 'anytty-dark',
      keyboardMode: keyboardMode,
      scrollInertia: inertia,
      historyPrefetchThresholdRows:
          (json['historyPrefetchThresholdRows'] as num?)?.round() ??
          (json['scrollbackPrefetchThresholdRows'] as num?)?.round() ??
          30,
      cursorBlink: json['cursorBlink'] as bool? ?? true,
      autoAcquireResizeOwner: json['autoAcquireResizeOwner'] as bool? ?? false,
    ).normalized();
  }

  @override
  bool operator ==(Object other) =>
      other is TerminalSettings &&
      other.fontSize == fontSize &&
      other.fontFamily == fontFamily &&
      other.themeId == themeId &&
      other.keyboardMode == keyboardMode &&
      other.scrollInertia == scrollInertia &&
      other.historyPrefetchThresholdRows == historyPrefetchThresholdRows &&
      other.cursorBlink == cursorBlink &&
      other.autoAcquireResizeOwner == autoAcquireResizeOwner;

  @override
  int get hashCode => Object.hash(
    fontSize,
    fontFamily,
    themeId,
    keyboardMode,
    scrollInertia,
    historyPrefetchThresholdRows,
    cursorBlink,
    autoAcquireResizeOwner,
  );
}

const terminalFontFamilies = <String>[
  'JetBrainsMonoNerd',
  'FiraCodeNerd',
  'CascadiaCodeNerd',
  'HackNerd',
  'IosevkaNerd',
  'monospace',
];

String terminalFontLabel(String family) => switch (family) {
  'JetBrainsMonoNerd' => 'JetBrains Mono',
  'FiraCodeNerd' => 'Fira Code',
  'CascadiaCodeNerd' => 'Cascadia Code',
  'HackNerd' => 'Hack',
  'IosevkaNerd' => 'Iosevka',
  'monospace' => 'System Mono',
  _ => family,
};

const defaultTerminalSettings = TerminalSettings();

TerminalMomentumProfile resolveTerminalMomentumProfile(int inertia) {
  final value = inertia.clamp(0, 100);
  if (value == 0) {
    return const TerminalMomentumProfile(
      enabled: false,
      deceleration: 0,
      minimumVelocity: 0,
    );
  }
  const anchors = <({int value, double deceleration, double minimumVelocity})>[
    (value: 1, deceleration: 0.9, minimumVelocity: 80),
    (value: 25, deceleration: 0.95, minimumVelocity: 60),
    (value: 60, deceleration: 0.985, minimumVelocity: 20),
    (value: 100, deceleration: 0.99, minimumVelocity: 10),
  ];
  var upperIndex = anchors.indexWhere((anchor) => anchor.value >= value);
  if (upperIndex < 0) upperIndex = anchors.length - 1;
  final upper = anchors[upperIndex];
  final lower = anchors[upperIndex == 0 ? 0 : upperIndex - 1];
  final range = upper.value - lower.value;
  final ratio = range == 0 ? 0.0 : (value - lower.value) / range;
  return TerminalMomentumProfile(
    enabled: true,
    deceleration:
        lower.deceleration + (upper.deceleration - lower.deceleration) * ratio,
    minimumVelocity:
        lower.minimumVelocity +
        (upper.minimumVelocity - lower.minimumVelocity) * ratio,
  );
}
