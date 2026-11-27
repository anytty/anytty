part of 'terminal_petal_menu_settings_screen.dart';


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
                  ? 0.14
                  : 0.05,
            ),
            blurRadius: 18,
            offset: const Offset(0, 7),
          ),
        ],
      ),
      child: Column(
        children: [
          SizedBox(
            height: 54,
            child: MediaQuery.withClampedTextScaling(
              maxScaleFactor: 1.25,
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
                        color: palette.accent,
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Text(
                            title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          Text(
                            anyttyText(
                              context,
                              en: 'Level $level',
                              zh: '第 $level 层',
                            ),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 10,
                            ),
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
                      style: TextStyle(
                        color: palette.muted,
                        fontSize: 10,
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
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
          ),
          Divider(height: 1, color: palette.border),
          SizedBox(
            height: 292,
            child: ColoredBox(
              color: palette.surfaceRaised.withValues(alpha: 0.72),
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
                      pageStart: start,
                      hapticsEnabled: hapticsEnabled,
                      onOpen: onOpen,
                      onBack: onBack,
                      onDropOnNode: onDropOnNode,
                      onDropOnSlot: onDropOnSlot,
                      onMoveOut: onMoveOut,
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
    required this.pageStart,
    required this.hapticsEnabled,
    required this.onOpen,
    required this.onBack,
    required this.onDropOnNode,
    required this.onDropOnSlot,
    required this.onMoveOut,
  });

  static const nodeExtent = 58.0;
  static const hubExtent = 72.0;

  final _PetalEditorNode? parent;
  final List<_PetalEditorNode> nodes;
  final int pageStart;
  final bool hapticsEnabled;
  final ValueChanged<_PetalEditorNode> onOpen;
  final VoidCallback? onBack;
  final void Function(String sourceId, String targetId) onDropOnNode;
  final void Function(String sourceId, int targetIndex) onDropOnSlot;
  final ValueChanged<String> onMoveOut;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final size = constraints.biggest;
      final center = Offset(size.width / 2, size.height / 2);
      final radiusX = math.max(0.0, (size.width - nodeExtent) / 2 - 5);
      final radiusY = math.max(0.0, (size.height - nodeExtent) / 2 - 8);
      final positions = List.generate(8, (index) {
        final angle = -math.pi / 2 + index * math.pi / 4;
        return center +
            Offset(math.cos(angle) * radiusX, math.sin(angle) * radiusY);
      }, growable: false);
      final palette = AnyttyPalette.of(context);
      return Stack(
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
  });

  final _PetalEditorNode? node;
  final bool hapticsEnabled;
  final ValueChanged<_PetalEditorNode> onOpen;
  final void Function(String sourceId, String targetId) onDropOnNode;
  final ValueChanged<String> onDropOnSlot;

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
      if (current == null) return _EmptyPetalSlot(highlighted: highlighted);
      return LongPressDraggable<String>(
        data: current.id,
        delay: const Duration(milliseconds: 280),
        dragAnchorStrategy: pointerDragAnchorStrategy,
        onDragStarted: hapticsEnabled
            ? () => unawaited(HapticFeedback.selectionClick())
            : null,
        feedback: Material(
          color: Colors.transparent,
          child: _PetalBubble(node: current, highlighted: true, dragging: true),
        ),
        childWhenDragging: Opacity(
          opacity: 0.24,
          child: _PetalBubble(
            node: current,
            highlighted: false,
            dragging: false,
          ),
        ),
        child: Semantics(
          button: current.children.isNotEmpty,
          label: _presentation(context, current.id).label,
          hint: current.children.isEmpty
              ? null
              : anyttyText(
                  context,
                  en: '${current.children.length} inner actions',
                  zh: '包含 ${current.children.length} 个内部操作',
                ),
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: current.children.isEmpty ? null : () => onOpen(current),
            child: _PetalBubble(
              key: ValueKey('petal-editor-action-${current.id}'),
              node: current,
              highlighted: highlighted,
              dragging: false,
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
        border: Border.all(
          color: highlighted ? palette.accent : palette.borderStrong,
          width: highlighted ? 2 : 1,
        ),
      ),
    );
  }
}
