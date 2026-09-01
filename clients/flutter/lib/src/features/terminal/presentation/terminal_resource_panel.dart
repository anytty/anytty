import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';

List<TerminalResourceUsage> terminalResourceSamples(TerminalInfo terminal) {
  final samples = terminal.resourceHistory
      .map((sample) => sample.deepCopy())
      .toList();
  if (terminal.hasResources()) {
    final current = terminal.resources;
    final duplicate =
        samples.isNotEmpty &&
        current.sampledAtUnixNano == samples.last.sampledAtUnixNano &&
        current.pid == samples.last.pid &&
        current.cpuPercentX100 == samples.last.cpuPercentX100 &&
        current.memoryBytes == samples.last.memoryBytes;
    if (!duplicate) samples.add(current.deepCopy());
  }
  return List.unmodifiable(samples);
}

typedef TerminalResourceTotals = ({
  int cpuX100,
  int memoryBytes,
  int reportingCount,
  int runningCount,
});

TerminalResourceTotals? terminalResourceTotals(
  Iterable<TerminalInfo> terminals,
) {
  final running = terminals
      .where(
        (terminal) => terminal.state == TerminalState.TERMINAL_STATE_RUNNING,
      )
      .toList(growable: false);
  final current = <TerminalResourceUsage>[
    for (final terminal in running) ?_currentTerminalResource(terminal),
  ];
  if (current.isEmpty) return null;
  return (
    cpuX100: current.fold<int>(
      0,
      (sum, sample) => sum + math.max(0, sample.cpuPercentX100),
    ),
    memoryBytes: current.fold<int>(
      0,
      (sum, sample) => sum + math.max(0, sample.memoryBytes.toInt()),
    ),
    reportingCount: current.length,
    runningCount: running.length,
  );
}

typedef TerminalResourceSeries = ({
  List<double> cpuPercent,
  List<double> memoryBytes,
});

TerminalResourceSeries terminalAggregateResourceSeries(
  Iterable<TerminalInfo> terminals,
) {
  final histories = terminals
      .where(
        (terminal) => terminal.state == TerminalState.TERMINAL_STATE_RUNNING,
      )
      .map(terminalResourceSamples)
      .where((samples) => samples.isNotEmpty)
      .toList(growable: false);
  final sampleCount = histories.fold<int>(
    0,
    (largest, samples) => math.max(largest, samples.length),
  );
  final cpu = <double>[];
  final memory = <double>[];
  for (var position = 0; position < sampleCount; position++) {
    final distanceFromLatest = sampleCount - position;
    var cpuTotal = 0.0;
    var memoryTotal = 0.0;
    for (final samples in histories) {
      final index = samples.length - distanceFromLatest;
      if (index < 0) continue;
      cpuTotal += math.max(0, samples[index].cpuPercentX100) / 100;
      memoryTotal += math.max(0, samples[index].memoryBytes.toInt()).toDouble();
    }
    cpu.add(cpuTotal);
    memory.add(memoryTotal);
  }
  return (
    cpuPercent: List.unmodifiable(cpu),
    memoryBytes: List.unmodifiable(memory),
  );
}

TerminalResourceUsage? _currentTerminalResource(TerminalInfo terminal) {
  final samples = terminalResourceSamples(terminal);
  return samples.isEmpty ? null : samples.last;
}

enum TerminalResourceMetric { cpu, memory }

final class TerminalResourceStrip extends StatelessWidget {
  const TerminalResourceStrip({super.key, required this.terminals});

  final List<TerminalInfo> terminals;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final totals = terminalResourceTotals(terminals);
    final runningCount = terminals
        .where(
          (terminal) => terminal.state == TerminalState.TERMINAL_STATE_RUNNING,
        )
        .length;
    final canOpen = totals != null;
    final cpu = totals == null
        ? '--'
        : '${(totals.cpuX100 / 100).toStringAsFixed(1)}%';
    final memory = totals == null
        ? '--'
        : formatResourceBytes(totals.memoryBytes);
    return Container(
      margin: const EdgeInsets.fromLTRB(14, 3, 14, 8),
      padding: const EdgeInsets.symmetric(vertical: 10),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(18),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(
              alpha: Theme.of(context).brightness == Brightness.dark
                  ? 0.16
                  : 0.045,
            ),
            blurRadius: 18,
            offset: const Offset(0, 5),
          ),
        ],
      ),
      child: SizedBox(
        height: 48,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: _ResourceStripCell(
                label: anyttyText(context, en: 'Running', zh: '运行中'),
                value: '$runningCount',
              ),
            ),
            VerticalDivider(width: 1, color: palette.border),
            Expanded(
              child: _ResourceStripCell(
                key: const ValueKey('terminal-resource-strip-cpu'),
                label: 'CPU',
                value: cpu,
                enabled: canOpen,
                onTap: () => showTerminalAggregateResourceDetails(
                  context,
                  terminals,
                  initialMetric: TerminalResourceMetric.cpu,
                ),
              ),
            ),
            VerticalDivider(width: 1, color: palette.border),
            Expanded(
              child: _ResourceStripCell(
                key: const ValueKey('terminal-resource-strip-memory'),
                label: anyttyText(context, en: 'Memory', zh: '内存'),
                value: memory,
                enabled: canOpen,
                onTap: () => showTerminalAggregateResourceDetails(
                  context,
                  terminals,
                  initialMetric: TerminalResourceMetric.memory,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _ResourceStripCell extends StatelessWidget {
  const _ResourceStripCell({
    super.key,
    required this.label,
    required this.value,
    this.enabled = false,
    this.onTap,
  });

  final String label;
  final String value;
  final bool enabled;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final content = Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Text(
          label,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            color: palette.muted,
            fontSize: 9,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          value,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            color: palette.text,
            fontFamily: 'JetBrainsMonoNerd',
            fontSize: 14,
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
    if (!enabled || onTap == null) return content;
    return Semantics(
      button: true,
      label: '$label $value',
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onTap,
          borderRadius: BorderRadius.circular(12),
          child: content,
        ),
      ),
    );
  }
}

final class TerminalResourceInlineSummary extends StatelessWidget {
  const TerminalResourceInlineSummary({
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
    final current = _currentTerminalResource(terminal);
    final cpu = current == null
        ? '--'
        : '${(current.cpuPercentX100 / 100).toStringAsFixed(1)}%';
    final memory = current == null
        ? '--'
        : formatResourceBytes(current.memoryBytes.toInt());
    final cpuSamples = samples
        .map((sample) => math.max(0, sample.cpuPercentX100) / 100)
        .toList(growable: false);
    final memorySamples = samples
        .map((sample) => math.max(0, sample.memoryBytes.toInt()).toDouble())
        .toList(growable: false);
    return Semantics(
      button: current != null,
      label: current == null
          ? 'Resource snapshot unavailable'
          : 'Resource snapshot, $cpu CPU, $memory memory',
      child: InkWell(
        key: ValueKey('resources-${terminal.ref.terminalId}'),
        borderRadius: BorderRadius.circular(4),
        onTap: current == null ? null : onTap,
        child: SizedBox(
          height: 24,
          child: Row(
            children: [
              Expanded(
                child: _InlineMetric(
                  label: 'CPU',
                  value: cpu,
                  color: palette.accent,
                  samples: cpuSamples,
                  chartKey: ValueKey(
                    'terminal-resource-sparkline-cpu-${terminal.ref.terminalId}',
                  ),
                ),
              ),
              Container(
                width: 1,
                height: 18,
                margin: const EdgeInsets.symmetric(horizontal: 7),
                color: palette.border,
              ),
              Expanded(
                child: _InlineMetric(
                  label: 'RAM',
                  value: memory,
                  color: palette.warning,
                  samples: memorySamples,
                  chartKey: ValueKey(
                    'terminal-resource-sparkline-memory-${terminal.ref.terminalId}',
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

final class _InlineMetric extends StatelessWidget {
  const _InlineMetric({
    required this.label,
    required this.value,
    required this.color,
    required this.samples,
    required this.chartKey,
  });

  final String label;
  final String value;
  final Color color;
  final List<double> samples;
  final Key chartKey;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final chartWidth = (constraints.maxWidth * 0.34).clamp(28.0, 46.0);
        return Row(
          children: [
            SizedBox(
              key: chartKey,
              width: chartWidth,
              height: 18,
              child: CustomPaint(
                painter: _InlineSparklinePainter(
                  values: samples,
                  color: color,
                  baselineColor: palette.border,
                ),
              ),
            ),
            const SizedBox(width: 6),
            Expanded(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    maxLines: 1,
                    style: TextStyle(
                      color: color,
                      fontSize: 7,
                      height: 1,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    value,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.text,
                      fontFamily: 'JetBrainsMonoNerd',
                      fontSize: 9,
                      height: 1,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ],
              ),
            ),
          ],
        );
      },
    );
  }
}

final class _InlineSparklinePainter extends CustomPainter {
  const _InlineSparklinePainter({
    required this.values,
    required this.color,
    required this.baselineColor,
  });

  final List<double> values;
  final Color color;
  final Color baselineColor;

  @override
  void paint(Canvas canvas, Size size) {
    final baselinePaint = Paint()
      ..color = baselineColor.withValues(alpha: 0.7)
      ..strokeWidth = 1;
    canvas.drawLine(
      Offset(0, size.height - 1),
      Offset(size.width, size.height - 1),
      baselinePaint,
    );
    if (values.isEmpty) return;

    final minimum = values.reduce(math.min);
    final maximum = values.reduce(math.max);
    final range = maximum - minimum;
    final path = Path();
    for (var index = 0; index < values.length; index++) {
      final x = values.length == 1
          ? size.width / 2
          : size.width * index / (values.length - 1);
      final normalized = range == 0 ? 0.5 : (values[index] - minimum) / range;
      final y = size.height - 2 - normalized * (size.height - 5);
      if (index == 0) {
        path.moveTo(x, y);
      } else {
        path.lineTo(x, y);
      }
    }

    final linePaint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.6
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    if (values.length == 1) {
      final y = size.height / 2;
      canvas.drawLine(Offset(0, y), Offset(size.width, y), linePaint);
    } else {
      canvas.drawPath(path, linePaint);
    }
  }

  @override
  bool shouldRepaint(covariant _InlineSparklinePainter oldDelegate) {
    if (oldDelegate.color != color ||
        oldDelegate.baselineColor != baselineColor ||
        oldDelegate.values.length != values.length) {
      return true;
    }
    for (var index = 0; index < values.length; index++) {
      if (oldDelegate.values[index] != values[index]) return true;
    }
    return false;
  }
}

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

final class _TerminalResourceDetails extends StatelessWidget {
  const _TerminalResourceDetails({required this.terminal});

  final TerminalInfo terminal;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final samples = terminalResourceSamples(terminal);
    final current = samples.last;
    final label = terminal.name.trim().isNotEmpty
        ? terminal.name.trim()
        : terminal.ref.terminalId;
    return SizedBox(
      height: math.min(MediaQuery.sizeOf(context).height * 0.68, 520),
      child: Column(
        children: [
          SizedBox(
            height: 64,
            child: Padding(
              padding: const EdgeInsets.only(left: 16, right: 4),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        Text(
                          'Resource snapshots · ${samples.length} sample${samples.length == 1 ? '' : 's'}',
                          style: TextStyle(color: palette.muted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'Close resource details',
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
                Row(
                  children: [
                    Expanded(
                      child: _CurrentMetric(
                        label: 'CPU',
                        value:
                            '${(current.cpuPercentX100 / 100).toStringAsFixed(1)}%',
                        detail: 'process ${current.pid}',
                        color: palette.accent,
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: _CurrentMetric(
                        label: 'Memory',
                        value: formatResourceBytes(current.memoryBytes.toInt()),
                        detail: 'resident usage',
                        color: palette.warning,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 18),
                _ChartSection(
                  label: 'CPU history',
                  value:
                      '${(current.cpuPercentX100 / 100).toStringAsFixed(1)}%',
                  values: [
                    for (final sample in samples) sample.cpuPercentX100 / 100,
                  ],
                  minimumCeiling: 100,
                  color: palette.accent,
                ),
                const SizedBox(height: 18),
                _ChartSection(
                  label: 'Memory history',
                  value: formatResourceBytes(current.memoryBytes.toInt()),
                  values: [
                    for (final sample in samples) sample.memoryBytes.toDouble(),
                  ],
                  color: palette.warning,
                ),
                const SizedBox(height: 14),
                Text(
                  'Values come from the latest terminal inventory snapshot.',
                  style: TextStyle(color: palette.muted, fontSize: 11),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _CurrentMetric extends StatelessWidget {
  const _CurrentMetric({
    required this.label,
    required this.value,
    required this.detail,
    required this.color,
  });

  final String label;
  final String value;
  final String detail;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 102,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: TextStyle(
              color: color,
              fontSize: 10,
              fontWeight: FontWeight.w700,
            ),
          ),
          const Spacer(),
          Text(
            value,
            style: TextStyle(
              color: palette.text,
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 18,
              fontWeight: FontWeight.w700,
            ),
          ),
          Text(detail, style: TextStyle(color: palette.muted, fontSize: 9)),
        ],
      ),
    );
  }
}

final class _ChartSection extends StatelessWidget {
  const _ChartSection({
    required this.label,
    required this.value,
    required this.values,
    required this.color,
    this.minimumCeiling = 0,
  });

  final String label;
  final String value;
  final List<double> values;
  final Color color;
  final double minimumCeiling;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                label,
                style: TextStyle(
                  color: palette.text,
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            Text(
              value,
              style: TextStyle(
                color: color,
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 11,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Container(
          height: 78,
          padding: const EdgeInsets.all(8),
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            borderRadius: BorderRadius.circular(8),
          ),
          child: ResourceSparkline(
            values: values,
            minimumCeiling: minimumCeiling,
            color: color,
          ),
        ),
      ],
    );
  }
}

String formatResourceBytes(int bytes) {
  if (bytes < 1024) return '$bytes B';
  if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KiB';
  if (bytes < 1024 * 1024 * 1024) {
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MiB';
  }
  return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GiB';
}

bool _sameValues(List<double> left, List<double> right) {
  if (left.length != right.length) return false;
  for (var index = 0; index < left.length; index++) {
    if (left[index] != right[index]) return false;
  }
  return true;
}
