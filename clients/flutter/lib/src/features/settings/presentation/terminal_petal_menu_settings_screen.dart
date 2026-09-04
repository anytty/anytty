import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/anytty_ui.dart';
import '../../../app/providers.dart';
import '../../terminal/domain/terminal_petal_menu_preferences.dart';

final class TerminalPetalMenuSettingsScreen extends ConsumerStatefulWidget {
  const TerminalPetalMenuSettingsScreen({super.key});

  @override
  ConsumerState<TerminalPetalMenuSettingsScreen> createState() =>
      _TerminalPetalMenuSettingsScreenState();
}

final class _TerminalPetalMenuSettingsScreenState
    extends ConsumerState<TerminalPetalMenuSettingsScreen> {
  final List<String> _path = <String>[];
  ScaffoldMessengerState? _feedbackMessenger;
  int _page = 0;

  @override
  void dispose() {
    final messenger = _feedbackMessenger;
    if (messenger != null && messenger.mounted) {
      messenger.removeCurrentSnackBar();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final preferences =
        ref.watch(terminalPetalMenuPreferencesProvider).valueOrNull ??
        TerminalPetalMenuPreferences.defaults;
    final rawRoots = _petalTree(preferences.layout);
    final visibleRoots = _petalTree(preferences.visibleLayout);
    final location = _resolveLocation(visibleRoots, _path);
    final hiddenNodes = _flattenPetals(rawRoots)
        .where((node) => preferences.hiddenActionIds.contains(node.id))
        .toList(growable: false);
    final currentParentId = location.parent?.id;
    final compactHeader =
        MediaQuery.sizeOf(context).width < 420 ||
        MediaQuery.textScalerOf(context).scale(1) > 1.35;

    return Scaffold(
      appBar: AppBar(
        toolbarHeight: AnyttyUi.appBarHeight(context, subtitleLines: 2),
        leading: AnyttyIconButton(
          tooltip: anyttyText(context, en: 'Back to settings', zh: '返回设置'),
          onPressed: context.pop,
          icon: LucideIcons.chevronLeft,
        ),
        titleSpacing: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              anyttyText(context, en: 'Petal menu', zh: '花瓣菜单'),
              style: AnyttyUi.title(context),
            ),
            Text(
              anyttyText(
                context,
                en: 'Terminal long-press actions',
                zh: '终端长按操作',
              ),
              style: AnyttyUi.muted(context),
              maxLines: 2,
              softWrap: true,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
        actions: [
          if (compactHeader)
            AnyttyIconButton(
              key: const ValueKey('petal-menu-restore-defaults'),
              tooltip: anyttyText(context, en: 'Restore default', zh: '恢复默认'),
              onPressed: preferences == TerminalPetalMenuPreferences.defaults
                  ? null
                  : () => unawaited(_confirmRestoreDefaults(preferences)),
              icon: LucideIcons.rotateCcw,
            )
          else
            AnyttyPillButton(
              key: const ValueKey('petal-menu-restore-defaults'),
              onPressed: preferences == TerminalPetalMenuPreferences.defaults
                  ? null
                  : () => unawaited(_confirmRestoreDefaults(preferences)),
              icon: LucideIcons.rotateCcw,
              label: anyttyText(context, en: 'Restore default', zh: '恢复默认'),
            ),
          const SizedBox(width: 4),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 576),
            child: CustomScrollView(
              slivers: [
                SliverPadding(
                  padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
                  sliver: SliverList.list(
                    children: [
                      _SettingsGroup(
                        children: [
                          _ToggleRow(
                            icon: LucideIcons.flower2,
                            label: anyttyText(
                              context,
                              en: 'Long-press petal menu',
                              zh: '长按花瓣菜单',
                            ),
                            description: anyttyText(
                              context,
                              en: 'Available inside the terminal canvas',
                              zh: '在终端区域内启用',
                            ),
                            value: preferences.enabled,
                            switchKey: const ValueKey('petal-menu-enabled'),
                            onChanged: (value) => unawaited(
                              _save(preferences.copyWith(enabled: value)),
                            ),
                          ),
                          _ToggleRow(
                            icon: LucideIcons.vibrate,
                            label: anyttyText(
                              context,
                              en: 'Haptic feedback',
                              zh: '触觉反馈',
                            ),
                            description: anyttyText(
                              context,
                              en: 'Feedback while dragging and dropping',
                              zh: '拖动和放置时反馈',
                            ),
                            value: preferences.hapticsEnabled,
                            switchKey: const ValueKey('petal-menu-haptics'),
                            onChanged: (value) => unawaited(
                              _save(
                                preferences.copyWith(hapticsEnabled: value),
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 24),
                      _SectionHeading(
                        label: anyttyText(
                          context,
                          en: 'Current menu',
                          zh: '当前菜单',
                        ),
                        trailing: anyttyText(
                          context,
                          en: '${location.children.length} actions',
                          zh: '${location.children.length} 个操作',
                        ),
                      ),
                      const SizedBox(height: 8),
                      _PetalLayoutPanel(
                        key: const ValueKey('petal-layout-editor'),
                        parent: location.parent,
                        nodes: location.children,
                        page: _page,
                        hapticsEnabled: preferences.hapticsEnabled,
                        onPageChanged: (page) => setState(() => _page = page),
                        onOpen: (node) => setState(() {
                          _path
                            ..clear()
                            ..addAll(location.path)
                            ..add(node.id);
                          _page = 0;
                        }),
                        onBack: location.parent == null
                            ? null
                            : () => setState(() {
                                _path.removeLast();
                                _page = 0;
                              }),
                        onDropOnNode: (sourceId, targetId) => unawaited(
                          _dropOnNode(
                            preferences: preferences,
                            roots: rawRoots,
                            sourceId: sourceId,
                            targetId: targetId,
                          ),
                        ),
                        onDropOnSlot: (sourceId, targetIndex) => unawaited(
                          _moveToCurrentRing(
                            preferences: preferences,
                            sourceId: sourceId,
                            parentActionId: currentParentId,
                            targetIndex: targetIndex,
                          ),
                        ),
                        onMoveOut: (sourceId) =>
                            unawaited(_moveOut(preferences, sourceId)),
                        onMoveUp: (sourceId, targetIndex) => unawaited(
                          _moveToCurrentRing(
                            preferences: preferences,
                            sourceId: sourceId,
                            parentActionId: currentParentId,
                            targetIndex: targetIndex,
                          ),
                        ),
                        onMoveDown: (sourceId, targetIndex) => unawaited(
                          _moveToCurrentRing(
                            preferences: preferences,
                            sourceId: sourceId,
                            parentActionId: currentParentId,
                            targetIndex: targetIndex,
                          ),
                        ),
                        onAddToSlot: hiddenNodes.isEmpty
                            ? null
                            : (targetIndex) => unawaited(
                                _showAddActionPicker(
                                  preferences: preferences,
                                  parentActionId: currentParentId,
                                  targetIndex: targetIndex,
                                  nodes: hiddenNodes,
                                ),
                              ),
                      ),
                      const SizedBox(height: 24),
                      _SectionHeading(
                        label: anyttyText(
                          context,
                          en: 'Action library',
                          zh: '可用动作',
                        ),
                        trailing: '${hiddenNodes.length}',
                      ),
                      const SizedBox(height: 8),
                      _PetalActionLibrary(
                        nodes: hiddenNodes,
                        hapticsEnabled: preferences.hapticsEnabled,
                        onRemove: (sourceId) => unawaited(
                          _moveToLibrary(preferences, rawRoots, sourceId),
                        ),
                        onAdd: (sourceId) => unawaited(
                          _moveToCurrentRing(
                            preferences: preferences,
                            sourceId: sourceId,
                            parentActionId: currentParentId,
                            targetIndex: location.children.length,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _confirmRestoreDefaults(
    TerminalPetalMenuPreferences preferences,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => Dialog(
        backgroundColor: Colors.transparent,
        insetPadding: const EdgeInsets.all(24),
        child: AnyttyCard(
          radius: 16,
          depth: 2,
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                anyttyText(
                  dialogContext,
                  en: 'Restore the default petal menu?',
                  zh: '恢复花瓣菜单默认配置？',
                ),
                style: AnyttyUi.sectionTitle(dialogContext),
              ),
              const SizedBox(height: 10),
              Text(
                anyttyText(
                  dialogContext,
                  en: 'This replaces the current order, levels, hidden actions, and menu switches with the built-in configuration.',
                  zh: '这会用内置配置替换当前的动作顺序、层级、隐藏动作和花瓣菜单开关。',
                ),
                style: AnyttyUi.body(dialogContext),
              ),
              const SizedBox(height: 18),
              Wrap(
                alignment: WrapAlignment.end,
                spacing: 8,
                runSpacing: 8,
                children: [
                  AnyttyPillButton(
                    outlined: true,
                    label: anyttyText(dialogContext, en: 'Cancel', zh: '取消'),
                    onPressed: () => Navigator.of(dialogContext).pop(false),
                  ),
                  AnyttyPillButton(
                    key: const ValueKey('petal-menu-restore-confirm'),
                    label: anyttyText(
                      dialogContext,
                      en: 'Restore default',
                      zh: '恢复默认',
                    ),
                    onPressed: () => Navigator.of(dialogContext).pop(true),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
    if (!mounted || confirmed != true) return;
    final saved = await _commit(
      previous: preferences,
      next: TerminalPetalMenuPreferences.defaults,
      message: anyttyText(
        context,
        en: 'Petal menu restored',
        zh: '花瓣菜单已恢复默认布局',
      ),
    );
    if (!mounted || !saved) return;
    setState(() {
      _path.clear();
      _page = 0;
    });
  }

  Future<void> _dropOnNode({
    required TerminalPetalMenuPreferences preferences,
    required List<_PetalEditorNode> roots,
    required String sourceId,
    required String targetId,
  }) async {
    if (sourceId == targetId) return;
    final sourceParentId = _parentIdFor(roots, sourceId);
    final targetParentId = _parentIdFor(roots, targetId);
    final sourceIsHidden = preferences.hiddenActionIds.contains(sourceId);
    final reparenting = sourceIsHidden || sourceParentId != targetParentId;
    List<TerminalPetalMenuPlacement> layout;
    String message;

    if (!sourceIsHidden && sourceParentId == targetParentId) {
      final oldIndex = preferences.layout.indexWhere(
        (placement) => placement.id == sourceId,
      );
      final newIndex = preferences.layout.indexWhere(
        (placement) => placement.id == targetId,
      );
      layout = reorderTerminalPetalSubtree(
        preferences.layout,
        oldIndex: oldIndex,
        newIndex: newIndex,
      );
      message = anyttyText(
        context,
        en: '${_presentation(context, sourceId).label} reordered',
        zh: '${_presentation(context, sourceId).label}位置已更新',
      );
    } else {
      layout = reparentTerminalPetalSubtree(
        preferences.layout,
        actionId: sourceId,
        parentActionId: targetId,
      );
      if (_sameLayout(layout, preferences.layout)) {
        _showMessage(
          anyttyText(
            context,
            en: 'This branch cannot be nested any deeper',
            zh: '这个分支不能继续放入更深层级',
          ),
        );
        return;
      }
      message = anyttyText(
        context,
        en: '${_presentation(context, sourceId).label} moved inside ${_presentation(context, targetId).label}',
        zh: '${_presentation(context, sourceId).label}已放入${_presentation(context, targetId).label}',
      );
    }

    final hidden = {...preferences.hiddenActionIds}..remove(sourceId);
    final next = preferences.copyWith(layout: layout, hiddenActionIds: hidden);
    final saved = await _commit(
      previous: preferences,
      next: next,
      message: message,
    );
    if (!mounted || !saved || !reparenting) return;
    final nextPath = _pathToNode(_petalTree(next.visibleLayout), targetId);
    if (nextPath == null) return;
    setState(() {
      _path
        ..clear()
        ..addAll(nextPath);
      _page = 0;
    });
  }

  Future<void> _moveToCurrentRing({
    required TerminalPetalMenuPreferences preferences,
    required String sourceId,
    required String? parentActionId,
    required int targetIndex,
  }) async {
    var layout = _moveToParent(
      preferences.layout,
      actionId: sourceId,
      parentActionId: parentActionId,
    );
    layout = _moveSiblingToIndex(
      layout,
      actionId: sourceId,
      parentActionId: parentActionId,
      targetIndex: targetIndex,
    );
    final hidden = {...preferences.hiddenActionIds}..remove(sourceId);
    final next = preferences.copyWith(layout: layout, hiddenActionIds: hidden);
    if (next == preferences) return;
    await _commit(
      previous: preferences,
      next: next,
      message: anyttyText(
        context,
        en: '${_presentation(context, sourceId).label} added',
        zh: '${_presentation(context, sourceId).label}已加入当前菜单',
      ),
    );
  }

  Future<void> _moveOut(
    TerminalPetalMenuPreferences preferences,
    String sourceId,
  ) async {
    final layout = outdentTerminalPetalSubtree(
      preferences.layout,
      actionId: sourceId,
    );
    if (_sameLayout(layout, preferences.layout)) return;
    final next = preferences.copyWith(layout: layout);
    final saved = await _commit(
      previous: preferences,
      next: next,
      message: anyttyText(
        context,
        en: '${_presentation(context, sourceId).label} moved outward',
        zh: '${_presentation(context, sourceId).label}已移到外层',
      ),
    );
    if (!mounted || !saved) return;
    setState(() {
      if (_path.isNotEmpty) _path.removeLast();
      _page = 0;
    });
  }

  Future<void> _showAddActionPicker({
    required TerminalPetalMenuPreferences preferences,
    required String? parentActionId,
    required int targetIndex,
    required List<_PetalEditorNode> nodes,
  }) async {
    final sourceId = await showModalBottomSheet<String>(
      context: context,
      useSafeArea: true,
      showDragHandle: true,
      builder: (context) => SafeArea(
        top: false,
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.sizeOf(context).height * 0.68,
          ),
          child: ListView.separated(
            shrinkWrap: true,
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 20),
            itemCount: nodes.length,
            separatorBuilder: (_, _) => const SizedBox(height: 8),
            itemBuilder: (context, index) {
              final node = nodes[index];
              final presentation = _presentation(context, node.id);
              return AnyttyCard(
                radius: 14,
                depth: 1,
                onTap: () => Navigator.pop(context, node.id),
                padding: const EdgeInsets.symmetric(
                  horizontal: 14,
                  vertical: 10,
                ),
                child: Row(
                  children: [
                    Icon(
                      presentation.icon,
                      color: AnyttyPalette.of(context).strong,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(
                        presentation.label,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: AnyttyUi.body(context)
                            .copyWith(fontWeight: FontWeight.w600),
                      ),
                    ),
                    const Icon(Icons.add_rounded),
                  ],
                ),
              );
            },
          ),
        ),
      ),
    );
    if (!mounted || sourceId == null) return;
    await _moveToCurrentRing(
      preferences: preferences,
      sourceId: sourceId,
      parentActionId: parentActionId,
      targetIndex: targetIndex,
    );
  }

  Future<void> _moveToLibrary(
    TerminalPetalMenuPreferences preferences,
    List<_PetalEditorNode> roots,
    String sourceId,
  ) async {
    if (preferences.hiddenActionIds.contains(sourceId)) return;
    final node = _findNode(roots, sourceId);
    if (node == null) return;
    final hidden = {...preferences.hiddenActionIds, ..._subtreeIds(node)};
    final next = preferences.copyWith(hiddenActionIds: hidden);
    await _commit(
      previous: preferences,
      next: next,
      message: anyttyText(
        context,
        en: '${_presentation(context, sourceId).label} moved to the library',
        zh: '${_presentation(context, sourceId).label}已移到动作库',
      ),
    );
  }

  Future<bool> _commit({
    required TerminalPetalMenuPreferences previous,
    required TerminalPetalMenuPreferences next,
    required String message,
  }) async {
    if (next == previous) return false;
    final saved = await _save(next);
    if (!mounted || !saved) return false;
    if (next.hapticsEnabled) unawaited(HapticFeedback.lightImpact());
    final messenger = ScaffoldMessenger.of(context);
    _feedbackMessenger = messenger;
    messenger.hideCurrentSnackBar();
    messenger.showSnackBar(
      SnackBar(
        content: Text(message),
        duration: const Duration(seconds: 4),
        persist: false,
        action: SnackBarAction(
          label: anyttyText(context, en: 'Undo', zh: '撤销'),
          onPressed: () => unawaited(_save(previous)),
        ),
      ),
    );
    return true;
  }

  Future<bool> _save(TerminalPetalMenuPreferences value) async {
    try {
      await ref.read(terminalPetalMenuPreferencesProvider.notifier).save(value);
      return true;
    } catch (error) {
      if (!mounted) return false;
      _showMessage(
        '${anyttyText(context, en: 'Could not save petal menu settings', zh: '无法保存花瓣菜单设置')}: $error',
      );
      return false;
    }
  }

  void _showMessage(String message) {
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    _feedbackMessenger = messenger;
    messenger.hideCurrentSnackBar();
    messenger.showSnackBar(SnackBar(content: Text(message)));
  }
}

final class _SectionHeading extends StatelessWidget {
  const _SectionHeading({required this.label, required this.trailing});

  final String label;
  final String trailing;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(child: Text(label, style: AnyttyUi.sectionTitle(context))),
        Text(trailing, style: AnyttyUi.muted(context)),
      ],
    );
  }
}

final class _SettingsGroup extends StatelessWidget {
  const _SettingsGroup({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return DecoratedBox(
      decoration: AnyttyUi.cardDecoration(context, radius: 16, depth: 1),
      child: Column(
        children: [
          for (var index = 0; index < children.length; index += 1) ...[
            children[index],
            if (index != children.length - 1)
              Divider(height: 1, indent: 58, color: palette.track),
          ],
        ],
      ),
    );
  }
}

final class _ToggleRow extends StatelessWidget {
  const _ToggleRow({
    required this.icon,
    required this.label,
    required this.description,
    required this.value,
    required this.switchKey,
    required this.onChanged,
  });

  final IconData icon;
  final String label;
  final String description;
  final bool value;
  final Key switchKey;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final lineHeight =
        MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5);
    return ConstrainedBox(
      constraints: BoxConstraints(minHeight: math.max(66, lineHeight * 4 + 12)),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 6, 10, 6),
        child: Row(
          children: [
            Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: palette.surfaceRaised,
                borderRadius: BorderRadius.circular(8),
              ),
              alignment: Alignment.center,
              child: Icon(icon, size: 18, color: palette.strong),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    label,
                    style: AnyttyUi.body(context)
                        .copyWith(fontWeight: FontWeight.w600),
                  ),
                  const SizedBox(height: 2),
                  Text(description, style: AnyttyUi.muted(context)),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Semantics(
              label: label,
              child: Switch.adaptive(
                key: switchKey,
                value: value,
                activeTrackColor: palette.accent,
                onChanged: onChanged,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _PetalLayoutPanel extends StatelessWidget {
  const _PetalLayoutPanel({
    super.key,
    required this.parent,
    required this.nodes,
    required this.page,
    required this.hapticsEnabled,
    required this.onPageChanged,
    required this.onOpen,
    required this.onBack,
    required this.onDropOnNode,
    required this.onDropOnSlot,
    required this.onMoveOut,
    required this.onMoveUp,
    required this.onMoveDown,
    required this.onAddToSlot,
  });

  static const pageSize = 8;

  final _PetalEditorNode? parent;
  final List<_PetalEditorNode> nodes;
  final int page;
  final bool hapticsEnabled;
  final ValueChanged<int> onPageChanged;
  final ValueChanged<_PetalEditorNode> onOpen;
  final VoidCallback? onBack;
  final void Function(String sourceId, String targetId) onDropOnNode;
  final void Function(String sourceId, int targetIndex) onDropOnSlot;
  final ValueChanged<String> onMoveOut;
  final void Function(String sourceId, int targetIndex) onMoveUp;
  final void Function(String sourceId, int targetIndex) onMoveDown;
  final void Function(int targetIndex)? onAddToSlot;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final pageCount = math.max(1, (nodes.length / pageSize).ceil());
    final currentPage = page.clamp(0, pageCount - 1);
    final start = currentPage * pageSize;
    final visibleNodes = nodes
        .skip(start)
        .take(pageSize)
        .toList(growable: false);
    final title = parent == null
        ? anyttyText(context, en: 'Main menu', zh: '主菜单')
        : _presentation(context, parent!.id).label;
    final level = parent == null ? 1 : parent!.depth + 2;

    return DecoratedBox(
      decoration: AnyttyUi.cardDecoration(context, radius: 16, depth: 2),
      child: Column(
        children: [
          ConstrainedBox(
            constraints: BoxConstraints(
              minHeight: math.max(
                54,
                MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5) * 2 +
                    10,
              ),
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12),
              child: Row(
                children: [
                  Container(
                    width: 32,
                    height: 32,
                    decoration: BoxDecoration(
                      color: palette.accent.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    alignment: Alignment.center,
                    child: Icon(
                      parent == null
                          ? LucideIcons.flower2
                          : _iconForPetal(parent!.id),
                      size: 17,
                      color: palette.strong,
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      mainAxisAlignment: MainAxisAlignment.center,
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(
                          title,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: AnyttyUi.body(context),
                        ),
                        Text(
                          anyttyText(
                            context,
                            en: 'Level $level',
                            zh: '第 $level 层',
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: AnyttyUi.muted(context),
                        ),
                      ],
                    ),
                  ),
                  if (pageCount > 1)
                    _PageButton(
                      key: const ValueKey('petal-editor-page-previous'),
                      icon: LucideIcons.chevronLeft,
                      enabled: currentPage > 0,
                      onPressed: () => onPageChanged(currentPage - 1),
                    ),
                  Text(
                    pageCount > 1
                        ? '${currentPage + 1}/$pageCount'
                        : '${visibleNodes.length} / $pageSize',
                    style: AnyttyUi.muted(context),
                  ),
                  if (pageCount > 1)
                    _PageButton(
                      key: const ValueKey('petal-editor-page-next'),
                      icon: LucideIcons.chevronRight,
                      enabled: currentPage < pageCount - 1,
                      onPressed: () => onPageChanged(currentPage + 1),
                    ),
                ],
              ),
            ),
          ),
          Divider(height: 1, color: palette.track),
          SizedBox(
            height: MediaQuery.textScalerOf(context).scale(1) > 1.5
                ? math.max(205, AnyttyUi.controlHeight(context) * 8)
                : 205,
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: palette.surfaceRaised.withValues(alpha: 0.72),
                borderRadius: const BorderRadius.vertical(
                  bottom: Radius.circular(16),
                ),
              ),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 420),
                  child: AnimatedSwitcher(
                    duration: AnyttyMotion.resolve(
                      context,
                      AnyttyMotion.standard,
                    ),
                    switchInCurve: AnyttyMotion.emphasized,
                    switchOutCurve: Curves.easeIn,
                    transitionBuilder: (child, animation) => FadeTransition(
                      opacity: animation,
                      child: ScaleTransition(
                        scale: Tween(begin: 0.96, end: 1.0).animate(animation),
                        child: child,
                      ),
                    ),
                    child: _PetalRadialCanvas(
                      key: ValueKey(
                        'petal-ring-${parent?.id ?? 'root'}-$currentPage',
                      ),
                      parent: parent,
                      nodes: visibleNodes,
                      totalNodeCount: nodes.length,
                      pageStart: start,
                      hapticsEnabled: hapticsEnabled,
                      onOpen: onOpen,
                      onBack: onBack,
                      onDropOnNode: onDropOnNode,
                      onDropOnSlot: onDropOnSlot,
                      onMoveOut: onMoveOut,
                      onMoveUp: onMoveUp,
                      onMoveDown: onMoveDown,
                      onAddToSlot: onAddToSlot,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _PetalRadialCanvas extends StatelessWidget {
  const _PetalRadialCanvas({
    super.key,
    required this.parent,
    required this.nodes,
    required this.totalNodeCount,
    required this.pageStart,
    required this.hapticsEnabled,
    required this.onOpen,
    required this.onBack,
    required this.onDropOnNode,
    required this.onDropOnSlot,
    required this.onMoveOut,
    required this.onMoveUp,
    required this.onMoveDown,
    required this.onAddToSlot,
  });

  static const nodeExtent = 46.0;
  static const hubExtent = 46.0;

  final _PetalEditorNode? parent;
  final List<_PetalEditorNode> nodes;
  final int totalNodeCount;
  final int pageStart;
  final bool hapticsEnabled;
  final ValueChanged<_PetalEditorNode> onOpen;
  final VoidCallback? onBack;
  final void Function(String sourceId, String targetId) onDropOnNode;
  final void Function(String sourceId, int targetIndex) onDropOnSlot;
  final ValueChanged<String> onMoveOut;
  final void Function(String sourceId, int targetIndex) onMoveUp;
  final void Function(String sourceId, int targetIndex) onMoveDown;
  final void Function(int targetIndex)? onAddToSlot;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final size = constraints.biggest;
      final center = Offset(size.width / 2, size.height / 2);
      final radiusX = math.max(0.0, (size.width - nodeExtent) / 2 - 5);
      final labelHeight =
          MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5) * 2 + 10;
      final minRadiusY = MediaQuery.textScalerOf(context).scale(1) > 1.5
          ? 72.0
          : 40.0;
      final radiusY = math.max(
        minRadiusY,
        (size.height - nodeExtent) / 2 - labelHeight,
      );
      final positions = List.generate(8, (index) {
        final angle = -math.pi / 2 + index * math.pi / 4;
        return center +
            Offset(math.cos(angle) * radiusX, math.sin(angle) * radiusY);
      }, growable: false);
      final labelWidth = math.min(100.0, math.max(36.0, size.width - 16));
      final palette = AnyttyPalette.of(context);
      return Stack(
        clipBehavior: Clip.none,
        children: [
          Positioned.fill(
            child: CustomPaint(
              painter: _PetalEditorGuidePainter(
                center: center,
                positions: positions,
                color: palette.borderStrong,
              ),
            ),
          ),
          for (var index = 0; index < positions.length; index += 1)
            Positioned(
              left: positions[index].dx - nodeExtent / 2,
              top: positions[index].dy - nodeExtent / 2,
              width: nodeExtent,
              height: nodeExtent,
              child: _PetalSlot(
                key: ValueKey('petal-editor-slot-$index'),
                node: index < nodes.length ? nodes[index] : null,
                hapticsEnabled: hapticsEnabled,
                onOpen: onOpen,
                onDropOnNode: onDropOnNode,
                onDropOnSlot: (sourceId) =>
                    onDropOnSlot(sourceId, pageStart + index),
                onMoveOut: onMoveOut,
                onAddToSlot: onAddToSlot == null
                    ? null
                    : () => onAddToSlot!(pageStart + index),
                labelWidth: labelWidth,
                labelLeft:
                    (positions[index].dx -
                            labelWidth / 2 -
                            (positions[index].dx - nodeExtent / 2))
                        .clamp(
                          8 - (positions[index].dx - nodeExtent / 2),
                          size.width -
                              labelWidth -
                              8 -
                              (positions[index].dx - nodeExtent / 2),
                        )
                        .toDouble(),
                labelAbove: positions[index].dy > center.dy,
                onMoveUp: pageStart + index == 0
                    ? null
                    : (sourceId) => onMoveUp(sourceId, pageStart + index - 1),
                onMoveDown: pageStart + index >= totalNodeCount - 1
                    ? null
                    : (sourceId) => onMoveDown(sourceId, pageStart + index + 1),
              ),
            ),
          Positioned(
            left: center.dx - hubExtent / 2,
            top: center.dy - hubExtent / 2,
            width: hubExtent,
            height: hubExtent,
            child: _PetalEditorHub(
              parent: parent,
              currentNodes: nodes,
              onBack: onBack,
              onMoveOut: onMoveOut,
            ),
          ),
        ],
      );
    },
  );
}

final class _PetalSlot extends StatelessWidget {
  const _PetalSlot({
    super.key,
    required this.node,
    required this.hapticsEnabled,
    required this.onOpen,
    required this.onDropOnNode,
    required this.onDropOnSlot,
    required this.onMoveOut,
    required this.onAddToSlot,
    required this.labelWidth,
    required this.labelLeft,
    required this.labelAbove,
    this.onMoveUp,
    this.onMoveDown,
  });

  final _PetalEditorNode? node;
  final bool hapticsEnabled;
  final ValueChanged<_PetalEditorNode> onOpen;
  final void Function(String sourceId, String targetId) onDropOnNode;
  final ValueChanged<String> onDropOnSlot;
  final ValueChanged<String> onMoveOut;
  final VoidCallback? onAddToSlot;
  final double labelWidth;
  final double labelLeft;
  final bool labelAbove;
  final ValueChanged<String>? onMoveUp;
  final ValueChanged<String>? onMoveDown;

  @override
  Widget build(BuildContext context) => DragTarget<String>(
    onWillAcceptWithDetails: (details) => details.data != node?.id,
    onAcceptWithDetails: (details) {
      final target = node;
      if (target == null) {
        onDropOnSlot(details.data);
      } else {
        onDropOnNode(details.data, target.id);
      }
    },
    builder: (context, candidates, _) {
      final highlighted = candidates.isNotEmpty;
      final current = node;
      if (current == null) {
        return Semantics(
          container: true,
          button: onAddToSlot != null,
          label: anyttyText(context, en: 'Empty menu slot', zh: '空菜单槽位'),
          hint: anyttyText(
            context,
            en: onAddToSlot == null
                ? 'Drop an action here'
                : 'Tap to choose an action, or drop one here',
            zh: onAddToSlot == null ? '将动作拖到这里' : '点击选择操作，或将操作拖到这里',
          ),
          onTap: onAddToSlot,
          child: _EmptyPetalSlot(highlighted: highlighted),
        );
      }
      return LongPressDraggable<String>(
        data: current.id,
        delay: const Duration(milliseconds: 280),
        dragAnchorStrategy: pointerDragAnchorStrategy,
        onDragStarted: hapticsEnabled
            ? () {
                unawaited(HapticFeedback.selectionClick());
              }
            : null,
        feedback: Material(
          color: Colors.transparent,
          child: _PetalBubble(
            node: current,
            highlighted: true,
            dragging: true,
            labelWidth: labelWidth,
            labelLeft: labelLeft,
            labelAbove: labelAbove,
          ),
        ),
        childWhenDragging: Opacity(
          opacity: 0.24,
          child: _PetalBubble(
            node: current,
            highlighted: false,
            dragging: false,
            labelWidth: labelWidth,
            labelLeft: labelLeft,
            labelAbove: labelAbove,
          ),
        ),
        child: Semantics(
          button: true,
          label: _presentation(context, current.id).label,
          hint: anyttyText(
            context,
            en: current.children.isEmpty
                ? 'Tap to move this action to the library; use increase or decrease to reorder'
                : '${current.children.length} inner actions; use increase or decrease to reorder',
            zh: current.children.isEmpty
                ? '点击将此动作移回动作库，也可使用增加或减少操作排序'
                : '包含 ${current.children.length} 个内部操作，也可使用增加或减少操作排序',
          ),
          onTap: current.children.isEmpty
              ? () => onMoveOut(current.id)
              : () => onOpen(current),
          onIncrease: onMoveUp == null ? null : () => onMoveUp!(current.id),
          onDecrease: onMoveDown == null ? null : () => onMoveDown!(current.id),
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: current.children.isEmpty ? null : () => onOpen(current),
            child: _PetalBubble(
              key: ValueKey('petal-editor-action-${current.id}'),
              node: current,
              highlighted: highlighted,
              dragging: false,
              labelWidth: labelWidth,
              labelLeft: labelLeft,
              labelAbove: labelAbove,
            ),
          ),
        ),
      );
    },
  );
}

final class _EmptyPetalSlot extends StatelessWidget {
  const _EmptyPetalSlot({required this.highlighted});

  final bool highlighted;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return AnimatedContainer(
      duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
      decoration: BoxDecoration(
        color: highlighted
            ? palette.accent.withValues(alpha: 0.16)
            : Colors.transparent,
        shape: BoxShape.circle,
        boxShadow: [
          BoxShadow(
            color: highlighted
                ? palette.accent.withValues(alpha: 0.72)
                : palette.track.withValues(alpha: 0.72),
            blurRadius: 0,
            spreadRadius: highlighted ? 2 : 1,
          ),
        ],
      ),
    );
  }
}

final class _PetalBubble extends StatelessWidget {
  const _PetalBubble({
    super.key,
    required this.node,
    required this.highlighted,
    required this.dragging,
    this.labelWidth = 100,
    this.labelLeft = -27,
    this.labelAbove = false,
  });

  final _PetalEditorNode node;
  final bool highlighted;
  final bool dragging;
  final double labelWidth;
  final double labelLeft;
  final bool labelAbove;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final presentation = _presentation(context, node.id);
    final foreground = highlighted ? palette.accentText : palette.text;
    return AnimatedScale(
      scale: highlighted ? 1.06 : 1,
      duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
      child: AnimatedContainer(
        width: _PetalRadialCanvas.nodeExtent,
        height: _PetalRadialCanvas.nodeExtent,
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        padding: EdgeInsets.zero,
        decoration: AnyttyUi.controlDecoration(
          context,
          selected: highlighted || dragging,
          color: highlighted ? palette.accent : palette.surface,
        ),
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Center(child: Icon(presentation.icon, size: 18, color: foreground)),
            Positioned(
              height: MediaQuery.textScalerOf(context).scale(18) * 2 + 4,
              top: labelAbove
                  ? -(MediaQuery.textScalerOf(context).scale(18) * 2 + 4)
                  : _PetalRadialCanvas.nodeExtent + 2,
              left: labelLeft,
              width: labelWidth,
              child: Text(
                presentation.label,
                textAlign: TextAlign.center,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: AnyttyUi.body(context)
                    .copyWith(color: foreground, fontWeight: FontWeight.w600),
              ),
            ),
            if (node.children.isNotEmpty)
              Positioned(
                right: -4,
                top: -4,
                child: Container(
                  width: 18,
                  height: 18,
                  decoration: AnyttyUi.controlDecoration(
                    context,
                    selected: true,
                    color: highlighted ? palette.accentText : palette.accent,
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    '${node.children.length}',
                    style: AnyttyUi.body(context).copyWith(
                      color: highlighted ? palette.accent : palette.accentText,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

final class _PetalEditorHub extends StatelessWidget {
  const _PetalEditorHub({
    required this.parent,
    required this.currentNodes,
    required this.onBack,
    required this.onMoveOut,
  });

  final _PetalEditorNode? parent;
  final List<_PetalEditorNode> currentNodes;
  final VoidCallback? onBack;
  final ValueChanged<String> onMoveOut;

  @override
  Widget build(BuildContext context) => DragTarget<String>(
    onWillAcceptWithDetails: (details) =>
        parent != null && currentNodes.any((node) => node.id == details.data),
    onAcceptWithDetails: (details) => onMoveOut(details.data),
    builder: (context, candidates, _) {
      final palette = AnyttyPalette.of(context);
      final highlighted = candidates.isNotEmpty;
      final label = parent == null
          ? anyttyText(context, en: 'Main', zh: '主菜单')
          : _presentation(context, parent!.id).label;
      return Semantics(
        button: onBack != null,
        label: onBack == null
            ? anyttyText(context, en: 'Main menu', zh: '主菜单')
            : anyttyText(context, en: 'Back to outer menu', zh: '返回上层花瓣'),
        child: AnimatedContainer(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
          decoration: AnyttyUi.controlDecoration(
            context,
            selected: highlighted,
            color: highlighted ? palette.accent : palette.background,
          ),
          child: InkWell(
            key: const ValueKey('petal-editor-hub'),
            customBorder: const CircleBorder(),
            onTap: onBack,
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                Center(
                  child: Icon(
                    parent == null ? LucideIcons.flower2 : LucideIcons.undo2,
                    size: 20,
                    color: highlighted ? palette.accentText : palette.strong,
                  ),
                ),
                Positioned(
                  top: _PetalRadialCanvas.hubExtent + 2,
                  left: -27,
                  width: 100,
                  child: Text(
                    label,
                    textAlign: TextAlign.center,
                    maxLines: 2,
                    softWrap: true,
                    overflow: TextOverflow.ellipsis,
                    style: AnyttyUi.body(context).copyWith(
                      color: highlighted ? palette.accentText : palette.text,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    },
  );
}

final class _PetalActionLibrary extends StatelessWidget {
  const _PetalActionLibrary({
    required this.nodes,
    required this.hapticsEnabled,
    required this.onRemove,
    required this.onAdd,
  });

  final List<_PetalEditorNode> nodes;
  final bool hapticsEnabled;
  final ValueChanged<String> onRemove;
  final ValueChanged<String> onAdd;

  @override
  Widget build(BuildContext context) => DragTarget<String>(
    key: const ValueKey('petal-action-library-drop-zone'),
    onWillAcceptWithDetails: (details) =>
        !nodes.any((node) => node.id == details.data),
    onAcceptWithDetails: (details) => onRemove(details.data),
    builder: (context, candidates, _) {
      final palette = AnyttyPalette.of(context);
      final highlighted = candidates.isNotEmpty;
      return AnimatedContainer(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        padding: const EdgeInsets.all(8),
        decoration: BoxDecoration(
          color: highlighted
              ? palette.accent.withValues(alpha: 0.12)
              : Colors.transparent,
          borderRadius: BorderRadius.circular(14),
          boxShadow: highlighted
              ? [
                  BoxShadow(
                    color: palette.accent.withValues(alpha: 0.34),
                    blurRadius: 8,
                    spreadRadius: 1,
                  ),
                ]
              : null,
        ),
        child: nodes.isEmpty
            ? ConstrainedBox(
                constraints: const BoxConstraints(minHeight: 78),
                child: Center(
                  child: Text(
                    anyttyText(
                      context,
                      en: 'All actions are in the menu',
                      zh: '所有动作均已加入菜单',
                    ),
                    style: AnyttyUi.muted(context),
                  ),
                ),
              )
            : LayoutBuilder(
                builder: (context, constraints) {
                  final singleColumn =
                      constraints.maxWidth < 420 ||
                      MediaQuery.textScalerOf(context).scale(1) > 1.5;
                  final width = singleColumn
                      ? constraints.maxWidth
                      : (constraints.maxWidth - 8) / 2;
                  final lineHeight =
                      MediaQuery.textScalerOf(context).scale(14.5) *
                      (18 / 14.5);
                  final tileHeight = lineHeight > 22
                      ? lineHeight * 2 + 52
                      : AnyttyUi.controlHeight(context) + 18;
                  return Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      for (final node in nodes)
                        SizedBox(
                          width: width,
                          height: tileHeight,
                          child: _LibraryActionTile(
                            node: node,
                            hapticsEnabled: hapticsEnabled,
                            onAdd: () => onAdd(node.id),
                          ),
                        ),
                    ],
                  );
                },
              ),
      );
    },
  );
}

final class _LibraryActionTile extends StatelessWidget {
  const _LibraryActionTile({
    required this.node,
    required this.hapticsEnabled,
    required this.onAdd,
  });

  final _PetalEditorNode node;
  final bool hapticsEnabled;
  final VoidCallback onAdd;

  @override
  Widget build(BuildContext context) {
    final tile = _LibraryTileContent(node: node);
    return LongPressDraggable<String>(
      data: node.id,
      delay: const Duration(milliseconds: 280),
      dragAnchorStrategy: pointerDragAnchorStrategy,
      onDragStarted: hapticsEnabled
          ? () {
              unawaited(HapticFeedback.selectionClick());
            }
          : null,
      feedback: Material(
        color: Colors.transparent,
        child: SizedBox(
          width: 180,
          height: MediaQuery.textScalerOf(context).scale(14.5) > 22
              ? MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5) * 2 +
                    52
              : AnyttyUi.controlHeight(context) + 18,
          child: tile,
        ),
      ),
      childWhenDragging: Opacity(opacity: 0.25, child: tile),
      child: Semantics(
        button: true,
        label: anyttyText(
          context,
          en: 'Add ${_presentation(context, node.id).label}',
          zh: '添加${_presentation(context, node.id).label}',
        ),
        child: InkWell(
          key: ValueKey('petal-library-action-${node.id}'),
          borderRadius: BorderRadius.circular(14),
          onTap: onAdd,
          child: tile,
        ),
      ),
    );
  }
}

final class _LibraryTileContent extends StatelessWidget {
  const _LibraryTileContent({required this.node});

  final _PetalEditorNode node;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final presentation = _presentation(context, node.id);
    return Container(
      padding: const EdgeInsets.fromLTRB(9, 7, 8, 7),
      decoration: AnyttyUi.cardDecoration(
        context,
        radius: 14,
        depth: 1,
        color: palette.surface,
      ),
      child: Row(
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              borderRadius: BorderRadius.circular(8),
            ),
            alignment: Alignment.center,
            child: Icon(presentation.icon, size: 18, color: palette.strong),
          ),
          const SizedBox(width: 9),
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Flexible(
                  child: Text(
                    presentation.label,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: AnyttyUi.body(context)
                        .copyWith(fontWeight: FontWeight.w600),
                  ),
                ),
                const SizedBox(height: 2),
                Flexible(
                  child: Text(
                    presentation.group,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: AnyttyUi.muted(context),
                  ),
                ),
              ],
            ),
          ),
          Icon(LucideIcons.gripVertical, size: 17, color: palette.strong),
        ],
      ),
    );
  }
}

final class _PageButton extends StatelessWidget {
  const _PageButton({
    super.key,
    required this.icon,
    required this.enabled,
    required this.onPressed,
  });

  final IconData icon;
  final bool enabled;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => AnyttyIconButton(
    tooltip: anyttyText(
      context,
      en: icon == LucideIcons.chevronLeft ? 'Previous page' : 'Next page',
      zh: icon == LucideIcons.chevronLeft ? '上一页' : '下一页',
    ),
    onPressed: enabled ? onPressed : null,
    icon: icon,
    iconSize: 18,
  );
}

final class _PetalEditorGuidePainter extends CustomPainter {
  const _PetalEditorGuidePainter({
    required this.center,
    required this.positions,
    required this.color,
  });

  final Offset center;
  final List<Offset> positions;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    if (positions.isEmpty) return;
    final paint = Paint()
      ..color = color.withValues(alpha: 0.34)
      ..strokeWidth = 1
      ..style = PaintingStyle.stroke;
    for (final position in positions) {
      canvas.drawLine(center, position, paint);
    }
    final horizontalRadius = (positions[2].dx - center.dx).abs();
    final verticalRadius = (positions.first.dy - center.dy).abs();
    canvas.drawOval(
      Rect.fromCenter(
        center: center,
        width: horizontalRadius * 2,
        height: verticalRadius * 2,
      ),
      paint,
    );
  }

  @override
  bool shouldRepaint(covariant _PetalEditorGuidePainter oldDelegate) =>
      oldDelegate.center != center ||
      oldDelegate.positions != positions ||
      oldDelegate.color != color;
}

final class _PetalEditorNode {
  const _PetalEditorNode({
    required this.id,
    required this.depth,
    required this.children,
  });

  final String id;
  final int depth;
  final List<_PetalEditorNode> children;
}

typedef _PetalLocation = ({
  List<String> path,
  _PetalEditorNode? parent,
  List<_PetalEditorNode> children,
});

_PetalLocation _resolveLocation(
  List<_PetalEditorNode> roots,
  List<String> requestedPath,
) {
  final validPath = <String>[];
  var children = roots;
  _PetalEditorNode? parent;
  for (final id in requestedPath) {
    final next = children.where((node) => node.id == id).firstOrNull;
    if (next == null || next.children.isEmpty) break;
    validPath.add(id);
    parent = next;
    children = next.children;
  }
  return (
    path: List.unmodifiable(validPath),
    parent: parent,
    children: children,
  );
}

List<_PetalEditorNode> _petalTree(List<TerminalPetalMenuPlacement> layout) {
  var cursor = 0;
  List<_PetalEditorNode> buildLevel(int depth) {
    final result = <_PetalEditorNode>[];
    while (cursor < layout.length && layout[cursor].depth == depth) {
      final placement = layout[cursor];
      cursor += 1;
      final children = cursor < layout.length && layout[cursor].depth > depth
          ? buildLevel(depth + 1)
          : const <_PetalEditorNode>[];
      result.add(
        _PetalEditorNode(
          id: placement.id,
          depth: placement.depth,
          children: children,
        ),
      );
    }
    return List.unmodifiable(result);
  }

  return buildLevel(0);
}

List<_PetalEditorNode> _flattenPetals(List<_PetalEditorNode> roots) => [
  for (final node in roots) ...[node, ..._flattenPetals(node.children)],
];

_PetalEditorNode? _findNode(List<_PetalEditorNode> roots, String id) {
  for (final node in roots) {
    if (node.id == id) return node;
    final child = _findNode(node.children, id);
    if (child != null) return child;
  }
  return null;
}

String? _parentIdFor(List<_PetalEditorNode> roots, String id) {
  for (final node in roots) {
    if (node.children.any((child) => child.id == id)) return node.id;
    final parent = _parentIdFor(node.children, id);
    if (parent != null) return parent;
  }
  return null;
}

List<String>? _pathToNode(List<_PetalEditorNode> roots, String id) {
  for (final node in roots) {
    if (node.id == id) return [id];
    final childPath = _pathToNode(node.children, id);
    if (childPath != null) return [node.id, ...childPath];
  }
  return null;
}

Set<String> _subtreeIds(_PetalEditorNode node) => {
  node.id,
  for (final child in node.children) ..._subtreeIds(child),
};

List<TerminalPetalMenuPlacement> _moveToParent(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
  required String? parentActionId,
}) {
  if (parentActionId != null) {
    return reparentTerminalPetalSubtree(
      layout,
      actionId: actionId,
      parentActionId: parentActionId,
    );
  }
  var moved = layout;
  for (var guard = 0; guard <= terminalPetalMaximumDepth; guard += 1) {
    final placement = moved.where((item) => item.id == actionId).firstOrNull;
    if (placement == null || placement.depth == 0) break;
    final next = outdentTerminalPetalSubtree(moved, actionId: actionId);
    if (_sameLayout(moved, next)) break;
    moved = next;
  }
  return moved;
}

List<TerminalPetalMenuPlacement> _moveSiblingToIndex(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
  required String? parentActionId,
  required int targetIndex,
}) {
  var moved = layout;
  for (var guard = 0; guard < terminalPetalActionIds.length; guard += 1) {
    final roots = _petalTree(moved);
    final siblings = parentActionId == null
        ? roots
        : _findNode(roots, parentActionId)?.children ??
              const <_PetalEditorNode>[];
    final sourceIndex = siblings.indexWhere((node) => node.id == actionId);
    if (sourceIndex < 0 || siblings.isEmpty) break;
    final destination = targetIndex.clamp(0, siblings.length - 1);
    if (sourceIndex == destination) break;
    moved = moveTerminalPetalSubtree(
      moved,
      actionId: actionId,
      offset: destination < sourceIndex ? -1 : 1,
    );
  }
  return moved;
}

bool _sameLayout(
  List<TerminalPetalMenuPlacement> left,
  List<TerminalPetalMenuPlacement> right,
) {
  if (identical(left, right)) return true;
  if (left.length != right.length) return false;
  for (var index = 0; index < left.length; index += 1) {
    if (left[index] != right[index]) return false;
  }
  return true;
}

typedef _PetalActionPresentation = ({
  String label,
  IconData icon,
  String group,
});

IconData _iconForPetal(String id) => switch (id) {
  'history' => LucideIcons.history,
  'search' => LucideIcons.search,
  'more' => LucideIcons.ellipsis,
  'selection' => LucideIcons.scanText,
  'paste' => LucideIcons.clipboardPaste,
  'enter' => LucideIcons.cornerDownLeft,
  'escape' => LucideIcons.badgeX,
  'resources' => LucideIcons.activity,
  'command-bar' => LucideIcons.slidersHorizontal,
  'copy-screen' => LucideIcons.copy,
  'files' => LucideIcons.folderOpen,
  'input-tools' => LucideIcons.command,
  'quick-keys' => LucideIcons.zap,
  'keyboard' => LucideIcons.keyboard,
  'tab' => LucideIcons.arrowRightToLine,
  'backspace' => LucideIcons.delete,
  'delete' => LucideIcons.eraser,
  'interrupt' => LucideIcons.circleStop,
  'eof' => LucideIcons.logOut,
  'suspend' => LucideIcons.pause,
  'clear' => LucideIcons.eraser,
  'navigation-tools' => LucideIcons.navigation,
  'arrow-left' => LucideIcons.arrowLeft,
  'arrow-down' => LucideIcons.arrowDown,
  'arrow-up' => LucideIcons.arrowUp,
  'arrow-right' => LucideIcons.arrowRight,
  'home' => LucideIcons.home,
  'end' => LucideIcons.arrowRightToLine,
  'page-up' => LucideIcons.chevronsUp,
  'page-down' => LucideIcons.chevronsDown,
  'session-tools' => LucideIcons.panelsTopLeft,
  'split' => LucideIcons.columns2,
  'split-rows' => LucideIcons.rows2,
  'split-columns' => LucideIcons.columns2,
  'sync-input' => LucideIcons.gitCompareArrows,
  'resize' => LucideIcons.maximize2,
  'reconnect' => LucideIcons.refreshCw,
  'settings' => LucideIcons.settings,
  _ => LucideIcons.circleDot,
};

String _groupForPetal(BuildContext context, String id) => switch (id) {
  'history' || 'search' => anyttyText(context, en: 'History', zh: '历史'),
  'selection' ||
  'paste' ||
  'copy-screen' => anyttyText(context, en: 'Edit', zh: '编辑'),
  'quick-keys' ||
  'keyboard' ||
  'input-tools' => anyttyText(context, en: 'Input', zh: '输入'),
  'enter' ||
  'escape' ||
  'tab' ||
  'backspace' ||
  'delete' ||
  'interrupt' ||
  'eof' ||
  'suspend' ||
  'clear' => anyttyText(context, en: 'Keys', zh: '按键'),
  'navigation-tools' ||
  'arrow-left' ||
  'arrow-down' ||
  'arrow-up' ||
  'arrow-right' ||
  'home' ||
  'end' ||
  'page-up' ||
  'page-down' => anyttyText(context, en: 'Navigation', zh: '导航'),
  'session-tools' ||
  'split' ||
  'split-rows' ||
  'split-columns' ||
  'sync-input' ||
  'resize' ||
  'reconnect' => anyttyText(context, en: 'Layout', zh: '布局'),
  _ => anyttyText(context, en: 'Terminal', zh: '终端'),
};

_PetalActionPresentation _presentation(BuildContext context, String id) => (
  label: switch (id) {
    'history' => anyttyText(context, en: 'History', zh: '历史记录'),
    'search' => anyttyText(context, en: 'Search', zh: '搜索'),
    'more' => anyttyText(context, en: 'More', zh: '更多'),
    'selection' => anyttyText(context, en: 'Select', zh: '选择'),
    'paste' => anyttyText(context, en: 'Paste', zh: '粘贴'),
    'enter' => anyttyText(context, en: 'Enter', zh: '回车'),
    'escape' => anyttyText(context, en: 'Escape', zh: '退出键'),
    'resources' => anyttyText(context, en: 'Resources', zh: '资源快照'),
    'command-bar' => anyttyText(context, en: 'Shortcut bar', zh: '快捷栏'),
    'copy-screen' => anyttyText(context, en: 'Copy screen', zh: '复制当前屏幕'),
    'files' => anyttyText(context, en: 'Current folder', zh: '当前目录'),
    'input-tools' => anyttyText(context, en: 'Input actions', zh: '输入操作'),
    'quick-keys' => anyttyText(context, en: 'Quick Keys', zh: '快捷键面板'),
    'keyboard' => anyttyText(context, en: 'Keyboard', zh: '系统键盘'),
    'tab' => anyttyText(context, en: 'Tab', zh: '制表键'),
    'backspace' => anyttyText(context, en: 'Backspace', zh: '退格'),
    'delete' => anyttyText(context, en: 'Delete', zh: '删除'),
    'interrupt' => anyttyText(
      context,
      en: 'Interrupt (Ctrl+C)',
      zh: '中断（Ctrl+C）',
    ),
    'eof' => anyttyText(context, en: 'EOF (Ctrl+D)', zh: '结束输入（Ctrl+D）'),
    'suspend' => anyttyText(context, en: 'Suspend (Ctrl+Z)', zh: '挂起（Ctrl+Z）'),
    'clear' => anyttyText(context, en: 'Clear screen', zh: '清屏'),
    'navigation-tools' => anyttyText(
      context,
      en: 'Navigation keys',
      zh: '导航按键',
    ),
    'arrow-left' => anyttyText(context, en: 'Left', zh: '向左'),
    'arrow-down' => anyttyText(context, en: 'Down', zh: '向下'),
    'arrow-up' => anyttyText(context, en: 'Up', zh: '向上'),
    'arrow-right' => anyttyText(context, en: 'Right', zh: '向右'),
    'home' => anyttyText(context, en: 'Home', zh: '行首'),
    'end' => anyttyText(context, en: 'End', zh: '行尾'),
    'page-up' => anyttyText(context, en: 'Page up', zh: '向上翻页'),
    'page-down' => anyttyText(context, en: 'Page down', zh: '向下翻页'),
    'session-tools' => anyttyText(context, en: 'Layout actions', zh: '布局操作'),
    'split' => anyttyText(context, en: 'Split terminal', zh: '分屏终端'),
    'split-rows' => anyttyText(context, en: 'Top / bottom split', zh: '上下分屏'),
    'split-columns' => anyttyText(
      context,
      en: 'Left / right split',
      zh: '左右分屏',
    ),
    'sync-input' => anyttyText(context, en: 'Synchronized input', zh: '同步输入'),
    'resize' => anyttyText(context, en: 'Resize', zh: '调整尺寸'),
    'reconnect' => anyttyText(context, en: 'Reconnect', zh: '重新连接'),
    'settings' => anyttyText(context, en: 'Terminal settings', zh: '终端设置'),
    _ => id,
  },
  icon: _iconForPetal(id),
  group: _groupForPetal(context, id),
);
