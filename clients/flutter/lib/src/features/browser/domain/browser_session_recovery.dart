import 'dart:async';

// A failed provider Future can complete immediately. Retries must yield to the
// event loop, otherwise recovery starves rendering and native platform replies.
final class BrowserSessionRecovery {
  BrowserSessionRecovery({
    required this.recover,
    required this.needsRecovery,
    required this.onError,
  });

  final Future<void> Function() recover;
  final bool Function() needsRecovery;
  final void Function(Object, StackTrace) onError;
  Timer? _timer;
  bool _running = false;
  bool _cancelled = false;
  int _retryMilliseconds = 250;

  bool get isRunning => _running;

  void start() {
    if (_running || _cancelled) return;
    _running = true;
    _timer = Timer(Duration.zero, _attempt);
  }

  void cancel() {
    _cancelled = true;
    _running = false;
    _timer?.cancel();
    _timer = null;
  }

  Future<void> _attempt() async {
    _timer = null;
    if (_cancelled || !needsRecovery()) {
      _running = false;
      return;
    }
    try {
      await recover();
    } catch (error, stack) {
      if (!_cancelled) onError(error, stack);
    }
    if (_cancelled || !needsRecovery()) {
      _running = false;
      return;
    }
    _timer = Timer(Duration(milliseconds: _retryMilliseconds), _attempt);
    _retryMilliseconds = (_retryMilliseconds * 2).clamp(250, 5000);
  }
}
