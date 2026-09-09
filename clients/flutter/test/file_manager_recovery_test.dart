import 'dart:async';

import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/files/presentation/file_manager_screen.dart';
import 'package:anytty_native/src/features/terminal/data/endpoint_session_client.dart';
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    hide EventEnvelope;
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets(
    'blocks stale files after suspend and reloads with the replacement session',
    (tester) async {
      SharedPreferences.setMockInitialValues({});
      final runtime = _FileRuntime();
      final first = (await tester.runAsync(
        () => EndpointSessionClient.open(runtime, 'device'),
      ))!;
      final replacement = Completer<EndpointSessionClient>();
      var opens = 0;
      final container = ProviderContainer(
        overrides: [
          endpointSessionProvider('device').overrideWith(
            (ref) => ++opens == 1 ? Future.value(first) : replacement.future,
          ),
        ],
      );
      await tester.pumpWidget(
        UncontrolledProviderScope(
          container: container,
          child: MaterialApp(
            theme: anyttyTheme(Brightness.light),
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context).copyWith(disableAnimations: true),
              child: child!,
            ),
            home: const FileManagerScreen(
              endpointId: 'device',
              endpointLabel: 'My device',
              initialPath: '/work',
            ),
          ),
        ),
      );
      for (var frame = 0; frame < 10; frame++) {
        await tester.runAsync(
          () => Future<void>.delayed(const Duration(milliseconds: 10)),
        );
        await tester.pump(const Duration(milliseconds: 100));
      }
      expect(
        find.text('before.txt'),
        findsOneWidget,
        reason:
            'lists=${runtime.lists}; ${tester.widgetList<Text>(find.byType(Text)).map((text) => text.data).join(', ')}',
      );
      expect(runtime.lists, 1);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();
      expect(find.text('Restoring connection'), findsOneWidget);
      final blocked = tester
          .widgetList<IgnorePointer>(find.byType(IgnorePointer))
          .any((widget) => widget.ignoring);
      expect(blocked, isTrue);
      expect(find.byTooltip('Close files'), findsOneWidget);

      container.invalidate(endpointSessionProvider('device'));
      await tester.pump();
      runtime.filename = 'after.txt';
      replacement.complete(
        (await tester.runAsync(
          () => EndpointSessionClient.open(runtime, 'device'),
        ))!,
      );
      for (var frame = 0; frame < 10; frame++) {
        await tester.runAsync(
          () => Future<void>.delayed(const Duration(milliseconds: 10)),
        );
        await tester.pump(const Duration(milliseconds: 100));
      }
      expect(find.text('Restoring connection'), findsNothing);
      expect(find.text('after.txt'), findsOneWidget);
      expect(find.text('before.txt'), findsNothing);
      expect(runtime.lists, 2);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox.shrink());
      container.dispose();
      final closing = runtime.controller.close();
      await tester.pump();
      await closing;
    },
  );

  testWidgets('a stuck session has a deadline, retry, and an accessible exit', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    final pending = Completer<EndpointSessionClient>();
    final container = ProviderContainer(
      overrides: [
        endpointSessionProvider('offline')
            .overrideWith((ref) => pending.future),
      ],
    );
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          theme: anyttyTheme(Brightness.light),
          home: const FileManagerScreen(
            endpointId: 'offline',
            endpointLabel: 'Offline device',
            initialPath: '/',
          ),
        ),
      ),
    );
    await tester.pump(const Duration(seconds: 21));
    await tester.pump();
    expect(find.text('Unable to refresh files'), findsOneWidget);
    expect(find.text('Retry'), findsOneWidget);
    expect(find.byTooltip('Close files'), findsOneWidget);
    await tester.tap(find.text('Retry'));
    await tester.pump();
    expect(find.text('Restoring connection'), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
    container.dispose();
  });
}

class _FileRuntime implements AnyttyEngineRuntime {
  final controller = StreamController<EventEnvelope>.broadcast();
  String filename = 'before.txt';
  int lists = 0;
  int next = 1;
  final stamp = EndpointSessionStamp(
    endpointId: 'device',
    routeId: 'direct',
    generation: Int64(1),
  );

  @override
  Stream<EventEnvelope> get events => controller.stream;
  @override
  Stream<int> get foregroundResumes => const Stream.empty();
  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) =>
      EndpointDemandLease(() {});
  @override
  int openSession(OpenSessionRequest request) {
    final handle = next++;
    controller.add(
      EventEnvelope(
        openSession: OpenSessionResult(
          operationHandle: Int64(handle),
          sessionHandle: Int64(100 + handle),
          session: stamp,
        ),
      ),
    );
    return handle;
  }

  @override
  int execute(int sessionHandle, CommandEnvelope request) {
    final handle = next++;
    lists++;
    controller.add(
      EventEnvelope(
        execute: ExecuteResult(
          operationHandle: Int64(handle),
          sessionHandle: Int64(sessionHandle),
          result: ResultEnvelope(
            originSession: stamp,
            fileList: FileListResult(
              path: request.fileList.path,
              entries: [
                FileEntry(
                  name: filename,
                  path: '${request.fileList.path}/$filename',
                  type: FileEntryType.FILE_ENTRY_TYPE_FILE,
                ),
              ],
            ),
          ),
        ),
      ),
    );
    return handle;
  }

  @override
  int command(EngineCommand command) => throw UnimplementedError();
  @override
  void cancel(int operationHandle) {}
  @override
  void release(int handle) {}
  @override
  void closeSession(int sessionHandle) {}
}
