part of 'connection_screen.dart';


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
    : ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP;

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
