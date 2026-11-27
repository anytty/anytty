import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../domain/live_screen_store.dart';
import '../domain/terminal_settings.dart';
import 'terminal_canvas.dart';

/// 全屏毛玻璃预览层：终端 / 会话两个页签，替换原来的底部弹框。
///
/// 设计稿 v0.2：底层终端缩到 0.945（由调用方做），这里负责玻璃、卡片网格、
/// 错峰入场和点选收束。缩略图复用 TerminalCanvas，按屏幕尺寸缩放绘制。

const String terminalPreviewTabTerminals = 'terminals';
const String terminalPreviewTabSessions = 'sessions';

final class TerminalPreviewItem {
  const TerminalPreviewItem({
    required this.id,
    required this.label,
    required this.meta,
    this.exited = false,
  });

  final String id;
  final String label;
  final String meta;
  final bool exited;
}

final class TerminalPreviewGroup {
  const TerminalPreviewGroup({
    required this.endpointId,
    required this.label,
    required this.current,
    required this.items,
    this.activeId,
  });

  final String endpointId;
  final String label;
  final bool current;
  final List<TerminalPreviewItem> items;
  final String? activeId;
}

final class TerminalPreviewSelection {
  const TerminalPreviewSelection({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminalId,
  });

  final String endpointId;
  final String endpointLabel;
  final String terminalId;
}

typedef TerminalPreviewScreenLoader = Future<CanonicalLiveScreen?> Function(
  String endpointId,
  String terminalId,
);

/// 打开全屏毛玻璃预览。
Future<TerminalPreviewSelection?> showTerminalPreviewOverlay({
  required BuildContext context,
  required List<TerminalPreviewGroup> groups,
  required TerminalSettings settings,
  required TerminalPreviewScreenLoader loadScreen,
  String initialTab = terminalPreviewTabTerminals,
}) {
  return showGeneralDialog<TerminalPreviewSelection>(
    context: context,
    barrierDismissible: true,
    barrierLabel: MaterialLocalizations.of(context).modalBarrierDismissLabel,
    barrierColor: Colors.transparent,
    transitionDuration: const Duration(milliseconds: 260),
    pageBuilder: (dialogContext, _, _) => _TerminalPreviewOverlay(
      groups: groups,
      settings: settings,
      loadScreen: loadScreen,
      initialTab: initialTab,
    ),
    transitionBuilder: (context, animation, _, child) {
      final curved = CurvedAnimation(
        parent: animation,
        curve: Curves.easeOutCubic,
        reverseCurve: Curves.easeInCubic,
      );
      return FadeTransition(
        opacity: curved,
        child: ScaleTransition(
          scale: Tween<double>(begin: 0.97, end: 1).animate(curved),
          child: child,
        ),
      );
    },
  );
}

final class _TerminalPreviewOverlay extends StatefulWidget {
  const _TerminalPreviewOverlay({
    required this.groups,
    required this.settings,
    required this.loadScreen,
    required this.initialTab,
  });

  final List<TerminalPreviewGroup> groups;
  final TerminalSettings settings;
  final TerminalPreviewScreenLoader loadScreen;
  final String initialTab;

  @override
  State<_TerminalPreviewOverlay> createState() =>
      _TerminalPreviewOverlayState();
}

final class _TerminalPreviewOverlayState extends State<_TerminalPreviewOverlay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _entrance = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 460),
  )..forward();
  late String _tab = widget.initialTab;
  String? _pickedKey;
  bool _closing = false;

  @override
  void dispose() {
    _entrance.dispose();
    super.dispose();
  }

  void _pick(TerminalPreviewGroup group, TerminalPreviewItem item) {
    if (_closing) return;
    _closing = true;
    setState(() => _pickedKey = '${group.endpointId}\u0000${item.id}');
    Future<void>.delayed(const Duration(milliseconds: 150), () {
      if (!mounted) return;
      Navigator.of(context).pop(
        TerminalPreviewSelection(
          endpointId: group.endpointId,
          endpointLabel: group.label,
          terminalId: item.id,
        ),
      );
    });
  }

  void _close() {
    if (_closing) return;
    _closing = true;
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final totalTerminals = widget.groups.fold<int>(
      0,
      (sum, group) => sum + group.items.length,
    );
    return Material(
      color: Colors.transparent,
      child: Stack(
        fit: StackFit.expand,
        children: [
          // 毛玻璃：模糊 + 暗色调 + 顶部青色光晕
          Positioned.fill(
            child: GestureDetector(
              onTap: _close,
              child: ClipRect(
                child: BackdropFilter(
                  filter: ui.ImageFilter.blur(sigmaX: 26, sigmaY: 26),
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        begin: Alignment.topCenter,
                        end: Alignment.bottomCenter,
                        colors: [
                          palette.background.withValues(alpha: 0.62),
                          palette.background.withValues(alpha: 0.84),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
          SafeArea(
            child: Column(
              children: [
                _buildHeader(context, palette, totalTerminals),
                _buildTabs(context, palette),
                Expanded(
                  child: AnimatedSwitcher(
                    duration: const Duration(milliseconds: 180),
                    child: _tab == terminalPreviewTabTerminals
                        ? _buildTerminalsTab(context, palette)
                        : _buildSessionsTab(context, palette),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(18, 6, 18, 12),
                  child: Text(
                    _tab == terminalPreviewTabTerminals
                        ? '点按切换 · 长按固定 · 缩略图为实时画面'
                        : '点按进入会话 · 每张卡显示最近的终端画面',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: palette.faint, fontSize: 10.5),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildHeader(BuildContext context, AnyttyPalette palette, int total) {
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onVerticalDragEnd: (details) {
        if ((details.primaryVelocity ?? 0) > 240) _close();
      },
      child: Padding(
        padding: const EdgeInsets.fromLTRB(18, 14, 14, 4),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _tab == terminalPreviewTabTerminals ? '终端' : '会话',
                    style: TextStyle(
                      color: palette.text,
                      fontSize: 19,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.2,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    _tab == terminalPreviewTabTerminals
                        ? '$total 个终端 · ${widget.groups.length} 个会话'
                        : '${widget.groups.length} 个会话 · $total 个终端',
                    style: TextStyle(color: palette.muted, fontSize: 11.5),
                  ),
                ],
              ),
            ),
            _RoundIconButton(
              icon: LucideIcons.x,
              palette: palette,
              onTap: _close,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTabs(BuildContext context, AnyttyPalette palette) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 10, 18, 6),
      child: Container(
        padding: const EdgeInsets.all(4),
        decoration: BoxDecoration(
          color: palette.surface.withValues(alpha: 0.55),
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: palette.border.withValues(alpha: 0.6)),
        ),
        child: Stack(
          children: [
            AnimatedAlign(
              duration: const Duration(milliseconds: 260),
              curve: Curves.easeOutCubic,
              alignment: _tab == terminalPreviewTabTerminals
                  ? Alignment.centerLeft
                  : Alignment.centerRight,
              child: FractionallySizedBox(
                widthFactor: 0.5,
                child: Container(
                  height: 34,
                  decoration: BoxDecoration(
                    color: palette.accent.withValues(alpha: 0.16),
                    borderRadius: BorderRadius.circular(11),
                    border: Border.all(
                      color: palette.accent.withValues(alpha: 0.38),
                    ),
                  ),
                ),
              ),
            ),
            Row(
              children: [
                _TabButton(
                  label: '终端',
                  selected: _tab == terminalPreviewTabTerminals,
                  palette: palette,
                  onTap: () =>
                      setState(() => _tab = terminalPreviewTabTerminals),
                ),
                _TabButton(
                  label: '会话',
                  selected: _tab == terminalPreviewTabSessions,
                  palette: palette,
                  onTap: () =>
                      setState(() => _tab = terminalPreviewTabSessions),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildTerminalsTab(BuildContext context, AnyttyPalette palette) {
    return ListView(
      key: const ValueKey(terminalPreviewTabTerminals),
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
      children: [
        for (final group in widget.groups) ...[
          _GroupHeader(group: group, palette: palette),
          _PreviewGrid(
            group: group,
            palette: palette,
            settings: widget.settings,
            entrance: _entrance,
            screenFor: (id) =>
                () => widget.loadScreen(group.endpointId, id),
            pickedKey: _pickedKey,
            onPick: (item) => _pick(group, item),
          ),
          const SizedBox(height: 18),
        ],
      ],
    );
  }

  Widget _buildSessionsTab(BuildContext context, AnyttyPalette palette) {
    return ListView(
      key: const ValueKey(terminalPreviewTabSessions),
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
      children: [
        for (var index = 0; index < widget.groups.length; index += 1)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: _SessionCard(
              group: widget.groups[index],
              palette: palette,
              settings: widget.settings,
              entrance: _entrance,
              delayIndex: index,
              screenFor: (id) =>
                  () => widget.loadScreen(widget.groups[index].endpointId, id),
              onTap: () {
                final items = widget.groups[index].items;
                if (items.isEmpty) return;
                final active = widget.groups[index].activeId;
                final item = items.firstWhere(
                  (candidate) => candidate.id == active,
                  orElse: () => items.first,
                );
                _pick(widget.groups[index], item);
              },
            ),
          ),
      ],
    );
  }
}

final class _TabButton extends StatelessWidget {
  const _TabButton({
    required this.label,
    required this.selected,
    required this.palette,
    required this.onTap,
  });

  final String label;
  final bool selected;
  final AnyttyPalette palette;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: SizedBox(
          height: 34,
          child: Center(
            child: AnimatedDefaultTextStyle(
              duration: const Duration(milliseconds: 180),
              style: TextStyle(
                color: selected ? palette.text : palette.muted,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
              child: Text(label),
            ),
          ),
        ),
      ),
    );
  }
}

final class _RoundIconButton extends StatelessWidget {
  const _RoundIconButton({
    required this.icon,
    required this.palette,
    required this.onTap,
  });

  final IconData icon;
  final AnyttyPalette palette;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: palette.surface.withValues(alpha: 0.65),
          shape: BoxShape.circle,
          border: Border.all(color: palette.border.withValues(alpha: 0.7)),
        ),
        child: Icon(icon, size: 16, color: palette.text),
      ),
    );
  }
}

final class _GroupHeader extends StatelessWidget {
  const _GroupHeader({required this.group, required this.palette});

  final TerminalPreviewGroup group;
  final AnyttyPalette palette;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(4, 2, 4, 10),
      child: Row(
        children: [
          Container(
            width: 7,
            height: 7,
            decoration: BoxDecoration(
              color: group.current ? palette.success : palette.muted,
              shape: BoxShape.circle,
              boxShadow: group.current
                  ? [
                      BoxShadow(
                        color: palette.success.withValues(alpha: 0.7),
                        blurRadius: 8,
                      ),
                    ]
                  : null,
            ),
          ),
          const SizedBox(width: 8),
          Flexible(
            child: Text(
              group.label,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.muted,
                fontSize: 11.5,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          const Spacer(),
          Text(
            '${group.items.length} 个终端',
            style: TextStyle(
              color: palette.faint,
              fontSize: 10,
              fontFamily: 'JetBrainsMonoNerd',
            ),
          ),
        ],
      ),
    );
  }
}

final class _PreviewGrid extends StatelessWidget {
  const _PreviewGrid({
    required this.group,
    required this.palette,
    required this.settings,
    required this.entrance,
    required this.screenFor,
    required this.pickedKey,
    required this.onPick,
  });

  final TerminalPreviewGroup group;
  final AnyttyPalette palette;
  final TerminalSettings settings;
  final Animation<double> entrance;
  final Future<CanonicalLiveScreen?> Function() Function(String terminalId)
  screenFor;
  final String? pickedKey;
  final ValueChanged<TerminalPreviewItem> onPick;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      padding: EdgeInsets.zero,
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        mainAxisExtent: 152,
      ),
      itemCount: group.items.length,
      itemBuilder: (context, index) {
        final item = group.items[index];
        final key = '${group.endpointId}\u0000${item.id}';
        return _StaggeredEntrance(
          entrance: entrance,
          index: index,
          child: _PreviewCard(
            palette: palette,
            settings: settings,
            loadScreen: screenFor(item.id),
            item: item,
            active: item.id == group.activeId,
            picked: pickedKey == key,
            onTap: () => onPick(item),
          ),
        );
      },
    );
  }
}

final class _StaggeredEntrance extends StatelessWidget {
  const _StaggeredEntrance({
    required this.entrance,
    required this.index,
    required this.child,
  });

  final Animation<double> entrance;
  final int index;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final delay = (index * 0.07).clamp(0.0, 0.45);
    return AnimatedBuilder(
      animation: entrance,
      builder: (context, child) {
        final t = ((entrance.value - delay) / (1 - delay)).clamp(0.0, 1.0);
        final eased = Curves.easeOutCubic.transform(t);
        return Opacity(
          opacity: eased,
          child: Transform.translate(
            offset: Offset(0, (1 - eased) * 16),
            child: Transform.scale(scale: 0.96 + 0.04 * eased, child: child),
          ),
        );
      },
      child: child,
    );
  }
}

final class _PreviewCard extends StatelessWidget {
  const _PreviewCard({
    required this.palette,
    required this.settings,
    required this.loadScreen,
    required this.item,
    required this.active,
    required this.picked,
    required this.onTap,
  });

  final AnyttyPalette palette;
  final TerminalSettings settings;
  final Future<CanonicalLiveScreen?> Function() loadScreen;
  final TerminalPreviewItem item;
  final bool active;
  final bool picked;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return AnimatedScale(
      scale: picked ? 1.05 : 1,
      duration: const Duration(milliseconds: 140),
      curve: Curves.easeOutCubic,
      child: GestureDetector(
        onTap: onTap,
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 160),
          decoration: BoxDecoration(
            color: palette.surface.withValues(alpha: 0.78),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: picked
                  ? palette.accent.withValues(alpha: 0.7)
                  : active
                  ? palette.accent.withValues(alpha: 0.55)
                  : palette.border.withValues(alpha: 0.9),
            ),
            boxShadow: picked || active
                ? [
                    BoxShadow(
                      color: palette.accent.withValues(alpha: 0.25),
                      blurRadius: 22,
                    ),
                  ]
                : null,
          ),
          clipBehavior: Clip.antiAlias,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Expanded(
                child: Stack(
                  fit: StackFit.expand,
                  children: [
                    _LiveTerminalThumbnail(
                      loadScreen: loadScreen,
                      settings: settings,
                      palette: palette,
                    ),
                    Positioned(
                      top: 7,
                      right: 7,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 6,
                          vertical: 2,
                        ),
                        decoration: BoxDecoration(
                          color: palette.background.withValues(alpha: 0.8),
                          borderRadius: BorderRadius.circular(999),
                          border: Border.all(
                            color: palette.border.withValues(alpha: 0.8),
                          ),
                        ),
                        child: Text(
                          active
                              ? 'LIVE'
                              : item.exited
                              ? 'EXIT'
                              : '10fps',
                          style: TextStyle(
                            color: active ? palette.accent : palette.muted,
                            fontSize: 8.5,
                            fontFamily: 'JetBrainsMonoNerd',
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.fromLTRB(10, 7, 10, 8),
                decoration: BoxDecoration(
                  border: Border(
                    top: BorderSide(
                      color: palette.border.withValues(alpha: 0.55),
                    ),
                  ),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      item.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 3),
                    Text(
                      item.meta,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.faint,
                        fontSize: 9,
                        fontFamily: 'JetBrainsMonoNerd',
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _SessionCard extends StatelessWidget {
  const _SessionCard({
    required this.group,
    required this.palette,
    required this.settings,
    required this.entrance,
    required this.delayIndex,
    required this.screenFor,
    required this.onTap,
  });

  final TerminalPreviewGroup group;
  final AnyttyPalette palette;
  final TerminalSettings settings;
  final Animation<double> entrance;
  final int delayIndex;
  final Future<CanonicalLiveScreen?> Function() Function(String terminalId)
  screenFor;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final activeId = group.activeId;
    final item = group.items.isEmpty
        ? null
        : group.items.firstWhere(
            (candidate) => candidate.id == activeId,
            orElse: () => group.items.first,
          );
    return _StaggeredEntrance(
      entrance: entrance,
      index: delayIndex,
      child: GestureDetector(
        onTap: onTap,
        child: Container(
          height: 132,
          decoration: BoxDecoration(
            color: palette.surface.withValues(alpha: 0.78),
            borderRadius: BorderRadius.circular(18),
            border: Border.all(
              color: group.current
                  ? palette.accent.withValues(alpha: 0.55)
                  : palette.border.withValues(alpha: 0.9),
            ),
            boxShadow: group.current
                ? [
                    BoxShadow(
                      color: palette.accent.withValues(alpha: 0.18),
                      blurRadius: 24,
                    ),
                  ]
                : null,
          ),
          clipBehavior: Clip.antiAlias,
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              SizedBox(
                width: 140,
                child: item == null
                    ? _ThumbnailPlaceholder(palette: palette)
                    : _LiveTerminalThumbnail(
                        loadScreen: screenFor(item.id),
                        settings: settings,
                        palette: palette,
                      ),
              ),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(12, 12, 12, 12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Row(
                        children: [
                          Container(
                            width: 8,
                            height: 8,
                            decoration: BoxDecoration(
                              color: group.current
                                  ? palette.success
                                  : palette.muted,
                              shape: BoxShape.circle,
                              boxShadow: group.current
                                  ? [
                                      BoxShadow(
                                        color: palette.success.withValues(
                                          alpha: 0.7,
                                        ),
                                        blurRadius: 9,
                                      ),
                                    ]
                                  : null,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              group.label,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                color: palette.text,
                                fontSize: 14,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 6),
                      Text(
                        '${group.items.length} 个终端',
                        style: TextStyle(color: palette.muted, fontSize: 11),
                      ),
                      if (item != null) ...[
                        const SizedBox(height: 4),
                        Text(
                          item.label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: palette.faint,
                            fontSize: 9.5,
                            fontFamily: 'JetBrainsMonoNerd',
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _TerminalThumbnail extends StatelessWidget {
  const _TerminalThumbnail({required this.screen, required this.settings});

  final CanonicalLiveScreen screen;
  final TerminalSettings settings;

  @override
  Widget build(BuildContext context) {
    final metrics = settings.metrics;
    final size = Size(
      screen.cols * metrics.cellWidth,
      screen.rows * metrics.rowHeight,
    );
    if (size.width <= 0 || size.height <= 0) {
      return const SizedBox.shrink();
    }
    return ClipRect(
      child: FittedBox(
        fit: BoxFit.cover,
        alignment: Alignment.topLeft,
        child: SizedBox(
          width: size.width,
          height: size.height,
          child: IgnorePointer(
            child: TerminalCanvas(
              screen: screen,
              settings: settings,
              reserveLongPress: false,
            ),
          ),
        ),
      ),
    );
  }
}

final class _ThumbnailPlaceholder extends StatelessWidget {
  const _ThumbnailPlaceholder({required this.palette});

  final AnyttyPalette palette;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [
            palette.background.withValues(alpha: 0.9),
            palette.surface.withValues(alpha: 0.7),
          ],
        ),
      ),
      child: Center(
        child: SizedBox(
          width: 14,
          height: 14,
          child: CircularProgressIndicator(
            strokeWidth: 1.4,
            color: palette.faint.withValues(alpha: 0.6),
          ),
        ),
      ),
    );
  }
}

/// 每 1.2s 刷新一帧的缩略图：打开预览时保持"活"的感觉，但不会像主终端那样
/// 每帧都渲染。新帧到达前一直显示上一帧，避免闪烁。
final class _LiveTerminalThumbnail extends StatefulWidget {
  const _LiveTerminalThumbnail({
    required this.loadScreen,
    required this.settings,
    required this.palette,
  });

  final Future<CanonicalLiveScreen?> Function() loadScreen;
  final TerminalSettings settings;
  final AnyttyPalette palette;

  @override
  State<_LiveTerminalThumbnail> createState() => _LiveTerminalThumbnailState();
}

final class _LiveTerminalThumbnailState extends State<_LiveTerminalThumbnail> {
  CanonicalLiveScreen? _screen;
  Timer? _timer;
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    unawaited(_refresh());
    _timer = Timer.periodic(
      const Duration(milliseconds: 1200),
      (_) => unawaited(_refresh()),
    );
  }

  Future<void> _refresh() async {
    if (_loading) return;
    _loading = true;
    try {
      final next = await widget.loadScreen();
      if (mounted && next != null) {
        setState(() => _screen = next);
      }
    } catch (_) {
      // 预览失败保留上一帧。
    } finally {
      _loading = false;
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final screen = _screen;
    if (screen == null) {
      return _ThumbnailPlaceholder(palette: widget.palette);
    }
    return _TerminalThumbnail(screen: screen, settings: widget.settings);
  }
}
