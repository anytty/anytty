import 'package:anytty_native/src/features/terminal/domain/terminal_links.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final row = ScreenRow(
    cells: [
      ScreenCell(content: 'ab', width: 2),
      ScreenCell(
        content: '\u754c',
        width: 2,
        linkUrl: 'https://example.com/docs',
        linkParams: 'id=docs',
      ),
      ScreenCell(content: 'z', width: 1),
    ],
  );

  test('hits canonical links across the full display width of a cell', () {
    expect(terminalLinkAtColumn(row, 1), isNull);
    expect(terminalLinkAtColumn(row, 2)?.url, 'https://example.com/docs');
    expect(terminalLinkAtColumn(row, 3)?.params, 'id=docs');
    expect(terminalLinkAtColumn(row, 4), isNull);
  });

  test('allows external schemes and rejects executable or local schemes', () {
    expect(externalTerminalLink('https://example.com')?.host, 'example.com');
    expect(externalTerminalLink('mailto:dev@example.com')?.scheme, 'mailto');
    expect(externalTerminalLink('javascript:alert(1)'), isNull);
    expect(externalTerminalLink('data:text/plain,no'), isNull);
    expect(externalTerminalLink('file:///etc/passwd'), isNull);
    expect(externalTerminalLink('example.com/no-scheme'), isNull);
  });

  test('finds absolute, relative, quoted, and compiler output paths', () {
    final paths = terminalPathsInRow(
      ScreenRow(
        cells: [
          ScreenCell(
            content:
                'lib/main.dart:42:7 ../logs/app.log "docs/Release Notes.md"',
            width: 60,
          ),
        ],
      ),
    );

    expect(paths.map((path) => path.path), [
      'lib/main.dart',
      '../logs/app.log',
      'docs/Release Notes.md',
    ]);
    expect(paths.first.raw, 'lib/main.dart:42:7');
    final absoluteRow = ScreenRow(
      cells: [ScreenCell(content: '/srv/app/main.go:9', width: 18)],
    );
    expect(terminalPathAtColumn(absoluteRow, 5)?.path, '/srv/app/main.go');
  });

  test('maps paths after wide terminal cells to display columns', () {
    final wideRow = ScreenRow(
      cells: [
        ScreenCell(content: '\u754c', width: 2),
        ScreenCell(content: ' src/main.dart', width: 14),
      ],
    );

    final path = terminalPathsInRow(wideRow).single;
    expect(path.startColumn, 3);
    expect(path.endColumn, 16);
    expect(terminalPathAtColumn(wideRow, 3)?.path, 'src/main.dart');
    expect(terminalPathAtColumn(wideRow, 2), isNull);
  });

  test(
    'does not classify URLs, email addresses, or ordinary words as paths',
    () {
      final paths = terminalPathsInRow(
        ScreenRow(
          cells: [
            ScreenCell(
              content: 'https://example.com/a dev@example.com build complete',
              width: 52,
            ),
          ],
        ),
      );

      expect(paths, isEmpty);
    },
  );

  test(
    'resolves relative terminal paths against the current working folder',
    () {
      expect(
        resolveTerminalPath('src/main.dart:12:4', cwd: '/home/ada/project'),
        '/home/ada/project/src/main.dart',
      );
      expect(
        resolveTerminalPath('../logs/app.log', cwd: '/home/ada/project'),
        '/home/ada/logs/app.log',
      );
      expect(
        resolveTerminalPath(r'..\logs\app.log', cwd: r'C:\work\app'),
        'C:/work/logs/app.log',
      );
      expect(
        resolveTerminalPath('/var/../srv/app', cwd: '/home/ada/project'),
        '/srv/app',
      );
      expect(resolveTerminalPath('src/main.dart', cwd: ''), isNull);
      expect(resolveTerminalPath('~/src/main.dart', cwd: '/home/ada'), isNull);
    },
  );

  test('removes a descriptive label before a relative path', () {
    for (final content in [
      '新版APK:clients/flutter/build/app-release.apk',
      'APK：clients/flutter/build/app-release.apk',
      '新版APK：[app-release.apk](clients/flutter/build/app-release.apk)',
    ]) {
      final labeledRow = ScreenRow(
        cells: [ScreenCell(content: content, width: content.length)],
      );
      final path = terminalPathsInRow(labeledRow).single;
      expect(path.raw, 'clients/flutter/build/app-release.apk');
      expect(path.path, 'clients/flutter/build/app-release.apk');
      expect(path.startColumn, greaterThan(0));
    }
  });

  test('projects one soft-wrapped path onto every physical row', () {
    final first = ScreenRow(
      cells: [ScreenCell(content: '/workspaces/anytty/clients/', width: 27)],
      wrapped: true,
    );
    final second = ScreenRow(
      cells: [ScreenCell(content: 'flutter/lib/src/main.dart:18', width: 28)],
    );
    final layout = TerminalPathLayout.screenRows([first, second]);
    const expected = '/workspaces/anytty/clients/flutter/lib/src/main.dart';

    expect(layout.pathsInRow(0).single.path, expected);
    expect(layout.pathsInRow(1).single.path, expected);
    expect(layout.pathAt(1, 3)?.startColumn, 0);
    expect(layout.pathsInRow(0).single.endColumn, 27);
    expect(layout.pathsInRow(1).single.endColumn, 28);
  });

  test('prefers the foreground process cwd and keeps fallback candidates', () {
    expect(
      resolveTerminalPathCandidates(
        'clients/flutter/pubspec.yaml',
        foregroundCwd: '/workspaces/anytty',
        liveCwd: '/home/ada',
        terminalCwd: '/home/ada',
      ),
      [
        '/workspaces/anytty/clients/flutter/pubspec.yaml',
        '/home/ada/clients/flutter/pubspec.yaml',
      ],
    );
    expect(
      resolveTerminalPathCandidates(
        '/srv/app/main.go',
        foregroundCwd: '/workspaces/anytty',
        liveCwd: '/home/ada',
        terminalCwd: '/tmp',
      ),
      ['/srv/app/main.go'],
    );
  });
}
