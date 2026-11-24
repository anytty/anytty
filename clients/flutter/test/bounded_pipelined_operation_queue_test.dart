import 'dart:async';

import 'package:anytty_native/src/features/terminal/domain/bounded_serial_operation_queue.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('保持提交顺序，同时允许最多 N 个在飞', () async {
    final queue = BoundedPipelinedOperationQueue(
      maximumInflight: 2,
      maximumOperations: 16,
      maximumCost: 1024,
    );
    final started = <int>[];
    final gates = <Completer<void>>[];

    final futures = <Future<void>>[];
    for (var index = 0; index < 4; index += 1) {
      final gate = Completer<void>();
      gates.add(gate);
      futures.add(
        queue.schedule<void>(
          cost: 1,
          operation: () async {
            started.add(index);
            await gate.future;
          },
        ),
      );
    }

    await Future<void>.delayed(Duration.zero);
    expect(started, [0, 1], reason: 'maxInflight=2 时只能起两个');
    expect(queue.inflight, 2);

    gates[1].complete();
    await Future<void>.delayed(Duration.zero);
    expect(started, [0, 1, 2], reason: '空出槽位后按提交顺序启动下一个');

    gates[0].complete();
    gates[2].complete();
    await Future<void>.delayed(Duration.zero);
    expect(started, [0, 1, 2, 3]);

    gates[3].complete();
    await Future.wait(futures);
    expect(queue.inflight, 0);
    expect(queue.pendingOperations, 0);
  });

  test('单个操作失败不会卡住后续操作', () async {
    final queue = BoundedPipelinedOperationQueue(
      maximumInflight: 1,
      maximumOperations: 8,
      maximumCost: 64,
    );
    final completed = <int>[];

    final failed = queue.schedule<void>(
      cost: 1,
      operation: () async => throw StateError('boom'),
    );
    final next = queue.schedule<void>(
      cost: 1,
      operation: () async => completed.add(1),
    );

    await expectLater(failed, throwsStateError);
    await next;
    expect(completed, [1]);
  });

  test('超过在队上限时报错，不影响已入队操作', () async {
    final queue = BoundedPipelinedOperationQueue(
      maximumInflight: 1,
      maximumOperations: 2,
      maximumCost: 64,
    );
    final gate = Completer<void>();
    final first = queue.schedule<void>(cost: 1, operation: () => gate.future);
    final second = queue.schedule<void>(cost: 1, operation: () async {});
    final third = queue.schedule<void>(
      cost: 1,
      operation: () async {},
      overflowError: () => StateError('full'),
    );

    await expectLater(third, throwsStateError);
    gate.complete();
    await Future.wait([first, second]);
  });
}
