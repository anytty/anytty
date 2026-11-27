part of 'file_manager_screen.dart';


enum _FileEntryAction { preview, download, copyPath, copy, cut, rename, delete }

enum _DirectoryAction {
  refresh,
  hidden,
  transfers,
  sortName,
  sortModified,
  sortSize,
  sortType,
}

bool _isDirectory(FileEntry entry) =>
    entry.type == FileEntryType.FILE_ENTRY_TYPE_DIRECTORY;

IconData _fileIcon(String name) {
  final extension = name.contains('.')
      ? name.split('.').last.toLowerCase()
      : '';
  return switch (extension) {
    'png' || 'jpg' || 'jpeg' || 'gif' || 'webp' => Icons.image_outlined,
    'mp4' || 'mov' || 'webm' => Icons.play_circle_outline_rounded,
    'md' ||
    'txt' ||
    'log' ||
    'json' ||
    'yaml' ||
    'yml' => Icons.description_outlined,
    _ => Icons.insert_drive_file_outlined,
  };
}

String _fileMeta(FileEntry entry) {
  final parts = <String>[
    if (_isDirectory(entry)) 'Folder' else _formatBytes(entry.size.toInt()),
  ];
  if (entry.modifiedAtUnixNano.toInt() > 0) {
    final modified = DateTime.fromMicrosecondsSinceEpoch(
      entry.modifiedAtUnixNano.toInt() ~/ 1000,
      isUtc: true,
    ).toLocal();
    parts.add(
      '${modified.year.toString().padLeft(4, '0')}-${modified.month.toString().padLeft(2, '0')}-${modified.day.toString().padLeft(2, '0')} '
      '${modified.hour.toString().padLeft(2, '0')}:${modified.minute.toString().padLeft(2, '0')}',
    );
  }
  if (entry.linkTarget.isNotEmpty) parts.add('-> ${entry.linkTarget}');
  return parts.join(' · ');
}

String _formatBytes(int bytes) {
  if (bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  var value = bytes.toDouble();
  var unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return '${value >= 10 || unit == 0 ? value.toStringAsFixed(0) : value.toStringAsFixed(1)} ${units[unit]}';
}

String _sortLabel(_FileSort sort) => switch (sort) {
  _FileSort.name => 'name',
  _FileSort.modified => 'modified',
  _FileSort.size => 'size',
  _FileSort.type => 'type',
};

IconData _sortIcon(_FileSort sort) => switch (sort) {
  _FileSort.name => Icons.sort_by_alpha_rounded,
  _FileSort.modified => Icons.schedule_rounded,
  _FileSort.size => Icons.data_usage_rounded,
  _FileSort.type => Icons.category_outlined,
};

final class _FileToolbarButton extends StatelessWidget {
  const _FileToolbarButton({
    required this.icon,
    required this.label,
    required this.onPressed,
    this.danger = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final bool danger;

  @override
  Widget build(BuildContext context) => Expanded(
    child: IconButton(
      tooltip: label,
      onPressed: onPressed,
      style: IconButton.styleFrom(
        minimumSize: const Size(44, 52),
        foregroundColor: danger ? const Color(0xffdc2626) : null,
      ),
      icon: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 20),
          const SizedBox(height: 2),
          Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 10.5),
          ),
        ],
      ),
    ),
  );
}
