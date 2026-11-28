part of 'terminal_workspace_screen.dart';

final class _TerminalSettingsSheet extends StatefulWidget {
  const _TerminalSettingsSheet({
    required this.initialSettings,
    required this.onChanged,
    required this.keyboardMode,
    required this.onKeyboardModeChanged,
  });

  final TerminalSettings initialSettings;
  final Future<void> Function(TerminalSettings) onChanged;
  final TerminalKeyboardMode keyboardMode;
  final Future<void> Function(TerminalKeyboardMode) onKeyboardModeChanged;

  @override
  State<_TerminalSettingsSheet> createState() => _TerminalSettingsSheetState();
}

final class _TerminalSettingsSheetState extends State<_TerminalSettingsSheet> {
  late TerminalSettings _settings;

  @override
  void initState() {
    super.initState();
    _settings = widget.initialSettings.normalized();
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 24),
              children: [
                const _SettingsBandTitle(label: 'Input'),
                _SettingsRow(
                  label: 'Keyboard',
                  child: _TerminalKeyboardModeControl(
                    value: widget.keyboardMode,
                    onChanged: (mode) =>
                        unawaited(widget.onKeyboardModeChanged(mode)),
                  ),
                ),
                const _SettingsBandTitle(label: 'Appearance'),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 12, 12, 6),
                  child: _TerminalSettingsPreview(settings: _settings),
                ),
                _SettingsRow(
                  label: 'Font size',
                  child: _FontSizeStepper(
                    value: _settings.fontSize,
                    onChanged: (value) =>
                        _commit(_settings.copyWith(fontSize: value)),
                  ),
                ),
                _SettingsRow(
                  label: 'Font',
                  child: DropdownButton<String>(
                    value: _settings.fontFamily,
                    underline: const SizedBox.shrink(),
                    borderRadius: BorderRadius.circular(6),
                    onChanged: (value) {
                      if (value != null) {
                        unawaited(
                          _commit(_settings.copyWith(fontFamily: value)),
                        );
                      }
                    },
                    items: [
                      for (final family in terminalFontFamilies)
                        DropdownMenuItem(
                          value: family,
                          child: Text(
                            terminalFontLabel(family),
                            style: TextStyle(
                              fontFamily: family,
                              fontFamilyFallback: terminalFontFamilyFallback,
                              fontSize: 13,
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
                SwitchListTile.adaptive(
                  title: const Text('Cursor blink'),
                  value: _settings.cursorBlink,
                  onChanged: (value) =>
                      _commit(_settings.copyWith(cursorBlink: value)),
                ),
                SwitchListTile.adaptive(
                  title: const Text('Automatically own terminal size'),
                  value: _settings.autoAcquireResizeOwner,
                  onChanged: (value) => _commit(
                    _settings.copyWith(autoAcquireResizeOwner: value),
                  ),
                ),
                const _SettingsBandTitle(label: 'Touch and history'),
                _SettingsSlider(
                  label: 'Scroll inertia',
                  value: _settings.scrollInertia,
                  maximum: 100,
                  onChanged: (value) => setState(
                    () => _settings = _settings.copyWith(scrollInertia: value),
                  ),
                  onChangeEnd: (value) =>
                      _commit(_settings.copyWith(scrollInertia: value)),
                ),
                _SettingsSlider(
                  label: 'API page prefetch distance',
                  value: _settings.historyPrefetchThresholdRows,
                  maximum: 200,
                  onChanged: (value) => setState(
                    () => _settings = _settings.copyWith(
                      historyPrefetchThresholdRows: value,
                    ),
                  ),
                  onChangeEnd: (value) => _commit(
                    _settings.copyWith(historyPrefetchThresholdRows: value),
                  ),
                ),
                const _SettingsBandTitle(label: 'Terminal theme'),
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      final columns = constraints.maxWidth >= 560 ? 3 : 2;
                      return GridView.builder(
                        shrinkWrap: true,
                        physics: const NeverScrollableScrollPhysics(),
                        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                          crossAxisCount: columns,
                          mainAxisExtent: 56,
                          crossAxisSpacing: 8,
                          mainAxisSpacing: 8,
                        ),
                        itemCount: terminalThemes.length,
                        itemBuilder: (context, index) {
                          final theme = terminalThemes[index];
                          final selected = theme.id == _settings.themeId;
                          void selectTheme() => unawaited(
                            _commit(_settings.copyWith(themeId: theme.id)),
                          );

                          return Semantics(
                            button: true,
                            selected: selected,
                            label: '${theme.label} theme palette preview',
                            onTap: selectTheme,
                            child: ExcludeSemantics(
                              child: OutlinedButton(
                                onPressed: selectTheme,
                                style: OutlinedButton.styleFrom(
                                  alignment: Alignment.centerLeft,
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 10,
                                  ),
                                  side: BorderSide(
                                    color: selected
                                        ? Theme.of(context).colorScheme.primary
                                        : Theme.of(context).dividerColor,
                                    width: selected ? 2 : 1,
                                  ),
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                ),
                                child: Row(
                                  children: [
                                    _TerminalThemeSwatch(theme: theme),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      child: Text(
                                        theme.label,
                                        maxLines: 2,
                                        overflow: TextOverflow.ellipsis,
                                        style: const TextStyle(fontSize: 11),
                                      ),
                                    ),
                                    if (selected)
                                      const Icon(Icons.check_rounded, size: 16),
                                  ],
                                ),
                              ),
                            ),
                          );
                        },
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _commit(TerminalSettings next) async {
    final previous = _settings;
    setState(() => _settings = next);
    try {
      await widget.onChanged(next);
    } catch (error) {
      if (!mounted) return;
      setState(() => _settings = previous);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not save terminal settings: $error')),
      );
    }
  }
}

final class _TerminalSettingsPreview extends StatelessWidget {
  const _TerminalSettingsPreview({required this.settings});

  final TerminalSettings settings;

  @override
  Widget build(BuildContext context) {
    final theme = settings.theme;
    final background = _terminalThemeColor(theme.background);
    final foreground = _terminalThemeColor(theme.foreground);
    final muted = _terminalThemeColor(theme.ansi[8]);
    final accent = _terminalThemeColor(theme.ansi[2]);
    final prompt = _terminalThemeColor(theme.ansi[3]);
    final fontSize = settings.fontSize.toDouble().clamp(10, 20).toDouble();
    final lineStyle = TextStyle(
      color: foreground,
      fontFamily: settings.fontFamily,
      fontFamilyFallback: terminalFontFamilyFallback,
      fontSize: fontSize,
      height: 1.25,
    );
    return Semantics(
      readOnly: true,
      label:
          'Terminal settings preview, ${theme.label}, '
          '${terminalFontLabel(settings.fontFamily)}, '
          '${settings.fontSize} point',
      child: ExcludeSemantics(
        child: Container(
          height: 104,
          clipBehavior: Clip.antiAlias,
          decoration: BoxDecoration(
            color: background,
            border: Border.all(color: muted.withValues(alpha: 0.5)),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 6,
                ),
                child: Row(
                  children: [
                    Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: accent,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 7),
                    Expanded(
                      child: Text(
                        '${theme.label}  ·  ${terminalFontLabel(settings.fontFamily)}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: muted,
                          fontFamily: settings.fontFamily,
                          fontFamilyFallback: terminalFontFamilyFallback,
                          fontSize: 10,
                        ),
                      ),
                    ),
                    Text(
                      '${settings.fontSize} pt',
                      style: TextStyle(
                        color: muted,
                        fontFamily: settings.fontFamily,
                        fontFamilyFallback: terminalFontFamilyFallback,
                        fontSize: 10,
                      ),
                    ),
                  ],
                ),
              ),
              Divider(height: 1, color: muted.withValues(alpha: 0.35)),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(10, 7, 10, 6),
                  child: RichText(
                    maxLines: 2,
                    overflow: TextOverflow.clip,
                    text: TextSpan(
                      style: lineStyle,
                      children: [
                        TextSpan(
                          text: 'anytty',
                          style: lineStyle.copyWith(color: accent),
                        ),
                        const TextSpan(text: ' in '),
                        TextSpan(
                          text: '~/workspace\n',
                          style: lineStyle.copyWith(
                            color: _terminalThemeColor(theme.ansi[6]),
                          ),
                        ),
                        TextSpan(
                          text: r'$ ',
                          style: lineStyle.copyWith(color: prompt),
                        ),
                        const TextSpan(text: 'git status --short'),
                        if (settings.cursorBlink)
                          TextSpan(
                            text: ' ',
                            style: lineStyle.copyWith(
                              backgroundColor: _terminalThemeColor(
                                theme.cursor,
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _SettingsBandTitle extends StatelessWidget {
  const _SettingsBandTitle({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Container(
    color: Theme.of(context).colorScheme.surfaceContainerLow,
    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
    child: Text(
      label.toUpperCase(),
      style: TextStyle(
        color: Theme.of(context).colorScheme.onSurfaceVariant,
        fontSize: 11,
        fontWeight: FontWeight.w600,
      ),
    ),
  );
}

final class _SettingsRow extends StatelessWidget {
  const _SettingsRow({required this.label, required this.child});

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) => Container(
    constraints: const BoxConstraints(minHeight: 56),
    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 6),
    decoration: BoxDecoration(
      border: Border(bottom: BorderSide(color: Theme.of(context).dividerColor)),
    ),
    child: Row(
      children: [
        Expanded(child: Text(label)),
        const SizedBox(width: 12),
        child,
      ],
    ),
  );
}

final class _FontSizeStepper extends StatelessWidget {
  const _FontSizeStepper({required this.value, required this.onChanged});

  final int value;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      IconButton(
        tooltip: 'Decrease font size',
        onPressed: value > 8 ? () => onChanged(value - 1) : null,
        icon: const Icon(Icons.remove_rounded),
      ),
      SizedBox(
        width: 42,
        child: Text(
          '$value',
          textAlign: TextAlign.center,
          style: const TextStyle(fontFamily: 'JetBrainsMonoNerd', fontSize: 12),
        ),
      ),
      IconButton(
        tooltip: 'Increase font size',
        onPressed: value < 32 ? () => onChanged(value + 1) : null,
        icon: const Icon(Icons.add_rounded),
      ),
    ],
  );
}

final class _SettingsSlider extends StatelessWidget {
  const _SettingsSlider({
    required this.label,
    required this.value,
    required this.maximum,
    required this.onChanged,
    required this.onChangeEnd,
  });

  final String label;
  final int value;
  final int maximum;
  final ValueChanged<int> onChanged;
  final ValueChanged<int> onChangeEnd;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(20, 10, 12, 8),
    child: Column(
      children: [
        Row(
          children: [
            Expanded(child: Text(label)),
            Text(
              '$value',
              style: const TextStyle(
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 12,
              ),
            ),
          ],
        ),
        Slider(
          value: value.toDouble(),
          max: maximum.toDouble(),
          divisions: maximum,
          onChanged: (value) => onChanged(value.round()),
          onChangeEnd: (value) => onChangeEnd(value.round()),
        ),
      ],
    ),
  );
}

final class _TerminalThemeSwatch extends StatelessWidget {
  const _TerminalThemeSwatch({required this.theme});

  final TerminalTheme theme;

  @override
  Widget build(BuildContext context) {
    final colors = [
      theme.background,
      theme.foreground,
      theme.ansi[1],
      theme.ansi[2],
      theme.ansi[4],
    ];
    return Container(
      width: 32,
      height: 32,
      padding: const EdgeInsets.all(1),
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Row(
        children: [
          for (final color in colors)
            SizedBox(
              width: 6,
              height: 30,
              child: ColoredBox(color: _terminalThemeColor(color)),
            ),
        ],
      ),
    );
  }
}

Color _terminalThemeColor(int rgb) => Color(0xff000000 | rgb);
