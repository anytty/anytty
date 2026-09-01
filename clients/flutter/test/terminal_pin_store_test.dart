import 'package:anytty_native/src/features/terminal/data/terminal_pin_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('stores normalized pin order per endpoint', () async {
    SharedPreferences.setMockInitialValues({});
    const store = TerminalPinStore();

    expect(await store.save('endpoint-a', ['b', 'a', 'b', '']), ['b', 'a']);
    expect(await store.load('endpoint-a'), ['b', 'a']);
    expect(await store.load('endpoint-b'), isEmpty);
  });
}
