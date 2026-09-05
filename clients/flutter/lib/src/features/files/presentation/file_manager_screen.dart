import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/apipb/file.pb.dart';
import '../domain/file_preview_safety.dart';
import '../domain/file_path.dart';
import 'file_transfer_controller.dart';
import 'file_transfer_sheet.dart';
import 'path_bookmarks_sheet.dart';

Future<void> showAnyttyFileManager({
  required BuildContext context,
  required String endpointId,
  required String endpointLabel,
  required String initialPath,
}) {
  return Navigator.of(context).push<void>(
    PageRouteBuilder<void>(
      opaque: true,
      transitionDuration: Duration.zero,
      reverseTransitionDuration: Duration.zero,
      pageBuilder: (context, _, _) => FileManagerScreen(
        endpointId: endpointId,
        endpointLabel: endpointLabel,
        initialPath: initialPath,
      ),
      transitionsBuilder: (context, animation, _, child) => child,
    ),
  );
}

enum _FileSort { name, modified, size, type }

enum _FileClipboardMode { copy, cut }

final class _FileClipboard {
  const _FileClipboard(this.mode, this.paths);

  final _FileClipboardMode mode;
  final List<String> paths;
}

final class FileManagerScreen extends ConsumerStatefulWidget {
  const FileManagerScreen({
    super.key,
    required this.endpointId,
    required this.endpointLabel,
    required this.initialPath,
  });

  final String endpointId;
  final String endpointLabel;
  final String initialPath;

  @override
  ConsumerState<FileManagerScreen> createState() => _FileManagerScreenState();
}

final class _FileManagerScreenState extends ConsumerState<FileManagerScreen> {
  final _scrollController = ScrollController();
  final _newDirectoryController = TextEditingController();
  final _renameController = TextEditingController();
  late final FileTransferController _transfers;
  List<FileEntry> _entries = const [];
  String _currentPath = '/';
  String _nextCursor = '';
  String? _error;
  String? _renamingPath;
  bool _loading = true;
  bool _loadingMore = false;
  bool _operationPending = false;
  bool _newDirectoryOpen = false;
  bool _showHidden = false;
  bool _selectionMode = false;
  _FileSort _sort = _FileSort.name;
  bool _sortAscending = true;
  Set<String> _selectedPaths = const {};
  _FileClipboard? _clipboard;
  var _requestEpoch = 0;
  var _handledUploadCompletionRevision = 0;

  @override
  void initState() {
    super.initState();
    _currentPath = normalizeFilePath(
      widget.initialPath.trim().isEmpty ? '/' : widget.initialPath,
    );
    _scrollController.addListener(_handleScroll);
    _transfers = ref.read(fileTransferControllerProvider);
    _handledUploadCompletionRevision = _transfers.uploadCompletionRevision(
      widget.endpointId,
    );
    _transfers.addListener(_handleTransfersChanged);
    unawaited(_loadPath(_currentPath));
  }

  @override
  void dispose() {
    _requestEpoch += 1;
    _scrollController
      ..removeListener(_handleScroll)
      ..dispose();
    _newDirectoryController.dispose();
    _renameController.dispose();
    _transfers.removeListener(_handleTransfersChanged);
    super.dispose();
  }

  void _handleTransfersChanged() {
    final revision = _transfers.uploadCompletionRevision(widget.endpointId);
    if (revision <= _handledUploadCompletionRevision) return;
    _handledUploadCompletionRevision = revision;
    if (mounted) unawaited(_loadPath(_currentPath));
  }

  void _handleScroll() {
    if (!_scrollController.hasClients ||
        _nextCursor.isEmpty ||
        _loadingMore ||
        _loading) {
      return;
    }
    if (_scrollController.position.extentAfter < 320) {
      unawaited(_loadMore());
    }
  }

  Future<void> _loadPath(String path) async {
    final normalized = normalizeFilePath(path);
    final epoch = ++_requestEpoch;
    setState(() {
      _loading = true;
      _error = null;
      _selectionMode = false;
      _selectedPaths = const {};
    });
    try {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      final page = await session.listFiles(path: normalized, limit: 250);
      if (!mounted || epoch != _requestEpoch) return;
      setState(() {
        _currentPath = normalizeFilePath(
          page.path.isEmpty ? normalized : page.path,
        );
        _entries = page.entries.map((entry) => entry.deepCopy()).toList();
        _nextCursor = page.nextCursor;
        _loading = false;
      });
      if (_scrollController.hasClients) _scrollController.jumpTo(0);
    } catch (error) {
      if (!mounted || epoch != _requestEpoch) return;
      setState(() {
        _loading = false;
        _error = error.toString();
      });
    }
  }

  Future<void> _loadMore() async {
    if (_nextCursor.isEmpty || _loadingMore) return;
    final epoch = _requestEpoch;
    final cursor = _nextCursor;
    setState(() => _loadingMore = true);
    try {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      final page = await session.listFiles(
        path: _currentPath,
        cursor: cursor,
        limit: 250,
      );
      if (!mounted || epoch != _requestEpoch) return;
      final known = _entries.map(_entryPath).toSet();
      setState(() {
        _entries = [
          ..._entries,
          ...page.entries
              .where((entry) => known.add(_entryPath(entry)))
              .map((entry) => entry.deepCopy()),
        ];
        _nextCursor = page.nextCursor == cursor ? '' : page.nextCursor;
      });
    } catch (error) {
      if (mounted && epoch == _requestEpoch) _showError(error);
    } finally {
      if (mounted && epoch == _requestEpoch) {
        setState(() => _loadingMore = false);
      }
    }
  }

  String _entryPath(FileEntry entry) => normalizeFilePath(
    entry.path.isEmpty ? joinFilePath(_currentPath, entry.name) : entry.path,
  );

  List<FileEntry> get _visibleEntries {
    final entries = _entries
        .where((entry) => _showHidden || !entry.name.startsWith('.'))
        .toList(growable: false);
    entries.sort((left, right) {
      final leftDirectory = _isDirectory(left);
      final rightDirectory = _isDirectory(right);
      if (leftDirectory != rightDirectory) return leftDirectory ? -1 : 1;
      final comparison = switch (_sort) {
        _FileSort.modified => left.modifiedAtUnixNano.compareTo(
          right.modifiedAtUnixNano,
        ),
        _FileSort.size => left.size.compareTo(right.size),
        _FileSort.type => left.type.value.compareTo(right.type.value),
        _FileSort.name => left.name.toLowerCase().compareTo(
          right.name.toLowerCase(),
        ),
      };
      final result = comparison == 0
          ? left.name.toLowerCase().compareTo(right.name.toLowerCase())
          : comparison;
      return _sortAscending ? result : -result;
    });
    return entries;
  }

  bool _handleBack() {
    if (_newDirectoryOpen) {
      setState(() {
        _newDirectoryOpen = false;
        _newDirectoryController.clear();
      });
      return true;
    }
    if (_renamingPath != null) {
      setState(() {
        _renamingPath = null;
        _renameController.clear();
      });
      return true;
    }
    if (_selectionMode) {
      setState(() {
        _selectionMode = false;
        _selectedPaths = const {};
      });
      return true;
    }
    if (_clipboard != null) {
      setState(() => _clipboard = null);
      return true;
    }
    if (parentFilePath(_currentPath) != _currentPath) {
      unawaited(_loadPath(parentFilePath(_currentPath)));
      return true;
    }
    return false;
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return PopScope<Object?>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && !_handleBack()) Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: palette.background,
        appBar: AppBar(
          automaticallyImplyLeading: false,
          backgroundColor: palette.background,
          foregroundColor: palette.text,
          leading: IconButton(
            tooltip: 'Close files',
            onPressed: () => Navigator.of(context).pop(),
            icon: const Icon(Icons.chevron_left_rounded),
          ),
          titleSpacing: 0,
          title: Row(
            children: [
              Icon(Icons.folder_rounded, size: 20, color: palette.muted),
              const SizedBox(width: 8),
              const Text(
                'Files',
                style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
        body: Column(
          children: [
            if (_selectionMode)
              _buildSelectionHeader(palette)
            else
              _buildPathBar(palette),
            Expanded(child: _buildDirectoryBody(palette)),
            _buildBottomToolbar(palette),
          ],
        ),
      ),
    );
  }

  Widget _buildPathBar(AnyttyPalette palette) {
    final breadcrumbs = filePathBreadcrumbs(_currentPath);
    return Container(
      decoration: BoxDecoration(
        color: palette.background,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      child: Container(
        height: 50,
        decoration: BoxDecoration(
          color: palette.surface,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(6),
        ),
        child: ListView.separated(
          padding: const EdgeInsets.symmetric(horizontal: 8),
          scrollDirection: Axis.horizontal,
          itemCount: breadcrumbs.length + 1,
          separatorBuilder: (context, index) =>
              Icon(Icons.chevron_right_rounded, size: 16, color: palette.faint),
          itemBuilder: (context, index) {
            if (index == 0) {
              return Icon(
                Icons.storage_rounded,
                size: 17,
                color: palette.muted,
              );
            }
            final breadcrumb = breadcrumbs[index - 1];
            final active = index == breadcrumbs.length;
            return TextButton(
              onPressed: active ? null : () => _loadPath(breadcrumb.path),
              style: TextButton.styleFrom(
                minimumSize: const Size(24, 48),
                padding: const EdgeInsets.symmetric(horizontal: 5),
                foregroundColor: active ? palette.text : palette.muted,
                disabledForegroundColor: palette.text,
              ),
              child: Text(
                breadcrumb.label,
                style: TextStyle(
                  fontSize: 14,
                  fontWeight: active ? FontWeight.w600 : FontWeight.w500,
                ),
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildSelectionHeader(AnyttyPalette palette) => Container(
    height: 48,
    decoration: BoxDecoration(
      border: Border(bottom: BorderSide(color: palette.border)),
    ),
    child: Row(
      children: [
        TextButton(
          onPressed: () => setState(() {
            _selectionMode = false;
            _selectedPaths = const {};
          }),
          child: const Text('Cancel'),
        ),
        Expanded(
          child: Text(
            '${_selectedPaths.length} selected',
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
          ),
        ),
        TextButton(
          onPressed: () {
            final paths = _visibleEntries.map(_entryPath).toSet();
            setState(() {
              _selectedPaths = _selectedPaths.length == paths.length
                  ? const {}
                  : paths;
            });
          },
          child: Text(
            _selectedPaths.length == _visibleEntries.length
                ? 'Deselect all'
                : 'Select all',
          ),
        ),
      ],
    ),
  );

  Widget _buildDirectoryBody(AnyttyPalette palette) {
    if (_loading && _entries.isEmpty) {
      return const Center(child: CircularProgressIndicator(strokeWidth: 2));
    }
    if (_error case final error? when _entries.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.error_outline_rounded,
                color: palette.danger,
                size: 32,
              ),
              const SizedBox(height: 12),
              Text(
                error,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: TextStyle(color: palette.muted, fontSize: 13),
              ),
              const SizedBox(height: 12),
              OutlinedButton.icon(
                onPressed: () => _loadPath(_currentPath),
                icon: const Icon(Icons.refresh_rounded),
                label: const Text('Retry'),
              ),
            ],
          ),
        ),
      );
    }

    final entries = _visibleEntries;
    return RefreshIndicator(
      onRefresh: () => _loadPath(_currentPath),
      child: ListView.builder(
        controller: _scrollController,
        padding: const EdgeInsets.only(bottom: 12),
        itemCount:
            entries.length +
            (_newDirectoryOpen ? 1 : 0) +
            (_loadingMore ? 1 : 0) +
            (entries.isEmpty && !_newDirectoryOpen ? 1 : 0),
        itemBuilder: (context, index) {
          if (_newDirectoryOpen) {
            if (index == 0) return _buildNewDirectoryRow(palette);
            index -= 1;
          }
          if (entries.isEmpty) {
            return SizedBox(
              height: 160,
              child: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.folder_outlined, color: palette.faint, size: 34),
                    const SizedBox(height: 10),
                    Text(
                      'Empty directory',
                      style: TextStyle(color: palette.muted),
                    ),
                  ],
                ),
              ),
            );
          }
          if (index >= entries.length) {
            return const SizedBox(
              height: 56,
              child: Center(
                child: SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            );
          }
          return _buildEntryRow(entries[index], palette);
        },
      ),
    );
  }

  Widget _buildNewDirectoryRow(AnyttyPalette palette) => Container(
    margin: const EdgeInsets.all(8),
    padding: const EdgeInsets.all(8),
    decoration: BoxDecoration(
      color: palette.surface,
      border: Border.all(color: palette.border),
      borderRadius: BorderRadius.circular(6),
    ),
    child: Row(
      children: [
        Icon(Icons.folder_rounded, color: palette.text),
        const SizedBox(width: 8),
        Expanded(
          child: TextField(
            controller: _newDirectoryController,
            autofocus: true,
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _createDirectory(),
            decoration: const InputDecoration(
              hintText: 'Directory name',
              isDense: true,
              border: InputBorder.none,
            ),
          ),
        ),
        IconButton(
          tooltip: 'Create directory',
          onPressed: _operationPending ? null : _createDirectory,
          icon: const Icon(Icons.check_rounded),
        ),
        IconButton(
          tooltip: 'Cancel new directory',
          onPressed: () => setState(() {
            _newDirectoryOpen = false;
            _newDirectoryController.clear();
          }),
          icon: const Icon(Icons.close_rounded),
        ),
      ],
    ),
  );

  Widget _buildEntryRow(FileEntry entry, AnyttyPalette palette) {
    final path = _entryPath(entry);
    final directory = _isDirectory(entry);
    final selected = _selectedPaths.contains(path);
    final renaming = _renamingPath == path;
    return Container(
      constraints: const BoxConstraints(minHeight: 56),
      decoration: BoxDecoration(
        color: selected ? palette.surfaceRaised : null,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          if (_selectionMode)
            Padding(
              padding: const EdgeInsets.only(left: 12),
              child: Icon(
                selected
                    ? Icons.check_circle_rounded
                    : Icons.radio_button_unchecked_rounded,
                size: 24,
                color: selected ? palette.accent : palette.faint,
              ),
            ),
          Padding(
            padding: const EdgeInsets.only(left: 12),
            child: Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: palette.surface,
                border: Border.all(color: palette.border),
                borderRadius: BorderRadius.circular(6),
              ),
              child: Icon(
                directory ? Icons.folder_rounded : _fileIcon(entry.name),
                size: 21,
                color: directory ? palette.text : palette.muted,
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: renaming
                ? TextField(
                    controller: _renameController,
                    autofocus: true,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _commitRename(path),
                    decoration: const InputDecoration(
                      labelText: 'Rename entry',
                      isDense: true,
                    ),
                  )
                : InkWell(
                    onTap: () => _activateEntry(entry),
                    onLongPress: () => _startSelection(path),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            entry.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.text,
                              fontSize: 15,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            _fileMeta(entry),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 12,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
          ),
          if (renaming) ...[
            IconButton(
              tooltip: 'Save rename',
              onPressed: _operationPending ? null : () => _commitRename(path),
              icon: const Icon(Icons.check_rounded),
            ),
            IconButton(
              tooltip: 'Cancel rename',
              onPressed: () => setState(() => _renamingPath = null),
              icon: const Icon(Icons.close_rounded),
            ),
          ] else if (!_selectionMode) ...[
            IconButton(
              tooltip: 'More actions for ${entry.name}',
              onPressed: () => _openEntryActions(entry),
              icon: Icon(Icons.more_vert_rounded, color: palette.muted),
            ),
            SizedBox(
              width: 28,
              child: directory
                  ? Icon(Icons.chevron_right_rounded, color: palette.faint)
                  : null,
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildBottomToolbar(AnyttyPalette palette) {
    if (_clipboard case final clipboard?) {
      return SafeArea(
        top: false,
        child: Container(
          height: 58,
          decoration: BoxDecoration(
            border: Border(top: BorderSide(color: palette.border)),
          ),
          padding: const EdgeInsets.all(8),
          child: Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  onPressed: () => setState(() => _clipboard = null),
                  icon: const Icon(Icons.close_rounded),
                  label: const Text('Cancel'),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: FilledButton.icon(
                  onPressed: _operationPending
                      ? null
                      : () => _pasteClipboard(clipboard),
                  icon: const Icon(Icons.content_paste_rounded),
                  label: const Text('Paste'),
                ),
              ),
            ],
          ),
        ),
      );
    }

    if (_selectionMode && _selectedPaths.isNotEmpty) {
      return SafeArea(
        top: false,
        child: Container(
          height: 60,
          decoration: BoxDecoration(
            border: Border(top: BorderSide(color: palette.border)),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              _FileToolbarButton(
                icon: Icons.copy_rounded,
                label: 'Copy',
                onPressed: () => _setClipboard(_FileClipboardMode.copy),
              ),
              _FileToolbarButton(
                icon: Icons.drive_file_move_outline,
                label: 'Cut',
                onPressed: () => _setClipboard(_FileClipboardMode.cut),
              ),
              _FileToolbarButton(
                icon: Icons.content_copy_rounded,
                label: 'Path',
                onPressed: () => _copyPaths(_selectedPaths.toList()),
              ),
              _FileToolbarButton(
                icon: Icons.delete_outline_rounded,
                label: 'Delete',
                danger: true,
                onPressed: _confirmBatchDelete,
              ),
            ],
          ),
        ),
      );
    }

    return SafeArea(
      top: false,
      child: Container(
        height: 60,
        decoration: BoxDecoration(
          color: palette.background,
          border: Border(top: BorderSide(color: palette.border)),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _FileToolbarButton(
              icon: Icons.checklist_rounded,
              label: 'Select',
              onPressed: () => setState(() => _selectionMode = true),
            ),
            _FileToolbarButton(
              icon: Icons.create_new_folder_outlined,
              label: 'New',
              onPressed: () => setState(() => _newDirectoryOpen = true),
            ),
            _FileToolbarButton(
              icon: Icons.bookmark_border_rounded,
              label: 'Bookmarks',
              onPressed: _openBookmarks,
            ),
            _FileToolbarButton(
              icon: Icons.content_copy_rounded,
              label: 'Path',
              onPressed: () => _copyPaths([_currentPath]),
            ),
            _FileToolbarButton(
              icon: Icons.upload_rounded,
              label: 'Upload',
              onPressed: _pickAndUpload,
            ),
            _FileToolbarButton(
              icon: Icons.more_vert_rounded,
              label: 'More',
              onPressed: _openDirectoryActions,
            ),
          ],
        ),
      ),
    );
  }

  void _activateEntry(FileEntry entry) {
    final path = _entryPath(entry);
    if (_selectionMode) {
      setState(() {
        final selected = {..._selectedPaths};
        selected.contains(path) ? selected.remove(path) : selected.add(path);
        _selectedPaths = selected;
      });
      return;
    }
    HapticFeedback.selectionClick();
    if (_isDirectory(entry)) {
      unawaited(_loadPath(path));
    } else {
      unawaited(
        showModalBottomSheet<void>(
          context: context,
          sheetAnimationStyle: AnimationStyle.noAnimation,
          isScrollControlled: true,
          useSafeArea: true,
          builder: (context) => FilePreviewSheet(
            endpointId: widget.endpointId,
            entry: entry,
            path: path,
          ),
        ),
      );
    }
  }

  void _startSelection(String path) {
    HapticFeedback.mediumImpact();
    setState(() {
      _selectionMode = true;
      _selectedPaths = {..._selectedPaths, path};
    });
  }

  Future<void> _openEntryActions(FileEntry entry) async {
    final path = _entryPath(entry);
    final action = await showModalBottomSheet<_FileEntryAction>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (!_isDirectory(entry))
              ListTile(
                leading: const Icon(Icons.visibility_outlined),
                title: const Text('Preview'),
                onTap: () => Navigator.pop(context, _FileEntryAction.preview),
              ),
            if (!_isDirectory(entry))
              ListTile(
                leading: const Icon(Icons.download_rounded),
                title: const Text('Download'),
                onTap: () => Navigator.pop(context, _FileEntryAction.download),
              ),
            ListTile(
              leading: const Icon(Icons.content_copy_rounded),
              title: const Text('Copy path'),
              onTap: () => Navigator.pop(context, _FileEntryAction.copyPath),
            ),
            ListTile(
              leading: const Icon(Icons.copy_rounded),
              title: const Text('Copy'),
              onTap: () => Navigator.pop(context, _FileEntryAction.copy),
            ),
            ListTile(
              leading: const Icon(Icons.drive_file_move_outline),
              title: const Text('Cut'),
              onTap: () => Navigator.pop(context, _FileEntryAction.cut),
            ),
            ListTile(
              leading: const Icon(Icons.edit_outlined),
              title: const Text('Rename'),
              onTap: () => Navigator.pop(context, _FileEntryAction.rename),
            ),
            ListTile(
              leading: const Icon(Icons.delete_outline_rounded),
              title: const Text('Delete'),
              textColor: const Color(0xffdc2626),
              iconColor: const Color(0xffdc2626),
              onTap: () => Navigator.pop(context, _FileEntryAction.delete),
            ),
          ],
        ),
      ),
    );
    if (!mounted || action == null) return;
    switch (action) {
      case _FileEntryAction.preview:
        _activateEntry(entry);
      case _FileEntryAction.download:
        unawaited(
          _transfers.startDownload(
            endpointId: widget.endpointId,
            endpointLabel: widget.endpointLabel,
            remotePath: path,
            name: entry.name,
            size: entry.size.toInt(),
          ),
        );
        unawaited(showFileTransferCenter(context, _transfers));
      case _FileEntryAction.copyPath:
        unawaited(_copyPaths([path]));
      case _FileEntryAction.copy:
        setState(
          () => _clipboard = _FileClipboard(_FileClipboardMode.copy, [path]),
        );
      case _FileEntryAction.cut:
        setState(
          () => _clipboard = _FileClipboard(_FileClipboardMode.cut, [path]),
        );
      case _FileEntryAction.rename:
        _renameController.text = entry.name;
        setState(() => _renamingPath = path);
      case _FileEntryAction.delete:
        unawaited(_confirmDelete(path));
    }
  }

  Future<void> _openDirectoryActions() async {
    final action = await showModalBottomSheet<_DirectoryAction>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      showDragHandle: true,
      builder: (context) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.refresh_rounded),
              title: const Text('Refresh'),
              onTap: () => Navigator.pop(context, _DirectoryAction.refresh),
            ),
            ListTile(
              leading: Icon(
                _showHidden
                    ? Icons.visibility_off_outlined
                    : Icons.visibility_outlined,
              ),
              title: Text(
                _showHidden ? 'Hide hidden files' : 'Show hidden files',
              ),
              onTap: () => Navigator.pop(context, _DirectoryAction.hidden),
            ),
            if (_transfers.items.isNotEmpty)
              ListTile(
                leading: const Icon(Icons.swap_vert_rounded),
                title: const Text('Transfers'),
                onTap: () => Navigator.pop(context, _DirectoryAction.transfers),
              ),
            for (final sort in _FileSort.values)
              ListTile(
                leading: Icon(
                  _sort == sort ? Icons.check_rounded : _sortIcon(sort),
                ),
                title: Text('Sort by ${_sortLabel(sort)}'),
                trailing: _sort == sort
                    ? Icon(
                        _sortAscending
                            ? Icons.arrow_upward_rounded
                            : Icons.arrow_downward_rounded,
                      )
                    : null,
                onTap: () => Navigator.pop(
                  context,
                  _DirectoryAction.values[3 + sort.index],
                ),
              ),
          ],
        ),
      ),
    );
    if (!mounted || action == null) return;
    if (action == _DirectoryAction.refresh) {
      unawaited(_loadPath(_currentPath));
    } else if (action == _DirectoryAction.hidden) {
      setState(() => _showHidden = !_showHidden);
    } else if (action == _DirectoryAction.transfers) {
      unawaited(showFileTransferCenter(context, _transfers));
    } else {
      final sort = _FileSort.values[action.index - 3];
      setState(() {
        if (_sort == sort) {
          _sortAscending = !_sortAscending;
        } else {
          _sort = sort;
          _sortAscending = true;
        }
      });
    }
  }

  Future<void> _createDirectory() async {
    final name = _newDirectoryController.text.trim();
    if (name.isEmpty || _operationPending) return;
    final succeeded = await _runOperation(() async {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      await session.createDirectory(joinFilePath(_currentPath, name));
    }, success: 'Directory created');
    if (!mounted || !succeeded) return;
    setState(() {
      _newDirectoryOpen = false;
      _newDirectoryController.clear();
    });
  }

  Future<void> _commitRename(String path) async {
    final name = _renameController.text.trim();
    if (name.isEmpty || _operationPending) return;
    final succeeded = await _runOperation(() async {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      await session.renameFile(path, joinFilePath(parentFilePath(path), name));
    }, success: 'Entry renamed');
    if (mounted && succeeded) setState(() => _renamingPath = null);
  }

  Future<void> _confirmDelete(String path) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete entry?'),
        content: Text(path),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xffdc2626),
            ),
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _runOperation(() async {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      await session.deleteFile(path);
    }, success: 'Entry deleted');
  }

  Future<void> _confirmBatchDelete() async {
    final paths = _selectedPaths.toList(growable: false);
    if (paths.isEmpty) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text('Delete ${paths.length} entries?'),
        content: const Text('Directories and their contents will be removed.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: const Color(0xffdc2626),
            ),
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    final succeeded = await _runOperation(() async {
      final session = await ref.read(
        endpointSessionProvider(widget.endpointId).future,
      );
      for (final path in paths) {
        await session.deleteFile(path);
      }
    }, success: '${paths.length} entries deleted');
    if (mounted && succeeded) {
      setState(() {
        _selectionMode = false;
        _selectedPaths = const {};
      });
    }
  }

  void _setClipboard(_FileClipboardMode mode) {
    final paths = _selectedPaths.toList(growable: false);
    if (paths.isEmpty) return;
    setState(() {
      _clipboard = _FileClipboard(mode, paths);
      _selectionMode = false;
      _selectedPaths = const {};
    });
  }

  Future<void> _pasteClipboard(_FileClipboard clipboard) async {
    final succeeded = await _runOperation(
      () async {
        final session = await ref.read(
          endpointSessionProvider(widget.endpointId).future,
        );
        if (clipboard.mode == _FileClipboardMode.copy) {
          await session.copyFiles(clipboard.paths, _currentPath);
        } else {
          await session.moveFiles(clipboard.paths, _currentPath);
        }
      },
      success: clipboard.mode == _FileClipboardMode.copy
          ? 'Items copied'
          : 'Items moved',
    );
    if (mounted && succeeded) setState(() => _clipboard = null);
  }

  Future<bool> _runOperation(
    Future<void> Function() operation, {
    required String success,
  }) async {
    if (_operationPending) return false;
    setState(() => _operationPending = true);
    try {
      await operation();
      if (!mounted) return false;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(success)));
      await _loadPath(_currentPath);
      return true;
    } catch (error) {
      if (mounted) _showError(error);
      return false;
    } finally {
      if (mounted) setState(() => _operationPending = false);
    }
  }

  Future<void> _copyPaths(List<String> paths) async {
    if (paths.isEmpty) return;
    await Clipboard.setData(ClipboardData(text: paths.join('\n')));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            paths.length == 1 ? 'Path copied' : '${paths.length} paths copied',
          ),
        ),
      );
    }
  }

  Future<void> _pickAndUpload() async {
    try {
      final picked = await _transfers.pickAndUpload(
        endpointId: widget.endpointId,
        endpointLabel: widget.endpointLabel,
        remoteDirectory: _currentPath,
      );
      if (picked && mounted) {
        await showFileTransferCenter(context, _transfers);
      }
    } catch (error) {
      if (mounted) _showError(error);
    }
  }

  Future<void> _openBookmarks() async {
    final path = await showPathBookmarks(
      context: context,
      endpointId: widget.endpointId,
      currentPath: _currentPath,
    );
    if (path != null && mounted) await _loadPath(path);
  }

  void _showError(Object error) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(error.toString())));
  }
}

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

final class FilePreviewSheet extends ConsumerWidget {
  const FilePreviewSheet({
    super.key,
    required this.endpointId,
    required this.entry,
    required this.path,
    this.preview,
  });

  final String endpointId;
  final FileEntry entry;
  final String path;
  final Future<FilePreviewResult>? preview;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return FractionallySizedBox(
      heightFactor: 0.9,
      child: Scaffold(
        appBar: AppBar(
          automaticallyImplyLeading: false,
          title: Text(entry.name, maxLines: 1, overflow: TextOverflow.ellipsis),
          actions: [
            IconButton(
              tooltip: 'Close preview',
              constraints: const BoxConstraints.tightFor(width: 48, height: 48),
              onPressed: () => Navigator.pop(context),
              icon: const Icon(Icons.close_rounded),
            ),
          ],
        ),
        body: FutureBuilder<FilePreviewResult>(
          future:
              preview ??
              ref
                  .read(endpointSessionProvider(endpointId).future)
                  .then((session) => session.previewFile(path)),
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return const Center(
                child: CircularProgressIndicator(strokeWidth: 2),
              );
            }
            if (snapshot.hasError) {
              return Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(
                    snapshot.error.toString(),
                    textAlign: TextAlign.center,
                  ),
                ),
              );
            }
            final preview = snapshot.requireData;
            final bytes = Uint8List.fromList(preview.content);
            final mimeType = normalizedPreviewMimeType(preview.mimeType);
            if (mimeType.startsWith('image/')) {
              return InteractiveViewer(
                minScale: 0.5,
                maxScale: 5,
                child: Center(
                  child: Image(
                    image: ResizeImage(
                      MemoryImage(bytes),
                      width: maximumPreviewImageDimension,
                      height: maximumPreviewImageDimension,
                      policy: ResizeImagePolicy.fit,
                    ),
                    fit: BoxFit.contain,
                    gaplessPlayback: true,
                    errorBuilder: (context, error, stackTrace) => const Padding(
                      padding: EdgeInsets.all(24),
                      child: Text('The image could not be decoded safely.'),
                    ),
                  ),
                ),
              );
            }
            final textLike =
                mimeType.startsWith('text/') ||
                const [
                  'application/json',
                  'application/xml',
                  'application/yaml',
                  'application/x-yaml',
                ].contains(mimeType);
            if (!textLike) {
              return Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text('Preview is not available for $mimeType.'),
                ),
              );
            }
            final text = utf8.decode(bytes, allowMalformed: true);
            return Column(
              children: [
                if (preview.truncated)
                  const MaterialBanner(
                    content: Text('Preview truncated'),
                    actions: [SizedBox.shrink()],
                  ),
                Expanded(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: SelectableText(
                      text,
                      style: const TextStyle(
                        fontFamily: 'JetBrainsMonoNerd',
                        fontSize: 13,
                        height: 1.45,
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
