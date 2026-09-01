import 'dart:async';
import 'dart:io';

import 'package:anytty_native/src/features/files/data/endpoint_file_transfer.dart';
import 'package:anytty_native/src/features/terminal/data/endpoint_session_client.dart';
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    hide EventEnvelope;
import 'package:anytty_native/src/generated/proto/apipb/application.pb.dart'
    as application;
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/file.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/history.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/wirepb/terminal.pb.dart'
    as wire;
import 'package:anytty_native/src/native/anytty_runtime.dart';
import 'package:anytty_native/src/native/binding_operation.dart';
import 'package:cryptography/dart.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('opens frozen history at explicit local projection columns', () async {
    final terminal = TerminalRef(endpointId: 'studio', terminalId: 'shell');
    var historyCalls = 0;
    final runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.terminalGet => ResultEnvelope(
          terminalGet: TerminalGetResult(
            terminal: TerminalInfo(
              ref: terminal,
              state: TerminalState.TERMINAL_STATE_EXITED,
              size: TerminalSize(cols: 80, rows: 24),
            ),
          ),
        ),
        CommandEnvelope_Command.historyWindow => () {
          historyCalls += 1;
          final cols = command.historyWindow.cols;
          return ResultEnvelope(
            historyWindow: HistoryWindowResult(
              terminal: terminal,
              token: 'frozen-$historyCalls',
              operation:
                  HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
              size: TerminalSize(cols: cols, rows: 24),
              rows: [
                HistoryRow(
                  logicalLineId: Int64(42),
                  row: ScreenRow(
                    cells: [
                      ScreenCell(
                        content: 'abcdefghijklmnopqrstuvwxyz0123456789',
                        width: 36,
                      ),
                    ],
                  ),
                ),
              ],
              historyGeneration: Int64(7),
              logicalTotal: 1,
              viewportAnchor: HistoryViewportAnchor(
                topLineId: Int64(42),
                topCellOffset: 0,
                screenCols: 80,
                screenRows: 24,
              ),
            ),
          );
        }(),
        CommandEnvelope_Command.historyRelease => ResultEnvelope(
          acknowledge: AcknowledgeResult(),
        ),
        _ => throw StateError('unexpected command'),
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');
    final connection = await TerminalConnection.open(client, terminal);

    final wide = await connection.openHistory(cols: 37);
    final narrow = await connection.openHistory(cols: 20);

    expect(wide.history.cols, 37);
    expect(wide.history.rows, hasLength(1));
    expect(narrow.history.cols, 20);
    expect(narrow.history.rows, hasLength(2));
    expect(
      runtime.commands
          .where(
            (command) =>
                command.whichCommand() == CommandEnvelope_Command.historyWindow,
          )
          .map((command) => command.historyWindow.cols),
      [37, 20],
    );
    expect(
      runtime.commands
          .where(
            (command) =>
                command.whichCommand() == CommandEnvelope_Command.historyWindow,
          )
          .map((command) => command.historyWindow.limit),
      [512, 512],
    );
    expect(narrow.history.token, isNot(wide.history.token));
    expect(narrow.history.generation, wide.history.generation);

    await connection.releaseHistory(wide.history);
    await connection.releaseHistory(narrow.history);
    await connection.close();
    client.close();
    await client.closed;
  });

  test(
    'pages frozen alternate-screen rows through the history cursor',
    () async {
      final terminal = TerminalRef(
        endpointId: 'studio',
        terminalId: 'alt-shell',
      );
      var historyCalls = 0;
      final runtime = _SessionRuntime(
        executeResult: (command) => switch (command.whichCommand()) {
          CommandEnvelope_Command.terminalGet => ResultEnvelope(
            terminalGet: TerminalGetResult(
              terminal: TerminalInfo(
                ref: terminal,
                state: TerminalState.TERMINAL_STATE_EXITED,
                size: TerminalSize(cols: 12, rows: 2),
              ),
            ),
          ),
          CommandEnvelope_Command.historyWindow => () {
            historyCalls += 1;
            if (historyCalls == 1) {
              return ResultEnvelope(
                historyWindow: HistoryWindowResult(
                  terminal: terminal,
                  token: 'alt-frozen',
                  operation:
                      HistoryWindowOperation.HISTORY_WINDOW_OPERATION_REPLACE,
                  size: TerminalSize(cols: 6, rows: 2),
                  rows: [
                    _alternateHistoryRow(50, 0, 'first frame row'),
                    _alternateHistoryRow(51, 1, 'second frame row'),
                  ],
                  historyGeneration: Int64(9),
                  logicalTotal: 52,
                  hasMore: true,
                  viewportAnchor: HistoryViewportAnchor(
                    topLineId: Int64(50),
                    screenCols: 12,
                    screenRows: 2,
                  ),
                ),
              );
            }
            return ResultEnvelope(
              historyWindow: HistoryWindowResult(
                terminal: terminal,
                token: 'alt-frozen',
                operation:
                    HistoryWindowOperation.HISTORY_WINDOW_OPERATION_PREPEND,
                size: TerminalSize(cols: 6, rows: 2),
                rows: [
                  HistoryRow(
                    logicalLineId: Int64(40),
                    segment:
                        HistoryCursorSegment.HISTORY_CURSOR_SEGMENT_COMMITTED,
                    row: ScreenRow(
                      cells: [ScreenCell(content: 'older primary', width: 13)],
                    ),
                  ),
                ],
                historyGeneration: Int64(9),
                logicalTotal: 52,
              ),
            );
          }(),
          CommandEnvelope_Command.historyRelease => ResultEnvelope(
            acknowledge: AcknowledgeResult(),
          ),
          _ => throw StateError('unexpected command'),
        },
      );
      final client = await EndpointSessionClient.open(runtime, 'studio');
      final connection = await TerminalConnection.open(client, terminal);

      final latest = await connection.openHistory(cols: 6);
      final merged = await connection.loadOlderHistory(latest.history);

      expect(latest.history.rows, hasLength(2));
      expect(latest.history.rows.every((row) => row.fixedGrid), isTrue);
      expect(latest.history.rows.map((row) => row.screenRows), [0, 1]);
      expect(latest.history.rows.map((row) => row.screenRowSet), [true, true]);
      expect(latest.history.rows.map((row) => row.screenCols), [12, 12]);
      expect(
        latest.history.rows.map((row) => row.segment),
        everyElement(
          HistoryCursorSegment.HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME,
        ),
      );
      expect(merged.prependedRows, 3);
      expect(merged.history.rows.map((row) => row.logicalLineId.toInt()), [
        40,
        40,
        40,
        50,
        51,
      ]);
      final olderCommand = runtime.commands
          .where(
            (command) =>
                command.whichCommand() == CommandEnvelope_Command.historyWindow,
          )
          .last
          .historyWindow;
      expect(olderCommand.token, 'alt-frozen');
      expect(olderCommand.limit, 512);
      expect(olderCommand.historyGeneration, Int64(9));
      expect(olderCommand.beforeCursor.lineId, Int64(50));
      expect(
        olderCommand.beforeCursor.segment,
        HistoryCursorSegment.HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME,
      );

      await connection.releaseHistory(merged.history);
      await connection.close();
      client.close();
      await client.closed;
    },
  );

  test('releases a binding session after its natural close event', () async {
    final runtime = _SessionRuntime();
    final client = await EndpointSessionClient.open(runtime, 'studio');

    runtime.closeNaturally();
    final closed = await client.closed;
    await Future<void>.delayed(Duration.zero);

    expect(closed.sessionHandle.toInt(), 21);
    expect(runtime.released, [11, 21]);

    client.close();
    await Future<void>.delayed(Duration.zero);
    expect(runtime.released, [11, 21]);
  });

  test('explicit close waits for the close event before release', () async {
    final runtime = _SessionRuntime();
    final client = await EndpointSessionClient.open(runtime, 'studio');

    client.close();
    expect(runtime.released, [11]);
    await client.closed;
    await Future<void>.delayed(Duration.zero);

    expect(runtime.closedSessions, [21]);
    expect(runtime.released, [11, 21]);
  });

  test('ignores a close event from a superseded session generation', () async {
    final runtime = _SessionRuntime();
    final client = await EndpointSessionClient.open(runtime, 'studio');

    runtime.emitSessionClosed(
      EndpointSessionStamp(
        endpointId: 'studio',
        routeId: 'direct',
        generation: Int64(2),
      ),
    );
    await Future<void>.delayed(Duration.zero);
    expect(client.isClosed, isFalse);

    runtime.closeNaturally();
    final closed = await client.closed;
    expect(closed.session.generation.toInt(), 1);
  });

  test(
    'rejects a command result from a superseded session generation',
    () async {
      final runtime = _SessionRuntime(
        executeResult: (_) => ResultEnvelope(
          originSession: EndpointSessionStamp(
            endpointId: 'studio',
            routeId: 'direct',
            generation: Int64(2),
          ),
          acknowledge: AcknowledgeResult(),
        ),
      );
      final client = await EndpointSessionClient.open(runtime, 'studio');

      await expectLater(
        client.execute(CommandEnvelope(terminalList: TerminalListCommand())),
        throwsA(
          isA<NativeSessionException>().having(
            (error) => error.message,
            'message',
            contains('did not match the active session'),
          ),
        ),
      );

      client.close();
      await client.closed;
    },
  );

  test(
    'drops application events from a superseded session generation',
    () async {
      final runtime = _SessionRuntime();
      final client = await EndpointSessionClient.open(runtime, 'studio');
      final observed = <String>[];
      final subscription = client.watchApplicationEvents().listen(
        (event) => observed.add(event.eventId),
      );

      runtime.emitApplicationEvent(
        application.EventEnvelope(
          eventId: 'stale',
          originSession: EndpointSessionStamp(
            endpointId: 'studio',
            routeId: 'direct',
            generation: Int64(2),
          ),
        ),
      );
      runtime.emitApplicationEvent(
        application.EventEnvelope(
          eventId: 'current',
          originSession: EndpointSessionStamp(
            endpointId: 'studio',
            routeId: 'direct',
            generation: Int64(1),
          ),
        ),
      );
      await Future<void>.delayed(Duration.zero);

      expect(observed, ['current']);
      await subscription.cancel();
      client.close();
      await client.closed;
    },
  );

  test('opens an isolated route test with the exact override', () async {
    final runtime = _SessionRuntime();
    final client = await EndpointSessionClient.open(
      runtime,
      'studio',
      routeOverride: 'ssh-office',
    );

    expect(runtime.lastOpenRequest?.endpointId, 'studio');
    expect(runtime.lastOpenRequest?.routeOverride, 'ssh-office');
    client.close();
    await client.closed;
  });

  test('retries a retryable unavailable open result', () async {
    final runtime = _SessionRuntime(openFailures: 1);

    final client = await openEndpointSessionWithRetry(
      runtime,
      'studio',
      initialRetryDelay: Duration.zero,
      maximumRetryDelay: Duration.zero,
    );

    expect(runtime.openCalls, 2);
    expect(runtime.released, [11, 12]);
    client.close();
    await client.closed;
  });

  test('does not retry a user-action open failure', () async {
    final runtime = _SessionRuntime(
      openFailures: 1,
      openFailureCode: ApiErrorCode.API_ERROR_CODE_DAEMON_BLOCKED,
    );

    await expectLater(
      openEndpointSessionWithRetry(
        runtime,
        'studio',
        initialRetryDelay: Duration.zero,
      ),
      throwsA(
        isA<NativeSessionException>().having(
          (error) => error.code,
          'code',
          ApiErrorCode.API_ERROR_CODE_DAEMON_BLOCKED,
        ),
      ),
    );
    expect(runtime.openCalls, 1);
  });

  test('executes typed file commands through the endpoint session', () async {
    final runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.fileList => ResultEnvelope(
          fileList: FileListResult(
            path: '/workspace',
            entries: [
              FileEntry(
                path: '/workspace/src',
                name: 'src',
                type: FileEntryType.FILE_ENTRY_TYPE_DIRECTORY,
              ),
            ],
          ),
        ),
        CommandEnvelope_Command.fileMkdir => ResultEnvelope(
          fileOperation: FileOperationResult(
            path: command.fileMkdir.path,
            success: true,
          ),
        ),
        _ => throw StateError('unexpected command'),
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');

    final page = await client.listFiles(path: '/workspace', limit: 5000);
    final created = await client.createDirectory('/workspace/build');

    expect(page.path, '/workspace');
    expect(page.entries.single.name, 'src');
    expect(created.path, '/workspace/build');
    expect(runtime.commands[0].fileList.limit, 1000);
    expect(runtime.commands[1].fileMkdir.recursive, isTrue);
    client.close();
    await client.closed;
  });

  test('surfaces daemon file operation errors', () async {
    final runtime = _SessionRuntime(
      executeResult: (_) => ResultEnvelope(
        fileOperation: FileOperationResult(
          success: false,
          errorCode: 'already_exists',
          errorMessage: 'Directory already exists',
        ),
      ),
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');

    await expectLater(
      client.createDirectory('/workspace/src'),
      throwsA(
        isA<NativeSessionException>().having(
          (error) => error.message,
          'message',
          'Directory already exists',
        ),
      ),
    );
    client.close();
    await client.closed;
  });

  test('downloads early resource frames and verifies the digest', () async {
    final bytes = <int>[1, 2, 3, 4, 5, 6];
    final hash = await const DartSha256().hash(bytes);
    late _SessionRuntime runtime;
    runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.fileDownloadOpen => ResultEnvelope(
          fileTransferOpen: FileTransferOpenResult(
            transfer: FileTransferHandle(
              resource: ResourceHandle(
                kind: ResourceKind.RESOURCE_KIND_FILE_TRANSFER,
                opaqueToken: [1],
              ),
              path: command.fileDownloadOpen.path,
              size: Int64(bytes.length),
              chunkBytes: 3,
              windowBytes: Int64(6),
            ),
          ),
        ),
        CommandEnvelope_Command.fileTransferCancel => ResultEnvelope(
          fileTransferCancel: FileTransferCancelResult(cancelled: true),
        ),
        _ => throw StateError('unexpected command'),
      },
      onResourceOpen: (runtime, handle) {
        runtime.emitResourceFrame(
          handle,
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
          wire.FileTransferData(
            offset: Int64.ZERO,
            data: bytes.sublist(0, 3),
          ).writeToBuffer(),
        );
        runtime.emitResourceFrame(
          handle,
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
          wire.FileTransferData(
            offset: Int64(3),
            data: bytes.sublist(3),
          ).writeToBuffer(),
        );
      },
      onResourceSend: (runtime, handle, frame) {
        final acknowledgement = wire.FileTransferAck.fromBuffer(frame.payload);
        expect(acknowledgement.offset.toInt(), bytes.length);
        expect(acknowledgement.windowBytes.toInt(), bytes.length);
        runtime.emitResourceFrame(
          handle,
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH,
          wire.FileTransferFinish(
            size: Int64(bytes.length),
            sha256: hash.bytes,
          ).writeToBuffer(),
        );
        runtime.emitResourceClosed(handle);
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');
    final directory = await Directory.systemTemp.createTemp('anytty-download-');
    final destination = File('${directory.path}/download.bin');

    final result = await EndpointFileTransfer(client).downloadToFile(
      remotePath: '/remote/download.bin',
      destination: destination,
    );

    expect(await destination.readAsBytes(), bytes);
    expect(result.sha256, hash.bytes);
    final acknowledgements = runtime.sentResourceFrames
        .where(
          (frame) =>
              frame.type ==
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
        )
        .map((frame) => wire.FileTransferAck.fromBuffer(frame.payload))
        .toList();
    expect(acknowledgements.map((ack) => ack.offset.toInt()), [6]);
    await Future<void>.delayed(Duration.zero);
    expect(runtime.released, contains(41));
    client.close();
    await client.closed;
    await directory.delete(recursive: true);
  });

  test('resumes a partial download with source fencing', () async {
    final bytes = List<int>.generate(12, (index) => index + 1);
    final hash = await const DartSha256().hash(bytes);
    late _SessionRuntime runtime;
    runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.fileDownloadOpen => () {
          expect(command.fileDownloadOpen.offset.toInt(), 6);
          expect(command.fileDownloadOpen.expectedSize.toInt(), bytes.length);
          expect(
            command.fileDownloadOpen.expectedModifiedAtUnixNano.toInt(),
            123,
          );
          return ResultEnvelope(
            fileTransferOpen: FileTransferOpenResult(
              transfer: FileTransferHandle(
                resource: ResourceHandle(
                  kind: ResourceKind.RESOURCE_KIND_FILE_TRANSFER,
                  opaqueToken: [3],
                ),
                path: command.fileDownloadOpen.path,
                offset: Int64(6),
                size: Int64(bytes.length),
                modifiedAtUnixNano: Int64(123),
                chunkBytes: 3,
                windowBytes: Int64(6),
              ),
            ),
          );
        }(),
        CommandEnvelope_Command.fileTransferCancel => ResultEnvelope(
          fileTransferCancel: FileTransferCancelResult(cancelled: true),
        ),
        _ => throw StateError('unexpected command'),
      },
      onResourceOpen: (runtime, handle) {
        for (var offset = 6; offset < bytes.length; offset += 3) {
          runtime.emitResourceFrame(
            handle,
            ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
            wire.FileTransferData(
              offset: Int64(offset),
              data: bytes.sublist(offset, offset + 3),
            ).writeToBuffer(),
          );
        }
      },
      onResourceSend: (runtime, handle, frame) {
        final acknowledgement = wire.FileTransferAck.fromBuffer(frame.payload);
        expect(acknowledgement.offset.toInt(), bytes.length);
        expect(acknowledgement.windowBytes.toInt(), 6);
        runtime.emitResourceFrame(
          handle,
          ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH,
          wire.FileTransferFinish(
            size: Int64(bytes.length),
            sha256: hash.bytes,
          ).writeToBuffer(),
        );
        runtime.emitResourceClosed(handle);
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');
    final directory = await Directory.systemTemp.createTemp('anytty-resume-');
    final destination = File('${directory.path}/download.bin');
    await File('${destination.path}.part').writeAsBytes(bytes.sublist(0, 6));

    await EndpointFileTransfer(client).downloadToFile(
      remotePath: '/remote/download.bin',
      destination: destination,
      resume: true,
      expectedSize: bytes.length,
      expectedModifiedAtUnixNano: 123,
    );

    expect(await destination.readAsBytes(), bytes);
    client.close();
    await client.closed;
    await directory.delete(recursive: true);
  });

  test(
    'releases an interrupted download while retaining its partial',
    () async {
      final cancellation = Completer<void>();
      late _SessionRuntime runtime;
      runtime = _SessionRuntime(
        executeResult: (command) => switch (command.whichCommand()) {
          CommandEnvelope_Command.fileDownloadOpen => ResultEnvelope(
            fileTransferOpen: FileTransferOpenResult(
              transfer: FileTransferHandle(
                resource: ResourceHandle(
                  kind: ResourceKind.RESOURCE_KIND_FILE_TRANSFER,
                  opaqueToken: [4],
                ),
                path: command.fileDownloadOpen.path,
                size: Int64(9),
                modifiedAtUnixNano: Int64(456),
                chunkBytes: 3,
                windowBytes: Int64(6),
              ),
            ),
          ),
          CommandEnvelope_Command.releaseResource => ResultEnvelope(
            acknowledge: AcknowledgeResult(),
          ),
          CommandEnvelope_Command.fileTransferCancel => ResultEnvelope(
            fileTransferCancel: FileTransferCancelResult(cancelled: true),
          ),
          _ => throw StateError('unexpected command'),
        },
        onResourceOpen: (runtime, handle) {
          runtime.emitResourceFrame(
            handle,
            ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
            wire.FileTransferData(
              offset: Int64.ZERO,
              data: [1, 2, 3],
            ).writeToBuffer(),
          );
        },
      );
      final client = await EndpointSessionClient.open(runtime, 'studio');
      final directory = await Directory.systemTemp.createTemp('anytty-pause-');
      final destination = File('${directory.path}/download.bin');

      await expectLater(
        EndpointFileTransfer(client).downloadToFile(
          remotePath: '/remote/download.bin',
          destination: destination,
          resume: true,
          cancelWhen: cancellation.future,
          preserveOnInterruption: () => true,
          onProgress: (transferred, _) {
            if (transferred == 3 && !cancellation.isCompleted) {
              cancellation.complete();
            }
          },
        ),
        throwsA(isA<BindingOperationCancelledException>()),
      );

      expect(await File('${destination.path}.part').readAsBytes(), [1, 2, 3]);
      expect(
        runtime.commands.map((command) => command.whichCommand()),
        contains(CommandEnvelope_Command.releaseResource),
      );
      expect(
        runtime.commands.map((command) => command.whichCommand()),
        isNot(contains(CommandEnvelope_Command.fileTransferCancel)),
      );
      client.close();
      await client.closed;
      await directory.delete(recursive: true);
    },
  );

  test('uploads chunks in order and verifies the completion digest', () async {
    final bytes = <int>[9, 8, 7, 6, 5, 4, 3];
    final hash = await const DartSha256().hash(bytes);
    late _SessionRuntime runtime;
    runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.fileUploadOpen => ResultEnvelope(
          fileTransferOpen: FileTransferOpenResult(
            transfer: FileTransferHandle(
              resource: ResourceHandle(
                kind: ResourceKind.RESOURCE_KIND_FILE_TRANSFER,
                opaqueToken: [2],
              ),
              path: command.fileUploadOpen.path,
              size: Int64(bytes.length),
              chunkBytes: 3,
              windowBytes: Int64(3),
            ),
          ),
        ),
        CommandEnvelope_Command.fileTransferCancel => ResultEnvelope(
          fileTransferCancel: FileTransferCancelResult(cancelled: true),
        ),
        _ => throw StateError('unexpected command'),
      },
      onResourceSend: (runtime, handle, frame) {
        switch (frame.type) {
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA:
            final data = wire.FileTransferData.fromBuffer(frame.payload);
            runtime.emitResourceFrame(
              handle,
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
              wire.FileTransferAck(
                offset: data.offset + data.data.length,
                windowBytes: Int64(data.data.length),
              ).writeToBuffer(),
            );
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH:
            runtime.emitResourceFrame(
              handle,
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT,
              wire.FileTransferResult(
                path: '/remote/upload.bin',
                size: Int64(bytes.length),
                sha256: hash.bytes,
              ).writeToBuffer(),
            );
            runtime.emitResourceClosed(handle);
          default:
            throw StateError('unexpected resource frame');
        }
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');
    final directory = await Directory.systemTemp.createTemp('anytty-upload-');
    final source = File('${directory.path}/upload.bin');
    await source.writeAsBytes(bytes);

    final result = await EndpointFileTransfer(client)
        .uploadFile(source: source, remotePath: '/remote/upload.bin');

    final chunks = runtime.sentResourceFrames
        .where(
          (frame) =>
              frame.type ==
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
        )
        .map((frame) => wire.FileTransferData.fromBuffer(frame.payload))
        .toList();
    expect(chunks.map((chunk) => chunk.offset.toInt()), [0, 3, 6]);
    expect(chunks.expand((chunk) => chunk.data), bytes);
    expect(result.sha256, hash.bytes);
    await Future<void>.delayed(Duration.zero);
    expect(runtime.released, contains(41));
    client.close();
    await client.closed;
    await directory.delete(recursive: true);
  });

  test('resumes an upload from its opaque daemon handle', () async {
    final bytes = <int>[9, 8, 7, 6, 5, 4, 3, 2, 1];
    final hash = await const DartSha256().hash(bytes);
    FileUploadResumeHandle? openedResume;
    late _SessionRuntime runtime;
    runtime = _SessionRuntime(
      executeResult: (command) => switch (command.whichCommand()) {
        CommandEnvelope_Command.fileUploadOpen => () {
          expect(command.fileUploadOpen.resume.opaqueToken, [7, 7]);
          return ResultEnvelope(
            fileTransferOpen: FileTransferOpenResult(
              transfer: FileTransferHandle(
                resource: ResourceHandle(
                  kind: ResourceKind.RESOURCE_KIND_FILE_TRANSFER,
                  opaqueToken: [5],
                ),
                path: command.fileUploadOpen.path,
                offset: Int64(3),
                size: Int64(bytes.length),
                resume: FileUploadResumeHandle(opaqueToken: [8, 8]),
                chunkBytes: 3,
                windowBytes: Int64(3),
              ),
            ),
          );
        }(),
        CommandEnvelope_Command.fileTransferCancel => ResultEnvelope(
          fileTransferCancel: FileTransferCancelResult(cancelled: true),
        ),
        _ => throw StateError('unexpected command'),
      },
      onResourceSend: (runtime, handle, frame) {
        switch (frame.type) {
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA:
            final data = wire.FileTransferData.fromBuffer(frame.payload);
            runtime.emitResourceFrame(
              handle,
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_ACK,
              wire.FileTransferAck(
                offset: data.offset + data.data.length,
                windowBytes: Int64(data.data.length),
              ).writeToBuffer(),
            );
          case ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_FINISH:
            runtime.emitResourceFrame(
              handle,
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_RESULT,
              wire.FileTransferResult(
                path: '/remote/upload.bin',
                size: Int64(bytes.length),
                sha256: hash.bytes,
              ).writeToBuffer(),
            );
            runtime.emitResourceClosed(handle);
          default:
            throw StateError('unexpected resource frame');
        }
      },
    );
    final client = await EndpointSessionClient.open(runtime, 'studio');
    final directory = await Directory.systemTemp.createTemp(
      'anytty-upload-resume-',
    );
    final source = File('${directory.path}/upload.bin');
    await source.writeAsBytes(bytes);

    await EndpointFileTransfer(client).uploadFile(
      source: source,
      remotePath: '/remote/upload.bin',
      resume: FileUploadResumeHandle(opaqueToken: [7, 7]),
      onOpened: (transfer) => openedResume = transfer.resume.deepCopy(),
    );

    final chunks = runtime.sentResourceFrames
        .where(
          (frame) =>
              frame.type ==
              ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_FILE_DATA,
        )
        .map((frame) => wire.FileTransferData.fromBuffer(frame.payload))
        .toList();
    expect(chunks.map((chunk) => chunk.offset.toInt()), [3, 6]);
    expect(chunks.expand((chunk) => chunk.data), bytes.sublist(3));
    expect(openedResume?.opaqueToken, [8, 8]);
    client.close();
    await client.closed;
    await directory.delete(recursive: true);
  });
}

HistoryRow _alternateHistoryRow(int lineId, int screenRow, String text) {
  return HistoryRow(
    logicalLineId: Int64(lineId),
    rowKind: 'alt-screen-frame',
    segment: HistoryCursorSegment.HISTORY_CURSOR_SEGMENT_CURRENT_ALT_FRAME,
    fixedGrid: true,
    screenCols: 12,
    screenRows: screenRow,
    screenRowSet: true,
    row: ScreenRow(
      cells: [ScreenCell(content: text, width: text.length)],
    ),
  );
}

typedef _ResourceOpenHandler = void Function(
  _SessionRuntime runtime,
  int handle,
);
typedef _ResourceSendHandler = void Function(
  _SessionRuntime runtime,
  int handle,
  ResourceStreamFrame frame,
);

final class _SessionRuntime
    implements AnyttyEngineRuntime, AnyttyResourceStreamRuntime {
  _SessionRuntime({
    this.openFailures = 0,
    this.openFailureCode = ApiErrorCode.API_ERROR_CODE_UNAVAILABLE,
    this.executeResult,
    this.onResourceOpen,
    this.onResourceSend,
  });

  final StreamController<EventEnvelope> _events =
      StreamController<EventEnvelope>.broadcast();
  final List<int> released = [];
  final List<int> closedSessions = [];
  final int openFailures;
  final ApiErrorCode openFailureCode;
  final ResultEnvelope Function(CommandEnvelope command)? executeResult;
  final _ResourceOpenHandler? onResourceOpen;
  final _ResourceSendHandler? onResourceSend;
  final List<CommandEnvelope> commands = [];
  final List<ResourceStreamFrame> sentResourceFrames = [];
  final List<int> closedResourceStreams = [];
  int openCalls = 0;
  int executeCalls = 0;
  bool _sessionClosed = false;
  OpenSessionRequest? lastOpenRequest;

  EndpointSessionStamp get _activeStamp {
    final route = lastOpenRequest?.routeOverride.trim() ?? '';
    return EndpointSessionStamp(
      endpointId: lastOpenRequest?.endpointId ?? 'studio',
      routeId: route.isEmpty ? 'direct' : route,
      generation: Int64(1),
    );
  }

  @override
  Stream<EventEnvelope> get events => _events.stream;

  @override
  Stream<int> get foregroundResumes => const Stream<int>.empty();

  @override
  int openSession(OpenSessionRequest request) {
    lastOpenRequest = request.deepCopy();
    openCalls += 1;
    final operationHandle = 10 + openCalls;
    scheduleMicrotask(() {
      if (openCalls <= openFailures) {
        _events.add(
          EventEnvelope(
            openSession: OpenSessionResult(
              requestId: request.requestId,
              operationHandle: Int64(operationHandle),
              error: ApiError(
                code: openFailureCode,
                message: 'session unavailable',
                retryable: true,
                attempted: true,
              ),
            ),
          ),
        );
        return;
      }
      _events.add(
        EventEnvelope(
          openSession: OpenSessionResult(
            requestId: request.requestId,
            operationHandle: Int64(operationHandle),
            sessionHandle: Int64(21),
            session: _activeStamp,
          ),
        ),
      );
    });
    return operationHandle;
  }

  void closeNaturally() => _emitClosed();

  void emitSessionClosed(EndpointSessionStamp stamp) {
    _events.add(
      EventEnvelope(
        sessionClosed: SessionClosedEvent(
          sessionHandle: Int64(21),
          session: stamp,
        ),
      ),
    );
  }

  @override
  void closeSession(int sessionHandle) {
    closedSessions.add(sessionHandle);
    if (_sessionClosed) throw StateError('session already closed');
    _emitClosed();
  }

  void _emitClosed() {
    if (_sessionClosed) return;
    _sessionClosed = true;
    scheduleMicrotask(() {
      _events.add(
        EventEnvelope(
          sessionClosed: SessionClosedEvent(
            sessionHandle: Int64(21),
            session: _activeStamp,
          ),
        ),
      );
    });
  }

  @override
  void release(int handle) => released.add(handle);

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) =>
      EndpointDemandLease(() {});

  @override
  void cancel(int operationHandle) => throw UnimplementedError();

  @override
  int command(EngineCommand command) => throw UnimplementedError();

  @override
  int execute(int sessionHandle, CommandEnvelope request) {
    final handler = executeResult;
    if (handler == null) throw UnimplementedError();
    executeCalls += 1;
    final operationHandle = 100 + executeCalls;
    commands.add(request.deepCopy());
    scheduleMicrotask(() {
      final result = handler(request).deepCopy();
      if (!result.hasOriginSession()) result.originSession = _activeStamp;
      _events.add(
        EventEnvelope(
          execute: ExecuteResult(
            operationHandle: Int64(operationHandle),
            sessionHandle: Int64(sessionHandle),
            result: result,
          ),
        ),
      );
    });
    return operationHandle;
  }

  void emitApplicationEvent(application.EventEnvelope event) {
    _events.add(
      EventEnvelope(
        application: ApplicationEvent(sessionHandle: Int64(21), event: event),
      ),
    );
  }

  @override
  int openResourceStream(int sessionHandle, OpenResourceStreamRequest request) {
    const handle = 41;
    onResourceOpen?.call(this, handle);
    return handle;
  }

  @override
  void sendResourceStreamFrame(int streamHandle, ResourceStreamFrame frame) {
    sentResourceFrames.add(frame.deepCopy());
    onResourceSend?.call(this, streamHandle, frame.deepCopy());
  }

  @override
  void closeResourceStream(int streamHandle) {
    closedResourceStreams.add(streamHandle);
    emitResourceClosed(streamHandle);
  }

  void emitResourceFrame(
    int streamHandle,
    ResourceStreamFrameType type,
    List<int> payload,
  ) {
    _events.add(
      EventEnvelope(
        resourceStreamFrame: ResourceStreamFrame(
          streamHandle: Int64(streamHandle),
          type: type,
          payload: payload,
        ),
      ),
    );
  }

  void emitResourceClosed(int streamHandle) {
    _events.add(
      EventEnvelope(
        resourceStreamClosed: ResourceStreamClosedEvent(
          streamHandle: Int64(streamHandle),
        ),
      ),
    );
  }
}
