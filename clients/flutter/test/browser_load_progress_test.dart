import 'package:flutter_test/flutter_test.dart';

import 'package:anytty_native/src/features/browser/domain/browser_load_progress.dart';

void main() {
  test('fake loading progress is monotonic and eases toward its ceiling', () {
    final values = [
      browserFakeLoadProgress(Duration.zero),
      browserFakeLoadProgress(const Duration(milliseconds: 250)),
      browserFakeLoadProgress(const Duration(seconds: 1)),
      browserFakeLoadProgress(const Duration(seconds: 5)),
      browserFakeLoadProgress(const Duration(minutes: 1)),
    ];

    expect(values, orderedEquals(values.toList()..sort()));
    expect(values.first, closeTo(0.06, 0.001));
    expect(values[2], greaterThan(0.35));
    expect(values[3], lessThan(0.92));
    expect(values.last, closeTo(0.92, 0.001));
  });
}
