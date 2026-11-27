part of 'settings_screen.dart';


final class _FontPickerSheet extends StatelessWidget {
  const _FontPickerSheet({required this.selected});

  final String selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      top: false,
      child: SizedBox(
        height: math.min(MediaQuery.sizeOf(context).height * 0.62, 480),
        child: Column(
          children: [
            _PickerHeader(
              title: anyttyText(context, en: 'Choose font', zh: '选择字体'),
              closeTooltip: anyttyText(
                context,
                en: 'Close font picker',
                zh: '关闭字体选择',
              ),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
                children: [
                  Text(
                    anyttyText(context, en: 'FONT PREVIEW', zh: '字体预览'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surfaceRaised,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      children: [
                        for (
                          var index = 0;
                          index < terminalFontFamilies.length;
                          index++
                        ) ...[
                          _FontChoiceRow(
                            value: terminalFontFamilies[index],
                            selected: terminalFontFamilies[index] == selected,
                          ),
                          if (index != terminalFontFamilies.length - 1)
                            Divider(height: 1, color: palette.border),
                        ],
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _FontChoiceRow extends StatelessWidget {
  const _FontChoiceRow({required this.value, required this.selected});

  final String value;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      label: anyttyText(
        context,
        en: '${terminalFontLabel(value)} font preview',
        zh: '${terminalFontLabel(value)} 字体预览',
      ),
      onTap: () => Navigator.pop(context, value),
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('terminal-font-$value'),
        onTap: () => Navigator.pop(context, value),
        child: SizedBox(
          height: 78,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        terminalFontLabel(value),
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 5),
                      Text(
                        r'anytty ~/workspace $ Aa 01 {}[]',
                        maxLines: 1,
                        overflow: TextOverflow.clip,
                        style: TextStyle(
                          color: palette.muted,
                          fontFamily: value,
                          fontSize: 13,
                          letterSpacing: 0,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                SizedBox.square(
                  dimension: 24,
                  child: selected
                      ? Icon(
                          Icons.check_rounded,
                          color: palette.accent,
                          size: 21,
                        )
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

Future<String?> _showThemePicker(BuildContext context, String selected) {
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
    builder: (context) => _ThemePickerSheet(selected: selected),
  );
}

final class _ThemePickerSheet extends StatefulWidget {
  const _ThemePickerSheet({required this.selected});

  final String selected;

  @override
  State<_ThemePickerSheet> createState() => _ThemePickerSheetState();
}

final class _ThemePickerSheetState extends State<_ThemePickerSheet> {
  final GlobalKey _selectedKey = GlobalKey();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final selectedContext = _selectedKey.currentContext;
      if (!mounted || selectedContext == null) return;
      Scrollable.ensureVisible(
        selectedContext,
        alignment: 0.42,
        duration: AnyttyMotion.disabled(context)
            ? Duration.zero
            : AnyttyMotion.standard,
        curve: AnyttyMotion.emphasized,
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final dark = terminalThemes.where((theme) => theme.dark).toList();
    final light = terminalThemes.where((theme) => !theme.dark).toList();
    return SafeArea(
      top: false,
      child: SizedBox(
        height: math.min(MediaQuery.sizeOf(context).height * 0.78, 680),
        child: Column(
          children: [
            _PickerHeader(
              title: anyttyText(
                context,
                en: 'Choose terminal theme',
                zh: '选择终端主题',
              ),
              closeTooltip: anyttyText(
                context,
                en: 'Close theme picker',
                zh: '关闭主题选择',
              ),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
                children: [
                  _ThemeChoiceGroup(
                    label: anyttyText(context, en: 'DARK THEMES', zh: '深色主题'),
                    themes: dark,
                    selected: widget.selected,
                    selectedKey: _selectedKey,
                  ),
                  const SizedBox(height: 20),
                  _ThemeChoiceGroup(
                    label: anyttyText(context, en: 'LIGHT THEMES', zh: '浅色主题'),
                    themes: light,
                    selected: widget.selected,
                    selectedKey: _selectedKey,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _ThemeChoiceGroup extends StatelessWidget {
  const _ThemeChoiceGroup({
    required this.label,
    required this.themes,
    required this.selected,
    required this.selectedKey,
  });

  final String label;
  final List<TerminalTheme> themes;
  final String selected;
  final GlobalKey selectedKey;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          label,
          style: TextStyle(
            color: palette.muted,
            fontSize: 11,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        Container(
          clipBehavior: Clip.antiAlias,
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            border: Border.all(color: palette.border),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            children: [
              for (var index = 0; index < themes.length; index++) ...[
                _ThemeChoiceRow(
                  key: themes[index].id == selected ? selectedKey : null,
                  theme: themes[index],
                  selected: themes[index].id == selected,
                ),
                if (index != themes.length - 1)
                  Divider(height: 1, color: palette.border),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

final class _ThemeChoiceRow extends StatelessWidget {
  const _ThemeChoiceRow({
    super.key,
    required this.theme,
    required this.selected,
  });

  final TerminalTheme theme;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      label: anyttyText(
        context,
        en: '${theme.label} theme palette preview',
        zh: '${theme.label} 主题配色预览',
      ),
      onTap: () => Navigator.pop(context, theme.id),
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('terminal-theme-${theme.id}'),
        onTap: () => Navigator.pop(context, theme.id),
        child: SizedBox(
          height: 62,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    theme.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                _ThemeSwatches(theme: theme),
                const SizedBox(width: 10),
                SizedBox.square(
                  dimension: 24,
                  child: selected
                      ? Icon(
                          Icons.check_rounded,
                          color: palette.accent,
                          size: 21,
                        )
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _ThemeSwatches extends StatelessWidget {
  const _ThemeSwatches({required this.theme, this.compact = false});

  final TerminalTheme theme;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final colors = [
      theme.background,
      theme.foreground,
      theme.ansi[1],
      theme.ansi[2],
      theme.ansi[4],
    ];
    final size = compact ? 11.0 : 18.0;
    final gap = compact ? 3.0 : 4.0;
    return SizedBox(
      height: size,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (var index = 0; index < colors.length; index++) ...[
            Container(
              width: size,
              height: size,
              decoration: BoxDecoration(
                color: _terminalColor(colors[index]),
                borderRadius: BorderRadius.circular(compact ? 2 : 3),
                border: Border.all(
                  color: palette.borderStrong.withValues(alpha: 0.55),
                  width: 0.5,
                ),
              ),
            ),
            if (index != colors.length - 1) SizedBox(width: gap),
          ],
        ],
      ),
    );
  }
}

final class _PickerHeader extends StatelessWidget {
  const _PickerHeader({required this.title, required this.closeTooltip});

  final String title;
  final String closeTooltip;

  @override
  Widget build(BuildContext context) => SizedBox(
    height: 58,
    child: Padding(
      padding: const EdgeInsets.only(left: 16, right: 4),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
            ),
          ),
          IconButton(
            tooltip: closeTooltip,
            onPressed: () => Navigator.pop(context),
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
    ),
  );
}

final class _AppearancePicker extends StatelessWidget {
  const _AppearancePicker({required this.value, required this.onChanged});

  final AppAppearance value;
  final ValueChanged<AppAppearance> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 58,
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'Light', zh: '浅色'),
              icon: Icons.light_mode_outlined,
              selected: value == AppAppearance.light,
              onPressed: () => onChanged(AppAppearance.light),
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'Dark', zh: '深色'),
              icon: Icons.dark_mode_outlined,
              selected: value == AppAppearance.dark,
              onPressed: () => onChanged(AppAppearance.dark),
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'System', zh: '跟随系统'),
              icon: Icons.brightness_auto_outlined,
              selected: value == AppAppearance.system,
              onPressed: () => onChanged(AppAppearance.system),
            ),
          ),
        ],
      ),
    );
  }
}

final class _AppearanceOption extends StatelessWidget {
  const _AppearanceOption({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      inMutuallyExclusiveGroup: true,
      label: anyttyText(
        context,
        en: '$label interface theme',
        zh: '$label界面主题',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: selected ? palette.accent : Colors.transparent,
        borderRadius: BorderRadius.circular(6),
        child: InkWell(
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                icon,
                size: 17,
                color: selected ? palette.accentText : palette.muted,
              ),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: selected ? palette.accentText : palette.text,
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
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
