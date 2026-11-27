part of 'endpoint_session_client.dart';

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
  String toString() {
    final value = code;
    if (value == null) return message;
    final codeName = value.name
        .replaceFirst('API_ERROR_CODE_', '')
        .toLowerCase();
    return '$message\ncode=$codeName; retryable=$retryable; attempted=$attempted';
  }
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
  Future<int> startBrowserProxy() async => (await _browserProxyListen()).port;

  @override
  Future<void> stopBrowserProxy(int port) async {
    if (_closeRequested || isClosed) return;
    await _browserProxyListen(stopPort: port);
  }

  Future<BrowserProxyListenResult> _browserProxyListen({
    int stopPort = 0,
  }) async {
    final result = await runBindingOperation<BrowserProxyListenResult>(
      runtime: _runtime,
      begin: () => _runtime.command(
        EngineCommand(
          browserProxyListen: BrowserProxyListenRequest(
            requestId: newRequestId(),
            sessionHandle: Int64(sessionHandle),
            stop: stopPort != 0,
            port: stopPort,
          ),
        ),
      ),
      select: (event) =>
          event.whichEvent() == EventEnvelope_Event.browserProxyListen
          ? event.browserProxyListen.deepCopy()
          : null,
      operationHandle: (value) => value.operationHandle.toInt(),
      timeoutMessage: 'Native browser proxy operation timed out',
      timeout: const Duration(seconds: 15),
    );
    if (result.hasError()) {
      throw NativeSessionException(
        result.error.message,
        code: result.error.code,
        retryable: result.error.retryable,
      );
    }
    if (result.sessionHandle.toInt() != sessionHandle) {
      throw const NativeSessionException('Browser proxy session did not match');
    }
    return result;
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
          // 让服务端在这个 attachment channel 上推行级屏幕 delta（wire.TypeScreen），
          // 不再需要客户端每帧一次 LiveScreenNext 往返。
          screenStream: true,
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

  /// 打开 attachment 的资源流：服务端会在同一个 channel 上推 wire.TypeScreen，
  /// Go binding 应用 delta 后转成 SCREEN 帧交给这里。
  Future<AnyttyResourceStream> openAttachmentStream(
    AttachmentHandle attachment,
  ) {
    return AnyttyResourceStream.open(
      runtime: _runtime,
      sessionHandle: sessionHandle,
      request: OpenResourceStreamRequest(resource: attachment.resource),
    );
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
