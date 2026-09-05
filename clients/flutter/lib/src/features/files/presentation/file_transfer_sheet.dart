import 'dart:async';

import 'package:flutter/material.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import 'file_transfer_controller.dart';

Future<void> showFileTransferCenter(
  BuildContext context,
  FileTransferController controller,
) {
  return Navigator.of(context, rootNavigator: true).push<void>(
    PageRouteBuilder<void>(
      opaque: true,
      transitionDuration: AnyttyMotion.quick,
      reverseTransitionDuration: AnyttyMotion.quick,
      pageBuilder: (context, _, _) =>
          _FileTransferSheet(controller: controller),
      transitionsBuilder: (context, animation, secondaryAnimation, child) {
        final curved = CurvedAnimation(
          parent: animation,
          curve: AnyttyMotion.emphasized,
        );
        return FadeTransition(
          opacity: curved,
          child: SlideTransition(
            position: Tween<Offset>(
              begin: const Offset(0, 0.015),
              end: Offset.zero,
            ).animate(curved),
            child: child,
          ),
        );
      },
    ),
  );
}

final class FileTransferCenterAction extends StatelessWidget {
  const FileTransferCenterAction({
    super.key,
    required this.controller,
    this.showWhenEmpty = false,
    this.dimension = 40,
    this.iconSize = 18,
  });

  final FileTransferController controller;
  final bool showWhenEmpty;
  final double dimension;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) {
        if (!showWhenEmpty && controller.items.isEmpty) {
          return const SizedBox.shrink();
        }
        return IconButton(
          tooltip: anyttyText(context, en: 'Download center', zh: '下载中心'),
          constraints: BoxConstraints.tightFor(
            width: dimension,
            height: dimension,
          ),
          padding: EdgeInsets.all((dimension - iconSize) / 2),
          onPressed: () => showFileTransferCenter(context, controller),
          icon: Stack(
            clipBehavior: Clip.none,
            children: [
              Icon(Icons.download_rounded, size: iconSize),
              if (controller.hasActiveTransfers)
                Positioned(
                  right: -2,
                  top: -2,
                  child: Container(
                    width: 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: AnyttyPalette.of(context).success,
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: Theme.of(context).colorScheme.surface,
                        width: 1.5,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }
}

final class _FileTransferSheet extends StatefulWidget {
  const _FileTransferSheet({required this.controller});

  final FileTransferController controller;

  @override
  State<_FileTransferSheet> createState() => _FileTransferSheetState();
}

final class _FileTransferSheetState extends State<_FileTransferSheet> {
  final Set<String> _selectedIds = {};
  bool _selectionMode = false;
  String? _openingId;
  String? _openErrorId;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_handleControllerChanged);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_handleControllerChanged);
    super.dispose();
  }

  void _handleControllerChanged() {
    if (!mounted) return;
    final available = widget.controller.items.map((item) => item.id).toSet();
    setState(() {
      _selectedIds.removeWhere((id) => !available.contains(id));
      if (available.isEmpty) _selectionMode = false;
    });
  }

  void _exitSelectionMode() {
    setState(() {
      _selectionMode = false;
      _selectedIds.clear();
    });
  }

  void _toggleSelected(String id) {
    setState(() {
      if (!_selectedIds.add(id)) _selectedIds.remove(id);
    });
  }

  void _toggleSelectAll(List<FileTransferViewItem> items) {
    setState(() {
      if (items.isNotEmpty && _selectedIds.length == items.length) {
        _selectedIds.clear();
      } else {
        _selectedIds
          ..clear()
          ..addAll(items.map((item) => item.id));
      }
    });
  }

  void _pauseSelected(List<FileTransferViewItem> items) {
    for (final item in items) {
      if (_selectedIds.contains(item.id) && item.pausable) {
        widget.controller.pause(item.id);
      }
    }
  }

  void _startSelected(List<FileTransferViewItem> items) {
    for (final item in items) {
      if (!_selectedIds.contains(item.id) || !item.resumable) continue;
      _resume(item);
    }
  }

  void _resumeAll(List<FileTransferViewItem> items) {
    for (final item in items) {
      if (item.resumable) _resume(item);
    }
  }

  void _resume(FileTransferViewItem item) {
    if (item.status == FileTransferStatus.failed) {
      widget.controller.retry(item.id);
    } else {
      widget.controller.resume(item.id);
    }
  }

  Future<void> _openDownload(FileTransferViewItem item) async {
    if (_openingId != null) return;
    setState(() {
      _openingId = item.id;
      _openErrorId = null;
    });
    try {
      await widget.controller.openDownloadedFile(item.id);
    } catch (_) {
      if (mounted) setState(() => _openErrorId = item.id);
    } finally {
      if (mounted) setState(() => _openingId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final items = widget.controller.items;
    final selectedItems = items
        .where((item) => _selectedIds.contains(item.id))
        .toList();
    final selectedPausable = selectedItems.any((item) => item.pausable);
    final selectedStartable = selectedItems.any((item) => item.resumable);
    final hasResumable = items.any((item) => item.resumable);
    final hasCompleted = items.any(
      (item) => item.status == FileTransferStatus.completed,
    );
    final hasFailed = items.any(
      (item) =>
          item.status == FileTransferStatus.failed ||
          item.status == FileTransferStatus.cancelled,
    );

    return PopScope<void>(
      canPop: !_selectionMode,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && _selectionMode) _exitSelectionMode();
      },
      child: Scaffold(
        backgroundColor: palette.background,
        body: SafeArea(
          child: Column(
            children: [
              _selectionMode
                  ? _SelectionHeader(
                      selectedCount: _selectedIds.length,
                      summary: _summary(context, items),
                      allSelected:
                          items.isNotEmpty &&
                          _selectedIds.length == items.length,
                      onCancel: _exitSelectionMode,
                      onSelectAll: () => _toggleSelectAll(items),
                    )
                  : _TransferHeader(
                      summary: _summary(context, items),
                      canResumeAll: hasResumable,
                      onResumeAll: () => _resumeAll(items),
                      onClose: () => Navigator.pop(context),
                    ),
              Expanded(
                child: items.isEmpty
                    ? Center(
                        child: Text(
                          anyttyText(context, en: 'No transfers', zh: '暂无传输任务'),
                          style: TextStyle(color: palette.muted),
                        ),
                      )
                    : ListView.builder(
                        itemCount: items.length,
                        itemBuilder: (context, index) {
                          final item = items[index];
                          return _TransferRow(
                            item: item,
                            controller: widget.controller,
                            selectionMode: _selectionMode,
                            selected: _selectedIds.contains(item.id),
                            opening: _openingId == item.id,
                            openFailed: _openErrorId == item.id,
                            onToggleSelected: () => _toggleSelected(item.id),
                            onResume: () => _resume(item),
                            onOpen: () => unawaited(_openDownload(item)),
                          );
                        },
                      ),
              ),
              if (items.isNotEmpty)
                _selectionMode
                    ? _SelectionToolbar(
                        canPause: selectedPausable,
                        canStart: selectedStartable,
                        onPause: () => _pauseSelected(items),
                        onStart: () => _startSelected(items),
                      )
                    : _TransferToolbar(
                        canClearCompleted: hasCompleted,
                        canClearFailed: hasFailed,
                        onSelect: () => setState(() => _selectionMode = true),
                        onClearCompleted: widget.controller.clearCompleted,
                        onClearFailed: widget.controller.clearFailed,
                      ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _TransferHeader extends StatelessWidget {
  const _TransferHeader({
    required this.summary,
    required this.canResumeAll,
    required this.onResumeAll,
    required this.onClose,
  });

  final String summary;
  final bool canResumeAll;
  final VoidCallback onResumeAll;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 54,
      padding: const EdgeInsets.only(left: 16, right: 4),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  anyttyText(context, en: 'Transfer Center', zh: '传输中心'),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                Text(
                  summary,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(fontSize: 11, color: palette.muted),
                ),
              ],
            ),
          ),
          if (canResumeAll)
            FilledButton.icon(
              onPressed: onResumeAll,
              style: FilledButton.styleFrom(
                minimumSize: const Size(0, 40),
                padding: const EdgeInsets.symmetric(horizontal: 10),
              ),
              icon: const Icon(Icons.restart_alt_rounded, size: 18),
              label: Text(
                anyttyText(context, en: 'Resume all', zh: '全部继续'),
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          IconButton(
            tooltip: anyttyText(
              context,
              en: 'Close data transfer center',
              zh: '关闭传输中心',
            ),
            onPressed: onClose,
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
    );
  }
}

final class _SelectionHeader extends StatelessWidget {
  const _SelectionHeader({
    required this.selectedCount,
    required this.summary,
    required this.allSelected,
    required this.onCancel,
    required this.onSelectAll,
  });

  final int selectedCount;
  final String summary;
  final bool allSelected;
  final VoidCallback onCancel;
  final VoidCallback onSelectAll;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 54,
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          SizedBox(
            width: 88,
            child: TextButton(
              onPressed: onCancel,
              child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
            ),
          ),
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  anyttyText(
                    context,
                    en: '$selectedCount selected',
                    zh: '已选择 $selectedCount 项',
                  ),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                Text(
                  summary,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(fontSize: 11, color: palette.muted),
                ),
              ],
            ),
          ),
          SizedBox(
            width: 88,
            child: TextButton(
              onPressed: onSelectAll,
              child: Text(allSelected ? 'Deselect' : 'Select all'),
            ),
          ),
        ],
      ),
    );
  }
}

final class _TransferRow extends StatelessWidget {
  const _TransferRow({
    required this.item,
    required this.controller,
    required this.selectionMode,
    required this.selected,
    required this.opening,
    required this.openFailed,
    required this.onToggleSelected,
    required this.onResume,
    required this.onOpen,
  });

  final FileTransferViewItem item;
  final FileTransferController controller;
  final bool selectionMode;
  final bool selected;
  final bool opening;
  final bool openFailed;
  final VoidCallback onToggleSelected;
  final VoidCallback onResume;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final progress = item.totalBytes <= 0
        ? null
        : (item.transferredBytes / item.totalBytes).clamp(0.0, 1.0);
    final failed =
        item.status == FileTransferStatus.failed ||
        item.status == FileTransferStatus.cancelled;
    final canOpen =
        item.direction == FileTransferDirection.download &&
        item.status == FileTransferStatus.completed &&
        item.savedUri != null;
    final dark = Theme.of(context).brightness == Brightness.dark;
    final directionColor = item.direction == FileTransferDirection.download
        ? (dark ? const Color(0xff60a5fa) : const Color(0xff2563eb))
        : (dark ? const Color(0xffc4b5fd) : const Color(0xff7c3aed));
    final directionBackground = item.direction == FileTransferDirection.download
        ? (dark ? const Color(0xff172554) : const Color(0xffeff6ff))
        : (dark ? const Color(0xff2e1065) : const Color(0xfff5f3ff));

    final primaryAction = canOpen
        ? onOpen
        : item.pausable
        ? () => controller.pause(item.id)
        : item.resumable
        ? onResume
        : null;
    final primaryIcon = canOpen
        ? Icons.open_in_new_rounded
        : item.pausable
        ? Icons.pause_rounded
        : item.resumable
        ? Icons.play_arrow_rounded
        : null;
    final activeOrPaused =
        item.active || item.status == FileTransferStatus.paused;

    return Container(
      key: ValueKey('transfer-${item.id}'),
      height: 112,
      padding: const EdgeInsets.fromLTRB(10, 9, 8, 9),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          if (selectionMode)
            IconButton(
              tooltip: selected
                  ? 'Deselect ${item.name}'
                  : 'Select ${item.name}',
              onPressed: onToggleSelected,
              constraints: const BoxConstraints.tightFor(width: 38, height: 38),
              padding: EdgeInsets.zero,
              icon: Icon(
                selected
                    ? Icons.check_box_rounded
                    : Icons.check_box_outline_blank_rounded,
                color: selected ? palette.text : palette.muted,
                size: 20,
              ),
            ),
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: directionBackground,
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(
              item.direction == FileTransferDirection.download
                  ? Icons.download_rounded
                  : Icons.upload_rounded,
              size: 20,
              color: directionColor,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                if (canOpen && !selectionMode)
                  InkWell(
                    onTap: opening ? null : onOpen,
                    child: Text(
                      item.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  )
                else
                  Text(
                    item.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                const SizedBox(height: 3),
                Text(
                  '${item.direction == FileTransferDirection.download ? anyttyText(context, en: 'From', zh: '来自') : anyttyText(context, en: 'To', zh: '发送到')} ${item.endpointLabel}  ·  ${item.remotePath}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(fontSize: 10, color: palette.muted),
                ),
                const SizedBox(height: 3),
                Text(
                  openFailed
                      ? anyttyText(
                          context,
                          en: 'This file could not be opened',
                          zh: '无法打开此文件',
                        )
                      : _itemStatus(context, item),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 10,
                    color: failed || openFailed
                        ? palette.danger
                        : palette.muted,
                  ),
                ),
                const SizedBox(height: 7),
                LinearProgressIndicator(
                  minHeight: 3,
                  value: item.status == FileTransferStatus.completed
                      ? 1
                      : progress,
                  color: failed
                      ? palette.danger
                      : item.status == FileTransferStatus.completed
                      ? palette.success
                      : directionColor,
                  backgroundColor: palette.surfaceRaised,
                ),
              ],
            ),
          ),
          if (!selectionMode)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (primaryIcon != null)
                  IconButton(
                    tooltip: canOpen
                        ? 'Open ${item.name}'
                        : item.pausable
                        ? 'Pause ${item.name}'
                        : 'Resume ${item.name}',
                    constraints: const BoxConstraints.tightFor(
                      width: 36,
                      height: 40,
                    ),
                    padding: EdgeInsets.zero,
                    onPressed: opening ? null : primaryAction,
                    icon: opening
                        ? const SizedBox.square(
                            dimension: 16,
                            child: CircularProgressIndicator(strokeWidth: 1.8),
                          )
                        : Icon(primaryIcon, size: 18),
                  ),
                IconButton(
                  tooltip: activeOrPaused
                      ? 'Cancel ${item.name}'
                      : 'Remove ${item.name}',
                  constraints: const BoxConstraints.tightFor(
                    width: 36,
                    height: 40,
                  ),
                  padding: EdgeInsets.zero,
                  onPressed: activeOrPaused
                      ? () => controller.cancel(item.id)
                      : () => controller.dismiss(item.id),
                  icon: Icon(
                    activeOrPaused
                        ? Icons.close_rounded
                        : Icons.delete_outline_rounded,
                    size: 18,
                  ),
                ),
              ],
            ),
        ],
      ),
    );
  }
}

final class _TransferToolbar extends StatelessWidget {
  const _TransferToolbar({
    required this.canClearCompleted,
    required this.canClearFailed,
    required this.onSelect,
    required this.onClearCompleted,
    required this.onClearFailed,
  });

  final bool canClearCompleted;
  final bool canClearFailed;
  final VoidCallback onSelect;
  final VoidCallback onClearCompleted;
  final VoidCallback onClearFailed;

  @override
  Widget build(BuildContext context) => _BottomToolbar(
    children: [
      _ToolbarAction(
        tooltip: anyttyText(context, en: 'Select transfers', zh: '选择传输任务'),
        icon: Icons.check_box_rounded,
        label: anyttyText(context, en: 'Select', zh: '选择'),
        onPressed: onSelect,
      ),
      _ToolbarAction(
        tooltip: anyttyText(
          context,
          en: 'Delete all completed transfers',
          zh: '删除全部已完成任务',
        ),
        icon: Icons.delete_outline_rounded,
        label: anyttyText(context, en: 'Done', zh: '已完成'),
        onPressed: canClearCompleted ? onClearCompleted : null,
      ),
      _ToolbarAction(
        tooltip: anyttyText(
          context,
          en: 'Delete all failed transfers',
          zh: '删除全部失败任务',
        ),
        icon: Icons.delete_outline_rounded,
        label: anyttyText(context, en: 'Failed', zh: '失败'),
        onPressed: canClearFailed ? onClearFailed : null,
      ),
    ],
  );
}

final class _SelectionToolbar extends StatelessWidget {
  const _SelectionToolbar({
    required this.canPause,
    required this.canStart,
    required this.onPause,
    required this.onStart,
  });

  final bool canPause;
  final bool canStart;
  final VoidCallback onPause;
  final VoidCallback onStart;

  @override
  Widget build(BuildContext context) => _BottomToolbar(
    children: [
      _ToolbarAction(
        tooltip: anyttyText(
          context,
          en: 'Pause selected transfers',
          zh: '暂停所选任务',
        ),
        icon: Icons.pause_rounded,
        label: anyttyText(context, en: 'Pause', zh: '暂停'),
        onPressed: canPause ? onPause : null,
      ),
      _ToolbarAction(
        tooltip: anyttyText(
          context,
          en: 'Start selected transfers',
          zh: '开始所选任务',
        ),
        icon: Icons.play_arrow_rounded,
        label: anyttyText(context, en: 'Start', zh: '开始'),
        onPressed: canStart ? onStart : null,
      ),
    ],
  );
}

final class _BottomToolbar extends StatelessWidget {
  const _BottomToolbar({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 54,
      decoration: BoxDecoration(
        color: palette.background,
        border: Border(top: BorderSide(color: palette.border)),
      ),
      child: Row(children: children),
    );
  }
}

final class _ToolbarAction extends StatelessWidget {
  const _ToolbarAction({
    required this.tooltip,
    required this.icon,
    required this.label,
    required this.onPressed,
  });

  final String tooltip;
  final IconData icon;
  final String label;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Expanded(
      child: Tooltip(
        message: tooltip,
        child: InkWell(
          onTap: onPressed,
          child: Opacity(
            opacity: onPressed == null ? 0.4 : 1,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, size: 21, color: palette.muted),
                const SizedBox(height: 2),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w500,
                    color: palette.muted,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String _summary(BuildContext context, List<FileTransferViewItem> items) {
  final active = items.where((item) => item.active).length;
  final paused = items
      .where((item) => item.status == FileTransferStatus.paused)
      .length;
  final failed = items
      .where(
        (item) =>
            item.status == FileTransferStatus.failed ||
            item.status == FileTransferStatus.cancelled,
      )
      .length;
  final completed = items
      .where((item) => item.status == FileTransferStatus.completed)
      .length;
  final parts = <String>[
    if (active > 0)
      anyttyText(context, en: '$active active', zh: '$active 个进行中'),
    if (paused > 0)
      anyttyText(context, en: '$paused paused', zh: '$paused 个已暂停'),
    if (failed > 0)
      anyttyText(context, en: '$failed failed', zh: '$failed 个失败'),
    if (completed > 0)
      anyttyText(context, en: '$completed done', zh: '$completed 个完成'),
  ];
  return parts.isEmpty
      ? anyttyText(context, en: 'No transfers', zh: '暂无传输任务')
      : parts.join(' · ');
}

String _itemStatus(BuildContext context, FileTransferViewItem item) {
  final progress = item.totalBytes > 0
      ? '${_formatTransferBytes(item.transferredBytes)} / ${_formatTransferBytes(item.totalBytes)}'
      : _formatTransferBytes(item.transferredBytes);
  return switch (item.status) {
    FileTransferStatus.pending =>
      '$progress · ${anyttyText(context, en: 'Pending', zh: '等待中')}',
    FileTransferStatus.transferring =>
      '$progress · ${anyttyText(context, en: 'Transferring', zh: '传输中')}',
    FileTransferStatus.paused =>
      '$progress · ${anyttyText(context, en: 'Paused', zh: '已暂停')}',
    FileTransferStatus.saving =>
      '$progress · ${anyttyText(context, en: 'Saving', zh: '保存中')}',
    FileTransferStatus.completed =>
      '$progress · ${anyttyText(context, en: 'Completed', zh: '已完成')}',
    FileTransferStatus.cancelled =>
      item.error ??
          '$progress · ${anyttyText(context, en: 'Cancelled', zh: '已取消')}',
    FileTransferStatus.failed =>
      item.error ??
          '$progress · ${anyttyText(context, en: 'Failed', zh: '失败')}',
  };
}

String _formatTransferBytes(int bytes) {
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
