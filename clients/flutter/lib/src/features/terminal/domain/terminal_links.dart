import '../../../generated/proto/apipb/history.pb.dart';
import '../../files/domain/file_path.dart';

final class TerminalLink {
  const TerminalLink({required this.url, required this.params});

  final String url;
  final String params;
}

final class TerminalPathLink {
  const TerminalPathLink({
    required this.raw,
    required this.path,
    required this.startColumn,
    required this.endColumn,
  });

  final String raw;
  final String path;
  final int startColumn;
  final int endColumn;
}

final class TerminalPathLayout {
  TerminalPathLayout._(this._pathsByRow);

  factory TerminalPathLayout.screenRows(List<ScreenRow> rows) =>
      TerminalPathLayout._build(
        rows.map((row) => (row: row, wrapped: row.wrapped)).toList(),
      );

  factory TerminalPathLayout.historyRows(List<HistoryRow> rows) =>
      TerminalPathLayout._build(
        rows
            .map(
              (row) => (row: row.row, wrapped: row.wrapped || row.row.wrapped),
            )
            .toList(),
      );

  factory TerminalPathLayout._build(
    List<({ScreenRow row, bool wrapped})> rows,
  ) {
    final pathsByRow = List<List<TerminalPathLink>>.generate(
      rows.length,
      (_) => <TerminalPathLink>[],
    );
    var firstRow = 0;
    while (firstRow < rows.length) {
      var lastRow = firstRow;
      while (lastRow < rows.length - 1 && rows[lastRow].wrapped) {
        lastRow += 1;
      }
      _projectTerminalLogicalLine(
        rows,
        firstRow: firstRow,
        lastRow: lastRow,
        pathsByRow: pathsByRow,
      );
      firstRow = lastRow + 1;
    }
    return TerminalPathLayout._(
      List<List<TerminalPathLink>>.unmodifiable(
        pathsByRow.map(List<TerminalPathLink>.unmodifiable),
      ),
    );
  }

  final List<List<TerminalPathLink>> _pathsByRow;

  List<TerminalPathLink> pathsInRow(int row) =>
      row >= 0 && row < _pathsByRow.length
      ? _pathsByRow[row]
      : const <TerminalPathLink>[];

  TerminalPathLink? pathAt(int row, int column) {
    if (column < 0) return null;
    for (final path in pathsInRow(row)) {
      if (column >= path.startColumn && column < path.endColumn) return path;
    }
    return null;
  }
}

final _terminalPathCache = Expando<List<TerminalPathLink>>('terminal-paths');
final _terminalTokenPattern = RegExp(r'''(?:"[^"\r\n]+"|'[^'\r\n]+'|[^\s]+)''');
final _terminalLocationSuffix = RegExp(r':\d+(?::\d+)?$');
final _terminalScheme = RegExp(r'^[A-Za-z][A-Za-z0-9+.-]*://');
final _windowsAbsolutePath = RegExp(r'^[A-Za-z]:[\\/]');
final _knownFileName = RegExp(
  r'^(?:Dockerfile|Makefile|README|LICENSE|CHANGELOG|\.env)(?:\.[A-Za-z0-9_-]+)?$',
  caseSensitive: false,
);
const _knownFileExtensions = {
  'bash',
  'c',
  'cc',
  'cfg',
  'conf',
  'cpp',
  'css',
  'csv',
  'dart',
  'env',
  'fish',
  'go',
  'gradle',
  'h',
  'hpp',
  'html',
  'ini',
  'java',
  'js',
  'json',
  'jsx',
  'kt',
  'kts',
  'lock',
  'log',
  'm',
  'md',
  'mm',
  'php',
  'plist',
  'proto',
  'py',
  'rb',
  'rs',
  'scss',
  'sh',
  'sql',
  'svelte',
  'swift',
  'toml',
  'ts',
  'tsx',
  'txt',
  'vue',
  'xml',
  'yaml',
  'yml',
  'zsh',
};

TerminalLink? terminalLinkAtColumn(ScreenRow row, int column) {
  if (column < 0) return null;
  var current = 0;
  for (final cell in row.cells) {
    final width = cell.width > 0 ? cell.width : 0;
    if (column >= current && column < current + width) {
      final url = cell.linkUrl.trim();
      return url.isEmpty
          ? null
          : TerminalLink(url: url, params: cell.linkParams);
    }
    current += width;
  }
  return null;
}

List<TerminalPathLink> terminalPathsInRow(ScreenRow row) {
  final cached = _terminalPathCache[row];
  if (cached != null) return cached;
  final result = TerminalPathLayout.screenRows([row]).pathsInRow(0);
  _terminalPathCache[row] = result;
  return result;
}

TerminalPathLink? terminalPathAtColumn(ScreenRow row, int column) {
  if (column < 0) return null;
  for (final path in terminalPathsInRow(row)) {
    if (column >= path.startColumn && column < path.endColumn) return path;
  }
  return null;
}

String? resolveTerminalPath(String value, {required String cwd}) {
  var candidate = value.trim();
  if (candidate.length >= 2 &&
      ((candidate.startsWith('"') && candidate.endsWith('"')) ||
          (candidate.startsWith("'") && candidate.endsWith("'")))) {
    candidate = candidate.substring(1, candidate.length - 1);
  }
  candidate = candidate.replaceFirst(_terminalLocationSuffix, '');
  if (candidate.startsWith('file://')) {
    final uri = Uri.tryParse(candidate);
    if (uri == null || uri.scheme != 'file') return null;
    try {
      candidate = Uri.decodeComponent(uri.path);
    } on FormatException {
      return null;
    }
    if (uri.host.isNotEmpty) candidate = '//${uri.host}$candidate';
    if (RegExp(r'^/[A-Za-z]:/').hasMatch(candidate)) {
      candidate = candidate.substring(1);
    }
  }
  candidate = candidate.replaceAll('\\', '/');
  if (candidate.isEmpty || candidate.contains('\u0000')) return null;
  if (candidate == '~' ||
      candidate.startsWith('~/') ||
      candidate.startsWith(r'$')) {
    return null;
  }
  if (_isAbsoluteTerminalPath(candidate)) {
    return _cleanTerminalPath(candidate);
  }
  final base = cwd.trim().replaceAll('\\', '/');
  if (!_isAbsoluteTerminalPath(base)) return null;
  return _cleanTerminalPath(joinFilePath(base, candidate));
}

List<String> resolveTerminalPathCandidates(
  String value, {
  required String foregroundCwd,
  required String liveCwd,
  required String terminalCwd,
}) {
  final result = <String>[];
  final seen = <String>{};
  for (final cwd in [foregroundCwd, liveCwd, terminalCwd]) {
    final resolved = resolveTerminalPath(value, cwd: cwd);
    if (resolved != null && seen.add(resolved)) result.add(resolved);
    if (resolved != null && _isAbsoluteTerminalPath(value)) break;
  }
  return List<String>.unmodifiable(result);
}

({String raw, String path, int start, int end})? _terminalPathCandidate(
  String line,
  int matchStart,
  int matchEnd,
) {
  var start = matchStart;
  var end = matchEnd;
  var token = line.substring(start, end);
  if (token.length >= 2 &&
      ((token.startsWith('"') && token.endsWith('"')) ||
          (token.startsWith("'") && token.endsWith("'")))) {
    start += 1;
    end -= 1;
    token = line.substring(start, end);
  }
  final markdownTarget = token.indexOf('](');
  if (markdownTarget >= 0 && token.endsWith(')')) {
    final suffix = token.substring(markdownTarget + 2, token.length - 1);
    if (_looksLikeTerminalPath(suffix)) {
      start += markdownTarget + 2;
      end -= 1;
      token = suffix;
    }
  }
  while (token.isNotEmpty && '([{<'.contains(token[0])) {
    start += 1;
    token = line.substring(start, end);
  }
  while (token.isNotEmpty && ',;!?)]}>'.contains(token[token.length - 1])) {
    end -= 1;
    token = line.substring(start, end);
  }
  if (token.endsWith('.') && token != '.' && token != '..') {
    end -= 1;
    token = line.substring(start, end);
  }
  final labeledPathOffset = _labeledPathOffset(token);
  if (labeledPathOffset > 0) {
    start += labeledPathOffset;
    token = token.substring(labeledPathOffset);
  }
  final equals = token.lastIndexOf('=');
  if (equals > 0) {
    final suffix = token.substring(equals + 1);
    if (_looksLikeTerminalPath(suffix)) {
      start += equals + 1;
      token = suffix;
    }
  }
  if (!_looksLikeTerminalPath(token)) return null;
  final path = token.replaceFirst(_terminalLocationSuffix, '');
  if (path.isEmpty || path.length > 4096) return null;
  return (raw: token, path: path, start: start, end: end);
}

bool _looksLikeTerminalPath(String value) {
  if (value.isEmpty || value == '.' || value == '..') return false;
  if (_terminalScheme.hasMatch(value) && !value.startsWith('file://')) {
    return false;
  }
  final path = value.replaceFirst(_terminalLocationSuffix, '');
  if (path.contains(':') &&
      !path.startsWith('file://') &&
      !_windowsAbsolutePath.hasMatch(path)) {
    return false;
  }
  if (path.startsWith('/') ||
      path.startsWith('./') ||
      path.startsWith('../') ||
      path.startsWith('.\\') ||
      path.startsWith('..\\') ||
      path.startsWith('~/') ||
      path.startsWith('~\\') ||
      path.startsWith('\\\\') ||
      _windowsAbsolutePath.hasMatch(path) ||
      path.contains('/') ||
      path.contains('\\')) {
    return true;
  }
  if (_knownFileName.hasMatch(path)) return true;
  final dot = path.lastIndexOf('.');
  if (dot <= 0 || dot == path.length - 1) return false;
  return _knownFileExtensions.contains(path.substring(dot + 1).toLowerCase());
}

int _labeledPathOffset(String value) {
  if (_terminalScheme.hasMatch(value) ||
      value.startsWith('file://') ||
      _windowsAbsolutePath.hasMatch(value)) {
    return 0;
  }
  for (var offset = 1; offset < value.length - 1; offset += 1) {
    final separator = value.codeUnitAt(offset);
    if (separator != 0x3a && separator != 0xff1a) continue;
    final suffix = value.substring(offset + 1);
    if (_looksLikeTerminalPath(suffix)) return offset + 1;
  }
  return 0;
}

bool _isAbsoluteTerminalPath(String path) =>
    path.startsWith('/') ||
    path.startsWith('//') ||
    _windowsAbsolutePath.hasMatch(path);

String _cleanTerminalPath(String value) {
  final normalized = normalizeFilePath(value);
  String root;
  String remainder;
  final drive = RegExp(r'^([A-Za-z]:)/(.*)$').firstMatch(normalized);
  final unc = RegExp(r'^(//[^/]+/[^/]+)(?:/(.*))?$').firstMatch(normalized);
  if (drive != null) {
    root = '${drive.group(1)!}/';
    remainder = drive.group(2)!;
  } else if (unc != null) {
    root = unc.group(1)!;
    remainder = unc.group(2) ?? '';
  } else {
    root = '/';
    remainder = normalized.replaceFirst(RegExp(r'^/+'), '');
  }
  final segments = <String>[];
  for (final segment in remainder.split('/')) {
    if (segment.isEmpty || segment == '.') continue;
    if (segment == '..') {
      if (segments.isNotEmpty) segments.removeLast();
      continue;
    }
    segments.add(segment);
  }
  if (segments.isEmpty) return root;
  final prefix = root.endsWith('/') ? root : '$root/';
  return '$prefix${segments.join('/')}';
}

_TerminalRowProjection _projectTerminalRow(ScreenRow row) {
  final text = StringBuffer();
  final columns = <int>[];
  var column = 0;
  for (final cell in row.cells) {
    final width = cell.width > 0 ? cell.width : 0;
    if (width == 0) continue;
    final runes = cell.content.runes.toList(growable: false);
    if (runes.isEmpty) {
      for (var offset = 0; offset < width; offset += 1) {
        columns.add(column + offset);
        text.write(' ');
      }
      column += width;
      continue;
    }
    for (var index = 0; index < runes.length; index += 1) {
      final content = String.fromCharCode(runes[index]);
      final start = column + (index * width / runes.length).floor();
      for (var unit = 0; unit < content.length; unit += 1) {
        columns.add(start);
      }
      text.write(content);
    }
    column += width;
  }
  columns.add(column);
  return _TerminalRowProjection(text.toString(), columns);
}

void _projectTerminalLogicalLine(
  List<({ScreenRow row, bool wrapped})> rows, {
  required int firstRow,
  required int lastRow,
  required List<List<TerminalPathLink>> pathsByRow,
}) {
  final text = StringBuffer();
  final projections = <_LogicalRowProjection>[];
  for (var rowIndex = firstRow; rowIndex <= lastRow; rowIndex += 1) {
    final projection = _projectTerminalRow(rows[rowIndex].row);
    final start = text.length;
    text.write(projection.text);
    projections.add(
      _LogicalRowProjection(
        row: rowIndex,
        start: start,
        end: text.length,
        projection: projection,
      ),
    );
  }
  final line = text.toString();
  for (final match in _terminalTokenPattern.allMatches(line)) {
    final candidate = _terminalPathCandidate(line, match.start, match.end);
    if (candidate == null) continue;
    for (final row in projections) {
      final start = candidate.start.clamp(row.start, row.end);
      final end = candidate.end.clamp(row.start, row.end);
      if (end <= start) continue;
      pathsByRow[row.row].add(
        TerminalPathLink(
          raw: candidate.raw,
          path: candidate.path,
          startColumn: row.projection.columnAt(start - row.start),
          endColumn: row.projection.columnAt(end - row.start),
        ),
      );
    }
  }
}

final class _TerminalRowProjection {
  const _TerminalRowProjection(this.text, this.columns);

  final String text;
  final List<int> columns;

  int columnAt(int offset) => columns[offset.clamp(0, columns.length - 1)];
}

final class _LogicalRowProjection {
  const _LogicalRowProjection({
    required this.row,
    required this.start,
    required this.end,
    required this.projection,
  });

  final int row;
  final int start;
  final int end;
  final _TerminalRowProjection projection;
}

Uri? externalTerminalLink(String value) {
  final uri = Uri.tryParse(value.trim());
  if (uri == null || uri.scheme.isEmpty) return null;
  return const {
        'http',
        'https',
        'mailto',
        'tel',
      }.contains(uri.scheme.toLowerCase())
      ? uri
      : null;
}
