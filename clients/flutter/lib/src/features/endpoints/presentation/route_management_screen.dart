import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../data/endpoint_repository.dart';
import '../domain/route_management.dart';

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

final class RouteEditorScreen extends ConsumerWidget {
  const RouteEditorScreen({
    super.key,
    required this.endpointId,
    this.label,
    this.routeId,
    this.newKind,
  });

  final String endpointId;
  final String? label;
  final String? routeId;
  final EndpointRouteKind? newKind;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final registry = ref.watch(endpointRegistryProvider);
    return registry.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (error, _) => Scaffold(
        appBar: AppBar(),
        body: _RouteLoadError(
          message: error.toString(),
          onRetry: () => ref.invalidate(endpointRegistryProvider),
        ),
      ),
      data: (value) {
        final endpoint = _findEndpoint(value, endpointId);
        if (endpoint == null) {
          return Scaffold(
            appBar: AppBar(),
            body: const Center(child: Text('Endpoint is not configured')),
          );
        }
        final isNew = routeId == null;
        EndpointRouteConfigV1? route;
        if (routeId != null) {
          for (final candidate in endpoint.routes) {
            if (candidate.routeId == routeId) {
              route = candidate.deepCopy();
              break;
            }
          }
        } else if (newKind != null) {
          route = newEndpointRoute(endpoint, newKind!);
        }
        if (route == null) {
          return Scaffold(
            appBar: AppBar(),
            body: const Center(child: Text('Route is not configured')),
          );
        }
        return _RouteEditorForm(
          key: ValueKey('${endpoint.endpointId}:${route.routeId}:$isNew'),
          endpoint: endpoint,
          route: route,
          isNew: isNew,
          label: label,
        );
      },
    );
  }
}

final class _RouteEditorForm extends ConsumerStatefulWidget {
  const _RouteEditorForm({
    super.key,
    required this.endpoint,
    required this.route,
    required this.isNew,
    this.label,
  });

  final EndpointConfigV1 endpoint;
  final EndpointRouteConfigV1 route;
  final bool isNew;
  final String? label;

  @override
  ConsumerState<_RouteEditorForm> createState() => _RouteEditorFormState();
}

final class _RouteEditorFormState extends ConsumerState<_RouteEditorForm> {
  late final TextEditingController _routeId;
  late final TextEditingController _displayName;
  late final TextEditingController _directAddresses;
  late final TextEditingController _setupAddresses;
  late final TextEditingController _iceAddresses;
  late final TextEditingController _sshHost;
  late final TextEditingController _sshPort;
  late final TextEditingController _sshUser;
  late final TextEditingController _hostKeys;
  late final TextEditingController _proxyJump;
  late final TextEditingController _remoteSignaling;
  late final TextEditingController _remoteIce;
  late bool _advancedDirect;
  bool _saving = false;
  String? _error;

  EndpointRouteKind get _kind => endpointRouteKind(widget.route);

  @override
  void initState() {
    super.initState();
    final route = widget.route;
    final direct = route.hasDirectWebrtcTcp()
        ? route.directWebrtcTcp
        : DirectWebRTCTCPRouteConfig();
    final ssh = route.hasSshWebrtcTcp()
        ? route.sshWebrtcTcp
        : SSHWebRTCTCPRouteConfig(port: 22);
    _routeId = TextEditingController(text: route.routeId);
    _displayName = TextEditingController(text: route.displayName);
    _advancedDirect = !sameRouteValues(
      direct.signalingAddresses,
      direct.iceTcpAddresses,
    );
    _directAddresses = TextEditingController(
      text: direct.signalingAddresses.join('\n'),
    );
    _setupAddresses = TextEditingController(
      text: direct.signalingAddresses.join('\n'),
    );
    _iceAddresses = TextEditingController(
      text: direct.iceTcpAddresses.join('\n'),
    );
    _sshHost = TextEditingController(text: ssh.host);
    _sshPort = TextEditingController(text: '${ssh.port == 0 ? 22 : ssh.port}');
    _sshUser = TextEditingController(text: ssh.user);
    _hostKeys = TextEditingController(text: ssh.hostKeyFingerprints.join('\n'));
    _proxyJump = TextEditingController(text: ssh.proxyJump);
    _remoteSignaling = TextEditingController(text: ssh.remoteSignalingAddress);
    _remoteIce = TextEditingController(text: ssh.remoteIceTcpAddress);
  }

  @override
  void dispose() {
    for (final controller in [
      _routeId,
      _displayName,
      _directAddresses,
      _setupAddresses,
      _iceAddresses,
      _sshHost,
      _sshPort,
      _sshUser,
      _hostKeys,
      _proxyJump,
      _remoteSignaling,
      _remoteIce,
    ]) {
      controller.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back to routes',
          onPressed: _saving ? null : () => Navigator.maybePop(context),
          icon: const Icon(Icons.arrow_back_rounded),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.isNew
                  ? 'Add ${_routeKindLabel(_kind)} route'
                  : 'Edit route',
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            Text(
              widget.label?.trim().isNotEmpty == true
                  ? widget.label!.trim()
                  : widget.endpoint.endpointId,
              style: TextStyle(color: palette.muted, fontSize: 12),
            ),
          ],
        ),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 28),
        children: [
          _RouteSectionLabel(label: 'IDENTITY'),
          const SizedBox(height: 8),
          _RouteEditorPanel(
            children: [
              TextField(
                controller: _routeId,
                enabled: widget.isNew && !_saving,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(
                  labelText: 'Route ID',
                  helperText: 'Stable identifier used by Go and diagnostics',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _displayName,
                enabled: !_saving,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(labelText: 'Display name'),
              ),
            ],
          ),
          const SizedBox(height: 24),
          if (_kind == EndpointRouteKind.direct) ...[
            _RouteSectionLabel(label: 'DIRECT ENDPOINTS'),
            const SizedBox(height: 8),
            _RouteEditorPanel(
              children: [
                if (!_advancedDirect)
                  TextField(
                    controller: _directAddresses,
                    enabled: !_saving,
                    minLines: 3,
                    maxLines: 6,
                    decoration: const InputDecoration(
                      labelText: 'Direct addresses',
                      hintText: 'host:port, one per line',
                      helperText: 'Used for both setup and ICE TCP',
                    ),
                  )
                else ...[
                  TextField(
                    controller: _setupAddresses,
                    enabled: !_saving,
                    minLines: 3,
                    maxLines: 6,
                    decoration: const InputDecoration(
                      labelText: 'Setup addresses',
                      hintText: 'host:port, one per line',
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _iceAddresses,
                    enabled: !_saving,
                    minLines: 3,
                    maxLines: 6,
                    decoration: const InputDecoration(
                      labelText: 'ICE TCP addresses',
                      hintText: 'host:port, one per line',
                    ),
                  ),
                ],
                const SizedBox(height: 8),
                Material(
                  color: Colors.transparent,
                  child: SwitchListTile.adaptive(
                    contentPadding: EdgeInsets.zero,
                    title: const Text('Separate setup and ICE ports'),
                    subtitle: const Text(
                      'Enable only for legacy or explicitly split deployments',
                    ),
                    value: _advancedDirect,
                    onChanged: _saving
                        ? null
                        : (value) => setState(() {
                            if (!value && _advancedDirect) {
                              final fallback =
                                  _setupAddresses.text.trim().isEmpty
                                  ? _iceAddresses.text
                                  : _setupAddresses.text;
                              _directAddresses.text = fallback;
                            } else if (value && !_advancedDirect) {
                              _setupAddresses.text = _directAddresses.text;
                              _iceAddresses.text = _directAddresses.text;
                            }
                            _advancedDirect = value;
                          }),
                  ),
                ),
              ],
            ),
          ],
          if (_kind == EndpointRouteKind.ssh) ...[
            _RouteSectionLabel(label: 'SSH CONNECTION'),
            const SizedBox(height: 8),
            _RouteEditorPanel(
              children: [
                TextField(
                  controller: _sshHost,
                  enabled: !_saving,
                  textInputAction: TextInputAction.next,
                  decoration: const InputDecoration(labelText: 'SSH host'),
                ),
                const SizedBox(height: 12),
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      flex: 2,
                      child: TextField(
                        controller: _sshUser,
                        enabled: !_saving,
                        textInputAction: TextInputAction.next,
                        decoration: const InputDecoration(labelText: 'User'),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: TextField(
                        controller: _sshPort,
                        enabled: !_saving,
                        keyboardType: TextInputType.number,
                        textInputAction: TextInputAction.next,
                        inputFormatters: [
                          FilteringTextInputFormatter.digitsOnly,
                        ],
                        decoration: const InputDecoration(labelText: 'Port'),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _hostKeys,
                  enabled: !_saving,
                  minLines: 3,
                  maxLines: 6,
                  decoration: const InputDecoration(
                    labelText: 'Host key fingerprints',
                    hintText: 'SHA256:..., one per line',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _proxyJump,
                  enabled: !_saving,
                  textInputAction: TextInputAction.next,
                  decoration: const InputDecoration(
                    labelText: 'ProxyJump',
                    hintText: 'Optional',
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            _RouteSectionLabel(label: 'REMOTE ANYTTY'),
            const SizedBox(height: 8),
            _RouteEditorPanel(
              children: [
                TextField(
                  controller: _remoteSignaling,
                  enabled: !_saving,
                  textInputAction: TextInputAction.next,
                  decoration: const InputDecoration(
                    labelText: 'Remote setup address',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _remoteIce,
                  enabled: !_saving,
                  decoration: const InputDecoration(
                    labelText: 'Remote ICE TCP address',
                  ),
                ),
              ],
            ),
          ],
          if (_error != null) ...[
            const SizedBox(height: 16),
            _RouteInlineError(
              message: _error!,
              onClose: () => setState(() => _error = null),
            ),
          ],
          const SizedBox(height: 24),
          SizedBox(
            height: 48,
            child: FilledButton.icon(
              onPressed: _saving ? null : _save,
              icon: _saving
                  ? const SizedBox.square(
                      dimension: 17,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.save_outlined, size: 18),
              label: Text(_saving ? 'Saving...' : 'Save route'),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _save() async {
    if (_saving) return;
    final route = widget.route.deepCopy()
      ..routeId = _routeId.text.trim()
      ..displayName = _displayName.text.trim()
      ..source = EndpointSource.ENDPOINT_SOURCE_USER
      ..policySource = EndpointSource.ENDPOINT_SOURCE_USER;
    if (_kind == EndpointRouteKind.direct) {
      final direct = route.directWebrtcTcp.deepCopy();
      final setup = splitRouteValues(
        _advancedDirect ? _setupAddresses.text : _directAddresses.text,
      );
      final ice = splitRouteValues(
        _advancedDirect ? _iceAddresses.text : _directAddresses.text,
      );
      direct
        ..signalingAddresses.clear()
        ..signalingAddresses.addAll(setup)
        ..iceTcpAddresses.clear()
        ..iceTcpAddresses.addAll(ice);
      route.directWebrtcTcp = direct;
    } else if (_kind == EndpointRouteKind.ssh) {
      final ssh = route.sshWebrtcTcp.deepCopy()
        ..host = _sshHost.text.trim()
        ..port = int.tryParse(_sshPort.text) ?? 0
        ..user = _sshUser.text.trim()
        ..proxyJump = _proxyJump.text.trim()
        ..remoteSignalingAddress = _remoteSignaling.text.trim()
        ..remoteIceTcpAddress = _remoteIce.text.trim();
      ssh.hostKeyFingerprints
        ..clear()
        ..addAll(splitRouteValues(_hostKeys.text));
      route.sshWebrtcTcp = ssh;
    }
    final error = validateEndpointRoute(
      widget.endpoint,
      route,
      isNew: widget.isNew,
    );
    if (error != null) {
      setState(() => _error = error);
      return;
    }
    setState(() {
      _saving = true;
      _error = null;
    });
    var saved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final endpoint = replaceEndpointRoute(widget.endpoint, route);
      await EndpointRepository(runtime).upsertEndpoint(endpoint);
      saved = true;
      ref.invalidate(endpointRegistryProvider);
      ref.invalidate(connectionPolicyProvider(widget.endpoint.endpointId));
      await EndpointRepository(runtime)
          .disconnectEndpoint(widget.endpoint.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpoint.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpoint.endpointId));
      if (!mounted) return;
      Navigator.pop(context, true);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _saving = false;
        _error = saved
            ? 'Route was saved, but reconnect failed. Return to routes and refresh.'
            : _routeErrorText(error);
      });
    }
  }
}

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
