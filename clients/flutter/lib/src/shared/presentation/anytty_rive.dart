import 'package:flutter/widgets.dart';
import 'package:rive/rive.dart' as rive;

import '../../app/anytty_theme.dart';

enum AnyttyMascotScene {
  connecting(4, true),
  processing(3, true),
  history(1.2, false),
  success(1.5, false),
  failure(1.8, false),
  welcome(2.2, false),
  running(.9, true),
  searching(3.6, true),
  wake(2.6, false);

  const AnyttyMascotScene(this.seconds, this.loop);
  final double seconds;
  final bool loop;
}

/// Local Rive artwork. The host clock pauses offstage and in the background.
final class AnyttyRive extends StatefulWidget {
  const AnyttyRive({
    super.key,
    this.asset = 'anytty-mascot',
    this.timeline = 'processing',
    this.seconds = 3,
    this.loop = true,
    this.visible = true,
    this.crop = const Rect.fromLTWH(0, 0, 513, 546),
    this.gaze = Offset.zero,
  });
  final String asset;
  final String timeline;
  final double seconds;
  final bool loop;
  final bool visible;
  final Rect crop;
  final Offset gaze;
  @override
  State<AnyttyRive> createState() => _AnyttyRiveState();
}

final class _AnyttyRiveState extends State<AnyttyRive>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  late final AnimationController _clock = AnimationController(vsync: this);
  rive.File? _file;
  rive.Artboard? _artboard;
  final Map<String, rive.Animation> _animations = {};
  bool _foreground = true;
  int _generation = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    final lifecycle = WidgetsBinding.instance.lifecycleState;
    _foreground = lifecycle == null || lifecycle == AppLifecycleState.resumed;
    _load();
  }

  Future<void> _load() async {
    final generation = ++_generation;
    rive.File? file;
    rive.Artboard? artboard;
    final animations = <String, rive.Animation>{};
    try {
      file = await rive.File.asset(
        'assets/brand/motion/${widget.asset}.riv',
        riveFactory: rive.Factory.flutter,
      );
      if (!mounted || generation != _generation) {
        file?.dispose();
        return;
      }
      artboard = file?.defaultArtboard();
      if (artboard == null) throw StateError('Missing mascot artboard');
      for (final name in [
        widget.timeline,
        if (widget.asset.startsWith('browser-')) ...[
          'gaze-x',
          'gaze-y',
          'head-follow',
        ],
      ]) {
        final animation = artboard.animationNamed(name);
        if (animation == null) {
          throw StateError('Missing mascot timeline: $name');
        }
        animations[name] = animation;
      }
      setState(() {
        _file = file;
        _artboard = artboard;
        _animations.addAll(animations);
      });
      _sync();
    } catch (error) {
      for (final animation in animations.values) {
        animation.dispose();
      }
      artboard?.dispose();
      file?.dispose();
      debugPrint('AnyTTY motion unavailable: $error');
    }
  }

  void _release() {
    for (final animation in _animations.values) {
      animation.dispose();
    }
    _animations.clear();
    _artboard?.dispose();
    _artboard = null;
    _file?.dispose();
    _file = null;
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(AnyttyRive oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.asset != oldWidget.asset ||
        widget.timeline != oldWidget.timeline) {
      _clock.stop();
      _clock.value = 0;
      _release();
      _load();
    }
    _sync();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _sync();
  }

  void _sync() {
    _clock.duration = Duration(
      microseconds: (widget.seconds * 1000000).round(),
    );
    if (_artboard == null ||
        !widget.visible ||
        !_foreground ||
        AnyttyMotion.disabled(context) ||
        !TickerMode.valuesOf(context).enabled) {
      _clock.stop();
      if (AnyttyMotion.disabled(context)) _clock.value = 0;
    } else if (!_clock.isAnimating) {
      if (widget.loop) {
        _clock.repeat();
      } else if (!_clock.isCompleted) {
        _clock.forward();
      }
    }
  }

  @override
  void dispose() {
    ++_generation;
    WidgetsBinding.instance.removeObserver(this);
    _clock.dispose();
    _release();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => IgnorePointer(
    child: ExcludeSemantics(
      child: RepaintBoundary(
        child: _artboard == null
            ? ClipRect(
                child: FittedBox(
                  fit: BoxFit.contain,
                  child: SizedBox(
                    width: widget.crop.width,
                    height: widget.crop.height,
                    child: Stack(
                      clipBehavior: Clip.hardEdge,
                      children: [
                        Positioned(
                          left: -widget.crop.left,
                          top: -widget.crop.top,
                          width: widget.asset.startsWith('browser-')
                              ? 140
                              : 513,
                          height: widget.asset.startsWith('browser-')
                              ? 174
                              : 546,
                          child: Image.asset(
                            'assets/brand/motion/${widget.asset}.png',
                            fit: BoxFit.fill,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              )
            : CustomPaint(
                painter: _MascotPainter(
                  _artboard!,
                  _animations,
                  _clock,
                  widget.seconds,
                  widget.crop,
                  AnyttyMotion.disabled(context) ? Offset.zero : widget.gaze,
                ),
              ),
      ),
    ),
  );
}

final class _MascotPainter extends CustomPainter {
  _MascotPainter(
    this.artboard,
    this.animations,
    this.clock,
    this.seconds,
    this.crop,
    this.gaze,
  ) : super(repaint: clock);
  final rive.Artboard artboard;
  final Map<String, rive.Animation> animations;
  final Animation<double> clock;
  final double seconds;
  final Rect crop;
  final Offset gaze;
  @override
  void paint(Canvas canvas, Size size) {
    for (final entry in animations.entries) {
      entry.value.time = switch (entry.key) {
        'gaze-x' || 'head-follow' => (gaze.dx + 1) / 2,
        'gaze-y' => (gaze.dy + 1) / 2,
        _ => clock.value * seconds,
      };
      entry.value.advanceAndApply(0);
    }
    canvas.save();
    canvas.clipRect(Offset.zero & size);
    final renderer = rive.Renderer.make(canvas);
    renderer.align(
      rive.Fit.contain,
      Alignment.center,
      rive.AABB.fromValues(0, 0, size.width, size.height),
      rive.AABB.fromValues(crop.left, crop.top, crop.right, crop.bottom),
      1,
    );
    artboard.draw(renderer);
    renderer.dispose();
    canvas.restore();
  }

  @override
  bool shouldRepaint(_MascotPainter oldDelegate) =>
      oldDelegate.artboard != artboard ||
      oldDelegate.crop != crop ||
      oldDelegate.gaze != gaze ||
      oldDelegate.seconds != seconds;
}
