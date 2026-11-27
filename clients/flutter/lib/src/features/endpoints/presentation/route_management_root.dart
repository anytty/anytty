part of 'route_management_screen.dart';


final class RouteManagementScreen extends ConsumerStatefulWidget {
  const RouteManagementScreen({
    super.key,
    required this.endpointId,
    this.label,
  });

  final String endpointId;
  final String? label;

  @override
  ConsumerState<RouteManagementScreen> createState() =>
      _RouteManagementScreenState();
}

final class _RouteManagementScreenState
    extends ConsumerState<RouteManagementScreen> {
  String? _busyRouteId;
  String? _error;

  String get _label {
    final value = widget.label?.trim() ?? '';
    return value.isEmpty ? widget.endpointId : value;
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final registry = ref.watch(endpointRegistryProvider);
    final endpoint = _findEndpoint(registry.valueOrNull, widget.endpointId);
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back to connection',
          onPressed: () => Navigator.maybePop(context),
          icon: const Icon(Icons.arrow_back_rounded),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Routes',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
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
          PopupMenuButton<EndpointRouteKind>(
            tooltip: 'Add route',
            enabled: endpoint != null && _busyRouteId == null,
            icon: const Icon(Icons.add_rounded),
            onSelected: (kind) => _openEditor(kind: kind),
            itemBuilder: (context) => const [
              PopupMenuItem(
                value: EndpointRouteKind.direct,
                child: ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(Icons.wifi_tethering_rounded),
                  title: Text('Direct route'),
                ),
              ),
              PopupMenuItem(
                value: EndpointRouteKind.ssh,
                child: ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: Icon(Icons.key_rounded),
                  title: Text('SSH route'),
                ),
              ),
            ],
          ),
          IconButton(
            tooltip: 'Refresh routes',
            onPressed: registry.isLoading || _busyRouteId != null
                ? null
                : () => ref.invalidate(endpointRegistryProvider),
            icon: registry.isLoading
                ? const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.refresh_rounded),
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: registry.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, _) => _RouteLoadError(
          message: error.toString(),
          onRetry: () => ref.invalidate(endpointRegistryProvider),
        ),
        data: (value) {
          final current = _findEndpoint(value, widget.endpointId);
          if (current == null) {
            return _RouteLoadError(
              message: 'Endpoint is not configured',
              onRetry: () => ref.invalidate(endpointRegistryProvider),
            );
          }
          final routes = orderedEndpointRoutes(current);
          final enabledCount = routes.where((route) => route.enabled).length;
          return ListView(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
            children: [
              Text(
                'Routes are evaluated by the Go connection planner. Order controls priority when a route is eligible.',
                style: TextStyle(
                  color: palette.muted,
                  fontSize: 12,
                  height: 1.45,
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 12),
                _RouteInlineError(
                  message: _error!,
                  onClose: () => setState(() => _error = null),
                ),
              ],
              const SizedBox(height: 16),
              for (var index = 0; index < routes.length; index++) ...[
                _RouteRow(
                  route: routes[index],
                  busy: _busyRouteId == routes[index].routeId,
                  disabledByOtherOperation:
                      _busyRouteId != null &&
                      _busyRouteId != routes[index].routeId,
                  canMoveUp: index > 0,
                  canMoveDown: index < routes.length - 1,
                  showReorderControls: routes.length > 1,
                  onToggle: (enabled) {
                    if (!enabled &&
                        routes[index].enabled &&
                        enabledCount == 1) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text(
                            'Keep at least one route enabled for this device',
                          ),
                        ),
                      );
                      return;
                    }
                    unawaited(_toggle(current, routes[index], enabled));
                  },
                  onMoveUp: () => _move(current, routes[index], -1),
                  onMoveDown: () => _move(current, routes[index], 1),
                  onActions: () => _showRouteActions(
                    current,
                    routes[index],
                    canRemove: routes.length > 1,
                  ),
                ),
                if (index != routes.length - 1) const SizedBox(height: 10),
              ],
            ],
          );
        },
      ),
    );
  }

  Future<void> _openEditor({
    EndpointRouteConfigV1? route,
    EndpointRouteKind? kind,
  }) async {
    final query = <String, String>{'label': _label};
    if (route != null) query['routeId'] = route.routeId;
    if (kind != null) query['kind'] = kind.name;
    final uri = Uri(
      path: '/routes/${Uri.encodeComponent(widget.endpointId)}/edit',
      queryParameters: query,
    );
    final changed = await context.push<bool>(uri.toString());
    if (changed == true) ref.invalidate(endpointRegistryProvider);
  }

  Future<void> _toggle(
    EndpointConfigV1 endpoint,
    EndpointRouteConfigV1 route,
    bool enabled,
  ) async {
    final updated = route.deepCopy()
      ..enabled = enabled
      ..policySource = EndpointSource.ENDPOINT_SOURCE_USER;
    await _persist(
      replaceEndpointRoute(endpoint, updated),
      route.routeId,
      successMessage: enabled ? 'Route enabled' : 'Route disabled',
    );
  }

  Future<void> _move(
    EndpointConfigV1 endpoint,
    EndpointRouteConfigV1 route,
    int direction,
  ) => _persist(
    moveEndpointRoute(endpoint, route.routeId, direction),
    route.routeId,
    successMessage: 'Route priority updated',
  );

  Future<void> _persist(
    EndpointConfigV1 endpoint,
    String routeId, {
    required String successMessage,
  }) async {
    if (_busyRouteId != null) return;
    setState(() {
      _busyRouteId = routeId;
      _error = null;
    });
    var saved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      await EndpointRepository(runtime).upsertEndpoint(endpoint);
      saved = true;
      ref.invalidate(endpointRegistryProvider);
      ref.invalidate(connectionPolicyProvider(widget.endpointId));
      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpointId));
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(successMessage)));
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = saved
            ? 'Route was saved, but reconnect failed. Try refreshing.'
            : _routeErrorText(error);
      });
    } finally {
      if (mounted) setState(() => _busyRouteId = null);
    }
  }

  Future<void> _showRouteActions(
    EndpointConfigV1 endpoint,
    EndpointRouteConfigV1 route, {
    required bool canRemove,
  }) async {
    final kind = endpointRouteKind(route);
    final action = await showModalBottomSheet<_RouteAction>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      useSafeArea: true,
      showDragHandle: true,
      builder: (context) => SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                title: Text(_routeTitle(route)),
                subtitle: Text('${_routeKindLabel(kind)} · ${route.routeId}'),
                trailing: IconButton(
                  tooltip: 'Close route actions',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close_rounded),
                ),
              ),
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.play_circle_outline_rounded),
                title: const Text('Test route'),
                subtitle: const Text('Open an isolated session on this route'),
                enabled: route.enabled,
                onTap: route.enabled
                    ? () => Navigator.pop(context, _RouteAction.test)
                    : null,
              ),
              if (kind == EndpointRouteKind.direct ||
                  kind == EndpointRouteKind.ssh)
                ListTile(
                  leading: const Icon(Icons.edit_outlined),
                  title: const Text('Edit route'),
                  onTap: () => Navigator.pop(context, _RouteAction.edit),
                ),
              if (kind == EndpointRouteKind.ssh)
                ListTile(
                  leading: const Icon(Icons.key_rounded),
                  title: const Text('Prepare SSH key'),
                  subtitle: const Text(
                    'Create a key in platform secure storage',
                  ),
                  onTap: () =>
                      Navigator.pop(context, _RouteAction.provisionSsh),
                ),
              if ((kind == EndpointRouteKind.direct ||
                      kind == EndpointRouteKind.ssh) &&
                  canRemove)
                ListTile(
                  leading: Icon(
                    Icons.delete_outline_rounded,
                    color: AnyttyPalette.of(context).danger,
                  ),
                  title: Text(
                    'Remove route',
                    style: TextStyle(color: AnyttyPalette.of(context).danger),
                  ),
                  onTap: () => Navigator.pop(context, _RouteAction.remove),
                ),
            ],
          ),
        ),
      ),
    );
    if (!mounted || action == null) return;
    switch (action) {
      case _RouteAction.test:
        await _testRoute(route);
      case _RouteAction.edit:
        await _openEditor(route: route);
      case _RouteAction.provisionSsh:
        await _provisionSsh(route);
      case _RouteAction.remove:
        await _removeRoute(endpoint, route);
    }
  }

  Future<void> _testRoute(EndpointRouteConfigV1 route) async {
    if (_busyRouteId != null) return;
    setState(() {
      _busyRouteId = route.routeId;
      _error = null;
    });
    EndpointSessionClient? session;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      session = await EndpointSessionClient.open(
        runtime,
        widget.endpointId,
        routeOverride: route.routeId,
      );
      session.close();
      await session.closed.timeout(const Duration(seconds: 5));
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Connection test passed')));
    } catch (error) {
      session?.close();
      if (!mounted) return;
      setState(() => _error = _routeErrorText(error));
    } finally {
      if (mounted) setState(() => _busyRouteId = null);
    }
  }

  Future<void> _provisionSsh(EndpointRouteConfigV1 route) async {
    if (_busyRouteId != null) return;
    setState(() {
      _busyRouteId = route.routeId;
      _error = null;
    });
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final result = await EndpointRepository(runtime)
          .provisionSshCredential(widget.endpointId, route.routeId);
      ref.invalidate(endpointRegistryProvider);
      if (!mounted) return;
      await _showSshKey(result);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = _routeErrorText(error));
    } finally {
      if (mounted) setState(() => _busyRouteId = null);
    }
  }

  Future<void> _showSshKey(
    SSHCredentialProvisionResult result,
  ) => showModalBottomSheet<void>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    useSafeArea: true,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) => SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 0, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                const Expanded(
                  child: Text(
                    'SSH public key',
                    style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
                  ),
                ),
                IconButton(
                  tooltip: 'Close SSH key',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close_rounded),
                ),
              ],
            ),
            Text(
              result.keyFingerprint,
              style: TextStyle(
                color: AnyttyPalette.of(context).muted,
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 11,
              ),
            ),
            const SizedBox(height: 12),
            Container(
              constraints: const BoxConstraints(maxHeight: 180),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: AnyttyPalette.of(context).surfaceRaised,
                borderRadius: BorderRadius.circular(6),
                border: Border.all(color: AnyttyPalette.of(context).border),
              ),
              child: SingleChildScrollView(
                child: SelectableText(
                  result.authorizedKey,
                  style: const TextStyle(
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 11,
                    height: 1.4,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            SizedBox(
              height: 48,
              child: FilledButton.icon(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: result.authorizedKey));
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('SSH public key copied')),
                  );
                },
                icon: const Icon(Icons.copy_rounded, size: 18),
                label: const Text('Copy public key'),
              ),
            ),
          ],
        ),
      ),
    ),
  );

  Future<void> _removeRoute(
    EndpointConfigV1 endpoint,
    EndpointRouteConfigV1 route,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Remove route?'),
        content: Text(
          '${_routeTitle(route)} will be removed. Credentials no longer referenced by any route are deleted by Go in the same registry transaction.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: AnyttyPalette.of(context).danger,
            ),
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Remove'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _persist(
      removeEndpointRoute(endpoint, route.routeId),
      route.routeId,
      successMessage: 'Route removed',
    );
  }
}
