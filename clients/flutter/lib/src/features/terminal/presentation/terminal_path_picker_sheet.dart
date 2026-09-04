import 'dart:async';

import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';
import '../../files/domain/file_path.dart';

typedef TerminalDirectoryLoader = Future<PathListDirectoriesResult> Function(
  String prefix,
);

Future<String?> showTerminalPathPicker({
  required BuildContext context,
  required String initialPath,
  required TerminalDirectoryLoader loadDirectories,
}) {
  return showModalBottomSheet<String>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    isScrollControlled: true,
    showDragHandle: false,
    backgroundColor: Colors.transparent,
    barrierColor: Colors.black54,
    builder: (context) => TerminalPathPickerSheet(
      initialPath: initialPath,
      loadDirectories: loadDirectories,
    ),
  );
}

final class TerminalPathPickerSheet extends StatefulWidget {
  const TerminalPathPickerSheet({
    super.key,
    required this.initialPath,
    required this.loadDirectories,
  });

  final String initialPath;
  final TerminalDirectoryLoader loadDirectories;

  @override
  State<TerminalPathPickerSheet> createState() =>
      _TerminalPathPickerSheetState();
}

final class _TerminalPathPickerSheetState
    extends State<TerminalPathPickerSheet> {
  late String _path;
  List<PathDirectoryEntry> _entries = const [];
  String? _error;
  bool _loading = true;
  var _requestEpoch = 0;

  @override
  void initState() {
    super.initState();
    _path = _normalizePickerPath(widget.initialPath);
    unawaited(_loadPath(_path));
  }

  Future<void> _loadPath(String path) async {
    final normalized = _normalizePickerPath(path);
    final epoch = ++_requestEpoch;
    setState(() {
      _path = normalized;
      _entries = const [];
      _error = null;
      _loading = true;
    });
    try {
      final result = await widget.loadDirectories(_directoryPrefix(normalized));
      if (!mounted || epoch != _requestEpoch) return;
      setState(() {
        _entries = result.entries.toList(growable: false);
        _error = result.missing
            ? anyttyText(
                context,
                en: 'This directory is unavailable',
                zh: '此目录不可用',
              )
            : null;
        _loading = false;
      });
    } catch (error) {
      if (!mounted || epoch != _requestEpoch) return;
      setState(() {
        _entries = const [];
        _error = error.toString();
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final entries = [..._entries]
      ..sort(
        (left, right) =>
            left.name.toLowerCase().compareTo(right.name.toLowerCase()),
      );
    final canGoUp = !_isPickerRoot(_path);
    return FractionallySizedBox(
      alignment: Alignment.bottomCenter,
      heightFactor: 0.82,
      child: Material(
        color: palette.background,
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
          side: BorderSide(color: palette.border),
        ),
        child: SafeArea(
          top: false,
          child: Column(
            children: [
              const SizedBox(height: 10),
              Container(
                width: 48,
                height: 4,
                decoration: BoxDecoration(
                  color: palette.borderStrong,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              SizedBox(
                height: 54,
                child: Row(
                  children: [
                    const SizedBox(width: 20),
                    Expanded(
                      child: Text(
                        anyttyText(context, en: 'Choose directory', zh: '选择目录'),
                        style: TextStyle(
                          color: palette.text,
                          fontSize: 17,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: anyttyText(
                        context,
                        en: 'Close directory picker',
                        zh: '关闭目录选择器',
                      ),
                      onPressed: () => Navigator.pop(context),
                      icon: const Icon(Icons.close_rounded, size: 20),
                    ),
                    const SizedBox(width: 8),
                  ],
                ),
              ),
              Divider(height: 1, color: palette.border),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      anyttyText(context, en: 'Current directory', zh: '当前目录'),
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Container(
                      constraints: const BoxConstraints(minHeight: 48),
                      alignment: Alignment.centerLeft,
                      padding: const EdgeInsets.symmetric(horizontal: 14),
                      decoration: BoxDecoration(
                        color: palette.surface,
                        border: Border.all(color: palette.borderStrong),
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        _path,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: palette.text,
                          fontFamily: 'JetBrainsMonoNerd',
                          fontSize: 13,
                        ),
                      ),
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        Expanded(
                          child: SizedBox(
                            height: 48,
                            child: OutlinedButton.icon(
                              onPressed: canGoUp
                                  ? () => _loadPath(parentFilePath(_path))
                                  : null,
                              icon: const Icon(LucideIcons.chevronLeft),
                              label: Text(
                                anyttyText(context, en: 'Parent', zh: '上级目录'),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: SizedBox(
                            height: 48,
                            child: FilledButton.icon(
                              onPressed: () => Navigator.pop(context, _path),
                              icon: const Icon(LucideIcons.folderOpen),
                              label: Text(
                                anyttyText(
                                  context,
                                  en: 'Use this path',
                                  zh: '使用此路径',
                                ),
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              if (_error case final error?)
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                  child: Material(
                    color: palette.warning.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(6),
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              error,
                              style: TextStyle(
                                color: palette.text,
                                fontSize: 13,
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ),
                          IconButton(
                            tooltip: anyttyText(context, en: 'Retry', zh: '重试'),
                            onPressed: () => _loadPath(_path),
                            icon: const Icon(Icons.refresh_rounded),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              Expanded(
                child: _loading
                    ? const Center(child: CircularProgressIndicator())
                    : entries.isEmpty
                    ? Center(
                        child: Text(
                          anyttyText(
                            context,
                            en: 'No subdirectories',
                            zh: '没有子目录',
                          ),
                          style: TextStyle(color: palette.muted, fontSize: 13),
                        ),
                      )
                    : ListView.separated(
                        padding: const EdgeInsets.fromLTRB(16, 0, 16, 20),
                        itemCount: entries.length,
                        separatorBuilder: (_, _) =>
                            Divider(height: 1, color: palette.border),
                        itemBuilder: (context, index) {
                          final entry = entries[index];
                          return SizedBox(
                            height: 52,
                            child: ListTile(
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 8,
                              ),
                              leading: Icon(
                                LucideIcons.folder,
                                size: 18,
                                color: palette.muted,
                              ),
                              title: Text(
                                entry.name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: TextStyle(
                                  color: palette.text,
                                  fontSize: 14,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              trailing: Icon(
                                LucideIcons.chevronRight,
                                size: 17,
                                color: palette.faint,
                              ),
                              onTap: () =>
                                  _loadPath(normalizeFilePath(entry.path)),
                            ),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

String _normalizePickerPath(String path) {
  return normalizeFilePath(path);
}

String _directoryPrefix(String path) {
  final normalized = _normalizePickerPath(path);
  if (normalized == '/' || RegExp(r'^[A-Za-z]:/$').hasMatch(normalized)) {
    return normalized;
  }
  return '$normalized/';
}

bool _isPickerRoot(String path) {
  return path == '/' || path == '~' || RegExp(r'^[A-Za-z]:/$').hasMatch(path);
}
