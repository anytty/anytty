import 'dart:async';

final class TerminalRecoveryNoticeGate {
  TerminalRecoveryNoticeGate({
    required this.onVisibilityChanged,
    this.delay = const Duration(milliseconds: 1200),
  });

  final void Function(bool visible) onVisibilityChanged;
  final Duration delay;

  Timer? _timer;
  bool _visible = false;
  bool _disposed = false;

  bool get visible => _visible;

  void setRecovering(bool recovering) {
    if (_disposed) return;
    _timer?.cancel();
    _timer = null;
    if (!recovering) {
      _setVisible(false);
      return;
    }
    _setVisible(false);
    _timer = Timer(delay, () {
      _timer = null;
      if (!_disposed) _setVisible(true);
    });
  }

  void _setVisible(bool visible) {
    if (_visible == visible) return;
    _visible = visible;
    onVisibilityChanged(visible);
  }

  void dispose() {
    _disposed = true;
    _timer?.cancel();
    _timer = null;
  }
}
