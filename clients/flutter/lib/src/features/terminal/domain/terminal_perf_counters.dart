import 'dart:async';

import 'package:flutter/foundation.dart';

/// 只有用 `--dart-define=ANYTTY_PERF=true` 构建时才采集。
///
/// 生产构建里所有 `if (anyttyPerfEnabled)` 分支都会被 tree-shake，
/// 不产生计时器、不产生分配。
const bool anyttyPerfEnabled = bool.fromEnvironment('ANYTTY_PERF');

/// 固定桶直方图：O(1) 累加，不保留原始样本，适合长期在真机上跑。
final class PerfHistogram {
  PerfHistogram(this.boundaries) : _counts = List<int>.filled(boundaries.length + 1, 0);

  /// 每个桶的上界（升序）。
  final List<int> boundaries;
  final List<int> _counts;
  int _total = 0;

  int get count => _total;

  void add(int value) {
    _total += 1;
    for (var index = 0; index < boundaries.length; index += 1) {
      if (value <= boundaries[index]) {
        _counts[index] += 1;
        return;
      }
    }
    _counts[_counts.length - 1] += 1;
  }

  /// 百分位（0~1）。返回桶上界，未采样时返回 null。
  int? percentile(double ratio) {
    if (_total == 0) return null;
    final target = (_total * ratio).ceil().clamp(1, _total);
    var seen = 0;
    for (var index = 0; index < _counts.length; index += 1) {
      seen += _counts[index];
      if (seen >= target) {
        return index < boundaries.length ? boundaries[index] : boundaries.last;
      }
    }
    return boundaries.last;
  }

  String summary(String label, String unit) {
    final p50 = percentile(0.5);
    final p95 = percentile(0.95);
    if (p50 == null || p95 == null) return '$label=n/a';
    return '$label=$p50$unit/$p95$unit';
  }

  void reset() {
    _counts.fillRange(0, _counts.length, 0);
    _total = 0;
  }
}

/// 终端交互延迟的进程级计数器。
///
/// 只在 [anyttyPerfEnabled] 时采集并每 5 秒输出一行汇总，供
/// `adb logcat | grep ANYTTY_PERF` 取基线。
final class TerminalPerfCounters {
  TerminalPerfCounters._();

  static final TerminalPerfCounters instance = TerminalPerfCounters._();

  static const _timeBoundaries = <int>[
    200, 500, 1000, 2000, 4000, 8000, 16000, 24000,
    33000, 50000, 80000, 120000, 200000, 400000, 800000, 2000000,
  ];
  static const _byteBoundaries = <int>[
    512, 1024, 2048, 4096, 8192, 16384, 32768, 65536,
    131072, 262144, 524288, 1048576, 2097152, 4194304, 8388608, 16777216,
  ];

  final frameBytes = PerfHistogram(_byteBoundaries);
  final decodeUs = PerfHistogram(_timeBoundaries);
  final mergeUs = PerfHistogram(_timeBoundaries);
  final paintUs = PerfHistogram(_timeBoundaries);
  final pollRoundtripUs = PerfHistogram(_timeBoundaries);
  final inputQueueUs = PerfHistogram(_timeBoundaries);
  final inputAckUs = PerfHistogram(_timeBoundaries);
  final runsPainted = PerfHistogram(const [
    32, 64, 128, 256, 512, 768, 1024, 1536,
    2048, 3072, 4096, 6144, 8192, 12288, 16384, 32768,
  ]);

  int frames = 0;
  int fullReplaceFrames = 0;

  Timer? _timer;
  int _runsThisFrame = 0;

  void start() {
    if (!anyttyPerfEnabled || _timer != null) return;
    _timer = Timer.periodic(const Duration(seconds: 5), (_) => _report());
    debugPrint('ANYTTY_PERF enabled');
  }

  void noteDecode(int bytes, int micros) {
    if (!anyttyPerfEnabled) return;
    frameBytes.add(bytes);
    decodeUs.add(micros);
  }

  void noteMerge(int micros, {required bool fullReplace}) {
    if (!anyttyPerfEnabled) return;
    mergeUs.add(micros);
    frames += 1;
    if (fullReplace) fullReplaceFrames += 1;
  }

  void beginPaint() {
    if (!anyttyPerfEnabled) return;
    _runsThisFrame = 0;
  }

  void noteRun() {
    if (!anyttyPerfEnabled) return;
    _runsThisFrame += 1;
  }

  void endPaint(int micros) {
    if (!anyttyPerfEnabled) return;
    paintUs.add(micros);
    runsPainted.add(_runsThisFrame);
  }

  void notePollRoundtrip(int micros) {
    if (!anyttyPerfEnabled) return;
    pollRoundtripUs.add(micros);
  }

  void noteInputQueue(int micros) {
    if (!anyttyPerfEnabled) return;
    inputQueueUs.add(micros);
  }

  void noteInputAck(int micros) {
    if (!anyttyPerfEnabled) return;
    inputAckUs.add(micros);
  }

  String snapshot() {
    final parts = [
      'frames=$frames',
      'full=$fullReplaceFrames',
      frameBytes.summary('frame', 'B'),
      pollRoundtripUs.summary('poll', 'us'),
      decodeUs.summary('decode', 'us'),
      mergeUs.summary('merge', 'us'),
      paintUs.summary('paint', 'us'),
      runsPainted.summary('runs', ''),
      inputQueueUs.summary('in_q', 'us'),
      inputAckUs.summary('in_ack', 'us'),
    ];
    return parts.join(' ');
  }

  void _report() {
    debugPrint('ANYTTY_PERF ${snapshot()}');
    for (final histogram in <PerfHistogram>[
      frameBytes,
      decodeUs,
      mergeUs,
      paintUs,
      pollRoundtripUs,
      inputQueueUs,
      inputAckUs,
      runsPainted,
    ]) {
      histogram.reset();
    }
    frames = 0;
    fullReplaceFrames = 0;
  }
}

/// 计时辅助：只有 perf 打开时才真正计时。
final class PerfStopwatch {
  PerfStopwatch() : _watch = anyttyPerfEnabled ? (Stopwatch()..start()) : null;

  final Stopwatch? _watch;

  int? get elapsedUs => _watch?.elapsedMicroseconds;
}
