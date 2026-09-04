import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../../shared/presentation/fuzzy_highlight_text.dart';
import '../../browser/data/browser_device_data.dart';
import '../../files/presentation/file_transfer_sheet.dart';
import '../data/endpoint_repository.dart';
import '../domain/device_search.dart';

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
            Text(
              'ANYTTY',
              style: TextStyle(
                color: AnyttyPalette.of(context).accent,
                fontSize: 10,
                fontWeight: FontWeight.w800,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              anyttyText(context, en: 'Devices', zh: '设备'),
              style: const TextStyle(fontSize: 27, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 3),
            Text(
              anyttyText(
                context,
                en: 'Return to your workspace anytime',
                zh: '随时回到你的工作现场',
              ),
              style: TextStyle(
                color: AnyttyPalette.of(context).muted,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
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

final class _RaisedHeaderAction extends StatelessWidget {
  const _RaisedHeaderAction({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) =>
      Transform.translate(offset: const Offset(0, -18), child: child);
}

AsyncValue<EndpointCloudPresenceGetResult>? _cloudPresenceFor(
  WidgetRef ref,
  EndpointConfigV1 endpoint,
) {
  if (!endpoint.enabled ||
      _authorizationRequired(endpoint) ||
      !_hasEnabledCloudRoute(endpoint)) {
    return null;
  }
  return ref.watch(endpointCloudPresenceProvider(endpoint.endpointId));
}

final class _DeviceHeaderStatus extends StatelessWidget {
  const _DeviceHeaderStatus({
    required this.onlineCount,
    required this.totalCount,
    required this.refreshing,
    required this.onRefresh,
  });

  final int onlineCount;
  final int totalCount;
  final bool refreshing;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = anyttyText(
      context,
      en: '$onlineCount of $totalCount devices online. Refresh',
      zh: '$onlineCount/$totalCount 台在线，刷新',
    );
    return Tooltip(
      message: status,
      child: Semantics(
        button: true,
        liveRegion: true,
        label: status,
        child: InkResponse(
          key: const ValueKey('device-header-status'),
          onTap: refreshing ? null : onRefresh,
          radius: 24,
          child: SizedBox.square(
            dimension: 48,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                SizedBox.square(
                  dimension: 19,
                  child: refreshing
                      ? CircularProgressIndicator(
                          color: palette.accent,
                          strokeWidth: 2,
                        )
                      : Stack(
                          alignment: Alignment.center,
                          clipBehavior: Clip.none,
                          children: [
                            Icon(
                              Icons.refresh_rounded,
                              color: palette.muted,
                              size: 19,
                            ),
                            Positioned(
                              right: -1,
                              top: -1,
                              child: Container(
                                width: 7,
                                height: 7,
                                decoration: BoxDecoration(
                                  color: onlineCount > 0
                                      ? palette.success
                                      : palette.faint,
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: palette.background,
                                    width: 1.5,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                ),
                const SizedBox(height: 1),
                Text(
                  '$onlineCount/$totalCount',
                  textScaler: TextScaler.noScaling,
                  style: TextStyle(
                    color: palette.muted,
                    fontSize: 9,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _DeviceSearchField extends StatelessWidget {
  const _DeviceSearchField({
    required this.controller,
    required this.focusNode,
    required this.onClose,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 44,
      child: Focus(
        onKeyEvent: (node, event) {
          if (event is KeyDownEvent &&
              event.logicalKey == LogicalKeyboardKey.escape) {
            onClose();
            return KeyEventResult.handled;
          }
          return KeyEventResult.ignored;
        },
        child: TextField(
          key: const ValueKey('device-search-field'),
          controller: controller,
          focusNode: focusNode,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: anyttyText(context, en: 'Search devices', zh: '搜索设备'),
            prefixIcon: Icon(
              Icons.search_rounded,
              color: palette.muted,
              size: 19,
            ),
            suffixIcon: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (controller.text.isNotEmpty)
                  IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Clear search',
                      zh: '清除搜索',
                    ),
                    onPressed: controller.clear,
                    icon: const Icon(Icons.backspace_outlined, size: 17),
                  ),
                IconButton(
                  tooltip: anyttyText(context, en: 'Close search', zh: '关闭搜索'),
                  onPressed: onClose,
                  icon: const Icon(Icons.close_rounded, size: 18),
                ),
              ],
            ),
            filled: true,
            fillColor: palette.surface,
            contentPadding: const EdgeInsets.symmetric(vertical: 10),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(14),
              borderSide: BorderSide(color: palette.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(14),
              borderSide: BorderSide(color: palette.accent, width: 1.5),
            ),
          ),
        ),
      ),
    );
  }
}

final class _NoMatchingDevices extends StatelessWidget {
  const _NoMatchingDevices();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 36),
      child: Column(
        children: [
          Icon(Icons.search_off_rounded, color: palette.muted, size: 30),
          const SizedBox(height: 10),
          Text(
            anyttyText(context, en: 'No matching devices', zh: '没有匹配的设备'),
            style: TextStyle(color: palette.muted, fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

final class _DeviceSectionHeading extends StatelessWidget {
  const _DeviceSectionHeading({
    required this.count,
    required this.searchOpen,
    required this.searchController,
    required this.searchFocusNode,
    required this.onOpenSearch,
    required this.onCloseSearch,
  });

  final int count;
  final bool searchOpen;
  final TextEditingController searchController;
  final FocusNode searchFocusNode;
  final VoidCallback onOpenSearch;
  final VoidCallback onCloseSearch;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: searchOpen
          ? _DeviceSearchField(
              controller: searchController,
              focusNode: searchFocusNode,
              onClose: onCloseSearch,
          )
          : Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                Text(
                  anyttyText(context, en: 'My devices', zh: '我的设备'),
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  '$count',
                  style: TextStyle(
                    color: palette.faint,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                SizedBox.square(
                  dimension: 40,
                  child: IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Search devices',
                      zh: '搜索设备',
                    ),
                    onPressed: onOpenSearch,
                    padding: EdgeInsets.zero,
                    icon: const Icon(Icons.search_rounded, size: 20),
                  ),
                ),
              ],
            ),
    );
  }
}

final class _DeviceListPanel extends StatelessWidget {
  const _DeviceListPanel({
    required this.endpoints,
    required this.defaultEndpointId,
    required this.searchQuery,
    required this.onActions,
  });

  final List<EndpointConfigV1> endpoints;
  final String defaultEndpointId;
  final String searchQuery;
  final ValueChanged<EndpointConfigV1> onActions;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (var index = 0; index < endpoints.length; index++) ...[
          _DeviceRow(
            endpoint: endpoints[index],
            isDefault: endpoints[index].endpointId == defaultEndpointId,
            searchQuery: searchQuery,
            onActions: () => onActions(endpoints[index]),
          ),
          if (index != endpoints.length - 1) const SizedBox(height: 10),
        ],
      ],
    );
  }
}

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
              context.push(
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

final class _DeviceActionsSheet extends ConsumerStatefulWidget {
  const _DeviceActionsSheet({required this.endpoint, required this.isDefault});

  final EndpointConfigV1 endpoint;
  final bool isDefault;

  @override
  ConsumerState<_DeviceActionsSheet> createState() =>
      _DeviceActionsSheetState();
}

final class _DeviceActionsSheetState
    extends ConsumerState<_DeviceActionsSheet> {
  late final TextEditingController _labelController;
  bool _busy = false;
  String? _error;

  String get _label => widget.endpoint.label.trim().isEmpty
      ? widget.endpoint.endpointId
      : widget.endpoint.label.trim();

  @override
  void initState() {
    super.initState();
    _labelController = TextEditingController(text: _label);
  }

  @override
  void dispose() {
    _labelController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: EdgeInsets.only(
        bottom:
            MediaQuery.viewInsetsOf(context).bottom +
            MediaQuery.viewPaddingOf(context).bottom,
      ),
      child: SingleChildScrollView(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 576),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Container(
                        width: 40,
                        height: 40,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: palette.surfaceRaised,
                          border: Border.all(color: palette.border),
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: Icon(
                          _platformIcon(widget.endpoint.platform),
                          size: 22,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              _label,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                fontSize: 17,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            Text(
                              anyttyText(
                                context,
                                en: 'Device actions',
                                zh: '设备操作',
                              ),
                              style: TextStyle(
                                color: palette.muted,
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),
                      ),
                      IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Close device actions',
                          zh: '关闭设备操作',
                        ),
                        onPressed: _busy ? null : () => Navigator.pop(context),
                        icon: const Icon(Icons.close_rounded),
                      ),
                    ],
                  ),
                  const SizedBox(height: 20),
                  TextField(
                    controller: _labelController,
                    enabled: !_busy,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _saveLabel(),
                    decoration: InputDecoration(
                      labelText: anyttyText(
                        context,
                        en: 'Display name',
                        zh: '显示名称',
                      ),
                      suffixIcon: IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Save display name',
                          zh: '保存显示名称',
                        ),
                        onPressed: _busy ? null : _saveLabel,
                        icon: const Icon(Icons.check_rounded),
                      ),
                    ),
                  ),
                  if (_busy) ...[
                    const SizedBox(height: 12),
                    const LinearProgressIndicator(minHeight: 2),
                  ],
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Semantics(
                      liveRegion: true,
                      child: Text(
                        _error!,
                        style: TextStyle(color: palette.danger, fontSize: 12),
                      ),
                    ),
                  ],
                  const SizedBox(height: 12),
                  Divider(height: 1, color: palette.border),
                  if (_authorizationRequired(widget.endpoint)) ...[
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      enabled: !_busy,
                      leading: Icon(
                        Icons.shield_outlined,
                        color: palette.warning,
                      ),
                      title: Text(
                        anyttyText(context, en: 'Authorize device', zh: '授权设备'),
                      ),
                      subtitle: Text(
                        anyttyText(
                          context,
                          en: 'Fresh pairing required',
                          zh: '需要重新配对',
                        ),
                      ),
                      trailing: Icon(
                        Icons.chevron_right_rounded,
                        color: palette.faint,
                      ),
                      onTap: _authorize,
                    ),
                    Divider(height: 1, color: palette.border),
                  ],
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy && !widget.isDefault,
                    leading: const Icon(Icons.star_outline_rounded),
                    title: Text(
                      widget.isDefault
                          ? anyttyText(
                              context,
                              en: 'Default device',
                              zh: '默认设备',
                            )
                          : anyttyText(
                              context,
                              en: 'Set as default',
                              zh: '设为默认设备',
                            ),
                    ),
                    trailing: widget.isDefault
                        ? Icon(Icons.check_rounded, color: palette.success)
                        : null,
                    onTap: widget.isDefault ? null : _makeDefault,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: const Icon(Icons.route_outlined),
                    title: Text(
                      anyttyText(context, en: 'Connection', zh: '网络连接'),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Route, latency, traffic, and policy',
                        zh: '线路、延迟与连接策略',
                      ),
                    ),
                    trailing: Icon(
                      Icons.chevron_right_rounded,
                      color: palette.faint,
                    ),
                    onTap: _openConnection,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: const Icon(Icons.link_off_rounded),
                    title: Text(
                      anyttyText(context, en: 'Disconnect', zh: '断开连接'),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Close the current pooled connection',
                        zh: '关闭当前复用连接',
                      ),
                    ),
                    onTap: _disconnect,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: Icon(
                      Icons.delete_outline_rounded,
                      color: palette.danger,
                    ),
                    title: Text(
                      anyttyText(context, en: 'Remove device', zh: '移除设备'),
                      style: TextStyle(color: palette.danger),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Remove saved routes and authorization',
                        zh: '删除已保存的线路与授权',
                      ),
                    ),
                    onTap: _remove,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<EndpointRepository> _repository() async {
    final runtime = await ref.read(anyttyRuntimeProvider.future);
    return EndpointRepository(runtime);
  }

  Future<void> _saveLabel() async {
    final label = _labelController.text.trim();
    if (label.isEmpty) {
      setState(
        () => _error = anyttyText(
          context,
          en: 'Display name is required',
          zh: '请输入显示名称',
        ),
      );
      return;
    }
    if (label == widget.endpoint.label.trim()) return;
    final endpoint = widget.endpoint.deepCopy()
      ..label = label
      ..labelSource = EndpointSource.ENDPOINT_SOURCE_USER;
    await _perform(
      successMessage: anyttyText(
        context,
        en: 'Device name updated',
        zh: '设备名称已更新',
      ),
      action: () async {
        await (await _repository()).upsertEndpoint(
          endpoint,
          makeDefault: widget.isDefault,
        );
        ref.invalidate(endpointRegistryProvider);
      },
    );
  }

  Future<void> _makeDefault() => _perform(
    successMessage: anyttyText(
      context,
      en: 'Default device updated',
      zh: '默认设备已更新',
    ),
    action: () async {
      await (await _repository()).upsertEndpoint(
        widget.endpoint,
        makeDefault: true,
      );
      ref.invalidate(endpointRegistryProvider);
    },
  );

  void _openConnection() {
    final router = GoRouter.of(context);
    final location =
        '/connection/${Uri.encodeComponent(widget.endpoint.endpointId)}'
        '?label=${Uri.encodeQueryComponent(_label)}';
    Navigator.pop(context);
    unawaited(router.push(location));
  }

  void _authorize() {
    final navigator = Navigator.of(context, rootNavigator: true);
    final route = MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (_) => _PairingSheet(reauthorizeEndpoint: widget.endpoint),
    );
    Navigator.pop(context);
    unawaited(navigator.push(route));
  }

  Future<void> _disconnect() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          anyttyText(context, en: 'Disconnect device?', zh: '断开设备连接？'),
        ),
        content: Text(
          anyttyText(
            context,
            en: 'The current connection to $_label will be closed.',
            zh: '将关闭与 $_label 的当前连接。',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(anyttyText(context, en: 'Disconnect', zh: '断开')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _perform(
      successMessage: anyttyText(
        context,
        en: 'Device disconnected',
        zh: '设备连接已断开',
      ),
      action: () async {
        await (await _repository()).disconnectEndpoint(
          widget.endpoint.endpointId,
        );
      },
    );
  }

  Future<void> _remove() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(anyttyText(context, en: 'Remove device?', zh: '移除设备？')),
        content: Text(
          anyttyText(
            context,
            en: '$_label and its saved routes and authorization will be removed from this app.',
            zh: '将从本应用移除 $_label，以及已保存的线路和授权。',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red.shade700),
            onPressed: () => Navigator.pop(context, true),
            child: Text(anyttyText(context, en: 'Remove', zh: '移除')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _perform(
      successMessage: anyttyText(context, en: 'Device removed', zh: '设备已移除'),
      action: () async {
        final endpointId = widget.endpoint.endpointId;
        await (await _repository()).deleteEndpoint(endpointId);
        await clearBrowserDeviceData(endpointId);
        ref.invalidate(endpointRegistryProvider);
      },
    );
  }

  Future<void> _perform({
    required String successMessage,
    required Future<void> Function() action,
  }) async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await action();
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      Navigator.pop(context);
      messenger.showSnackBar(SnackBar(content: Text(successMessage)));
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = error.toString();
      });
    }
  }
}

final class _EmptyDevices extends StatelessWidget {
  const _EmptyDevices({required this.onPair});

  final VoidCallback onPair;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(Icons.dns_outlined, size: 24, color: palette.text),
          ),
          const SizedBox(height: 20),
          Text(
            anyttyText(context, en: 'No paired devices', zh: '暂无已配对设备'),
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            anyttyText(
              context,
              en: 'Pair this app with an AnyTTY endpoint to begin.',
              zh: '与 AnyTTY 端点配对后即可开始使用。',
            ),
            style: TextStyle(color: palette.muted, height: 1.5),
          ),
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton.icon(
              onPressed: onPair,
              icon: const Icon(Icons.qr_code_scanner_rounded, size: 18),
              label: Text(anyttyText(context, en: 'Scan service', zh: '扫描服务')),
            ),
          ),
        ],
      ),
    );
  }
}

final class _RegistryError extends StatelessWidget {
  const _RegistryError({
    required this.message,
    required this.busy,
    required this.onRetry,
    required this.onCopyDiagnostics,
    required this.onReset,
  });

  final String message;
  final bool busy;
  final VoidCallback onRetry;
  final VoidCallback onCopyDiagnostics;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off_outlined, color: palette.danger, size: 36),
            const SizedBox(height: 12),
            const Text(
              'Could not load devices',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 6),
            Text(
              message,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
              style: TextStyle(color: palette.muted, fontSize: 12),
            ),
            const SizedBox(height: 18),
            ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 48),
              child: FilledButton.icon(
                onPressed: busy ? null : onRetry,
                icon: busy
                    ? const SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.refresh_rounded),
                label: const Text('Retry startup'),
              ),
            ),
            const SizedBox(height: 8),
            ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 48),
              child: OutlinedButton.icon(
                onPressed: busy ? null : onCopyDiagnostics,
                icon: const Icon(Icons.copy_all_rounded),
                label: const Text('Copy redacted diagnostics'),
              ),
            ),
            const SizedBox(height: 4),
            TextButton(
              onPressed: busy ? null : onReset,
              style: TextButton.styleFrom(
                minimumSize: const Size(48, 48),
                foregroundColor: palette.danger,
              ),
              child: const Text('Reset saved devices'),
            ),
          ],
        ),
      ),
    );
  }
}

final class _DeviceLoading extends StatelessWidget {
  const _DeviceLoading();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Semantics(
        liveRegion: true,
        label: 'Loading devices',
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox.square(
              dimension: 18,
              child: CircularProgressIndicator(
                color: palette.text,
                strokeWidth: 2,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'Loading devices',
              style: TextStyle(color: palette.muted, fontSize: 13),
            ),
          ],
        ),
      ),
    );
  }
}

IconData _platformIcon(String value) {
  return switch (value.toLowerCase()) {
    'darwin' || 'macos' => Icons.laptop_mac_rounded,
    'windows' => Icons.desktop_windows_rounded,
    'linux' => Icons.terminal_rounded,
    _ => Icons.computer_rounded,
  };
}

Future<void> _showPairingSheet(
  BuildContext context, {
  EndpointConfigV1? reauthorizeEndpoint,
}) async {
  await Navigator.of(context).push<void>(
    MaterialPageRoute(
      fullscreenDialog: true,
      builder: (_) => _PairingSheet(reauthorizeEndpoint: reauthorizeEndpoint),
    ),
  );
}

final class _PairingSheet extends ConsumerStatefulWidget {
  const _PairingSheet({this.reauthorizeEndpoint});

  final EndpointConfigV1? reauthorizeEndpoint;

  @override
  ConsumerState<_PairingSheet> createState() => _PairingSheetState();
}

final class _PairingSheetState extends ConsumerState<_PairingSheet> {
  final _controller = TextEditingController();
  final _scanner = MobileScannerController();
  bool _submitting = false;
  bool _scanAccepted = false;
  bool _pasteOpen = false;
  EndpointSharePreview? _sharePreview;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    _scanner.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final sharePreview = _sharePreview;
    final reauthorizeEndpoint = widget.reauthorizeEndpoint;
    final reauthorizeLabel = reauthorizeEndpoint == null
        ? null
        : reauthorizeEndpoint.label.trim().isEmpty
        ? reauthorizeEndpoint.endpointId
        : reauthorizeEndpoint.label.trim();
    return Scaffold(
      appBar: AppBar(
        leading: Padding(
          padding: EdgeInsets.only(left: 12),
          child: Icon(
            sharePreview == null
                ? reauthorizeEndpoint == null
                      ? Icons.qr_code_scanner_rounded
                      : Icons.shield_outlined
                : Icons.move_to_inbox_outlined,
            size: 20,
          ),
        ),
        leadingWidth: 44,
        titleSpacing: 0,
        title: Text(
          sharePreview != null
              ? 'Import configuration'
              : reauthorizeEndpoint == null
              ? 'Pair device'
              : 'Authorize device',
          style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
        actions: [
          IconButton(
            tooltip: 'Close',
            onPressed: _submitting ? null : () => Navigator.pop(context),
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
      body: sharePreview == null
          ? SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(
                16,
                20,
                16,
                24 + MediaQuery.viewInsetsOf(context).bottom,
              ),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 448),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (reauthorizeLabel != null) ...[
                        _ReauthorizationTarget(
                          endpoint: reauthorizeEndpoint!,
                          label: reauthorizeLabel,
                        ),
                        const SizedBox(height: 12),
                      ],
                      Container(
                        height: 336,
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: const Color(0xff09090b),
                          border: Border.all(color: const Color(0xff27272a)),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(8),
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              MobileScanner(
                                controller: _scanner,
                                onDetect: _handleBarcode,
                              ),
                              Center(
                                child: IgnorePointer(
                                  child: Container(
                                    width: 216,
                                    height: 216,
                                    decoration: BoxDecoration(
                                      border: Border.all(
                                        color: Colors.white.withValues(
                                          alpha: 0.72,
                                        ),
                                        width: 2,
                                      ),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                  ),
                                ),
                              ),
                              Positioned(
                                top: 4,
                                right: 4,
                                child: IconButton(
                                  tooltip: 'Toggle flashlight',
                                  color: Colors.white,
                                  style: IconButton.styleFrom(
                                    backgroundColor: Colors.black54,
                                  ),
                                  onPressed: _submitting
                                      ? null
                                      : _scanner.toggleTorch,
                                  icon: const Icon(
                                    Icons.flashlight_on_outlined,
                                    size: 20,
                                  ),
                                ),
                              ),
                              if (_submitting)
                                const ColoredBox(
                                  color: Color(0x9909090b),
                                  child: Center(
                                    child: CircularProgressIndicator(
                                      color: Colors.white,
                                      strokeWidth: 2,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      OutlinedButton(
                        onPressed: _submitting
                            ? null
                            : () => setState(() => _pasteOpen = !_pasteOpen),
                        style: OutlinedButton.styleFrom(
                          minimumSize: const Size.fromHeight(48),
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                        ),
                        child: Row(
                          children: [
                            const Icon(Icons.content_paste_rounded, size: 17),
                            const SizedBox(width: 8),
                            const Expanded(
                              child: Text(
                                'Paste pairing payload',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            AnimatedRotation(
                              turns: _pasteOpen ? 0.5 : 0,
                              duration: const Duration(milliseconds: 200),
                              child: const Icon(
                                Icons.keyboard_arrow_down_rounded,
                                size: 18,
                              ),
                            ),
                          ],
                        ),
                      ),
                      if (_pasteOpen) ...[
                        const SizedBox(height: 16),
                        const Text(
                          'Pairing payload',
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _controller,
                          minLines: 4,
                          maxLines: 7,
                          enabled: !_submitting,
                          autocorrect: false,
                          decoration: InputDecoration(
                            hintText: 'Paste the pairing payload',
                            alignLabelWithHint: true,
                            errorText: _error,
                            errorMaxLines: 4,
                          ),
                          onChanged: (_) => setState(() => _error = null),
                        ),
                        const SizedBox(height: 12),
                        SizedBox(
                          height: 48,
                          child: FilledButton(
                            onPressed:
                                _submitting || _controller.text.trim().isEmpty
                                ? null
                                : _submit,
                            child: _submitting
                                ? const SizedBox.square(
                                    dimension: 18,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                    ),
                                  )
                                : const Text('Continue'),
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            )
          : _EndpointSharePreviewView(
              preview: sharePreview,
              submitting: _submitting,
              error: _error,
              onCancel: _cancelSharePreview,
              onCommit: _commitShare,
            ),
    );
  }

  void _handleBarcode(BarcodeCapture capture) {
    if (_scanAccepted || _submitting) return;
    for (final barcode in capture.barcodes) {
      final value = barcode.rawValue?.trim();
      if (value == null || value.isEmpty) continue;
      _scanAccepted = true;
      _controller.text = value;
      setState(() => _error = null);
      _submit();
      return;
    }
  }

  Future<void> _submit() async {
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final repository = EndpointRepository(runtime);
      final input = _controller.text.trim();
      if (input.startsWith('anytty://share?payload=')) {
        if (widget.reauthorizeEndpoint != null) {
          throw const AnyttyOperationException(
            'A fresh pairing code is required to authorize this device',
          );
        }
        final preview = await repository.receiveEndpointShare(input);
        await _scanner.stop();
        if (!mounted) return;
        setState(() {
          _sharePreview = preview;
          _submitting = false;
          _pasteOpen = false;
        });
        return;
      }
      final result = await repository.importPairing(
        input,
        expectedEndpointId: widget.reauthorizeEndpoint?.endpointId,
      );
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      final label = result.endpoint.label.trim().isEmpty
          ? result.endpoint.endpointId
          : result.endpoint.label.trim();
      ref.invalidate(endpointRegistryProvider);
      Navigator.pop(context);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            result.authorizationRequired
                ? '$label still requires authorization'
                : widget.reauthorizeEndpoint == null
                ? '$label paired'
                : '$label authorized',
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _scanAccepted = false;
        _error = error.toString();
        _pasteOpen = true;
      });
    }
  }

  void _cancelSharePreview() {
    if (_submitting) return;
    setState(() {
      _sharePreview = null;
      _scanAccepted = false;
      _error = null;
    });
  }

  Future<void> _commitShare() async {
    final preview = _sharePreview;
    if (_submitting || preview == null) return;
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final result = await EndpointRepository(runtime)
          .commitEndpointShare(preview.importToken);
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      final label = result.endpoint.label.trim().isEmpty
          ? result.endpoint.endpointId
          : result.endpoint.label.trim();
      ref.invalidate(endpointRegistryProvider);
      Navigator.pop(context);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            result.authorizationRequired
                ? '$label imported. Authorization is still required.'
                : '$label configuration imported',
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _error = error.toString();
      });
    }
  }
}

final class _ReauthorizationTarget extends StatelessWidget {
  const _ReauthorizationTarget({required this.endpoint, required this.label});

  final EndpointConfigV1 endpoint;
  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      label: '$label, authorization required',
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: palette.surfaceRaised,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              _platformIcon(endpoint.platform),
              color: palette.text,
              size: 22,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  Text(
                    endpoint.endpointId,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: palette.muted, fontSize: 11),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Text(
              'FRESH PAIRING',
              style: TextStyle(
                color: palette.warning,
                fontSize: 10,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _EndpointSharePreviewView extends StatelessWidget {
  const _EndpointSharePreviewView({
    required this.preview,
    required this.submitting,
    required this.error,
    required this.onCancel,
    required this.onCommit,
  });

  final EndpointSharePreview preview;
  final bool submitting;
  final String? error;
  final VoidCallback onCancel;
  final VoidCallback onCommit;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final label = preview.label.trim().isEmpty
        ? preview.endpointId
        : preview.label.trim();
    final fingerprint = preview.hasIdentity()
        ? preview.identity.deviceFingerprint.trim()
        : '';
    final policyChanges = <String>[
      if (preview.connectModeChanged) 'Connection mode',
      if (preview.selectionPolicyChanged) 'Route preference',
    ];
    return SafeArea(
      top: false,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 520),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Container(
                          width: 44,
                          height: 44,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: palette.surfaceRaised,
                            border: Border.all(color: palette.border),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Icon(
                            Icons.computer_rounded,
                            size: 23,
                            color: palette.text,
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                label,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                  fontSize: 17,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              const SizedBox(height: 2),
                              Text(
                                preview.endpointId,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: TextStyle(
                                  color: palette.muted,
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 8,
                            vertical: 4,
                          ),
                          decoration: BoxDecoration(
                            color: palette.success.withValues(alpha: 0.12),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(
                            'VERIFIED',
                            style: TextStyle(
                              color: palette.success,
                              fontSize: 10,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                  if (fingerprint.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _ShareDetail(
                      label: 'DEVICE IDENTITY',
                      child: SelectionArea(
                        child: Text(
                          fingerprint,
                          style: TextStyle(
                            color: palette.muted,
                            fontFamily: 'JetBrainsMono',
                            fontSize: 11,
                            height: 1.5,
                          ),
                        ),
                      ),
                    ),
                  ],
                  const SizedBox(height: 20),
                  Text(
                    'ROUTE CHANGES',
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  if (preview.routeDiffs.isEmpty)
                    _ShareDetail(
                      label: 'NO ROUTE CHANGES',
                      child: Text(
                        'Existing routes stay unchanged.',
                        style: TextStyle(color: palette.muted, fontSize: 12),
                      ),
                    )
                  else
                    Container(
                      clipBehavior: Clip.antiAlias,
                      decoration: BoxDecoration(
                        color: palette.surface,
                        border: Border.all(color: palette.border),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Column(
                        children: [
                          for (
                            var index = 0;
                            index < preview.routeDiffs.length;
                            index++
                          ) ...[
                            _ShareRouteDiffRow(diff: preview.routeDiffs[index]),
                            if (index != preview.routeDiffs.length - 1)
                              Divider(height: 1, color: palette.border),
                          ],
                        ],
                      ),
                    ),
                  const SizedBox(height: 12),
                  _ShareDetail(
                    label: policyChanges.isEmpty
                        ? 'POLICY UNCHANGED'
                        : 'POLICY CHANGES',
                    child: Text(
                      policyChanges.isEmpty
                          ? 'Connection policy stays unchanged.'
                          : '${policyChanges.join(' and ')} will be updated.',
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  ),
                  const SizedBox(height: 12),
                  _ShareDetail(
                    label: 'CREDENTIALS STAY LOCAL',
                    icon: Icons.key_off_outlined,
                    child: Text(
                      preview.credentialDescriptors.isEmpty
                          ? 'No local credential setup is required by this import.'
                          : '${preview.credentialDescriptors.length} route credential${preview.credentialDescriptors.length == 1 ? '' : 's'} must be prepared on this device after import.',
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  ),
                  if (error != null) ...[
                    const SizedBox(height: 12),
                    Semantics(
                      liveRegion: true,
                      child: Text(
                        error!,
                        style: TextStyle(color: palette.danger, fontSize: 12),
                      ),
                    ),
                  ],
                  if (submitting) ...[
                    const SizedBox(height: 12),
                    const LinearProgressIndicator(minHeight: 2),
                  ],
                  const SizedBox(height: 20),
                  SizedBox(
                    height: 48,
                    child: FilledButton.icon(
                      onPressed: submitting ? null : onCommit,
                      icon: const Icon(Icons.download_rounded, size: 18),
                      label: const Text('Import configuration'),
                    ),
                  ),
                  const SizedBox(height: 8),
                  TextButton(
                    onPressed: submitting ? null : onCancel,
                    child: const Text('Scan a different code'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _ShareDetail extends StatelessWidget {
  const _ShareDetail({required this.label, required this.child, this.icon});

  final String label;
  final Widget child;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (icon != null) ...[
            Icon(icon, color: palette.muted, size: 18),
            const SizedBox(width: 10),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: TextStyle(
                    color: palette.muted,
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 5),
                child,
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _ShareRouteDiffRow extends StatelessWidget {
  const _ShareRouteDiffRow({required this.diff});

  final EndpointShareRouteDiff diff;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final action = diff.action.trim().isEmpty ? 'update' : diff.action.trim();
    final actionColor = switch (action.toLowerCase()) {
      'add' || 'added' || 'create' => palette.success,
      'remove' || 'removed' || 'delete' => palette.danger,
      _ => palette.accent,
    };
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      child: Row(
        children: [
          Icon(Icons.alt_route_rounded, color: palette.muted, size: 18),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  diff.routeId,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                Text(
                  diff.routeKind.trim().isEmpty
                      ? 'Route'
                      : diff.routeKind.trim(),
                  style: TextStyle(color: palette.muted, fontSize: 11),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            action.toUpperCase(),
            style: TextStyle(
              color: actionColor,
              fontSize: 10,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}
