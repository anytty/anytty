part of 'terminal_petal_menu_settings_screen.dart';


final class _PetalBubble extends StatelessWidget {
  const _PetalBubble({
    super.key,
    required this.node,
    required this.highlighted,
    required this.dragging,
  });

  final _PetalEditorNode node;
  final bool highlighted;
  final bool dragging;

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
        padding: const EdgeInsets.fromLTRB(5, 8, 5, 6),
        decoration: BoxDecoration(
          color: highlighted ? palette.accent : palette.surface,
          shape: BoxShape.circle,
          border: Border.all(
            color: highlighted ? palette.accent : palette.borderStrong,
            width: highlighted ? 2 : 1,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(
                alpha: dragging || highlighted ? 0.25 : 0.12,
              ),
              blurRadius: dragging || highlighted ? 14 : 7,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(presentation.icon, size: 18, color: foreground),
                const SizedBox(height: 2),
                SizedBox(
                  width: 48,
                  height: 12,
                  child: FittedBox(
                    fit: BoxFit.scaleDown,
                    child: Text(
                      presentation.label,
                      maxLines: 1,
                      style: TextStyle(
                        color: foreground,
                        fontSize: 9,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                ),
              ],
            ),
            if (node.children.isNotEmpty)
              Positioned(
                right: -4,
                top: -4,
                child: Container(
                  width: 18,
                  height: 18,
                  decoration: BoxDecoration(
                    color: highlighted ? palette.accentText : palette.accent,
                    shape: BoxShape.circle,
                    border: Border.all(color: palette.surface, width: 1.5),
                  ),
                  alignment: Alignment.center,
                  child: Text(
                    '${node.children.length}',
                    style: TextStyle(
                      color: highlighted ? palette.accent : palette.accentText,
                      fontSize: 9,
                      fontWeight: FontWeight.w800,
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
          decoration: BoxDecoration(
            color: highlighted ? palette.accent : palette.background,
            shape: BoxShape.circle,
            border: Border.all(
              color: highlighted ? palette.accent : palette.borderStrong,
              width: highlighted ? 2 : 1.5,
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: 0.12),
                blurRadius: 10,
                offset: const Offset(0, 4),
              ),
            ],
          ),
          child: InkWell(
            key: const ValueKey('petal-editor-hub'),
            customBorder: const CircleBorder(),
            onTap: onBack,
            child: Padding(
              padding: const EdgeInsets.all(9),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(
                    parent == null ? LucideIcons.flower2 : LucideIcons.undo2,
                    size: 20,
                    color: highlighted ? palette.accentText : palette.accent,
                  ),
                  const SizedBox(height: 3),
                  SizedBox(
                    width: 48,
                    height: 12,
                    child: FittedBox(
                      fit: BoxFit.scaleDown,
                      child: Text(
                        label,
                        maxLines: 1,
                        style: TextStyle(
                          color: highlighted
                              ? palette.accentText
                              : palette.text,
                          fontSize: 9,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
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
              : palette.surfaceRaised.withValues(alpha: 0.58),
          border: Border.all(
            color: highlighted ? palette.accent : palette.border,
            width: highlighted ? 2 : 1,
          ),
          borderRadius: BorderRadius.circular(8),
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
                    style: TextStyle(color: palette.muted, fontSize: 12),
                  ),
                ),
              )
            : LayoutBuilder(
                builder: (context, constraints) {
                  final singleColumn =
                      constraints.maxWidth < 340 ||
                      MediaQuery.textScalerOf(context).scale(1) > 1.5;
                  final width = singleColumn
                      ? constraints.maxWidth
                      : (constraints.maxWidth - 8) / 2;
                  return Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      for (final node in nodes)
                        SizedBox(
                          width: width,
                          height: singleColumn ? 70 : 62,
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
          ? () => unawaited(HapticFeedback.selectionClick())
          : null,
      feedback: Material(
        color: Colors.transparent,
        child: SizedBox(width: 180, height: 62, child: tile),
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
          borderRadius: BorderRadius.circular(8),
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
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.04),
            blurRadius: 8,
            offset: const Offset(0, 3),
          ),
        ],
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
            child: Icon(presentation.icon, size: 18, color: palette.muted),
          ),
          const SizedBox(width: 9),
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  presentation.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  presentation.group,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: palette.muted, fontSize: 10),
                ),
              ],
            ),
          ),
          Icon(LucideIcons.gripVertical, size: 17, color: palette.faint),
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
  Widget build(BuildContext context) => IconButton(
    visualDensity: VisualDensity.compact,
    constraints: const BoxConstraints.tightFor(width: 36, height: 48),
    onPressed: enabled ? onPressed : null,
    icon: Icon(icon, size: 18),
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
  'preview' => LucideIcons.layoutGrid,
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
  'preview' ||
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
    'preview' => anyttyText(
      context,
      en: 'Terminal preview',
      zh: '多终端预览',
    ),
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
