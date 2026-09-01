final class FileBreadcrumb {
  const FileBreadcrumb({required this.label, required this.path});

  final String label;
  final String path;

  @override
  bool operator ==(Object other) =>
      other is FileBreadcrumb && other.label == label && other.path == path;

  @override
  int get hashCode => Object.hash(label, path);
}

String normalizeFilePath(String path) {
  final normalized = path.trim().replaceAll('\\', '/');
  if (normalized.isEmpty || normalized == '/') return '/';
  if (RegExp(r'^[A-Za-z]:/+$').hasMatch(normalized)) {
    return '${normalized.substring(0, 2)}/';
  }
  return normalized.replaceFirst(RegExp(r'/+$'), '');
}

String parentFilePath(String path) {
  final normalized = normalizeFilePath(path);
  if (normalized == '/' || RegExp(r'^[A-Za-z]:/$').hasMatch(normalized)) {
    return normalized;
  }
  final uncRoot = RegExp(r'^(//[^/]+/[^/]+)(?:/|$)').firstMatch(normalized);
  if (uncRoot?.group(1) == normalized) return normalized;
  final index = normalized.lastIndexOf('/');
  if (index == 2 && RegExp(r'^[A-Za-z]:').hasMatch(normalized)) {
    return '${normalized.substring(0, 2)}/';
  }
  if (index <= 0) return '/';
  return normalized.substring(0, index);
}

String joinFilePath(String base, String name) {
  final normalizedBase = normalizeFilePath(base);
  final child = name.replaceAll('\\', '/').replaceFirst(RegExp(r'^/+'), '');
  if (normalizedBase == '/') return '/$child';
  return '${normalizedBase.replaceFirst(RegExp(r'/+$'), '')}/$child';
}

List<FileBreadcrumb> filePathBreadcrumbs(String path) {
  final normalized = normalizeFilePath(path);
  final drive = RegExp(r'^([A-Za-z]:)(?:/(.*))?$').firstMatch(normalized);
  if (drive != null) {
    final driveName = drive.group(1)!;
    return <FileBreadcrumb>[
      const FileBreadcrumb(label: '/', path: '/'),
      ..._breadcrumbsFromSegments('$driveName/', driveName, drive.group(2)),
    ];
  }

  final unc = RegExp(r'^(//[^/]+/[^/]+)(?:/(.*))?$').firstMatch(normalized);
  if (unc != null) {
    return _breadcrumbsFromSegments(unc.group(1)!, unc.group(1)!, unc.group(2));
  }

  return _breadcrumbsFromSegments(
    '/',
    '/',
    normalized.replaceFirst(RegExp(r'^/+'), ''),
  );
}

List<FileBreadcrumb> _breadcrumbsFromSegments(
  String rootPath,
  String rootLabel,
  String? remainder,
) {
  final breadcrumbs = <FileBreadcrumb>[
    FileBreadcrumb(label: rootLabel, path: rootPath),
  ];
  var current = rootPath.replaceFirst(RegExp(r'/+$'), '');
  for (final segment
      in remainder?.split('/').where((part) => part.isNotEmpty) ??
          const <String>[]) {
    current = '$current/$segment';
    breadcrumbs.add(FileBreadcrumb(label: segment, path: current));
  }
  return breadcrumbs;
}

String fileBasename(String path) {
  final normalized = path
      .replaceAll('\\', '/')
      .replaceFirst(RegExp(r'/+$'), '');
  final index = normalized.lastIndexOf('/');
  return index >= 0 ? normalized.substring(index + 1) : normalized;
}
