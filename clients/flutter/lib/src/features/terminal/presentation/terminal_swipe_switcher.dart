import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/physics.dart';
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

  TerminalSwipeTarget withScreen(CanonicalLiveScreen? next) =>
      TerminalSwipeTarget(
        id: id,
        label: label,
        subtitle: subtitle,
        screen: next ?? screen,
      );
}

/// 手势开始时异步预取的快照：目标顺序 + 当前位置。
final class TerminalSwipeSnapshot {
  const TerminalSwipeSnapshot({
    required this.targets,
    required this.currentIndex,
  });

  final List<TerminalSwipeTarget> targets;
  final int currentIndex;
}

/// 工作区向手势层提供的数据与动作面。实现方持有 ref / context。
abstract interface class TerminalSwipeBridge {
  /// 拖拽开始时调用一次；返回排序后的目标和当前位置。
  Future<TerminalSwipeSnapshot> snapshot(TerminalSwipeMode mode);

  /// 提交前预热目标终端：attach + resize 到目标网格，并返回 resize 后的一帧。
  ///
  /// 返回 null 表示预热失败或超时，调用方应直接提交（可能有一帧延迟）。
  Future<CanonicalLiveScreen?> prepareTerminal(
    String terminalId, {
    required int cols,
    required int rows,
  });

  /// 把预览帧交给上层持有。上层渲染在 pane 之上，跨连接重建存活。
  void holdPreview(CanonicalLiveScreen screen, String terminalId);

  /// 新终端首帧已出现，释放上层垫帧。
  void releasePreview(String terminalId);

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
/// - 边界外按 0.3 橡皮筋衰减；越界拖动松手开面板。
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

  /// 跟手区结束后，超出的位移按这个比例继续跟手（越小越“粘”）。
  static const double stickyRatio = 0.55;

  /// 前一段位移完全跟手，占容器宽度的比例。
  static const double stickyKnee = 0.3;

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

  /// 粘性阻尼：前 [stickyKnee] 宽度 1:1 跟手，之后逐渐变粘。
  double sticky(double dx, double width) {
    if (width <= 0) return dx;
    final distance = dx.abs();
    final knee = width * stickyKnee;
    if (distance <= knee) return dx;
    return dx.sign * (knee + (distance - knee) * stickyRatio);
  }

  /// 把手指位移映射为画面位移：界内跟手，界外橡皮筋。
  double visualOffset(
    double dx, {
    required bool canNext,
    required bool canPrevious,
  }) {
    if (dx < 0) {
      if (canNext) return dx;
      return -openSheetThreshold * rubberBandFactor -
          ((-dx - openSheetThreshold).clamp(0, double.infinity) *
              rubberBandFactor);
    }
    if (canPrevious) return dx;
    return openSheetThreshold * rubberBandFactor +
        ((dx - openSheetThreshold).clamp(0, double.infinity) *
            rubberBandFactor);
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
      final overdrag = (dx.abs() - openSheetThreshold).clamp(
        0,
        double.infinity,
      );
      if (overdrag > 0) return TerminalSwipeDecision.openSwitcher;
      return TerminalSwipeDecision.cancel;
    }
    final distance = dx.abs();
    final aligned = next
        ? velocity.pixelsPerSecond.dx < 0
        : velocity.pixelsPerSecond.dx > 0;
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
    this.allowTerminalSwitch = true,
    this.allowDeviceSwitch = true,
    this.targetCols = 0,
    this.targetRows = 0,
    required this.child,
  });

  final TerminalSwipeBridge bridge;
  final bool enabled;
  final TerminalSettings settings;
  final bool hapticsEnabled;
  final String currentScreenTerminalId;

  /// 是否允许单指切终端（终端列表页只允许切机器）。
  final bool allowTerminalSwitch;
  final bool allowDeviceSwitch;

  /// 当前 pane 的网格尺寸，切换前把目标终端 resize 到这个尺寸。
  final int targetCols;
  final int targetRows;

  final Widget child;

  @override
  State<TerminalSwipeSwitcher> createState() => _TerminalSwipeSwitcherState();
}

final class _TerminalSwipeSwitcherState extends State<TerminalSwipeSwitcher>
    with SingleTickerProviderStateMixin {
  /// 画面横向位移（像素），用 spring 驱动。
  late final AnimationController _offset = AnimationController.unbounded(
    vsync: this,
  )..addListener(() => setState(() {}));

  final Set<int> _downPointers = {};

  bool _engaged = false;
  bool _committing = false;
  bool _thresholdPulsed = false;
  bool _boundaryPulsed = false;
  TerminalSwipeMode? _mode;
  Future<TerminalSwipeSnapshot>? _snapshotFuture;
  TerminalSwipeSnapshot? _snapshot;

  double _dragDx = 0;
  double _dragDy = 0;
  Offset _lastGlobal = Offset.zero;
  Offset _startLocal = Offset.zero;
  bool _ignored = false;
  double _width = 0;
  double _height = 0;
  int _railIndex = 0;
  int _railCount = 0;

  TerminalSwipeLogic get _logic =>
      TerminalSwipeLogic(sensitivity: widget.settings.gestureSensitivity);

  void _resetDrag() {
    _engaged = false;
    _thresholdPulsed = false;
    _boundaryPulsed = false;
    _dragDx = 0;
    _dragDy = 0;
    _offset.stop();
    _offset.value = 0;
  }

  void _onHorizontalDragStart(DragStartDetails details) {
    if (!widget.enabled || _committing) return;
    // 模式在角度门控通过后再决定：拖动开始可能发生在第二个手指落下之前。
    _resetDrag();
    _mode = null;
    _ignored = false;
    _lastGlobal = details.globalPosition;
    final box = context.findRenderObject();
    _startLocal = box is RenderBox
        ? box.globalToLocal(details.globalPosition)
        : details.globalPosition;
    _snapshot = null;
    _snapshotFuture = null;
  }

  /// 单指切换只允许从右下角的 rail 或右缘起手：滚动优先，斜滑不再误触。
  /// 双指（devices）不受限制，它本来就不会和滚动冲突。
  bool _startsOnRail() {
    if (_width <= 0 || _height <= 0) return true;
    final rail = Rect.fromLTWH(_width - 70, _height - 92, 70, 92);
    if (rail.contains(_startLocal)) return true;
    return _startLocal.dx >= _width - 28;
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
    if (!widget.enabled || _committing) return;
    // 水平 recognizer 会把 delta.dy 归零，角度门控必须用真实位移。
    final raw = details.globalPosition - _lastGlobal;
    _lastGlobal = details.globalPosition;
    _dragDx += raw.dx;
    _dragDy += raw.dy;
    if (!_engaged) {
      if (_ignored) return;
      if (!_logic.shouldEngage(_dragDx, _dragDy)) return;
      final pointers = _downPointers.length;
      if (pointers < 2 && !_startsOnRail()) {
        _ignored = true;
        return;
      }
      final mode = pointers >= 2
          ? (widget.allowDeviceSwitch && widget.settings.gestureDeviceSwitch
                ? TerminalSwipeMode.devices
                : null)
          : (widget.allowTerminalSwitch && widget.settings.gestureTerminalSwitch
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
    final sticky = _logic.sticky(_dragDx, _width);
    _offset.value = _logic.visualOffset(
      sticky,
      canNext: canNext,
      canPrevious: canPrevious,
    );
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
    if (_ignored) {
      _ignored = false;
      _resetDrag();
      if (mounted) setState(() {});
      return;
    }
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
        await _commit(mode, target, details.velocity);
      case TerminalSwipeDecision.openSwitcher:
        await _springBack(details.velocity);
        widget.bridge.openSwitcher();
      case TerminalSwipeDecision.cancel:
        await _springBack(details.velocity);
    }
  }

  void _onHorizontalDragCancel() {
    if (_committing) return;
    _ignored = false;
    unawaited(_springBack(Velocity.zero));
  }

  /// 提交：先预热（attach + resize + 首帧），再滑出，最后切换。
  Future<void> _commit(
    TerminalSwipeMode mode,
    TerminalSwipeTarget? target,
    Velocity velocity,
  ) async {
    if (target == null) {
      await _springBack(velocity);
      return;
    }
    _committing = true;
    final width = _width;
    final draggingLeft = _dragDx < 0;
    var prepared = target;
    if (mode == TerminalSwipeMode.terminals && widget.targetCols > 0) {
      final screen = await widget.bridge.prepareTerminal(
        target.id,
        cols: widget.targetCols,
        rows: widget.targetRows,
      );
      if (!mounted) return;
      if (screen != null) prepared = target.withScreen(screen);
    }
    if (!mounted) return;
    // 用预热后的帧替换预览，滑动过程中看到的是真画面。
    setState(() {
      _snapshot = TerminalSwipeSnapshot(
        targets: [
          for (var i = 0; i < (_snapshot?.targets.length ?? 0); i++)
            _snapshot!.targets[i].id == prepared.id
                ? prepared
                : _snapshot!.targets[i],
        ],
        currentIndex: _snapshot?.currentIndex ?? 0,
      );
    });
    final destination = draggingLeft ? -width : width;
    await _offset.animateTo(
      destination,
      duration: const Duration(milliseconds: 170),
      curve: Curves.easeOutCubic,
    );
    if (!mounted) return;
    // 滑到终点、画面已完全由预览帧铺满，再交棒给上层垫帧并切换：
    // 这样切换瞬间的连接重建被同一帧盖住，不会闪黑。
    if (prepared.screen != null && mode == TerminalSwipeMode.terminals) {
      widget.bridge.holdPreview(prepared.screen!, prepared.id);
    }
    _resetDrag();
    setState(() {});
    switch (mode) {
      case TerminalSwipeMode.terminals:
        widget.bridge.commitTerminal(prepared.id);
      case TerminalSwipeMode.devices:
        widget.bridge.commitDevice(prepared.id, prepared.label);
    }
    _committing = false;
  }

  Future<void> _springBack(Velocity velocity) async {
    if (!mounted) return;
    final from = _offset.value;
    if (from.abs() < 0.5) {
      _resetDrag();
      setState(() {});
      return;
    }
    final simulation = SpringSimulation(
      const SpringDescription(mass: 1, stiffness: 320, damping: 22),
      from,
      0,
      velocity.pixelsPerSecond.dx,
    );
    await _offset.animateWith(simulation);
    if (!mounted) return;
    _resetDrag();
    setState(() {});
  }

  void _pulse({bool light = false, bool medium = false, bool heavy = false}) {
    if (!widget.hapticsEnabled) return;
    if (light) HapticFeedback.selectionClick();
    if (medium) HapticFeedback.mediumImpact();
    if (heavy) HapticFeedback.heavyImpact();
  }

  @override
  void dispose() {
    _offset.dispose();
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
    final offset = _offset.value;

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
            _width = width;
            _height = constraints.maxHeight;
            _railCount = targetCount;
            _railIndex = currentIndex;
            return ClipRect(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  Transform.translate(
                    offset: Offset(offset, 0),
                    child: widget.child,
                  ),
                  if (preview != null)
                    Positioned.fill(
                      child: Transform.translate(
                        offset: Offset(
                          directionNext ? width + offset : -width + offset,
                          0,
                        ),
                        child: _buildPreview(context, preview, palette),
                      ),
                    ),
                  if (widget.enabled)
                    Positioned(
                      right: 10,
                      bottom: 14,
                      child: IgnorePointer(
                        child: _SwipeRail(
                          palette: palette,
                          engaged: _engaged,
                          index: _railIndex,
                          count: _railCount,
                        ),
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
    AnyttyPalette palette,
  ) {
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
    return ColoredBox(
      color: background,
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
    final progress = _width > 0
        ? (_offset.value.abs() / (_width * 0.35)).clamp(0.0, 1.0)
        : 0.0;
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
          SizedBox(
            width: 15,
            height: 15,
            child: CircularProgressIndicator(
              value: progress,
              strokeWidth: 2.2,
              backgroundColor: palette.border,
              valueColor: AlwaysStoppedAnimation<Color>(palette.accent),
            ),
          ),
          const SizedBox(width: 8),
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


/// 右下角的切换轨道：单指横滑只从这里（或右缘）起手，避免和滚动抢手势。
final class _SwipeRail extends StatelessWidget {
  const _SwipeRail({
    required this.palette,
    required this.engaged,
    required this.index,
    required this.count,
  });

  final AnyttyPalette palette;
  final bool engaged;
  final int index;
  final int count;

  @override
  Widget build(BuildContext context) {
    final dots = count > 0 && count <= 5 ? count : 3;
    final active = count > 0 ? index.clamp(0, dots - 1) : 0;
    return AnimatedContainer(
      duration: const Duration(milliseconds: 180),
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 8),
      decoration: BoxDecoration(
        color: palette.surface.withValues(alpha: 0.72),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(
          color: engaged
              ? palette.accent.withValues(alpha: 0.6)
              : palette.border,
        ),
        boxShadow: const [
          BoxShadow(color: Color(0x66000000), blurRadius: 18, offset: Offset(0, 8)),
        ],
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 18,
            height: 18,
            alignment: Alignment.center,
            child: Container(
              width: 9,
              height: 9,
              decoration: BoxDecoration(
                color: palette.accent,
                shape: BoxShape.circle,
                boxShadow: [
                  BoxShadow(
                    color: palette.accent.withValues(alpha: 0.75),
                    blurRadius: 9,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 5),
          for (var i = 0; i < dots; i += 1) ...[
            Container(
              width: i == active ? 4 : 3,
              height: i == active ? 4 : 3,
              margin: const EdgeInsets.symmetric(vertical: 1.5),
              decoration: BoxDecoration(
                color: i == active
                    ? palette.accent
                    : palette.muted.withValues(alpha: 0.35),
                shape: BoxShape.circle,
              ),
            ),
          ],
          if (count > 0) ...[
            const SizedBox(height: 4),
            Text(
              '${index + 1}/$count',
              style: TextStyle(
                color: palette.faint,
                fontSize: 8,
                fontFamily: 'JetBrainsMonoNerd',
              ),
            ),
          ],
        ],
      ),
    );
  }
}
