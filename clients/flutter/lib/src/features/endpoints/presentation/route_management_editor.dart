part of 'route_management_screen.dart';


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
