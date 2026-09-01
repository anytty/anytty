import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../data/connection_repository.dart';
import '../data/endpoint_repository.dart';

final class ConnectionScreen extends ConsumerStatefulWidget {
  const ConnectionScreen({super.key, required this.endpointId, this.label});

  final String endpointId;
  final String? label;

  @override
  ConsumerState<ConnectionScreen> createState() => _ConnectionScreenState();
}

final class _ConnectionScreenState extends ConsumerState<ConnectionScreen> {
  ConnectionPolicy? _draft;
  bool _applying = false;
  bool _refreshing = false;
  String? _actionError;

  String get _label {
    final value = widget.label?.trim() ?? '';
    return value.isEmpty ? widget.endpointId : value;
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final policyAsync = ref.watch(connectionPolicyProvider(widget.endpointId));
    final diagnosticsAsync = ref.watch(
      connectionDiagnosticsProvider(widget.endpointId),
    );
    final savedPolicy = policyAsync.valueOrNull?.policy;
    final effectiveDraft = _draft ?? savedPolicy?.deepCopy();
    final changed =
        savedPolicy != null &&
        effectiveDraft != null &&
        !_samePolicy(savedPolicy, effectiveDraft);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back to devices',
          onPressed: () => Navigator.maybePop(context),
          icon: const Icon(Icons.arrow_back_rounded),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              anyttyText(context, en: 'Connection', zh: '网络连接'),
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            Text(
              _label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.muted,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: 'Refresh connection diagnostics',
            onPressed: _refreshing || _applying ? null : _refresh,
            icon: _refreshing
                ? const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.refresh_rounded),
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        children: [
          _SectionLabel(
            label: anyttyText(context, en: 'CURRENT PATH', zh: '当前连接'),
          ),
          const SizedBox(height: 8),
          _ConnectionSummary(diagnostics: diagnosticsAsync),
          if (_actionError != null) ...[
            const SizedBox(height: 12),
            _InlineMessage(
              message: _actionError!,
              icon: Icons.error_outline_rounded,
              color: palette.danger,
              onClose: () => setState(() => _actionError = null),
            ),
          ],
          const SizedBox(height: 24),
          _SectionLabel(
            label: anyttyText(context, en: 'ROUTE PREFERENCE', zh: '路由偏好'),
          ),
          const SizedBox(height: 6),
          Text(
            anyttyText(
              context,
              en: 'Auto chooses the best available route.',
              zh: '自动模式会选择当前最合适的可用路由。',
            ),
            style: TextStyle(color: palette.muted, fontSize: 12, height: 1.4),
          ),
          const SizedBox(height: 10),
          if (policyAsync.isLoading && policyAsync.valueOrNull == null)
            const _PolicyLoading()
          else if (policyAsync.hasError && policyAsync.valueOrNull == null)
            _PolicyError(
              message: policyAsync.error.toString(),
              onRetry: _refresh,
            )
          else if (policyAsync.valueOrNull case final state?)
            _RoutePreferenceList(
              state: state,
              value: effectiveDraft!.routePreference,
              enabled: !_applying,
              onChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.routePreference = value,
              ),
            ),
          if (policyAsync.valueOrNull case final state?) ...[
            const SizedBox(height: 24),
            _SectionLabel(
              label: anyttyText(context, en: 'CLOUD PATH', zh: '云端路径'),
            ),
            const SizedBox(height: 10),
            _CloudPolicyControls(
              state: state,
              policy: effectiveDraft!,
              enabled: !_applying,
              onCloudChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.cloudRelayMode = value,
              ),
              onTransportChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.relayTransport = value,
              ),
            ),
          ],
          const SizedBox(height: 24),
          _SectionLabel(
            label: anyttyText(context, en: 'CONNECTION ROUTES', zh: '连接路由'),
          ),
          const SizedBox(height: 8),
          Material(
            color: palette.surface,
            clipBehavior: Clip.antiAlias,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
              side: BorderSide(color: palette.border),
            ),
            child: ListTile(
              leading: const Icon(Icons.alt_route_rounded),
              title: Text(anyttyText(context, en: 'Manage routes', zh: '管理路由')),
              subtitle: Text(
                anyttyText(
                  context,
                  en: 'Add, order, and test routes',
                  zh: '添加、排序和测试连接路由',
                ),
              ),
              trailing: Icon(Icons.chevron_right_rounded, color: palette.faint),
              onTap: () => context.push(
                Uri(
                  path: '/routes/${Uri.encodeComponent(widget.endpointId)}',
                  queryParameters: {'label': _label},
                ).toString(),
              ),
            ),
          ),
          const SizedBox(height: 24),
          _DiagnosticsSection(
            diagnostics: diagnosticsAsync,
            onCopyRedacted: _copyRedactedDiagnostics,
          ),
        ],
      ),
      bottomNavigationBar: _ConnectionFooter(
        changed: changed,
        applying: _applying,
        enabled: savedPolicy != null,
        onApply: effectiveDraft == null ? null : () => _apply(effectiveDraft),
      ),
    );
  }

  void _updateDraft(
    ConnectionPolicy current,
    void Function(ConnectionPolicy next) update,
  ) {
    final next = current.deepCopy();
    update(next);
    setState(() {
      _draft = next;
      _actionError = null;
    });
  }

  Future<void> _refresh() async {
    if (_refreshing || _applying) return;
    setState(() {
      _refreshing = true;
      _actionError = null;
    });
    try {
      await Future.wait<Object>([
        ref.refresh(connectionPolicyProvider(widget.endpointId).future),
        ref.refresh(connectionDiagnosticsProvider(widget.endpointId).future),
      ]);
      if (mounted) setState(() => _draft = null);
    } catch (_) {
      // Each independent provider renders its own actionable error state.
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  Future<void> _apply(ConnectionPolicy draft) async {
    if (_applying) return;
    setState(() {
      _applying = true;
      _actionError = null;
    });
    var policySaved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      await ConnectionRepository(runtime).applyPolicy(widget.endpointId, draft);
      policySaved = true;
      if (mounted) setState(() => _draft = null);
      ref.invalidate(connectionPolicyProvider(widget.endpointId));

      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpointId));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Connection policy applied')),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _actionError = policySaved
            ? 'Policy was saved, but reconnect failed: $error'
            : error.toString();
      });
    } finally {
      if (mounted) setState(() => _applying = false);
    }
  }

  Future<void> _copyRedactedDiagnostics() async {
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final current = ref
          .read(connectionDiagnosticsProvider(widget.endpointId))
          .valueOrNull;
      final report = runtime.buildRedactedDiagnostics(
        endpointId: widget.endpointId,
        session: current?.session,
      );
      await Clipboard.setData(ClipboardData(text: report));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Redacted diagnostics copied')),
      );
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not copy diagnostics: $error')),
      );
    }
  }
}

final class _ConnectionSummary extends StatelessWidget {
  const _ConnectionSummary({required this.diagnostics});

  final AsyncValue<EndpointConnectionDiagnostics> diagnostics;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final value = diagnostics.valueOrNull;
    final snapshot = value?.snapshot;
    final connected = snapshot?.connected ?? false;
    final loading = diagnostics.isLoading && value == null;
    final route = loading
        ? anyttyText(context, en: 'Connecting', zh: '正在连接')
        : snapshot == null
        ? anyttyText(context, en: 'Unavailable', zh: '不可用')
        : _routeLabel(snapshot.routeKind);
    final status = loading
        ? anyttyText(context, en: 'Reading connection state', zh: '正在读取连接状态')
        : snapshot == null
        ? anyttyText(
            context,
            en: 'Connection state could not be read',
            zh: '无法读取连接状态',
          )
        : connected
        ? _pathLabel(snapshot.observedPath)
        : anyttyText(context, en: 'Not connected', zh: '未连接');

    return Container(
      constraints: const BoxConstraints(minHeight: 74),
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            child: Row(
              children: [
                Container(
                  width: 42,
                  height: 42,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: connected
                        ? palette.accent.withValues(alpha: 0.13)
                        : palette.surfaceRaised,
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: loading
                      ? const SizedBox.square(
                          dimension: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Icon(
                          connected
                              ? Icons.route_rounded
                              : Icons.cloud_off_outlined,
                          color: connected ? palette.accent : palette.muted,
                          size: 22,
                        ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        route,
                        style: const TextStyle(
                          fontSize: 17,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        status,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.muted, fontSize: 12),
                      ),
                    ],
                  ),
                ),
                if (connected && snapshot != null)
                  Column(
                    crossAxisAlignment: CrossAxisAlignment.end,
                    children: [
                      Text(
                        _formatLatency(snapshot.roundTripNanos.toInt()),
                        style: const TextStyle(
                          fontFamily: 'JetBrainsMonoNerd',
                          fontSize: 12,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      Text(
                        'LATENCY',
                        style: TextStyle(
                          color: palette.muted,
                          fontSize: 8,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ],
                  ),
              ],
            ),
          ),
          if (diagnostics.hasError)
            _InlineMessage(
              message: diagnostics.error.toString(),
              icon: Icons.info_outline_rounded,
              color: palette.warning,
            ),
        ],
      ),
    );
  }
}

final class _RoutePreferenceList extends StatelessWidget {
  const _RoutePreferenceList({
    required this.state,
    required this.value,
    required this.enabled,
    required this.onChanged,
  });

  final ConnectionPolicyState state;
  final EndpointRoutePreference value;
  final bool enabled;
  final ValueChanged<EndpointRoutePreference> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final normalized = _normalizeRoutePreference(value);
    final options = [
      const _RouteOption(
        value: EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO,
        routeKind: null,
        icon: Icons.auto_awesome_outlined,
        label: 'Auto',
        description: 'Race all routes allowed by the planner',
      ),
      const _RouteOption(
        value: EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT,
        routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
        icon: Icons.wifi_tethering_rounded,
        label: 'Direct',
        description: 'Connect without an SSH or Cloud relay',
      ),
      const _RouteOption(
        value: EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH,
        routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH,
        icon: Icons.key_rounded,
        label: 'SSH',
        description: 'Use a configured SSH route',
      ),
      const _RouteOption(
        value: EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD,
        routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
        icon: Icons.cloud_outlined,
        label: 'Cloud',
        description: 'Use the managed Cloud path',
      ),
    ];

    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          for (var index = 0; index < options.length; index++) ...[
            _RoutePreferenceRow(
              option: options[index],
              selected: normalized == options[index].value,
              availability: options[index].routeKind == null
                  ? null
                  : _availability(state, options[index].routeKind!),
              enabled: enabled,
              onTap: () => onChanged(options[index].value),
            ),
            if (index != options.length - 1)
              Divider(height: 1, color: palette.border),
          ],
        ],
      ),
    );
  }
}

final class _RouteOption {
  const _RouteOption({
    required this.value,
    required this.routeKind,
    required this.icon,
    required this.label,
    required this.description,
  });

  final EndpointRoutePreference value;
  final ConnectionRouteKind? routeKind;
  final IconData icon;
  final String label;
  final String description;
}

final class _RoutePreferenceRow extends StatelessWidget {
  const _RoutePreferenceRow({
    required this.option,
    required this.selected,
    required this.availability,
    required this.enabled,
    required this.onTap,
  });

  final _RouteOption option;
  final bool selected;
  final ConnectionPolicyRouteAvailability? availability;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final available =
        option.routeKind == null || availability?.available == true;
    final interactive = enabled && available;
    final color = interactive ? palette.text : palette.faint;
    return Semantics(
      button: true,
      selected: selected,
      enabled: interactive,
      label: '${option.label} connection route',
      child: InkWell(
        onTap: interactive ? onTap : null,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 64),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
            child: Row(
              children: [
                Icon(option.icon, size: 20, color: color),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        option.label,
                        style: TextStyle(
                          color: color,
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        available
                            ? option.description
                            : _availabilityReason(availability?.reason),
                        style: TextStyle(
                          color: interactive ? palette.muted : palette.faint,
                          fontSize: 11,
                          height: 1.3,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Icon(
                  selected
                      ? Icons.radio_button_checked_rounded
                      : Icons.radio_button_unchecked_rounded,
                  size: 20,
                  color: selected && interactive
                      ? palette.accent
                      : palette.faint,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _CloudPolicyControls extends StatelessWidget {
  const _CloudPolicyControls({
    required this.state,
    required this.policy,
    required this.enabled,
    required this.onCloudChanged,
    required this.onTransportChanged,
  });

  final ConnectionPolicyState state;
  final ConnectionPolicy policy;
  final bool enabled;
  final ValueChanged<ManagedWebRTCRelayMode> onCloudChanged;
  final ValueChanged<ManagedWebRTCRelayTransport> onTransportChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final cloudAvailable =
        _availability(
          state,
          ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
        )?.available ==
        true;
    final route = _normalizeRoutePreference(policy.routePreference);
    final controlsEnabled =
        enabled &&
        cloudAvailable &&
        (route == EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO ||
            route ==
                EndpointRoutePreference
                    .ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD);
    final cloudMode = _normalizeCloudMode(policy.cloudRelayMode);
    final relayEnabled =
        controlsEnabled &&
        cloudMode != ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_DIRECT;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _ControlLabel(
            label: 'Path',
            hint: controlsEnabled
                ? 'Choose how Cloud reaches the endpoint'
                : cloudAvailable
                ? 'Select Auto or Cloud to adjust'
                : _availabilityReason(
                    _availability(
                      state,
                      ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
                    )?.reason,
                  ),
          ),
          const SizedBox(height: 8),
          SegmentedButton<ManagedWebRTCRelayMode>(
            showSelectedIcon: false,
            segments: const [
              ButtonSegment(
                value: ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO,
                label: Text('Auto'),
              ),
              ButtonSegment(
                value: ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_DIRECT,
                label: Text('P2P'),
              ),
              ButtonSegment(
                value:
                    ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
                label: Text('Relay'),
              ),
            ],
            selected: {cloudMode},
            onSelectionChanged: controlsEnabled
                ? (value) => onCloudChanged(value.single)
                : null,
          ),
          const SizedBox(height: 18),
          _ControlLabel(
            label: 'Relay transport',
            hint: relayEnabled
                ? 'Auto prefers the transport selected by Go'
                : 'Relay transport is not active for this path',
          ),
          const SizedBox(height: 8),
          SegmentedButton<ManagedWebRTCRelayTransport>(
            showSelectedIcon: false,
            segments: const [
              ButtonSegment(
                value: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO,
                label: Text('Auto'),
              ),
              ButtonSegment(
                value: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_UDP,
                label: Text('UDP'),
              ),
              ButtonSegment(
                value: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
                label: Text('TCP'),
              ),
            ],
            selected: {_normalizeRelayTransport(policy.relayTransport)},
            onSelectionChanged: relayEnabled
                ? (value) => onTransportChanged(value.single)
                : null,
          ),
        ],
      ),
    );
  }
}

final class _ControlLabel extends StatelessWidget {
  const _ControlLabel({required this.label, required this.hint});

  final String label;
  final String hint;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 2),
        Text(
          hint,
          style: TextStyle(color: palette.muted, fontSize: 11, height: 1.3),
        ),
      ],
    );
  }
}

final class _DiagnosticsSection extends StatelessWidget {
  const _DiagnosticsSection({
    required this.diagnostics,
    required this.onCopyRedacted,
  });

  final AsyncValue<EndpointConnectionDiagnostics> diagnostics;
  final VoidCallback onCopyRedacted;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final value = diagnostics.valueOrNull;
    final snapshot = value?.snapshot;
    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: ExpansionTile(
        key: const ValueKey('connection-diagnostics'),
        tilePadding: const EdgeInsets.symmetric(horizontal: 12),
        childrenPadding: EdgeInsets.zero,
        shape: const Border(),
        collapsedShape: const Border(),
        leading: const Icon(Icons.monitor_heart_outlined, size: 20),
        title: Text(
          anyttyText(context, en: 'Technical details', zh: '技术详情'),
          style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600),
        ),
        subtitle: Text(
          snapshot == null
              ? anyttyText(
                  context,
                  en: 'Transport details unavailable',
                  zh: '传输详情不可用',
                )
              : anyttyText(
                  context,
                  en: 'Addresses, traffic, and connection generation',
                  zh: '地址、流量与连接代次',
                ),
          style: TextStyle(color: palette.muted, fontSize: 11),
        ),
        children: [
          Divider(height: 1, color: palette.border),
          if (snapshot == null)
            Padding(
              padding: const EdgeInsets.all(14),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(
                  diagnostics.isLoading
                      ? 'Reading diagnostics...'
                      : diagnostics.error?.toString() ?? 'Not provided',
                  style: TextStyle(color: palette.muted, fontSize: 12),
                ),
              ),
            )
          else ...[
            _DiagnosticRow(label: 'Route ID', value: _text(snapshot.routeId)),
            _DiagnosticRow(
              label: 'Candidate pair',
              value: _compactIdentifier(snapshot.candidatePairId),
              copyValue: snapshot.candidatePairId,
            ),
            _DiagnosticRow(
              label: 'Generation',
              value: value!.session.generation.toString(),
            ),
            _DiagnosticRow(
              label: 'Selection reason',
              value: _text(snapshot.selectionReason),
            ),
            _DiagnosticRow(
              label: 'Local candidate',
              value:
                  '${_address(snapshot.localIp, snapshot.localPort)} (${_candidateLabel(snapshot.localCandidateType)}, ${_transportLabel(snapshot.localProtocol)})',
            ),
            if (snapshot.localRelatedIp.trim().isNotEmpty)
              _DiagnosticRow(
                label: 'Local base',
                value: _address(
                  snapshot.localRelatedIp,
                  snapshot.localRelatedPort,
                ),
              ),
            _DiagnosticRow(
              label: 'Remote candidate',
              value:
                  '${_address(snapshot.remoteIp, snapshot.remotePort)} (${_candidateLabel(snapshot.remoteCandidateType)}, ${_transportLabel(snapshot.remoteProtocol)})',
            ),
            if (snapshot.remoteRelatedIp.trim().isNotEmpty)
              _DiagnosticRow(
                label: 'Remote base',
                value: _address(
                  snapshot.remoteRelatedIp,
                  snapshot.remoteRelatedPort,
                ),
              ),
            _DiagnosticRow(
              label: 'Relay transport',
              value: _transportLabel(snapshot.relayTransport),
            ),
            _DiagnosticRow(
              label: 'Network class',
              value: _text(snapshot.networkClass),
            ),
            _DiagnosticRow(
              label: 'Packets / loss events',
              value:
                  '${snapshot.packetsSent.toString()} / ${snapshot.lossEvents.toString()}',
            ),
            _DiagnosticRow(
              label: 'Sampled',
              value: _formatSampleTime(snapshot.sampledAtUnixNano.toInt()),
            ),
            if (_sameNatMapping(snapshot))
              Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(Icons.hub_outlined, size: 17, color: palette.accent),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        'Both public candidates match. The peers are likely behind the same NAT and connected through router mappings.',
                        style: TextStyle(
                          color: palette.muted,
                          fontSize: 11,
                          height: 1.4,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
          ],
          Padding(
            padding: const EdgeInsets.all(12),
            child: SizedBox(
              width: double.infinity,
              height: 48,
              child: OutlinedButton.icon(
                key: const ValueKey('copy-redacted-diagnostics'),
                onPressed: onCopyRedacted,
                icon: const Icon(Icons.copy_all_rounded, size: 18),
                label: const Text('Copy redacted report'),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _DiagnosticRow extends StatelessWidget {
  const _DiagnosticRow({
    required this.label,
    required this.value,
    this.copyValue,
  });

  final String label;
  final String value;
  final String? copyValue;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      constraints: const BoxConstraints(minHeight: 48),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 112,
            child: Text(
              label,
              style: TextStyle(color: palette.muted, fontSize: 11),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.right,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 11,
                height: 1.35,
              ),
            ),
          ),
          if (copyValue?.trim().isNotEmpty == true) ...[
            const SizedBox(width: 4),
            IconButton(
              tooltip: 'Copy $label',
              constraints: const BoxConstraints.tightFor(width: 48, height: 48),
              onPressed: () {
                Clipboard.setData(ClipboardData(text: copyValue!));
                ScaffoldMessenger.of(context)
                    .showSnackBar(SnackBar(content: Text('$label copied')));
              },
              icon: const Icon(Icons.copy_rounded, size: 16),
            ),
          ],
        ],
      ),
    );
  }
}

final class _ConnectionFooter extends StatelessWidget {
  const _ConnectionFooter({
    required this.changed,
    required this.applying,
    required this.enabled,
    required this.onApply,
  });

  final bool changed;
  final bool applying;
  final bool enabled;
  final VoidCallback? onApply;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(16, 10, 16, 10),
        decoration: BoxDecoration(
          color: palette.background,
          border: Border(top: BorderSide(color: palette.border)),
        ),
        child: SizedBox(
          height: 48,
          child: FilledButton.icon(
            onPressed: enabled && changed && !applying ? onApply : null,
            icon: applying
                ? const SizedBox.square(
                    dimension: 17,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.sync_rounded, size: 18),
            label: Text(applying ? 'Applying...' : 'Apply and reconnect'),
          ),
        ),
      ),
    );
  }
}

final class _PolicyLoading extends StatelessWidget {
  const _PolicyLoading();

  @override
  Widget build(BuildContext context) => const SizedBox(
    height: 92,
    child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
  );
}

final class _PolicyError extends StatelessWidget {
  const _PolicyError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        children: [
          Icon(Icons.error_outline_rounded, color: palette.danger),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.muted, fontSize: 12),
            ),
          ),
          TextButton(onPressed: onRetry, child: const Text('Retry')),
        ],
      ),
    );
  }
}

final class _InlineMessage extends StatelessWidget {
  const _InlineMessage({
    required this.message,
    required this.icon,
    required this.color,
    this.onClose,
  });

  final String message;
  final IconData icon;
  final Color color;
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
      color: color.withValues(alpha: 0.09),
      child: Row(
        children: [
          Icon(icon, size: 17, color: color),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.text, fontSize: 11, height: 1.3),
            ),
          ),
          if (onClose != null)
            IconButton(
              tooltip: 'Dismiss',
              onPressed: onClose,
              icon: const Icon(Icons.close_rounded, size: 17),
            ),
        ],
      ),
    );
  }
}

final class _SectionLabel extends StatelessWidget {
  const _SectionLabel({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Text(
      label,
      style: TextStyle(
        color: palette.muted,
        fontSize: 11,
        fontWeight: FontWeight.w700,
      ),
    );
  }
}

ConnectionPolicyRouteAvailability? _availability(
  ConnectionPolicyState state,
  ConnectionRouteKind kind,
) {
  for (final route in state.routes) {
    if (route.routeKind == kind) return route;
  }
  return null;
}

EndpointRoutePreference _normalizeRoutePreference(
  EndpointRoutePreference value,
) =>
    value == EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT ||
        value == EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH ||
        value == EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD
    ? value
    : EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO;

ManagedWebRTCRelayMode _normalizeCloudMode(ManagedWebRTCRelayMode value) =>
    value == ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_DIRECT ||
        value == ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY
    ? value
    : ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO;

ManagedWebRTCRelayTransport _normalizeRelayTransport(
  ManagedWebRTCRelayTransport value,
) =>
    value == ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_UDP ||
        value == ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP
    ? value
    : ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO;

bool _samePolicy(ConnectionPolicy left, ConnectionPolicy right) =>
    _normalizeRoutePreference(left.routePreference) ==
        _normalizeRoutePreference(right.routePreference) &&
    _normalizeCloudMode(left.cloudRelayMode) ==
        _normalizeCloudMode(right.cloudRelayMode) &&
    _normalizeRelayTransport(left.relayTransport) ==
        _normalizeRelayTransport(right.relayTransport);

String _availabilityReason(ConnectionPolicyAvailabilityReason? reason) {
  if (reason ==
      ConnectionPolicyAvailabilityReason
          .CONNECTION_POLICY_AVAILABILITY_REASON_ROUTE_DISABLED) {
    return 'Disabled in the saved route';
  }
  if (reason ==
      ConnectionPolicyAvailabilityReason
          .CONNECTION_POLICY_AVAILABILITY_REASON_PLATFORM_UNSUPPORTED) {
    return 'Not supported on this endpoint';
  }
  if (reason ==
      ConnectionPolicyAvailabilityReason
          .CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE) {
    return 'Required credential is unavailable';
  }
  if (reason ==
      ConnectionPolicyAvailabilityReason
          .CONNECTION_POLICY_AVAILABILITY_REASON_CLOUD_UNAVAILABLE) {
    return 'Cloud service is unavailable';
  }
  return 'Route is not configured';
}

String _routeLabel(ConnectionRouteKind kind) {
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL) return 'Local';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT) return 'Direct';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH) return 'SSH';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD) return 'Cloud';
  return 'Unknown route';
}

String _pathLabel(ConnectionObservedPath path) {
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT) {
    return 'Direct peer path';
  }
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return 'Single relay path';
  }
  return 'Connected';
}

String _candidateLabel(ConnectionCandidateType value) {
  if (value == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_HOST) {
    return 'host';
  }
  if (value ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE) {
    return 'srflx';
  }
  if (value ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE) {
    return 'prflx';
  }
  if (value == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_RELAY) {
    return 'relay';
  }
  return 'unknown';
}

String _transportLabel(ConnectionTransport value) {
  if (value == ConnectionTransport.CONNECTION_TRANSPORT_UDP) return 'UDP';
  if (value == ConnectionTransport.CONNECTION_TRANSPORT_TCP) return 'TCP';
  return 'Not provided';
}

String _address(String ip, int port) {
  final value = ip.trim();
  if (value.isEmpty) return 'Not provided';
  if (port <= 0) return value;
  return value.contains(':') ? '[$value]:$port' : '$value:$port';
}

String _text(String value) => value.trim().isEmpty ? 'Not provided' : value;

String _compactIdentifier(String value) {
  final normalized = value.trim();
  if (normalized.isEmpty) return 'Not provided';
  if (normalized.length <= 36) return normalized;
  return '${normalized.substring(0, 18)}...${normalized.substring(normalized.length - 12)}';
}

String _formatLatency(int nanos) {
  if (nanos <= 0) return '--';
  final milliseconds = nanos / 1000000;
  return milliseconds >= 10
      ? '${milliseconds.round()} ms'
      : '${milliseconds.toStringAsFixed(1)} ms';
}

String _formatSampleTime(int unixNanos) {
  if (unixNanos <= 0) return 'Not provided';
  final value = DateTime.fromMillisecondsSinceEpoch(unixNanos ~/ 1000000)
      .toLocal();
  String two(int part) => part.toString().padLeft(2, '0');
  return '${value.year}-${two(value.month)}-${two(value.day)} '
      '${two(value.hour)}:${two(value.minute)}:${two(value.second)}';
}

bool _sameNatMapping(ConnectionSnapshot snapshot) {
  final reflexive =
      snapshot.localCandidateType ==
          ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE ||
      snapshot.remoteCandidateType ==
          ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE;
  return snapshot.observedPath ==
          ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT &&
      reflexive &&
      snapshot.localIp.trim().isNotEmpty &&
      snapshot.localIp.trim() == snapshot.remoteIp.trim();
}
