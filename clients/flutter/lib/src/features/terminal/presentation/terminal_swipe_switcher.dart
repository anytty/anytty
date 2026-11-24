import 'dart:async';
import 'dart:ui' show lerpDouble;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../app/anytty_theme.dart';
import '../domain/live_screen_store.dart';
import '../domain/terminal_settings.dart';
import 'terminal_canvas.dart';

/// 手势切换的目标层：terminal（同机终端标签）或 device（session / 机器）。
enum TerminalSwipeMode { terminals, devices }

/// 拖拽结束后的决策。
enum TerminalSwipeDecision {
  /// 距离/速度都不足，弹回原位。
  cancel,

  /// 切到下一个（左滑）。
  next,

  /// 切到上一个（右滑）。
  previous,

  /// 在边界继续甩出，打开现有切换器面板。
  openSwitcher,
}

/// 一个可切换目标。terminal 模式带预取的画面；device 模式只有标签。
final class TerminalSwipeTarget {
  const TerminalSwipeTarget({
    required this.id,
    required this.label,
    this.subtitle,
    this.screen,
  });

  final String id;
  final String label;
  final String? subtitle;
  final CanonicalLiveScreen? screen;
}

/// 手势开始时异步预取的快照：目标顺序 + 当前位置。
final class TerminalSwipeSnapshot {
  const TerminalSwipeSnapshot({required this.targets, required this.currentIndex});

  final List<TerminalSwipeTarget> targets;
  final int currentIndex;

  bool get isEmpty => targets.isEmpty;
}

/// 工作区向手势层提供的数据与动作面。实现方持有 ref / context。
abstract interface class TerminalSwipeBridge {
  /// 拖拽开始时调用一次；返回排序后的目标和当前位置。
  Future<TerminalSwipeSnapshot> snapshot(TerminalSwipeMode mode);

  /// 同机内切换终端。
  void commitTerminal(String terminalId);

  /// 切换机器（session）。
  void commitDevice(String endpointId, String label);

  /// 到头继续甩出：打开既有切换器面板。
  void openSwitcher();
}

/// 纯手势判定逻辑，独立成类便于单测。
///
/// 触发规则（设计稿 §5）：
/// - 角度门控 |dx| >= |dy| * 2 且 |dx| > engageSlop 才接管；
/// - 松手时位移 >= commitDistance 或同向速度 >= flingVelocity 即切换；
/// - 边界外按 0.3 橡皮筋衰减；累计越界 >= openSheetOverdrag 松手开面板。
final class TerminalSwipeLogic {
  TerminalSwipeLogic({required this.sensitivity})
    : factor = sensitivity.thresholdFactor;

  final TerminalSwipeSensitivity sensitivity;
  final double factor;

  static const double engageSlop = 6.0;
  static const double angleRatio = 2.0;
  static const double commitDistance = 32.0;
  static const double flingVelocity = 350.0;
  static const double rubberBandFactor = 0.3;
  static const double openSheetOverdrag = 56.0;

  double get engageThreshold => engageSlop;
  double get commitThreshold => commitDistance * factor;
  double get flingThreshold => flingVelocity * factor;
  double get openSheetThreshold => openSheetOverdrag * factor;

  /// 横向是否已明显占优；竖向拖动（滚动历史）永远不接管。
  bool shouldEngage(double dx, double dy) {
    final absoluteX = dx.abs();
    final absoluteY = dy.abs();
    if (absoluteX < engageThreshold) return false;
    return absoluteX >= absoluteY * angleRatio;
  }

  /// 把手指位移映射为画面位移：界内跟手，界外橡皮筋。
  double visualOffset(double dx, {required bool canNext, required bool canPrevious}) {
    if (dx < 0) {
      if (canNext) return dx;
      return -openSheetThreshold * rubberBandFactor -
          ((-dx - openSheetThreshold).clamp(0, double.infinity) * rubberBandFactor);
    }
    if (canPrevious) return dx;
    return openSheetThreshold * rubberBandFactor +
        ((dx - openSheetThreshold).clamp(0, double.infinity) * rubberBandFactor);
  }

  /// 松手决策。dx < 0 代表向左滑（切下一个）。
  TerminalSwipeDecision decide(
    double dx,
    Velocity velocity, {
    required bool canNext,
    required bool canPrevious,
  }) {
    if (dx == 0) return TerminalSwipeDecision.cancel;
    final next = dx < 0;
    final allowed = next ? canNext : canPrevious;
    if (!allowed) {
      // 到头甩出面板：以界外的真实拖动距离衡量。
      final overdrag = (dx.abs() - openSheetThreshold).clamp(0, double.infinity);
      if (overdrag > 0) return TerminalSwipeDecision.openSwitcher;
      return TerminalSwipeDecision.cancel;
    }
    final distance = dx.abs();
    final aligned = next ? velocity.pixelsPerSecond.dx < 0 : velocity.pixelsPerSecond.dx > 0;
    final flung = aligned && velocity.pixelsPerSecond.dx.abs() >= flingThreshold;
    if (distance >= commitThreshold || flung) {
      return next ? TerminalSwipeDecision.next : TerminalSwipeDecision.previous;
    }
    return TerminalSwipeDecision.cancel;
  }
}

/// 手势切换层：单指横滑切终端、双指横滑切机器，拖拽实时预览。
///
/// 挂在花瓣菜单区域外层：长按（花瓣）与横滑天然不竞争，竖滑（历史
/// 滚动）由内层 vertical drag 认领，本层只认横向。
class TerminalSwipeSwitcher extends StatefulWidget {
  const TerminalSwipeSwitcher({
    super.key,
    required this.bridge,
    required this.enabled,
    required this.settings,
    required this.hapticsEnabled,
    required this.currentScreenTerminalId,
    required this.child,
  });

  final TerminalSwipeBridge bridge;
  final bool enabled;
  final TerminalSettings settings;
  final bool hapticsEnabled;
  final String currentScreenTerminalId;
  final Widget child;

  @override
  State<TerminalSwipeSwitcher> createState() => _TerminalSwipeSwitcherState();
}

final class _TerminalSwipeSwitcherState extends State<TerminalSwipeSwitcher>
    with SingleTickerProviderStateMixin {
  late final AnimationController _spring = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 200),
  );

  @override
  void initState() {
    super.initState();
    _spring.addListener(() {
      if (mounted) setState(() {});
    });
  }

  final Set<int> _downPointers = {};

  bool _engaged = false;
  bool _thresholdPulsed = false;
  bool _boundaryPulsed = false;
  TerminalSwipeMode? _mode;
  Future<TerminalSwipeSnapshot>? _snapshotFuture;
  TerminalSwipeSnapshot? _snapshot;

  double _dragDx = 0;
  double _dragDy = 0;
  double _visualOffset = 0;
  Offset _lastGlobal = Offset.zero;

  /// 切换后保留的预览画面：真实流的首帧到达前挡住 “Waiting for snapshot”。
  TerminalSwipeTarget? _holdTarget;
  bool _holdFading = false;

  void _resetDrag() {
    _engaged = false;
    _thresholdPulsed = false;
    _boundaryPulsed = false;
    _dragDx = 0;
    _dragDy = 0;
    _visualOffset = 0;
    _spring.stop();
  }

  TerminalSwipeLogic get _logic =>
      TerminalSwipeLogic(sensitivity: widget.settings.gestureSensitivity);

  void _onHorizontalDragStart(DragStartDetails details) {
    if (!widget.enabled) return;
    // 模式在角度门控通过后再决定：拖动开始可能发生在第二个手指落下之前。
    _resetDrag();
    _mode = null;
    _lastGlobal = details.globalPosition;
    _snapshot = null;
    _snapshotFuture = null;
  }

  /// 在角度门控通过后才预取快照：斜向滑动不应该产生任何请求。
  void _ensureSnapshot(TerminalSwipeMode mode) {
    if (_snapshotFuture != null) return;
    final future = widget.bridge.snapshot(mode);
    _snapshotFuture = future;
    unawaited(
      future
          .then((snapshot) {
            if (!mounted || _mode != mode) return;
            setState(() => _snapshot = snapshot);
          })
          .catchError((_) {}),
    );
  }

  void _onHorizontalDragUpdate(DragUpdateDetails details) {
    if (!widget.enabled) return;
    // 水平 recognizer 会把 delta.dy 归零，角度门控必须用真实位移。
    final raw = details.globalPosition - _lastGlobal;
    _lastGlobal = details.globalPosition;
    _dragDx += raw.dx;
    _dragDy += raw.dy;
    if (!_engaged) {
      if (!_logic.shouldEngage(_dragDx, _dragDy)) return;
      final pointers = _downPointers.length;
      final mode = pointers >= 2
          ? (widget.settings.gestureDeviceSwitch
                ? TerminalSwipeMode.devices
                : null)
          : (widget.settings.gestureTerminalSwitch
                ? TerminalSwipeMode.terminals
                : null);
      if (mode == null) return;
      _mode = mode;
      _engaged = true;
      _ensureSnapshot(mode);
      _pulse(light: true);
    }
    final snapshot = _snapshot;
    final currentIndex = snapshot?.currentIndex ?? 0;
    final targetCount = snapshot?.targets.length ?? 0;
    // 快照未就绪时乐观放行：跟手感优先，边界在松手时再判定。
    final canNext = snapshot == null || currentIndex + 1 < targetCount;
    final canPrevious = snapshot == null || currentIndex > 0;
    final offset = _logic.visualOffset(
      _dragDx,
      canNext: canNext,
      canPrevious: canPrevious,
    );
    setState(() => _visualOffset = offset);
    if (_dragDx.abs() >= _logic.commitThreshold && !_thresholdPulsed) {
      _thresholdPulsed = true;
      _pulse(medium: true);
    }
    final atBoundary = _dragDx < 0 ? !canNext : !canPrevious;
    if (atBoundary && !_boundaryPulsed) {
      _boundaryPulsed = true;
      _pulse(heavy: true);
    }
  }

  Future<void> _onHorizontalDragEnd(DragEndDetails details) async {
    if (!_engaged || _mode == null) {
      _resetDrag();
      if (mounted) setState(() {});
      return;
    }
    var snapshot = _snapshot;
    if (snapshot == null) {
      try {
        snapshot = await _snapshotFuture?.timeout(
          const Duration(milliseconds: 220),
        );
      } catch (_) {
        snapshot = null;
      }
      if (!mounted) return;
      if (snapshot != null) setState(() => _snapshot = snapshot);
    }
    final mode = _mode!;
    final currentIndex = snapshot?.currentIndex ?? 0;
    final targetCount = snapshot?.targets.length ?? 0;
    final decision = _logic.decide(
      _dragDx,
      details.velocity,
      canNext: currentIndex + 1 < targetCount,
      canPrevious: currentIndex > 0,
    );
    switch (decision) {
      case TerminalSwipeDecision.next:
      case TerminalSwipeDecision.previous:
        final step = decision == TerminalSwipeDecision.next ? 1 : -1;
        final index = (currentIndex + step).clamp(0, targetCount - 1);
        final target = snapshot == null || targetCount == 0
            ? null
            : snapshot.targets[index];
        _commit(mode, target);
      case TerminalSwipeDecision.openSwitcher:
        _animateBack();
        widget.bridge.openSwitcher();
      case TerminalSwipeDecision.cancel:
        _animateBack();
    }
  }

  void _onHorizontalDragCancel() {
    _resetDrag();
    if (mounted) setState(() {});
  }

  void _commit(TerminalSwipeMode mode, TerminalSwipeTarget? target) {
    if (target == null) {
      _animateBack();
      return;
    }
    switch (mode) {
      case TerminalSwipeMode.terminals:
        setState(() {
          _holdTarget = target;
          _holdFading = false;
        });
        _resetDrag();
        widget.bridge.commitTerminal(target.id);
      case TerminalSwipeMode.devices:
        _resetDrag();
        if (mounted) setState(() {});
        widget.bridge.commitDevice(target.id, target.label);
    }
  }

  void _animateBack() {
    _resetDrag();
    if (!mounted) return;
    _spring
      ..value = 1.0
      ..reverse();
    setState(() {});
  }

  void _pulse({bool light = false, bool medium = false, bool heavy = false}) {
    if (!widget.hapticsEnabled) return;
    if (light) HapticFeedback.selectionClick();
    if (medium) HapticFeedback.mediumImpact();
    if (heavy) HapticFeedback.heavyImpact();
  }

  @override
  void didUpdateWidget(TerminalSwipeSwitcher oldWidget) {
    super.didUpdateWidget(oldWidget);
    // 真实流首帧到达：预览保持层开始淡出。
    if (_holdTarget != null &&
        !_holdFading &&
        widget.currentScreenTerminalId == _holdTarget!.id) {
      _releaseHold();
    }
  }

  void _releaseHold() {
    if (!mounted) return;
    setState(() => _holdFading = true);
    Timer(const Duration(milliseconds: 260), () {
      if (mounted && _holdFading) {
        setState(() {
          _holdTarget = null;
          _holdFading = false;
        });
      }
    });
  }

  @override
  void dispose() {
    _spring.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final dragging = _engaged && _mode != null;
    final snapshot = _snapshot;
    final currentIndex = snapshot?.currentIndex ?? 0;
    final targetCount = snapshot?.targets.length ?? 0;
    final directionNext = _dragDx < 0;
    final previewIndex = directionNext ? currentIndex + 1 : currentIndex - 1;
    final preview = dragging && previewIndex >= 0 && previewIndex < targetCount
        ? snapshot!.targets[previewIndex]
        : null;

    final springOffset = lerpDouble(
      _visualOffset,
      0,
      Curves.easeOutCubic.transform(_spring.value),
    )!;

    return Listener(
      onPointerDown: (event) => _downPointers.add(event.pointer),
      onPointerUp: (event) => _downPointers.remove(event.pointer),
      onPointerCancel: (event) => _downPointers.remove(event.pointer),
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onHorizontalDragStart: widget.enabled ? _onHorizontalDragStart : null,
        onHorizontalDragUpdate: _onHorizontalDragUpdate,
        onHorizontalDragEnd: _onHorizontalDragEnd,
        onHorizontalDragCancel: _onHorizontalDragCancel,
        child: LayoutBuilder(
          builder: (context, constraints) {
            final width = constraints.maxWidth;
            return ClipRect(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  Transform.translate(
                    offset: Offset(springOffset, 0),
                    child: widget.child,
                  ),
                  if (preview != null)
                    Positioned.fill(
                      child: Transform.translate(
                        offset: Offset(
                          directionNext ? width + springOffset : -width + springOffset,
                          0,
                        ),
                        child: _buildPreview(context, preview, palette),
                      ),
                    ),
                  if (dragging && preview != null)
                    Positioned(
                      top: 10,
                      left: 0,
                      right: 0,
                      child: Center(
                        child: _buildChip(context, preview, palette),
                      ),
                    ),
                  if (_holdTarget != null)
                    Positioned.fill(
                      child: IgnorePointer(
                        child: AnimatedOpacity(
                          opacity: _holdFading ? 0 : 1,
                          duration: const Duration(milliseconds: 220),
                          child: _buildPreview(
                            context,
                            _holdTarget!,
                            palette,
                            opaque: true,
                          ),
                        ),
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      ),
    );
  }

  Widget _buildPreview(
    BuildContext context,
    TerminalSwipeTarget target,
    AnyttyPalette palette, {
    bool opaque = false,
  }) {
    final screen = target.screen;
    final background = Color(0xff000000 | widget.settings.theme.background);
    if (screen != null) {
      return ColoredBox(
        color: background,
        child: TerminalCanvas(
          screen: screen,
          settings: widget.settings,
          reserveLongPress: true,
        ),
      );
    }
    // 设备预览：没有本地画面，用卡片表示。
    return ColoredBox(
      color: opaque ? background : palette.surface,
      child: Center(
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
          decoration: BoxDecoration(
            color: palette.surface,
            borderRadius: BorderRadius.circular(14),
            border: Border.all(color: palette.border),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.computer_rounded, size: 20, color: palette.accent),
              const SizedBox(width: 10),
              Flexible(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      target.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (target.subtitle case final subtitle?)
                      Text(
                        subtitle,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.muted, fontSize: 11),
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

  Widget _buildChip(
    BuildContext context,
    TerminalSwipeTarget target,
    AnyttyPalette palette,
  ) {
    final snapshot = _snapshot;
    final count = snapshot?.targets.length ?? 0;
    final index = (snapshot?.currentIndex ?? 0) + (_dragDx < 0 ? 1 : -1) + 1;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: palette.surface.withValues(alpha: 0.92),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: palette.border),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$index/$count',
            style: TextStyle(
              color: palette.muted,
              fontSize: 11,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(width: 8),
          Text(
            target.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: palette.text, fontSize: 12),
          ),
        ],
      ),
    );
  }
}
