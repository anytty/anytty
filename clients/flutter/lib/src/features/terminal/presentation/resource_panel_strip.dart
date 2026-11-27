part of 'terminal_resource_panel.dart';


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
