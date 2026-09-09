import 'package:flutter/widgets.dart';

import 'anytty_rive.dart';
export 'anytty_rive.dart' show AnyttyMascotScene;

final class AnyttyBrandMark extends StatelessWidget {
  const AnyttyBrandMark({super.key, this.height = 48});
  static const asset = 'assets/brand/mascot.png';
  final double height;
  @override
  Widget build(BuildContext context) => Image.asset(
    asset,
    width: height * 417 / 490,
    height: height,
    fit: BoxFit.contain,
    filterQuality: FilterQuality.high,
    semanticLabel: 'AnyTTY',
  );
}

final class AnyttyBrandLoader extends StatelessWidget {
  const AnyttyBrandLoader({
    super.key,
    this.height = 64,
    this.scene = AnyttyMascotScene.processing,
  });
  final double height;
  final AnyttyMascotScene scene;
  @override
  Widget build(BuildContext context) => SizedBox(
    height: height + 16,
    width: height + 24,
    child: Center(
      child: SizedBox(
        height: height,
        width: height * 513 / 546,
        child: AnyttyRive(
          timeline: scene.name,
          seconds: scene.seconds,
          loop: scene.loop,
        ),
      ),
    ),
  );
}
