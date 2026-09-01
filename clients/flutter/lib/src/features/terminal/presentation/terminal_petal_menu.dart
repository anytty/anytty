import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';

const terminalPetalMenuKey = ValueKey('terminal-petal-menu');
const terminalPetalNodeExtent = 56.0;

@immutable
final class TerminalPetalMenuItem {
  const TerminalPetalMenuItem({
    required this.id,
    required this.label,
    required this.icon,
    this.enabled = true,
    this.children = const [],
  });

  final String id;
  final String label;
  final IconData icon;
  final bool enabled;
  final List<TerminalPetalMenuItem> children;

  bool get hasChildren => children.isNotEmpty;
}

@immutable
final class TerminalPetalMenuSelection {
  TerminalPetalMenuSelection({required List<int> path})
    : path = List.unmodifiable(path);

  final List<int> path;

  String get key => path.join(':');
}

@immutable
final class _TerminalPetalMenuLayer {
  const _TerminalPetalMenuLayer({
    required this.parentPath,
    required this.actions,
    required this.positions,
    required this.start,
  });

  final List<int> parentPath;
  final List<TerminalPetalMenuItem> actions;
  final List<Offset> positions;
  final Offset start;
}

@immutable
final class TerminalPetalMenuSession {
  const TerminalPetalMenuSession._({
    required this.viewport,
    required this.origin,
    required this.center,
    required this.pointer,
    required this.actions,
    required this.rootRadius,
    required this.expandedPath,
    required this.selection,
  });

  factory TerminalPetalMenuSession.start({
    required Size viewport,
    required Offset origin,
    required List<TerminalPetalMenuItem> actions,
  }) {
    final shortestSide = math.min(viewport.width, viewport.height);
    final preferredRadius = (shortestSide * 0.24).clamp(50.0, 78.0);
    final maximumRadius = math.max(
      50.0,
      shortestSide / 2 - terminalPetalNodeExtent / 2 - 8,
    );
    final rootRadius = math
        .max(preferredRadius, _radiusForPetalCount(actions.length))
        .clamp(50.0, maximumRadius)
        .toDouble();
    final menuInset = rootRadius + terminalPetalNodeExtent / 2 + 8;
    final center = Offset(
      _clampAxis(origin.dx, viewport.width, menuInset),
      _clampAxis(origin.dy, viewport.height, menuInset),
    );
    return TerminalPetalMenuSession._(
      viewport: viewport,
      origin: origin,
      center: center,
      pointer: origin,
      actions: List.unmodifiable(actions),
      rootRadius: rootRadius,
      expandedPath: null,
      selection: null,
    );
  }

  final Size viewport;
  final Offset origin;
  final Offset center;
  final Offset pointer;
  final List<TerminalPetalMenuItem> actions;
  final double rootRadius;
  final List<int>? expandedPath;
  final TerminalPetalMenuSelection? selection;

  List<Offset> get rootPositions => List.generate(
    actions.length,
    (index) => center + _polar(rootRadius, _rootAngle(index)),
    growable: false,
  );

  List<Offset> childPositions(int rootIndex) =>
      positionsForParent(<int>[rootIndex]);

  List<Offset> positionsForParent(List<int> parentPath) {
    if (parentPath.isEmpty) return rootPositions;
    final children = actionAtPath(parentPath).children;
    if (children.isEmpty) return const [];
    final parentAngle = _angleForPath(parentPath);
    final depth = parentPath.length;
    final maximumFan = _maximumFan(depth);
    final outerRadius = math.max(
      rootRadius + 62 * depth,
      _radiusForFan(children.length, maximumFan),
    );
    final spread = _levelSpread(children.length, depth, outerRadius);
    final middle = (children.length - 1) / 2;
    return List.generate(children.length, (index) {
      final angle = parentAngle + (index - middle) * spread;
      final desired = center + _polar(outerRadius, angle);
      const inset = terminalPetalNodeExtent / 2 + 6;
      return Offset(
        _clampAxis(desired.dx, viewport.width, inset),
        _clampAxis(desired.dy, viewport.height, inset),
      );
    }, growable: false);
  }

  TerminalPetalMenuItem? get selectedAction {
    final selected = selection;
    if (selected == null) return null;
    return actionAtPath(selected.path);
  }

  bool get pointerTouchesSelection {
    final selected = selection;
    if (selected == null) return false;
    final point = positionForPath(selected.path);
    return (pointer - point).distance <= 48;
  }

  List<_TerminalPetalMenuLayer> get _visibleLayers {
    final layers = <_TerminalPetalMenuLayer>[
      _TerminalPetalMenuLayer(
        parentPath: const [],
        actions: actions,
        positions: rootPositions,
        start: center,
      ),
    ];
    final expanded = expandedPath;
    if (expanded == null) return layers;
    for (var depth = 1; depth <= expanded.length; depth += 1) {
      final parentPath = expanded.sublist(0, depth);
      final parent = actionAtPath(parentPath);
      if (!parent.hasChildren) break;
      layers.add(
        _TerminalPetalMenuLayer(
          parentPath: List.unmodifiable(parentPath),
          actions: parent.children,
          positions: positionsForParent(parentPath),
          start: positionForPath(parentPath),
        ),
      );
    }
    return layers;
  }

  TerminalPetalMenuItem actionAtPath(List<int> path) {
    var action = actions[path.first];
    for (var depth = 1; depth < path.length; depth += 1) {
      action = action.children[path[depth]];
    }
    return action;
  }

  Offset positionForPath(List<int> path) {
    if (path.length == 1) return rootPositions[path.first];
    return positionsForParent(path.sublist(0, path.length - 1))[path.last];
  }

  TerminalPetalMenuSession move(Offset nextPointer) {
    if ((nextPointer - origin).distance < 20) {
      return _copyWith(
        pointer: nextPointer,
        clearExpandedPath: true,
        clearSelection: true,
      );
    }

    for (final layer in _visibleLayers.reversed) {
      final hit = _nearestEnabled(
        points: layer.positions,
        enabled: layer.actions
            .map((action) => action.enabled)
            .toList(growable: false),
        pointer: nextPointer,
        maximumDistance: 48,
      );
      if (hit == null) continue;
      final path = <int>[...layer.parentPath, hit];
      return _selectPath(path, nextPointer);
    }

    final expanded = expandedPath;
    if (expanded != null) {
      return _copyWith(
        pointer: nextPointer,
        expandedPath: expanded,
        selection: TerminalPetalMenuSelection(path: expanded),
      );
    }
    return _copyWith(pointer: nextPointer, clearSelection: true);
  }

  TerminalPetalMenuSession _selectPath(List<int> path, Offset nextPointer) {
    final action = actionAtPath(path);
    final canExpand = action.hasChildren && path.length <= 2;
    final parentPath = path.length > 1
        ? path.sublist(0, path.length - 1)
        : null;
    return _copyWith(
      pointer: nextPointer,
      expandedPath: canExpand ? path : parentPath,
      clearExpandedPath: !canExpand && parentPath == null,
      selection: TerminalPetalMenuSelection(path: path),
    );
  }

  TerminalPetalMenuSession _copyWith({
    Offset? pointer,
    List<int>? expandedPath,
    bool clearExpandedPath = false,
    TerminalPetalMenuSelection? selection,
    bool clearSelection = false,
  }) => TerminalPetalMenuSession._(
    viewport: viewport,
    origin: origin,
    center: center,
    pointer: pointer ?? this.pointer,
    actions: actions,
    rootRadius: rootRadius,
    expandedPath: clearExpandedPath
        ? null
        : expandedPath == null
        ? this.expandedPath
        : List.unmodifiable(expandedPath),
    selection: clearSelection ? null : selection ?? this.selection,
  );

  double _rootAngle(int index) {
    if (actions.isEmpty) return -math.pi / 2;
    return -math.pi / 2 + index * math.pi * 2 / actions.length;
  }

  double _angleForPath(List<int> path) {
    var angle = _rootAngle(path.first);
    var action = actions[path.first];
    for (var depth = 1; depth < path.length; depth += 1) {
      final siblings = action.children;
      final middle = (siblings.length - 1) / 2;
      final maximumFan = _maximumFan(depth);
      final radius = math.max(
        rootRadius + 62 * depth,
        _radiusForFan(siblings.length, maximumFan),
      );
      angle +=
          (path[depth] - middle) * _levelSpread(siblings.length, depth, radius);
      action = siblings[path[depth]];
    }
    return angle;
  }

  double _levelSpread(int count, int depth, double radius) {
    if (count <= 1) return 0;
    final chord = terminalPetalNodeExtent + 8;
    final preferredGap = 2 * math.asin(math.min(1, chord / (2 * radius)));
    return math.min(preferredGap, _maximumFan(depth) / (count - 1));
  }
}

double _radiusForPetalCount(int count) {
  if (count <= 1) return 50;
  final chord = terminalPetalNodeExtent + 8;
  return chord / (2 * math.sin(math.pi / count));
}

double _maximumFan(int depth) => depth == 1 ? 4.8 : 4.2;

double _radiusForFan(int count, double maximumFan) {
  if (count <= 1) return 0;
  final halfAngle = maximumFan / (2 * (count - 1));
  final chord = terminalPetalNodeExtent + 8;
  return chord / (2 * math.sin(halfAngle));
}

final class TerminalPetalMenuOverlay extends StatefulWidget {
  const TerminalPetalMenuOverlay({super.key, required this.child});

  final Widget child;

  @override
  State<TerminalPetalMenuOverlay> createState() =>
      _TerminalPetalMenuOverlayState();
}

final class _TerminalPetalMenuOverlayState
    extends State<TerminalPetalMenuOverlay> {
  final OverlayPortalController _controller = OverlayPortalController(
    debugLabel: 'terminal-petal-menu',
  );
  final GlobalKey _viewportKey = GlobalKey(
    debugLabel: 'terminal-petal-viewport',
  );
  TerminalPetalMenuSession? _session;
  Object? _owner;
  ValueChanged<TerminalPetalMenuItem>? _onSelected;
  bool _hapticsEnabled = true;

  @override
  Widget build(BuildContext context) {
    return OverlayPortal.overlayChildLayoutBuilder(
      controller: _controller,
      overlayChildBuilder: (context, info) {
        final session = _session;
        if (session == null) return const SizedBox.shrink();
        return Positioned.fill(
          child: IgnorePointer(
            child: Stack(
              clipBehavior: Clip.none,
              children: [
                Transform(
                  transform: info.childPaintTransform,
                  transformHitTests: false,
                  child: SizedBox.fromSize(
                    size: info.childSize,
                    child: TerminalPetalMenu(session: session),
                  ),
                ),
              ],
            ),
          ),
        );
      },
      child: SizedBox(
        key: _viewportKey,
        width: double.infinity,
        height: double.infinity,
        child: _TerminalPetalMenuOverlayScope(host: this, child: widget.child),
      ),
    );
  }

  void open({
    required Object owner,
    required Offset globalOrigin,
    required List<TerminalPetalMenuItem> actions,
    required ValueChanged<TerminalPetalMenuItem> onSelected,
    required bool hapticsEnabled,
  }) {
    final viewport = _viewportBox;
    if (viewport == null || !viewport.hasSize) return;
    setState(() {
      _owner = owner;
      _onSelected = onSelected;
      _hapticsEnabled = hapticsEnabled;
      _session = TerminalPetalMenuSession.start(
        viewport: viewport.size,
        origin: viewport.globalToLocal(globalOrigin),
        actions: actions,
      );
    });
    _controller.show();
    if (hapticsEnabled) unawaited(HapticFeedback.mediumImpact());
  }

  void move({required Object owner, required Offset globalPointer}) {
    final current = _session;
    final viewport = _viewportBox;
    if (current == null || _owner != owner || viewport == null) return;
    final next = current.move(viewport.globalToLocal(globalPointer));
    final selectionChanged = next.selection?.key != current.selection?.key;
    setState(() => _session = next);
    if (_hapticsEnabled && selectionChanged && next.pointerTouchesSelection) {
      unawaited(HapticFeedback.lightImpact());
    }
  }

  void commit(Object owner) {
    if (_owner != owner) return;
    final selected = _session?.selectedAction;
    final onSelected = _onSelected;
    final hapticsEnabled = _hapticsEnabled;
    _close();
    if (selected == null || onSelected == null) return;
    if (hapticsEnabled) unawaited(HapticFeedback.mediumImpact());
    onSelected(selected);
  }

  void cancel(Object owner) {
    if (_owner == owner) _close();
  }

  RenderBox? get _viewportBox =>
      _viewportKey.currentContext?.findRenderObject() as RenderBox?;

  void _close() {
    if (_session == null) return;
    setState(() {
      _session = null;
      _owner = null;
      _onSelected = null;
    });
    _controller.hide();
  }
}

final class _TerminalPetalMenuOverlayScope extends InheritedWidget {
  const _TerminalPetalMenuOverlayScope({
    required this.host,
    required super.child,
  });

  final _TerminalPetalMenuOverlayState host;

  static _TerminalPetalMenuOverlayState? maybeOf(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<_TerminalPetalMenuOverlayScope>()
          ?.host;

  @override
  bool updateShouldNotify(_TerminalPetalMenuOverlayScope oldWidget) => false;
}

final class TerminalPetalMenuRegion extends StatefulWidget {
  const TerminalPetalMenuRegion({
    super.key,
    required this.actions,
    required this.onSelected,
    required this.child,
    this.enabled = true,
    this.hapticsEnabled = true,
    this.onOpened,
  });

  final List<TerminalPetalMenuItem> actions;
  final ValueChanged<TerminalPetalMenuItem> onSelected;
  final Widget child;
  final bool enabled;
  final bool hapticsEnabled;
  final VoidCallback? onOpened;

  @override
  State<TerminalPetalMenuRegion> createState() =>
      _TerminalPetalMenuRegionState();
}

final class _TerminalPetalMenuRegionState
    extends State<TerminalPetalMenuRegion> {
  TerminalPetalMenuSession? _session;
  _TerminalPetalMenuOverlayState? _overlayHost;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final nextHost = _TerminalPetalMenuOverlayScope.maybeOf(context);
    if (_overlayHost != nextHost) _overlayHost?.cancel(this);
    _overlayHost = nextHost;
  }

  @override
  void didUpdateWidget(TerminalPetalMenuRegion oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.enabled && !widget.enabled) {
      _session = null;
      _overlayHost?.cancel(this);
    }
  }

  @override
  void dispose() {
    _overlayHost?.cancel(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) => GestureDetector(
        behavior: HitTestBehavior.translucent,
        excludeFromSemantics: true,
        onLongPressStart: widget.enabled
            ? (details) => _open(details, constraints.biggest)
            : null,
        onLongPressMoveUpdate: widget.enabled ? _move : null,
        onLongPressEnd: widget.enabled ? (_) => _commit() : null,
        onLongPressCancel: widget.enabled ? _cancel : null,
        child: Stack(
          fit: StackFit.expand,
          children: [
            widget.child,
            if (_session case final session?)
              Positioned.fill(child: TerminalPetalMenu(session: session)),
          ],
        ),
      ),
    );
  }

  void _open(LongPressStartDetails details, Size viewport) {
    widget.onOpened?.call();
    final host = _overlayHost;
    if (host != null) {
      host.open(
        owner: this,
        globalOrigin: details.globalPosition,
        actions: widget.actions,
        onSelected: widget.onSelected,
        hapticsEnabled: widget.hapticsEnabled,
      );
      return;
    }
    setState(() {
      _session = TerminalPetalMenuSession.start(
        viewport: viewport,
        origin: details.localPosition,
        actions: widget.actions,
      );
    });
    if (widget.hapticsEnabled) {
      unawaited(HapticFeedback.mediumImpact());
    }
  }

  void _move(LongPressMoveUpdateDetails details) {
    final host = _overlayHost;
    if (host != null) {
      host.move(owner: this, globalPointer: details.globalPosition);
      return;
    }
    final current = _session;
    if (current == null) return;
    final next = current.move(details.localPosition);
    final selectionChanged = next.selection?.key != current.selection?.key;
    setState(() => _session = next);
    if (widget.hapticsEnabled &&
        selectionChanged &&
        next.pointerTouchesSelection) {
      unawaited(HapticFeedback.lightImpact());
    }
  }

  void _commit() {
    final host = _overlayHost;
    if (host != null) {
      host.commit(this);
      return;
    }
    final current = _session;
    if (current == null) return;
    final selected = current.selectedAction;
    setState(() => _session = null);
    if (selected == null) return;
    if (widget.hapticsEnabled) {
      unawaited(HapticFeedback.mediumImpact());
    }
    widget.onSelected(selected);
  }

  void _cancel() {
    final host = _overlayHost;
    if (host != null) {
      host.cancel(this);
      return;
    }
    if (_session == null) return;
    setState(() => _session = null);
  }
}

final class TerminalPetalMenu extends StatelessWidget {
  const TerminalPetalMenu({super.key, required this.session});

  final TerminalPetalMenuSession session;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final layers = session._visibleLayers;
    final expanded = session.expandedPath;
    final guideSegments = <({Offset from, Offset to})>[
      for (final layer in layers)
        for (final point in layer.positions) (from: layer.start, to: point),
    ];
    return IgnorePointer(
      key: terminalPetalMenuKey,
      child: ExcludeSemantics(
        child: Stack(
          children: [
            const Positioned.fill(child: ColoredBox(color: Color(0x66000000))),
            Positioned.fill(
              child: CustomPaint(
                painter: _TerminalPetalGuidePainter(
                  origin: session.origin,
                  center: session.center,
                  segments: guideSegments,
                  color: palette.borderStrong.withValues(alpha: 0.72),
                ),
              ),
            ),
            for (
              var layerIndex = 0;
              layerIndex < layers.length;
              layerIndex += 1
            )
              for (
                var index = 0;
                index < layers[layerIndex].actions.length;
                index += 1
              )
                _AnimatedPetalNode(
                  start: layers[layerIndex].start,
                  target: layers[layerIndex].positions[index],
                  action: layers[layerIndex].actions[index],
                  selected: _samePath(session.selection?.path, [
                    ...layers[layerIndex].parentPath,
                    index,
                  ]),
                  active: _samePath(expanded, [
                    ...layers[layerIndex].parentPath,
                    index,
                  ]),
                  child: layers[layerIndex].parentPath.isNotEmpty,
                  opacity: switch (layers.length - 1 - layerIndex) {
                    0 => 1,
                    1 => 0.62,
                    _ => 0.42,
                  },
                ),
            _positionedAt(
              session.center,
              const _TerminalPetalCancelNode(),
              extent: 40,
            ),
            if ((session.origin - session.center).distance > 6)
              _positionedAt(
                session.origin,
                DecoratedBox(
                  decoration: BoxDecoration(
                    color: palette.accent,
                    shape: BoxShape.circle,
                  ),
                ),
                extent: 8,
              ),
          ],
        ),
      ),
    );
  }
}

final class _AnimatedPetalNode extends StatelessWidget {
  const _AnimatedPetalNode({
    required this.start,
    required this.target,
    required this.action,
    required this.selected,
    required this.active,
    required this.opacity,
    this.child = false,
  });

  final Offset start;
  final Offset target;
  final TerminalPetalMenuItem action;
  final bool selected;
  final bool active;
  final double opacity;
  final bool child;

  @override
  Widget build(BuildContext context) {
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0, end: 1),
      duration: AnyttyMotion.resolve(
        context,
        child ? AnyttyMotion.quick : AnyttyMotion.standard,
      ),
      curve: AnyttyMotion.emphasized,
      builder: (context, value, _) {
        final position = Offset.lerp(start, target, value)!;
        return _positionedAt(
          position,
          AnimatedOpacity(
            key: ValueKey('terminal-petal-opacity-${action.id}'),
            opacity: opacity,
            duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
            curve: Curves.easeOutCubic,
            child: Transform.scale(
              scale: 0.78 + value * 0.22,
              child: _TerminalPetalNode(
                action: action,
                selected: selected,
                active: active,
              ),
            ),
          ),
        );
      },
    );
  }
}

final class _TerminalPetalNode extends StatelessWidget {
  const _TerminalPetalNode({
    required this.action,
    required this.selected,
    required this.active,
  });

  final TerminalPetalMenuItem action;
  final bool selected;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final foreground = selected
        ? palette.accentText
        : action.enabled
        ? palette.text
        : palette.faint;
    return AnimatedScale(
      scale: selected ? 1.12 : 1,
      duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
      curve: AnyttyMotion.emphasized,
      child: AnimatedContainer(
        key: ValueKey('terminal-petal-action-${action.id}'),
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        curve: AnyttyMotion.emphasized,
        width: terminalPetalNodeExtent,
        height: terminalPetalNodeExtent,
        padding: const EdgeInsets.fromLTRB(6, 7, 6, 6),
        decoration: BoxDecoration(
          color: selected
              ? palette.accent
              : active
              ? palette.surfaceRaised
              : palette.surface,
          shape: BoxShape.circle,
          border: Border.all(
            color: selected || active ? palette.accent : palette.borderStrong,
            width: selected ? 2 : 1,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: selected ? 0.48 : 0.3),
              blurRadius: selected ? 14 : 8,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        foregroundDecoration: action.enabled
            ? null
            : BoxDecoration(
                color: palette.background.withValues(alpha: 0.38),
                shape: BoxShape.circle,
              ),
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(action.icon, size: 17, color: foreground),
                const SizedBox(height: 3),
                SizedBox(
                  width: 44,
                  height: 11,
                  child: FittedBox(
                    fit: BoxFit.scaleDown,
                    child: Text(
                      action.label,
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
            if (action.hasChildren)
              Positioned(
                top: -1,
                right: -1,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: active ? palette.accent : palette.muted,
                    shape: BoxShape.circle,
                  ),
                  child: const SizedBox.square(dimension: 7),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

final class _TerminalPetalCancelNode extends StatelessWidget {
  const _TerminalPetalCancelNode();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        color: palette.surface.withValues(alpha: 0.94),
        shape: BoxShape.circle,
        border: Border.all(color: palette.borderStrong),
      ),
      child: Center(child: Icon(LucideIcons.x, size: 15, color: palette.muted)),
    );
  }
}

final class _TerminalPetalGuidePainter extends CustomPainter {
  const _TerminalPetalGuidePainter({
    required this.origin,
    required this.center,
    required this.segments,
    required this.color,
  });

  final Offset origin;
  final Offset center;
  final List<({Offset from, Offset to})> segments;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 1.25
      ..style = PaintingStyle.stroke;
    if ((origin - center).distance > 6) canvas.drawLine(origin, center, paint);
    for (final segment in segments) {
      canvas.drawLine(segment.from, segment.to, paint);
    }
  }

  @override
  bool shouldRepaint(covariant _TerminalPetalGuidePainter oldDelegate) =>
      oldDelegate.origin != origin ||
      oldDelegate.center != center ||
      oldDelegate.segments != segments ||
      oldDelegate.color != color;
}

Widget _positionedAt(Offset position, Widget child, {double? extent}) {
  final size = extent ?? terminalPetalNodeExtent;
  return Positioned(
    left: position.dx - size / 2,
    top: position.dy - size / 2,
    width: size,
    height: size,
    child: child,
  );
}

Offset _polar(double radius, double angle) =>
    Offset(math.cos(angle) * radius, math.sin(angle) * radius);

double _clampAxis(double value, double extent, double inset) {
  if (!extent.isFinite || extent <= 0) return value;
  if (extent <= inset * 2) return extent / 2;
  return value.clamp(inset, extent - inset).toDouble();
}

int? _nearestEnabled({
  required List<Offset> points,
  required List<bool> enabled,
  required Offset pointer,
  required double maximumDistance,
}) {
  int? nearest;
  var nearestDistance = maximumDistance;
  for (var index = 0; index < points.length; index += 1) {
    if (!enabled[index]) continue;
    final distance = (pointer - points[index]).distance;
    if (distance <= nearestDistance) {
      nearest = index;
      nearestDistance = distance;
    }
  }
  return nearest;
}

bool _samePath(List<int>? left, List<int>? right) {
  if (left == null || right == null || left.length != right.length) {
    return false;
  }
  for (var index = 0; index < left.length; index += 1) {
    if (left[index] != right[index]) return false;
  }
  return true;
}
