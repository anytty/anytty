import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:fixnum/fixnum.dart';

import '../../../generated/proto/apipb/application.pb.dart'
    hide EventEnvelope, EventEnvelope_Event;
import '../../../generated/proto/apipb/application.pb.dart' as application;
import '../../../generated/proto/apipb/common.pb.dart';
import '../../../generated/proto/apipb/file.pb.dart';
import '../../../generated/proto/apipb/history.pb.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../native/anytty_resource_stream.dart';
import '../../../native/anytty_runtime.dart';
import '../../../native/binding_operation.dart';
import '../../../native/request_id.dart';
import '../../../native/runtime_diagnostics.dart';
import '../../../native/terminal_input_encoder.dart';
import '../../files/domain/file_preview_safety.dart';
import '../../browser/data/browser_http_proxy.dart';
import '../domain/bounded_serial_operation_queue.dart';
import '../domain/history_store.dart';
import '../domain/live_screen_store.dart';
import '../domain/resize_control.dart';
import '../domain/terminal_inventory.dart';
import '../domain/terminal_metrics.dart';

final class NativeSessionException implements Exception {
  const NativeSessionException(
    this.message, {
    this.code,
    this.retryable = false,
    this.attempted = false,
  });

  final String message;
  final ApiErrorCode? code;
  final bool retryable;
  final bool attempted;

  @override
  String toString() => message;
}

enum TerminalDeliveryState { awaitingFrame, ready, recovering, stalled }

bool endpointSessionStampsEqual(
  EndpointSessionStamp left,
  EndpointSessionStamp right,
) {
  return left.endpointId == right.endpointId &&
      left.routeId == right.routeId &&
      left.generation == right.generation;
}

bool _isCompleteEndpointSessionStamp(EndpointSessionStamp stamp) {
  return stamp.endpointId.trim().isNotEmpty &&
      stamp.routeId.trim().isNotEmpty &&
      stamp.generation != Int64.ZERO;
}

final class EndpointSessionClient implements BrowserProxySession {
  EndpointSessionClient._(
    this._runtime,
    this.sessionHandle,
    EndpointSessionStamp stamp,
    this._sessionEvents,
    this._closedCompleter,
  ) : stamp = stamp.deepCopy() {
    unawaited(_closedCompleter.future.then((_) => _releaseClosedSession()));
  }

  final AnyttyEngineRuntime _runtime;
  final int sessionHandle;
  final EndpointSessionStamp stamp;
  final StreamSubscription<EventEnvelope> _sessionEvents;
  final Completer<SessionClosedEvent> _closedCompleter;
  bool _closeRequested = false;
  bool _sessionReleased = false;

  Future<SessionClosedEvent> get closed => _closedCompleter.future;

  bool get isClosed => _closedCompleter.isCompleted;

  void recordRenderLatency(Duration latency) {
    final runtime = _runtime;
    if (runtime is RuntimeDiagnosticsSink) {
      (runtime as RuntimeDiagnosticsSink).recordRenderLatency(
        session: stamp,
        latency: latency,
      );
    }
  }

  static Future<EndpointSessionClient> open(
    AnyttyEngineRuntime runtime,
    String endpointId, {
    String routeOverride = '',
    Future<void>? cancelWhen,
  }) async {
    final closedCompleter = Completer<SessionClosedEvent>();
    final earlyClosures = <int, List<SessionClosedEvent>>{};
    int? sessionHandle;
    EndpointSessionStamp? openedStamp;

    void acceptClosure(SessionClosedEvent event) {
      final eventHandle = event.sessionHandle.toInt();
      if (sessionHandle == null) {
        final closures = earlyClosures.putIfAbsent(eventHandle, () => []);
        if (closures.length < 4) closures.add(event.deepCopy());
      } else if (eventHandle == sessionHandle &&
          openedStamp != null &&
          event.hasSession() &&
          endpointSessionStampsEqual(event.session, openedStamp) &&
          !closedCompleter.isCompleted) {
        closedCompleter.complete(event.deepCopy());
      }
    }

    final sessionEvents = runtime.events.listen(
      (event) {
        if (event.whichEvent() == EventEnvelope_Event.sessionClosed) {
          acceptClosure(event.sessionClosed);
        }
      },
      onDone: () {
        if (sessionHandle != null && !closedCompleter.isCompleted) {
          closedCompleter.complete(
            SessionClosedEvent(sessionHandle: Int64(sessionHandle)),
          );
        }
      },
    );
    try {
      final result = await runBindingOperation<OpenSessionResult>(
        runtime: runtime,
        begin: () => runtime.openSession(
          OpenSessionRequest(
            requestId: newRequestId(),
            endpointId: endpointId,
            routeOverride: routeOverride.trim(),
            intent: ConnectIntent.CONNECT_INTENT_INTERACTIVE,
          ),
        ),
        select: (event) => event.whichEvent() == EventEnvelope_Event.openSession
            ? event.openSession.deepCopy()
            : null,
        operationHandle: (value) => value.operationHandle.toInt(),
        timeoutMessage: 'Endpoint session timed out',
        timeout: const Duration(seconds: 45),
        cancelWhen: cancelWhen,
      );
      _throwBindingError(result.hasError() ? result.error : null);
      if (!result.hasSession() ||
          result.sessionHandle.toInt() == 0 ||
          !_isCompleteEndpointSessionStamp(result.session) ||
          result.session.endpointId != endpointId) {
        throw const NativeSessionException(
          'Endpoint session response was incomplete',
        );
      }
      sessionHandle = result.sessionHandle.toInt();
      openedStamp = result.session.deepCopy();
      for (final earlyClosure in earlyClosures[sessionHandle] ?? const []) {
        acceptClosure(earlyClosure);
        if (closedCompleter.isCompleted) break;
      }
      return EndpointSessionClient._(
        runtime,
        sessionHandle,
        result.session,
        sessionEvents,
        closedCompleter,
      );
    } catch (_) {
      await sessionEvents.cancel();
      rethrow;
    }
  }

  Future<List<TerminalInfo>> listTerminals() async {
    final result = await execute(
      CommandEnvelope(terminalList: TerminalListCommand()),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalList) {
      throw const NativeSessionException(
        'Terminal list response was incomplete',
      );
    }
    return result.terminalList.terminals
        .map((terminal) => terminal.deepCopy())
        .toList(growable: false);
  }

  Future<FileListResult> listFiles({
    required String path,
    String cursor = '',
    int limit = 500,
  }) async {
    final result = await execute(
      CommandEnvelope(
        fileList: FileListCommand(
          path: path,
          cursor: cursor,
          limit: limit.clamp(1, 1000),
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.fileList) {
      throw const NativeSessionException('File list response was incomplete');
    }
    return result.fileList.deepCopy();
  }

  Future<FileEntry> statFile(String path) async {
    final result = await execute(
      CommandEnvelope(fileStat: FileStatCommand(path: path)),
    );
    if (result.whichResult() != ResultEnvelope_Result.fileStat ||
        !result.fileStat.hasEntry()) {
      throw const NativeSessionException('File stat response was incomplete');
    }
    return result.fileStat.entry.deepCopy();
  }

  Future<FilePreviewResult> previewFile(
    String path, {
    int maxBytes = maximumFilePreviewBytes,
  }) async {
    final requestedBytes = maxBytes.clamp(1, maximumFilePreviewBytes);
    final result = await execute(
      CommandEnvelope(
        filePreview: FilePreviewCommand(
          path: path,
          maxBytes: Int64(requestedBytes),
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.filePreview ||
        !result.filePreview.hasEntry()) {
      throw const NativeSessionException(
        'File preview response was incomplete',
      );
    }
    final preview = result.filePreview.deepCopy();
    try {
      await validateFilePreview(preview, requestedBytes: requestedBytes);
    } on FilePreviewSafetyException catch (error) {
      throw NativeSessionException(error.message);
    }
    return preview;
  }

  Future<FileOperationResult> createDirectory(String path) =>
      _executeFileOperation(
        CommandEnvelope(
          fileMkdir: FileMkdirCommand(path: path, recursive: true),
        ),
        'Create directory',
      );

  Future<FileOperationResult> renameFile(String path, String newPath) =>
      _executeFileOperation(
        CommandEnvelope(
          fileRename: FileRenameCommand(path: path, newPath: newPath),
        ),
        'Rename file',
      );

  Future<FileOperationResult> deleteFile(String path) => _executeFileOperation(
    CommandEnvelope(fileDelete: FileDeleteCommand(path: path, recursive: true)),
    'Delete file',
  );

  Future<FileBatchResult> copyFiles(
    List<String> paths,
    String targetDirectory,
  ) => _executeFileBatch(
    CommandEnvelope(
      fileCopy: FileCopyCommand(paths: paths, targetDirectory: targetDirectory),
    ),
    'Copy files',
  );

  Future<FileBatchResult> moveFiles(
    List<String> paths,
    String targetDirectory,
  ) => _executeFileBatch(
    CommandEnvelope(
      fileMove: FileMoveCommand(paths: paths, targetDirectory: targetDirectory),
    ),
    'Move files',
  );

  Future<FileTransferHandle> openFileDownload(
    String path, {
    int offset = 0,
    int expectedSize = 0,
    int expectedModifiedAtUnixNano = 0,
  }) async {
    final result = await execute(
      CommandEnvelope(
        fileDownloadOpen: FileDownloadOpenCommand(
          path: path,
          offset: Int64(offset),
          expectedSize: Int64(expectedSize),
          expectedModifiedAtUnixNano: Int64(expectedModifiedAtUnixNano),
        ),
      ),
    );
    return _requireFileTransfer(result, 'Download');
  }

  Future<FileTransferHandle> openFileUpload({
    required String path,
    required int size,
    bool overwrite = false,
    FileUploadResumeHandle? resume,
  }) async {
    final result = await execute(
      CommandEnvelope(
        fileUploadOpen: FileUploadOpenCommand(
          path: path,
          size: Int64(size),
          overwrite: overwrite,
          resume: resume,
        ),
      ),
    );
    return _requireFileTransfer(result, 'Upload');
  }

  Future<AnyttyResourceStream> openFileResourceStream(
    FileTransferHandle transfer,
  ) {
    if (!transfer.hasResource()) {
      throw const NativeSessionException(
        'File transfer resource was incomplete',
      );
    }
    return AnyttyResourceStream.open(
      runtime: _runtime,
      sessionHandle: sessionHandle,
      request: OpenResourceStreamRequest(
        resource: transfer.resource,
        initialUploadOffset: transfer.offset,
      ),
    );
  }

  @override
  Future<ResourceHandle> openBrowserProxy({
    required String host,
    required int port,
  }) async {
    final result = await execute(
      CommandEnvelope(
        browserProxyOpen: BrowserProxyOpenCommand(host: host, port: port),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.browserProxyOpen ||
        !result.browserProxyOpen.hasResource()) {
      throw const NativeSessionException(
        'Browser proxy response was incomplete',
      );
    }
    return result.browserProxyOpen.resource.deepCopy();
  }

  @override
  Future<AnyttyResourceStream> openBrowserResourceStream(
    ResourceHandle resource,
  ) {
    if (resource.kind != ResourceKind.RESOURCE_KIND_BROWSER_PROXY) {
      throw const NativeSessionException('Browser proxy resource was invalid');
    }
    return AnyttyResourceStream.open(
      runtime: _runtime,
      sessionHandle: sessionHandle,
      request: OpenResourceStreamRequest(resource: resource),
    );
  }

  Future<void> cancelFileTransfer(FileTransferHandle transfer) async {
    if (_closeRequested || isClosed) return;
    final result = await execute(
      CommandEnvelope(
        fileTransferCancel: FileTransferCancelCommand(
          transfer: transfer.hasResource() ? transfer.resource : null,
          uploadResume: !transfer.hasResource() && transfer.hasResume()
              ? transfer.resume
              : null,
        ),
      ),
      timeout: const Duration(seconds: 2),
    );
    if (result.whichResult() != ResultEnvelope_Result.fileTransferCancel ||
        !result.fileTransferCancel.cancelled) {
      throw const NativeSessionException(
        'File transfer cancellation was not acknowledged',
      );
    }
  }

  Future<void> releaseFileTransfer(FileTransferHandle transfer) async {
    if (_closeRequested || isClosed || !transfer.hasResource()) return;
    final result = await execute(
      CommandEnvelope(
        releaseResource: ReleaseResourceCommand(resource: transfer.resource),
      ),
      timeout: const Duration(seconds: 2),
    );
    if (result.whichResult() != ResultEnvelope_Result.acknowledge) {
      throw const NativeSessionException(
        'File transfer resource release was not acknowledged',
      );
    }
  }

  FileTransferHandle _requireFileTransfer(
    ResultEnvelope result,
    String operation,
  ) {
    if (result.whichResult() != ResultEnvelope_Result.fileTransferOpen ||
        !result.fileTransferOpen.hasTransfer() ||
        !result.fileTransferOpen.transfer.hasResource() ||
        result.fileTransferOpen.transfer.path.trim().isEmpty) {
      throw NativeSessionException('$operation response was incomplete');
    }
    final transfer = result.fileTransferOpen.transfer.deepCopy();
    final offset = transfer.offset.toInt();
    final size = transfer.size.toInt();
    if (size < 0 || offset < 0 || offset > size) {
      throw NativeSessionException('$operation metadata was invalid');
    }
    return transfer;
  }

  Future<FileOperationResult> _executeFileOperation(
    CommandEnvelope command,
    String operation,
  ) async {
    final result = await execute(command);
    if (result.whichResult() != ResultEnvelope_Result.fileOperation) {
      throw NativeSessionException('$operation response was incomplete');
    }
    final fileResult = result.fileOperation.deepCopy();
    if (!fileResult.success) {
      throw NativeSessionException(
        fileResult.errorMessage.isEmpty
            ? '$operation failed${fileResult.errorCode.isEmpty ? '' : ': ${fileResult.errorCode}'}'
            : fileResult.errorMessage,
      );
    }
    return fileResult;
  }

  Future<FileBatchResult> _executeFileBatch(
    CommandEnvelope command,
    String operation,
  ) async {
    final result = await execute(command);
    if (result.whichResult() != ResultEnvelope_Result.fileBatch) {
      throw NativeSessionException('$operation response was incomplete');
    }
    final batch = result.fileBatch.deepCopy();
    final failed = batch.results.where((item) => !item.success).firstOrNull;
    if (failed != null) {
      throw NativeSessionException(
        failed.errorMessage.isEmpty
            ? '$operation failed${failed.errorCode.isEmpty ? '' : ': ${failed.errorCode}'}'
            : failed.errorMessage,
      );
    }
    return batch;
  }

  Future<TerminalInfo> getTerminal(TerminalRef terminal) async {
    final result = await execute(
      CommandEnvelope(terminalGet: TerminalGetCommand(terminal: terminal)),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalGet ||
        !result.terminalGet.hasTerminal() ||
        !result.terminalGet.terminal.hasRef()) {
      throw const NativeSessionException(
        'Terminal get response was incomplete',
      );
    }
    return result.terminalGet.terminal.deepCopy();
  }

  Future<TerminalDefaults> terminalDefaults() async {
    final result = await execute(
      CommandEnvelope(terminalDefaults: TerminalDefaultsCommand()),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalDefaults ||
        !result.terminalDefaults.hasDefaults()) {
      throw const NativeSessionException(
        'Terminal defaults response was incomplete',
      );
    }
    return result.terminalDefaults.defaults.deepCopy();
  }

  Future<PathListDirectoriesResult> listDirectories({
    required String prefix,
    int limit = 100,
  }) async {
    final result = await execute(
      CommandEnvelope(
        pathListDirectories: PathListDirectoriesCommand(
          prefix: prefix,
          limit: limit.clamp(1, 100),
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.pathListDirectories) {
      throw const NativeSessionException(
        'Directory list response was incomplete',
      );
    }
    return result.pathListDirectories.deepCopy();
  }

  Future<TerminalInfo> createTerminal({
    required String name,
    required List<String> command,
    required String cwd,
    required List<String> environment,
    required int cols,
    required int rows,
    required String sizeLockMode,
  }) async {
    final result = await execute(
      CommandEnvelope(
        terminalCreate: TerminalCreateCommand(
          terminal: TerminalCreateSpec(
            terminalId: 'term-${newRequestId()}',
            name: name,
            command: command,
            size: TerminalSize(
              cols: cols.clamp(20, 500),
              rows: rows.clamp(4, 300),
            ),
            tags: <String, String>{
              if (cwd.trim().isNotEmpty) 'cwd': cwd.trim(),
              'anytty.size_lock': sizeLockMode,
            }.entries,
            cwd: cwd,
            env: environment,
          ),
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalCreate ||
        !result.terminalCreate.hasTerminal() ||
        !result.terminalCreate.terminal.hasRef() ||
        result.terminalCreate.terminal.ref.terminalId.isEmpty) {
      throw const NativeSessionException(
        'Terminal create response was incomplete',
      );
    }
    return result.terminalCreate.terminal.deepCopy();
  }

  Future<void> renameTerminal(TerminalInfo terminal, String name) async {
    final result = await execute(
      CommandEnvelope(
        terminalSetMetadata: TerminalSetMetadataCommand(
          terminal: terminal.ref,
          name: name,
          tags: terminal.tags.entries,
        ),
      ),
    );
    _requireAcknowledge(result, 'Terminal rename');
  }

  Future<void> restartTerminal(TerminalRef terminal) async {
    final result = await execute(
      CommandEnvelope(
        terminalRestart: TerminalRestartCommand(terminal: terminal),
      ),
    );
    _requireAcknowledge(result, 'Terminal restart');
  }

  Future<void> killTerminal(TerminalRef terminal) async {
    final result = await execute(
      CommandEnvelope(terminalKill: TerminalKillCommand(terminal: terminal)),
    );
    _requireAcknowledge(result, 'Terminal end');
  }

  Future<void> removeTerminal(TerminalRef terminal) async {
    final result = await execute(
      CommandEnvelope(
        terminalRemove: TerminalRemoveCommand(terminal: terminal),
      ),
    );
    _requireAcknowledge(result, 'Terminal remove');
  }

  Stream<application.EventEnvelope> watchApplicationEvents() {
    return _runtime.events
        .where(
          (event) =>
              event.whichEvent() == EventEnvelope_Event.application &&
              event.application.sessionHandle.toInt() == sessionHandle &&
              event.application.hasEvent() &&
              event.application.event.hasOriginSession() &&
              endpointSessionStampsEqual(
                event.application.event.originSession,
                stamp,
              ),
        )
        .map((event) => event.application.event.deepCopy());
  }

  Stream<int> watchForegroundResumes() => _runtime.foregroundResumes;

  Future<TerminalAttachResult> attach(TerminalRef terminal) async {
    final result = await execute(
      CommandEnvelope(
        terminalAttach: TerminalAttachCommand(
          terminal: terminal,
          mode: AttachmentMode.ATTACHMENT_MODE_COLLABORATOR,
          resizePolicy: ResizePolicy.RESIZE_POLICY_OWNER,
          surfaceId: 'flutter-${newRequestId()}',
          viewId: newRequestId(),
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalAttach ||
        !result.terminalAttach.hasAttachment() ||
        !result.terminalAttach.attachment.hasResource()) {
      throw const NativeSessionException(
        'Terminal attachment response was incomplete',
      );
    }
    return result.terminalAttach.deepCopy();
  }

  Future<TerminalResizeResult> resizeTerminal({
    required AttachmentHandle attachment,
    required TerminalSize size,
    required ResizePolicy policy,
    required bool takeOwnership,
    required Int64 expectedOwnerEpoch,
  }) async {
    final result = await execute(
      CommandEnvelope(
        terminalResize: TerminalResizeCommand(
          attachment: attachment.resource,
          size: size,
          resizePolicy: policy,
          takeOwnership: takeOwnership,
          expectedOwnerEpoch: expectedOwnerEpoch,
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalResize ||
        !result.terminalResize.hasSize() ||
        !result.terminalResize.hasResizeControl()) {
      throw const NativeSessionException(
        'Terminal resize response was incomplete',
      );
    }
    return result.terminalResize.deepCopy();
  }

  Future<TerminalResizeResult> setTerminalResizeLock({
    required AttachmentHandle attachment,
    required bool locked,
  }) async {
    final result = await execute(
      CommandEnvelope(
        terminalResizeLock: TerminalResizeLockCommand(
          attachment: attachment.resource,
          locked: locked,
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.terminalResize ||
        !result.terminalResize.hasSize() ||
        !result.terminalResize.hasResizeControl()) {
      throw const NativeSessionException(
        'Terminal resize lock response was incomplete',
      );
    }
    return result.terminalResize.deepCopy();
  }

  Future<NativeScreenResult> nextScreen(
    TerminalRef terminal,
    int observedRevision, {
    Future<void>? cancelWhen,
  }) async {
    final result = await execute(
      CommandEnvelope(
        liveScreenNext: LiveScreenNextCommand(
          terminal: terminal,
          observedRevision: Int64(observedRevision),
        ),
      ),
      timeout: const Duration(seconds: 45),
      cancelWhen: cancelWhen,
    );
    if (result.whichResult() != ResultEnvelope_Result.liveScreen) {
      throw const NativeSessionException('Live screen response was incomplete');
    }
    return result.liveScreen.deepCopy();
  }

  Future<HistoryWindowResult> historyWindow(
    HistoryWindowCommand command,
  ) async {
    final result = await execute(CommandEnvelope(historyWindow: command));
    if (result.whichResult() != ResultEnvelope_Result.historyWindow) {
      throw const NativeSessionException(
        'History window response was incomplete',
      );
    }
    return result.historyWindow.deepCopy();
  }

  Future<HistoryCopyResult> historyCopy(HistoryCopyCommand command) async {
    final result = await execute(CommandEnvelope(historyCopy: command));
    if (result.whichResult() != ResultEnvelope_Result.historyCopy) {
      throw const NativeSessionException(
        'History copy response was incomplete',
      );
    }
    return result.historyCopy.deepCopy();
  }

  Future<HistorySearchResult> historySearch(
    HistorySearchCommand command,
  ) async {
    final result = await execute(CommandEnvelope(historySearch: command));
    if (result.whichResult() != ResultEnvelope_Result.historySearch) {
      throw const NativeSessionException(
        'History search response was incomplete',
      );
    }
    return result.historySearch.deepCopy();
  }

  Future<void> releaseHistory({
    required TerminalRef terminal,
    required String token,
    required Int64 generation,
  }) async {
    if (_closeRequested ||
        isClosed ||
        token.isEmpty ||
        generation == Int64.ZERO) {
      return;
    }
    final result = await execute(
      CommandEnvelope(
        historyRelease: HistoryReleaseCommand(
          terminal: terminal,
          token: token,
          historyGeneration: generation,
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.acknowledge) {
      throw const NativeSessionException(
        'History release was not acknowledged',
      );
    }
  }

  Future<void> sendInput(AttachmentHandle attachment, Uint8List bytes) async {
    if (bytes.isEmpty) return;
    final result = await execute(
      CommandEnvelope(
        terminalInput: TerminalInputCommand(
          attachment: attachment.resource,
          data: bytes,
        ),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.acknowledge) {
      throw const NativeSessionException('Terminal input was not acknowledged');
    }
  }

  Future<void> detach(AttachmentHandle attachment) async {
    if (_closeRequested || isClosed) return;
    final result = await execute(
      CommandEnvelope(
        terminalDetach: TerminalDetachCommand(attachment: attachment.resource),
      ),
    );
    if (result.whichResult() != ResultEnvelope_Result.acknowledge) {
      throw const NativeSessionException(
        'Terminal detach was not acknowledged',
      );
    }
  }

  Future<ResultEnvelope> execute(
    CommandEnvelope command, {
    Duration timeout = const Duration(seconds: 30),
    Future<void>? cancelWhen,
  }) async {
    if (_closeRequested || isClosed) {
      throw StateError('Endpoint session is closed');
    }
    final result = await runBindingOperation<ExecuteResult>(
      runtime: _runtime,
      begin: () => _runtime.execute(sessionHandle, command),
      select: (event) => event.whichEvent() == EventEnvelope_Event.execute
          ? event.execute.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Endpoint command timed out',
      timeout: timeout,
      cancelWhen: cancelWhen,
    );
    _throwBindingError(result.hasError() ? result.error : null);
    if (result.sessionHandle.toInt() != sessionHandle ||
        !result.hasResult() ||
        !result.result.hasOriginSession() ||
        !endpointSessionStampsEqual(result.result.originSession, stamp)) {
      throw const NativeSessionException(
        'Endpoint command response did not match the active session',
      );
    }
    if (result.result.whichResult() == ResultEnvelope_Result.error) {
      _throwBindingError(result.result.error);
    }
    return result.result.deepCopy();
  }

  void close() {
    if (_closeRequested) return;
    _closeRequested = true;
    try {
      _runtime.closeSession(sessionHandle);
    } catch (_) {
      // A naturally closed generation may already have been released.
      if (!_closedCompleter.isCompleted) {
        _closedCompleter.complete(
          SessionClosedEvent(sessionHandle: Int64(sessionHandle)),
        );
      }
    }
  }

  Future<void> _releaseClosedSession() async {
    if (_sessionReleased) return;
    _sessionReleased = true;
    try {
      _runtime.release(sessionHandle);
    } catch (_) {
      // Runtime shutdown or a stale renderer can revoke the handle first.
    } finally {
      await _sessionEvents.cancel();
    }
  }

  void _requireAcknowledge(ResultEnvelope result, String operation) {
    if (result.whichResult() != ResultEnvelope_Result.acknowledge) {
      throw NativeSessionException('$operation was not acknowledged');
    }
  }
}

Future<EndpointSessionClient> openEndpointSessionWithRetry(
  AnyttyEngineRuntime runtime,
  String endpointId, {
  Future<void>? cancelWhen,
  Duration initialRetryDelay = const Duration(seconds: 1),
  Duration maximumRetryDelay = const Duration(seconds: 8),
  Duration maximumRetryDuration = const Duration(seconds: 90),
}) async {
  var retryDelay = initialRetryDelay;
  final startedAt = DateTime.now();
  for (;;) {
    try {
      return await EndpointSessionClient.open(
        runtime,
        endpointId,
        cancelWhen: cancelWhen,
      );
    } catch (error) {
      if (error is BindingOperationCancelledException ||
          !_retryableSessionOpenError(error)) {
        rethrow;
      }
      final elapsed = DateTime.now().difference(startedAt);
      if (maximumRetryDuration <= Duration.zero ||
          elapsed >= maximumRetryDuration) {
        rethrow;
      }
      final remaining = maximumRetryDuration - elapsed;
      final wait = retryDelay.compareTo(remaining) < 0 ? retryDelay : remaining;
      await _waitForSessionRetry(wait, cancelWhen);
      if (DateTime.now().difference(startedAt) >= maximumRetryDuration) {
        rethrow;
      }
      final doubledMilliseconds = retryDelay.inMilliseconds * 2;
      retryDelay = Duration(
        milliseconds: doubledMilliseconds.clamp(
          0,
          maximumRetryDelay.inMilliseconds,
        ),
      );
    }
  }
}

bool _retryableSessionOpenError(Object error) {
  if (error is TimeoutException) return true;
  if (error is! NativeSessionException || !error.retryable) return false;
  return switch (error.code) {
    ApiErrorCode.API_ERROR_CODE_UNAVAILABLE ||
    ApiErrorCode.API_ERROR_CODE_STALE_SESSION ||
    ApiErrorCode.API_ERROR_CODE_CANCELLED ||
    ApiErrorCode.API_ERROR_CODE_INTERNAL ||
    null => true,
    _ => false,
  };
}

Future<void> _waitForSessionRetry(
  Duration delay,
  Future<void>? cancelWhen,
) async {
  if (cancelWhen == null) {
    await Future<void>.delayed(delay);
    return;
  }
  var cancelled = false;
  final cancellation = cancelWhen.then((_) => cancelled = true);
  await Future.any<void>([Future<void>.delayed(delay), cancellation]);
  if (cancelled) throw const BindingOperationCancelledException();
}

final class TerminalConnection {
  static const _maximumPendingInputOperations = 128;
  static const _maximumPendingInputBytes = 1024 * 1024;
  static const _maximumPendingResizeOperations = 4;

  TerminalConnection._live(
    this._session,
    TerminalRef terminal,
    TerminalInfo info,
    TerminalAttachResult attached,
    this._inputEncoder,
  ) : terminal = terminal.deepCopy(),
      _terminalState = info.state,
      _historyOnly = false,
      _attachment = attached.attachment.deepCopy(),
      _resizeControl = attached.hasResizeControl()
          ? attached.resizeControl.deepCopy()
          : ResizeControl(),
      _terminalSize = attached.hasSize()
          ? attached.size.deepCopy()
          : info.hasSize()
          ? info.size.deepCopy()
          : TerminalSize(cols: 80, rows: 24);

  TerminalConnection._history(
    this._session,
    TerminalRef terminal,
    TerminalInfo info,
  ) : terminal = terminal.deepCopy(),
      _terminalState = info.state,
      _historyOnly = true,
      _attachment = null,
      _inputEncoder = null,
      _resizeControl = ResizeControl(
        canResize: false,
        reason: ResizeControlReason.RESIZE_CONTROL_REASON_OBSERVER,
      ),
      _terminalSize = info.hasSize()
          ? info.size.deepCopy()
          : TerminalSize(cols: 80, rows: 24);

  final EndpointSessionClient _session;
  final TerminalRef terminal;
  final bool _historyOnly;
  final AttachmentHandle? _attachment;
  final TerminalInputEncoder? _inputEncoder;
  final StreamController<CanonicalLiveScreen> _screens =
      StreamController<CanonicalLiveScreen>.broadcast();
  final StreamController<ResizeControl> _resizeControls =
      StreamController<ResizeControl>.broadcast();
  final StreamController<TerminalState> _terminalStates =
      StreamController<TerminalState>.broadcast();
  final StreamController<TerminalDeliveryState> _deliveryStates =
      StreamController<TerminalDeliveryState>.broadcast();
  CanonicalLiveScreen? _current;
  TerminalState _terminalState;
  ResizeControl _resizeControl;
  TerminalSize _terminalSize;
  TerminalCellMetrics _inputMetrics = defaultTerminalMetrics;
  StreamSubscription<application.EventEnvelope>? _applicationEvents;
  Timer? _screenPublishTimer;
  Timer? _presentationWatchdog;
  Stopwatch? _renderLatencyWatch;
  int _renderLatencyRevision = -1;
  Completer<void>? _screenPollCancellation;
  final BoundedSerialOperationQueue _inputQueue = BoundedSerialOperationQueue(
    maximumOperations: _maximumPendingInputOperations,
    maximumCost: _maximumPendingInputBytes,
  );
  int _mouseInputGeneration = 0;
  final BoundedSerialOperationQueue _resizeQueue = BoundedSerialOperationQueue(
    maximumOperations: _maximumPendingResizeOperations,
    maximumCost: _maximumPendingResizeOperations,
  );
  ({int cols, int rows})? _lastResizeRequest;
  bool _forceFullReplace = false;
  bool _inputDeliveryReady = false;
  bool _screenPolling = false;
  TerminalDeliveryState _deliveryState = TerminalDeliveryState.awaitingFrame;
  int _presentedRevision = -1;
  bool _closed = false;

  static Future<TerminalConnection> open(
    EndpointSessionClient session,
    TerminalRef terminal,
  ) async {
    var info = await session.getTerminal(terminal);
    late final TerminalConnection connection;
    if (terminalUsesHistoryOnly(info.state)) {
      connection = TerminalConnection._history(session, terminal, info);
    } else {
      TerminalAttachResult attached;
      try {
        attached = await session.attach(terminal);
      } catch (error, stackTrace) {
        info = await session.getTerminal(terminal);
        if (!terminalUsesHistoryOnly(info.state)) {
          Error.throwWithStackTrace(error, stackTrace);
        }
        connection = TerminalConnection._history(session, terminal, info);
        connection._applicationEvents = session.watchApplicationEvents().listen(
          connection._handleApplicationEvent,
        );
        return connection;
      }
      connection = TerminalConnection._live(
        session,
        terminal,
        info,
        attached,
        TerminalInputEncoder.open(),
      );
      final attachment = connection._attachment!;
      connection._resizeControl = projectResizeControl(
        incoming: connection._resizeControl,
        surfaceId: attachment.surfaceId,
        viewId: attachment.viewId,
      );
    }
    connection._applicationEvents = session.watchApplicationEvents().listen(
      connection._handleApplicationEvent,
    );
    if (!connection._historyOnly) unawaited(connection._pollScreens());
    return connection;
  }

  CanonicalLiveScreen? get current => _current;
  bool get historyOnly => _historyOnly;
  TerminalState get terminalState => _terminalState;
  ResizeControl get resizeControl => _resizeControl.deepCopy();
  TerminalSize get terminalSize => _terminalSize.deepCopy();
  TerminalDeliveryState get deliveryState => _deliveryState;

  void updateInputMetrics(TerminalCellMetrics metrics) {
    if (_inputMetrics == metrics) return;
    _inputMetrics = metrics;
    final current = _current;
    if (current != null) _applyInputGeometry(current);
  }

  Stream<CanonicalLiveScreen> watchScreens() async* {
    final current = _current;
    if (current != null) yield current;
    yield* _screens.stream;
  }

  Stream<ResizeControl> watchResizeControl() async* {
    yield _resizeControl.deepCopy();
    yield* _resizeControls.stream;
  }

  Stream<TerminalState> watchTerminalState() async* {
    yield _terminalState;
    yield* _terminalStates.stream;
  }

  Stream<TerminalDeliveryState> watchDeliveryState() async* {
    yield _deliveryState;
    yield* _deliveryStates.stream;
  }

  void acknowledgePresentation(int revision) {
    if (_closed || revision < 0 || revision < _presentedRevision) return;
    _presentedRevision = revision;
    final renderLatencyWatch = _renderLatencyWatch;
    if (renderLatencyWatch != null && revision >= _renderLatencyRevision) {
      renderLatencyWatch.stop();
      _session.recordRenderLatency(renderLatencyWatch.elapsed);
      _renderLatencyWatch = null;
      _renderLatencyRevision = -1;
    }
    _inputDeliveryReady = true;
    _publishDeliveryState(TerminalDeliveryState.ready);
    _presentationWatchdog?.cancel();
    _presentationWatchdog = null;
    final current = _current;
    if (current != null && current.revision.toInt() > revision) {
      _schedulePresentationWatchdog();
    }
  }

  Future<void> sendText(String text, {int modifiers = 0}) async {
    if (text.isEmpty) return;
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) {
      throw const NativeSessionException('Exited terminal is read-only');
    }
    if (modifiers == 0) {
      return _queueInput(Uint8List.fromList(utf8.encode(text)));
    }
    final output = BytesBuilder(copy: false);
    for (final codepoint in text.runes) {
      final usage = _asciiHidUsage(codepoint);
      if (usage == null) continue;
      output.add(
        inputEncoder.encodeKey(
          hidUsage: usage,
          modifiers: modifiers,
          unshiftedCodepoint: codepoint,
          text: String.fromCharCode(codepoint),
        ),
      );
    }
    return _queueInput(output.takeBytes());
  }

  Future<void> sendKey({
    required int hidUsage,
    int modifiers = 0,
    int unshiftedCodepoint = 0,
    String text = '',
  }) {
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) {
      return Future.error(
        const NativeSessionException('Exited terminal is read-only'),
      );
    }
    return _queueInput(
      inputEncoder.encodeKey(
        hidUsage: hidUsage,
        modifiers: modifiers,
        unshiftedCodepoint: unshiftedCodepoint,
        text: text,
      ),
    );
  }

  Future<void> sendPaste(String text, {bool allowUnsafe = false}) {
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) {
      return Future.error(
        const NativeSessionException('Exited terminal is read-only'),
      );
    }
    return _queueInput(
      inputEncoder.encodePaste(text, allowUnsafe: allowUnsafe),
    );
  }

  Future<void> sendScroll({
    required bool up,
    required double x,
    required double y,
    int modifiers = 0,
  }) {
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) {
      return Future.error(
        const NativeSessionException('Exited terminal is read-only'),
      );
    }
    return _queueInput(
      inputEncoder.encodeScroll(up: up, modifiers: modifiers, x: x, y: y),
      mouseInputGeneration: _mouseInputGeneration,
    );
  }

  Future<void> sendPrimaryClick({
    required double x,
    required double y,
    int modifiers = 0,
  }) {
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) {
      return Future.error(
        const NativeSessionException('Exited terminal is read-only'),
      );
    }
    return _queueInput(
      inputEncoder.encodePrimaryClick(modifiers: modifiers, x: x, y: y),
      mouseInputGeneration: _mouseInputGeneration,
    );
  }

  void cancelPendingMouseInput() {
    _mouseInputGeneration += 1;
  }

  Future<bool> fitViewport({required int cols, required int rows}) async {
    final attachment = _attachment;
    if (attachment == null) return false;
    final next = (cols: cols.clamp(20, 500), rows: rows.clamp(4, 300));
    if (!_resizeControl.canResize ||
        (_terminalSize.cols == next.cols && _terminalSize.rows == next.rows) ||
        _lastResizeRequest == next) {
      return false;
    }
    _lastResizeRequest = next;
    try {
      final result = await _queueResize(
        () => _session.resizeTerminal(
          attachment: attachment,
          size: TerminalSize(cols: next.cols, rows: next.rows),
          policy: ResizePolicy.RESIZE_POLICY_OWNER,
          takeOwnership: false,
          expectedOwnerEpoch: _ownerEpoch(),
        ),
      );
      return result.resized;
    } finally {
      if (_lastResizeRequest == next) _lastResizeRequest = null;
    }
  }

  Future<ResizeControl> requestResizeOwnership({
    required int cols,
    required int rows,
  }) async {
    final attachment = _attachment;
    if (attachment == null) {
      throw const NativeSessionException('Exited terminal has no resize owner');
    }
    await _queueResize(
      () => _session.resizeTerminal(
        attachment: attachment,
        size: TerminalSize(cols: cols.clamp(20, 500), rows: rows.clamp(4, 300)),
        policy: ResizePolicy.RESIZE_POLICY_OWNER,
        takeOwnership: true,
        expectedOwnerEpoch: _ownerEpoch(),
      ),
    );
    return _resizeControl.deepCopy();
  }

  Future<ResizeControl> releaseResizeOwnership() async {
    final attachment = _attachment;
    if (attachment == null) {
      throw const NativeSessionException('Exited terminal has no resize owner');
    }
    await _queueResize(
      () => _session.resizeTerminal(
        attachment: attachment,
        size: _terminalSize,
        policy: ResizePolicy.RESIZE_POLICY_FOLLOWER,
        takeOwnership: false,
        expectedOwnerEpoch: _ownerEpoch(),
      ),
    );
    return _resizeControl.deepCopy();
  }

  Future<ResizeControl> setResizeLock(bool locked) async {
    final attachment = _attachment;
    if (attachment == null) {
      throw const NativeSessionException('Exited terminal has no resize owner');
    }
    await _queueResize(
      () => _session.setTerminalResizeLock(
        attachment: attachment,
        locked: locked,
      ),
    );
    return _resizeControl.deepCopy();
  }

  Future<HistoryMerged> openHistory({
    int limit = maximumHistoryWindowRequestRows,
    int? cols,
  }) async {
    final projectionCols = cols != null && cols > 0
        ? cols
        : _current?.cols ?? _terminalSize.cols;
    final incoming = await _session.historyWindow(
      HistoryWindowCommand(
        terminal: terminal,
        mode: HistoryWindowMode.HISTORY_WINDOW_MODE_LATEST,
        limit: limit.clamp(1, maximumHistoryWindowRequestRows),
        cols: projectionCols,
      ),
    );
    _requireHistoryTerminal(incoming);
    final outcome = mergeHistoryWindow(current: null, incoming: incoming);
    if (outcome case final HistoryMerged merged) return merged;
    throw NativeSessionException((outcome as HistoryRejected).reason);
  }

  Future<HistoryMerged> loadOlderHistory(
    FrozenHistory current, {
    int limit = maximumHistoryWindowRequestRows,
  }) async {
    if (!current.hasMore || current.rows.isEmpty) {
      return HistoryMerged(history: current, prependedRows: 0);
    }
    final first = current.rows.first;
    final incoming = await _session.historyWindow(
      HistoryWindowCommand(
        terminal: terminal,
        mode: HistoryWindowMode.HISTORY_WINDOW_MODE_OLDER,
        limit: limit.clamp(1, maximumHistoryWindowRequestRows),
        cols: current.cols,
        token: current.token,
        historyGeneration: current.generation,
        beforeCursor: HistoryCursor(
          lineId: first.logicalLineId,
          rowInLine: first.rowInLine,
          segment: first.segment,
        ),
        boundaryFirstLineId: current.rows.first.logicalLineId,
        boundaryLastLineId: current.rows.last.logicalLineId,
      ),
    );
    _requireHistoryTerminal(incoming);
    final outcome = mergeHistoryWindow(current: current, incoming: incoming);
    if (outcome case final HistoryMerged merged) return merged;
    throw NativeSessionException((outcome as HistoryRejected).reason);
  }

  Future<void> releaseHistory(FrozenHistory history) {
    return _session.releaseHistory(
      terminal: terminal,
      token: history.token,
      generation: history.generation,
    );
  }

  Future<String> copyHistoryRange(
    FrozenHistory history,
    HistoryRange selection,
  ) async {
    _requireFrozenHistory(history);
    var remaining = selection.deepCopy();
    final chunks = <String>[];
    for (;;) {
      final result = await _session.historyCopy(
        HistoryCopyCommand(
          terminal: terminal,
          window: HistoryWindowCommand(
            terminal: terminal,
            limit: 1,
            cols: history.cols,
            token: history.token,
            historyGeneration: history.generation,
            boundaryFirstLineId: history.rows.first.logicalLineId,
            boundaryLastLineId: history.rows.last.logicalLineId,
            range: remaining,
          ),
          maxLines: 8192,
          maxBytes: 512 * 1024,
        ),
      );
      chunks.add(result.text);
      if (result.done) return chunks.join('\n');
      if (!result.hasNext() || result.next.lineId == Int64.ZERO) {
        throw const NativeSessionException(
          'History copy response had no continuation',
        );
      }
      if (result.next.lineId == remaining.startLineId &&
          result.next.col == remaining.startCol) {
        throw const NativeSessionException(
          'History copy continuation did not advance',
        );
      }
      remaining = HistoryRange(
        startLineId: result.next.lineId,
        startCol: result.next.col,
        endLineId: selection.endLineId,
        endCol: selection.endCol,
      );
    }
  }

  Future<TerminalHistorySearchOutcome> searchHistory({
    required FrozenHistory history,
    required String query,
    required HistorySearchDirection direction,
    required HistorySearchMode mode,
    HistoryTextPosition? start,
    int limit = 160,
  }) async {
    _requireFrozenHistory(history);
    if (query.isEmpty) {
      return TerminalHistorySearchOutcome(
        history: history,
        match: null,
        wrapped: false,
      );
    }
    final result = await _session.historySearch(
      HistorySearchCommand(
        terminal: terminal,
        token: history.token,
        historyGeneration: history.generation,
        query: query,
        direction: direction,
        cols: history.cols,
        limit: limit,
        start: start,
        mode: mode,
      ),
    );
    if (!result.found) {
      return TerminalHistorySearchOutcome(
        history: history,
        match: null,
        wrapped: result.wrapped,
      );
    }
    if (!result.hasMatch() || !result.hasWindow()) {
      throw const NativeSessionException(
        'History search result was incomplete',
      );
    }
    _requireHistoryTerminal(result.window);
    if (result.window.token != history.token ||
        result.window.historyGeneration != history.generation ||
        !result.window.hasSize() ||
        result.window.size.cols != history.cols) {
      throw const NativeSessionException(
        'History search changed the frozen snapshot',
      );
    }
    final merged = mergeHistoryWindow(
      current: history,
      incoming: result.window,
    );
    if (merged case final HistoryMerged value) {
      return TerminalHistorySearchOutcome(
        history: value.history,
        match: result.match,
        wrapped: result.wrapped,
      );
    }
    throw NativeSessionException((merged as HistoryRejected).reason);
  }

  Future<TerminalHistorySearchScanBatch> scanHistory({
    required FrozenHistory history,
    required String query,
    required HistorySearchMode mode,
    HistoryTextPosition? start,
    int maxMatches = 128,
  }) async {
    _requireFrozenHistory(history);
    final result = await _session.historySearch(
      HistorySearchCommand(
        terminal: terminal,
        token: history.token,
        historyGeneration: history.generation,
        query: query,
        direction: HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD,
        cols: history.cols,
        limit: 1,
        start: start,
        mode: mode,
        scan: true,
        maxMatches: maxMatches.clamp(1, 256),
      ),
    );
    return TerminalHistorySearchScanBatch(
      matches: result.scanMatches,
      next: result.hasScanNext() ? result.scanNext : null,
      done: result.scanDone,
    );
  }

  Future<void> close() async {
    if (_closed) return;
    _closed = true;
    _cancelScreenPoll();
    try {
      if (_attachment case final attachment?) {
        try {
          await _session.detach(attachment);
        } catch (_) {
          // Closing remains idempotent after a supervisor generation change.
        }
      }
    } finally {
      _screenPublishTimer?.cancel();
      _presentationWatchdog?.cancel();
      _renderLatencyWatch?.stop();
      _renderLatencyWatch = null;
      await _applicationEvents?.cancel();
      _inputEncoder?.close();
      await _screens.close();
      await _resizeControls.close();
      await _terminalStates.close();
      await _deliveryStates.close();
    }
  }

  Future<TerminalResizeResult> _queueResize(
    Future<TerminalResizeResult> Function() operation,
  ) {
    return _resizeQueue.schedule(
      cost: 1,
      overflowError: () =>
          const NativeSessionException('Terminal resize queue is full'),
      operation: () async {
        if (_closed) throw StateError('Terminal connection is closed');
        final result = await operation();
        if (!_closed) _acceptResizeResult(result);
        return result;
      },
    );
  }

  void _acceptResizeResult(TerminalResizeResult result) {
    final attachment = _attachment;
    if (attachment == null) return;
    _terminalSize = result.size.deepCopy();
    _publishResizeControl(
      projectResizeControl(
        incoming: result.resizeControl,
        surfaceId: attachment.surfaceId,
        viewId: attachment.viewId,
      ),
    );
  }

  void _handleApplicationEvent(application.EventEnvelope event) {
    if (_closed ||
        event.whichEvent() !=
            application.EventEnvelope_Event.terminalLifecycle) {
      return;
    }
    final lifecycle = event.terminalLifecycle;
    if (!lifecycle.hasTerminal() ||
        lifecycle.terminal.ref.endpointId != terminal.endpointId ||
        lifecycle.terminal.ref.terminalId != terminal.terminalId) {
      return;
    }
    if (lifecycle.terminal.hasSize()) {
      _terminalSize = lifecycle.terminal.size.deepCopy();
    }
    final nextState = lifecycle.terminal.state;
    if (nextState != _terminalState) {
      _terminalState = nextState;
      _terminalStates.add(nextState);
    }
    if (lifecycle.terminal.state == TerminalState.TERMINAL_STATE_EXITED ||
        lifecycle.terminal.state == TerminalState.TERMINAL_STATE_REMOVED) {
      _inputDeliveryReady = false;
    }
    final attachment = _attachment;
    if (attachment != null &&
        lifecycle.attachmentProjection &&
        lifecycle.hasResizeControl()) {
      final control = lifecycle.resizeControl.deepCopy();
      if (lifecycle.resizeEpoch != Int64.ZERO) {
        control.ensureOwnership().epoch = lifecycle.resizeEpoch;
      }
      _publishResizeControl(
        projectResizeControl(
          incoming: control,
          surfaceId: attachment.surfaceId,
          viewId: attachment.viewId,
        ),
      );
    }
  }

  void _publishResizeControl(ResizeControl control) {
    _resizeControl = control.deepCopy();
    if (!_closed) _resizeControls.add(_resizeControl.deepCopy());
  }

  Int64 _ownerEpoch() {
    return _resizeControl.hasOwnership()
        ? _resizeControl.ownership.epoch
        : Int64.ZERO;
  }

  Future<void> _pollScreens() async {
    if (_screenPolling) return;
    _screenPolling = true;
    try {
      while (!_closed && !terminalUsesHistoryOnly(_terminalState)) {
        final cancellation = Completer<void>();
        _screenPollCancellation = cancellation;
        try {
          final incoming = await _session.nextScreen(
            terminal,
            _forceFullReplace ? 0 : (_current?.revision.toInt() ?? 0),
            cancelWhen: cancellation.future,
          );
          if (_closed) return;
          final outcome = mergeLiveScreen(
            current: _current,
            incoming: incoming,
            connectionGeneration: _session.stamp.generation,
            expectedTerminal: terminal,
          );
          switch (outcome) {
            case LiveScreenMerged():
              _current = outcome.screen;
              _renderLatencyWatch = Stopwatch()..start();
              _renderLatencyRevision = outcome.screen.revision.toInt();
              if (outcome.damage.fullReplace) _forceFullReplace = false;
              if (outcome.screen.modes case final modes?) {
                _inputEncoder!.applyModes(modes);
              }
              _applyInputGeometry(outcome.screen);
              _scheduleScreenPublish();
              _schedulePresentationWatchdog();
            case LiveScreenRejected():
              if (outcome.requestFullReplace) {
                _forceFullReplace = true;
                _inputDeliveryReady = false;
                _publishDeliveryState(TerminalDeliveryState.recovering);
              }
          }
        } on BindingOperationCancelledException {
          if (_closed || terminalUsesHistoryOnly(_terminalState)) return;
          continue;
        } on TimeoutException {
          if (terminalUsesHistoryOnly(_terminalState)) return;
          continue;
        } catch (error, stackTrace) {
          _inputDeliveryReady = false;
          _publishDeliveryState(TerminalDeliveryState.recovering);
          if (!_closed && !terminalUsesHistoryOnly(_terminalState)) {
            _screens.addError(error, stackTrace);
          }
          return;
        } finally {
          if (identical(_screenPollCancellation, cancellation)) {
            _screenPollCancellation = null;
          }
        }
      }
    } finally {
      _screenPolling = false;
    }
  }

  void _cancelScreenPoll() {
    final cancellation = _screenPollCancellation;
    if (cancellation != null && !cancellation.isCompleted) {
      cancellation.complete();
    }
  }

  void _applyInputGeometry(CanonicalLiveScreen screen) {
    final inputEncoder = _inputEncoder;
    if (inputEncoder == null) return;
    inputEncoder.applyGeometry(
      TerminalInputGeometry(
        screenWidth: screen.cols * _inputMetrics.cellWidth,
        screenHeight: screen.rows * _inputMetrics.rowHeight,
        cellWidth: _inputMetrics.cellWidth,
        cellHeight: _inputMetrics.rowHeight,
      ),
    );
  }

  void _scheduleScreenPublish() {
    if (_screenPublishTimer != null || _closed) return;
    _screenPublishTimer = Timer(const Duration(milliseconds: 16), () {
      _screenPublishTimer = null;
      final current = _current;
      if (!_closed && current != null) _screens.add(current);
    });
  }

  void _schedulePresentationWatchdog() {
    if (_presentationWatchdog != null || _closed) return;
    _presentationWatchdog = Timer(const Duration(seconds: 2), () {
      _presentationWatchdog = null;
      final current = _current;
      if (_closed ||
          current == null ||
          _presentedRevision >= current.revision.toInt()) {
        return;
      }
      _inputDeliveryReady = false;
      _publishDeliveryState(TerminalDeliveryState.stalled);
    });
  }

  void _publishDeliveryState(TerminalDeliveryState state) {
    if (_deliveryState == state) return;
    _deliveryState = state;
    if (!_closed) _deliveryStates.add(state);
  }

  void _requireHistoryTerminal(HistoryWindowResult value) {
    if (!value.hasTerminal() ||
        value.terminal.endpointId != terminal.endpointId ||
        value.terminal.terminalId != terminal.terminalId) {
      throw const NativeSessionException('History terminal identity mismatch');
    }
  }

  void _requireFrozenHistory(FrozenHistory history) {
    if (history.token.isEmpty ||
        history.generation == Int64.ZERO ||
        history.cols <= 0 ||
        history.rows.isEmpty) {
      throw const NativeSessionException('Frozen history is incomplete');
    }
  }

  Future<void> _queueInput(Uint8List bytes, {int? mouseInputGeneration}) {
    if (_closed) {
      return Future.error(StateError('Terminal connection is closed'));
    }
    final attachment = _attachment;
    if (attachment == null) {
      return Future.error(
        const NativeSessionException('Exited terminal is read-only'),
      );
    }
    if (!_inputDeliveryReady) {
      return Future.error(
        const NativeSessionException(
          'Terminal input is paused while the screen recovers',
        ),
      );
    }
    return _inputQueue.schedule(
      cost: bytes.length,
      overflowError: () =>
          const NativeSessionException('Terminal input queue is full'),
      operation: () async {
        if (_closed ||
            (mouseInputGeneration != null &&
                mouseInputGeneration != _mouseInputGeneration)) {
          return;
        }
        await _session.sendInput(attachment, bytes);
      },
    );
  }
}

final class TerminalHistorySearchOutcome {
  TerminalHistorySearchOutcome({
    required this.history,
    required HistoryRange? match,
    required this.wrapped,
  }) : match = match?.deepCopy();

  final FrozenHistory history;
  final HistoryRange? match;
  final bool wrapped;
}

final class TerminalHistorySearchScanBatch {
  TerminalHistorySearchScanBatch({
    required List<HistoryRange> matches,
    required HistoryTextPosition? next,
    required this.done,
  }) : matches = List.unmodifiable(matches.map((match) => match.deepCopy())),
       next = next?.deepCopy();

  final List<HistoryRange> matches;
  final HistoryTextPosition? next;
  final bool done;
}

void _throwBindingError(ApiError? error) {
  if (error == null) return;
  throw NativeSessionException(
    error.message.isEmpty ? error.code.name : error.message,
    code: error.code,
    retryable: error.retryable,
    attempted: error.attempted,
  );
}

int? _asciiHidUsage(int codepoint) {
  const page = 0x00070000;
  if (codepoint >= 0x61 && codepoint <= 0x7a) {
    return page | (0x04 + codepoint - 0x61);
  }
  if (codepoint >= 0x41 && codepoint <= 0x5a) {
    return page | (0x04 + codepoint - 0x41);
  }
  if (codepoint >= 0x31 && codepoint <= 0x39) {
    return page | (0x1e + codepoint - 0x31);
  }
  if (codepoint == 0x30) return page | 0x27;
  return switch (codepoint) {
    0x0d || 0x0a => page | 0x28,
    0x1b => page | 0x29,
    0x08 => page | 0x2a,
    0x09 => page | 0x2b,
    0x20 => page | 0x2c,
    0x2d => page | 0x2d,
    0x3d => page | 0x2e,
    0x5b => page | 0x2f,
    0x5d => page | 0x30,
    0x5c => page | 0x31,
    0x3b => page | 0x33,
    0x27 => page | 0x34,
    0x60 => page | 0x35,
    0x2c => page | 0x36,
    0x2e => page | 0x37,
    0x2f => page | 0x38,
    _ => null,
  };
}
