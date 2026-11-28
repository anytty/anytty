part of 'terminal_workspace_screen.dart';

final class _WorkspaceNetworkAction extends StatelessWidget {
  const _WorkspaceNetworkAction({
    required this.diagnostics,
    required this.onPressed,
  });

  final AsyncValue<EndpointConnectionDiagnostics> diagnostics;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final snapshot = diagnostics.valueOrNull?.snapshot;
    final connected = snapshot?.connected ?? false;
    final label = snapshot == null
        ? diagnostics.isLoading
              ? '...'
              : '--'
        : _workspaceConnectionKind(snapshot);
    final color = connected ? palette.accent : palette.muted;
    final semanticLabel = anyttyText(
      context,
      en: 'Network status, $label, show details',
      zh: '网络状态，$label，查看详情',
    );
    return Semantics(
      button: true,
      label: semanticLabel,
      excludeSemantics: true,
      child: Tooltip(
        message: semanticLabel,
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            key: const ValueKey('terminal-network-status'),
            onTap: onPressed,
            child: SizedBox(
              width: 44,
              height: 38,
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.route_rounded, size: 16, color: color),
                  const SizedBox(height: 1),
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.fade,
                    style: TextStyle(
                      color: color,
                      fontSize: 7,
                      height: 1,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<void> _showWorkspaceConnectionDetails({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required String endpointLabel,
}) async {
  ref.invalidate(connectionDiagnosticsProvider(endpointId));
  final openSettings = await showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: AnyttyPalette.of(context).surface,
    barrierColor: AnyttyPalette.of(context).overlay,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
    ),
    builder: (context) => _WorkspaceConnectionDetailsSheet(
      endpointId: endpointId,
      endpointLabel: endpointLabel,
    ),
  );
  if (openSettings == true && context.mounted) {
    context.push(
      '/connection/${Uri.encodeComponent(endpointId)}'
      '?label=${Uri.encodeQueryComponent(endpointLabel)}',
    );
  }
}

final class _WorkspaceConnectionDetailsSheet extends ConsumerWidget {
  const _WorkspaceConnectionDetailsSheet({
    required this.endpointId,
    required this.endpointLabel,
  });

  final String endpointId;
  final String endpointLabel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AnyttyPalette.of(context);
    final diagnostics = ref.watch(connectionDiagnosticsProvider(endpointId));
    final value = diagnostics.valueOrNull;
    final snapshot = value?.snapshot;
    return SafeArea(
      top: false,
      child: FractionallySizedBox(
        heightFactor: 0.78,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(18, 4, 18, 14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          anyttyText(context, en: 'Network status', zh: '网络状态'),
                          style: const TextStyle(
                            fontSize: 19,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          endpointLabel,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.muted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: anyttyText(context, en: 'Close', zh: '关闭'),
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              if (diagnostics.isLoading && snapshot == null)
                const Expanded(
                  child: Center(
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                )
              else if (snapshot == null)
                Expanded(
                  child: _WorkspaceConnectionError(
                    message: anyttyText(
                      context,
                      en: 'Network details are temporarily unavailable.',
                      zh: '暂时无法读取网络详情。',
                    ),
                    onRetry: () => ref.invalidate(
                      connectionDiagnosticsProvider(endpointId),
                    ),
                  ),
                )
              else ...[
                _WorkspaceConnectionSummary(snapshot: snapshot),
                if (diagnostics.isRefreshing) ...[
                  const SizedBox(height: 8),
                  const LinearProgressIndicator(minHeight: 2),
                ],
                const SizedBox(height: 12),
                Expanded(
                  child: ListView(
                    padding: EdgeInsets.zero,
                    children: [
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Connection type',
                          zh: '连接类型',
                        ),
                        value: _workspaceConnectionKind(snapshot),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Route', zh: '连接路由'),
                        value:
                            '${_workspaceRouteLabel(context, snapshot.routeKind)}  ·  ${snapshot.routeId.trim().isEmpty ? '--' : snapshot.routeId}',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Observed path',
                          zh: '实际路径',
                        ),
                        value: _workspacePathLabel(
                          context,
                          snapshot.observedPath,
                        ),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Latency', zh: '延迟'),
                        value: _workspaceLatency(
                          snapshot.roundTripNanos.toInt(),
                        ),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Network', zh: '网络类型'),
                        value: _workspaceNetworkClass(snapshot.networkClass),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Transport', zh: '传输协议'),
                        value: _workspaceTransportSummary(snapshot),
                      ),
                      if (snapshot.observedPath ==
                          ConnectionObservedPath
                              .CONNECTION_OBSERVED_PATH_SINGLE_RELAY)
                        _WorkspaceConnectionDetailRow(
                          label: anyttyText(
                            context,
                            en: 'ICE transport',
                            zh: 'ICE 传输',
                          ),
                          value: _workspaceIceTransportSummary(snapshot),
                        ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Local candidate',
                          zh: '本地候选',
                        ),
                        value:
                            '${_workspaceAddress(snapshot.localIp, snapshot.localPort)}  (${_workspaceCandidateLabel(snapshot.localCandidateType)})',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Remote candidate',
                          zh: '远端候选',
                        ),
                        value:
                            '${_workspaceAddress(snapshot.remoteIp, snapshot.remotePort)}  (${_workspaceCandidateLabel(snapshot.remoteCandidateType)})',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Traffic', zh: '流量'),
                        value:
                            '↓ ${formatResourceBytes(snapshot.bytesReceived.toInt())}  ↑ ${formatResourceBytes(snapshot.bytesSent.toInt())}',
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
                SizedBox(
                  height: 48,
                  child: OutlinedButton.icon(
                    onPressed: () => Navigator.pop(context, true),
                    icon: const Icon(Icons.tune_rounded, size: 18),
                    label: Text(
                      anyttyText(
                        context,
                        en: 'Connection settings',
                        zh: '连接设置',
                      ),
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

final class _WorkspaceConnectionSummary extends StatelessWidget {
  const _WorkspaceConnectionSummary({required this.snapshot});

  final ConnectionSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final connected = snapshot.connected;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: connected
            ? palette.accent.withValues(alpha: 0.11)
            : palette.surfaceRaised,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: connected
              ? palette.accent.withValues(alpha: 0.35)
              : palette.border,
        ),
      ),
      child: Row(
        children: [
          Icon(
            connected ? Icons.route_rounded : Icons.cloud_off_outlined,
            color: connected ? palette.accent : palette.muted,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _workspaceConnectionKind(snapshot),
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  connected
                      ? anyttyText(context, en: 'Connected', zh: '连接正常')
                      : anyttyText(context, en: 'Disconnected', zh: '当前未连接'),
                  style: TextStyle(color: palette.muted, fontSize: 11),
                ),
              ],
            ),
          ),
          Text(
            _workspaceLatency(snapshot.roundTripNanos.toInt()),
            style: const TextStyle(
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 12,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

final class _WorkspaceConnectionDetailRow extends StatelessWidget {
  const _WorkspaceConnectionDetailRow({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      constraints: const BoxConstraints(minHeight: 44),
      padding: const EdgeInsets.symmetric(vertical: 9),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 98,
            child: Text(
              label,
              style: TextStyle(color: palette.muted, fontSize: 11),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.end,
              style: const TextStyle(
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 11,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _WorkspaceConnectionError extends StatelessWidget {
  const _WorkspaceConnectionError({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.cloud_off_outlined, color: palette.muted, size: 30),
          const SizedBox(height: 10),
          Text(message, style: TextStyle(color: palette.muted)),
          const SizedBox(height: 12),
          OutlinedButton(
            onPressed: onRetry,
            child: Text(anyttyText(context, en: 'Retry', zh: '重试')),
          ),
        ],
      ),
    );
  }
}

String _workspaceConnectionKind(ConnectionSnapshot snapshot) {
  if (!snapshot.connected) return 'Offline';
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH) {
    return 'SSH';
  }
  if (snapshot.observedPath ==
      ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return 'Relay';
  }
  if (snapshot.observedPath ==
      ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT) {
    return 'P2P';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT) {
    return 'Direct';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL) {
    return 'Local';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD) {
    return 'Cloud';
  }
  return 'Connected';
}

String _workspaceRouteLabel(BuildContext context, ConnectionRouteKind kind) {
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL) {
    return anyttyText(context, en: 'Local', zh: '本地');
  }
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT) return 'Direct';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH) return 'SSH';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD) {
    return anyttyText(context, en: 'Cloud', zh: '云端');
  }
  return '--';
}

String _workspacePathLabel(BuildContext context, ConnectionObservedPath path) {
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT) {
    return anyttyText(context, en: 'Direct peer path', zh: 'P2P 直连');
  }
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return anyttyText(context, en: 'Single relay path', zh: '单中继路径');
  }
  return anyttyText(context, en: 'Not reported', zh: '未上报');
}

String _workspaceTransportLabel(ConnectionTransport transport) {
  if (transport == ConnectionTransport.CONNECTION_TRANSPORT_UDP) return 'UDP';
  if (transport == ConnectionTransport.CONNECTION_TRANSPORT_TCP) return 'TCP';
  return '--';
}

String _workspaceTransportSummary(ConnectionSnapshot snapshot) {
  if (snapshot.observedPath ==
      ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return _workspaceTransportLabel(snapshot.relayTransport);
  }
  return _workspaceIceTransportSummary(snapshot);
}

String _workspaceIceTransportSummary(ConnectionSnapshot snapshot) {
  return '${_workspaceTransportLabel(snapshot.localProtocol)} / ${_workspaceTransportLabel(snapshot.remoteProtocol)}';
}

String _workspaceCandidateLabel(ConnectionCandidateType candidate) {
  if (candidate == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_HOST) {
    return 'host';
  }
  if (candidate ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE) {
    return 'srflx';
  }
  if (candidate ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE) {
    return 'prflx';
  }
  if (candidate == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_RELAY) {
    return 'relay';
  }
  return '--';
}

String _workspaceAddress(String ip, int port) {
  final value = ip.trim();
  if (value.isEmpty) return '--';
  if (port <= 0) return value;
  return value.contains(':') ? '[$value]:$port' : '$value:$port';
}

String _workspaceLatency(int nanoseconds) {
  if (nanoseconds <= 0) return '--';
  final milliseconds = nanoseconds / 1000000;
  return milliseconds >= 10
      ? '${milliseconds.round()} ms'
      : '${milliseconds.toStringAsFixed(1)} ms';
}

String _workspaceNetworkClass(String networkClass) {
  final value = networkClass.trim();
  if (value.isEmpty) return '--';
  if (value.toLowerCase() == 'wifi') return 'Wi-Fi';
  return value;
}

/// 已归档的连接尝试：重试之后仍然能看到上一轮为什么失败。
final class _ConnectionAttemptHistoryPanel extends StatelessWidget {
  const _ConnectionAttemptHistoryPanel({
    required this.snapshots,
    required this.expanded,
    required this.onToggle,
  });

  /// 最新一轮在前。
  final List<ConnectionAttemptSnapshot> snapshots;
  final bool expanded;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final count = snapshots.length;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      decoration: BoxDecoration(
        color: palette.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: palette.border),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          InkWell(
            onTap: onToggle,
            borderRadius: BorderRadius.circular(12),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  Icon(Icons.history_rounded, size: 15, color: palette.muted),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      anyttyText(
                        context,
                        en: 'Previous attempts · $count',
                        zh: '之前的尝试 · $count 次',
                      ),
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  ),
                  Icon(
                    expanded
                        ? Icons.expand_less_rounded
                        : Icons.expand_more_rounded,
                    size: 16,
                    color: palette.muted,
                  ),
                ],
              ),
            ),
          ),
          if (expanded)
            for (final snapshot in snapshots)
              Padding(
                padding: const EdgeInsets.only(top: 2, bottom: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      _connectionAttemptSnapshotTitle(context, snapshot),
                      style: TextStyle(
                        color: palette.faint,
                        fontSize: 11,
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      _connectionAttemptSnapshotSummary(context, snapshot),
                      style: TextStyle(
                        color: palette.muted,
                        fontSize: 12,
                        height: 1.4,
                      ),
                    ),
                  ],
                ),
              ),
        ],
      ),
    );
  }
}

String _connectionAttemptSnapshotTitle(
  BuildContext context,
  ConnectionAttemptSnapshot snapshot,
) {
  final started = snapshot.startedAt.toLocal();
  final clock =
      '${started.hour.toString().padLeft(2, '0')}:'
      '${started.minute.toString().padLeft(2, '0')}:'
      '${started.second.toString().padLeft(2, '0')}';
  final elapsed = snapshot.duration;
  final duration = elapsed.inSeconds >= 1
      ? '${elapsed.inSeconds}s'
      : '${elapsed.inMilliseconds}ms';
  return anyttyText(
    context,
    en: 'Attempt #${snapshot.index} · $clock · $duration',
    zh: '第 ${snapshot.index} 次 · $clock · $duration',
  );
}

String _connectionAttemptSnapshotSummary(
  BuildContext context,
  ConnectionAttemptSnapshot snapshot,
) {
  if (snapshot.routes.isEmpty) {
    return anyttyText(
      context,
      en: 'No route reported a result',
      zh: '没有路由返回结果',
    );
  }
  final lines = <String>[
    for (final route in snapshot.routes)
      '${_connectionRouteLabel(context, route.kind)} · '
          '${_connectionRouteSnapshotLabel(context, route)}'
          '${route.errorCode == null ? '' : ' · ${route.errorCode}'}'
          '${route.errorMessage == null ? '' : '\n${route.errorMessage}'}',
  ];
  return lines.join('\n');
}

String _connectionRouteLabel(BuildContext context, ConnectionRouteKind kind) {
  return switch (kind) {
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT => anyttyText(
      context,
      en: 'Direct',
      zh: '直连',
    ),
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH => 'SSH',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD => 'Cloud',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL => anyttyText(
      context,
      en: 'Local',
      zh: '本地',
    ),
    _ => anyttyText(context, en: 'Route', zh: '路由'),
  };
}

String _connectionRouteSnapshotLabel(
  BuildContext context,
  ConnectionRouteSnapshot route,
) {
  if (route.failed) {
    return anyttyText(context, en: 'unavailable', zh: '不可用');
  }
  return switch (route.stage) {
    'attempt_starting' => anyttyText(context, en: 'starting', zh: '开始尝试'),
    'authorization_preparing' => anyttyText(
      context,
      en: 'preparing authorization',
      zh: '准备连接授权',
    ),
    'edge_transport_ready' => anyttyText(
      context,
      en: 'signaling ready',
      zh: '信令就绪',
    ),
    'peer_open' => anyttyText(context, en: 'peer opened', zh: '对端已打开'),
    'ready' => anyttyText(context, en: 'ready', zh: '已就绪'),
    _ => route.stage,
  };
}

final class _ConnectionErrorDetails extends StatelessWidget {
  const _ConnectionErrorDetails({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) => ExpansionTile(
    key: ValueKey(message),
    initiallyExpanded: false,
    tilePadding: const EdgeInsets.symmetric(horizontal: 8),
    childrenPadding: const EdgeInsets.fromLTRB(8, 0, 8, 12),
    shape: const Border(),
    collapsedShape: const Border(),
    title: Text(
      anyttyText(context, en: 'Show errors', zh: '查看错误详情'),
      style: TextStyle(color: AnyttyPalette.of(context).muted, fontSize: 12),
    ),
    children: [
      Align(
        alignment: Alignment.centerLeft,
        child: SelectableText(
          message,
          style: TextStyle(
            color: AnyttyPalette.of(context).muted,
            fontSize: 12,
            height: 1.5,
          ),
        ),
      ),
    ],
  );
}

final class _ConnectionRouteDot extends StatefulWidget {
  const _ConnectionRouteDot({
    super.key,
    required this.active,
    required this.color,
  });
  final bool active;
  final Color color;

  @override
  State<_ConnectionRouteDot> createState() => _ConnectionRouteDotState();
}

final class _ConnectionRouteDotState extends State<_ConnectionRouteDot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _pulse = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1100),
  );
  late final Animation<double> _opacity = Tween(
    begin: 0.35,
    end: 1.0,
  ).animate(CurvedAnimation(parent: _pulse, curve: Curves.easeInOut));

  void _updateMotion() {
    if (widget.active && !MediaQuery.disableAnimationsOf(context)) {
      if (!_pulse.isAnimating) _pulse.repeat(reverse: true);
    } else {
      _pulse.stop();
      _pulse.value = 1;
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _updateMotion();
  }

  @override
  void didUpdateWidget(_ConnectionRouteDot oldWidget) {
    super.didUpdateWidget(oldWidget);
    _updateMotion();
  }

  @override
  void dispose() {
    _pulse.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: FadeTransition(
      opacity: _opacity,
      child: Container(
        width: 8,
        height: 8,
        decoration: BoxDecoration(shape: BoxShape.circle, color: widget.color),
      ),
    ),
  );
}

final class _ConnectionSpinner extends StatelessWidget {
  const _ConnectionSpinner({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return RepaintBoundary(
      child: SizedBox.square(
        dimension: 18,
        child: CircularProgressIndicator(color: color, strokeWidth: 2.2),
      ),
    );
  }
}

String _connectionLoadingLabel(
  BuildContext context,
  EndpointConnectionPhase phase,
) {
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_RESOLVING) {
    return anyttyText(context, en: 'Resolving device address', zh: '正在解析设备地址');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_SIGNALING) {
    return anyttyText(context, en: 'Negotiating connection', zh: '正在进行 ICE 协商');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING) {
    return anyttyText(context, en: 'Connecting to device', zh: '正在连接设备');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_AUTHORIZING) {
    return anyttyText(context, en: 'Authorizing this device', zh: '正在验证设备授权');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_READY) {
    return anyttyText(context, en: 'Loading terminal list', zh: '正在加载终端列表');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_OFFLINE) {
    return anyttyText(
      context,
      en: 'Connection interrupted, retrying',
      zh: '连接中断，正在重试',
    );
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING) {
    return anyttyText(context, en: 'Probing available routes', zh: '正在探测可用线路');
  }
  return anyttyText(context, en: 'Preparing connection', zh: '正在准备连接');
}

String _connectionModeLabel(
  BuildContext context,
  EndpointRoutePreference? preference,
) => switch (preference) {
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT => anyttyText(
    context,
    en: 'Currently using Direct mode',
    zh: '当前使用直连模式',
  ),
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH => anyttyText(
    context,
    en: 'Currently using SSH mode',
    zh: '当前使用 SSH 模式',
  ),
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD => anyttyText(
    context,
    en: 'Currently using Cloud mode',
    zh: '当前使用 Cloud 模式',
  ),
  _ => anyttyText(
    context,
    en: 'Automatically selecting a connection route',
    zh: '正在自动选择连接线路',
  ),
};

List<EndpointConnectionEvent> _orderedConnectionAttempts(
  Map<ConnectionRouteKind, EndpointConnectionEvent> attempts,
) {
  const order = [
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL,
  ];
  return order
      .map((kind) => attempts[kind])
      .whereType<EndpointConnectionEvent>()
      .toList(growable: false);
}

String _connectionAttemptLabel(
  BuildContext context,
  EndpointConnectionEvent event, {
  bool includeError = false,
}) {
  final route = switch (event.attemptedRouteKind) {
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT => anyttyText(
      context,
      en: 'Direct',
      zh: '直连',
    ),
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH => 'SSH',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD => 'Cloud',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL => anyttyText(
      context,
      en: 'Local',
      zh: '本地',
    ),
    _ => anyttyText(context, en: 'Connection', zh: '连接'),
  };
  final stage = switch (event.connectionStage) {
    'attempt_starting' => anyttyText(context, en: 'starting route', zh: '开始尝试'),
    'attempt_failed' => anyttyText(context, en: 'unavailable', zh: '暂不可用'),
    'authorization_preparing' => anyttyText(
      context,
      en: 'preparing authorization',
      zh: '准备连接授权',
    ),
    'credential_resolving' => anyttyText(
      context,
      en: 'loading credentials',
      zh: '读取连接凭据',
    ),
    'ssh_connecting' => anyttyText(
      context,
      en: 'establishing SSH connection',
      zh: '建立 SSH 连接',
    ),
    'ssh_tunnel_ready' => anyttyText(
      context,
      en: 'SSH tunnel ready',
      zh: 'SSH 隧道已建立',
    ),
    'peer_opening' => anyttyText(
      context,
      en: 'creating secure peer',
      zh: '创建安全连接',
    ),
    'ice_gathering' => anyttyText(
      context,
      en: 'gathering ICE candidates',
      zh: '收集 ICE 候选',
    ),
    'signaling' => anyttyText(
      context,
      en: 'exchanging signaling data',
      zh: '交换连接信令',
    ),
    'ice_connecting' => anyttyText(
      context,
      en: 'negotiating ICE connection',
      zh: '正在进行 ICE 协商',
    ),
    'cloud_cached_edge' => anyttyText(
      context,
      en: 'reusing known Cloud edge',
      zh: '连接已知 Cloud 边缘节点',
    ),
    'cloud_discovering' => anyttyText(
      context,
      en: 'locating device through Cloud',
      zh: '通过 Cloud 查找设备',
    ),
    'cloud_p2p_attempt' => anyttyText(
      context,
      en: 'attempting P2P hole punching',
      zh: '尝试 P2P 打洞',
    ),
    'cloud_relay_attempt' => anyttyText(
      context,
      en: 'establishing Relay channel',
      zh: '建立 Relay 转发通道',
    ),
    'cloud_direct_selected' => anyttyText(
      context,
      en: 'P2P connection established',
      zh: 'P2P 连接已建立',
    ),
    'cloud_relay_fallback' => anyttyText(
      context,
      en: 'P2P unavailable, switched to Relay',
      zh: 'P2P 未建立，已转入 Relay',
    ),
    'cloud_relay_selected' => anyttyText(
      context,
      en: 'Relay channel established',
      zh: 'Relay 转发通道已建立',
    ),
    'transport_authorizing' => anyttyText(
      context,
      en: 'verifying secure channel',
      zh: '验证安全通道',
    ),
    'protocol_opening' => anyttyText(
      context,
      en: 'opening AnyTTY protocol',
      zh: '建立 AnyTTY 会话',
    ),
    _ => _connectionLoadingLabel(context, event.phase),
  };
  final detail =
      includeError &&
          event.connectionStage == 'attempt_failed' &&
          event.hasError()
      ? event.error.message.trim()
      : '';
  if (detail.isEmpty) return '$route · $stage';
  return '$route · $stage\n$detail';
}

final class _DirectOnlyConnectionHelp extends StatelessWidget {
  const _DirectOnlyConnectionHelp({
    required this.applying,
    required this.error,
    required this.onUseAuto,
    required this.onOpenSettings,
  });

  final bool applying;
  final String? error;
  final VoidCallback? onUseAuto;
  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      liveRegion: true,
      container: true,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 42,
                height: 42,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: palette.warning.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(13),
                  border: Border.all(
                    color: palette.warning.withValues(alpha: 0.32),
                  ),
                ),
                child: applying
                    ? _ConnectionSpinner(color: palette.warning)
                    : Icon(
                        Icons.route_outlined,
                        size: 21,
                        color: palette.warning,
                      ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  anyttyText(
                    context,
                    en: 'Direct connection unavailable',
                    zh: '当前网络无法建立直连',
                  ),
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Text(
            anyttyText(
              context,
              en: 'Only direct connections are allowed, so other available routes are not being tried. Switch to Automatic to let AnyTTY choose a working route.',
              zh: '当前设置只允许直连，因此不会尝试其他可用线路。切换为自动连接后，AnyTTY 会选择能够连通的线路。',
            ),
            style: TextStyle(
              color: palette.muted,
              fontSize: 13,
              height: 1.5,
              fontWeight: FontWeight.w500,
            ),
          ),
          if (error != null) ...[
            const SizedBox(height: 10),
            Text(
              error!,
              style: TextStyle(
                color: palette.danger,
                fontSize: 12,
                height: 1.4,
              ),
            ),
          ],
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              FilledButton.icon(
                onPressed: applying ? null : onUseAuto,
                icon: const Icon(Icons.auto_awesome_outlined, size: 17),
                label: Text(
                  anyttyText(context, en: 'Use Automatic', zh: '改为自动连接'),
                ),
              ),
              TextButton.icon(
                onPressed: applying ? null : onOpenSettings,
                icon: const Icon(Icons.tune_rounded, size: 17),
                label: Text(
                  anyttyText(context, en: 'Connection settings', zh: '网络设置'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

