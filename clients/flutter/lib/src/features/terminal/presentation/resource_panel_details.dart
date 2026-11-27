part of 'terminal_resource_panel.dart';


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
