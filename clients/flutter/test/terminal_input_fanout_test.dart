import 'package:anytty_native/src/features/terminal/domain/terminal_input_fanout.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('attempts every target exactly once when all reject input', () async {
    final attempts = <String>[];

    await expectLater(
      dispatchTerminalInput(['shell', 'logs'], (target) async {
        attempts.add(target);
        throw StateError('$target rejected');
      }),
      throwsStateError,
    );

    expect(attempts, ['shell', 'logs']);
  });

  test('accepts partial success without retrying either target', () async {
    final attempts = <String, int>{};

    await dispatchTerminalInput(['shell', 'logs'], (target) async {
      attempts.update(target, (count) => count + 1, ifAbsent: () => 1);
      if (target == 'logs') throw StateError('logs rejected');
    });

    expect(attempts, {'shell': 1, 'logs': 1});
  });

  test('rejects an empty set of ready targets', () async {
    await expectLater(
      dispatchTerminalInput<String>(const [], (_) async {}),
      throwsStateError,
    );
  });
}
