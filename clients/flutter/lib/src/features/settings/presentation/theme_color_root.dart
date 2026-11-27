part of 'theme_color_settings_screen.dart';


enum _ColorToken { accent, background, surface }

final class ThemeColorSettingsScreen extends ConsumerStatefulWidget {
  const ThemeColorSettingsScreen({super.key});

  @override
  ConsumerState<ThemeColorSettingsScreen> createState() =>
      _ThemeColorSettingsScreenState();
}

final class _ThemeColorSettingsScreenState
    extends ConsumerState<ThemeColorSettingsScreen> {
  late final AppColorPreferencesController _controller;
  Timer? _saveTimer;
  AppColorPreferences? _draft;
  _ColorToken _token = _ColorToken.accent;
  bool _dirty = false;

  @override
  void initState() {
    super.initState();
    _controller = ref.read(appColorPreferencesProvider.notifier);
  }

  @override
  void dispose() {
    _saveTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final asyncColors = ref.watch(appColorPreferencesProvider);
    final stored = asyncColors.valueOrNull;
    if (_draft == null && stored != null) _draft = stored;
    final colors = _draft ?? stored ?? AppColorPreferences.defaults;
    final appearance =
        ref.watch(appAppearanceProvider).valueOrNull ?? AppAppearance.dark;
    final palette = AnyttyPalette.of(context);

    return PopScope(
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop || !_dirty) return;
        _saveTimer?.cancel();
        final draft = _draft;
        if (draft != null) unawaited(_controller.save(draft));
      },
      child: Scaffold(
        appBar: AppBar(
          leading: IconButton(
            tooltip: anyttyText(context, en: 'Back to settings', zh: '返回设置'),
            onPressed: () => unawaited(_close()),
            icon: const Icon(LucideIcons.chevronLeft),
          ),
          titleSpacing: 0,
          title: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                anyttyText(context, en: 'Theme colors', zh: '主题颜色'),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w600,
                ),
              ),
              Text(
                anyttyText(context, en: 'Live app palette', zh: '全局实时配色'),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: palette.muted,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
          actions: [
            IconButton(
              key: const ValueKey('theme-colors-reset'),
              tooltip: anyttyText(
                context,
                en: 'Restore default colors',
                zh: '恢复默认颜色',
              ),
              onPressed: colors == AppColorPreferences.defaults
                  ? null
                  : () => unawaited(_reset()),
              icon: const Icon(LucideIcons.rotateCcw),
            ),
            const SizedBox(width: 4),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 576),
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
                children: [
                  _SectionLabel(
                    label: anyttyText(context, en: 'APPEARANCE', zh: '显示模式'),
                  ),
                  const SizedBox(height: 8),
                  _ThemeModeSelector(
                    value: appearance,
                    onChanged: (value) => unawaited(_saveAppearance(value)),
                  ),
                  const SizedBox(height: 24),
                  _SectionHeader(
                    label: anyttyText(
                      context,
                      en: 'Color direction',
                      zh: '颜色方向',
                    ),
                    trailing: colorHex(colors.accent),
                  ),
                  const SizedBox(height: 8),
                  _PresetPicker(
                    value: colors.accent,
                    onChanged: (color) =>
                        _apply(colors.copyWith(accent: color)),
                  ),
                  const SizedBox(height: 24),
                  _SectionHeader(
                    label: anyttyText(context, en: 'Color token', zh: '颜色令牌'),
                    trailing: anyttyText(
                      context,
                      en: 'RGB precision',
                      zh: 'RGB 精确值',
                    ),
                  ),
                  const SizedBox(height: 8),
                  _ColorEditor(
                    token: _token,
                    color: _colorFor(colors, _token),
                    onTokenChanged: (token) => setState(() => _token = token),
                    onColorChanged: (color) =>
                        _apply(_withColor(colors, _token, color)),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  void _apply(AppColorPreferences next) {
    setState(() {
      _draft = next;
      _dirty = true;
    });
    _controller.preview(next);
    _saveTimer?.cancel();
    _saveTimer = Timer(
      const Duration(milliseconds: 240),
      () => unawaited(_persist(next)),
    );
  }

  Future<void> _persist(AppColorPreferences next) async {
    try {
      await _controller.save(next);
      if (mounted && _draft == next) _dirty = false;
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _draft =
            ref.read(appColorPreferencesProvider).valueOrNull ??
            AppColorPreferences.defaults;
        _dirty = false;
      });
      _showError(
        '${anyttyText(context, en: 'Could not save theme colors', zh: '无法保存主题颜色')}: $error',
      );
    }
  }

  Future<void> _reset() async {
    _saveTimer?.cancel();
    setState(() {
      _draft = AppColorPreferences.defaults;
      _dirty = false;
    });
    try {
      await _controller.save(AppColorPreferences.defaults);
    } catch (error) {
      if (!mounted) return;
      _showError(
        '${anyttyText(context, en: 'Could not restore theme colors', zh: '无法恢复主题颜色')}: $error',
      );
    }
  }

  Future<void> _saveAppearance(AppAppearance value) async {
    try {
      await ref.read(appAppearanceProvider.notifier).save(value);
    } catch (error) {
      if (!mounted) return;
      _showError(
        '${anyttyText(context, en: 'Could not save appearance', zh: '无法保存外观设置')}: $error',
      );
    }
  }

  Future<void> _close() async {
    _saveTimer?.cancel();
    final draft = _draft;
    if (_dirty && draft != null) await _persist(draft);
    if (mounted) context.pop();
  }

  void _showError(String message) {
    final messenger = ScaffoldMessenger.of(context);
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(message)));
  }
}

final class _SectionLabel extends StatelessWidget {
  const _SectionLabel({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Text(
    label,
    style: TextStyle(
      color: AnyttyPalette.of(context).muted,
      fontSize: 11,
      fontWeight: FontWeight.w600,
    ),
  );
}

final class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label, required this.trailing});

  final String label;
  final String trailing;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Expanded(
          child: Text(
            label,
            style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
          ),
        ),
        Text(
          trailing,
          style: TextStyle(
            color: palette.muted,
            fontSize: 11,
            fontFeatures: const [FontFeature.tabularFigures()],
          ),
        ),
      ],
    );
  }
}

final class _ThemeModeSelector extends StatelessWidget {
  const _ThemeModeSelector({required this.value, required this.onChanged});

  final AppAppearance value;
  final ValueChanged<AppAppearance> onChanged;

  @override
  Widget build(BuildContext context) {
    final options = <(AppAppearance, IconData, String)>[
      (
        AppAppearance.light,
        LucideIcons.sun,
        anyttyText(context, en: 'Light', zh: '浅色'),
      ),
      (
        AppAppearance.dark,
        LucideIcons.moon,
        anyttyText(context, en: 'Dark', zh: '深色'),
      ),
      (
        AppAppearance.system,
        LucideIcons.sunMoon,
        anyttyText(context, en: 'System', zh: '跟随系统'),
      ),
    ];
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
        children: [
          for (var index = 0; index < options.length; index += 1) ...[
            if (index > 0) const SizedBox(width: 4),
            Expanded(
              child: _ModeOption(
                appearance: options[index].$1,
                icon: options[index].$2,
                label: options[index].$3,
                selected: value == options[index].$1,
                onPressed: () => onChanged(options[index].$1),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

final class _ModeOption extends StatelessWidget {
  const _ModeOption({
    required this.appearance,
    required this.icon,
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final AppAppearance appearance;
  final IconData icon;
  final String label;
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
              const SizedBox(width: 5),
              Flexible(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: selected ? palette.accentText : palette.text,
                    fontSize: 12,
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
