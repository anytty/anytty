import 'dart:io';

import 'package:anytty_native/src/native/native_resource_writer.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'worker startup failure is surfaced and does not hang callers',
    () async {
      if (Platform.isAndroid || Platform.isIOS) return;
      await expectLater(
        NativeResourceWriter.start(1).timeout(const Duration(seconds: 2)),
        throwsStateError,
      );
    },
  );
}
