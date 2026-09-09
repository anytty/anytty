import 'package:anytty_native/src/features/files/data/file_manager_path_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() => SharedPreferences.setMockInitialValues({}));

  test(
    'stores the last successful path independently for each endpoint',
    () async {
      const store = FileManagerPathStore();

      await store.save('office mac', '/srv/app/');
      await store.save('server', '/var/log');

      expect(await store.load('office mac'), '/srv/app');
      expect(await store.load('server'), '/var/log');
      expect(await store.load('missing'), isNull);
    },
  );

  test('newer paths replace older paths for the same endpoint', () async {
    const store = FileManagerPathStore();

    await store.save('office mac', '/srv');
    await store.save('office mac', '/srv/app/current');

    expect(await store.load('office mac'), '/srv/app/current');
  });

  test('removing an endpoint clears only its remembered path', () async {
    const store = FileManagerPathStore();
    await store.save('removed', '/tmp/removed');
    await store.save('retained', '/tmp/retained');

    await store.remove('removed');

    expect(await store.load('removed'), isNull);
    expect(await store.load('retained'), '/tmp/retained');
  });
}
