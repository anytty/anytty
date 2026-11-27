part of 'terminal_workspace_screen.dart';

final class _TerminalToolsSheet extends StatelessWidget {
  const _TerminalToolsSheet({
    required this.terminalState,
    required this.terminalSize,
    required this.resizeControl,
    required this.inputEnabled,
    required this.historyActive,
    required this.historyActionEnabled,
    required this.keyboardMode,
    required this.searchActive,
    required this.resizeAvailable,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
    required this.onKeyboardModeChanged,
    required this.onClose,
    required this.onSelected,
  });

  final TerminalState terminalState;
  final TerminalSize terminalSize;
  final ResizeControl resizeControl;
  final bool inputEnabled;
  final bool historyActive;
  final bool historyActionEnabled;
  final TerminalKeyboardMode keyboardMode;
  final bool searchActive;
  final bool resizeAvailable;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;
  final ValueChanged<TerminalKeyboardMode> onKeyboardModeChanged;
  final VoidCallback onClose;
  final ValueChanged<_TerminalToolAction> onSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return TweenAnimationBuilder<double>(
      duration: AnyttyMotion.resolve(
        context,
        const Duration(milliseconds: 150),
      ),
      curve: Curves.easeOutCubic,
      tween: Tween(begin: 0, end: 1),
      builder: (context, value, child) => ClipRect(
        child: Opacity(
          opacity: value,
          child: FractionalTranslation(
            translation: Offset(0, 1 - value),
            child: child,
          ),
        ),
      ),
      child: Material(
        color: palette.surface,
        elevation: 12,
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(6),
          side: BorderSide(color: palette.borderStrong),
        ),
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: math.min(
              440,
              math.max(220, MediaQuery.sizeOf(context).height - 160),
            ),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              SizedBox(
                height: 42,
                child: Row(
                  children: [
                    const SizedBox(width: 10),
                    Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: _terminalStateColor(context, terminalState),
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 7),
                    Text(
                      'Terminal',
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        '${_terminalStateLabel(terminalState)}  ${terminalSize.cols}x${terminalSize.rows}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: palette.muted,
                          fontFamily: 'JetBrainsMonoNerd',
                          fontSize: 9,
                        ),
                      ),
                    ),
                    Text(
                      resizeControl.canResize ? 'OWNER' : 'FOLLOWER',
                      style: TextStyle(
                        color: palette.muted,
                        fontSize: 8,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    IconButton(
                      tooltip: 'Close terminal tools',
                      onPressed: onClose,
                      constraints: const BoxConstraints.tightFor(
                        width: 40,
                        height: 40,
                      ),
                      icon: const Icon(LucideIcons.x, size: 16),
                    ),
                  ],
                ),
              ),
              Container(
                decoration: BoxDecoration(
                  color: palette.surfaceRaised,
                  border: Border.symmetric(
                    horizontal: BorderSide(color: palette.border),
                  ),
                ),
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                child: Row(
                  children: [
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.clipboardPaste,
                        label: 'Paste',
                        enabled: inputEnabled,
                        onPressed: () => onSelected(_TerminalToolAction.paste),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.history,
                        label: historyActive ? 'Live' : 'History',
                        enabled: historyActionEnabled,
                        active: historyActive,
                        onPressed: () =>
                            onSelected(_TerminalToolAction.history),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.search,
                        label: 'Search',
                        active: searchActive,
                        onPressed: () => onSelected(_TerminalToolAction.search),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.scanText,
                        label: 'Select',
                        onPressed: () =>
                            onSelected(_TerminalToolAction.selection),
                      ),
                    ),
                  ],
                ),
              ),
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(8, 8, 8, 10),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const _ToolSectionLabel('TERMINAL'),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.slidersHorizontal,
                              label: 'Shortcut bar',
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.commandBar),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.maximize2,
                              label: 'Resize',
                              enabled: resizeAvailable,
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.resize),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.settings,
                              label: 'Settings',
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.settings),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 9),
                      const _ToolSectionLabel('SPLIT'),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowLeftToLine,
                              label: 'Left',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.left),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowRightToLine,
                              label: 'Right',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.right),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowUpToLine,
                              label: 'Above',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.above),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowDownToLine,
                              label: 'Below',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.below),
                            ),
                          ),
                          if (splitOpen)
                            Expanded(
                              child: _CompactToolAction(
                                icon: syncInput
                                    ? LucideIcons.link2
                                    : LucideIcons.unlink2,
                                label: 'Sync',
                                active: syncInput,
                                onPressed: onToggleSync,
                              ),
                            ),
                          if (splitOpen)
                            Expanded(
                              child: _CompactToolAction(
                                icon: LucideIcons.x,
                                label: 'Close',
                                onPressed: onCloseSplit,
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(height: 9),
                      const _ToolSectionLabel('KEYBOARD LAYOUT'),
                      const SizedBox(height: 4),
                      _TerminalKeyboardModeControl(
                        value: keyboardMode,
                        onChanged: onKeyboardModeChanged,
                      ),
                    ],
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

final class _TerminalKeyboardModeControl extends StatelessWidget {
  const _TerminalKeyboardModeControl({
    required this.value,
    required this.onChanged,
  });

  final TerminalKeyboardMode value;
  final ValueChanged<TerminalKeyboardMode> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    const options = <({TerminalKeyboardMode mode, String label})>[
      (mode: TerminalKeyboardMode.automatic, label: 'Auto'),
      (mode: TerminalKeyboardMode.resize, label: 'Resize'),
      (mode: TerminalKeyboardMode.shift, label: 'Shift'),
    ];
    return Semantics(
      label: 'Keyboard',
      container: true,
      child: Container(
        height: 38,
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          color: palette.surfaceRaised,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(4),
        ),
        child: Row(
          children: [
            for (final option in options)
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 1),
                  child: InkWell(
                    onTap: () {
                      unawaited(HapticFeedback.selectionClick());
                      onChanged(option.mode);
                    },
                    borderRadius: BorderRadius.circular(3),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: value == option.mode
                            ? Color.lerp(palette.surface, palette.accent, 0.18)
                            : Colors.transparent,
                        borderRadius: BorderRadius.circular(3),
                      ),
                      child: Center(
                        child: Text(
                          option.label,
                          style: TextStyle(
                            color: value == option.mode
                                ? palette.accent
                                : palette.muted,
                            fontSize: 10,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

final class _CompactToolAction extends StatelessWidget {
  const _CompactToolAction({
    required this.icon,
    required this.label,
    required this.onPressed,
    this.enabled = true,
    this.active = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final bool enabled;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final foreground = enabled
        ? active
              ? palette.accent
              : palette.text
        : palette.faint;
    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: label,
      child: SizedBox(
        height: 48,
        child: InkWell(
          onTap: enabled
              ? () {
                  unawaited(HapticFeedback.selectionClick());
                  onPressed();
                }
              : null,
          borderRadius: BorderRadius.circular(4),
          child: AnimatedContainer(
            duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
            margin: const EdgeInsets.symmetric(horizontal: 1),
            decoration: BoxDecoration(
              color: active
                  ? Color.lerp(palette.surface, palette.accent, 0.16)
                  : Colors.transparent,
              border: active
                  ? Border.all(
                      color: Color.lerp(palette.border, palette.accent, 0.48)!,
                    )
                  : null,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, size: 15, color: foreground),
                const SizedBox(height: 3),
                Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: foreground,
                    fontSize: 8.5,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _ToolSectionLabel extends StatelessWidget {
  const _ToolSectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) => Text(
    label,
    style: TextStyle(
      color: AnyttyPalette.of(context).faint,
      fontSize: 8,
      fontWeight: FontWeight.w700,
    ),
  );
}

final class _TerminalSettingsSheet extends StatefulWidget {
  const _TerminalSettingsSheet({
    required this.initialSettings,
    required this.onChanged,
  });

  final TerminalSettings initialSettings;
  final Future<void> Function(TerminalSettings) onChanged;

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
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 0, 8, 8),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    'Terminal settings',
                    style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
                  ),
                ),
                IconButton(
                  tooltip: 'Close terminal settings',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close_rounded),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 24),
              children: [
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

final class _ResizeControlSheet extends StatefulWidget {
  const _ResizeControlSheet({
    required this.connection,
    required this.initialControl,
    required this.viewportCells,
  });

  final TerminalConnection connection;
  final ResizeControl initialControl;
  final ({int cols, int rows})? viewportCells;

  @override
  State<_ResizeControlSheet> createState() => _ResizeControlSheetState();
}

final class _ResizeControlSheetState extends State<_ResizeControlSheet> {
  late ResizeControl _control;
  StreamSubscription<ResizeControl>? _subscription;
  bool _busy = false;
  String? _error;

  bool get _ownsResize =>
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_OWNER ||
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_SIZE_LOCKED;

  @override
  void initState() {
    super.initState();
    _control = widget.initialControl.deepCopy();
    _subscription = widget.connection.watchResizeControl().listen((control) {
      if (mounted) setState(() => _control = control);
    });
  }

  @override
  void dispose() {
    unawaited(_subscription?.cancel());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final size = _control.hasOwnership() && _control.ownership.hasSize()
        ? _control.ownership.size
        : widget.connection.terminalSize;
    final owner = _control.ownerSurfaceId.isNotEmpty
        ? _control.ownerSurfaceId
        : _control.hasOwnership()
        ? _control.ownership.ownerSurfaceId
        : '';
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Terminal size',
              style: TextStyle(
                color: palette.text,
                fontSize: 17,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 14),
            _ResizeStatusRow(
              icon: _control.sizeLocked
                  ? Icons.lock_outline_rounded
                  : _ownsResize
                  ? Icons.aspect_ratio_rounded
                  : Icons.visibility_outlined,
              label: _resizeControlStatus(_control),
              value: '${size.cols} x ${size.rows}',
            ),
            if (owner.isNotEmpty) ...[
              const SizedBox(height: 8),
              _ResizeStatusRow(
                icon: Icons.devices_other_rounded,
                label: 'Owner surface',
                value: owner,
              ),
            ],
            if (_error case final error?) ...[
              const SizedBox(height: 12),
              Text(
                error,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: palette.danger, fontSize: 12),
              ),
            ],
            const SizedBox(height: 18),
            if (_ownsResize) ...[
              FilledButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(
                        () => widget.connection.setResizeLock(
                          !_control.sizeLocked,
                        ),
                      ),
                icon: Icon(
                  _control.sizeLocked
                      ? Icons.lock_open_rounded
                      : Icons.lock_outline_rounded,
                ),
                label: Text(_control.sizeLocked ? 'Unlock size' : 'Lock size'),
              ),
              const SizedBox(height: 8),
              OutlinedButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(widget.connection.releaseResizeOwnership),
                icon: const Icon(Icons.logout_rounded),
                label: const Text('Release ownership'),
              ),
            ] else
              FilledButton.icon(
                onPressed: _busy ? null : _takeOwnership,
                icon: const Icon(Icons.aspect_ratio_rounded),
                label: const Text('Take resize ownership'),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _takeOwnership() async {
    final cells = widget.viewportCells;
    final size = widget.connection.terminalSize;
    await _run(
      () => widget.connection.requestResizeOwnership(
        cols: cells?.cols ?? size.cols,
        rows: cells?.rows ?? size.rows,
      ),
    );
  }

  Future<void> _run(Future<ResizeControl> Function() operation) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final control = await operation();
      if (mounted) setState(() => _control = control);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

final class _ResizeStatusRow extends StatelessWidget {
  const _ResizeStatusRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Icon(icon, color: palette.muted, size: 20),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            label,
            style: TextStyle(color: palette.text, fontSize: 13),
          ),
        ),
        const SizedBox(width: 12),
        Flexible(
          child: Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: palette.text,
              fontFamily: 'monospace',
              fontSize: 12,
            ),
          ),
        ),
      ],
    );
  }
}

String _resizeControlStatus(ResizeControl control) {
  if (control.sizeLocked) {
    return control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER
        ? 'Follower / locked'
        : 'Owner / locked';
  }
  return switch (control.reason) {
    ResizeControlReason.RESIZE_CONTROL_REASON_OWNER => 'Resize owner',
    ResizeControlReason.RESIZE_CONTROL_REASON_OBSERVER => 'Observer',
    ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER => 'Follower',
    _ => control.canResize ? 'Resize owner' : 'Unavailable',
  };
}

final class _KeyboardKeyButton extends StatefulWidget {
  const _KeyboardKeyButton({
    required this.keyboardVisible,
    required this.focusLocked,
    required this.onTap,
    required this.onLongPress,
  });

  final bool keyboardVisible;
  final bool focusLocked;
  final VoidCallback? onTap;
  final VoidCallback? onLongPress;

  @override
  State<_KeyboardKeyButton> createState() => _KeyboardKeyButtonState();
}

final class _KeyboardKeyButtonState extends State<_KeyboardKeyButton> {
  static const _longPressDelay = Duration(milliseconds: 400);

  Timer? _longPressTimer;
  int? _pointer;
  bool _longPressTriggered = false;

  @override
  void dispose() {
    _cancelLongPress();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final enabled = widget.onTap != null && widget.onLongPress != null;
    final active = widget.focusLocked || widget.keyboardVisible;
    final tooltip = widget.focusLocked
        ? 'Allow the system keyboard to open'
        : widget.keyboardVisible
        ? 'Hide system keyboard'
        : 'Show system keyboard';
    final background = widget.focusLocked
        ? palette.warning
        : widget.keyboardVisible
        ? palette.text
        : palette.surfaceRaised;
    final foreground = active
        ? palette.background
        : enabled
        ? palette.muted
        : palette.faint;

    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: tooltip,
      onTap: enabled ? _handleSemanticTap : null,
      child: Tooltip(
        message: tooltip,
        child: Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: enabled ? _handlePointerDown : null,
          onPointerUp: enabled ? _handlePointerUp : null,
          onPointerCancel: enabled ? _handlePointerCancel : null,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Center(
              child: Icon(LucideIcons.keyboard, size: 18, color: foreground),
            ),
          ),
        ),
      ),
    );
  }

  void _handlePointerDown(PointerDownEvent event) {
    if (_pointer != null) return;
    _pointer = event.pointer;
    _longPressTriggered = false;
    _longPressTimer = Timer(_longPressDelay, () {
      _longPressTimer = null;
      _longPressTriggered = true;
      HapticFeedback.mediumImpact();
      widget.onLongPress?.call();
    });
  }

  void _handlePointerUp(PointerUpEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _cancelLongPress();
    if (_longPressTriggered) {
      _longPressTriggered = false;
      return;
    }
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _handlePointerCancel(PointerCancelEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _longPressTriggered = false;
    _cancelLongPress();
  }

  void _handleSemanticTap() {
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _cancelLongPress() {
    _longPressTimer?.cancel();
    _longPressTimer = null;
  }
}

bool _mouseTrackingEnabled(TerminalModes? modes) =>
    modes?.mouseTracking == true ||
    modes?.mouseX10 == true ||
    modes?.mouseNormal == true ||
    modes?.mouseButtonEvent == true ||
    modes?.mouseAnyEvent == true;

const _usbKeyboardPage = 0x00070000;

/// follower 的屏幕尺寸可能和本机视口对不上（例如桌面持有 resize lease）：
/// 比视口宽时按比例缩小到视口宽度，宁可字小一点，也不要把行尾裁掉。
///
/// 缩放后的布局尺寸必须跟着视觉尺寸走，否则父级仍按原始尺寸占位，右边和
/// 下方会留出大片没画背景的空白。背景色铺满整个可用区域，内容顶部对齐。
final class _FitWidthCanvas extends StatelessWidget {
  const _FitWidthCanvas({
    required this.screen,
    required this.metrics,
    required this.background,
    required this.child,
  });

  final CanonicalLiveScreen screen;
  final TerminalCellMetrics metrics;
  final int background;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final naturalWidth = screen.cols * metrics.cellWidth;
        final naturalHeight = screen.rows * metrics.rowHeight;
        final scale = naturalWidth <= 0 || constraints.maxWidth <= 0
            ? 1.0
            : math.min(1.0, constraints.maxWidth / naturalWidth);
        final scaledWidth = naturalWidth * scale;
        final scaledHeight = naturalHeight * scale;
        return ColoredBox(
          color: Color(0xff000000 | background),
          child: (naturalWidth <= 0 || naturalHeight <= 0)
              ? child
              : Align(
                  alignment: Alignment.topLeft,
                  child: SizedBox(
                    width: scaledWidth,
                    height: scaledHeight,
                    child: FittedBox(
                      fit: BoxFit.fill,
                      alignment: Alignment.topLeft,
                      child: SizedBox(
                        width: naturalWidth,
                        height: naturalHeight,
                        child: child,
                      ),
                    ),
                  ),
                ),
        );
      },
    );
  }
}
