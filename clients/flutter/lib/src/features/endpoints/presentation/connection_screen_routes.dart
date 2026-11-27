part of 'connection_screen.dart';


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
