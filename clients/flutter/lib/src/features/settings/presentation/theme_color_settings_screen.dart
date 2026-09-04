import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/anytty_ui.dart';
import '../../../app/app_appearance.dart';
import '../../../app/app_color_preferences.dart';
import '../../../app/providers.dart';

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
    return PopScope(
      onPopInvokedWithResult: (didPop, result) {
        if (!didPop || !_dirty) return;
        _saveTimer?.cancel();
        final draft = _draft;
        if (draft != null) unawaited(_controller.save(draft));
      },
      child: Scaffold(
        appBar: AppBar(
          toolbarHeight: AnyttyUi.appBarHeight(context, subtitleLines: 2),
          leading: AnyttyIconButton(
            tooltip: anyttyText(context, en: 'Back to settings', zh: '返回设置'),
            onPressed: () => unawaited(_close()),
            icon: LucideIcons.chevronLeft,
          ),
          titleSpacing: 0,
          title: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                anyttyText(context, en: 'Theme colors', zh: '主题颜色'),
                style: AnyttyUi.title(context),
              ),
              Text(
                anyttyText(context, en: 'Live app palette', zh: '全局实时配色'),
                style: AnyttyUi.muted(context),
                maxLines: 2,
                softWrap: true,
                overflow: TextOverflow.ellipsis,
              ),
            ],
          ),
          actions: [
            AnyttyIconButton(
              key: const ValueKey('theme-colors-reset'),
              tooltip: anyttyText(
                context,
                en: 'Restore default colors',
                zh: '恢复默认颜色',
              ),
              onPressed: colors == AppColorPreferences.defaults
                  ? null
                  : () => unawaited(_reset()),
              icon: LucideIcons.rotateCcw,
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
  Widget build(BuildContext context) =>
      Text(label, style: AnyttyUi.sectionTitle(context));
}

final class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label, required this.trailing});

  final String label;
  final String trailing;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(child: Text(label, style: AnyttyUi.sectionTitle(context))),
        Flexible(
          child: Text(
            trailing,
            textAlign: TextAlign.end,
            style: AnyttyUi.muted(context)
                .copyWith(fontFeatures: const [FontFeature.tabularFigures()]),
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
    return LayoutBuilder(
      builder: (context, constraints) {
        final scale = MediaQuery.textScalerOf(context).scale(1);
        final compact = constraints.maxWidth < 420 || scale > 1.5;
        final columns = compact ? 2 : 3;
        final gap = 4.0;
        final lineHeight =
            MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5);
        final optionHeight = math
            .max(
              AnyttyUi.controlHeight(context),
              compact && scale > 1.5 ? lineHeight * 2 + 8 : 0,
            )
            .toDouble();
        final rows = (options.length + columns - 1) ~/ columns;
        final innerWidth = math.max(0.0, constraints.maxWidth - 8);
        final width = (innerWidth - gap * (columns - 1)) / columns;
        return Container(
          height: optionHeight * rows + gap * (rows - 1) + 8,
          padding: const EdgeInsets.all(4),
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            borderRadius: BorderRadius.circular(14),
            boxShadow: AnyttyUi.cardDecoration(
              context,
              radius: 14,
              depth: 1,
            ).boxShadow,
          ),
          child: Wrap(
            spacing: gap,
            runSpacing: gap,
            children: [
              for (final option in options)
                SizedBox(
                  width: width,
                  height: optionHeight,
                  child: _ModeOption(
                    appearance: option.$1,
                    icon: option.$2,
                    label: option.$3,
                    selected: value == option.$1,
                    onPressed: () => onChanged(option.$1),
                    height: optionHeight,
                  ),
                ),
            ],
          ),
        );
      },
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
    this.height,
  });

  final AppAppearance appearance;
  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onPressed;
  final double? height;

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
        color: Colors.transparent,
        child: DecoratedBox(
          decoration: AnyttyUi.pillDecoration(
            context,
            selected: selected,
            color: selected ? palette.accent : palette.surfaceRaised,
          ),
          child: InkWell(
            customBorder: const StadiumBorder(),
            onTap: onPressed,
            child: SizedBox(
              height: height ?? AnyttyUi.controlHeight(context),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(
                    icon,
                    size: 17,
                    color: selected ? palette.accentText : palette.strong,
                  ),
                  const SizedBox(width: 5),
                  Flexible(
                    child: Text(
                      label,
                      textAlign: TextAlign.center,
                      maxLines: 2,
                      softWrap: true,
                      overflow: TextOverflow.ellipsis,
                      style: AnyttyUi.body(context).copyWith(
                        color: selected ? palette.accentText : palette.text,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
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

final class _ColorPreset {
  const _ColorPreset(this.id, this.color, this.en, this.zh);

  final String id;
  final Color color;
  final String en;
  final String zh;
}

const _presets = <_ColorPreset>[
  _ColorPreset('cyan', Color(0xff32d5d0), 'Bright cyan', '亮青'),
  _ColorPreset('sky', Color(0xff32b9ef), 'Sky', '天蓝'),
  _ColorPreset('mint', Color(0xff42ddb0), 'Mint', '薄荷'),
  _ColorPreset('iris', Color(0xff819cff), 'Iris', '蓝紫'),
];

final class _PresetPicker extends StatelessWidget {
  const _PresetPicker({required this.value, required this.onChanged});

  final Color value;
  final ValueChanged<Color> onChanged;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final columns = constraints.maxWidth < 420 ? 2 : 4;
        final gap = 6.0;
        final width = (constraints.maxWidth - gap * (columns - 1)) / columns;
        return Wrap(
          spacing: gap,
          runSpacing: gap,
          children: [
            for (final preset in _presets)
              SizedBox(
                width: width,
                child: _PresetOption(
                  preset: preset,
                  selected: value == preset.color,
                  onPressed: () => onChanged(preset.color),
                ),
              ),
          ],
        );
      },
    );
  }
}

final class _PresetOption extends StatelessWidget {
  const _PresetOption({
    required this.preset,
    required this.selected,
    required this.onPressed,
  });

  final _ColorPreset preset;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final label = anyttyText(context, en: preset.en, zh: preset.zh);
    return Semantics(
      button: true,
      selected: selected,
      inMutuallyExclusiveGroup: true,
      label: label,
      onTap: onPressed,
      excludeSemantics: true,
      child: AnyttyCard(
        key: ValueKey('theme-preset-${preset.id}'),
        radius: 14,
        depth: selected ? 2 : 1,
        color: selected
            ? Color.alphaBlend(
                palette.accent.withValues(alpha: 0.12),
                palette.surfaceRaised,
              )
            : palette.surfaceRaised,
        onTap: onPressed,
        padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 8),
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 68),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Container(
                width: 24,
                height: 24,
                decoration: BoxDecoration(
                  color: preset.color,
                  shape: BoxShape.circle,
                  boxShadow: selected
                      ? [
                          BoxShadow(
                            color: palette.accent.withValues(alpha: 0.7),
                            blurRadius: 5,
                          ),
                        ]
                      : null,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                label,
                style: AnyttyUi.body(context).copyWith(
                  color: selected ? palette.text : palette.muted,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _ColorEditor extends StatelessWidget {
  const _ColorEditor({
    required this.token,
    required this.color,
    required this.onTokenChanged,
    required this.onColorChanged,
  });

  final _ColorToken token;
  final Color color;
  final ValueChanged<_ColorToken> onTokenChanged;
  final ValueChanged<Color> onColorChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final hsv = HSVColor.fromColor(color);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.surface,
        borderRadius: BorderRadius.circular(14),
        boxShadow: AnyttyUi.cardDecoration(
          context,
          radius: 14,
          depth: 1,
        ).boxShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _TokenSelector(value: token, onChanged: onTokenChanged),
          const SizedBox(height: 16),
          Row(
            children: [
              Container(
                width: 48,
                height: 48,
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(12),
                  boxShadow: AnyttyUi.cardDecoration(
                    context,
                    radius: 12,
                    depth: 1,
                    color: color,
                  ).boxShadow,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      _tokenLabel(context, token),
                      style: AnyttyUi.body(context)
                          .copyWith(fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      colorHex(color),
                      style: AnyttyUi.muted(context).copyWith(
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          _ColorPlane(color: color, onChanged: onColorChanged),
          const SizedBox(height: 8),
          _HueControl(
            hue: hsv.hue,
            onChanged: (hue) => onColorChanged(hsv.withHue(hue).toColor()),
          ),
          const SizedBox(height: 8),
          LayoutBuilder(
            builder: (context, constraints) {
              final fields = [
                _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-r'),
                  label: 'R',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Red channel',
                    zh: '红色通道',
                  ),
                  value: _channel(color, 16),
                  labelColor: palette.strong,
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 16, value)),
                ),
                _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-g'),
                  label: 'G',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Green channel',
                    zh: '绿色通道',
                  ),
                  value: _channel(color, 8),
                  labelColor: palette.strong,
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 8, value)),
                ),
                _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-b'),
                  label: 'B',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Blue channel',
                    zh: '蓝色通道',
                  ),
                  value: _channel(color, 0),
                  labelColor: palette.strong,
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 0, value)),
                ),
              ];
              final width = (constraints.maxWidth - 16) / 3;
              return Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final field in fields)
                    SizedBox(
                      width: constraints.maxWidth < 380
                          ? constraints.maxWidth
                          : width,
                      child: field,
                    ),
                ],
              );
            },
          ),
        ],
      ),
    );
  }
}

final class _TokenSelector extends StatelessWidget {
  const _TokenSelector({required this.value, required this.onChanged});

  final _ColorToken value;
  final ValueChanged<_ColorToken> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final scale = MediaQuery.textScalerOf(context).scale(1);
        final columns = constraints.maxWidth < 420 || scale > 1.5 ? 2 : 4;
        final gap = 4.0;
        final width = (constraints.maxWidth - gap * (columns - 1)) / columns;
        final rows = (_ColorToken.values.length + columns - 1) ~/ columns;
        final lineHeight = scale * (18 / 14.5) * 14.5;
        final tileHeight = math
            .max(
              AnyttyUi.controlHeight(context),
              scale > 1.5 ? lineHeight * 2 + 16 : 0,
            )
            .toDouble();
        return Container(
          height: tileHeight * rows + gap * (rows - 1),
          padding: EdgeInsets.zero,
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            borderRadius: BorderRadius.circular(14),
          ),
          child: Wrap(
            spacing: gap,
            runSpacing: gap,
            children: [
              for (final token in _ColorToken.values)
                SizedBox(
                  width: width,
                  height: tileHeight,
                  child: Semantics(
                    button: true,
                    selected: token == value,
                    inMutuallyExclusiveGroup: true,
                    label: _tokenLabel(context, token),
                    onTap: () => onChanged(token),
                    excludeSemantics: true,
                    child: DecoratedBox(
                      decoration: AnyttyUi.pillDecoration(
                        context,
                        selected: token == value,
                        color: token == value
                            ? palette.surface
                            : palette.surfaceRaised,
                      ),
                      child: Material(
                        color: Colors.transparent,
                        child: InkWell(
                          key: ValueKey('theme-token-${token.name}'),
                          customBorder: const StadiumBorder(),
                          onTap: () => onChanged(token),
                          child: Center(
                            child: Text(
                              _tokenLabel(context, token),
                              textAlign: TextAlign.center,
                              maxLines: 2,
                              softWrap: true,
                              overflow: TextOverflow.ellipsis,
                              style: AnyttyUi.body(context).copyWith(
                                color: token == value
                                    ? palette.text
                                    : palette.muted,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}

final class _ColorPlane extends StatelessWidget {
  const _ColorPlane({required this.color, required this.onChanged});

  final Color color;
  final ValueChanged<Color> onChanged;

  @override
  Widget build(BuildContext context) {
    final hsv = HSVColor.fromColor(color);
    final increasedColor = hsv
        .withSaturation((hsv.saturation + 0.05).clamp(0.0, 1.0))
        .toColor();
    final decreasedColor = hsv
        .withSaturation((hsv.saturation - 0.05).clamp(0.0, 1.0))
        .toColor();
    return Semantics(
      label: anyttyText(
        context,
        en: 'Color saturation and brightness',
        zh: '颜色饱和度和明度',
      ),
      value: colorHex(color),
      increasedValue: colorHex(increasedColor),
      decreasedValue: colorHex(decreasedColor),
      onIncrease: () => onChanged(increasedColor),
      onDecrease: () => onChanged(decreasedColor),
      child: LayoutBuilder(
        builder: (context, constraints) {
          const height = 168.0;
          final size = Size(constraints.maxWidth, height);
          void update(Offset position) {
            final saturation = (position.dx / size.width).clamp(0.0, 1.0);
            final value = 1 - (position.dy / size.height).clamp(0.0, 1.0);
            onChanged(
              HSVColor.fromAHSV(1, hsv.hue, saturation, value).toColor(),
            );
          }

          return GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTapDown: (details) => update(details.localPosition),
            onPanStart: (details) => update(details.localPosition),
            onPanUpdate: (details) => update(details.localPosition),
            child: CustomPaint(
              key: const ValueKey('theme-color-plane'),
              size: size,
              painter: _ColorPlanePainter(
                hue: hsv.hue,
                saturation: hsv.saturation,
                value: hsv.value,
              ),
            ),
          );
        },
      ),
    );
  }
}

final class _ColorPlanePainter extends CustomPainter {
  const _ColorPlanePainter({
    required this.hue,
    required this.saturation,
    required this.value,
  });

  final double hue;
  final double saturation;
  final double value;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final radius = BorderRadius.circular(14);
    canvas.save();
    canvas.clipRRect(radius.toRRect(rect));
    canvas.drawRect(
      rect,
      Paint()..color = HSVColor.fromAHSV(1, hue, 1, 1).toColor(),
    );
    canvas.drawRect(
      rect,
      Paint()
        ..shader = const LinearGradient(
          colors: [Colors.white, Color(0x00ffffff)],
        ).createShader(rect),
    );
    canvas.drawRect(
      rect,
      Paint()
        ..shader = const LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Color(0x00000000), Colors.black],
        ).createShader(rect),
    );
    canvas.restore();
    final thumb = Offset(saturation * size.width, (1 - value) * size.height);
    canvas.drawCircle(
      thumb,
      9,
      Paint()
        ..color = const Color(0x66000000)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 3),
    );
    canvas.drawCircle(
      thumb,
      7,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 2
        ..color = Colors.white,
    );
  }

  @override
  bool shouldRepaint(covariant _ColorPlanePainter oldDelegate) =>
      hue != oldDelegate.hue ||
      saturation != oldDelegate.saturation ||
      value != oldDelegate.value;
}

final class _HueControl extends StatelessWidget {
  const _HueControl({required this.hue, required this.onChanged});

  final double hue;
  final ValueChanged<double> onChanged;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: anyttyText(context, en: 'Hue', zh: '色相'),
      value: '${hue.round()}°',
      child: SizedBox(
        height: 48,
        child: Stack(
          alignment: Alignment.center,
          children: [
            Container(
              height: 8,
              margin: const EdgeInsets.symmetric(horizontal: 12),
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(4),
                gradient: const LinearGradient(
                  colors: [
                    Color(0xffff0000),
                    Color(0xffffff00),
                    Color(0xff00ff00),
                    Color(0xff00ffff),
                    Color(0xff0000ff),
                    Color(0xffff00ff),
                    Color(0xffff0000),
                  ],
                ),
              ),
            ),
            SliderTheme(
              data: SliderTheme.of(context).copyWith(
                activeTrackColor: Colors.transparent,
                inactiveTrackColor: Colors.transparent,
                trackHeight: 8,
                overlayShape: const RoundSliderOverlayShape(overlayRadius: 18),
              ),
              child: Slider(
                key: const ValueKey('theme-hue-slider'),
                value: hue,
                min: 0,
                max: 360,
                onChanged: onChanged,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _RgbChannelField extends StatefulWidget {
  const _RgbChannelField({
    super.key,
    required this.label,
    required this.semanticLabel,
    required this.value,
    required this.labelColor,
    required this.onChanged,
  });

  final String label;
  final String semanticLabel;
  final int value;
  final Color labelColor;
  final ValueChanged<int> onChanged;

  @override
  State<_RgbChannelField> createState() => _RgbChannelFieldState();
}

final class _RgbChannelFieldState extends State<_RgbChannelField> {
  late final TextEditingController _controller;
  late final FocusNode _focusNode;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: '${widget.value}');
    _focusNode = FocusNode()..addListener(_handleFocus);
  }

  @override
  void didUpdateWidget(_RgbChannelField oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!_focusNode.hasFocus && widget.value != oldWidget.value) {
      _setText(widget.value);
    }
  }

  @override
  void dispose() {
    _focusNode
      ..removeListener(_handleFocus)
      ..dispose();
    _controller.dispose();
    super.dispose();
  }

  void _handleFocus() {
    if (!_focusNode.hasFocus) _setText(widget.value);
  }

  void _setText(int value) {
    _controller.value = TextEditingValue(
      text: '$value',
      selection: TextSelection.collapsed(offset: '$value'.length),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Semantics(
      textField: true,
      label: widget.semanticLabel,
      value: '${widget.value}',
      child: TextField(
        controller: _controller,
        focusNode: _focusNode,
        keyboardType: TextInputType.number,
        textInputAction: TextInputAction.done,
        textAlign: TextAlign.center,
        inputFormatters: [
          FilteringTextInputFormatter.digitsOnly,
          LengthLimitingTextInputFormatter(3),
          _RgbValueFormatter(),
        ],
        decoration: InputDecoration(
          prefixText: '${widget.label} ',
          prefixStyle: AnyttyUi.body(context)
              .copyWith(color: widget.labelColor),
          contentPadding: const EdgeInsets.symmetric(horizontal: 8),
          constraints: BoxConstraints(
            minHeight: AnyttyUi.controlHeight(context),
          ),
        ),
        onChanged: (text) {
          final value = int.tryParse(text);
          if (value != null) widget.onChanged(value);
        },
        onSubmitted: (_) => _focusNode.unfocus(),
      ),
    );
  }
}

final class _RgbValueFormatter extends TextInputFormatter {
  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    if (newValue.text.isEmpty) return newValue;
    final value = int.tryParse(newValue.text);
    return value != null && value <= 255 ? newValue : oldValue;
  }
}

String _tokenLabel(BuildContext context, _ColorToken token) => switch (token) {
  _ColorToken.accent => anyttyText(context, en: 'Accent', zh: '强调色'),
  _ColorToken.background => anyttyText(context, en: 'Background', zh: '深色背景'),
  _ColorToken.surface => anyttyText(context, en: 'Surface', zh: '深色表面'),
};

Color _colorFor(AppColorPreferences colors, _ColorToken token) =>
    switch (token) {
      _ColorToken.accent => colors.accent,
      _ColorToken.background => colors.darkBackground,
      _ColorToken.surface => colors.darkSurface,
    };

AppColorPreferences _withColor(
  AppColorPreferences colors,
  _ColorToken token,
  Color color,
) => switch (token) {
  _ColorToken.accent => colors.copyWith(accent: color),
  _ColorToken.background => colors.copyWith(darkBackground: color),
  _ColorToken.surface => colors.copyWith(darkSurface: color),
};

int _channel(Color color, int shift) => (color.toARGB32() >> shift) & 0xff;

Color _replaceChannel(Color color, int shift, int channel) {
  final mask = ~(0xff << shift);
  final value = (color.toARGB32() & mask) | ((channel & 0xff) << shift);
  return Color(value);
}
