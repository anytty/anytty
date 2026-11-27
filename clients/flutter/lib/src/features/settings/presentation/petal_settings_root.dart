part of 'terminal_petal_menu_settings_screen.dart';


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

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: anyttyText(context, en: 'Back to settings', zh: '返回设置'),
          onPressed: context.pop,
          icon: const Icon(LucideIcons.chevronLeft),
        ),
        titleSpacing: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              anyttyText(context, en: 'Petal menu', zh: '花瓣菜单'),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            Text(
              anyttyText(
                context,
                en: 'Terminal long-press actions',
                zh: '终端长按操作',
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: AnyttyPalette.of(context).muted,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
        ),
        actions: [
          MediaQuery.withClampedTextScaling(
            maxScaleFactor: 1.1,
            child: TextButton.icon(
              key: const ValueKey('petal-menu-restore-defaults'),
              onPressed: preferences == TerminalPetalMenuPreferences.defaults
                  ? null
                  : () => unawaited(_confirmRestoreDefaults(preferences)),
              icon: const Icon(LucideIcons.rotateCcw, size: 17),
              label: Text(
                anyttyText(context, en: 'Restore default', zh: '恢复默认'),
              ),
            ),
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
      builder: (dialogContext) => AlertDialog(
        title: Text(
          anyttyText(
            dialogContext,
            en: 'Restore the default petal menu?',
            zh: '恢复花瓣菜单默认配置？',
          ),
        ),
        content: Text(
          anyttyText(
            dialogContext,
            en: 'This replaces the current order, levels, hidden actions, and menu switches with the built-in configuration.',
            zh: '这会用内置配置替换当前的动作顺序、层级、隐藏动作和花瓣菜单开关。',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(anyttyText(dialogContext, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            key: const ValueKey('petal-menu-restore-confirm'),
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(
              anyttyText(dialogContext, en: 'Restore default', zh: '恢复默认'),
            ),
          ),
        ],
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
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Expanded(
          child: Text(
            label,
            style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
          ),
        ),
        Text(
          trailing,
          style: TextStyle(
            color: palette.muted,
            fontSize: 11,
            fontFeatures: const [FontFeature.tabularFigures()],
          ),
        ),
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
    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(
              alpha: Theme.of(context).brightness == Brightness.dark
                  ? 0.12
                  : 0.04,
            ),
            blurRadius: 16,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: Column(
        children: [
          for (var index = 0; index < children.length; index += 1) ...[
            children[index],
            if (index != children.length - 1)
              Divider(height: 1, indent: 58, color: palette.border),
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
    return ConstrainedBox(
      constraints: const BoxConstraints(minHeight: 66),
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
              child: Icon(icon, size: 18, color: palette.muted),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    label,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    description,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: palette.muted, fontSize: 11),
                  ),
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
