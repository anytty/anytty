import 'dart:async';

import 'package:anytty_native/src/features/terminal/domain/bounded_serial_operation_queue.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'runs accepted operations in order with bounded resident cost',
    () async {
      final queue = BoundedSerialOperationQueue(
        maximumOperations: 3,
        maximumCost: 6,
      );
      final firstGate = Completer<void>();
      final started = <int>[];

      final first = queue.schedule(
        cost: 2,
        operation: () async {
          started.add(1);
          await firstGate.future;
          return 1;
        },
      );
      final second = queue.schedule(
        cost: 3,
        operation: () async {
          started.add(2);
          return 2;
        },
      );

      await Future<void>.delayed(Duration.zero);
      expect(started, [1]);
      expect(queue.pendingOperations, 2);
      expect(queue.pendingCost, 5);

      firstGate.complete();
      expect(await first, 1);
      expect(await second, 2);
      expect(started, [1, 2]);
      expect(queue.pendingOperations, 0);
      expect(queue.pendingCost, 0);
    },
  );

  test('rejects overflow without invoking the operation', () async {
    final queue = BoundedSerialOperationQueue(
      maximumOperations: 1,
      maximumCost: 4,
    );
    final gate = Completer<void>();
    var overflowInvoked = false;

    final accepted = queue.schedule(
      cost: 4,
      operation: () async => gate.future,
    );
    final rejected = queue.schedule<void>(
      cost: 1,
      overflowError: () => const FormatException('full'),
      operation: () async {
        overflowInvoked = true;
      },
    );

    await expectLater(rejected, throwsA(isA<FormatException>()));
    expect(overflowInvoked, isFalse);
    gate.complete();
    await accepted;
  });

  test('releases capacity after an operation fails', () async {
    final queue = BoundedSerialOperationQueue(
      maximumOperations: 1,
      maximumCost: 1,
    );

    await expectLater(
      queue.schedule<void>(
        cost: 1,
        operation: () async => throw StateError('failed'),
      ),
      throwsStateError,
    );

    expect(
      await queue.schedule(cost: 1, operation: () async => 'replacement'),
      'replacement',
    );
  });
}
