part of 'theme_color_settings_screen.dart';


final class _ColorPlanePainter extends CustomPainter {
  const _ColorPlanePainter({
    required this.hue,
    required this.saturation,
    required this.value,
    required this.borderColor,
  });

  final double hue;
  final double saturation;
  final double value;
  final Color borderColor;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final radius = BorderRadius.circular(8);
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
    canvas.drawRRect(
      radius.toRRect(rect.deflate(0.5)),
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1
        ..color = borderColor,
    );
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
      value != oldDelegate.value ||
      borderColor != oldDelegate.borderColor;
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
          prefixStyle: TextStyle(
            color: widget.labelColor,
            fontWeight: FontWeight.w800,
          ),
          contentPadding: const EdgeInsets.symmetric(horizontal: 8),
          constraints: const BoxConstraints(minHeight: 48, maxHeight: 48),
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
