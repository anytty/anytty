import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_path_picker_sheet.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('browses remote directories and returns the selected path', (
    tester,
  ) async {
    final prefixes = <String>[];
    String? selection;

    await tester.pumpWidget(
      MaterialApp(
        theme: anyttyTheme(Brightness.light),
        home: Navigator(
          onGenerateRoute: (_) => MaterialPageRoute<void>(
            builder: (context) => PopScope<String>(
              onPopInvokedWithResult: (didPop, result) {
                if (didPop) selection = result;
              },
              child: TerminalPathPickerSheet(
                initialPath: '/workspace',
                loadDirectories: (prefix) async {
                  prefixes.add(prefix);
                  return PathListDirectoriesResult(
                    entries: prefix == '/workspace/'
                        ? [
                            PathDirectoryEntry(
                              name: 'src',
                              path: '/workspace/src/',
                            ),
                          ]
                        : const [],
                  );
                },
              ),
            ),
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();
    expect(prefixes, ['/workspace/']);
    expect(find.text('src'), findsOneWidget);

    await tester.tap(find.text('src'));
    await tester.pumpAndSettle();

    expect(prefixes, ['/workspace/', '/workspace/src/']);
    expect(find.text('/workspace/src'), findsOneWidget);

    await tester.tap(find.text('Use this path'));
    await tester.pumpAndSettle();

    expect(selection, '/workspace/src');
  });
}
