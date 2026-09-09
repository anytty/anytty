import 'package:anytty_native/src/features/browser/domain/browser_navigation.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IP literals default to HTTP with ports, paths and queries intact', () {
    for (final input in [
      '192.168.1.1',
      '127.0.0.1:8080/app?q=1',
      '[2001:db8::1]:9000/app',
      '[::1]',
      'localhost:3000',
    ]) {
      expect(resolveBrowserNavigation(input).toString(), 'http://$input');
    }
    expect(resolveBrowserNavigation('::1').toString(), 'http://[::1]');
  });

  test('explicit schemes, domains and search keep their semantics', () {
    expect(
      resolveBrowserNavigation('https://192.168.1.1').toString(),
      'https://192.168.1.1',
    );
    expect(
      resolveBrowserNavigation('example.com:8443/a').toString(),
      'https://example.com:8443/a',
    );
    expect(
      resolveBrowserNavigation('192.168.1.1.example.com')?.scheme,
      'https',
    );
    expect(
      resolveBrowserNavigation('some search')?.queryParameters['q'],
      'some search',
    );
    expect(resolveBrowserNavigation('about:blank').toString(), 'about:blank');
  });
}
