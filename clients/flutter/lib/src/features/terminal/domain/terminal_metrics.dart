final class TerminalCellMetrics {
  const TerminalCellMetrics({
    required this.fontSize,
    required this.cellWidth,
    required this.rowHeight,
  });

  factory TerminalCellMetrics.fromFontSize(double fontSize) {
    final normalized = fontSize.clamp(8, 32).toDouble();
    return TerminalCellMetrics(
      fontSize: normalized,
      cellWidth: normalized * (8.44 / 14),
      rowHeight: normalized * (20 / 14),
    );
  }

  final double fontSize;
  final double cellWidth;
  final double rowHeight;

  @override
  bool operator ==(Object other) =>
      other is TerminalCellMetrics &&
      other.fontSize == fontSize &&
      other.cellWidth == cellWidth &&
      other.rowHeight == rowHeight;

  @override
  int get hashCode => Object.hash(fontSize, cellWidth, rowHeight);
}

const defaultTerminalMetrics = TerminalCellMetrics(
  fontSize: 14,
  cellWidth: 8.44,
  rowHeight: 20,
);

const terminalFontSize = 14.0;
const terminalCellWidth = 8.44;
const terminalRowHeight = 20.0;
