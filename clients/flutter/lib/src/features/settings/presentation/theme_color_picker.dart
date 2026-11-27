part of 'theme_color_settings_screen.dart';


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
        final width = (constraints.maxWidth - 18) / 4;
        return Wrap(
          spacing: 6,
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
      child: Material(
        color: selected
            ? Color.alphaBlend(
                palette.accent.withValues(alpha: 0.12),
                palette.surfaceRaised,
              )
            : palette.surfaceRaised,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(8),
          side: BorderSide(color: selected ? palette.accent : palette.border),
        ),
        child: InkWell(
          key: ValueKey('theme-preset-${preset.id}'),
          onTap: onPressed,
          borderRadius: BorderRadius.circular(8),
          child: ConstrainedBox(
            constraints: const BoxConstraints(minHeight: 68),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 8),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Container(
                    width: 24,
                    height: 24,
                    decoration: BoxDecoration(
                      color: preset.color,
                      shape: BoxShape.circle,
                      border: Border.all(color: palette.borderStrong, width: 2),
                    ),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: selected ? palette.text : palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
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
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
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
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(color: palette.borderStrong, width: 2),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      _tokenLabel(context, token),
                      style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      colorHex(color),
                      style: TextStyle(
                        color: palette.muted,
                        fontFamily: 'JetBrainsMonoNerd',
                        fontSize: 12,
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
          Row(
            children: [
              Expanded(
                child: _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-r'),
                  label: 'R',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Red channel',
                    zh: '红色通道',
                  ),
                  value: _channel(color, 16),
                  labelColor: const Color(0xffd95f5f),
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 16, value)),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-g'),
                  label: 'G',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Green channel',
                    zh: '绿色通道',
                  ),
                  value: _channel(color, 8),
                  labelColor: const Color(0xff238b57),
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 8, value)),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: _RgbChannelField(
                  key: ValueKey('theme-rgb-${token.name}-b'),
                  label: 'B',
                  semanticLabel: anyttyText(
                    context,
                    en: 'Blue channel',
                    zh: '蓝色通道',
                  ),
                  value: _channel(color, 0),
                  labelColor: const Color(0xff367cc4),
                  onChanged: (value) =>
                      onColorChanged(_replaceChannel(color, 0, value)),
                ),
              ),
            ],
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
    return Container(
      height: 48,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        borderRadius: BorderRadius.circular(7),
      ),
      child: Row(
        children: [
          for (final token in _ColorToken.values)
            Expanded(
              child: Semantics(
                button: true,
                selected: token == value,
                inMutuallyExclusiveGroup: true,
                label: _tokenLabel(context, token),
                onTap: () => onChanged(token),
                excludeSemantics: true,
                child: Material(
                  color: token == value ? palette.surface : Colors.transparent,
                  borderRadius: BorderRadius.circular(5),
                  child: InkWell(
                    key: ValueKey('theme-token-${token.name}'),
                    borderRadius: BorderRadius.circular(5),
                    onTap: () => onChanged(token),
                    child: Center(
                      child: Text(
                        _tokenLabel(context, token),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: token == value ? palette.text : palette.muted,
                          fontSize: 12,
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
    final palette = AnyttyPalette.of(context);
    return Semantics(
      label: anyttyText(
        context,
        en: 'Color saturation and brightness',
        zh: '颜色饱和度和明度',
      ),
      value: colorHex(color),
      image: true,
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
                borderColor: palette.borderStrong,
              ),
            ),
          );
        },
      ),
    );
  }
}
