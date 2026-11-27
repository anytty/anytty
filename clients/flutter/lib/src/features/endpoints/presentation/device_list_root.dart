part of 'device_list_screen.dart';

final class DeviceListScreen extends ConsumerStatefulWidget {
  const DeviceListScreen({super.key});

  @override
  ConsumerState<DeviceListScreen> createState() => _DeviceListScreenState();
}

final class _DeviceListScreenState extends ConsumerState<DeviceListScreen> {
  final _searchController = TextEditingController();
  final _searchFocusNode = FocusNode(debugLabel: 'device-search');
  bool _refreshing = false;
  bool _searchOpen = false;

  @override
  void initState() {
    super.initState();
    _searchController.addListener(_handleSearchChanged);
  }

  @override
  void dispose() {
    _searchController
      ..removeListener(_handleSearchChanged)
      ..dispose();
    _searchFocusNode.dispose();
    super.dispose();
  }

  void _handleSearchChanged() => setState(() {});

  void _openSearch() {
    if (_searchOpen) return;
    setState(() => _searchOpen = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _searchOpen) _searchFocusNode.requestFocus();
    });
  }

  void _closeSearch() {
    _searchFocusNode.unfocus();
    _searchController.clear();
    if (mounted) setState(() => _searchOpen = false);
  }

  @override
  Widget build(BuildContext context) {
    final registry = ref.watch(endpointRegistryProvider);
    final startupDiagnostics = ref.watch(startupDiagnosticsProvider);
    final deviceCount = registry.valueOrNull?.endpoints.length ?? 0;
    final endpoints =
        registry.valueOrNull?.endpoints ?? const <EndpointConfigV1>[];
    final visibleEndpoints = searchEndpoints(endpoints, _searchController.text);
    final onlineCount = endpoints.where((endpoint) {
      final presence = _cloudPresenceFor(ref, endpoint);
      return _deviceVisualState(endpoint, presence).online;
    }).length;
    final screen = Scaffold(
      appBar: AppBar(
        toolbarHeight: 76,
        titleSpacing: 20,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (MediaQuery.textScalerOf(context).scale(10) < 16) ...[
                  const ExcludeSemantics(child: AnyttyBrandMark(height: 18)),
                  const SizedBox(width: 6),
                ],
                Text(
                  'ANYTTY',
                  style: TextStyle(
                    color: AnyttyPalette.of(context).text,
                    fontSize: 10,
                    fontWeight: FontWeight.w800,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              anyttyText(context, en: 'Devices', zh: '设备'),
              style: const TextStyle(fontSize: 27, fontWeight: FontWeight.w700),
            ),
          ],
        ),
        actions: [
          _RaisedHeaderAction(
            child: IconButton(
              tooltip: anyttyText(context, en: 'Pair device', zh: '扫码配对'),
              onPressed: registry.hasValue
                  ? () => _showPairingSheet(context)
                  : null,
              icon: const Icon(Icons.qr_code_scanner_rounded),
            ),
          ),
          _RaisedHeaderAction(
            child: FileTransferCenterAction(
              controller: ref.read(fileTransferControllerProvider),
              showWhenEmpty: true,
              dimension: 48,
              iconSize: 22,
            ),
          ),
          _RaisedHeaderAction(
            child: _DeviceHeaderStatus(
              onlineCount: onlineCount,
              totalCount: deviceCount,
              refreshing: registry.isLoading || _refreshing,
              onRefresh: _refreshDevices,
            ),
          ),
          _RaisedHeaderAction(
            child: IconButton(
              tooltip: anyttyText(context, en: 'Settings', zh: '设置'),
              onPressed: () => context.push('/settings'),
              icon: const Icon(Icons.settings_outlined),
            ),
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: registry.when(
        loading: () => const _DeviceLoading(),
        error: (_, _) => _RegistryError(
          message: startupDiagnostics.safeFailureMessage,
          busy: _refreshing,
          onRetry: _retryStartup,
          onCopyDiagnostics: _copyStartupDiagnostics,
          onReset: _confirmLocalReset,
        ),
        data: (value) => RefreshIndicator(
          onRefresh: _refreshDevices,
          child: value.endpoints.isEmpty
              ? ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(16, 40, 16, 24),
                  children: [
                    _EmptyDevices(onPair: () => _showPairingSheet(context)),
                  ],
                )
              : ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.fromLTRB(16, 3, 16, 104),
                  children: [
                    _DeviceSectionHeading(
                      count: visibleEndpoints.length,
                      searchOpen: _searchOpen,
                      searchController: _searchController,
                      searchFocusNode: _searchFocusNode,
                      onOpenSearch: _openSearch,
                      onCloseSearch: _closeSearch,
                    ),
                    const SizedBox(height: 11),
                    if (visibleEndpoints.isEmpty)
                      const _NoMatchingDevices()
                    else
                      _DeviceListPanel(
                        endpoints: visibleEndpoints,
                        defaultEndpointId: value.defaultEndpointId,
                        searchQuery: _searchController.text,
                        onActions: (endpoint) => _showDeviceActionsSheet(
                          context,
                          endpoint: endpoint,
                          isDefault:
                              endpoint.endpointId == value.defaultEndpointId,
                        ),
                      ),
                  ],
                ),
        ),
      ),
    );
    return PopScope<Object?>(
      canPop: !_searchOpen,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && _searchOpen) _closeSearch();
      },
      child: screen,
    );
  }

  Future<void> _refreshDevices() async {
    if (_refreshing) return;
    setState(() => _refreshing = true);
    try {
      final registry = await ref.refresh(endpointRegistryProvider.future);
      await Future.wait<void>([
        for (final endpoint in registry.endpoints)
          if (_hasEnabledCloudRoute(endpoint))
            ref
                .refresh(
                  endpointCloudPresenceProvider(endpoint.endpointId).future,
                )
                .then<void>((_) {}, onError: (_, _) {}),
      ]);
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  Future<void> _retryStartup() async {
    if (_refreshing) return;
    setState(() => _refreshing = true);
    ref.invalidate(endpointRegistryProvider);
    ref.invalidate(anyttyRuntimeProvider);
    try {
      await ref.read(endpointRegistryProvider.future);
    } catch (_) {
      // The recovery panel renders the new allowlisted failure category.
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  Future<void> _copyStartupDiagnostics() async {
    final report = ref.read(startupDiagnosticsProvider).buildRedactedReport();
    await Clipboard.setData(ClipboardData(text: report));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Redacted startup diagnostics copied')),
    );
  }

  Future<void> _confirmLocalReset() async {
    if (_refreshing) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Reset saved devices?'),
        content: const Text(
          'This clears the local device registry on this phone. It does not change remote terminals.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Reset'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _refreshing = true);
    try {
      await ref.read(startupLocalResetProvider).reset();
      ref.read(startupDiagnosticsProvider).recordLocalReset();
      ref.invalidate(endpointRegistryProvider);
      ref.invalidate(anyttyRuntimeProvider);
      await ref.read(endpointRegistryProvider.future);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Local reset could not be completed')),
        );
      }
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }
}
