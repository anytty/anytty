part of 'endpoint_session_client.dart';

final class TerminalConnection {
  static const _maximumInflightInputs = 3;
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
  // 输入允许少量在飞：顺序由单条有序通道 + 服务端有序 PTY 写入保证，
  // 客户端不再为每个按键等一个 RTT。
  final BoundedPipelinedOperationQueue _inputQueue = BoundedPipelinedOperationQueue(
    maximumInflight: _maximumInflightInputs,
    maximumOperations: _maximumPendingInputOperations,
    maximumCost: _maximumPendingInputBytes,
  );
  int _mouseInputGeneration = 0;
  final BoundedSerialOperationQueue _resizeQueue = BoundedSerialOperationQueue(
    maximumOperations: _maximumPendingResizeOperations,
    maximumCost: _maximumPendingResizeOperations,
  );
  ({int cols, int rows})? _lastResizeRequest;
  bool _inputDeliveryReady = false;
  DateTime? _inputPausedSince;
  AnyttyResourceStream? _screenStream;
  StreamSubscription<ResourceStreamFrame>? _screenStreamFrames;
  Timer? _screenStreamRetry;
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
    if (!connection._historyOnly) unawaited(connection._startScreenStream());
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
    _inputPausedSince = null;
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
    _screenStreamRetry?.cancel();
    _screenStreamRetry = null;
    await _closeScreenStream();
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
      _pauseInputDelivery();
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

  /// 打开服务端推屏流：attach 声明了 screen_stream，服务端会在这个
  /// attachment channel 上连续推 TypeScreen（Go binding 已把 delta 应用成
  /// 完整行替换），这里只负责把行喂给本地屏幕存储。
  Future<void> _startScreenStream() async {
    if (_closed || _historyOnly) return;
    final attachment = _attachment;
    if (attachment == null) return;
    try {
      final stream = await _session.openAttachmentStream(attachment);
      if (_closed) {
        stream.close();
        return;
      }
      _screenStream = stream;
      _screenStreamFrames = stream.frames.listen(
        _handleScreenFrame,
        onError: (Object error, StackTrace stackTrace) {
          _handleScreenStreamEnded(stream);
          if (!_closed) _screens.addError(error, stackTrace);
        },
        cancelOnError: false,
      );
      unawaited(
        stream.closed.then<void>(
          (_) => _handleScreenStreamEnded(stream),
          onError: (Object _, StackTrace _) => _handleScreenStreamEnded(stream),
        ),
      );
    } catch (error, stackTrace) {
      _pauseInputDelivery();
      _publishDeliveryState(TerminalDeliveryState.recovering);
      if (!_closed) _screens.addError(error, stackTrace);
    }
  }

  void _handleScreenFrame(ResourceStreamFrame frame) {
    if (frame.type !=
        ResourceStreamFrameType.RESOURCE_STREAM_FRAME_TYPE_SCREEN) {
      return;
    }
    final NativeScreenResult incoming;
    try {
      incoming = NativeScreenResult.fromBuffer(frame.payload);
    } catch (_) {
      return;
    }
    _applyIncomingScreen(incoming);
  }

  /// 推流帧落地：行替换由 binding 拍平，这里只需要把变化行写进本地屏幕。
  void _applyIncomingScreen(NativeScreenResult incoming) {
    if (_closed) return;
    final mergeWatch = PerfStopwatch();
    final outcome = mergeLiveScreen(
      current: _current,
      incoming: incoming,
      connectionGeneration: _session.stamp.generation,
      expectedTerminal: terminal,
    );
    TerminalPerfCounters.instance.noteMerge(
      mergeWatch.elapsedUs ?? 0,
      fullReplace: outcome is LiveScreenMerged && outcome.damage.fullReplace,
    );
    switch (outcome) {
      case LiveScreenMerged():
        _current = outcome.screen;
        _renderLatencyWatch = Stopwatch()..start();
        _renderLatencyRevision = outcome.screen.revision.toInt();
        if (outcome.screen.modes case final modes?) {
          _inputEncoder?.applyModes(modes);
        }
        _applyInputGeometry(outcome.screen);
        _scheduleScreenPublish();
        _schedulePresentationWatchdog();
      case LiveScreenRejected():
        // 推流没有“再要一次整屏”的往返：重开一次 attachment stream，
        // 服务端 pump 会从 observed=0 重新发一整帧。
        _scheduleScreenStreamRetry();
    }
  }

  void _handleScreenStreamEnded(AnyttyResourceStream stream) {
    if (_closed || !identical(_screenStream, stream)) return;
    _screenStream = null;
    _screenStreamFrames = null;
    _pauseInputDelivery();
    _publishDeliveryState(TerminalDeliveryState.recovering);
    _scheduleScreenStreamRetry();
  }

  void _scheduleScreenStreamRetry() {
    if (_closed || _historyOnly) return;
    _screenStreamRetry?.cancel();
    _screenStreamRetry = Timer(const Duration(seconds: 1), () {
      if (!_closed) unawaited(_startScreenStream());
    });
  }

  Future<void> _closeScreenStream() async {
    _screenStreamRetry?.cancel();
    _screenStreamRetry = null;
    final subscription = _screenStreamFrames;
    _screenStreamFrames = null;
    if (subscription != null) await subscription.cancel();
    final stream = _screenStream;
    _screenStream = null;
    if (stream != null) stream.close();
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
    if (_closed) return;
    // 直接发布：这里再加 16ms 定时器只会平白多一帧延迟，
    // 下游重建频率本身由 vsync 合并。
    final current = _current;
    if (current != null) _screens.add(current);
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
      _pauseInputDelivery();
      _publishDeliveryState(TerminalDeliveryState.stalled);
    });
  }

  void _pauseInputDelivery() {
    _inputDeliveryReady = false;
    _inputPausedSince ??= DateTime.now();
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
      // 屏幕恢复通常只有一个 RTT。旧行为在这里直接抛错，结果是越卡越打不了字；
      // 现在允许短暂窗口继续输入，只有长时间恢复不了才报错。
      final pausedSince = _inputPausedSince;
      if (pausedSince != null &&
          DateTime.now().difference(pausedSince) >
              const Duration(seconds: 2)) {
        return Future.error(
          const NativeSessionException(
            'Terminal input is paused while the screen recovers',
          ),
        );
      }
    }
    final queueWatch = PerfStopwatch();
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
        TerminalPerfCounters.instance.noteInputQueue(
          queueWatch.elapsedUs ?? 0,
        );
        final ackWatch = PerfStopwatch();
        await _session.sendInput(attachment, bytes);
        TerminalPerfCounters.instance.noteInputAck(ackWatch.elapsedUs ?? 0);
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
