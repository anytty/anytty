import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../data/terminal_quick_keys_layout_store.dart';
import '../domain/terminal_quick_action.dart';
import 'terminal_command_bar.dart';

final class TerminalHeaderQuickKeysLayer extends StatelessWidget {
  const TerminalHeaderQuickKeysLayer({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) =>
      Positioned(left: 0, top: 0, right: 0, child: child);
}

final class TerminalQuickKeysPanel extends StatefulWidget {
  const TerminalQuickKeysPanel({
    super.key,
    required this.actions,
    required this.inputEnabled,
    required this.onClose,
    required this.onAction,
    this.attachedToHeader = false,
    this.layoutStore = const TerminalQuickKeysLayoutStore(),
  });

  final List<TerminalQuickAction> actions;
  final bool inputEnabled;
  final VoidCallback onClose;
  final ValueChanged<TerminalQuickAction> onAction;
  final bool attachedToHeader;
  final TerminalQuickKeysLayoutStore layoutStore;

  @override
  State<TerminalQuickKeysPanel> createState() => _TerminalQuickKeysPanelState();
}

final class _TerminalQuickKeysPanelState extends State<TerminalQuickKeysPanel> {
  late List<String> _layoutIds;
  Future<void> _saveTail = Future.value();
  bool _dragChanged = false;
  int _loadEpoch = 0;

  @override
  void initState() {
    super.initState();
    _layoutIds = widget.actions.map((action) => action.id).toList();
    unawaited(_loadLayout());
  }

  @override
  void didUpdateWidget(TerminalQuickKeysPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!identical(oldWidget.actions, widget.actions)) {
      _layoutIds = _reconciledIds(_layoutIds);
    }
  }

  Future<void> _loadLayout() async {
    final epoch = ++_loadEpoch;
    final stored = await widget.layoutStore.load();
    if (!mounted || epoch != _loadEpoch) return;
    setState(() => _layoutIds = _reconciledIds(stored));
  }

  List<String> _reconciledIds(List<String> preferred) =>
      terminalQuickKeysInLayoutOrder(
        actions: widget.actions,
        layoutIds: preferred,
      ).map((action) => action.id).toList(growable: false);

  List<TerminalQuickAction> get _orderedActions =>
      terminalQuickKeysInLayoutOrder(
        actions: widget.actions,
        layoutIds: _layoutIds,
      );

  void _move(String draggedId, String targetId) {
    final byId = {for (final action in widget.actions) action.id: action};
    if (byId[draggedId]?.kind != byId[targetId]?.kind) return;
    final next = swapTerminalQuickKeyLayout(
      layoutIds: _reconciledIds(_layoutIds),
      draggedId: draggedId,
      targetId: targetId,
    );
    if (_sameIds(next, _layoutIds)) return;
    setState(() {
      _layoutIds = next;
      _dragChanged = true;
    });
  }

  void _finishDrag() {
    if (!_dragChanged) return;
    _dragChanged = false;
    HapticFeedback.selectionClick();
    final snapshot = List<String>.unmodifiable(_reconciledIds(_layoutIds));
    _saveTail = _saveTail.then(
      (_) => widget.layoutStore.save(snapshot),
      onError: (_) => widget.layoutStore.save(snapshot),
    );
  }

  bool _sameIds(List<String> left, List<String> right) {
    if (left.length != right.length) return false;
    for (var index = 0; index < left.length; index += 1) {
      if (left[index] != right[index]) return false;
    }
    return true;
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    const columns = 4;
    final groups = [
      for (final kind in TerminalQuickActionKind.values)
        (
          kind: kind,
          actions: _orderedActions
              .where((action) => action.kind == kind)
              .toList(growable: false),
        ),
    ].where((group) => group.actions.isNotEmpty).toList(growable: false);
    final rowCount = groups.fold<int>(
      0,
      (count, group) => count + (group.actions.length / columns).ceil(),
    );
    final preferredHeight = 46.0 + groups.length * 25 + rowCount * 42;
    final height = preferredHeight.clamp(
      104.0,
      MediaQuery.sizeOf(context).height * 0.42,
    );
    return SizedBox(
      key: const ValueKey('terminal-quick-action-panel'),
      height: height,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: palette.surface,
          border: widget.attachedToHeader
              ? Border(bottom: BorderSide(color: palette.borderStrong))
              : Border.all(color: palette.borderStrong),
          borderRadius: widget.attachedToHeader
              ? const BorderRadius.vertical(bottom: Radius.circular(12))
              : const BorderRadius.all(Radius.circular(10)),
          boxShadow: widget.attachedToHeader
              ? const [
                  BoxShadow(
                    color: Color(0x30000000),
                    blurRadius: 12,
                    offset: Offset(0, 5),
                  ),
                ]
              : const [
                  BoxShadow(
                    color: Color(0x38000000),
                    blurRadius: 8,
                    offset: Offset(0, 2),
                  ),
                ],
        ),
        child: Column(
          children: [
            SizedBox(
              height: 40,
              child: Row(
                children: [
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      'Quick keys',
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 10,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: 'Close quick keys',
                    onPressed: widget.onClose,
                    icon: const Icon(LucideIcons.x, size: 16),
                    color: palette.muted,
                    visualDensity: VisualDensity.compact,
                  ),
                  const SizedBox(width: 2),
                ],
              ),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: ListView.builder(
                padding: const EdgeInsets.fromLTRB(6, 5, 6, 7),
                physics: const BouncingScrollPhysics(),
                itemCount: groups.length,
                itemBuilder: (context, groupIndex) {
                  final group = groups[groupIndex];
                  return _QuickKeyGroup(
                    kind: group.kind,
                    actions: group.actions,
                    inputEnabled: widget.inputEnabled,
                    onMove: _move,
                    onDragEnd: _finishDrag,
                    onAction: widget.onAction,
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _QuickKeyGroup extends StatelessWidget {
  const _QuickKeyGroup({
    required this.kind,
    required this.actions,
    required this.inputEnabled,
    required this.onMove,
    required this.onDragEnd,
    required this.onAction,
  });

  final TerminalQuickActionKind kind;
  final List<TerminalQuickAction> actions;
  final bool inputEnabled;
  final void Function(String draggedId, String targetId) onMove;
  final VoidCallback onDragEnd;
  final ValueChanged<TerminalQuickAction> onAction;

  String get _label => switch (kind) {
    TerminalQuickActionKind.key => 'KEY',
    TerminalQuickActionKind.chord => 'CHORD',
    TerminalQuickActionKind.text => 'TEXT',
  };

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 5),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(3, 2, 3, 4),
            child: Text(
              _label,
              style: TextStyle(
                color: palette.faint,
                fontSize: 8,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          LayoutBuilder(
            builder: (context, constraints) {
              const spacing = 4.0;
              final tileWidth = math.max(
                40.0,
                (constraints.maxWidth - spacing * 3) / 4,
              );
              return Wrap(
                spacing: spacing,
                runSpacing: spacing,
                children: [
                  for (final action in actions)
                    SizedBox(
                      key: ValueKey('terminal-quick-key-${action.id}'),
                      width: tileWidth,
                      height: 38,
                      child: DragTarget<String>(
                        onWillAcceptWithDetails: (details) {
                          final source = actions
                              .where((item) => item.id == details.data)
                              .firstOrNull;
                          return source != null && source.id != action.id;
                        },
                        onMove: (details) => onMove(details.data, action.id),
                        builder: (context, candidates, rejected) =>
                            LongPressDraggable<String>(
                              data: action.id,
                              delay: const Duration(milliseconds: 260),
                              hapticFeedbackOnStart: true,
                              onDragEnd: (_) => onDragEnd(),
                              feedback: Material(
                                color: Colors.transparent,
                                child: SizedBox(
                                  width: tileWidth,
                                  height: 38,
                                  child: _QuickKeyButton(
                                    action: action,
                                    enabled: false,
                                    elevated: true,
                                  ),
                                ),
                              ),
                              childWhenDragging: Opacity(
                                opacity: 0.32,
                                child: _QuickKeyButton(
                                  action: action,
                                  enabled: false,
                                ),
                              ),
                              child: _QuickKeyButton(
                                action: action,
                                enabled: inputEnabled,
                                highlighted: candidates.isNotEmpty,
                                onPressed: () {
                                  HapticFeedback.selectionClick();
                                  onAction(action);
                                },
                              ),
                            ),
                      ),
                    ),
                ],
              );
            },
          ),
        ],
      ),
    );
  }
}

final class _QuickKeyButton extends StatelessWidget {
  const _QuickKeyButton({
    required this.action,
    required this.enabled,
    this.onPressed,
    this.highlighted = false,
    this.elevated = false,
  });

  final TerminalQuickAction action;
  final bool enabled;
  final VoidCallback? onPressed;
  final bool highlighted;
  final bool elevated;

  @override
  Widget build(BuildContext context) {
    final colors = terminalQuickActionColors(
      action,
      enabled: enabled,
      palette: AnyttyPalette.of(context),
    );
    return Tooltip(
      message: action.accessibilityLabel,
      child: DecoratedBox(
        decoration: elevated
            ? const BoxDecoration(
                boxShadow: [BoxShadow(color: Color(0x66000000), blurRadius: 8)],
              )
            : const BoxDecoration(),
        child: OutlinedButton(
          onPressed: enabled ? onPressed : null,
          style: OutlinedButton.styleFrom(
            foregroundColor: colors.foreground,
            backgroundColor: highlighted
                ? colors.border.withValues(alpha: 0.22)
                : colors.background,
            disabledForegroundColor: colors.foreground,
            disabledBackgroundColor: colors.background,
            side: BorderSide(
              color: highlighted ? colors.foreground : colors.border,
            ),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(4),
            ),
            padding: const EdgeInsets.symmetric(horizontal: 4),
          ),
          child: Text(
            action.displayLabel,
            semanticsLabel: action.accessibilityLabel,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 9,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}
