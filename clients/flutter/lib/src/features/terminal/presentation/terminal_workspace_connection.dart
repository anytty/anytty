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
