import 'dart:math' as math;

/// Returns a monotonic progress value that approaches, but never reaches, the
/// loading ceiling until the real page load completes.
double browserFakeLoadProgress(Duration elapsed) {
  const floor = 0.06;
  const ceiling = 0.92;
  const timeConstantMilliseconds = 1800.0;
  final milliseconds = math.max(0, elapsed.inMilliseconds).toDouble();
  final curve = 1 - math.exp(-milliseconds / timeConstantMilliseconds);
  return floor + (ceiling - floor) * curve;
}
