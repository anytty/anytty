part of 'connection_screen.dart';


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
                ? 'TCP / UDP'
                : 'Relay transport is not active for this path',
          ),
          const SizedBox(height: 8),
          SegmentedButton<ManagedWebRTCRelayTransport>(
            showSelectedIcon: false,
            segments: const [
              ButtonSegment(
                value: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
                label: Text('TCP'),
              ),
              ButtonSegment(
                value: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_UDP,
                label: Text('UDP'),
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
