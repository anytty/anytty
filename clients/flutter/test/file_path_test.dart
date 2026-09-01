import 'package:anytty_native/src/features/files/domain/file_path.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('normalizes Unix and Windows paths without losing roots', () {
    expect(normalizeFilePath(' /srv/app/// '), '/srv/app');
    expect(normalizeFilePath(r'C:\Users\Ada\'), 'C:/Users/Ada');
    expect(normalizeFilePath('C:///'), 'C:/');
    expect(normalizeFilePath(''), '/');
  });

  test('finds parents for Unix drive and UNC roots', () {
    expect(parentFilePath('/srv/app'), '/srv');
    expect(parentFilePath('/'), '/');
    expect(parentFilePath('C:/Users/Ada'), 'C:/Users');
    expect(parentFilePath('C:/'), 'C:/');
    expect(parentFilePath('//server/share'), '//server/share');
    expect(parentFilePath('//server/share/app'), '//server/share');
  });

  test('joins child names and builds navigable breadcrumbs', () {
    expect(joinFilePath('/', 'tmp'), '/tmp');
    expect(joinFilePath('C:/Users', r'Ada\docs'), 'C:/Users/Ada/docs');
    expect(filePathBreadcrumbs('/srv/app'), const [
      FileBreadcrumb(label: '/', path: '/'),
      FileBreadcrumb(label: 'srv', path: '/srv'),
      FileBreadcrumb(label: 'app', path: '/srv/app'),
    ]);
    expect(filePathBreadcrumbs('C:/Users/Ada').map((item) => item.path), [
      '/',
      'C:/',
      'C:/Users',
      'C:/Users/Ada',
    ]);
  });
}
