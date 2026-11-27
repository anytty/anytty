part of 'device_list_screen.dart';

final class _DeviceRow extends ConsumerWidget {
  const _DeviceRow({
    required this.endpoint,
    required this.isDefault,
    required this.searchQuery,
    required this.onActions,
  });

  final EndpointConfigV1 endpoint;
  final bool isDefault;
  final String searchQuery;
  final VoidCallback onActions;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AnyttyPalette.of(context);
    final label = endpoint.label.trim().isEmpty
        ? endpoint.endpointId
        : endpoint.label.trim();
    final authorizationRequired = _authorizationRequired(endpoint);
    final presence = _cloudPresenceFor(ref, endpoint);
    final visualState = _deviceVisualState(endpoint, presence);
    final routes = _routeLabels(context, endpoint);
    final statusLabel = _deviceStateLabel(context, visualState.kind);
    return Material(
      color: Colors.transparent,
      borderRadius: BorderRadius.circular(18),
      child: Ink(
        decoration: BoxDecoration(
          color: palette.surface,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(18),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(
                alpha: Theme.of(context).brightness == Brightness.dark
                    ? 0.16
                    : 0.045,
              ),
              blurRadius: 18,
              offset: const Offset(0, 5),
            ),
          ],
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(18),
          onTap: () {
            HapticFeedback.selectionClick();
            if (authorizationRequired) {
              _showPairingSheet(context, reauthorizeEndpoint: endpoint);
            } else {
              context.go(
                '/terminal/${Uri.encodeComponent(endpoint.endpointId)}'
                '?label=${Uri.encodeQueryComponent(label)}',
              );
            }
          },
          child: SizedBox(
            height: 88,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(12, 10, 2, 10),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SizedBox(
                    width: 40,
                    child: Column(
                      children: [
                        Container(
                          width: 40,
                          height: 40,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: palette.surfaceRaised,
                            borderRadius: BorderRadius.circular(14),
                          ),
                          child: Icon(
                            _platformIcon(endpoint.platform),
                            size: 20,
                            color: palette.text,
                          ),
                        ),
                        Expanded(
                          child: Container(
                            width: 1,
                            margin: const EdgeInsets.symmetric(vertical: 4),
                            color: palette.border,
                          ),
                        ),
                        Container(
                          width: 8,
                          height: 8,
                          decoration: BoxDecoration(
                            color: visualState.color(palette),
                            shape: BoxShape.circle,
                            border: Border.all(
                              color: palette.surface,
                              width: 2,
                            ),
                            boxShadow: [
                              BoxShadow(
                                color: visualState.color(palette),
                                spreadRadius: 1,
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Padding(
                      padding: const EdgeInsets.only(top: 1),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Expanded(
                                child: FuzzyHighlightText(
                                  label,
                                  query: searchQuery,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: const TextStyle(
                                    fontSize: 15,
                                    fontWeight: FontWeight.w700,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 8),
                              Icon(
                                visualState.icon,
                                size: 13,
                                color: visualState.color(palette),
                              ),
                              const SizedBox(width: 4),
                              Text(
                                statusLabel,
                                style: TextStyle(
                                  color: visualState.color(palette),
                                  fontSize: 10,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 3),
                          FuzzyHighlightText(
                            [
                              endpoint.platform.isEmpty
                                  ? anyttyText(
                                      context,
                                      en: 'Unknown platform',
                                      zh: '未知平台',
                                    )
                                  : endpoint.platform,
                              if (routes.isNotEmpty) routes,
                            ].join('  ·  '),
                            query: searchQuery,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 11,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          const Spacer(),
                          Row(
                            children: [
                              if (isDefault) ...[
                                Icon(
                                  Icons.star_rounded,
                                  size: 13,
                                  color: palette.warning,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  anyttyText(
                                    context,
                                    en: 'Default',
                                    zh: '默认设备',
                                  ),
                                  style: TextStyle(
                                    color: palette.muted,
                                    fontSize: 10,
                                    fontWeight: FontWeight.w600,
                                  ),
                                ),
                                const SizedBox(width: 10),
                              ],
                              Expanded(
                                child: FuzzyHighlightText(
                                  endpoint.endpointId,
                                  query: searchQuery,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    color: palette.faint,
                                    fontFamily: 'JetBrainsMonoNerd',
                                    fontSize: 9,
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: 'More actions for $label',
                    onPressed: onActions,
                    icon: Icon(Icons.more_horiz_rounded, color: palette.muted),
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

bool _hasEnabledCloudRoute(EndpointConfigV1 endpoint) {
  return endpoint.routes.any(
    (route) =>
        route.enabled &&
        route.whichRoute() == EndpointRouteConfigV1_Route.managedWebrtc,
  );
}

bool _authorizationRequired(EndpointConfigV1 endpoint) {
  final remoteRoutes = endpoint.routes.where(
    (route) =>
        route.enabled &&
        route.whichRoute() != EndpointRouteConfigV1_Route.localUnix &&
        route.whichRoute() != EndpointRouteConfigV1_Route.notSet,
  );
  return remoteRoutes.isNotEmpty &&
      remoteRoutes.every((route) => route.credentialRef.trim().isEmpty);
}

enum _DeviceStateKind {
  online,
  ready,
  offline,
  paused,
  authorizationRequired,
  checking,
  unavailable,
}

final class _DeviceVisualState {
  const _DeviceVisualState({
    required this.kind,
    required this.online,
    required this.icon,
  });

  final _DeviceStateKind kind;
  final bool online;
  final IconData icon;

  Color color(AnyttyPalette palette) => switch (kind) {
    _DeviceStateKind.online || _DeviceStateKind.ready => palette.success,
    _DeviceStateKind.authorizationRequired => palette.warning,
    _DeviceStateKind.checking => palette.muted,
    _ => palette.faint,
  };
}

_DeviceVisualState _deviceVisualState(
  EndpointConfigV1 endpoint,
  AsyncValue<EndpointCloudPresenceGetResult>? presence,
) {
  if (_authorizationRequired(endpoint)) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.authorizationRequired,
      online: false,
      icon: Icons.shield_outlined,
    );
  }
  if (!endpoint.enabled) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.paused,
      online: false,
      icon: Icons.pause_circle_outline_rounded,
    );
  }
  if (presence == null) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.ready,
      online: true,
      icon: Icons.check_circle_outline_rounded,
    );
  }
  if (presence.isLoading && presence.valueOrNull == null) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.checking,
      online: false,
      icon: Icons.sync_rounded,
    );
  }
  if (presence.hasError && presence.valueOrNull == null) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.unavailable,
      online: false,
      icon: Icons.cloud_off_outlined,
    );
  }
  if (presence.valueOrNull?.online == true) {
    return const _DeviceVisualState(
      kind: _DeviceStateKind.online,
      online: true,
      icon: Icons.check_circle_outline_rounded,
    );
  }
  return const _DeviceVisualState(
    kind: _DeviceStateKind.offline,
    online: false,
    icon: Icons.cloud_off_outlined,
  );
}

String _deviceStateLabel(BuildContext context, _DeviceStateKind kind) =>
    switch (kind) {
      _DeviceStateKind.online => anyttyText(context, en: 'Online', zh: '在线'),
      _DeviceStateKind.ready => anyttyText(context, en: 'Ready', zh: '就绪'),
      _DeviceStateKind.offline => anyttyText(context, en: 'Offline', zh: '离线'),
      _DeviceStateKind.paused => anyttyText(context, en: 'Paused', zh: '已暂停'),
      _DeviceStateKind.authorizationRequired => anyttyText(
        context,
        en: 'Authorization required',
        zh: '需授权',
      ),
      _DeviceStateKind.checking => anyttyText(
        context,
        en: 'Checking',
        zh: '检查中',
      ),
      _DeviceStateKind.unavailable => anyttyText(
        context,
        en: 'Unavailable',
        zh: '不可用',
      ),
    };

String _routeLabels(BuildContext context, EndpointConfigV1 endpoint) {
  final configured = <EndpointRouteConfigV1_Route>{};
  final enabled = <EndpointRouteConfigV1_Route>{};
  for (final route in endpoint.routes) {
    final kind = route.whichRoute();
    if (kind == EndpointRouteConfigV1_Route.notSet) continue;
    configured.add(kind);
    if (route.enabled) enabled.add(kind);
  }
  final selected = switch (endpoint.selectionPolicy.routePreference) {
    EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT =>
      EndpointRouteConfigV1_Route.directWebrtcTcp,
    EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH =>
      EndpointRouteConfigV1_Route.sshWebrtcTcp,
    EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD =>
      EndpointRouteConfigV1_Route.managedWebrtc,
    _ => null,
  };
  if (selected != null) configured.add(selected);

  const order = [
    EndpointRouteConfigV1_Route.localUnix,
    EndpointRouteConfigV1_Route.directWebrtcTcp,
    EndpointRouteConfigV1_Route.sshWebrtcTcp,
    EndpointRouteConfigV1_Route.managedWebrtc,
  ];
  return [
    for (final kind in order)
      if (configured.contains(kind))
        _configuredRouteLabel(
          context,
          kind,
          enabled:
              enabled.contains(kind) && (selected == null || selected == kind),
        ),
  ].join('  ·  ');
}

String _configuredRouteLabel(
  BuildContext context,
  EndpointRouteConfigV1_Route kind, {
  required bool enabled,
}) {
  final label = switch (kind) {
    EndpointRouteConfigV1_Route.localUnix => anyttyText(
      context,
      en: 'Local',
      zh: '本地',
    ),
    EndpointRouteConfigV1_Route.directWebrtcTcp => anyttyText(
      context,
      en: 'Direct',
      zh: '直连',
    ),
    EndpointRouteConfigV1_Route.sshWebrtcTcp => 'SSH',
    EndpointRouteConfigV1_Route.managedWebrtc => 'Cloud',
    EndpointRouteConfigV1_Route.notSet => anyttyText(
      context,
      en: 'Route',
      zh: '线路',
    ),
  };
  if (enabled) return label;
  return anyttyText(context, en: '$label disabled', zh: '$label 已禁用');
}

Future<void> _showDeviceActionsSheet(
  BuildContext context, {
  required EndpointConfigV1 endpoint,
  required bool isDefault,
}) async {
  await showModalBottomSheet<void>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    useSafeArea: true,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (_) =>
        _DeviceActionsSheet(endpoint: endpoint, isDefault: isDefault),
  );
}
