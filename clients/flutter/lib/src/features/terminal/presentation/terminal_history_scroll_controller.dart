import 'package:flutter/widgets.dart';

/// Keeps the visible rows stationary when a history page is prepended.
final class TerminalHistoryScrollController extends ScrollController {
  void preservePrepend(double extent) {
    if (!hasClients || extent == 0) return;
    (position as _HistoryScrollPosition).preservePrepend(extent);
  }

  @override
  ScrollPosition createScrollPosition(
    ScrollPhysics physics,
    ScrollContext context,
    ScrollPosition? oldPosition,
  ) => _HistoryScrollPosition(
    physics: physics,
    context: context,
    oldPosition: oldPosition,
  );
}

final class _HistoryScrollPosition extends ScrollPositionWithSingleContext {
  _HistoryScrollPosition({
    required super.physics,
    required super.context,
    ScrollPosition? oldPosition,
  }) : super(oldPosition: oldPosition) {
    if (oldPosition is _HistoryScrollPosition) {
      _pendingPrepend = oldPosition._pendingPrepend;
    }
  }

  double _pendingPrepend = 0;

  void preservePrepend(double extent) {
    _pendingPrepend += extent;
    // A lazy list can reuse its current layout after a delegate update.
    // Explicitly invalidate the viewport so it consumes the correction.
    notifyListeners();
  }

  @override
  bool applyContentDimensions(double minScrollExtent, double maxScrollExtent) {
    final prepend = _pendingPrepend;
    if (prepend != 0) {
      _pendingPrepend = 0;
      // Use the current pixels, including any drag since the request started.
      // Correct during layout, before paint, without cancelling the gesture.
      correctBy(prepend);
      super.applyContentDimensions(minScrollExtent, maxScrollExtent);
      return false;
    }
    return super.applyContentDimensions(minScrollExtent, maxScrollExtent);
  }
}
