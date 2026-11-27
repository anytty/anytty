part of 'terminal_workspace_screen.dart';

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
