part of 'terminal_resource_panel.dart';


final class TerminalResourceSummary extends StatelessWidget {
  const TerminalResourceSummary({
    super.key,
    required this.terminal,
    required this.onTap,
  });

  final TerminalInfo terminal;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final samples = terminalResourceSamples(terminal);
    if (samples.isEmpty) return const SizedBox.shrink();
    final current = samples.last;
    return Semantics(
      button: true,
      label:
          'Resource snapshot, ${(current.cpuPercentX100 / 100).toStringAsFixed(1)} percent CPU, ${formatResourceBytes(current.memoryBytes.toInt())} memory',
      child: InkWell(
        key: ValueKey('resources-${terminal.ref.terminalId}'),
        borderRadius: BorderRadius.circular(6),
        onTap: onTap,
        child: Container(
          height: 48,
          padding: const EdgeInsets.symmetric(horizontal: 8),
          decoration: BoxDecoration(
            border: Border(top: BorderSide(color: palette.border)),
          ),
          child: Row(
            children: [
              _MetricValue(
                label: 'CPU',
                value: '${(current.cpuPercentX100 / 100).toStringAsFixed(1)}%',
                color: palette.accent,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: ResourceSparkline(
                  values: [
                    for (final sample in samples) sample.cpuPercentX100 / 100,
                  ],
                  minimumCeiling: 100,
                  color: palette.accent,
                ),
              ),
              const SizedBox(width: 12),
              _MetricValue(
                label: 'MEM',
                value: formatResourceBytes(current.memoryBytes.toInt()),
                color: palette.warning,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: ResourceSparkline(
                  values: [
                    for (final sample in samples) sample.memoryBytes.toDouble(),
                  ],
                  color: palette.warning,
                ),
              ),
              const SizedBox(width: 4),
              Icon(Icons.chevron_right_rounded, size: 17, color: palette.faint),
            ],
          ),
        ),
      ),
    );
  }
}

Future<void> showTerminalResourceDetails(
  BuildContext context,
  TerminalInfo terminal,
) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<void>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: palette.surface,
    barrierColor: palette.overlay,
    showDragHandle: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
    ),
    builder: (context) => _TerminalResourceDetails(terminal: terminal),
  );
}

Future<void> showTerminalAggregateResourceDetails(
  BuildContext context,
  List<TerminalInfo> terminals, {
  required TerminalResourceMetric initialMetric,
}) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<void>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: palette.surface,
    barrierColor: palette.overlay,
    showDragHandle: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
    ),
    builder: (context) => _AggregateResourceDetails(
      terminals: terminals,
      initialMetric: initialMetric,
    ),
  );
}

final class _AggregateResourceDetails extends StatefulWidget {
  const _AggregateResourceDetails({
    required this.terminals,
    required this.initialMetric,
  });

  final List<TerminalInfo> terminals;
  final TerminalResourceMetric initialMetric;

  @override
  State<_AggregateResourceDetails> createState() =>
      _AggregateResourceDetailsState();
}

final class _AggregateResourceDetailsState
    extends State<_AggregateResourceDetails> {
  late TerminalResourceMetric _metric = widget.initialMetric;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final totals = terminalResourceTotals(widget.terminals);
    final series = terminalAggregateResourceSeries(widget.terminals);
    final cpuSelected = _metric == TerminalResourceMetric.cpu;
    final values = cpuSelected ? series.cpuPercent : series.memoryBytes;
    final currentValue = totals == null
        ? '--'
        : cpuSelected
        ? '${(totals.cpuX100 / 100).toStringAsFixed(1)}%'
        : formatResourceBytes(totals.memoryBytes);
    final color = cpuSelected ? palette.accent : palette.warning;
    final title = cpuSelected
        ? anyttyText(context, en: 'CPU sampling', zh: 'CPU 采样曲线')
        : anyttyText(context, en: 'Memory sampling', zh: '内存采样曲线');
    return SizedBox(
      height: math.min(MediaQuery.sizeOf(context).height * 0.58, 430),
      child: Column(
        children: [
          SizedBox(
            height: 58,
            child: Padding(
              padding: const EdgeInsets.only(left: 18, right: 4),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        Text(
                          anyttyText(
                            context,
                            en: '${values.length} aggregate samples',
                            zh: '${values.length} 个聚合采样点',
                          ),
                          style: TextStyle(color: palette.muted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Close resource details',
                      zh: '关闭资源详情',
                    ),
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close_rounded, size: 19),
                  ),
                ],
              ),
            ),
          ),
          Divider(height: 1, color: palette.border),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
              children: [
                Container(
                  height: 48,
                  padding: const EdgeInsets.all(4),
                  decoration: BoxDecoration(
                    color: palette.surfaceRaised,
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: _AggregateMetricOption(
                          label: 'CPU',
                          selected: cpuSelected,
                          onTap: () => setState(
                            () => _metric = TerminalResourceMetric.cpu,
                          ),
                        ),
                      ),
                      const SizedBox(width: 4),
                      Expanded(
                        child: _AggregateMetricOption(
                          label: anyttyText(context, en: 'Memory', zh: '内存'),
                          selected: !cpuSelected,
                          onTap: () => setState(
                            () => _metric = TerminalResourceMetric.memory,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 20),
                _ChartSection(
                  label: title,
                  value: currentValue,
                  values: values,
                  minimumCeiling: cpuSelected ? 100 : 0,
                  color: color,
                ),
                const SizedBox(height: 14),
                Text(
                  anyttyText(
                    context,
                    en: 'The curve combines the latest samples from running terminals on this device.',
                    zh: '曲线汇总此设备上所有运行中终端的最新采样。',
                  ),
                  style: TextStyle(
                    color: palette.muted,
                    fontSize: 11,
                    height: 1.45,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _AggregateMetricOption extends StatelessWidget {
  const _AggregateMetricOption({
    required this.label,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      child: Material(
        color: selected ? palette.surface : Colors.transparent,
        borderRadius: BorderRadius.circular(10),
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(10),
          child: Center(
            child: Text(
              label,
              style: TextStyle(
                color: selected ? palette.text : palette.muted,
                fontSize: 12,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _MetricValue extends StatelessWidget {
  const _MetricValue({
    required this.label,
    required this.value,
    required this.color,
  });

  final String label;
  final String value;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ConstrainedBox(
      constraints: const BoxConstraints(minWidth: 52),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 8,
              fontWeight: FontWeight.w700,
            ),
          ),
          Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: palette.text,
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 10,
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

final class ResourceSparkline extends StatelessWidget {
  const ResourceSparkline({
    super.key,
    required this.values,
    required this.color,
    this.minimumCeiling = 0,
  });

  final List<double> values;
  final Color color;
  final double minimumCeiling;

  @override
  Widget build(BuildContext context) => CustomPaint(
    size: const Size(double.infinity, 22),
    painter: _ResourceSparklinePainter(
      values: values,
      color: color,
      minimumCeiling: minimumCeiling,
    ),
  );
}

final class _ResourceSparklinePainter extends CustomPainter {
  const _ResourceSparklinePainter({
    required this.values,
    required this.color,
    required this.minimumCeiling,
  });

  final List<double> values;
  final Color color;
  final double minimumCeiling;

  @override
  void paint(Canvas canvas, Size size) {
    if (values.isEmpty || size.isEmpty) return;
    final normalized = values.map((value) => math.max(0, value)).toList();
    final ceiling = math.max(minimumCeiling, normalized.reduce(math.max));
    final scale = ceiling <= 0 ? 1.0 : ceiling;
    final line = Paint()
      ..color = color
      ..strokeWidth = 1.5
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round
      ..style = PaintingStyle.stroke;
    final fill = Paint()
      ..shader = LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [color.withValues(alpha: 0.22), color.withValues(alpha: 0)],
      ).createShader(Offset.zero & size)
      ..style = PaintingStyle.fill;
    final path = Path();
    for (var index = 0; index < normalized.length; index++) {
      final x = normalized.length == 1
          ? size.width
          : size.width * index / (normalized.length - 1);
      final y = size.height - (normalized[index] / scale * size.height * 0.86);
      if (index == 0) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
    }
    if (normalized.length == 1) {
      canvas.drawCircle(
        Offset(size.width / 2, path.getBounds().top),
        2,
        line..style = PaintingStyle.fill,
      );
      return;
    }
    final area = Path.from(path)
      ..lineTo(size.width, size.height)
      ..lineTo(0, size.height)
      ..close();
    canvas.drawPath(area, fill);
    canvas.drawPath(path, line);
  }

  @override
  bool shouldRepaint(covariant _ResourceSparklinePainter oldDelegate) =>
      color != oldDelegate.color ||
      minimumCeiling != oldDelegate.minimumCeiling ||
      !_sameValues(values, oldDelegate.values);
}
