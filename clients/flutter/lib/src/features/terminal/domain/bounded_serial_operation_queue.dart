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

/// 有界流水线队列：保持提交顺序，但允许最多 [maximumInflight] 个操作
/// 同时在飞。
///
/// 终端输入用它替代「等上一个 ack 才发下一个」的串行队列：按键顺序由
/// 单条有序通道 + 服务端有序写入保证，客户端不必为每个按键付一个 RTT。
final class BoundedPipelinedOperationQueue {
  BoundedPipelinedOperationQueue({
    required this.maximumInflight,
    required this.maximumOperations,
    required this.maximumCost,
  }) : assert(maximumInflight > 0),
       assert(maximumOperations > 0),
       assert(maximumCost > 0);

  final int maximumInflight;
  final int maximumOperations;
  final int maximumCost;

  final List<_PendingOperation<Object?>> _pending = [];
  int _inflight = 0;
  int _pendingOperations = 0;
  int _pendingCost = 0;

  int get pendingOperations => _pendingOperations;
  int get pendingCost => _pendingCost;
  int get inflight => _inflight;

  Future<T> schedule<T>({
    required int cost,
    required Future<T> Function() operation,
    Object Function()? overflowError,
  }) {
    if (cost < 0) {
      return Future.error(ArgumentError.value(cost, 'cost must be >= 0'));
    }
    if (_pendingOperations >= maximumOperations ||
        cost > maximumCost - _pendingCost) {
      return Future.error(
        overflowError?.call() ?? StateError('Pipelined operation queue is full'),
      );
    }
    final completer = Completer<T>();
    _pendingOperations += 1;
    _pendingCost += cost;
    _pending.add(
      _PendingOperation<T>(completer: completer, cost: cost, operation: operation),
    );
    _drain();
    return completer.future;
  }

  void _drain() {
    while (_inflight < maximumInflight && _pending.isNotEmpty) {
      final next = _pending.removeAt(0);
      _inflight += 1;
      next.run(() {
        _inflight -= 1;
        _pendingOperations -= 1;
        _pendingCost -= next.cost;
        _drain();
      });
    }
  }
}

final class _PendingOperation<T> {
  _PendingOperation({
    required this.completer,
    required this.cost,
    required this.operation,
  });

  final Completer<T> completer;
  final int cost;
  final Future<T> Function() operation;

  Future<void> run(void Function() release) async {
    try {
      completer.complete(await operation());
    } catch (error, stackTrace) {
      completer.completeError(error, stackTrace);
    } finally {
      release();
    }
  }
}
