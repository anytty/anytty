import 'dart:async';

final class BoundedSerialOperationQueue {
  BoundedSerialOperationQueue({
    required this.maximumOperations,
    required this.maximumCost,
  }) : assert(maximumOperations > 0),
       assert(maximumCost > 0);

  final int maximumOperations;
  final int maximumCost;

  Future<void> _tail = Future.value();
  int _pendingOperations = 0;
  int _pendingCost = 0;

  int get pendingOperations => _pendingOperations;
  int get pendingCost => _pendingCost;

  Future<T> schedule<T>({
    required int cost,
    required Future<T> Function() operation,
    Object Function()? overflowError,
  }) {
    if (cost < 0) {
      return Future.error(ArgumentError.value(cost, 'cost', 'must be >= 0'));
    }
    if (_pendingOperations >= maximumOperations ||
        cost > maximumCost - _pendingCost) {
      return Future.error(
        overflowError?.call() ?? StateError('Serial operation queue is full'),
      );
    }

    _pendingOperations += 1;
    _pendingCost += cost;
    final completer = Completer<T>();

    Future<void> run() async {
      try {
        completer.complete(await operation());
      } catch (error, stackTrace) {
        completer.completeError(error, stackTrace);
      } finally {
        _pendingOperations -= 1;
        _pendingCost -= cost;
      }
    }

    _tail = _tail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}
