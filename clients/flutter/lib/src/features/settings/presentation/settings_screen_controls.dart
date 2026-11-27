part of 'settings_screen.dart';


final class _TerminalSettingsPreview extends StatefulWidget {
  const _TerminalSettingsPreview({required this.settings});

  final TerminalSettings settings;

  @override
  State<_TerminalSettingsPreview> createState() =>
      _TerminalSettingsPreviewState();
}

final class _TerminalSettingsPreviewState
    extends State<_TerminalSettingsPreview>
    with SingleTickerProviderStateMixin {
  late final AnimationController _cursor;

  @override
  void initState() {
    super.initState();
    _cursor = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 260),
      value: 1,
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _pulseCursor(motionDisabled: AnyttyMotion.disabled(context));
  }

  @override
  void didUpdateWidget(_TerminalSettingsPreview oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.settings != oldWidget.settings) {
      _pulseCursor(motionDisabled: AnyttyMotion.disabled(context));
    }
  }

  @override
  void dispose() {
    _cursor.dispose();
    super.dispose();
  }

  void _pulseCursor({required bool motionDisabled}) {
    if (motionDisabled || !widget.settings.cursorBlink) {
      _cursor.stop();
      _cursor.value = 1;
      return;
    }
    _cursor.repeat(reverse: true, count: 2).whenComplete(() {
      if (mounted) _cursor.value = 1;
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final terminalTheme = widget.settings.theme;
    final background = _terminalColor(terminalTheme.background);
    final foreground = _terminalColor(terminalTheme.foreground);
    final green = _terminalColor(terminalTheme.ansi[2]);
    final cyan = _terminalColor(terminalTheme.ansi[6]);
    final yellow = _terminalColor(terminalTheme.ansi[3]);
    final previewFontSize = widget.settings.fontSize
        .toDouble()
        .clamp(10, 18)
        .toDouble();
    final lineStyle = TextStyle(
      color: foreground,
      fontFamily: widget.settings.fontFamily,
      fontSize: previewFontSize,
      height: 1.35,
      letterSpacing: 0,
    );
    final motionDisabled = AnyttyMotion.disabled(context);
    return Semantics(
      label:
          'Terminal preview, ${terminalTheme.label}, ${widget.settings.fontSize} point ${widget.settings.fontFamily}',
      child: AnimatedContainer(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.standard),
        curve: AnyttyMotion.emphasized,
        height: 148,
        clipBehavior: Clip.antiAlias,
        decoration: BoxDecoration(
          color: background,
          border: Border.all(color: palette.borderStrong),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              height: 32,
              padding: const EdgeInsets.symmetric(horizontal: 10),
              decoration: BoxDecoration(
                color: Color.alphaBlend(
                  foreground.withValues(alpha: 0.06),
                  background,
                ),
                border: Border(
                  bottom: BorderSide(color: foreground.withValues(alpha: 0.12)),
                ),
              ),
              child: Row(
                children: [
                  Container(
                    width: 7,
                    height: 7,
                    decoration: BoxDecoration(
                      color: green,
                      shape: BoxShape.circle,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      terminalTheme.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: foreground.withValues(alpha: 0.72),
                        fontFamily: widget.settings.fontFamily,
                        fontSize: 10,
                        letterSpacing: 0,
                      ),
                    ),
                  ),
                  Text(
                    '${widget.settings.fontSize} pt',
                    style: TextStyle(
                      color: foreground.withValues(alpha: 0.56),
                      fontFamily: widget.settings.fontFamily,
                      fontSize: 9,
                      letterSpacing: 0,
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(12, 9, 12, 8),
                child: FittedBox(
                  alignment: Alignment.topLeft,
                  fit: BoxFit.scaleDown,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text.rich(
                        TextSpan(
                          style: lineStyle,
                          children: [
                            TextSpan(
                              text: 'anytty',
                              style: TextStyle(color: green),
                            ),
                            const TextSpan(text: ' in '),
                            TextSpan(
                              text: '~/workspace',
                              style: TextStyle(color: cyan),
                            ),
                          ],
                        ),
                      ),
                      Text.rich(
                        TextSpan(
                          style: lineStyle,
                          children: [
                            TextSpan(
                              text: r'$ ',
                              style: TextStyle(color: yellow),
                            ),
                            const TextSpan(text: 'git status --short'),
                          ],
                        ),
                      ),
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            ' M clients/flutter/lib/app.dart ',
                            style: lineStyle,
                          ),
                          AnimatedBuilder(
                            animation: _cursor,
                            builder: (context, child) => Opacity(
                              opacity:
                                  widget.settings.cursorBlink && !motionDisabled
                                  ? 0.28 + (_cursor.value * 0.72)
                                  : 1,
                              child: child,
                            ),
                            child: Container(
                              width: math.max(2, previewFontSize * 0.52),
                              height: previewFontSize * 1.05,
                              color: _terminalColor(terminalTheme.cursor),
                            ),
                          ),
                        ],
                      ),
                    ],
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

Color _terminalColor(int rgb) => Color(0xff000000 | rgb);

final class _SettingsRow extends StatelessWidget {
  const _SettingsRow({
    required this.label,
    required this.child,
    this.stacked = false,
    this.excludeLabelSemantics = false,
  });

  final String label;
  final Widget child;
  final bool stacked;
  final bool excludeLabelSemantics;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    child: stacked
        ? Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ExcludeSemantics(
                excluding: excludeLabelSemantics,
                child: Text(
                  label,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              const SizedBox(height: 8),
              child,
            ],
          )
        : Row(
            children: [
              Expanded(
                child: ExcludeSemantics(
                  excluding: excludeLabelSemantics,
                  child: Text(
                    label,
                    maxLines: 2,
                    overflow: TextOverflow.visible,
                    softWrap: true,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      height: 1.25,
                    ),
                  ),
                ),
              ),
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
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 50,
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            tooltip: anyttyText(context, en: 'Decrease font size', zh: '减小字体'),
            onPressed: value > 8 ? () => onChanged(value - 1) : null,
            icon: const Icon(Icons.remove_rounded, size: 17),
          ),
          SizedBox(
            width: 42,
            child: Text(
              '$value',
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
            ),
          ),
          IconButton(
            tooltip: anyttyText(context, en: 'Increase font size', zh: '增大字体'),
            onPressed: value < 32 ? () => onChanged(value + 1) : null,
            icon: const Icon(Icons.add_rounded, size: 17),
          ),
        ],
      ),
    );
  }
}

final class _SettingsDropdown<T> extends StatelessWidget {
  const _SettingsDropdown({
    required this.semanticsLabel,
    required this.value,
    required this.values,
    required this.labelFor,
    required this.onChanged,
  });

  final String semanticsLabel;
  final T value;
  final List<T> values;
  final String Function(T value) labelFor;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) => MergeSemantics(
    child: Semantics(
      label: semanticsLabel,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 190),
        child: DropdownButton<T>(
          value: value,
          isExpanded: true,
          underline: const SizedBox.shrink(),
          borderRadius: BorderRadius.circular(6),
          items: [
            for (final item in values)
              DropdownMenuItem<T>(
                value: item,
                child: Text(
                  labelFor(item),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
          ],
          onChanged: (next) {
            if (next != null) onChanged(next);
          },
        ),
      ),
    ),
  );
}

final class _FontPreviewButton extends StatelessWidget {
  const _FontPreviewButton({required this.value, required this.onPressed});

  final String value;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: anyttyText(
        context,
        en: 'Font preview, ${terminalFontLabel(value)}, choose font',
        zh: '字体预览，${terminalFontLabel(value)}，选择字体',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const ValueKey('terminal-font-picker'),
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(
              minWidth: 174,
              maxWidth: 174,
              minHeight: 48,
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(
                          'Aa 01 ~/',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: palette.text,
                            fontFamily: value,
                            fontSize: 14,
                            letterSpacing: 0,
                          ),
                        ),
                        Text(
                          terminalFontLabel(value),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.muted, fontSize: 10),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    Icons.chevron_right_rounded,
                    size: 18,
                    color: palette.faint,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _ThemePreviewButton extends StatelessWidget {
  const _ThemePreviewButton({required this.value, required this.onPressed});

  final TerminalTheme value;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: anyttyText(
        context,
        en: 'Theme preview, ${value.label}, choose terminal theme',
        zh: '主题预览，${value.label}，选择终端主题',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const ValueKey('terminal-theme-picker'),
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(
              minWidth: 174,
              maxWidth: 174,
              minHeight: 48,
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(
                          value.label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 5),
                        _ThemeSwatches(theme: value, compact: true),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    Icons.chevron_right_rounded,
                    size: 18,
                    color: palette.faint,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<String?> _showFontPicker(BuildContext context, String selected) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<String>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    useSafeArea: true,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: palette.surface,
    barrierColor: palette.overlay,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
    ),
    builder: (context) => _FontPickerSheet(selected: selected),
  );
}
