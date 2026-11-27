part of 'route_management_screen.dart';


final class _RouteRow extends StatelessWidget {
  const _RouteRow({
    required this.route,
    required this.busy,
    required this.disabledByOtherOperation,
    required this.canMoveUp,
    required this.canMoveDown,
    required this.showReorderControls,
    required this.onToggle,
    required this.onMoveUp,
    required this.onMoveDown,
    required this.onActions,
  });

  final EndpointRouteConfigV1 route;
  final bool busy;
  final bool disabledByOtherOperation;
  final bool canMoveUp;
  final bool canMoveDown;
  final bool showReorderControls;
  final ValueChanged<bool> onToggle;
  final VoidCallback onMoveUp;
  final VoidCallback onMoveDown;
  final VoidCallback onActions;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final kind = endpointRouteKind(route);
    final enabled = !busy && !disabledByOtherOperation;
    return Container(
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(8, 8, 4, 8),
            child: Row(
              children: [
                Switch.adaptive(
                  value: route.enabled,
                  onChanged: enabled ? onToggle : null,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _routeTitle(route),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: route.enabled ? palette.text : palette.faint,
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${_routeKindLabel(kind)} · ${route.routeId}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(color: palette.muted, fontSize: 11),
                      ),
                    ],
                  ),
                ),
                if (showReorderControls) ...[
                  IconButton(
                    tooltip: 'Move ${_routeTitle(route)} up',
                    onPressed: enabled && canMoveUp ? onMoveUp : null,
                    icon: const Icon(Icons.keyboard_arrow_up_rounded),
                  ),
                  IconButton(
                    tooltip: 'Move ${_routeTitle(route)} down',
                    onPressed: enabled && canMoveDown ? onMoveDown : null,
                    icon: const Icon(Icons.keyboard_arrow_down_rounded),
                  ),
                ],
                IconButton(
                  tooltip: 'More actions for ${_routeTitle(route)}',
                  onPressed: enabled ? onActions : null,
                  icon: const Icon(Icons.more_horiz_rounded),
                ),
              ],
            ),
          ),
          Divider(height: 1, color: palette.border),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
            child: Row(
              children: [
                Icon(_routeKindIcon(kind), size: 17, color: palette.muted),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    _routeSummary(route),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      height: 1.3,
                    ),
                  ),
                ),
                Text(
                  route.hasPriority() ? 'P${route.priority}' : 'AUTO',
                  style: TextStyle(
                    color: palette.faint,
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
          if (busy) const LinearProgressIndicator(minHeight: 2),
        ],
      ),
    );
  }
}

final class _RouteEditorPanel extends StatelessWidget {
  const _RouteEditorPanel({required this.children});

  final List<Widget> children;

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
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: children,
      ),
    );
  }
}

final class _RouteSectionLabel extends StatelessWidget {
  const _RouteSectionLabel({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Text(
    label,
    style: TextStyle(
      color: AnyttyPalette.of(context).muted,
      fontSize: 11,
      fontWeight: FontWeight.w700,
    ),
  );
}

final class _RouteInlineError extends StatelessWidget {
  const _RouteInlineError({required this.message, required this.onClose});

  final String message;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
      color: palette.danger.withValues(alpha: 0.09),
      child: Row(
        children: [
          Icon(Icons.error_outline_rounded, color: palette.danger, size: 18),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              style: TextStyle(color: palette.text, fontSize: 12),
            ),
          ),
          IconButton(
            tooltip: 'Dismiss error',
            onPressed: onClose,
            icon: const Icon(Icons.close_rounded, size: 18),
          ),
        ],
      ),
    );
  }
}

final class _RouteLoadError extends StatelessWidget {
  const _RouteLoadError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.route_outlined, size: 30),
          const SizedBox(height: 12),
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 16),
          OutlinedButton.icon(
            onPressed: onRetry,
            icon: const Icon(Icons.refresh_rounded),
            label: const Text('Retry'),
          ),
        ],
      ),
    ),
  );
}

enum _RouteAction { test, edit, provisionSsh, remove }

EndpointConfigV1? _findEndpoint(EndpointRegistryV1? registry, String id) {
  if (registry == null) return null;
  for (final endpoint in registry.endpoints) {
    if (endpoint.endpointId == id) return endpoint.deepCopy();
  }
  return null;
}

String _routeTitle(EndpointRouteConfigV1 route) {
  final displayName = route.displayName.trim();
  return displayName.isEmpty
      ? _routeKindLabel(endpointRouteKind(route))
      : displayName;
}

String _routeKindLabel(EndpointRouteKind kind) {
  switch (kind) {
    case EndpointRouteKind.direct:
      return 'Direct';
    case EndpointRouteKind.ssh:
      return 'SSH';
    case EndpointRouteKind.cloud:
      return 'Cloud';
    case EndpointRouteKind.local:
      return 'Local';
    case EndpointRouteKind.unknown:
      return 'Unknown';
  }
}

IconData _routeKindIcon(EndpointRouteKind kind) {
  switch (kind) {
    case EndpointRouteKind.direct:
      return Icons.wifi_tethering_rounded;
    case EndpointRouteKind.ssh:
      return Icons.key_rounded;
    case EndpointRouteKind.cloud:
      return Icons.cloud_outlined;
    case EndpointRouteKind.local:
      return Icons.cable_rounded;
    case EndpointRouteKind.unknown:
      return Icons.help_outline_rounded;
  }
}

String _routeSummary(EndpointRouteConfigV1 route) {
  switch (endpointRouteKind(route)) {
    case EndpointRouteKind.direct:
      final values = route.directWebrtcTcp.signalingAddresses;
      return values.isEmpty ? 'No setup address' : values.join(', ');
    case EndpointRouteKind.ssh:
      final ssh = route.sshWebrtcTcp;
      final host = ssh.host.trim().isEmpty ? 'Unconfigured' : ssh.host.trim();
      final port = ssh.port == 0 ? 22 : ssh.port;
      return '${ssh.user.trim().isEmpty ? 'user' : ssh.user.trim()}@$host:$port';
    case EndpointRouteKind.cloud:
      return 'Managed by AnyTTY Cloud';
    case EndpointRouteKind.local:
      return route.localUnix.socket.trim().isEmpty
          ? 'Local socket'
          : route.localUnix.socket;
    case EndpointRouteKind.unknown:
      return 'Unsupported route type';
  }
}

String _routeErrorText(Object error) {
  if (error is NativeSessionException) {
    final code = error.code?.name.toLowerCase() ?? '';
    if (code.contains('credential')) {
      return 'This route needs a credential before it can connect.';
    }
    if (code.contains('authorization') ||
        code.contains('unauthenticated') ||
        code.contains('forbidden')) {
      return 'Authorization for this route is no longer valid.';
    }
    if (code.contains('unavailable') || code.contains('stale')) {
      return 'This route is currently unreachable. Check both networks and try again.';
    }
  }
  if (error is AnyttyOperationException) {
    final message = error.message.toLowerCase();
    if (message.contains('credential')) {
      return 'This route needs a credential before it can connect.';
    }
    if (message.contains('route') || message.contains('config')) {
      return 'Go rejected this route configuration. Review the fields and try again.';
    }
  }
  return 'AnyTTY could not complete this route operation. Try again.';
}
