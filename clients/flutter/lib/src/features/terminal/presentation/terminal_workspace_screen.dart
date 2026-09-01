import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:fixnum/fixnum.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/apipb/history.pb.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../../native/android_ime_inset_platform.dart';
import '../../../native/android_terminal_input_platform.dart';
import '../../../native/external_uri_platform.dart';
import '../../../shared/presentation/fuzzy_highlight_text.dart';
import '../../endpoints/data/connection_repository.dart';
import '../../endpoints/data/endpoint_repository.dart';
import '../../files/presentation/file_manager_screen.dart';
import '../../files/presentation/file_transfer_sheet.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../../terminal/data/terminal_keyboard_mode_store.dart';
import '../../terminal/data/terminal_pin_store.dart';
import '../../terminal/domain/history_interaction.dart';
import '../../terminal/domain/history_store.dart';
import '../../terminal/domain/live_screen_store.dart';
import '../../terminal/domain/terminal_input_fanout.dart';
import '../../terminal/domain/terminal_form.dart';
import '../../terminal/domain/terminal_inventory.dart';
import '../../terminal/domain/terminal_links.dart';
import '../../terminal/domain/terminal_modifiers.dart';
import '../../terminal/domain/terminal_petal_menu_preferences.dart';
import '../../terminal/domain/terminal_quick_action.dart';
import '../../terminal/domain/terminal_settings.dart';
import '../../terminal/domain/terminal_soft_input.dart';
import '../../terminal/domain/terminal_split_layout.dart';
import 'terminal_canvas.dart';
import 'terminal_command_bar.dart';
import 'terminal_keyboard_inset.dart';
import 'terminal_petal_menu.dart';
import 'terminal_program_icon.dart';
import 'terminal_quick_keys_panel.dart';
import 'terminal_recovery_notice.dart';
import 'terminal_resource_panel.dart';
import 'terminal_switcher_sheet.dart';
import 'terminal_ui_theme.dart';

final class TerminalWorkspaceScreen extends ConsumerStatefulWidget {
  const TerminalWorkspaceScreen({
    super.key,
    required this.endpointId,
    this.terminalId,
    this.label,
  });

  final String endpointId;
  final String? terminalId;
  final String? label;

  @override
  ConsumerState<TerminalWorkspaceScreen> createState() =>
      _TerminalWorkspaceScreenState();
}

final class _TerminalWorkspaceScreenState
    extends ConsumerState<TerminalWorkspaceScreen>
    with WidgetsBindingObserver {
  final _surfaceKey = GlobalKey<_ActiveTerminalState>();
  late final ui.FlutterView _view;
  late final TerminalKeyboardInsetStabilizer _keyboardInsetStabilizer;
  StreamSubscription<double>? _nativeImeInsetSubscription;
  final ValueNotifier<double> _keyboardVisualInset = ValueNotifier(0);
  String? _activeTerminalId;
  double _keyboardInset = 0;
  bool _usingNativeKeyboardInsets = false;

  @override
  void initState() {
    super.initState();
    _view = WidgetsBinding.instance.platformDispatcher.views.first;
    _keyboardInsetStabilizer = TerminalKeyboardInsetStabilizer(
      onInsetChanged: (inset) {
        if (mounted) setState(() => _keyboardInset = inset);
      },
    );
    WidgetsBinding.instance.addObserver(this);
    if (defaultTargetPlatform == TargetPlatform.android) {
      final platform = AndroidImeInsetPlatform.instance;
      _nativeImeInsetSubscription = platform.physicalInsets.listen(
        _updateNativeKeyboardInset,
      );
      unawaited(
        platform.currentPhysicalInset().then((inset) {
          if (inset != null) _updateNativeKeyboardInset(inset);
        }),
      );
    }
    WidgetsBinding.instance.addPostFrameCallback((_) => _updateKeyboardInset());
  }

  @override
  void didChangeMetrics() => _updateKeyboardInset();

  void _updateKeyboardInset() {
    if (_usingNativeKeyboardInsets) return;
    final inset = _view.viewInsets.bottom / _view.devicePixelRatio;
    _keyboardVisualInset.value = inset;
    _keyboardInsetStabilizer.update(inset);
  }

  void _updateNativeKeyboardInset(double physicalInset) {
    if (!mounted) return;
    _usingNativeKeyboardInsets = true;
    final inset = math.max(0.0, physicalInset / _view.devicePixelRatio);
    _keyboardInsetStabilizer.update(inset);
    _keyboardVisualInset.value = inset;
  }

  @override
  void didUpdateWidget(TerminalWorkspaceScreen oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId ||
        oldWidget.terminalId != widget.terminalId) {
      _activeTerminalId = null;
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_nativeImeInsetSubscription?.cancel());
    _keyboardVisualInset.dispose();
    _keyboardInsetStabilizer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.terminalId == null) return _buildWorkspace(context);
    final settings =
        ref.watch(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    return Theme(
      data: terminalUiThemeData(settings.theme),
      child: Builder(
        builder: (context) {
          final palette = AnyttyPalette.of(context);
          return AnnotatedRegion<SystemUiOverlayStyle>(
            value: anyttySystemUiOverlayStyle(
              Theme.of(context).brightness,
              palette: palette,
            ),
            child: TerminalKeyboardMediaQuery(child: _buildWorkspace(context)),
          );
        },
      ),
    );
  }

  Widget _buildWorkspace(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final endpointLabel = widget.label?.trim().isNotEmpty == true
        ? widget.label!.trim()
        : widget.endpointId;
    final selectedTerminal = widget.terminalId;
    final headerTerminalId = selectedTerminal == null
        ? null
        : _activeTerminalId ?? selectedTerminal;
    final terminalItems = ref
        .watch(terminalListProvider(widget.endpointId))
        .valueOrNull;
    final connectionDiagnostics = selectedTerminal == null
        ? ref.watch(connectionDiagnosticsProvider(widget.endpointId))
        : null;
    final activeTerminal = selectedTerminal == null
        ? null
        : terminalItems
              ?.where((terminal) => terminal.ref.terminalId == headerTerminalId)
              .firstOrNull;
    final title = activeTerminal == null
        ? endpointLabel
        : _terminalDisplayName(activeTerminal);
    final listResourceTotals = selectedTerminal == null && terminalItems != null
        ? terminalResourceTotals(terminalItems)
        : null;
    final listSubtitle = listResourceTotals == null
        ? anyttyText(context, en: 'Terminals', zh: '终端')
        : anyttyText(
            context,
            en: 'Terminals · ${listResourceTotals.runningCount} running · CPU ${(listResourceTotals.cpuX100 / 100).toStringAsFixed(1)}% · RAM ${formatResourceBytes(listResourceTotals.memoryBytes)}',
            zh: '终端 · ${listResourceTotals.runningCount} 运行 · CPU ${(listResourceTotals.cpuX100 / 100).toStringAsFixed(1)}% · 内存 ${formatResourceBytes(listResourceTotals.memoryBytes)}',
          );
    final cwd = activeTerminal == null
        ? ''
        : (activeTerminal.liveCwd.isNotEmpty
              ? activeTerminal.liveCwd
              : activeTerminal.cwd);
    final terminalReady = headerTerminalId != null
        ? ref
              .watch(
                terminalConnectionProvider((
                  endpointId: widget.endpointId,
                  terminalId: headerTerminalId,
                )),
              )
              .hasValue
        : false;
    final VoidCallback? terminalSwitcherAction =
        selectedTerminal != null &&
            terminalItems != null &&
            terminalItems.isNotEmpty
        ? () => _openTerminalSwitcher(
            context: context,
            endpointId: widget.endpointId,
            endpointLabel: endpointLabel,
            terminals: terminalItems,
            activeTerminalId: headerTerminalId,
          )
        : null;
    final workspace = PopScope<Object?>(
      canPop: selectedTerminal == null,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop || selectedTerminal == null) return;
        if (_surfaceKey.currentState?.handleBack() != true) context.pop();
      },
      child: Scaffold(
        resizeToAvoidBottomInset: selectedTerminal == null,
        backgroundColor: palette.background,
        appBar: AppBar(
          toolbarHeight: 36,
          leadingWidth: 40,
          titleSpacing: 0,
          actionsIconTheme: const IconThemeData(size: 17),
          backgroundColor: selectedTerminal == null
              ? palette.background
              : palette.surface,
          foregroundColor: palette.text,
          title: Tooltip(
            message: selectedTerminal == null
                ? endpointLabel
                : 'Switch terminal: $endpointLabel · $title${cwd.isEmpty ? '' : ' · $cwd'}',
            child: Semantics(
              button: terminalSwitcherAction != null,
              label: terminalSwitcherAction == null
                  ? null
                  : 'Switch terminal, $title, $endpointLabel',
              onTap: terminalSwitcherAction,
              excludeSemantics: terminalSwitcherAction != null,
              child: InkWell(
                onTap: terminalSwitcherAction,
                child: ConstrainedBox(
                  constraints: const BoxConstraints(minHeight: 34),
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 12,
                            height: 1,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        Text(
                          selectedTerminal == null
                              ? listSubtitle
                              : cwd.isEmpty
                              ? endpointLabel
                              : '$endpointLabel  ·  $cwd',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: palette.muted,
                            fontSize: 8,
                            height: 1,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
          actions: [
            FileTransferCenterAction(
              controller: ref.read(fileTransferControllerProvider),
              dimension: 38,
              iconSize: 17,
            ),
            if (selectedTerminal == null)
              IconButton(
                tooltip: anyttyText(context, en: 'Files', zh: '文件管理'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 36,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: () => showAnyttyFileManager(
                  context: context,
                  endpointId: widget.endpointId,
                  endpointLabel: endpointLabel,
                  initialPath: '/',
                ),
                icon: const Icon(Icons.folder_outlined, size: 17),
              ),
            if (selectedTerminal == null)
              _WorkspaceNetworkAction(
                diagnostics: connectionDiagnostics!,
                onPressed: () => _showWorkspaceConnectionDetails(
                  context: context,
                  ref: ref,
                  endpointId: widget.endpointId,
                  endpointLabel: endpointLabel,
                ),
              ),
            if (selectedTerminal == null)
              IconButton(
                tooltip: anyttyText(context, en: 'Create terminal', zh: '新建终端'),
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: () => _createTerminal(
                  context: context,
                  ref: ref,
                  endpointId: widget.endpointId,
                  label: endpointLabel,
                ),
                icon: const Icon(Icons.add_rounded),
              ),
            if (selectedTerminal != null && activeTerminal != null)
              IconButton(
                tooltip: 'Files',
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: terminalReady
                    ? () => showAnyttyFileManager(
                        context: context,
                        endpointId: widget.endpointId,
                        endpointLabel: endpointLabel,
                        initialPath: cwd,
                      )
                    : null,
                icon: const Icon(Icons.folder_outlined, size: 17),
              ),
            if (selectedTerminal != null && activeTerminal != null)
              IconButton(
                tooltip: 'Split below',
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: terminalReady
                    ? () => _surfaceKey.currentState?.splitActiveBelow()
                    : null,
                icon: const Icon(Icons.horizontal_split_rounded, size: 17),
              ),
            if (selectedTerminal != null && activeTerminal != null)
              IconButton(
                tooltip: 'Terminal tools',
                constraints: const BoxConstraints.tightFor(
                  width: 38,
                  height: 38,
                ),
                padding: const EdgeInsets.all(9),
                onPressed: terminalReady
                    ? () {
                        unawaited(HapticFeedback.mediumImpact());
                        _surfaceKey.currentState?.toggleTools();
                      }
                    : null,
                icon: const Icon(Icons.tune_rounded, size: 17),
              ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: selectedTerminal == null
              ? _TerminalList(
                  endpointId: widget.endpointId,
                  label: endpointLabel,
                )
              : _ActiveTerminal(
                  endpointId: widget.endpointId,
                  endpointLabel: endpointLabel,
                  terminalId: selectedTerminal,
                  keyboardVisualInset: _keyboardVisualInset,
                  keyboardInset: _keyboardInset,
                  settledKeyboardInset: _keyboardInset,
                  surfaceKey: _surfaceKey,
                  onActiveTerminalChanged: (terminalId) {
                    if (mounted && _activeTerminalId != terminalId) {
                      setState(() => _activeTerminalId = terminalId);
                    }
                  },
                ),
        ),
      ),
    );
    return workspace;
  }

  Future<void> _openTerminalSwitcher({
    required BuildContext context,
    required String endpointId,
    required String endpointLabel,
    required List<TerminalInfo> terminals,
    required String? activeTerminalId,
  }) async {
    final currentTerminals = await _orderSwitcherTerminals(
      endpointId,
      terminals,
    );
    if (!context.mounted) return;

    final endpoints = <TerminalSwitcherEndpoint>[];
    try {
      final registry = await ref.read(endpointRegistryProvider.future);
      final saved = registry.endpoints
          .where(
            (endpoint) => endpoint.enabled || endpoint.endpointId == endpointId,
          )
          .toList(growable: false);
      final current = saved.where(
        (endpoint) => endpoint.endpointId == endpointId,
      );
      final others = saved.where(
        (endpoint) => endpoint.endpointId != endpointId,
      );
      for (final endpoint in [...current, ...others]) {
        final label = endpoint.label.trim().isEmpty
            ? endpoint.endpointId
            : endpoint.label.trim();
        endpoints.add(
          TerminalSwitcherEndpoint(
            endpointId: endpoint.endpointId,
            label: label,
            current: endpoint.endpointId == endpointId,
            terminals: endpoint.endpointId == endpointId
                ? currentTerminals
                : null,
            activeTerminalId: endpoint.endpointId == endpointId
                ? activeTerminalId
                : null,
          ),
        );
      }
    } catch (_) {
      // The current endpoint remains switchable if registry refresh is offline.
    }
    if (!endpoints.any((endpoint) => endpoint.current)) {
      endpoints.insert(
        0,
        TerminalSwitcherEndpoint(
          endpointId: endpointId,
          label: endpointLabel,
          current: true,
          terminals: currentTerminals,
          activeTerminalId: activeTerminalId,
        ),
      );
    }
    if (!context.mounted) return;
    final selection = await showAnyttyTerminalSwitcher(
      context: context,
      endpoints: endpoints,
      loadTerminals: (remoteEndpointId) async => _orderSwitcherTerminals(
        remoteEndpointId,
        await ref.read(terminalListProvider(remoteEndpointId).future),
      ),
    );
    if (selection == null || !context.mounted) return;
    if (selection.endpointId == endpointId) {
      _surfaceKey.currentState?.switchTerminal(selection.terminalId);
      return;
    }
    context.go(
      '/terminal/${Uri.encodeComponent(selection.endpointId)}/'
      '${Uri.encodeComponent(selection.terminalId)}'
      '?label=${Uri.encodeQueryComponent(selection.endpointLabel)}',
    );
  }

  Future<List<TerminalInfo>> _orderSwitcherTerminals(
    String endpointId,
    List<TerminalInfo> terminals,
  ) async {
    List<String> pinnedIds;
    try {
      pinnedIds = await const TerminalPinStore().load(endpointId);
    } catch (_) {
      pinnedIds = const [];
    }
    final running = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.running,
      ),
      pinnedIds,
    );
    final exited = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.exited,
      ),
      pinnedIds,
    );
    final known = {...running, ...exited};
    return [
      ...running,
      ...exited,
      ...terminals.where((terminal) => !known.contains(terminal)),
    ];
  }
}

final class _WorkspaceNetworkAction extends StatelessWidget {
  const _WorkspaceNetworkAction({
    required this.diagnostics,
    required this.onPressed,
  });

  final AsyncValue<EndpointConnectionDiagnostics> diagnostics;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final snapshot = diagnostics.valueOrNull?.snapshot;
    final connected = snapshot?.connected ?? false;
    final label = snapshot == null
        ? diagnostics.isLoading
              ? '...'
              : '--'
        : _workspaceConnectionKind(snapshot);
    final color = connected ? palette.accent : palette.muted;
    final semanticLabel = anyttyText(
      context,
      en: 'Network status, $label, show details',
      zh: '网络状态，$label，查看详情',
    );
    return Semantics(
      button: true,
      label: semanticLabel,
      excludeSemantics: true,
      child: Tooltip(
        message: semanticLabel,
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            key: const ValueKey('terminal-network-status'),
            onTap: onPressed,
            child: SizedBox(
              width: 44,
              height: 38,
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.route_rounded, size: 16, color: color),
                  const SizedBox(height: 1),
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.fade,
                    style: TextStyle(
                      color: color,
                      fontSize: 7,
                      height: 1,
                      fontWeight: FontWeight.w800,
                    ),
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

Future<void> _showWorkspaceConnectionDetails({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required String endpointLabel,
}) async {
  ref.invalidate(connectionDiagnosticsProvider(endpointId));
  final openSettings = await showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: AnyttyPalette.of(context).surface,
    barrierColor: AnyttyPalette.of(context).overlay,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
    ),
    builder: (context) => _WorkspaceConnectionDetailsSheet(
      endpointId: endpointId,
      endpointLabel: endpointLabel,
    ),
  );
  if (openSettings == true && context.mounted) {
    context.push(
      '/connection/${Uri.encodeComponent(endpointId)}'
      '?label=${Uri.encodeQueryComponent(endpointLabel)}',
    );
  }
}

final class _WorkspaceConnectionDetailsSheet extends ConsumerWidget {
  const _WorkspaceConnectionDetailsSheet({
    required this.endpointId,
    required this.endpointLabel,
  });

  final String endpointId;
  final String endpointLabel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AnyttyPalette.of(context);
    final diagnostics = ref.watch(connectionDiagnosticsProvider(endpointId));
    final value = diagnostics.valueOrNull;
    final snapshot = value?.snapshot;
    return SafeArea(
      top: false,
      child: FractionallySizedBox(
        heightFactor: 0.78,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(18, 4, 18, 14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          anyttyText(context, en: 'Network status', zh: '网络状态'),
                          style: const TextStyle(
                            fontSize: 19,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          endpointLabel,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.muted, fontSize: 11),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: anyttyText(context, en: 'Close', zh: '关闭'),
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              if (diagnostics.isLoading && snapshot == null)
                const Expanded(
                  child: Center(
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                )
              else if (snapshot == null)
                Expanded(
                  child: _WorkspaceConnectionError(
                    message: anyttyText(
                      context,
                      en: 'Network details are temporarily unavailable.',
                      zh: '暂时无法读取网络详情。',
                    ),
                    onRetry: () => ref.invalidate(
                      connectionDiagnosticsProvider(endpointId),
                    ),
                  ),
                )
              else ...[
                _WorkspaceConnectionSummary(snapshot: snapshot),
                if (diagnostics.isRefreshing) ...[
                  const SizedBox(height: 8),
                  const LinearProgressIndicator(minHeight: 2),
                ],
                const SizedBox(height: 12),
                Expanded(
                  child: ListView(
                    padding: EdgeInsets.zero,
                    children: [
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Connection type',
                          zh: '连接类型',
                        ),
                        value: _workspaceConnectionKind(snapshot),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Route', zh: '连接路由'),
                        value:
                            '${_workspaceRouteLabel(context, snapshot.routeKind)}  ·  ${snapshot.routeId.trim().isEmpty ? '--' : snapshot.routeId}',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Observed path',
                          zh: '实际路径',
                        ),
                        value: _workspacePathLabel(
                          context,
                          snapshot.observedPath,
                        ),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Latency', zh: '延迟'),
                        value: _workspaceLatency(
                          snapshot.roundTripNanos.toInt(),
                        ),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Network', zh: '网络类型'),
                        value: _workspaceNetworkClass(snapshot.networkClass),
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Transport', zh: '传输协议'),
                        value:
                            '${_workspaceTransportLabel(snapshot.localProtocol)} / ${_workspaceTransportLabel(snapshot.remoteProtocol)}',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Local candidate',
                          zh: '本地候选',
                        ),
                        value:
                            '${_workspaceAddress(snapshot.localIp, snapshot.localPort)}  (${_workspaceCandidateLabel(snapshot.localCandidateType)})',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(
                          context,
                          en: 'Remote candidate',
                          zh: '远端候选',
                        ),
                        value:
                            '${_workspaceAddress(snapshot.remoteIp, snapshot.remotePort)}  (${_workspaceCandidateLabel(snapshot.remoteCandidateType)})',
                      ),
                      _WorkspaceConnectionDetailRow(
                        label: anyttyText(context, en: 'Traffic', zh: '流量'),
                        value:
                            '↓ ${formatResourceBytes(snapshot.bytesReceived.toInt())}  ↑ ${formatResourceBytes(snapshot.bytesSent.toInt())}',
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
                SizedBox(
                  height: 48,
                  child: OutlinedButton.icon(
                    onPressed: () => Navigator.pop(context, true),
                    icon: const Icon(Icons.tune_rounded, size: 18),
                    label: Text(
                      anyttyText(
                        context,
                        en: 'Connection settings',
                        zh: '连接设置',
                      ),
                    ),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

final class _WorkspaceConnectionSummary extends StatelessWidget {
  const _WorkspaceConnectionSummary({required this.snapshot});

  final ConnectionSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final connected = snapshot.connected;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: connected
            ? palette.accent.withValues(alpha: 0.11)
            : palette.surfaceRaised,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: connected
              ? palette.accent.withValues(alpha: 0.35)
              : palette.border,
        ),
      ),
      child: Row(
        children: [
          Icon(
            connected ? Icons.route_rounded : Icons.cloud_off_outlined,
            color: connected ? palette.accent : palette.muted,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _workspaceConnectionKind(snapshot),
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  connected
                      ? anyttyText(context, en: 'Connected', zh: '连接正常')
                      : anyttyText(context, en: 'Disconnected', zh: '当前未连接'),
                  style: TextStyle(color: palette.muted, fontSize: 11),
                ),
              ],
            ),
          ),
          Text(
            _workspaceLatency(snapshot.roundTripNanos.toInt()),
            style: const TextStyle(
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 12,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}

final class _WorkspaceConnectionDetailRow extends StatelessWidget {
  const _WorkspaceConnectionDetailRow({
    required this.label,
    required this.value,
  });

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      constraints: const BoxConstraints(minHeight: 44),
      padding: const EdgeInsets.symmetric(vertical: 9),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 98,
            child: Text(
              label,
              style: TextStyle(color: palette.muted, fontSize: 11),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              value,
              textAlign: TextAlign.end,
              style: const TextStyle(
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 11,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _WorkspaceConnectionError extends StatelessWidget {
  const _WorkspaceConnectionError({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.cloud_off_outlined, color: palette.muted, size: 30),
          const SizedBox(height: 10),
          Text(message, style: TextStyle(color: palette.muted)),
          const SizedBox(height: 12),
          OutlinedButton(
            onPressed: onRetry,
            child: Text(anyttyText(context, en: 'Retry', zh: '重试')),
          ),
        ],
      ),
    );
  }
}

String _workspaceConnectionKind(ConnectionSnapshot snapshot) {
  if (!snapshot.connected) return 'Offline';
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH) {
    return 'SSH';
  }
  if (snapshot.observedPath ==
      ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return 'Relay';
  }
  if (snapshot.observedPath ==
      ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT) {
    return 'P2P';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT) {
    return 'Direct';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL) {
    return 'Local';
  }
  if (snapshot.routeKind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD) {
    return 'Cloud';
  }
  return 'Connected';
}

String _workspaceRouteLabel(BuildContext context, ConnectionRouteKind kind) {
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL) {
    return anyttyText(context, en: 'Local', zh: '本地');
  }
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT) return 'Direct';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH) return 'SSH';
  if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD) {
    return anyttyText(context, en: 'Cloud', zh: '云端');
  }
  return '--';
}

String _workspacePathLabel(BuildContext context, ConnectionObservedPath path) {
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT) {
    return anyttyText(context, en: 'Direct peer path', zh: 'P2P 直连');
  }
  if (path == ConnectionObservedPath.CONNECTION_OBSERVED_PATH_SINGLE_RELAY) {
    return anyttyText(context, en: 'Single relay path', zh: '单中继路径');
  }
  return anyttyText(context, en: 'Not reported', zh: '未上报');
}

String _workspaceTransportLabel(ConnectionTransport transport) {
  if (transport == ConnectionTransport.CONNECTION_TRANSPORT_UDP) return 'UDP';
  if (transport == ConnectionTransport.CONNECTION_TRANSPORT_TCP) return 'TCP';
  return '--';
}

String _workspaceCandidateLabel(ConnectionCandidateType candidate) {
  if (candidate == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_HOST) {
    return 'host';
  }
  if (candidate ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE) {
    return 'srflx';
  }
  if (candidate ==
      ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_PEER_REFLEXIVE) {
    return 'prflx';
  }
  if (candidate == ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_RELAY) {
    return 'relay';
  }
  return '--';
}

String _workspaceAddress(String ip, int port) {
  final value = ip.trim();
  if (value.isEmpty) return '--';
  if (port <= 0) return value;
  return value.contains(':') ? '[$value]:$port' : '$value:$port';
}

String _workspaceLatency(int nanoseconds) {
  if (nanoseconds <= 0) return '--';
  final milliseconds = nanoseconds / 1000000;
  return milliseconds >= 10
      ? '${milliseconds.round()} ms'
      : '${milliseconds.toStringAsFixed(1)} ms';
}

String _workspaceNetworkClass(String networkClass) {
  final value = networkClass.trim();
  if (value.isEmpty) return '--';
  if (value.toLowerCase() == 'wifi') return 'Wi-Fi';
  return value;
}

enum _TerminalAction { rename, restart, end, remove }

enum _TerminalToolAction {
  commandBar,
  quickKeys,
  keyboard,
  enter,
  escape,
  tab,
  backspace,
  delete,
  interrupt,
  eof,
  suspend,
  clear,
  arrowLeft,
  arrowDown,
  arrowUp,
  arrowRight,
  home,
  end,
  pageUp,
  pageDown,
  paste,
  history,
  search,
  selection,
  copyScreen,
  resources,
  files,
  split,
  splitRows,
  splitColumns,
  syncInput,
  resize,
  reconnect,
  settings,
}

enum _TerminalRowAction {
  rename,
  restart,
  end,
  remove,
  pin,
  unpin,
  moveUp,
  moveDown,
}

final class _TerminalCreateInput {
  const _TerminalCreateInput({
    required this.name,
    required this.command,
    required this.cwd,
    required this.environment,
    required this.cols,
    required this.rows,
    required this.sizeLockMode,
  });

  final String name;
  final List<String> command;
  final String cwd;
  final List<String> environment;
  final int cols;
  final int rows;
  final String sizeLockMode;
}

Future<void> _createTerminal({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required String label,
}) async {
  final messenger = ScaffoldMessenger.of(context);
  messenger.showSnackBar(
    const SnackBar(content: Text('Loading terminal defaults')),
  );
  try {
    final session = await ref.read(endpointSessionProvider(endpointId).future);
    final defaults = await session.terminalDefaults();
    messenger.hideCurrentSnackBar();
    if (!context.mounted) return;
    final input = await showModalBottomSheet<_TerminalCreateInput>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: false,
      backgroundColor: Colors.transparent,
      barrierColor: Colors.black54,
      builder: (context) => _TerminalCreateSheet(defaults: defaults),
    );
    if (input == null || !context.mounted) return;
    messenger.showSnackBar(const SnackBar(content: Text('Creating terminal')));
    final terminal = await session.createTerminal(
      name: input.name,
      command: input.command,
      cwd: input.cwd,
      environment: input.environment,
      cols: input.cols,
      rows: input.rows,
      sizeLockMode: input.sizeLockMode,
    );
    ref.invalidate(terminalListProvider(endpointId));
    if (!context.mounted) return;
    messenger.hideCurrentSnackBar();
    context.push(
      '/terminal/${Uri.encodeComponent(endpointId)}/'
      '${Uri.encodeComponent(terminal.ref.terminalId)}'
      '?label=${Uri.encodeQueryComponent(label)}',
    );
  } catch (error) {
    messenger.hideCurrentSnackBar();
    if (!context.mounted) return;
    messenger.showSnackBar(SnackBar(content: Text(error.toString())));
  }
}

Future<bool> _manageTerminal({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required TerminalInfo terminal,
  required _TerminalAction action,
}) async {
  String? nextName;
  if (action == _TerminalAction.rename) {
    final controller = TextEditingController(text: terminal.name);
    nextName = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Rename terminal'),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 120,
          textInputAction: TextInputAction.done,
          onSubmitted: (value) => Navigator.pop(context, value.trim()),
          decoration: const InputDecoration(labelText: 'Name'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, controller.text.trim()),
            child: const Text('Rename'),
          ),
        ],
      ),
    );
    controller.dispose();
    if (nextName == null || !context.mounted) return false;
  }

  if (action == _TerminalAction.end || action == _TerminalAction.remove) {
    final destructive = action == _TerminalAction.remove;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(destructive ? 'Delete terminal record?' : 'End process?'),
        content: Text(
          destructive
              ? 'The exited terminal and its saved history will be removed.'
              : 'The process will stop. Its terminal record and history will remain.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor: const Color(0xffdc2626),
                  )
                : null,
            onPressed: () => Navigator.pop(context, true),
            child: Text(destructive ? 'Delete' : 'End'),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return false;
  }

  try {
    final session = await ref.read(endpointSessionProvider(endpointId).future);
    switch (action) {
      case _TerminalAction.rename:
        await session.renameTerminal(terminal, nextName!);
      case _TerminalAction.restart:
        await session.restartTerminal(terminal.ref);
      case _TerminalAction.end:
        await session.killTerminal(terminal.ref);
      case _TerminalAction.remove:
        await session.removeTerminal(terminal.ref);
    }
    ref.invalidate(terminalListProvider(endpointId));
    if (context.mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(_terminalActionSuccess(action))));
    }
    return true;
  } catch (error) {
    if (context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
    return false;
  }
}

String _terminalActionSuccess(_TerminalAction action) {
  return switch (action) {
    _TerminalAction.rename => 'Terminal renamed',
    _TerminalAction.restart => 'Terminal restarted',
    _TerminalAction.end => 'Terminal process ended',
    _TerminalAction.remove => 'Terminal record deleted',
  };
}

final class _TerminalCreateSheet extends StatefulWidget {
  const _TerminalCreateSheet({required this.defaults});

  final TerminalDefaults defaults;

  @override
  State<_TerminalCreateSheet> createState() => _TerminalCreateSheetState();
}

final class _TerminalCreateSheetState extends State<_TerminalCreateSheet> {
  late final TextEditingController _name;
  late final TextEditingController _command;
  late final TextEditingController _cwd;
  late final TextEditingController _environmentPaste;
  final List<_TerminalEnvironmentControllers> _environment = [];
  String _sizeLockMode = 'off';
  bool _pasteOpen = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController();
    _command = TextEditingController(
      text: formatTerminalCommand(widget.defaults.defaultCommand),
    );
    _cwd = TextEditingController(text: widget.defaults.defaultCwd);
    _environmentPaste = TextEditingController();
  }

  @override
  void dispose() {
    _name.dispose();
    _command.dispose();
    _cwd.dispose();
    _environmentPaste.dispose();
    for (final entry in _environment) {
      entry.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final keyboardInset = MediaQuery.viewInsetsOf(context).bottom;
    return AnimatedPadding(
      duration: AnyttyMotion.resolve(
        context,
        const Duration(milliseconds: 200),
      ),
      curve: Curves.easeOutCubic,
      padding: EdgeInsets.only(bottom: keyboardInset),
      child: FractionallySizedBox(
        alignment: Alignment.bottomCenter,
        heightFactor: 0.85,
        child: Material(
          color: palette.background,
          clipBehavior: Clip.antiAlias,
          shape: RoundedRectangleBorder(
            borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
            side: BorderSide(color: palette.border),
          ),
          child: SafeArea(
            top: false,
            child: Column(
              children: [
                const SizedBox(height: 10),
                Container(
                  width: 48,
                  height: 4,
                  decoration: BoxDecoration(
                    color: palette.borderStrong,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                SizedBox(
                  height: 54,
                  child: Row(
                    children: [
                      const SizedBox(width: 20),
                      Expanded(
                        child: Text(
                          'New terminal',
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 17,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      IconButton(
                        tooltip: 'Close New terminal',
                        onPressed: () => Navigator.pop(context),
                        icon: const Icon(Icons.close_rounded, size: 20),
                      ),
                      const SizedBox(width: 8),
                    ],
                  ),
                ),
                Divider(height: 1, color: palette.border),
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 20),
                    children: [
                      _labelledField(
                        context,
                        label: 'Name',
                        child: TextField(
                          controller: _name,
                          maxLength: 120,
                          textInputAction: TextInputAction.next,
                          decoration: _fieldDecoration(
                            context,
                            counterText: '',
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: 'Command',
                        child: TextField(
                          controller: _command,
                          autocorrect: false,
                          enableSuggestions: false,
                          textInputAction: TextInputAction.next,
                          style: const TextStyle(
                            fontFamily: 'JetBrainsMonoNerd',
                            fontSize: 15,
                          ),
                          decoration: _fieldDecoration(context),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: 'Working directory',
                        child: TextField(
                          controller: _cwd,
                          autocorrect: false,
                          enableSuggestions: false,
                          textInputAction: TextInputAction.next,
                          style: const TextStyle(
                            fontFamily: 'JetBrainsMonoNerd',
                            fontSize: 15,
                          ),
                          decoration: _fieldDecoration(context),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _buildEnvironmentEditor(context),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: 'Size policy',
                        child: DropdownButtonFormField<String>(
                          initialValue: _sizeLockMode,
                          isExpanded: true,
                          decoration: _fieldDecoration(context),
                          items: const [
                            DropdownMenuItem(
                              value: 'off',
                              child: Text('Resizable'),
                            ),
                            DropdownMenuItem(
                              value: 'warn',
                              child: Text('Warn before resizing'),
                            ),
                            DropdownMenuItem(
                              value: 'lock',
                              child: Text('Locked size'),
                            ),
                          ],
                          onChanged: (value) {
                            if (value != null) _sizeLockMode = value;
                          },
                        ),
                      ),
                      if (_error case final error?) ...[
                        const SizedBox(height: 12),
                        Text(
                          error,
                          style: const TextStyle(
                            color: Color(0xffdc2626),
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
                Container(
                  decoration: BoxDecoration(
                    color: palette.background,
                    border: Border(top: BorderSide(color: palette.border)),
                  ),
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                  child: SizedBox(
                    width: double.infinity,
                    height: 48,
                    child: FilledButton.icon(
                      onPressed: _submit,
                      icon: const Icon(Icons.add_rounded, size: 18),
                      label: const Text('Create terminal'),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildEnvironmentEditor(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          'Environment',
          style: TextStyle(
            color: palette.text,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        for (var index = 0; index < _environment.length; index++) ...[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                flex: 9,
                child: TextField(
                  controller: _environment[index].key,
                  autocorrect: false,
                  enableSuggestions: false,
                  decoration: _fieldDecoration(context, hintText: 'Key'),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                flex: 11,
                child: TextField(
                  controller: _environment[index].value,
                  autocorrect: false,
                  enableSuggestions: false,
                  decoration: _fieldDecoration(context, hintText: 'Value'),
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 48,
                height: 48,
                child: OutlinedButton(
                  onPressed: () => _removeEnvironment(index),
                  style: OutlinedButton.styleFrom(
                    padding: EdgeInsets.zero,
                    foregroundColor: palette.muted,
                    side: BorderSide(color: palette.borderStrong),
                  ),
                  child: const Icon(Icons.delete_outline_rounded, size: 18),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
        ],
        Row(
          children: [
            Expanded(
              child: SizedBox(
                height: 48,
                child: OutlinedButton.icon(
                  onPressed: _addEnvironment,
                  icon: const Icon(Icons.add_rounded, size: 18),
                  label: const Text('Add'),
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: SizedBox(
                height: 48,
                child: OutlinedButton.icon(
                  onPressed: () => setState(() => _pasteOpen = true),
                  icon: const Icon(Icons.content_paste_rounded, size: 17),
                  label: const Text('Paste'),
                ),
              ),
            ),
          ],
        ),
        if (_pasteOpen) ...[
          const SizedBox(height: 8),
          Container(
            decoration: BoxDecoration(
              color: palette.surface,
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(6),
            ),
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    const Expanded(
                      child: Text(
                        'Paste environment',
                        style: TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: 'Close environment paste',
                      onPressed: () => setState(() => _pasteOpen = false),
                      icon: const Icon(Icons.close_rounded, size: 18),
                    ),
                  ],
                ),
                TextField(
                  controller: _environmentPaste,
                  minLines: 4,
                  maxLines: 7,
                  autocorrect: false,
                  enableSuggestions: false,
                  style: const TextStyle(
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 14,
                  ),
                  decoration: _fieldDecoration(
                    context,
                    hintText: 'KEY=value\nexport OTHER=value',
                  ),
                ),
                const SizedBox(height: 8),
                SizedBox(
                  height: 48,
                  child: FilledButton(
                    onPressed: _applyEnvironmentPaste,
                    child: const Text('Apply environment'),
                  ),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }

  Widget _labelledField(
    BuildContext context, {
    required String label,
    required Widget child,
  }) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          label,
          style: TextStyle(
            color: palette.text,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }

  InputDecoration _fieldDecoration(
    BuildContext context, {
    String? hintText,
    String? counterText,
  }) {
    final palette = AnyttyPalette.of(context);
    return InputDecoration(
      hintText: hintText,
      counterText: counterText,
      filled: true,
      fillColor: palette.surface,
      isDense: true,
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(6),
        borderSide: BorderSide(color: palette.borderStrong),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(6),
        borderSide: BorderSide(color: palette.accent, width: 2),
      ),
    );
  }

  void _addEnvironment() {
    setState(() => _environment.add(_TerminalEnvironmentControllers()));
  }

  void _removeEnvironment(int index) {
    final removed = _environment.removeAt(index);
    removed.dispose();
    setState(() {});
  }

  void _applyEnvironmentPaste() {
    try {
      final parsed = parseTerminalEnvironment(_environmentPaste.text);
      for (final entry in _environment) {
        entry.dispose();
      }
      _environment
        ..clear()
        ..addAll(parsed.map(_TerminalEnvironmentControllers.fromEntry));
      setState(() {
        _environmentPaste.clear();
        _pasteOpen = false;
        _error = null;
      });
    } on FormatException catch (error) {
      setState(() => _error = error.message);
    }
  }

  void _submit() {
    try {
      final command = parseTerminalCommand(_command.text);
      final environment = parseTerminalEnvironment(
        _environment
            .where(
              (entry) =>
                  entry.key.text.trim().isNotEmpty ||
                  entry.value.text.isNotEmpty,
            )
            .map((entry) => '${entry.key.text.trim()}=${entry.value.text}')
            .join('\n'),
      );
      Navigator.pop(
        context,
        _TerminalCreateInput(
          name: _name.text.trim(),
          command: command,
          cwd: _cwd.text.trim(),
          environment: environment,
          cols: 80,
          rows: 24,
          sizeLockMode: _sizeLockMode,
        ),
      );
    } on FormatException catch (error) {
      setState(() => _error = error.message);
    }
  }
}

final class _TerminalEnvironmentControllers {
  _TerminalEnvironmentControllers({String key = '', String value = ''})
    : key = TextEditingController(text: key),
      value = TextEditingController(text: value);

  factory _TerminalEnvironmentControllers.fromEntry(String entry) {
    final separator = entry.indexOf('=');
    return _TerminalEnvironmentControllers(
      key: separator < 0 ? entry : entry.substring(0, separator),
      value: separator < 0 ? '' : entry.substring(separator + 1),
    );
  }

  final TextEditingController key;
  final TextEditingController value;

  void dispose() {
    key.dispose();
    value.dispose();
  }
}

final class _TerminalList extends ConsumerStatefulWidget {
  const _TerminalList({required this.endpointId, required this.label});

  final String endpointId;
  final String label;

  @override
  ConsumerState<_TerminalList> createState() => _TerminalListState();
}

final class _TerminalListState extends ConsumerState<_TerminalList>
    with WidgetsBindingObserver {
  static const _pinStore = TerminalPinStore();
  static const _refreshInterval = Duration(seconds: 2);

  final _searchController = TextEditingController();
  TerminalStatusFilter _status = TerminalStatusFilter.running;
  Set<String> _selectedTagIds = const {};
  List<String> _pinnedIds = const [];
  int _pinLoadEpoch = 0;
  Timer? _refreshTimer;
  bool _refreshInFlight = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _searchController.addListener(_handleSearchChanged);
    unawaited(_loadPins());
    _scheduleRefresh();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _refreshTimer?.cancel();
    _searchController
      ..removeListener(_handleSearchChanged)
      ..dispose();
    super.dispose();
  }

  void _handleSearchChanged() => setState(() {});

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _scheduleRefresh(Duration.zero);
    } else {
      _refreshTimer?.cancel();
    }
  }

  @override
  void didUpdateWidget(_TerminalList oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId) {
      _refreshTimer?.cancel();
      _status = TerminalStatusFilter.running;
      _searchController.clear();
      _selectedTagIds = const {};
      _pinnedIds = const [];
      unawaited(_loadPins());
      _scheduleRefresh(Duration.zero);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final terminals = ref.watch(terminalListProvider(widget.endpointId));
    return terminals.when(
      loading: () => _TerminalListLoading(
        endpointId: widget.endpointId,
        label: widget.label,
      ),
      error: (error, _) => _TerminalFailure(
        message: error.toString(),
        onRetry: () => ref.invalidate(terminalListProvider(widget.endpointId)),
      ),
      data: (items) {
        if (items.isEmpty) {
          return Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.terminal_rounded, size: 36, color: palette.muted),
                const SizedBox(height: 12),
                Text(
                  anyttyText(context, en: 'No terminals', zh: '暂无终端'),
                  style: const TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          );
        }
        final tags = terminalTagOptions(items);
        final filtered = sortPinnedTerminals(
          filterTerminals(
            terminals: items,
            status: _status,
            tagIds: _selectedTagIds,
            query: _searchController.text,
          ),
          _pinnedIds,
        );
        final runningCount = filterTerminals(
          terminals: items,
          status: TerminalStatusFilter.running,
        ).length;
        final exitedCount = filterTerminals(
          terminals: items,
          status: TerminalStatusFilter.exited,
        ).length;
        return Column(
          children: [
            _TerminalSearchField(controller: _searchController),
            _TerminalFilterBar(
              status: _status,
              runningCount: runningCount,
              exitedCount: exitedCount,
              totalCount: items.length,
              selectedTagCount: _selectedTagIds.length,
              tagsAvailable: tags.isNotEmpty,
              onStatusChanged: (value) => setState(() => _status = value),
              onTags: () => _showTagFilters(tags, items),
            ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: () => _refreshInventory(force: true),
                child: filtered.isEmpty
                    ? ListView(
                        physics: const AlwaysScrollableScrollPhysics(),
                        children: [
                          const SizedBox(height: 120),
                          Icon(
                            Icons.filter_alt_off_rounded,
                            color: palette.muted,
                            size: 32,
                          ),
                          const SizedBox(height: 10),
                          Center(
                            child: Text(
                              anyttyText(
                                context,
                                en: 'No matching terminals',
                                zh: '没有符合条件的终端',
                              ),
                            ),
                          ),
                        ],
                      )
                    : LayoutBuilder(
                        builder: (context, constraints) {
                          const horizontalPadding = 12.0;
                          const gap = 8.0;
                          final columns = constraints.maxWidth >= 720 ? 2 : 1;
                          final contentWidth =
                              constraints.maxWidth - horizontalPadding * 2;
                          final itemWidth =
                              (contentWidth - gap * (columns - 1)) / columns;
                          return ListView(
                            physics: const AlwaysScrollableScrollPhysics(),
                            padding: const EdgeInsets.fromLTRB(
                              horizontalPadding,
                              8,
                              horizontalPadding,
                              32,
                            ),
                            children: [
                              Wrap(
                                spacing: gap,
                                runSpacing: gap,
                                children: [
                                  for (final terminal in filtered)
                                    SizedBox(
                                      width: itemWidth,
                                      child: _buildTerminalRow(terminal),
                                    ),
                                ],
                              ),
                            ],
                          );
                        },
                      ),
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _buildTerminalRow(TerminalInfo terminal) {
    final terminalId = terminal.ref.terminalId;
    final pinIndex = _pinnedIds.indexOf(terminalId);
    return _TerminalRow(
      terminal: terminal,
      pinned: pinIndex >= 0,
      canMoveUp: pinIndex > 0,
      canMoveDown: pinIndex >= 0 && pinIndex < _pinnedIds.length - 1,
      searchQuery: _searchController.text,
      onTap: () => _openTerminal(terminalId),
      onAction: (action) => _handleRowAction(terminal, action),
    );
  }

  void _scheduleRefresh([Duration delay = _refreshInterval]) {
    _refreshTimer?.cancel();
    _refreshTimer = Timer(delay, _refreshInventory);
  }

  Future<void> _refreshInventory({bool force = false}) async {
    if (!mounted || _refreshInFlight) return;
    final appActive =
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    final routeActive = ModalRoute.of(context)?.isCurrent ?? false;
    if (!force && (!appActive || !routeActive)) {
      _scheduleRefresh();
      return;
    }
    _refreshInFlight = true;
    try {
      await ref
          .refresh(terminalListProvider(widget.endpointId).future)
          .then<void>((_) {});
    } catch (_) {
      // The current snapshot stays visible and the next scheduled poll retries.
    } finally {
      _refreshInFlight = false;
      if (mounted) _scheduleRefresh();
    }
  }

  Future<void> _loadPins() async {
    final epoch = ++_pinLoadEpoch;
    try {
      final loaded = await _pinStore.load(widget.endpointId);
      if (!mounted || epoch != _pinLoadEpoch) return;
      setState(() => _pinnedIds = loaded);
    } catch (_) {
      // Pinning is optional local presentation state.
    }
  }

  Future<void> _savePins(List<String> next) async {
    final previous = _pinnedIds;
    setState(() => _pinnedIds = next);
    try {
      final saved = await _pinStore.save(widget.endpointId, next);
      if (mounted) setState(() => _pinnedIds = saved);
    } catch (error) {
      if (!mounted) return;
      setState(() => _pinnedIds = previous);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _openTerminal(String terminalId) {
    context.push(
      '/terminal/${Uri.encodeComponent(widget.endpointId)}/'
      '${Uri.encodeComponent(terminalId)}'
      '?label=${Uri.encodeQueryComponent(widget.label)}',
    );
  }

  void _handleRowAction(TerminalInfo terminal, _TerminalRowAction action) {
    final terminalId = terminal.ref.terminalId;
    switch (action) {
      case _TerminalRowAction.pin:
      case _TerminalRowAction.unpin:
        unawaited(_savePins(toggleTerminalPin(_pinnedIds, terminalId)));
      case _TerminalRowAction.moveUp:
        unawaited(_savePins(movePinnedTerminal(_pinnedIds, terminalId, -1)));
      case _TerminalRowAction.moveDown:
        unawaited(_savePins(movePinnedTerminal(_pinnedIds, terminalId, 1)));
      case _TerminalRowAction.rename:
      case _TerminalRowAction.restart:
      case _TerminalRowAction.end:
      case _TerminalRowAction.remove:
        final remoteAction = switch (action) {
          _TerminalRowAction.rename => _TerminalAction.rename,
          _TerminalRowAction.restart => _TerminalAction.restart,
          _TerminalRowAction.end => _TerminalAction.end,
          _TerminalRowAction.remove => _TerminalAction.remove,
          _ => throw StateError('Local terminal action was not handled'),
        };
        unawaited(
          _manageTerminal(
            context: context,
            ref: ref,
            endpointId: widget.endpointId,
            terminal: terminal,
            action: remoteAction,
          ),
        );
    }
  }

  Future<void> _showTagFilters(
    List<TerminalTagOption> options,
    List<TerminalInfo> terminals,
  ) async {
    final palette = AnyttyPalette.of(context);
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: false,
      backgroundColor: palette.surface,
      barrierColor: palette.overlay,
      shape: RoundedRectangleBorder(
        borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
        side: BorderSide(color: palette.border),
      ),
      builder: (context) => _TerminalTagFilterSheet(
        options: options,
        selected: _selectedTagIds,
        totalCount: terminals.length,
        resultCount: (selected) => filterTerminals(
          terminals: terminals,
          status: _status,
          tagIds: selected,
          query: _searchController.text,
        ).length,
        onChanged: (selected) {
          if (mounted) setState(() => _selectedTagIds = selected);
        },
      ),
    );
  }
}

final class _TerminalListLoading extends ConsumerStatefulWidget {
  const _TerminalListLoading({required this.endpointId, required this.label});

  final String endpointId;
  final String label;

  @override
  ConsumerState<_TerminalListLoading> createState() =>
      _TerminalListLoadingState();
}

final class _TerminalListLoadingState
    extends ConsumerState<_TerminalListLoading> {
  final Map<ConnectionRouteKind, EndpointConnectionEvent> _attempts = {};
  bool _applyingAuto = false;
  String? _actionError;

  @override
  void didUpdateWidget(_TerminalListLoading oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId) {
      _attempts.clear();
      _applyingAuto = false;
      _actionError = null;
    }
  }

  void _captureProgress(EndpointConnectionEvent? event) {
    if (!mounted || event == null) return;
    if (event.phase ==
        EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING) {
      if (_attempts.isNotEmpty) setState(_attempts.clear);
      return;
    }
    final kind = event.attemptedRouteKind;
    if (kind == ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED) return;
    setState(() => _attempts[kind] = event.deepCopy());
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final progress = ref.watch(
      endpointConnectionProgressProvider(widget.endpointId),
    );
    ref.listen<AsyncValue<EndpointConnectionEvent>>(
      endpointConnectionProgressProvider(widget.endpointId),
      (_, next) => _captureProgress(next.valueOrNull),
    );
    final policyState = ref
        .watch(connectionPolicyProvider(widget.endpointId))
        .valueOrNull;
    final directOnly =
        policyState?.policy.routePreference ==
        EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT;
    final phase =
        progress.valueOrNull?.phase ??
        EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING;
    final waitingForNetwork =
        phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_OFFLINE;
    final connectionFailed =
        waitingForNetwork || progress.valueOrNull?.hasError() == true;
    final showDirectHelp =
        directOnly &&
        phase != EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_READY &&
        connectionFailed;
    final attempts = Map<ConnectionRouteKind, EndpointConnectionEvent>.of(
      _attempts,
    );
    final latest = progress.valueOrNull;
    if (latest != null &&
        latest.attemptedRouteKind !=
            ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED) {
      attempts[latest.attemptedRouteKind] = latest;
    }
    final attemptEvents = _orderedConnectionAttempts(attempts);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 360),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: AnimatedSwitcher(
            duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
            child: showDirectHelp
                ? _DirectOnlyConnectionHelp(
                    key: const ValueKey('direct-only-help'),
                    applying: _applyingAuto,
                    error: _actionError,
                    onUseAuto: policyState == null
                        ? null
                        : () => _useAutomaticConnection(policyState),
                    onOpenSettings: _openConnectionSettings,
                  )
                : Row(
                    key: const ValueKey('connection-progress'),
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Container(
                        width: 42,
                        height: 42,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: palette.surfaceRaised,
                          borderRadius: BorderRadius.circular(13),
                          border: Border.all(color: palette.border),
                        ),
                        child: SizedBox.square(
                          dimension: 18,
                          child: CircularProgressIndicator(
                            color: waitingForNetwork
                                ? palette.warning
                                : palette.accent,
                            strokeWidth: 2.2,
                          ),
                        ),
                      ),
                      const SizedBox(width: 12),
                      Flexible(
                        child: Semantics(
                          liveRegion: true,
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                _connectionModeLabel(
                                  context,
                                  policyState?.policy.routePreference,
                                ),
                                style: TextStyle(
                                  color: palette.muted,
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              const SizedBox(height: 4),
                              if (attemptEvents.isEmpty)
                                Text(
                                  _connectionLoadingLabel(context, phase),
                                  key: ValueKey(phase.value),
                                  maxLines: 2,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    color: waitingForNetwork
                                        ? palette.warning
                                        : palette.text,
                                    fontSize: 14,
                                    fontWeight: FontWeight.w600,
                                  ),
                                )
                              else
                                for (final event in attemptEvents)
                                  Padding(
                                    padding: const EdgeInsets.only(bottom: 3),
                                    child: Row(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        Container(
                                          width: 6,
                                          height: 6,
                                          decoration: BoxDecoration(
                                            shape: BoxShape.circle,
                                            color:
                                                event.connectionStage ==
                                                    'attempt_failed'
                                                ? palette.warning
                                                : palette.accent,
                                          ),
                                        ),
                                        const SizedBox(width: 7),
                                        Flexible(
                                          child: Text(
                                            _connectionAttemptLabel(
                                              context,
                                              event,
                                            ),
                                            key: ValueKey((
                                              event.attemptedRouteKind.value,
                                              event.connectionStage,
                                            )),
                                            maxLines: 2,
                                            overflow: TextOverflow.ellipsis,
                                            style: TextStyle(
                                              color:
                                                  event.connectionStage ==
                                                      'attempt_failed'
                                                  ? palette.warning
                                                  : palette.text,
                                              fontSize: 13,
                                              fontWeight: FontWeight.w600,
                                            ),
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                            ],
                          ),
                        ),
                      ),
                    ],
                  ),
          ),
        ),
      ),
    );
  }

  Future<void> _useAutomaticConnection(ConnectionPolicyState state) async {
    if (_applyingAuto) return;
    setState(() {
      _applyingAuto = true;
      _actionError = null;
    });
    var policySaved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final policy = state.policy.deepCopy()
        ..routePreference =
            EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO;
      await ConnectionRepository(runtime)
          .applyPolicy(widget.endpointId, policy);
      policySaved = true;
      ref.invalidate(connectionPolicyProvider(widget.endpointId));
      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpointId));
      ref.invalidate(terminalListProvider(widget.endpointId));
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _actionError = policySaved
            ? anyttyText(
                context,
                en: 'Automatic mode was saved, but reconnecting failed. Try again.',
                zh: '已改为自动连接，但重新连接失败，请重试。',
              )
            : anyttyText(
                context,
                en: 'Could not change the connection mode. Try again.',
                zh: '无法更改连接模式，请重试。',
              );
      });
    } finally {
      if (mounted) setState(() => _applyingAuto = false);
    }
  }

  void _openConnectionSettings() {
    context.push(
      '/connection/${Uri.encodeComponent(widget.endpointId)}'
      '?label=${Uri.encodeQueryComponent(widget.label)}',
    );
  }
}

String _connectionLoadingLabel(
  BuildContext context,
  EndpointConnectionPhase phase,
) {
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_RESOLVING) {
    return anyttyText(context, en: 'Resolving device address', zh: '正在解析设备地址');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_SIGNALING) {
    return anyttyText(context, en: 'Negotiating connection', zh: '正在进行 ICE 协商');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING) {
    return anyttyText(context, en: 'Connecting to device', zh: '正在连接设备');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_AUTHORIZING) {
    return anyttyText(context, en: 'Authorizing this device', zh: '正在验证设备授权');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_READY) {
    return anyttyText(context, en: 'Loading terminal list', zh: '正在加载终端列表');
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_OFFLINE) {
    return anyttyText(
      context,
      en: 'Connection interrupted, retrying',
      zh: '连接中断，正在重试',
    );
  }
  if (phase == EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING) {
    return anyttyText(context, en: 'Probing available routes', zh: '正在探测可用线路');
  }
  return anyttyText(context, en: 'Preparing connection', zh: '正在准备连接');
}

String _connectionModeLabel(
  BuildContext context,
  EndpointRoutePreference? preference,
) => switch (preference) {
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT => anyttyText(
    context,
    en: 'Currently using Direct mode',
    zh: '当前使用直连模式',
  ),
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_SSH => anyttyText(
    context,
    en: 'Currently using SSH mode',
    zh: '当前使用 SSH 模式',
  ),
  EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_MANAGED_CLOUD => anyttyText(
    context,
    en: 'Currently using Cloud mode',
    zh: '当前使用 Cloud 模式',
  ),
  _ => anyttyText(
    context,
    en: 'Automatically selecting a connection route',
    zh: '正在自动选择连接线路',
  ),
};

List<EndpointConnectionEvent> _orderedConnectionAttempts(
  Map<ConnectionRouteKind, EndpointConnectionEvent> attempts,
) {
  const order = [
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL,
  ];
  return order
      .map((kind) => attempts[kind])
      .whereType<EndpointConnectionEvent>()
      .toList(growable: false);
}

String _connectionAttemptLabel(
  BuildContext context,
  EndpointConnectionEvent event,
) {
  final route = switch (event.attemptedRouteKind) {
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT => anyttyText(
      context,
      en: 'Direct',
      zh: '直连',
    ),
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH => 'SSH',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD => 'Cloud',
    ConnectionRouteKind.CONNECTION_ROUTE_KIND_LOCAL => anyttyText(
      context,
      en: 'Local',
      zh: '本地',
    ),
    _ => anyttyText(context, en: 'Connection', zh: '连接'),
  };
  final stage = switch (event.connectionStage) {
    'attempt_starting' => anyttyText(context, en: 'starting route', zh: '开始尝试'),
    'attempt_failed' => anyttyText(context, en: 'route failed', zh: '线路尝试失败'),
    'authorization_preparing' => anyttyText(
      context,
      en: 'preparing authorization',
      zh: '准备连接授权',
    ),
    'credential_resolving' => anyttyText(
      context,
      en: 'loading credentials',
      zh: '读取连接凭据',
    ),
    'ssh_connecting' => anyttyText(
      context,
      en: 'establishing SSH connection',
      zh: '建立 SSH 连接',
    ),
    'ssh_tunnel_ready' => anyttyText(
      context,
      en: 'SSH tunnel ready',
      zh: 'SSH 隧道已建立',
    ),
    'peer_opening' => anyttyText(
      context,
      en: 'creating secure peer',
      zh: '创建安全连接',
    ),
    'ice_gathering' => anyttyText(
      context,
      en: 'gathering ICE candidates',
      zh: '收集 ICE 候选',
    ),
    'signaling' => anyttyText(
      context,
      en: 'exchanging signaling data',
      zh: '交换连接信令',
    ),
    'ice_connecting' => anyttyText(
      context,
      en: 'negotiating ICE connection',
      zh: '正在进行 ICE 协商',
    ),
    'cloud_cached_edge' => anyttyText(
      context,
      en: 'reusing known Cloud edge',
      zh: '连接已知 Cloud 边缘节点',
    ),
    'cloud_discovering' => anyttyText(
      context,
      en: 'locating device through Cloud',
      zh: '通过 Cloud 查找设备',
    ),
    'cloud_p2p_attempt' => anyttyText(
      context,
      en: 'attempting P2P hole punching',
      zh: '尝试 P2P 打洞',
    ),
    'cloud_relay_attempt' => anyttyText(
      context,
      en: 'establishing Relay channel',
      zh: '建立 Relay 转发通道',
    ),
    'cloud_direct_selected' => anyttyText(
      context,
      en: 'P2P connection established',
      zh: 'P2P 连接已建立',
    ),
    'cloud_relay_fallback' => anyttyText(
      context,
      en: 'P2P unavailable, switched to Relay',
      zh: 'P2P 未建立，已转入 Relay',
    ),
    'cloud_relay_selected' => anyttyText(
      context,
      en: 'Relay channel established',
      zh: 'Relay 转发通道已建立',
    ),
    'transport_authorizing' => anyttyText(
      context,
      en: 'verifying secure channel',
      zh: '验证安全通道',
    ),
    'protocol_opening' => anyttyText(
      context,
      en: 'opening AnyTTY protocol',
      zh: '建立 AnyTTY 会话',
    ),
    _ => _connectionLoadingLabel(context, event.phase),
  };
  return '$route · $stage';
}

final class _DirectOnlyConnectionHelp extends StatelessWidget {
  const _DirectOnlyConnectionHelp({
    super.key,
    required this.applying,
    required this.error,
    required this.onUseAuto,
    required this.onOpenSettings,
  });

  final bool applying;
  final String? error;
  final VoidCallback? onUseAuto;
  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      liveRegion: true,
      container: true,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 42,
                height: 42,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: palette.warning.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(13),
                  border: Border.all(
                    color: palette.warning.withValues(alpha: 0.32),
                  ),
                ),
                child: applying
                    ? SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(
                          color: palette.warning,
                          strokeWidth: 2.2,
                        ),
                      )
                    : Icon(
                        Icons.route_outlined,
                        size: 21,
                        color: palette.warning,
                      ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  anyttyText(
                    context,
                    en: 'Direct connection unavailable',
                    zh: '当前网络无法建立直连',
                  ),
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Text(
            anyttyText(
              context,
              en: 'Only direct connections are allowed, so other available routes are not being tried. Switch to Automatic to let AnyTTY choose a working route.',
              zh: '当前设置只允许直连，因此不会尝试其他可用线路。切换为自动连接后，AnyTTY 会选择能够连通的线路。',
            ),
            style: TextStyle(
              color: palette.muted,
              fontSize: 13,
              height: 1.5,
              fontWeight: FontWeight.w500,
            ),
          ),
          if (error != null) ...[
            const SizedBox(height: 10),
            Text(
              error!,
              style: TextStyle(
                color: palette.danger,
                fontSize: 12,
                height: 1.4,
              ),
            ),
          ],
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              FilledButton.icon(
                onPressed: applying ? null : onUseAuto,
                icon: const Icon(Icons.auto_awesome_outlined, size: 17),
                label: Text(
                  anyttyText(context, en: 'Use Automatic', zh: '改为自动连接'),
                ),
              ),
              TextButton.icon(
                onPressed: applying ? null : onOpenSettings,
                icon: const Icon(Icons.tune_rounded, size: 17),
                label: Text(
                  anyttyText(context, en: 'Connection settings', zh: '网络设置'),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

final class _TerminalRow extends StatelessWidget {
  const _TerminalRow({
    required this.terminal,
    required this.pinned,
    required this.canMoveUp,
    required this.canMoveDown,
    required this.searchQuery,
    required this.onTap,
    required this.onAction,
  });

  final TerminalInfo terminal;
  final bool pinned;
  final bool canMoveUp;
  final bool canMoveDown;
  final String searchQuery;
  final VoidCallback onTap;
  final ValueChanged<_TerminalRowAction> onAction;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final name = terminal.name.trim().isEmpty
        ? terminal.ref.terminalId
        : terminal.name.trim();
    final program = terminalProgramName(terminal);
    final details = <String>[
      if (program.isNotEmpty) program,
      if (terminal.liveCwd.trim().isNotEmpty)
        terminal.liveCwd.trim()
      else if (terminal.cwd.trim().isNotEmpty)
        terminal.cwd.trim(),
      if (program.isEmpty && terminal.command.isNotEmpty)
        terminal.command.join(' '),
    ];
    return SizedBox(
      height: 88,
      child: Card(
        elevation: 1,
        shadowColor: Colors.black.withValues(
          alpha: Theme.of(context).brightness == Brightness.dark ? 0.20 : 0.08,
        ),
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(18),
          side: BorderSide(color: palette.border),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(18),
          onTap: () {
            HapticFeedback.selectionClick();
            onTap();
          },
          child: Padding(
            padding: const EdgeInsets.fromLTRB(9, 7, 2, 7),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Stack(
                  clipBehavior: Clip.none,
                  children: [
                    Container(
                      width: 40,
                      height: 40,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: palette.surfaceRaised,
                        borderRadius: BorderRadius.circular(13),
                      ),
                      child: Icon(
                        terminalProgramIcon(terminal),
                        size: 20,
                        color: palette.text,
                      ),
                    ),
                    Positioned(
                      right: -2,
                      bottom: -2,
                      child: Container(
                        width: 10,
                        height: 10,
                        decoration: BoxDecoration(
                          color: _terminalStateColor(context, terminal.state),
                          shape: BoxShape.circle,
                          border: Border.all(color: palette.surface, width: 2),
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(width: 11),
                Expanded(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: FuzzyHighlightText(
                              name,
                              query: searchQuery,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                fontSize: 14,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                          ),
                          if (pinned)
                            Icon(
                              LucideIcons.pin,
                              size: 13,
                              color: palette.warning,
                            ),
                        ],
                      ),
                      const SizedBox(height: 2),
                      FuzzyHighlightText(
                        details.isEmpty
                            ? terminal.ref.terminalId
                            : details.join('  ·  '),
                        query: searchQuery,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: palette.muted,
                          fontFamily: 'JetBrainsMonoNerd',
                          fontSize: 10,
                        ),
                      ),
                      const SizedBox(height: 4),
                      SizedBox(
                        height: 24,
                        child: TerminalResourceInlineSummary(
                          terminal: terminal,
                          onTap: () =>
                              showTerminalResourceDetails(context, terminal),
                        ),
                      ),
                    ],
                  ),
                ),
                PopupMenuButton<_TerminalRowAction>(
                  tooltip: 'Terminal actions',
                  onSelected: onAction,
                  icon: Icon(Icons.more_vert_rounded, color: palette.muted),
                  itemBuilder: (context) => [
                    const PopupMenuItem(
                      value: _TerminalRowAction.rename,
                      child: ListTile(
                        leading: Icon(Icons.edit_rounded),
                        title: Text('Rename'),
                      ),
                    ),
                    PopupMenuItem(
                      value: pinned
                          ? _TerminalRowAction.unpin
                          : _TerminalRowAction.pin,
                      child: ListTile(
                        leading: Icon(
                          pinned
                              ? Icons.push_pin_outlined
                              : Icons.push_pin_rounded,
                        ),
                        title: Text(pinned ? 'Unpin' : 'Pin'),
                      ),
                    ),
                    if (pinned && canMoveUp)
                      const PopupMenuItem(
                        value: _TerminalRowAction.moveUp,
                        child: ListTile(
                          leading: Icon(Icons.arrow_upward_rounded),
                          title: Text('Move pinned up'),
                        ),
                      ),
                    if (pinned && canMoveDown)
                      const PopupMenuItem(
                        value: _TerminalRowAction.moveDown,
                        child: ListTile(
                          leading: Icon(Icons.arrow_downward_rounded),
                          title: Text('Move pinned down'),
                        ),
                      ),
                    if (terminal.state == TerminalState.TERMINAL_STATE_EXITED)
                      const PopupMenuItem(
                        value: _TerminalRowAction.restart,
                        child: ListTile(
                          leading: Icon(Icons.restart_alt_rounded),
                          title: Text('Restart'),
                        ),
                      ),
                    if (terminal.state ==
                            TerminalState.TERMINAL_STATE_RUNNING ||
                        terminal.state == TerminalState.TERMINAL_STATE_CREATED)
                      const PopupMenuItem(
                        value: _TerminalRowAction.end,
                        child: ListTile(
                          leading: Icon(Icons.stop_circle_outlined),
                          title: Text('End process'),
                        ),
                      ),
                    if (terminal.state == TerminalState.TERMINAL_STATE_EXITED)
                      const PopupMenuItem(
                        value: _TerminalRowAction.remove,
                        child: ListTile(
                          leading: Icon(Icons.delete_outline_rounded),
                          title: Text('Delete record'),
                        ),
                      ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _TerminalSearchField extends StatelessWidget {
  const _TerminalSearchField({required this.controller});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 6, 14, 4),
      child: SizedBox(
        height: 42,
        child: TextField(
          key: const ValueKey('terminal-list-search-field'),
          controller: controller,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: anyttyText(context, en: 'Search terminals', zh: '搜索终端'),
            prefixIcon: Icon(
              Icons.search_rounded,
              color: palette.muted,
              size: 18,
            ),
            suffixIcon: controller.text.isEmpty
                ? null
                : IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Clear search',
                      zh: '清除搜索',
                    ),
                    onPressed: controller.clear,
                    icon: const Icon(Icons.close_rounded, size: 17),
                  ),
            filled: true,
            fillColor: palette.surface,
            contentPadding: const EdgeInsets.symmetric(vertical: 9),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(13),
              borderSide: BorderSide(color: palette.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(13),
              borderSide: BorderSide(color: palette.accent, width: 1.5),
            ),
          ),
        ),
      ),
    );
  }
}

final class _TerminalFilterBar extends StatelessWidget {
  const _TerminalFilterBar({
    required this.status,
    required this.runningCount,
    required this.exitedCount,
    required this.totalCount,
    required this.selectedTagCount,
    required this.tagsAvailable,
    required this.onStatusChanged,
    required this.onTags,
  });

  final TerminalStatusFilter status;
  final int runningCount;
  final int exitedCount;
  final int totalCount;
  final int selectedTagCount;
  final bool tagsAvailable;
  final ValueChanged<TerminalStatusFilter> onStatusChanged;
  final VoidCallback onTags;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 0, 14, 4),
      child: SizedBox(
        height: 56,
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Expanded(
              child: Container(
                padding: const EdgeInsets.all(4),
                decoration: BoxDecoration(
                  color: palette.surfaceRaised,
                  borderRadius: BorderRadius.circular(14),
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    _TerminalFilterOption(
                      label: anyttyText(context, en: 'Running', zh: '运行中'),
                      count: runningCount,
                      selected: status == TerminalStatusFilter.running,
                      onPressed: () =>
                          onStatusChanged(TerminalStatusFilter.running),
                    ),
                    const SizedBox(width: 4),
                    _TerminalFilterOption(
                      label: anyttyText(context, en: 'Exited', zh: '已退出'),
                      count: exitedCount,
                      selected: status == TerminalStatusFilter.exited,
                      onPressed: () =>
                          onStatusChanged(TerminalStatusFilter.exited),
                    ),
                    const SizedBox(width: 4),
                    _TerminalFilterOption(
                      label: anyttyText(context, en: 'All', zh: '全部'),
                      count: totalCount,
                      selected: status == TerminalStatusFilter.all,
                      onPressed: () =>
                          onStatusChanged(TerminalStatusFilter.all),
                    ),
                  ],
                ),
              ),
            ),
            if (tagsAvailable) ...[
              const SizedBox(width: 8),
              Semantics(
                button: true,
                selected: selectedTagCount > 0,
                label: anyttyText(
                  context,
                  en: selectedTagCount == 0
                      ? 'Filter by tags'
                      : 'Filter by tags, $selectedTagCount selected',
                  zh: selectedTagCount == 0
                      ? '按标签筛选'
                      : '按标签筛选，已选 $selectedTagCount 个',
                ),
                child: Material(
                  color: selectedTagCount > 0
                      ? palette.accent
                      : palette.surfaceRaised,
                  borderRadius: BorderRadius.circular(14),
                  child: InkWell(
                    borderRadius: BorderRadius.circular(14),
                    onTap: onTags,
                    child: SizedBox.square(
                      dimension: 56,
                      child: Icon(
                        LucideIcons.listFilter,
                        size: 19,
                        color: selectedTagCount > 0
                            ? palette.accentText
                            : palette.muted,
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

final class _TerminalFilterOption extends StatelessWidget {
  const _TerminalFilterOption({
    required this.label,
    required this.selected,
    required this.onPressed,
    this.count,
  });

  final String label;
  final int? count;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        enabled: true,
        label: count == null ? label : '$label, $count terminals',
        onTap: onPressed,
        excludeSemantics: true,
        child: Material(
          color: selected ? palette.surface : Colors.transparent,
          borderRadius: BorderRadius.circular(10),
          child: InkWell(
            borderRadius: BorderRadius.circular(10),
            onTap: onPressed,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 3),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Flexible(
                    child: Text(
                      label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: selected ? palette.text : palette.muted,
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  if (count != null) ...[
                    const SizedBox(width: 4),
                    Text(
                      '$count',
                      style: TextStyle(
                        color: selected ? palette.muted : palette.faint,
                        fontSize: 11,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _TerminalTagFilterSheet extends StatefulWidget {
  const _TerminalTagFilterSheet({
    required this.options,
    required this.selected,
    required this.totalCount,
    required this.resultCount,
    required this.onChanged,
  });

  final List<TerminalTagOption> options;
  final Set<String> selected;
  final int totalCount;
  final int Function(Set<String> selected) resultCount;
  final ValueChanged<Set<String>> onChanged;

  @override
  State<_TerminalTagFilterSheet> createState() =>
      _TerminalTagFilterSheetState();
}

final class _TerminalTagFilterSheetState
    extends State<_TerminalTagFilterSheet> {
  late Set<String> _selected;

  @override
  void initState() {
    super.initState();
    _selected = {...widget.selected};
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final shown = widget.resultCount(_selected);
    return SafeArea(
      top: false,
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.sizeOf(context).height * 0.85,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox(
              height: 64,
              child: Stack(
                children: [
                  Positioned(
                    top: 8,
                    left: 0,
                    right: 0,
                    child: Center(
                      child: Container(
                        width: 48,
                        height: 4,
                        decoration: BoxDecoration(
                          color: palette.borderStrong,
                          borderRadius: BorderRadius.circular(2),
                        ),
                      ),
                    ),
                  ),
                  const Positioned(
                    left: 20,
                    top: 27,
                    child: Text(
                      'Terminal tags',
                      style: TextStyle(
                        fontSize: 17,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  Positioned(
                    right: 4,
                    top: 16,
                    child: IconButton(
                      tooltip: 'Close terminal tags',
                      onPressed: () => Navigator.pop(context),
                      icon: const Icon(Icons.close_rounded, size: 20),
                    ),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      'Showing $shown of ${widget.totalCount} terminals',
                      style: TextStyle(
                        color: palette.muted,
                        fontSize: 12,
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                    ),
                  ),
                  if (_selected.isNotEmpty)
                    TextButton(
                      onPressed: _clear,
                      style: TextButton.styleFrom(
                        minimumSize: const Size(0, 48),
                        padding: const EdgeInsets.symmetric(horizontal: 8),
                      ),
                      child: const Text(
                        'Reset tags',
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                ],
              ),
            ),
            Flexible(
              child: ListView.separated(
                shrinkWrap: true,
                padding: const EdgeInsets.symmetric(horizontal: 16),
                itemCount: widget.options.length,
                separatorBuilder: (_, _) =>
                    Divider(height: 1, color: palette.border),
                itemBuilder: (context, index) {
                  final option = widget.options[index];
                  final checked = _selected.contains(option.id);
                  return Semantics(
                    button: true,
                    checked: checked,
                    child: InkWell(
                      onTap: () => _toggle(option.id),
                      child: ConstrainedBox(
                        constraints: const BoxConstraints(minHeight: 48),
                        child: Row(
                          children: [
                            Icon(
                              Icons.tag_rounded,
                              size: 16,
                              color: palette.muted,
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Text(
                                option.label,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w500,
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Text(
                              option.count.toString(),
                              style: TextStyle(
                                color: palette.muted,
                                fontSize: 12,
                                fontFeatures: const [
                                  FontFeature.tabularFigures(),
                                ],
                              ),
                            ),
                            const SizedBox(width: 12),
                            Container(
                              width: 20,
                              height: 20,
                              decoration: BoxDecoration(
                                color: checked
                                    ? palette.accent
                                    : Colors.transparent,
                                border: Border.all(
                                  color: checked
                                      ? palette.accent
                                      : palette.borderStrong,
                                ),
                                borderRadius: BorderRadius.circular(4),
                              ),
                              child: checked
                                  ? Icon(
                                      Icons.check_rounded,
                                      size: 14,
                                      color: palette.accentText,
                                    )
                                  : null,
                            ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
              child: SizedBox(
                height: 48,
                width: double.infinity,
                child: FilledButton.icon(
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.filter_list_rounded, size: 17),
                  label: Text(
                    'Show $shown ${shown == 1 ? 'terminal' : 'terminals'}',
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _toggle(String id) {
    setState(() {
      if (!_selected.add(id)) _selected.remove(id);
    });
    widget.onChanged(Set.unmodifiable(_selected));
  }

  void _clear() {
    setState(_selected.clear);
    widget.onChanged(const {});
  }
}

String _terminalStateLabel(TerminalState state) {
  return switch (state) {
    TerminalState.TERMINAL_STATE_CREATED => 'Starting',
    TerminalState.TERMINAL_STATE_RUNNING => 'Running',
    TerminalState.TERMINAL_STATE_EXITED => 'Exited',
    TerminalState.TERMINAL_STATE_REMOVED => 'Removed',
    _ => 'Unknown',
  };
}

String _terminalDisplayName(TerminalInfo terminal) {
  if (terminal.name.trim().isNotEmpty) return terminal.name.trim();
  if (terminal.foregroundProcess.trim().isNotEmpty) {
    return terminal.foregroundProcess.trim();
  }
  if (terminal.command.isNotEmpty && terminal.command.first.trim().isNotEmpty) {
    return terminal.command.first.trim();
  }
  return terminal.ref.terminalId;
}

Color _terminalStateColor(BuildContext context, TerminalState state) {
  final palette = AnyttyPalette.of(context);
  return switch (state) {
    TerminalState.TERMINAL_STATE_CREATED => palette.warning,
    TerminalState.TERMINAL_STATE_RUNNING => palette.success,
    TerminalState.TERMINAL_STATE_EXITED => palette.muted,
    TerminalState.TERMINAL_STATE_REMOVED => palette.danger,
    _ => palette.muted,
  };
}

enum _TerminalSplitTarget { left, right, above, below }

final class _TerminalInputOperation {
  const _TerminalInputOperation.text(this.text, this.modifiers)
    : hidUsage = null,
      unshiftedCodepoint = 0,
      paste = false,
      allowUnsafePaste = false;

  const _TerminalInputOperation.key({
    required this.hidUsage,
    required this.modifiers,
    this.unshiftedCodepoint = 0,
    this.text = '',
  }) : paste = false,
       allowUnsafePaste = false;

  const _TerminalInputOperation.paste(this.text, this.allowUnsafePaste)
    : hidUsage = null,
      modifiers = 0,
      unshiftedCodepoint = 0,
      paste = true;

  final int? hidUsage;
  final int modifiers;
  final int unshiftedCodepoint;
  final String text;
  final bool paste;
  final bool allowUnsafePaste;

  Future<void> send(TerminalConnection connection) {
    if (paste) {
      return connection.sendPaste(text, allowUnsafe: allowUnsafePaste);
    }
    final usage = hidUsage;
    if (usage != null) {
      return connection.sendKey(
        hidUsage: usage,
        modifiers: modifiers,
        unshiftedCodepoint: unshiftedCodepoint,
        text: text,
      );
    }
    return connection.sendText(text, modifiers: modifiers);
  }
}

final class _ActiveTerminal extends ConsumerStatefulWidget {
  const _ActiveTerminal({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminalId,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.surfaceKey,
    required this.onActiveTerminalChanged,
  }) : super(key: surfaceKey);

  final String endpointId;
  final String endpointLabel;
  final String terminalId;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final GlobalKey<_ActiveTerminalState> surfaceKey;
  final ValueChanged<String> onActiveTerminalChanged;

  @override
  ConsumerState<_ActiveTerminal> createState() => _ActiveTerminalState();
}

final class _ActiveTerminalState extends ConsumerState<_ActiveTerminal> {
  static const _pinStore = TerminalPinStore();

  TerminalSplitNode _splitRoot = primaryTerminalPane;
  String _activePaneKey = primaryTerminalPaneKey;
  late String _primaryTerminalId;
  final Map<String, GlobalKey<_TerminalSurfaceState>> _surfaceKeys = {};
  var _splitSequence = 0;
  bool _syncInput = false;
  List<String> _pinnedIds = const [];

  @override
  void initState() {
    super.initState();
    _primaryTerminalId = widget.terminalId;
    unawaited(_loadPins());
  }

  @override
  void didUpdateWidget(_ActiveTerminal oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.endpointId != widget.endpointId ||
        oldWidget.terminalId != widget.terminalId) {
      _primaryTerminalId = widget.terminalId;
      _splitRoot = primaryTerminalPane;
      _activePaneKey = primaryTerminalPaneKey;
      _surfaceKeys.clear();
      _syncInput = false;
      unawaited(_loadPins());
    }
  }

  Future<void> _loadPins() async {
    try {
      final pinned = await _pinStore.load(widget.endpointId);
      if (mounted) setState(() => _pinnedIds = pinned);
    } catch (_) {
      // Pin order only affects which terminal is chosen for the next split.
    }
  }

  GlobalKey<_TerminalSurfaceState> _surfaceKey(String paneKey) {
    return _surfaceKeys.putIfAbsent(
      paneKey,
      () => GlobalKey<_TerminalSurfaceState>(),
    );
  }

  _TerminalSurfaceState? get _activeSurface =>
      _surfaceKeys[_activePaneKey]?.currentState;

  bool get _hasSplit => terminalPaneKeys(_splitRoot).length > 1;

  void toggleTools() => _activeSurface?.toggleTools();

  void splitActiveBelow() => _splitActive(_TerminalSplitTarget.below);

  bool handleBack() {
    if (_activeSurface?.handleBack() == true) return true;
    if (!_hasSplit) return false;
    _closeActiveSplit();
    return true;
  }

  void switchTerminal(String terminalId) {
    final paneKeys = terminalPaneKeys(_splitRoot);
    if (terminalId == _primaryTerminalId) {
      _activatePane(primaryTerminalPaneKey);
      return;
    }
    final existingPaneKey = terminalPaneKey(terminalId);
    if (paneKeys.contains(existingPaneKey)) {
      _activatePane(existingPaneKey);
      return;
    }
    if (isEmptyTerminalPaneKey(_activePaneKey)) {
      final emptyPaneKey = _activePaneKey;
      setState(() {
        _splitRoot = assignTerminalToPane(
          root: _splitRoot,
          targetPaneKey: emptyPaneKey,
          terminalId: terminalId,
        );
        _activePaneKey = existingPaneKey;
      });
      widget.onActiveTerminalChanged(terminalId);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) {
          _surfaceKeys[existingPaneKey]?.currentState?._showKeyboard();
        }
      });
      return;
    }
    setState(() {
      _primaryTerminalId = terminalId;
      _activePaneKey = primaryTerminalPaneKey;
    });
    widget.onActiveTerminalChanged(terminalId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _surfaceKeys[primaryTerminalPaneKey]?.currentState?._showKeyboard();
      }
    });
  }

  void _activatePane(String paneKey) {
    if (_activePaneKey == paneKey) return;
    HapticFeedback.selectionClick();
    setState(() => _activePaneKey = paneKey);
    final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
    if (terminalId != null) {
      widget.onActiveTerminalChanged(terminalId);
    } else {
      FocusManager.instance.primaryFocus?.unfocus();
    }
  }

  void _splitActive(_TerminalSplitTarget target) {
    final terminals = ref
        .read(terminalListProvider(widget.endpointId))
        .valueOrNull;
    if (terminals == null) return;
    final displayed = terminalPaneKeys(_splitRoot)
        .map((paneKey) => terminalIdForPane(paneKey, _primaryTerminalId))
        .whereType<String>()
        .toSet();
    if (!terminals.any(
      (terminal) => !displayed.contains(terminal.ref.terminalId),
    )) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No other terminal is available')),
      );
      return;
    }
    final horizontal =
        target == _TerminalSplitTarget.left ||
        target == _TerminalSplitTarget.right;
    final before =
        target == _TerminalSplitTarget.left ||
        target == _TerminalSplitTarget.above;
    _splitSequence += 1;
    final splitId = 'split-$_splitSequence';
    final paneKey = emptyTerminalPaneKey(splitId);
    setState(() {
      _splitRoot = splitTerminalPaneEmpty(
        root: _splitRoot,
        targetPaneKey: _activePaneKey,
        direction: horizontal
            ? TerminalSplitDirection.columns
            : TerminalSplitDirection.rows,
        splitId: splitId,
        placement: before
            ? TerminalSplitPlacement.before
            : TerminalSplitPlacement.after,
      );
      _activePaneKey = paneKey;
    });
    FocusManager.instance.primaryFocus?.unfocus();
  }

  Future<void> _chooseTerminalForPane(String paneKey) async {
    if (!isEmptyTerminalPaneKey(paneKey) ||
        !terminalPaneKeys(_splitRoot).contains(paneKey)) {
      return;
    }
    final terminals = ref
        .read(terminalListProvider(widget.endpointId))
        .valueOrNull;
    if (terminals == null) return;
    final displayed = terminalPaneKeys(_splitRoot)
        .map((key) => terminalIdForPane(key, _primaryTerminalId))
        .whereType<String>()
        .toSet();
    final available = _orderedSplitTerminals(terminals)
        .where((terminal) => !displayed.contains(terminal.ref.terminalId))
        .toList(growable: false);
    if (available.isEmpty) return;
    final selection = await showAnyttyTerminalSwitcher(
      context: context,
      endpoints: [
        TerminalSwitcherEndpoint(
          endpointId: widget.endpointId,
          label: widget.endpointLabel,
          current: true,
          terminals: available,
        ),
      ],
      loadTerminals: (_) async => available,
    );
    if (!mounted || selection == null) return;
    final terminalId = selection.terminalId;
    final newPaneKey = terminalPaneKey(terminalId);
    if (terminalPaneKeys(_splitRoot).contains(newPaneKey)) {
      _activatePane(newPaneKey);
      return;
    }
    setState(() {
      _splitRoot = assignTerminalToPane(
        root: _splitRoot,
        targetPaneKey: paneKey,
        terminalId: terminalId,
      );
      _activePaneKey = newPaneKey;
    });
    widget.onActiveTerminalChanged(terminalId);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        _surfaceKeys[newPaneKey]?.currentState?._showKeyboard();
      }
    });
  }

  List<TerminalInfo> _orderedSplitTerminals(List<TerminalInfo> terminals) {
    final running = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.running,
      ),
      _pinnedIds,
    );
    final exited = sortPinnedTerminals(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.exited,
      ),
      _pinnedIds,
    );
    final known = {...running, ...exited};
    return [
      ...running,
      ...exited,
      ...terminals.where((terminal) => !known.contains(terminal)),
    ];
  }

  void _closeActiveSplit() {
    if (!_hasSplit) return;
    var paneKey = _activePaneKey;
    if (paneKey == primaryTerminalPaneKey) {
      paneKey = terminalPaneKeys(_splitRoot)
          .lastWhere((candidate) => candidate != primaryTerminalPaneKey);
    }
    setState(() {
      _splitRoot =
          removeTerminalPane(_splitRoot, paneKey) ?? primaryTerminalPane;
      _surfaceKeys.remove(paneKey);
      _activePaneKey = primaryTerminalPaneKey;
      _syncInput = false;
    });
    widget.onActiveTerminalChanged(_primaryTerminalId);
  }

  Future<void> _sendInput(_TerminalInputOperation operation) async {
    final paneKeys = _syncInput && _hasSplit
        ? terminalPaneKeys(_splitRoot)
        : [_activePaneKey];
    final targets = paneKeys
        .map((paneKey) => _surfaceKeys[paneKey]?.currentState)
        .whereType<_TerminalSurfaceState>()
        .toList(growable: false);
    if (targets.isEmpty) {
      throw const NativeSessionException('The active terminal is not ready');
    }
    await dispatchTerminalInput(
      targets,
      (target) => operation.send(target.widget.connection),
    );
  }

  void _removeUnavailablePanes(List<TerminalInfo> terminals) {
    final available = terminals
        .map((terminal) => terminal.ref.terminalId)
        .toSet();
    final invalid = terminalPaneKeys(_splitRoot)
        .where((paneKey) {
          if (paneKey == primaryTerminalPaneKey) return false;
          if (isEmptyTerminalPaneKey(paneKey)) return false;
          final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
          return terminalId == _primaryTerminalId ||
              !available.contains(terminalId);
        })
        .toList(growable: false);
    if (invalid.isEmpty) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      setState(() {
        for (final paneKey in invalid) {
          _splitRoot =
              removeTerminalPane(_splitRoot, paneKey) ?? primaryTerminalPane;
          _surfaceKeys.remove(paneKey);
          if (_activePaneKey == paneKey) {
            _activePaneKey = primaryTerminalPaneKey;
          }
        }
        _syncInput = false;
      });
      final activeTerminalId = terminalIdForPane(
        _activePaneKey,
        _primaryTerminalId,
      );
      if (activeTerminalId != null) {
        widget.onActiveTerminalChanged(activeTerminalId);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final terminalsAsync = ref.watch(terminalListProvider(widget.endpointId));
    final terminals = terminalsAsync.valueOrNull ?? const <TerminalInfo>[];
    _removeUnavailablePanes(terminals);
    final byId = {
      for (final terminal in terminals) terminal.ref.terminalId: terminal,
    };
    final keyboardFocusLayout =
        _hasSplit &&
        widget.keyboardInset > 100 &&
        MediaQuery.orientationOf(context) == Orientation.landscape;
    return TerminalPetalMenuOverlay(
      child: Stack(
        fit: StackFit.expand,
        children: [
          TerminalKeyboardWorkspace(
            visualInset: widget.keyboardVisualInset,
            child: Column(
              children: [
                Expanded(
                  child: Stack(
                    fit: StackFit.expand,
                    children: [
                      if (keyboardFocusLayout)
                        _buildKeyboardFocusedPanes(byId, widget.keyboardInset)
                      else
                        _buildSplitNode(_splitRoot, byId, widget.keyboardInset),
                      if (_activeSurface case final surface?)
                        ValueListenableBuilder<int>(
                          valueListenable: surface.controlRevision,
                          builder: (context, _, _) => Stack(
                            fit: StackFit.expand,
                            children: surface.buildExternalOverlays(),
                          ),
                        ),
                    ],
                  ),
                ),
                if (_activeSurface case final surface?)
                  ValueListenableBuilder<int>(
                    valueListenable: surface.controlRevision,
                    builder: (context, _, _) => surface.buildExternalKeyBar(),
                  ),
              ],
            ),
          ),
          if (_activeSurface case final surface?)
            ValueListenableBuilder<int>(
              valueListenable: surface.controlRevision,
              builder: (context, _, _) => Stack(
                fit: StackFit.expand,
                children: surface.buildHeaderOverlays(),
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildSplitNode(
    TerminalSplitNode node,
    Map<String, TerminalInfo> terminals,
    double keyboardInset,
  ) {
    return switch (node) {
      TerminalPaneNode() => _buildTerminalPane(
        node.paneKey,
        terminals,
        keyboardInset: keyboardInset,
      ),
      TerminalSplitBranch() => LayoutBuilder(
        builder: (context, constraints) {
          final columns = node.direction == TerminalSplitDirection.columns;
          final extent = columns ? constraints.maxWidth : constraints.maxHeight;
          final divider = extent * node.ratio / 100;
          final first = columns
              ? Positioned(
                  left: 0,
                  top: 0,
                  bottom: 0,
                  width: math.max(0, divider - 0.5),
                  child: _buildSplitNode(node.first, terminals, keyboardInset),
                )
              : Positioned(
                  left: 0,
                  top: 0,
                  right: 0,
                  height: math.max(0, divider - 0.5),
                  child: _buildSplitNode(node.first, terminals, keyboardInset),
                );
          final second = columns
              ? Positioned(
                  left: divider + 0.5,
                  top: 0,
                  right: 0,
                  bottom: 0,
                  child: _buildSplitNode(node.second, terminals, keyboardInset),
                )
              : Positioned(
                  left: 0,
                  top: divider + 0.5,
                  right: 0,
                  bottom: 0,
                  child: _buildSplitNode(node.second, terminals, keyboardInset),
                );
          return Stack(
            fit: StackFit.expand,
            clipBehavior: Clip.hardEdge,
            children: [
              first,
              second,
              Positioned(
                left: columns ? divider - 10 : 0,
                top: columns ? 0 : divider - 10,
                right: columns ? null : 0,
                bottom: columns ? 0 : null,
                width: columns ? 20 : null,
                height: columns ? null : 20,
                child: _TerminalSplitDivider(
                  direction: node.direction,
                  onDrag: (delta) {
                    if (extent <= 0) return;
                    setState(() {
                      _splitRoot = updateTerminalSplitRatio(
                        _splitRoot,
                        node.id,
                        node.ratio + delta / extent * 100,
                      );
                    });
                  },
                ),
              ),
            ],
          );
        },
      ),
    };
  }

  Widget _buildKeyboardFocusedPanes(
    Map<String, TerminalInfo> terminals,
    double keyboardInset,
  ) {
    return Stack(
      fit: StackFit.expand,
      children: [
        for (final paneKey in terminalPaneKeys(_splitRoot))
          Positioned.fill(
            child: TickerMode(
              enabled: paneKey == _activePaneKey,
              child: Offstage(
                offstage: paneKey != _activePaneKey,
                child: _buildTerminalPane(
                  paneKey,
                  terminals,
                  keyboardInset: keyboardInset,
                  showPaneHeader: false,
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _buildTerminalPane(
    String paneKey,
    Map<String, TerminalInfo> terminals, {
    required double keyboardInset,
    bool showPaneHeader = true,
  }) {
    final terminalId = terminalIdForPane(paneKey, _primaryTerminalId);
    final terminal = terminalId == null ? null : terminals[terminalId];
    final active = paneKey == _activePaneKey;
    return GestureDetector(
      behavior: HitTestBehavior.translucent,
      excludeFromSemantics: true,
      onTapDown: (_) => _activatePane(paneKey),
      child: Stack(
        fit: StackFit.expand,
        children: [
          ColoredBox(
            color: AnyttyPalette.of(context).background,
            child: Column(
              children: [
                if (_hasSplit && showPaneHeader)
                  _TerminalPaneHeader(terminal: terminal, active: active),
                Expanded(
                  child: isEmptyTerminalPaneKey(paneKey)
                      ? _EmptyTerminalPane(
                          active: active,
                          onChoose: () => _chooseTerminalForPane(paneKey),
                        )
                      : terminal == null || terminalId == null
                      ? const _WaitingForSnapshot(label: 'Opening terminal')
                      : _TerminalPaneConnection(
                          endpointId: widget.endpointId,
                          endpointLabel: widget.endpointLabel,
                          terminal: terminal,
                          keyboardVisualInset: widget.keyboardVisualInset,
                          keyboardInset: keyboardInset,
                          settledKeyboardInset: widget.settledKeyboardInset,
                          surfaceKey: _surfaceKey(paneKey),
                          active: active,
                          onActivate: () => _activatePane(paneKey),
                          externalControls: true,
                          onSurfaceReady: () {
                            if (mounted) setState(() {});
                          },
                          onInput: _sendInput,
                          splitOpen: _hasSplit,
                          syncInput: _syncInput,
                          canSplit:
                              terminalPaneKeys(_splitRoot).length <
                              terminals.length,
                          onSplit: _splitActive,
                          onToggleSync: () =>
                              setState(() => _syncInput = !_syncInput),
                          onCloseSplit: _closeActiveSplit,
                        ),
                ),
              ],
            ),
          ),
          if (active && _hasSplit)
            IgnorePointer(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.fromBorderSide(
                    BorderSide(color: AnyttyPalette.of(context).accent),
                  ),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

final class _TerminalPaneConnection extends ConsumerWidget {
  const _TerminalPaneConnection({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminal,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.surfaceKey,
    required this.active,
    required this.onActivate,
    required this.externalControls,
    required this.onSurfaceReady,
    required this.onInput,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
  });

  final String endpointId;
  final String endpointLabel;
  final TerminalInfo terminal;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final GlobalKey<_TerminalSurfaceState> surfaceKey;
  final bool active;
  final VoidCallback onActivate;
  final bool externalControls;
  final VoidCallback onSurfaceReady;
  final Future<void> Function(_TerminalInputOperation) onInput;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final terminalId = terminal.ref.terminalId;
    final key = (endpointId: endpointId, terminalId: terminalId);
    final settings =
        ref.watch(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    final keyboardMode =
        ref.watch(terminalKeyboardModeProvider(key)).valueOrNull ??
        settings.keyboardMode;
    return ref
        .watch(terminalConnectionProvider(key))
        .when(
          loading: () => const _WaitingForSnapshot(label: 'Opening terminal'),
          error: (error, _) => _TerminalFailure(
            dark: true,
            message: error.toString(),
            onRetry: () => ref.invalidate(terminalConnectionProvider(key)),
          ),
          data: (connection) => _TerminalSurface(
            key: surfaceKey,
            connection: connection,
            keyboardMode: keyboardMode,
            keyboardVisualInset: keyboardVisualInset,
            keyboardInset: keyboardInset,
            settledKeyboardInset: settledKeyboardInset,
            settings: settings,
            active: active,
            onActivate: onActivate,
            externalControls: externalControls,
            manageKeyboardInset: false,
            onSurfaceReady: onSurfaceReady,
            onInput: onInput,
            onOpenFiles: () => showAnyttyFileManager(
              context: context,
              endpointId: endpointId,
              endpointLabel: endpointLabel,
              initialPath: terminal.liveCwd.isNotEmpty
                  ? terminal.liveCwd
                  : terminal.cwd,
            ),
            onShowResources: () =>
                showTerminalResourceDetails(context, terminal),
            splitOpen: splitOpen,
            syncInput: syncInput,
            canSplit: canSplit,
            onSplit: onSplit,
            onToggleSync: onToggleSync,
            onCloseSplit: onCloseSplit,
            onSettingsChanged: (next) =>
                ref.read(terminalSettingsProvider.notifier).save(next),
            onKeyboardModeChanged: (next) async {
              await const TerminalKeyboardModeStore().save(
                endpointId: endpointId,
                terminalId: terminalId,
                mode: next,
              );
              ref.invalidate(terminalKeyboardModeProvider(key));
            },
            onReconnect: () => ref.invalidate(terminalConnectionProvider(key)),
          ),
        );
  }
}

final class _EmptyTerminalPane extends StatelessWidget {
  const _EmptyTerminalPane({required this.active, required this.onChoose});

  final bool active;
  final VoidCallback onChoose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 230),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  LucideIcons.panelTopOpen,
                  size: 28,
                  color: active ? palette.accent : palette.muted,
                ),
                const SizedBox(height: 12),
                Text(
                  anyttyText(
                    context,
                    en: 'Choose a terminal for this pane',
                    zh: '为这个窗格选择终端',
                  ),
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 14),
                FilledButton.icon(
                  key: const ValueKey('choose-split-terminal'),
                  onPressed: onChoose,
                  icon: const Icon(LucideIcons.terminal, size: 16),
                  label: Text(
                    anyttyText(context, en: 'Choose terminal', zh: '选择终端'),
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

final class _TerminalPaneHeader extends StatelessWidget {
  const _TerminalPaneHeader({required this.terminal, required this.active});

  final TerminalInfo? terminal;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final title = terminal == null
        ? 'Terminal'
        : _terminalDisplayName(terminal!);
    final cwd = terminal == null
        ? ''
        : terminal!.liveCwd.isNotEmpty
        ? terminal!.liveCwd
        : terminal!.cwd;
    return Container(
      height: 28,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          Icon(
            Icons.terminal_rounded,
            size: 14,
            color: active ? palette.text : palette.faint,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: active ? palette.text : palette.muted,
                fontSize: 11,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
          if (cwd.isNotEmpty) ...[
            const SizedBox(width: 8),
            Flexible(
              child: Text(
                cwd,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: palette.faint,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 10,
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

final class _TerminalSplitDivider extends StatelessWidget {
  const _TerminalSplitDivider({required this.direction, required this.onDrag});

  final TerminalSplitDirection direction;
  final ValueChanged<double> onDrag;

  @override
  Widget build(BuildContext context) {
    final columns = direction == TerminalSplitDirection.columns;
    return MouseRegion(
      cursor: columns
          ? SystemMouseCursors.resizeColumn
          : SystemMouseCursors.resizeRow,
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onHorizontalDragUpdate: columns
            ? (details) => onDrag(details.delta.dx)
            : null,
        onVerticalDragUpdate: columns
            ? null
            : (details) => onDrag(details.delta.dy),
        child: Center(
          child: ColoredBox(
            color: AnyttyPalette.of(context).borderStrong,
            child: SizedBox(
              width: columns ? 1 : double.infinity,
              height: columns ? double.infinity : 1,
            ),
          ),
        ),
      ),
    );
  }
}

final class _TerminalSurface extends StatefulWidget {
  const _TerminalSurface({
    super.key,
    required this.connection,
    required this.keyboardMode,
    required this.keyboardVisualInset,
    required this.keyboardInset,
    required this.settledKeyboardInset,
    required this.settings,
    required this.active,
    required this.onActivate,
    required this.externalControls,
    required this.manageKeyboardInset,
    required this.onSurfaceReady,
    required this.onInput,
    required this.onOpenFiles,
    required this.onShowResources,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
    required this.onSettingsChanged,
    required this.onKeyboardModeChanged,
    required this.onReconnect,
  });

  final TerminalConnection connection;
  final TerminalKeyboardMode keyboardMode;
  final ValueListenable<double> keyboardVisualInset;
  final double keyboardInset;
  final double settledKeyboardInset;
  final TerminalSettings settings;
  final bool active;
  final VoidCallback onActivate;
  final bool externalControls;
  final bool manageKeyboardInset;
  final VoidCallback onSurfaceReady;
  final Future<void> Function(_TerminalInputOperation) onInput;
  final Future<void> Function() onOpenFiles;
  final Future<void> Function() onShowResources;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;
  final Future<void> Function(TerminalSettings) onSettingsChanged;
  final Future<void> Function(TerminalKeyboardMode) onKeyboardModeChanged;
  final VoidCallback onReconnect;

  @override
  State<_TerminalSurface> createState() => _TerminalSurfaceState();
}

final class _TerminalSurfaceState extends State<_TerminalSurface> {
  final ValueNotifier<int> controlRevision = ValueNotifier(0);
  final FocusNode _inputFocus = FocusNode(debugLabel: 'terminal-input');
  final FocusNode _searchFocus = FocusNode(debugLabel: 'terminal-search');
  final TextEditingController _inputController = TextEditingController();
  final TextEditingController _searchController = TextEditingController();
  final ScrollController _historyScroll = ScrollController();
  final TerminalHistoryHighlights _historyHighlights =
      TerminalHistoryHighlights();
  TerminalModifierState _modifiers = const TerminalModifierState();
  bool _clearingInput = false;
  int _keyboardRequestEpoch = 0;
  bool _fnOpen = false;
  bool _keyboardFocusLocked = false;
  bool _toolsOpen = false;
  late TerminalKeyboardMode _keyboardMode;
  bool _historyLoading = false;
  bool _historyLoadingVisible = false;
  bool _historyPositioned = false;
  bool _historyPresented = false;
  bool _selectionMode = false;
  bool _searchOpen = false;
  bool _searching = false;
  bool _searchScanActive = false;
  bool _copying = false;
  int _searchEpoch = 0;
  int _searchMatchCount = 0;
  FrozenHistory? _history;
  FrozenHistory? _historyLayoutSource;
  HistoryLayout? _historyLayoutCache;
  HistorySelection? _selection;
  HistoryRange? _searchMatch;
  List<HistoryRange> _searchMatches = const [];
  HistorySearchMode _searchMode = HistorySearchMode.HISTORY_SEARCH_MODE_TEXT;
  bool _searchWrapped = false;
  String? _historyError;
  bool _historyRequiresReload = false;
  late ResizeControl _resizeControl;
  StreamSubscription<ResizeControl>? _resizeSubscription;
  late TerminalState _terminalState;
  StreamSubscription<TerminalState>? _terminalStateSubscription;
  late TerminalDeliveryState _deliveryState;
  StreamSubscription<TerminalDeliveryState>? _deliveryStateSubscription;
  late final TerminalRecoveryNoticeGate _recoveryNoticeGate;
  bool _recoveryNoticeVisible = false;
  ({int cols, int rows})? _viewportCells;
  ({int cols, int rows})? _historyViewportCells;
  int? _historyProjectionFailedCols;
  double? _fullTerminalViewportHeight;
  bool _resizePending = false;
  double _liveDragDistance = 0;
  Offset _lastLiveDragPosition = Offset.zero;
  Timer? _momentumTimer;
  Timer? _historyLoadingDelay;
  int _historyRequestEpoch = 0;
  DateTime? _historyScrollSampleTime;
  double? _historyScrollSampleOffset;
  double _historyUpwardVelocityRows = 0;
  double _momentumVelocity = 0;
  int _momentumTicks = 0;
  bool _autoAcquireAttempted = false;
  late final String _androidInputOwner;
  StreamSubscription<AndroidTerminalInputEvent>? _androidInputSubscription;
  final ValueNotifier<({bool active, String text})> _androidComposition =
      ValueNotifier((active: false, text: ''));

  bool get _readOnly =>
      widget.connection.historyOnly || terminalUsesHistoryOnly(_terminalState);

  void _notifyControls() => controlRevision.value += 1;

  HistoryLayout _historyLayoutFor(FrozenHistory history) {
    if (identical(_historyLayoutSource, history) &&
        _historyLayoutCache != null) {
      return _historyLayoutCache!;
    }
    final layout = HistoryLayout.fromHistory(history);
    _historyLayoutSource = history;
    _historyLayoutCache = layout;
    return layout;
  }

  @override
  void initState() {
    super.initState();
    _keyboardMode = widget.keyboardMode;
    _androidInputOwner = 'terminal-input-${identityHashCode(this)}';
    if (defaultTargetPlatform == TargetPlatform.android) {
      _androidInputSubscription = AndroidTerminalInputPlatform.instance
          .eventsFor(_androidInputOwner)
          .listen(_handleAndroidTerminalInput);
    }
    _resetSoftKeyboardBuffer();
    _inputController.addListener(_handleTextInput);
    _inputFocus.addListener(_handleInputFocusChange);
    _historyScroll.addListener(_handleHistoryScroll);
    _recoveryNoticeGate = TerminalRecoveryNoticeGate(
      onVisibilityChanged: (visible) {
        if (!mounted || _recoveryNoticeVisible == visible) return;
        setState(() => _recoveryNoticeVisible = visible);
      },
    );
    widget.connection.updateInputMetrics(widget.settings.metrics);
    _bindResizeControl();
    _bindTerminalState();
    _bindDeliveryState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onSurfaceReady();
    });
  }

  @override
  void didUpdateWidget(_TerminalSurface oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.connection != widget.connection) {
      _clearAndroidComposition();
      _historyRequestEpoch += 1;
      _historyLoadingDelay?.cancel();
      if (_history case final history?) {
        _releaseHistory(oldWidget.connection, history);
      }
      _modifiers = const TerminalModifierState();
      _fnOpen = false;
      _toolsOpen = false;
      _history = null;
      _historyLoading = false;
      _historyLoadingVisible = false;
      _historyPositioned = false;
      _historyPresented = false;
      _resetHistoryInteraction();
      unawaited(_resizeSubscription?.cancel());
      unawaited(_terminalStateSubscription?.cancel());
      unawaited(_deliveryStateSubscription?.cancel());
      _recoveryNoticeGate.setRecovering(false);
      _viewportCells = null;
      _historyViewportCells = null;
      _historyProjectionFailedCols = null;
      _fullTerminalViewportHeight = null;
      _autoAcquireAttempted = false;
      widget.connection.updateInputMetrics(widget.settings.metrics);
      _bindResizeControl();
      _bindTerminalState();
      _bindDeliveryState();
    } else if (oldWidget.settings.metrics != widget.settings.metrics) {
      final firstHistoryRow = _historyScroll.hasClients
          ? _historyScroll.offset / oldWidget.settings.metrics.rowHeight
          : null;
      _viewportCells = null;
      _historyViewportCells = null;
      widget.connection.updateInputMetrics(widget.settings.metrics);
      if (firstHistoryRow != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!mounted || !_historyScroll.hasClients) return;
          _historyScroll.jumpTo(
            (firstHistoryRow * widget.settings.metrics.rowHeight).clamp(
              0.0,
              _historyScroll.position.maxScrollExtent,
            ),
          );
        });
      }
    }
    if (oldWidget.keyboardMode != widget.keyboardMode) {
      _keyboardMode = widget.keyboardMode;
      _viewportCells = null;
    }
    if (!oldWidget.settings.autoAcquireResizeOwner &&
        widget.settings.autoAcquireResizeOwner) {
      _autoAcquireAttempted = false;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(_autoAcquireResizeOwnership());
      });
    }
    if (oldWidget.settings.scrollInertia != widget.settings.scrollInertia) {
      _cancelMomentum();
    }
    if (oldWidget.active && !widget.active) {
      if (defaultTargetPlatform == TargetPlatform.android) {
        _clearAndroidComposition();
        unawaited(
          AndroidTerminalInputPlatform.instance.release(_androidInputOwner),
        );
      } else {
        _inputFocus.unfocus();
      }
    }
    if (oldWidget.active != widget.active ||
        oldWidget.splitOpen != widget.splitOpen ||
        oldWidget.syncInput != widget.syncInput ||
        oldWidget.canSplit != widget.canSplit) {
      _notifyControls();
    }
  }

  @override
  void dispose() {
    unawaited(_androidInputSubscription?.cancel());
    if (defaultTargetPlatform == TargetPlatform.android) {
      unawaited(
        AndroidTerminalInputPlatform.instance.release(_androidInputOwner),
      );
    }
    _inputController
      ..removeListener(_handleTextInput)
      ..dispose();
    _inputFocus.removeListener(_handleInputFocusChange);
    _inputFocus.dispose();
    _searchFocus.dispose();
    _searchController.dispose();
    _historyScroll.dispose();
    _historyHighlights.dispose();
    _androidComposition.dispose();
    _cancelMomentum();
    _historyLoadingDelay?.cancel();
    _recoveryNoticeGate.dispose();
    _historyRequestEpoch += 1;
    unawaited(_resizeSubscription?.cancel());
    unawaited(_terminalStateSubscription?.cancel());
    unawaited(_deliveryStateSubscription?.cancel());
    if (_history case final history?) {
      _releaseHistory(widget.connection, history);
    }
    controlRevision.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, outerConstraints) {
        final visibleHeight = widget.manageKeyboardInset
            ? math.max(0.0, outerConstraints.maxHeight - widget.keyboardInset)
            : outerConstraints.maxHeight;
        return Align(
          alignment: Alignment.topCenter,
          child: SizedBox(
            height: visibleHeight,
            child: Column(
              children: [
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      _scheduleHistoryProjectionFit(constraints.biggest);
                      return Focus(
                        canRequestFocus: widget.active,
                        onKeyEvent: _handleHardwareKey,
                        child: Stack(
                          fit: StackFit.expand,
                          children: [
                            _buildTerminalContent(),
                            if (defaultTargetPlatform != TargetPlatform.android)
                              Positioned(
                                left: 0,
                                bottom: 0,
                                width: 1,
                                height: 1,
                                child: ExcludeSemantics(
                                  child: ClipRect(
                                    child: EditableText(
                                      controller: _inputController,
                                      focusNode: _inputFocus,
                                      style: const TextStyle(
                                        color: Colors.transparent,
                                        fontSize: 1,
                                      ),
                                      cursorColor: Colors.transparent,
                                      backgroundCursorColor: Colors.transparent,
                                      keyboardType: TextInputType.multiline,
                                      textInputAction: TextInputAction.newline,
                                      autocorrect: false,
                                      enableSuggestions: false,
                                      smartDashesType: SmartDashesType.disabled,
                                      smartQuotesType: SmartQuotesType.disabled,
                                      onEditingComplete: () {},
                                      onSubmitted: (_) => _sendEnter(),
                                      maxLines: null,
                                    ),
                                  ),
                                ),
                              ),
                            if (defaultTargetPlatform == TargetPlatform.android)
                              Positioned.fill(
                                child: IgnorePointer(
                                  child:
                                      ValueListenableBuilder<
                                        ({bool active, String text})
                                      >(
                                        valueListenable: _androidComposition,
                                        builder: (context, composition, _) =>
                                            composition.active
                                            ? _TerminalCompositionOverlay(
                                                text: composition.text,
                                              )
                                            : const SizedBox.shrink(),
                                      ),
                                ),
                              ),
                            if (!widget.externalControls)
                              ...buildHeaderOverlays(),
                            if (!widget.externalControls)
                              ...buildExternalOverlays(),
                          ],
                        ),
                      );
                    },
                  ),
                ),
                if (!widget.externalControls) buildExternalKeyBar(),
              ],
            ),
          ),
        );
      },
    );
  }

  List<Widget> buildExternalOverlays() {
    return [
      if (_toolsOpen)
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: toggleTools,
          ),
        ),
      if (_toolsOpen)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _TerminalToolsSheet(
            terminalState: _terminalState,
            terminalSize: widget.connection.terminalSize,
            resizeControl: _resizeControl,
            inputEnabled:
                !_readOnly &&
                _history == null &&
                !_historyLoading &&
                !_searchOpen &&
                _deliveryState == TerminalDeliveryState.ready,
            historyActive: _history != null || _historyLoading,
            historyActionEnabled: !_readOnly || _history == null,
            keyboardMode: _keyboardMode,
            searchActive: _searchOpen,
            resizeAvailable: !_readOnly,
            splitOpen: widget.splitOpen,
            syncInput: widget.syncInput,
            canSplit: widget.canSplit,
            onSplit: _requestSplit,
            onToggleSync: widget.onToggleSync,
            onCloseSplit: _requestCloseSplit,
            onKeyboardModeChanged: (mode) =>
                unawaited(_changeKeyboardMode(mode)),
            onClose: toggleTools,
            onSelected: _handleTerminalToolAction,
          ),
        ),
      if (_searchOpen)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _HistorySearchBar(
            controller: _searchController,
            focusNode: _searchFocus,
            mode: _searchMode,
            searching: _searching,
            scanning: _searchScanActive,
            matchCount: _searchMatchCount,
            wrapped: _searchWrapped,
            onQueryChanged: _handleSearchQueryChanged,
            onModeChanged: _changeSearchMode,
            onPrevious: () => _navigateSearch(
              HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD,
            ),
            onNext: () => _navigateSearch(
              HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD,
            ),
            onClose: _closeSearch,
          ),
        ),
      if (_selectionMode)
        Positioned(
          left: 8,
          right: 8,
          bottom: 8,
          child: _TerminalSelectionToolbar(
            selectionAvailable: _selection != null,
            copyPending: _copying,
            onSelectAll: _selectAllLoaded,
            onSelectVisible: _selectVisible,
            onCopy: _copySelection,
            onClose: _closeSelection,
          ),
        ),
    ];
  }

  List<Widget> buildHeaderOverlays() {
    return [
      if (_fnOpen)
        TerminalHeaderQuickKeysLayer(
          child: Consumer(
            builder: (context, ref, _) => TerminalQuickKeysPanel(
              actions:
                  ref.watch(terminalQuickActionsProvider).valueOrNull ??
                  defaultTerminalQuickActions,
              inputEnabled:
                  !_readOnly &&
                  _history == null &&
                  !_historyLoading &&
                  _deliveryState == TerminalDeliveryState.ready,
              onClose: _toggleFn,
              onAction: _runQuickAction,
              attachedToHeader: true,
            ),
          ),
        ),
    ];
  }

  Widget buildExternalKeyBar() {
    final inputEnabled =
        !_readOnly &&
        _history == null &&
        !_historyLoading &&
        !_selectionMode &&
        !_searchOpen &&
        _deliveryState == TerminalDeliveryState.ready;
    return Consumer(
      builder: (context, ref, _) => TerminalCommandBar(
        actions:
            ref.watch(terminalQuickActionsProvider).valueOrNull ??
            defaultTerminalQuickActions,
        inputEnabled: inputEnabled,
        modifiers: _modifiers,
        keyboardControl: _KeyboardKeyButton(
          keyboardVisible: widget.keyboardInset > 100,
          focusLocked: _keyboardFocusLocked,
          onTap: inputEnabled ? _toggleKeyboardVisibility : null,
          onLongPress: inputEnabled ? _toggleKeyboardFocusLock : null,
        ),
        onAction: _runQuickAction,
        onConfigure: _openCommandBarEditor,
        functionKeysActive: _fnOpen,
        onFunctionKeys: _toggleFn,
      ),
    );
  }

  void _openCommandBarEditor() {
    final container = ProviderScope.containerOf(context);
    unawaited(
      showAnyttyCommandBarEditor(
        context: context,
        actions:
            container.read(terminalQuickActionsProvider).valueOrNull ??
            defaultTerminalQuickActions,
        onSave: container.read(terminalQuickActionsProvider.notifier).save,
      ),
    );
  }

  void _requestSplit(_TerminalSplitTarget target) {
    setState(() => _toolsOpen = false);
    _notifyControls();
    widget.onSplit(target);
  }

  void _requestCloseSplit() {
    setState(() => _toolsOpen = false);
    _notifyControls();
    widget.onCloseSplit();
  }

  Widget _buildTerminalContent() {
    if (_readOnly) {
      if (_history case final history?) {
        return TerminalHistoryPresentation(
          ready: _historyPresented,
          fallback: ColoredBox(
            color: _terminalThemeColor(widget.settings.theme.background),
          ),
          child: _buildHistoryContent(history),
        );
      }
      if (_historyLoading) {
        if (_historyLoadingVisible) {
          return const _WaitingForSnapshot(label: 'Loading history');
        }
        return ColoredBox(
          color: _terminalThemeColor(widget.settings.theme.background),
        );
      }
      if (_historyError case final historyError?) {
        return _TerminalFailure(
          dark: true,
          message: historyError,
          onRetry: () => unawaited(_ensureHistory()),
        );
      }
    }

    final history = _history;
    return TerminalHistoryPresentation(
      ready: history != null && _historyPresented,
      fallbackInteractive: history == null && !_historyLoading,
      fallback: _buildLiveTerminalContent(),
      child: history == null
          ? const SizedBox.expand()
          : _buildHistoryContent(history),
    );
  }

  Widget _buildLiveTerminalContent() {
    return StreamBuilder<CanonicalLiveScreen>(
      stream: widget.connection.watchScreens(),
      initialData: widget.connection.current,
      builder: (context, snapshot) {
        final screen = snapshot.data ?? widget.connection.current;
        if (snapshot.hasError && screen == null) {
          return _TerminalFailure(
            dark: true,
            message: snapshot.error.toString(),
            onRetry: widget.onReconnect,
          );
        }
        if (screen == null) {
          if (_historyLoading) {
            if (_historyLoadingVisible) {
              return const _WaitingForSnapshot(label: 'Loading history');
            }
            return ColoredBox(
              color: _terminalThemeColor(widget.settings.theme.background),
            );
          }
          return const _WaitingForSnapshot(
            label: 'Waiting for screen snapshot',
          );
        }
        return _buildKeyboardAdjustedTerminal(
          screen: screen,
          child: TerminalKeyboardFrameFreeze(
            visualInset: widget.keyboardVisualInset,
            settledInset: widget.settledKeyboardInset,
            child: RepaintBoundary(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  Consumer(
                    builder: (context, ref, _) {
                      final preferences =
                          ref
                              .watch(terminalPetalMenuPreferencesProvider)
                              .valueOrNull ??
                          TerminalPetalMenuPreferences.defaults;
                      final actions = _terminalPetalActions(preferences);
                      return TerminalPetalMenuRegion(
                        actions: actions,
                        enabled:
                            preferences.enabled &&
                            actions.isNotEmpty &&
                            !_toolsOpen &&
                            !_fnOpen &&
                            !_historyLoading &&
                            !_searchOpen &&
                            !_selectionMode,
                        hapticsEnabled: preferences.hapticsEnabled,
                        onOpened: () {
                          widget.onActivate();
                          _handleLiveInteractionCancel();
                        },
                        onSelected: (action) =>
                            unawaited(_handleTerminalPetalAction(action.id)),
                        child: TerminalCanvas(
                          screen: screen,
                          settings: widget.settings,
                          reserveLongPress: false,
                          onPresented:
                              widget.connection.acknowledgePresentation,
                          onLinkTap: _openTerminalLink,
                          onTerminalTap: (position) =>
                              _handleLiveTap(screen, position),
                          onInteractionStart: _handleLiveInteractionStart,
                          onInteractionCancel: _handleLiveInteractionCancel,
                          onVerticalDragStart: (details) =>
                              _lastLiveDragPosition = details.localPosition,
                          onVerticalDragUpdate: (details) =>
                              _handleLiveDrag(screen, details),
                          onVerticalDragEnd: (details) =>
                              _handleLiveDragEnd(screen, details),
                          onVerticalDragCancel: _handleLiveInteractionCancel,
                        ),
                      );
                    },
                  ),
                  if (snapshot.hasError)
                    _TerminalDeliveryBanner(
                      message: snapshot.error.toString(),
                      onRetry: widget.onReconnect,
                      error: true,
                    )
                  else if (_deliveryState == TerminalDeliveryState.stalled)
                    _TerminalDeliveryBanner(
                      message: 'Terminal rendering is paused',
                      onRetry: widget.onReconnect,
                      error: false,
                    )
                  else if (_deliveryState == TerminalDeliveryState.recovering &&
                      _recoveryNoticeVisible)
                    const _TerminalDeliveryBanner(
                      message: 'Reconnecting terminal',
                      error: false,
                    ),
                  if (_historyLoadingVisible)
                    const _HistoryLoadingStatus(label: 'Loading history'),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildKeyboardAdjustedTerminal({
    required CanonicalLiveScreen screen,
    required Widget child,
  }) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final keyboardInset = widget.keyboardInset;
        final settledKeyboardInset = widget.settledKeyboardInset;
        final alternate =
            screen.alternateScreen || screen.modes?.alternateScreen == true;
        final mode = resolveTerminalKeyboardMode(
          _keyboardMode,
          alternateScreen: alternate,
        );
        final shifting =
            keyboardInset > 0 && mode == TerminalKeyboardMode.shift;
        final resizing =
            keyboardInset > 0 && mode == TerminalKeyboardMode.resize;
        final keyboardAnimating =
            (keyboardInset - settledKeyboardInset).abs() > 1;
        if (keyboardInset <= 1 && settledKeyboardInset <= 1) {
          _fullTerminalViewportHeight = constraints.maxHeight;
        }
        final preserveGrid = shifting || keyboardAnimating;
        final layoutHeight = resizing
            ? math.max(0.0, constraints.maxHeight - keyboardInset)
            : preserveGrid
            ? _fullTerminalViewportHeight ?? constraints.maxHeight
            : constraints.maxHeight;
        _scheduleViewportFit(Size(constraints.maxWidth, layoutHeight));
        if (!preserveGrid) {
          return Align(
            // The workspace moves upward as one unit. Bottom alignment places a
            // resized terminal below the clipped area before that translation.
            alignment: resizing ? Alignment.bottomLeft : Alignment.topLeft,
            child: SizedBox(
              width: constraints.maxWidth,
              height: layoutHeight,
              child: child,
            ),
          );
        }

        final shift = shifting
            ? resolveTerminalKeyboardShift(
                keyboardInset: keyboardInset,
                visibleHeight: constraints.maxHeight,
                cursorRow: screen.cursor?.row,
                rowHeight: widget.settings.metrics.rowHeight,
              )
            : 0.0;
        return ClipRect(
          child: OverflowBox(
            alignment: Alignment.topLeft,
            minWidth: constraints.maxWidth,
            maxWidth: constraints.maxWidth,
            minHeight: layoutHeight,
            maxHeight: layoutHeight,
            child: Transform.translate(
              offset: Offset(0, -shift),
              child: SizedBox(
                width: constraints.maxWidth,
                height: layoutHeight,
                child: child,
              ),
            ),
          ),
        );
      },
    );
  }

  void _bindResizeControl() {
    _resizeControl = widget.connection.resizeControl;
    _resizeSubscription = widget.connection.watchResizeControl().listen((
      control,
    ) {
      if (!mounted) return;
      setState(() => _resizeControl = control);
      _notifyControls();
      if (control.canResize && _viewportCells != null) {
        unawaited(_fitViewport());
      }
    });
  }

  void _bindTerminalState() {
    _terminalState = widget.connection.terminalState;
    _terminalStateSubscription = widget.connection.watchTerminalState().listen((
      state,
    ) {
      if (!mounted) return;
      final becameReadOnly =
          !terminalUsesHistoryOnly(_terminalState) &&
          terminalUsesHistoryOnly(state);
      setState(() => _terminalState = state);
      _notifyControls();
      if (becameReadOnly) {
        _cancelMomentum();
        FocusManager.instance.primaryFocus?.unfocus();
        setState(() {
          _modifiers = const TerminalModifierState();
          _fnOpen = false;
        });
        _notifyControls();
        _scheduleReadOnlyHistory();
      }
    });
    if (_readOnly) _scheduleReadOnlyHistory();
  }

  void _bindDeliveryState() {
    _deliveryState = widget.connection.deliveryState;
    _updateRecoveryNotice(_deliveryState);
    _deliveryStateSubscription = widget.connection.watchDeliveryState().listen((
      state,
    ) {
      if (mounted) {
        setState(() => _deliveryState = state);
        _updateRecoveryNotice(state);
        _notifyControls();
      }
    });
  }

  void _updateRecoveryNotice(TerminalDeliveryState state) {
    _recoveryNoticeGate.setRecovering(
      state == TerminalDeliveryState.recovering,
    );
  }

  void _scheduleReadOnlyHistory() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted &&
          _readOnly &&
          _historyViewportCells != null &&
          _history == null &&
          !_historyLoading) {
        unawaited(_ensureHistory());
      }
    });
  }

  void _scheduleHistoryProjectionFit(Size viewport) {
    if (viewport.isEmpty ||
        !viewport.width.isFinite ||
        !viewport.height.isFinite) {
      return;
    }
    final cells = (
      cols: (viewport.width / widget.settings.metrics.cellWidth).floor().clamp(
        20,
        500,
      ),
      rows: (viewport.height / widget.settings.metrics.rowHeight).floor().clamp(
        4,
        300,
      ),
    );
    if (_historyViewportCells?.cols != cells.cols) {
      _historyProjectionFailedCols = null;
    }
    _historyViewportCells = cells;
    final history = _history;
    if (history != null &&
        history.cols != cells.cols &&
        _historyProjectionFailedCols != cells.cols &&
        !_historyLoading) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(
          _restartHistoryProjection(expected: history, cols: cells.cols),
        );
      });
    } else if (_readOnly && history == null && !_historyLoading) {
      _scheduleReadOnlyHistory();
    }
  }

  void _scheduleViewportFit(Size viewport) {
    if (_readOnly ||
        _history != null ||
        viewport.isEmpty ||
        !viewport.width.isFinite ||
        !viewport.height.isFinite) {
      return;
    }
    final cells = (
      cols: (viewport.width / widget.settings.metrics.cellWidth).floor().clamp(
        20,
        500,
      ),
      rows: (viewport.height / widget.settings.metrics.rowHeight).floor().clamp(
        4,
        300,
      ),
    );
    if (_viewportCells == cells) return;
    _viewportCells = cells;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (widget.settings.autoAcquireResizeOwner &&
          !_resizeControl.canResize &&
          !_autoAcquireAttempted) {
        unawaited(_autoAcquireResizeOwnership());
      } else {
        unawaited(_fitViewport());
      }
    });
  }

  Future<void> _autoAcquireResizeOwnership() async {
    final target = _viewportCells;
    if (target == null || _autoAcquireAttempted || _readOnly) return;
    _autoAcquireAttempted = true;
    try {
      final control = await widget.connection.requestResizeOwnership(
        cols: target.cols,
        rows: target.rows,
      );
      if (!mounted) return;
      setState(() => _resizeControl = control);
      await _fitViewport();
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not acquire terminal size: $error')),
      );
    }
  }

  Future<void> _fitViewport() async {
    if (_readOnly ||
        _resizePending ||
        _history != null ||
        !_resizeControl.canResize) {
      return;
    }
    _resizePending = true;
    try {
      while (mounted && _history == null && _resizeControl.canResize) {
        final target = _viewportCells;
        if (target == null) return;
        await widget.connection.fitViewport(
          cols: target.cols,
          rows: target.rows,
        );
        if (_viewportCells == target) return;
      }
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('Resize failed: $error')));
    } finally {
      _resizePending = false;
    }
  }

  Future<void> _showResizeControls() async {
    if (_readOnly) return;
    FocusManager.instance.primaryFocus?.unfocus();
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      backgroundColor: AnyttyPalette.of(context).surface,
      showDragHandle: true,
      builder: (context) => _ResizeControlSheet(
        connection: widget.connection,
        initialControl: _resizeControl,
        viewportCells: _viewportCells,
      ),
    );
  }

  Future<void> _showTerminalSettings() async {
    FocusManager.instance.primaryFocus?.unfocus();
    await showModalBottomSheet<void>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (context) => FractionallySizedBox(
        heightFactor: 0.86,
        child: _TerminalSettingsSheet(
          initialSettings: widget.settings,
          onChanged: widget.onSettingsChanged,
        ),
      ),
    );
  }

  Future<void> _changeKeyboardMode(TerminalKeyboardMode next) async {
    if (_keyboardMode == next) return;
    final previous = _keyboardMode;
    setState(() {
      _keyboardMode = next;
      _viewportCells = null;
    });
    _notifyControls();
    try {
      await widget.onKeyboardModeChanged(next);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _keyboardMode = previous;
        _viewportCells = null;
      });
      _notifyControls();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not save keyboard mode: $error')),
      );
    }
  }

  void toggleTools() {
    setState(() {
      _toolsOpen = !_toolsOpen;
      if (_toolsOpen) _fnOpen = false;
    });
    _notifyControls();
  }

  bool handleBack() {
    if (_fnOpen) {
      setState(() => _fnOpen = false);
      _notifyControls();
      return true;
    }
    if (_toolsOpen) {
      setState(() => _toolsOpen = false);
      _notifyControls();
      return true;
    }
    if (_selectionMode) {
      _closeSelection();
      return true;
    }
    if (_searchOpen) {
      _closeSearch();
      return true;
    }
    if (_history != null && !_readOnly) {
      unawaited(_toggleHistory());
      return true;
    }
    return false;
  }

  Future<void> _handleTerminalToolAction(_TerminalToolAction action) async {
    if (!mounted) return;
    setState(() => _toolsOpen = false);
    _notifyControls();
    switch (action) {
      case _TerminalToolAction.commandBar:
        _openCommandBarEditor();
      case _TerminalToolAction.quickKeys:
        _toggleFn();
      case _TerminalToolAction.keyboard:
        _toggleKeyboardVisibility();
      case _TerminalToolAction.enter:
        _sendEnter();
      case _TerminalToolAction.escape:
        _sendHid(_usbKeyboardPage | 0x29);
      case _TerminalToolAction.tab:
        _sendHid(_usbKeyboardPage | 0x2b);
      case _TerminalToolAction.backspace:
        _sendHid(_usbKeyboardPage | 0x2a);
      case _TerminalToolAction.delete:
        _sendHid(_usbKeyboardPage | 0x4c);
      case _TerminalToolAction.interrupt:
        _sendText('\x03');
      case _TerminalToolAction.eof:
        _sendText('\x04');
      case _TerminalToolAction.suspend:
        _sendText('\x1a');
      case _TerminalToolAction.clear:
        _sendText('\x0c');
      case _TerminalToolAction.arrowLeft:
        _sendHid(_usbKeyboardPage | 0x50);
      case _TerminalToolAction.arrowDown:
        _sendHid(_usbKeyboardPage | 0x51);
      case _TerminalToolAction.arrowUp:
        _sendHid(_usbKeyboardPage | 0x52);
      case _TerminalToolAction.arrowRight:
        _sendHid(_usbKeyboardPage | 0x4f);
      case _TerminalToolAction.home:
        _sendHid(_usbKeyboardPage | 0x4a);
      case _TerminalToolAction.end:
        _sendHid(_usbKeyboardPage | 0x4d);
      case _TerminalToolAction.pageUp:
        _sendHid(_usbKeyboardPage | 0x4b);
      case _TerminalToolAction.pageDown:
        _sendHid(_usbKeyboardPage | 0x4e);
      case _TerminalToolAction.paste:
        await _pasteClipboard();
      case _TerminalToolAction.history:
        await _toggleHistory();
      case _TerminalToolAction.search:
        await _toggleSearch();
      case _TerminalToolAction.selection:
        await _openSelection();
      case _TerminalToolAction.copyScreen:
        await _copyLiveScreen();
      case _TerminalToolAction.resources:
        await widget.onShowResources();
      case _TerminalToolAction.files:
        await widget.onOpenFiles();
      case _TerminalToolAction.split:
        _requestSplit(_TerminalSplitTarget.below);
      case _TerminalToolAction.splitRows:
        _requestSplit(_TerminalSplitTarget.below);
      case _TerminalToolAction.splitColumns:
        _requestSplit(_TerminalSplitTarget.right);
      case _TerminalToolAction.syncInput:
        widget.onToggleSync();
      case _TerminalToolAction.resize:
        await _showResizeControls();
      case _TerminalToolAction.reconnect:
        widget.onReconnect();
      case _TerminalToolAction.settings:
        await _showTerminalSettings();
    }
    _notifyControls();
  }

  List<TerminalPetalMenuItem> _terminalPetalActions(
    TerminalPetalMenuPreferences preferences,
  ) {
    final layout = preferences.visibleLayout;
    var cursor = 0;
    List<TerminalPetalMenuItem> buildLevel(int depth) {
      final actions = <TerminalPetalMenuItem>[];
      while (cursor < layout.length && layout[cursor].depth == depth) {
        final placement = layout[cursor];
        cursor += 1;
        final children = cursor < layout.length && layout[cursor].depth > depth
            ? buildLevel(depth + 1)
            : const <TerminalPetalMenuItem>[];
        final action = _terminalPetalAction(placement.id, children: children);
        if (action != null) actions.add(action);
      }
      return List.unmodifiable(actions);
    }

    return buildLevel(0);
  }

  TerminalPetalMenuItem? _terminalPetalAction(
    String id, {
    List<TerminalPetalMenuItem> children = const [],
  }) => switch (id) {
    'history' => TerminalPetalMenuItem(
      id: 'history',
      label: _history == null ? 'History' : 'Live',
      icon: LucideIcons.history,
      enabled: !_readOnly || _history == null,
      children: children,
    ),
    'search' => TerminalPetalMenuItem(
      id: 'search',
      label: 'Search',
      icon: LucideIcons.search,
      children: children,
    ),
    'selection' => TerminalPetalMenuItem(
      id: 'selection',
      label: 'Select',
      icon: LucideIcons.scanText,
      children: children,
    ),
    'paste' => TerminalPetalMenuItem(
      id: 'paste',
      label: 'Paste',
      icon: LucideIcons.clipboardPaste,
      enabled:
          !_readOnly &&
          _history == null &&
          !_historyLoading &&
          _deliveryState == TerminalDeliveryState.ready,
      children: children,
    ),
    'enter' => TerminalPetalMenuItem(
      id: 'enter',
      label: 'Enter',
      icon: LucideIcons.cornerDownLeft,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'escape' => TerminalPetalMenuItem(
      id: 'escape',
      label: 'Esc',
      icon: LucideIcons.badgeX,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'resources' => TerminalPetalMenuItem(
      id: 'resources',
      label: 'Resources',
      icon: LucideIcons.activity,
      children: children,
    ),
    'more' => TerminalPetalMenuItem(
      id: 'more',
      label: 'More',
      icon: LucideIcons.ellipsis,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'input-tools' => TerminalPetalMenuItem(
      id: 'input-tools',
      label: 'Input',
      icon: LucideIcons.command,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'navigation-tools' => TerminalPetalMenuItem(
      id: 'navigation-tools',
      label: 'Navigate',
      icon: LucideIcons.navigation,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'session-tools' => TerminalPetalMenuItem(
      id: 'session-tools',
      label: 'Layout',
      icon: LucideIcons.panelsTopLeft,
      enabled: children.isNotEmpty,
      children: children,
    ),
    'command-bar' => TerminalPetalMenuItem(
      id: 'command-bar',
      label: 'Shortcut',
      icon: LucideIcons.slidersHorizontal,
      children: children,
    ),
    'copy-screen' => TerminalPetalMenuItem(
      id: 'copy-screen',
      label: 'Copy',
      icon: LucideIcons.copy,
      children: children,
    ),
    'quick-keys' => TerminalPetalMenuItem(
      id: 'quick-keys',
      label: 'Quick Keys',
      icon: LucideIcons.zap,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'keyboard' => TerminalPetalMenuItem(
      id: 'keyboard',
      label: 'Keyboard',
      icon: LucideIcons.keyboard,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'tab' => TerminalPetalMenuItem(
      id: 'tab',
      label: 'Tab',
      icon: LucideIcons.arrowRightToLine,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'backspace' => TerminalPetalMenuItem(
      id: 'backspace',
      label: 'Backspace',
      icon: LucideIcons.delete,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'delete' => TerminalPetalMenuItem(
      id: 'delete',
      label: 'Delete',
      icon: LucideIcons.eraser,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'interrupt' => TerminalPetalMenuItem(
      id: 'interrupt',
      label: 'Ctrl+C',
      icon: LucideIcons.circleStop,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'eof' => TerminalPetalMenuItem(
      id: 'eof',
      label: 'Ctrl+D',
      icon: LucideIcons.logOut,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'suspend' => TerminalPetalMenuItem(
      id: 'suspend',
      label: 'Ctrl+Z',
      icon: LucideIcons.pause,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'clear' => TerminalPetalMenuItem(
      id: 'clear',
      label: 'Clear',
      icon: LucideIcons.eraser,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-left' => TerminalPetalMenuItem(
      id: 'arrow-left',
      label: 'Left',
      icon: LucideIcons.arrowLeft,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-down' => TerminalPetalMenuItem(
      id: 'arrow-down',
      label: 'Down',
      icon: LucideIcons.arrowDown,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-up' => TerminalPetalMenuItem(
      id: 'arrow-up',
      label: 'Up',
      icon: LucideIcons.arrowUp,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'arrow-right' => TerminalPetalMenuItem(
      id: 'arrow-right',
      label: 'Right',
      icon: LucideIcons.arrowRight,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'home' => TerminalPetalMenuItem(
      id: 'home',
      label: 'Home',
      icon: LucideIcons.home,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'end' => TerminalPetalMenuItem(
      id: 'end',
      label: 'End',
      icon: LucideIcons.arrowRightToLine,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'page-up' => TerminalPetalMenuItem(
      id: 'page-up',
      label: 'PgUp',
      icon: LucideIcons.chevronsUp,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'page-down' => TerminalPetalMenuItem(
      id: 'page-down',
      label: 'PgDn',
      icon: LucideIcons.chevronsDown,
      enabled: _terminalPetalInputEnabled,
      children: children,
    ),
    'split' => TerminalPetalMenuItem(
      id: 'split',
      label: 'Split',
      icon: LucideIcons.rows2,
      enabled: widget.canSplit,
      children: children,
    ),
    'split-rows' => TerminalPetalMenuItem(
      id: 'split-rows',
      label: 'Rows',
      icon: LucideIcons.rows2,
      enabled: widget.canSplit,
      children: children,
    ),
    'split-columns' => TerminalPetalMenuItem(
      id: 'split-columns',
      label: 'Columns',
      icon: LucideIcons.columns2,
      enabled: widget.canSplit,
      children: children,
    ),
    'sync-input' => TerminalPetalMenuItem(
      id: 'sync-input',
      label: widget.syncInput ? 'Synced' : 'Sync',
      icon: LucideIcons.gitCompareArrows,
      enabled: widget.splitOpen,
      children: children,
    ),
    'resize' => TerminalPetalMenuItem(
      id: 'resize',
      label: 'Resize',
      icon: LucideIcons.maximize2,
      enabled: !_readOnly,
      children: children,
    ),
    'files' => TerminalPetalMenuItem(
      id: 'files',
      label: 'Files',
      icon: LucideIcons.folderOpen,
      children: children,
    ),
    'reconnect' => TerminalPetalMenuItem(
      id: 'reconnect',
      label: 'Reconnect',
      icon: LucideIcons.refreshCw,
      children: children,
    ),
    'settings' => TerminalPetalMenuItem(
      id: 'settings',
      label: 'Settings',
      icon: LucideIcons.settings,
      children: children,
    ),
    _ => null,
  };

  bool get _terminalPetalInputEnabled =>
      !_readOnly &&
      _history == null &&
      !_historyLoading &&
      !_searchOpen &&
      _deliveryState == TerminalDeliveryState.ready;

  Future<void> _handleTerminalPetalAction(String id) async {
    final action = switch (id) {
      'paste' => _TerminalToolAction.paste,
      'history' => _TerminalToolAction.history,
      'search' => _TerminalToolAction.search,
      'selection' => _TerminalToolAction.selection,
      'copy-screen' => _TerminalToolAction.copyScreen,
      'command-bar' => _TerminalToolAction.commandBar,
      'quick-keys' => _TerminalToolAction.quickKeys,
      'keyboard' => _TerminalToolAction.keyboard,
      'enter' => _TerminalToolAction.enter,
      'escape' => _TerminalToolAction.escape,
      'tab' => _TerminalToolAction.tab,
      'backspace' => _TerminalToolAction.backspace,
      'delete' => _TerminalToolAction.delete,
      'interrupt' => _TerminalToolAction.interrupt,
      'eof' => _TerminalToolAction.eof,
      'suspend' => _TerminalToolAction.suspend,
      'clear' => _TerminalToolAction.clear,
      'arrow-left' => _TerminalToolAction.arrowLeft,
      'arrow-down' => _TerminalToolAction.arrowDown,
      'arrow-up' => _TerminalToolAction.arrowUp,
      'arrow-right' => _TerminalToolAction.arrowRight,
      'home' => _TerminalToolAction.home,
      'end' => _TerminalToolAction.end,
      'page-up' => _TerminalToolAction.pageUp,
      'page-down' => _TerminalToolAction.pageDown,
      'resources' => _TerminalToolAction.resources,
      'files' => _TerminalToolAction.files,
      'split' => _TerminalToolAction.split,
      'split-rows' => _TerminalToolAction.splitRows,
      'split-columns' => _TerminalToolAction.splitColumns,
      'sync-input' => _TerminalToolAction.syncInput,
      'resize' => _TerminalToolAction.resize,
      'reconnect' => _TerminalToolAction.reconnect,
      'settings' => _TerminalToolAction.settings,
      _ => null,
    };
    if (action != null) await _handleTerminalToolAction(action);
  }

  Widget _buildHistoryContent(FrozenHistory history) {
    final layout = _historyLayoutFor(history);
    final viewportRows =
        _historyViewportCells?.rows ??
        (history.anchor.screenRows > 0 ? history.anchor.screenRows : 24);
    final trailingRows = historyViewportTailRows(
      history: history,
      viewportRows: viewportRows,
    );
    return Stack(
      fit: StackFit.expand,
      children: [
        ScrollbarTheme(
          data: ScrollbarTheme.of(context).copyWith(
            thumbColor: WidgetStateProperty.resolveWith((states) {
              final foreground = _terminalThemeColor(
                widget.settings.theme.foreground,
              );
              return foreground.withValues(
                alpha: states.contains(WidgetState.dragged) ? 0.72 : 0.38,
              );
            }),
            thickness: WidgetStateProperty.resolveWith(
              (states) => states.contains(WidgetState.dragged) ? 8 : 3,
            ),
            radius: const Radius.circular(3),
            minThumbLength: 24,
          ),
          child: Scrollbar(
            controller: _historyScroll,
            interactive: true,
            child: TerminalHistoryCanvas(
              rows: history.rows,
              cols: history.cols,
              settings: widget.settings,
              scrollController: _historyScroll,
              layout: layout,
              highlights: _historyHighlights,
              searchMatches: _searchMatches,
              trailingRows: trailingRows,
              selectionEnabled: _selectionMode,
              canLoadOlder: history.hasMore || _historyLoading,
              onLinkTap: _openTerminalLink,
              onSelectionChanged: (selection) {
                final selectionBecameAvailable = _selection == null;
                _selection = selection;
                _syncHistoryHighlights();
                if (selectionBecameAvailable) _notifyControls();
              },
            ),
          ),
        ),
        Positioned(
          top: 8,
          left: 8,
          right: 8,
          child: _HistoryContextBar(
            liveEnabled: !_readOnly,
            searchActive: _searchOpen,
            selectionActive: _selectionMode,
            status: _historyLoadingVisible
                ? 'Loading older rows'
                : '${history.rows.length} / ${history.logicalTotal}',
            onLive: _toggleHistory,
            onSearch: _toggleSearch,
            onSelection: _openSelection,
          ),
        ),
        if (_historyError case final error?)
          Positioned(
            left: 8,
            right: 8,
            bottom: 8,
            child: Material(
              color: const Color(0xff7f1d1d),
              borderRadius: BorderRadius.circular(4),
              child: Row(
                children: [
                  Expanded(
                    child: Padding(
                      padding: const EdgeInsets.only(
                        left: 10,
                        top: 8,
                        bottom: 8,
                      ),
                      child: Text(
                        error,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 12,
                        ),
                      ),
                    ),
                  ),
                  if (_historyRequiresReload)
                    IconButton(
                      tooltip: 'Reload frozen history',
                      color: Colors.white,
                      onPressed: _reloadHistory,
                      icon: const Icon(Icons.refresh_rounded),
                    ),
                  IconButton(
                    tooltip: 'Dismiss history error',
                    color: Colors.white,
                    onPressed: () => setState(() {
                      _historyError = null;
                      _historyRequiresReload = false;
                    }),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _toggleHistory() async {
    if (_readOnly) {
      if (_history == null) await _ensureHistory();
      return;
    }
    if (_history != null) {
      final history = _history!;
      setState(() {
        _cancelHistoryRequest();
        _history = null;
        _historyError = null;
        _historyRequiresReload = false;
        _historyPositioned = false;
        _historyPresented = false;
        _resetHistoryInteraction();
      });
      _releaseHistory(widget.connection, history);
      return;
    }
    if (_historyLoading) return;
    await _ensureHistory();
  }

  Future<FrozenHistory?> _ensureHistory() async {
    if (_history case final history?) return history;
    if (_historyLoading) return null;
    final connection = widget.connection;
    final projectionCols = _historyViewportCells?.cols;
    final requestEpoch = ++_historyRequestEpoch;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
      _historyPresented = false;
      _fnOpen = false;
    });
    _scheduleHistoryLoadingIndicator(requestEpoch);
    try {
      final merged = await connection.openHistory(cols: projectionCols);
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection) {
        _releaseHistory(connection, merged.history);
        return null;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _history = merged.history;
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
      });
      _scheduleHistoryPosition(merged.history);
      return merged.history;
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection) {
        return null;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = frozenHistoryRequiresReload(error.toString());
      });
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
      return null;
    }
  }

  Future<void> _restartHistoryProjection({
    required FrozenHistory expected,
    required int cols,
  }) async {
    if (_historyLoading || _history != expected || expected.cols == cols) {
      return;
    }
    final connection = widget.connection;
    final requestEpoch = ++_historyRequestEpoch;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
    });
    _scheduleHistoryLoadingIndicator(requestEpoch);
    try {
      final merged = await connection.openHistory(cols: cols);
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        _releaseHistory(connection, merged.history);
        return;
      }
      if (_historyViewportCells?.cols != cols) {
        _releaseHistory(connection, merged.history);
        _historyLoadingDelay?.cancel();
        setState(() {
          _historyLoading = false;
          _historyLoadingVisible = false;
        });
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _history = merged.history;
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
        _historyProjectionFailedCols = null;
        _historyPositioned = false;
        _resetHistoryInteraction();
      });
      _scheduleHistoryPosition(merged.history);
      if (expected.token != merged.history.token ||
          expected.generation != merged.history.generation) {
        _releaseHistory(connection, expected);
      }
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = true;
        _historyProjectionFailedCols = cols;
      });
    }
  }

  void _scheduleHistoryPosition(FrozenHistory expected) {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _history != expected || !_historyScroll.hasClients) {
        return;
      }
      final position = _historyScroll.position;
      final anchorRow = historyViewportAnchorRow(expected);
      final target = (anchorRow * widget.settings.metrics.rowHeight).clamp(
        0.0,
        position.maxScrollExtent,
      );
      _resetHistoryScrollSample(target.toDouble(), clearVelocity: true);
      _historyScroll.jumpTo(target);
      setState(() {
        _historyPositioned = true;
        _historyPresented = true;
      });
      _handleHistoryScroll();
    });
  }

  void _handleHistoryScroll() {
    _sampleHistoryScrollVelocity();
    final history = _history;
    final prefetch = adaptiveHistoryPrefetchPlan(
      baseThresholdRows: widget.settings.historyPrefetchThresholdRows,
      upwardVelocityRowsPerSecond: _historyUpwardVelocityRows,
    );
    if (_searchOpen ||
        !_historyPositioned ||
        _historyLoading ||
        history == null ||
        !history.hasMore ||
        !_historyScroll.hasClients ||
        _historyScroll.position.pixels >
            prefetch.thresholdRows * widget.settings.metrics.rowHeight) {
      return;
    }
    unawaited(_loadOlderHistory(history, limit: prefetch.requestRows));
  }

  void _sampleHistoryScrollVelocity() {
    if (!_historyScroll.hasClients) return;
    final now = DateTime.now();
    final offset = _historyScroll.position.pixels;
    final previousTime = _historyScrollSampleTime;
    final previousOffset = _historyScrollSampleOffset;
    _historyScrollSampleTime = now;
    _historyScrollSampleOffset = offset;
    if (previousTime == null || previousOffset == null) return;
    final elapsedMicros = now.difference(previousTime).inMicroseconds;
    if (elapsedMicros < 4000 || elapsedMicros > 500000) {
      _historyUpwardVelocityRows *= 0.6;
      return;
    }
    final upwardRows =
        (previousOffset - offset) / widget.settings.metrics.rowHeight;
    final instantaneous = math.max(
      0.0,
      upwardRows * Duration.microsecondsPerSecond / elapsedMicros,
    );
    _historyUpwardVelocityRows = instantaneous > 0
        ? math.max(instantaneous, _historyUpwardVelocityRows * 0.72)
        : _historyUpwardVelocityRows * 0.6;
  }

  void _resetHistoryScrollSample(double offset, {required bool clearVelocity}) {
    _historyScrollSampleTime = DateTime.now();
    _historyScrollSampleOffset = offset;
    if (clearVelocity) _historyUpwardVelocityRows = 0;
  }

  Future<void> _loadOlderHistory(
    FrozenHistory expected, {
    int limit = maximumHistoryWindowRequestRows,
  }) async {
    final connection = widget.connection;
    final requestEpoch = ++_historyRequestEpoch;
    final previousOffset = _historyScroll.position.pixels;
    setState(() {
      _historyLoading = true;
      _historyLoadingVisible = false;
      _historyError = null;
      _historyRequiresReload = false;
    });
    _scheduleHistoryLoadingIndicator(requestEpoch);
    try {
      final merged = await connection.loadOlderHistory(expected, limit: limit);
      if (merged.prependedRows == 0 && merged.history.hasMore) {
        throw const NativeSessionException('History request did not advance');
      }
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _history = merged.history;
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyRequiresReload = false;
      });
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (_historyScroll.hasClients) {
          final target =
              previousOffset +
              merged.prependedRows * widget.settings.metrics.rowHeight;
          final clampedTarget = target
              .clamp(0, _historyScroll.position.maxScrollExtent)
              .toDouble();
          _resetHistoryScrollSample(clampedTarget, clearVelocity: false);
          _historyScroll.jumpTo(clampedTarget);
        }
        _handleHistoryScroll();
      });
    } catch (error) {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          connection != widget.connection ||
          _history != expected) {
        return;
      }
      _historyLoadingDelay?.cancel();
      setState(() {
        _historyLoading = false;
        _historyLoadingVisible = false;
        _historyError = error.toString();
        _historyRequiresReload = frozenHistoryRequiresReload(error.toString());
      });
    }
  }

  void _scheduleHistoryLoadingIndicator(int requestEpoch) {
    _historyLoadingDelay?.cancel();
    _historyLoadingDelay = Timer(const Duration(seconds: 2), () {
      if (!mounted ||
          requestEpoch != _historyRequestEpoch ||
          !_historyLoading) {
        return;
      }
      setState(() => _historyLoadingVisible = true);
    });
  }

  void _cancelHistoryRequest() {
    _historyRequestEpoch += 1;
    _historyLoadingDelay?.cancel();
    _historyLoadingDelay = null;
    _historyLoading = false;
    _historyLoadingVisible = false;
  }

  Future<void> _reloadHistory() async {
    if (_historyLoading) return;
    final previous = _history;
    setState(() {
      _historyProjectionFailedCols = null;
      _history = null;
      _historyError = null;
      _historyRequiresReload = false;
      _historyPositioned = false;
      _historyPresented = false;
      _resetHistoryInteraction();
    });
    if (previous != null) _releaseHistory(widget.connection, previous);
    await _ensureHistory();
  }

  Future<void> _openSelection() async {
    _hideKeyboard();
    final history = await _ensureHistory();
    if (!mounted || history == null) return;
    setState(() {
      _selectionMode = true;
      _selection = null;
      _searchOpen = false;
      _searchEpoch += 1;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _closeSelection() {
    setState(() {
      _selectionMode = false;
      _selection = null;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _selectAllLoaded() {
    final history = _history;
    if (history == null || history.rows.isEmpty) return;
    _selection = selectAllHistory(_historyLayoutFor(history));
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _selectVisible() {
    final history = _history;
    if (history == null || history.rows.isEmpty || !_historyScroll.hasClients) {
      return;
    }
    final position = _historyScroll.position;
    final first = (position.pixels / widget.settings.metrics.rowHeight).floor();
    final last =
        ((position.pixels + position.viewportDimension) /
                widget.settings.metrics.rowHeight)
            .ceil()
            .clamp(1, history.rows.length) -
        1;
    _selection = selectVisibleHistory(
      layout: _historyLayoutFor(history),
      firstRow: first,
      lastRow: last,
    );
    _syncHistoryHighlights();
    _notifyControls();
  }

  Future<void> _copyLiveScreen() async {
    final screen = widget.connection.current;
    if (screen == null) {
      _showTerminalNotice(
        anyttyText(
          context,
          en: 'No terminal screen is available',
          zh: '当前没有可复制的终端画面',
        ),
      );
      return;
    }
    final text = screen.screenRows
        .map(
          (row) => terminalRowSemanticText(
            row,
            maxCharacters: math.max(512, screen.cols * 4),
          ),
        )
        .join('\n')
        .trimRight();
    if (text.isEmpty) {
      _showTerminalNotice(
        anyttyText(context, en: 'The terminal screen is empty', zh: '终端画面为空'),
      );
      return;
    }
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    _showTerminalNotice(
      anyttyText(context, en: 'Terminal screen copied', zh: '已复制当前终端画面'),
    );
  }

  void _showTerminalNotice(String message) {
    final messenger = ScaffoldMessenger.maybeOf(context);
    if (messenger == null) return;
    messenger
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Text(message),
          duration: const Duration(milliseconds: 1500),
        ),
      );
  }

  Future<void> _copySelection() async {
    final history = _history;
    final selection = _selection;
    if (history == null || selection == null || _copying) return;
    setState(() => _copying = true);
    try {
      final range = normalizeHistorySelection(
        layout: _historyLayoutFor(history),
        selection: selection,
      );
      final text = await widget.connection.copyHistoryRange(history, range);
      await Clipboard.setData(ClipboardData(text: text));
      if (!mounted || !_sameFrozenHistory(history, _history)) return;
      setState(() {
        _copying = false;
        _selectionMode = false;
        _selection = null;
      });
      _syncHistoryHighlights();
      _notifyControls();
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Selection copied')));
    } catch (error) {
      if (!mounted) return;
      setState(() => _copying = false);
      _notifyControls();
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _toggleSearch() async {
    if (_searchOpen) {
      _closeSearch();
      return;
    }
    final history = await _ensureHistory();
    if (!mounted || history == null) return;
    setState(() {
      if (_historyLoading) _cancelHistoryRequest();
      _selectionMode = false;
      _selection = null;
      _searchOpen = true;
    });
    _syncHistoryHighlights();
    _notifyControls();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) _searchFocus.requestFocus();
    });
  }

  void _closeSearch() {
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _searchOpen = false;
      _searching = false;
      _searchScanActive = false;
      _searchEpoch += 1;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _changeSearchMode(HistorySearchMode mode) {
    if (mode == _searchMode) return;
    setState(() {
      _searchEpoch += 1;
      _searchMode = mode;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
    if (_searchController.text.isNotEmpty) {
      unawaited(
        _runSearch(HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD),
      );
    }
  }

  void _handleSearchQueryChanged(String query) {
    setState(() {
      _searchEpoch += 1;
      _searching = false;
      _searchScanActive = false;
      _searchMatch = null;
      _searchMatches = const [];
      _searchMatchCount = 0;
      _searchWrapped = false;
    });
    _syncHistoryHighlights();
    _notifyControls();
  }

  void _navigateSearch(HistorySearchDirection direction) {
    if (_searching) return;
    final next = adjacentHistorySearchMatch(
      matches: _searchMatches,
      current: _searchMatch,
      forward:
          direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_FORWARD,
    );
    if (next == null) {
      if (_searchScanActive) return;
      unawaited(_runSearch(direction));
      return;
    }
    final history = _history;
    if (history == null) return;
    final targetRow = historyRowIndexForRange(
      layout: _historyLayoutFor(history),
      range: next.match,
    );
    if (targetRow == null) {
      unawaited(_loadAdjacentSearchMatch(direction));
      return;
    }
    final wrappedChanged = _searchWrapped != next.wrapped;
    _searchMatch = next.match;
    _searchWrapped = next.wrapped;
    _selection = null;
    _syncHistoryHighlights();
    _scrollToHistoryRange(history, next.match);
    if (wrappedChanged) _notifyControls();
  }

  Future<void> _loadAdjacentSearchMatch(
    HistorySearchDirection direction,
  ) async {
    final history = _history;
    final currentMatch = _searchMatch;
    final query = _searchController.text;
    if (history == null ||
        currentMatch == null ||
        query.isEmpty ||
        _searching) {
      return;
    }
    final epoch = _searchEpoch;
    final start =
        direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD
        ? HistoryTextPosition(
            lineId: currentMatch.startLineId,
            col: currentMatch.startCol,
          )
        : HistoryTextPosition(
            lineId: currentMatch.endLineId,
            col: currentMatch.endCol,
          );
    setState(() {
      _searching = true;
      _historyError = null;
    });
    _notifyControls();
    try {
      final result = await widget.connection.searchHistory(
        history: history,
        query: query,
        direction: direction,
        mode: _searchMode,
        start: start,
      );
      if (!mounted ||
          epoch != _searchEpoch ||
          _searchController.text != query ||
          !_sameFrozenHistory(history, _history)) {
        return;
      }
      setState(() {
        _history = result.history;
        _searchMatch = result.match;
        _searchWrapped = result.wrapped;
        _searching = false;
        _selection = null;
      });
      _syncHistoryHighlights();
      if (result.match case final match?) {
        _scrollToHistoryRange(result.history, match);
      }
      _notifyControls();
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searching = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  Future<void> _runSearch(HistorySearchDirection direction) async {
    final history = _history;
    final query = _searchController.text;
    if (history == null || query.isEmpty || _searching) {
      if (query.isEmpty && mounted) {
        setState(() {
          _searchMatch = null;
          _searchMatches = const [];
          _searchMatchCount = 0;
          _searchWrapped = false;
          _searchScanActive = false;
        });
        _syncHistoryHighlights();
        _notifyControls();
      }
      return;
    }
    final epoch = ++_searchEpoch;
    final mode = _searchMode;
    final currentMatch = _searchMatch;
    final start = currentMatch == null
        ? null
        : direction == HistorySearchDirection.HISTORY_SEARCH_DIRECTION_BACKWARD
        ? HistoryTextPosition(
            lineId: currentMatch.startLineId,
            col: currentMatch.startCol,
          )
        : HistoryTextPosition(
            lineId: currentMatch.endLineId,
            col: currentMatch.endCol,
          );
    setState(() {
      _searching = true;
      _searchScanActive = false;
      _searchMatchCount = 0;
      _searchMatches = const [];
      _historyError = null;
    });
    _notifyControls();
    try {
      final result = await widget.connection.searchHistory(
        history: history,
        query: query,
        direction: direction,
        mode: mode,
        start: start,
      );
      if (!mounted ||
          epoch != _searchEpoch ||
          _searchController.text != query ||
          !_sameFrozenHistory(history, _history)) {
        return;
      }
      setState(() {
        _history = result.history;
        _searchMatch = result.match;
        _searchWrapped = result.wrapped;
        _searching = false;
        _searchScanActive = true;
        _selection = null;
      });
      _syncHistoryHighlights();
      if (result.match case final match?) {
        _scrollToHistoryRange(result.history, match);
      }
      _notifyControls();
      unawaited(
        _scanSearch(
          history: result.history,
          query: query,
          mode: mode,
          epoch: epoch,
        ),
      );
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searching = false;
        _searchScanActive = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  Future<void> _scanSearch({
    required FrozenHistory history,
    required String query,
    required HistorySearchMode mode,
    required int epoch,
  }) async {
    const maximumIndexedMatches = 512;
    final indexedMatches = <HistoryRange>[];
    var matchCount = 0;
    var publishedInitialIndex = false;
    HistoryTextPosition? start;
    try {
      while (true) {
        final batch = await widget.connection.scanHistory(
          history: history,
          query: query,
          mode: mode,
          start: start,
          maxMatches: 128,
        );
        if (!mounted ||
            epoch != _searchEpoch ||
            !_sameFrozenHistory(history, _history)) {
          return;
        }
        matchCount += batch.matches.length;
        final previousIndexedCount = indexedMatches.length;
        if (indexedMatches.length < maximumIndexedMatches) {
          indexedMatches.addAll(
            batch.matches.take(maximumIndexedMatches - indexedMatches.length),
          );
        }
        final indexChanged = indexedMatches.length != previousIndexedCount;
        final publishIndex =
            indexChanged &&
            (!publishedInitialIndex ||
                indexedMatches.length == maximumIndexedMatches ||
                batch.done);
        if (publishIndex) {
          setState(() {
            _searchMatches = List.unmodifiable(indexedMatches);
            _searchMatchCount = matchCount;
            _searchScanActive = !batch.done;
          });
          publishedInitialIndex = true;
        } else {
          _searchMatchCount = matchCount;
          _searchScanActive = !batch.done;
        }
        _notifyControls();
        if (batch.done) return;
        final next = batch.next;
        if (next == null || next.lineId == Int64.ZERO) {
          throw const NativeSessionException(
            'History search scan had no continuation',
          );
        }
        if (start != null &&
            next.lineId == start.lineId &&
            next.col == start.col) {
          throw const NativeSessionException(
            'History search scan did not advance',
          );
        }
        start = next;
      }
    } catch (error) {
      if (!mounted || epoch != _searchEpoch) return;
      setState(() {
        _searchScanActive = false;
        _historyError = error.toString();
      });
      _notifyControls();
    }
  }

  void _scrollToHistoryRange(FrozenHistory history, HistoryRange range) {
    final rowIndex = historyRowIndexForRange(
      layout: _historyLayoutFor(history),
      range: range,
    );
    if (rowIndex == null) return;
    void jumpToMatch() {
      if (!mounted || !_historyScroll.hasClients) return;
      final position = _historyScroll.position;
      if (historyRowIsVisibleInViewport(
        rowIndex: rowIndex,
        rowHeight: widget.settings.metrics.rowHeight,
        scrollOffset: position.pixels,
        viewportExtent: position.viewportDimension,
      )) {
        return;
      }
      final estimatedMax = math.max(
        position.maxScrollExtent,
        history.rows.length * widget.settings.metrics.rowHeight -
            position.viewportDimension,
      );
      final target =
          (rowIndex * widget.settings.metrics.rowHeight -
                  position.viewportDimension / 2)
              .clamp(0.0, estimatedMax);
      _historyScroll.jumpTo(target);
    }

    if (_historyScroll.hasClients) {
      jumpToMatch();
    } else {
      WidgetsBinding.instance.addPostFrameCallback((_) => jumpToMatch());
    }
  }

  Future<void> _pasteClipboard() async {
    if (_history != null || _historyLoading) return;
    try {
      final data = await Clipboard.getData(Clipboard.kTextPlain);
      final text = data?.text ?? '';
      if (!mounted || text.isEmpty) return;
      final unsafe =
          text.length > 1000 ||
          text.contains('\n') ||
          text.contains('\r') ||
          text.contains('\u001b');
      if (unsafe) {
        final accepted = await showDialog<bool>(
          context: context,
          builder: (context) => AlertDialog(
            title: const Text('Paste into terminal?'),
            content: Text(
              text,
              maxLines: 8,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text('Paste'),
              ),
            ],
          ),
        );
        if (accepted != true || !mounted) return;
      }
      _reportInput(widget.onInput(_TerminalInputOperation.paste(text, unsafe)));
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _openTerminalLink(TerminalLink link) async {
    final uri = externalTerminalLink(link.url);
    if (uri == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('This terminal link scheme is blocked')),
      );
      return;
    }
    FocusManager.instance.primaryFocus?.unfocus();
    final accepted = await showModalBottomSheet<bool>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      showDragHandle: true,
      builder: (context) => SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Open terminal link?',
                style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
              ),
              const SizedBox(height: 12),
              SelectableText(
                uri.toString(),
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
              if (link.params.trim().isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(
                  link.params.trim(),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    color: Color(0xff71717a),
                    fontSize: 11,
                  ),
                ),
              ],
              const SizedBox(height: 18),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton.icon(
                    onPressed: () => Navigator.pop(context, true),
                    icon: const Icon(Icons.open_in_new_rounded),
                    label: const Text('Open'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
    if (accepted != true || !mounted) return;
    try {
      await MethodChannelExternalUriLauncher.instance.open(uri);
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  void _handleLiveInteractionStart() {
    _cancelMomentum();
    _liveDragDistance = 0;
    widget.connection.cancelPendingMouseInput();
  }

  void _handleLiveInteractionCancel() {
    _cancelMomentum();
    _liveDragDistance = 0;
  }

  void _handleLiveTap(CanonicalLiveScreen screen, Offset position) {
    if (!widget.active ||
        _readOnly ||
        _history != null ||
        _historyLoading ||
        _deliveryState != TerminalDeliveryState.ready) {
      return;
    }
    if (_mouseTrackingEnabled(screen.modes)) {
      _dispatchModifiedInput(
        (modifiers) => widget.connection.sendPrimaryClick(
          x: position.dx,
          y: position.dy,
          modifiers: modifiers,
        ),
      );
    }
    _showKeyboard();
  }

  void _handleLiveDrag(CanonicalLiveScreen screen, DragUpdateDetails details) {
    if (_readOnly || _history != null || _historyLoading) return;
    _lastLiveDragPosition = details.localPosition;
    _liveDragDistance += details.delta.dy;
    final alternate =
        screen.alternateScreen || screen.modes?.alternateScreen == true;
    if (!alternate) {
      if (_liveDragDistance >= widget.settings.metrics.rowHeight * 2) {
        _liveDragDistance = 0;
        unawaited(_ensureHistory());
      } else if (_liveDragDistance < 0) {
        _liveDragDistance = 0;
      }
      return;
    }

    final modes = screen.modes;
    final mouseTracking = _mouseTrackingEnabled(modes);
    if (!mouseTracking && modes?.alternateScroll != true) {
      _liveDragDistance = 0;
      return;
    }

    var steps = 0;
    while (_liveDragDistance.abs() >= widget.settings.metrics.rowHeight &&
        steps < 8) {
      final up = _liveDragDistance > 0;
      _liveDragDistance += up
          ? -widget.settings.metrics.rowHeight
          : widget.settings.metrics.rowHeight;
      steps += 1;
      _sendAlternateScroll(
        up: up,
        position: details.localPosition,
        mouseTracking: mouseTracking,
      );
    }
  }

  void _handleLiveDragEnd(CanonicalLiveScreen screen, DragEndDetails details) {
    _liveDragDistance = 0;
    final alternate =
        screen.alternateScreen || screen.modes?.alternateScreen == true;
    final modes = screen.modes;
    final mouseTracking = _mouseTrackingEnabled(modes);
    if (!alternate ||
        (!mouseTracking && modes?.alternateScroll != true) ||
        MediaQuery.maybeOf(context)?.disableAnimations == true) {
      return;
    }
    final profile = resolveTerminalMomentumProfile(
      widget.settings.scrollInertia,
    );
    final velocity = details.velocity.pixelsPerSecond.dy;
    if (!profile.enabled || velocity.abs() < profile.minimumVelocity) return;
    _momentumVelocity = velocity;
    _momentumTicks = 0;
    _momentumTimer = Timer.periodic(const Duration(milliseconds: 50), (_) {
      _momentumTicks += 1;
      _momentumVelocity *= math.pow(profile.deceleration, 3).toDouble();
      if (!mounted ||
          _momentumTicks >= 120 ||
          _momentumVelocity.abs() < profile.minimumVelocity ||
          _history != null) {
        _cancelMomentum();
        return;
      }
      _liveDragDistance += _momentumVelocity * 0.05;
      if (_liveDragDistance.abs() < widget.settings.metrics.rowHeight) return;
      final up = _liveDragDistance > 0;
      _liveDragDistance += up
          ? -widget.settings.metrics.rowHeight
          : widget.settings.metrics.rowHeight;
      _sendAlternateScroll(
        up: up,
        position: _lastLiveDragPosition,
        mouseTracking: mouseTracking,
      );
    });
  }

  void _sendAlternateScroll({
    required bool up,
    required Offset position,
    required bool mouseTracking,
  }) {
    if (mouseTracking) {
      _dispatchModifiedInput(
        (modifiers) => widget.connection.sendScroll(
          up: up,
          x: position.dx,
          y: position.dy,
          modifiers: modifiers,
        ),
      );
    } else {
      _sendHid(_usbKeyboardPage | (up ? 0x52 : 0x51));
    }
  }

  void _cancelMomentum() {
    _momentumTimer?.cancel();
    _momentumTimer = null;
    _momentumVelocity = 0;
    _momentumTicks = 0;
  }

  void _resetHistoryInteraction() {
    _searchEpoch += 1;
    _selectionMode = false;
    _selection = null;
    _searchOpen = false;
    _searching = false;
    _searchScanActive = false;
    _copying = false;
    _searchMatch = null;
    _searchMatches = const [];
    _searchMatchCount = 0;
    _searchWrapped = false;
    _searchController.clear();
    _historyHighlights.clear();
  }

  void _syncHistoryHighlights() {
    final history = _history;
    _historyHighlights.update(
      selection: _selection,
      layout: history == null ? null : _historyLayoutFor(history),
      searchMatch: _searchMatch,
    );
  }

  bool _sameFrozenHistory(FrozenHistory left, FrozenHistory? right) {
    return right != null &&
        left.token == right.token &&
        left.generation == right.generation &&
        left.cols == right.cols;
  }

  void _releaseHistory(TerminalConnection connection, FrozenHistory history) {
    unawaited(connection.releaseHistory(history).catchError((_) {}));
  }

  void _handleInputFocusChange() {
    if (!_inputFocus.hasFocus) _keyboardRequestEpoch += 1;
    if (mounted) {
      setState(() {});
      _notifyControls();
    }
  }

  void _showKeyboard() {
    if (!widget.active || _keyboardFocusLocked) return;
    final requestEpoch = ++_keyboardRequestEpoch;
    if (defaultTargetPlatform == TargetPlatform.android) {
      unawaited(_requestAndroidKeyboard());
      return;
    }
    _inputFocus.requestFocus();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted ||
          requestEpoch != _keyboardRequestEpoch ||
          !widget.active ||
          _keyboardFocusLocked ||
          !_inputFocus.hasFocus) {
        return;
      }
      unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.show'));
    });
  }

  Future<void> _requestAndroidKeyboard() async {
    try {
      await AndroidTerminalInputPlatform.instance.show(_androidInputOwner);
    } catch (error, stackTrace) {
      if (kDebugMode) {
        debugPrint('AnyTTYTerminalInput platform error: $error');
        debugPrintStack(stackTrace: stackTrace);
      }
    }
  }

  void _hideKeyboard() {
    _keyboardRequestEpoch += 1;
    if (defaultTargetPlatform == TargetPlatform.android) {
      _clearAndroidComposition();
      unawaited(AndroidTerminalInputPlatform.instance.hide(_androidInputOwner));
      return;
    }
    unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.hide'));
    _inputFocus.unfocus();
  }

  void _toggleKeyboardVisibility() {
    if (_keyboardFocusLocked) {
      setState(() => _keyboardFocusLocked = false);
      _notifyControls();
      _showKeyboard();
      return;
    }
    if (widget.keyboardInset > 100) {
      _hideKeyboard();
    } else {
      _showKeyboard();
    }
  }

  void _toggleKeyboardFocusLock() {
    final locked = !_keyboardFocusLocked;
    setState(() => _keyboardFocusLocked = locked);
    _notifyControls();
    if (locked) _hideKeyboard();
  }

  void _toggleModifier(TerminalModifier modifier) {
    setState(() => _modifiers = _modifiers.cycle(modifier));
    _notifyControls();
  }

  void _toggleFn() {
    setState(() => _fnOpen = !_fnOpen);
    _notifyControls();
  }

  void _handleTextInput() {
    if (_clearingInput) return;
    final value = _inputController.value;
    final edit = decodeTerminalSoftInput(value.text);
    if (edit.isIdle) return;
    if (!edit.backspace &&
        value.composing.isValid &&
        !value.composing.isCollapsed) {
      return;
    }
    _resetSoftKeyboardBuffer();
    if (edit.backspace) {
      _sendHid(_usbKeyboardPage | 0x2a);
      return;
    }
    if (edit.text.isEmpty) return;
    _sendSoftInputText(edit.text);
  }

  void _handleAndroidTerminalInput(AndroidTerminalInputEvent event) {
    if (!mounted ||
        !widget.active ||
        _keyboardFocusLocked ||
        event.owner != _androidInputOwner) {
      return;
    }
    switch (event) {
      case AndroidTerminalCompositionInput(:final text, :final active):
        _updateAndroidComposition(active: active, text: text);
      case AndroidTerminalTextInput(:final text):
        _clearAndroidComposition();
        _sendSoftInputText(text);
      case AndroidTerminalBackspaceInput(:final count):
        for (var index = 0; index < count; index += 1) {
          _sendHid(_usbKeyboardPage | 0x2a);
        }
      case AndroidTerminalKeyInput(
        :final hidUsage,
        :final modifiers,
        :final unshiftedCodepoint,
        :final text,
      ):
        if (hidUsage case final usage?) {
          _dispatchModifiedInput(
            (combinedModifiers) => widget.onInput(
              _TerminalInputOperation.key(
                hidUsage: usage,
                modifiers: combinedModifiers,
                unshiftedCodepoint: unshiftedCodepoint,
                text: text,
              ),
            ),
            additionalModifiers: modifiers,
          );
        } else if (text.isNotEmpty) {
          _sendText(text);
        }
    }
  }

  void _updateAndroidComposition({required bool active, required String text}) {
    final next = (active: active, text: active ? text : '');
    if (_androidComposition.value == next) return;
    _androidComposition.value = next;
  }

  void _clearAndroidComposition() =>
      _updateAndroidComposition(active: false, text: '');

  void _sendSoftInputText(String text) {
    final parts = text
        .replaceAll('\r\n', '\n')
        .replaceAll('\r', '\n')
        .split('\n');
    for (var index = 0; index < parts.length; index += 1) {
      final part = parts[index];
      if (part.isNotEmpty) _sendText(part);
      if (index < parts.length - 1) {
        _sendHid(_usbKeyboardPage | 0x28);
      }
    }
  }

  void _runQuickAction(TerminalQuickAction action) {
    switch (action.kind) {
      case TerminalQuickActionKind.key:
        final key = action.key;
        if (key == null) return;
        if (key.modifier case final modifier?) {
          _toggleModifier(modifier);
          return;
        }
        if (key.text case final text?) {
          _sendText(text);
          return;
        }
        if (key.hidUsage case final usage?) {
          _sendHid(
            usage,
            unshiftedCodepoint: key.unshiftedCodepoint,
            text: key.text ?? '',
          );
        }
      case TerminalQuickActionKind.chord:
        final key = action.key;
        final usage = key?.hidUsage;
        if (key == null || usage == null) return;
        _sendHid(
          usage,
          additionalModifiers: action.modifiers,
          unshiftedCodepoint: key.unshiftedCodepoint,
          text: key.text ?? '',
        );
      case TerminalQuickActionKind.text:
        _sendSoftInputText(action.text);
        if (action.sendEnter) _sendHid(_usbKeyboardPage | 0x28);
    }
  }

  void _resetSoftKeyboardBuffer() {
    _clearingInput = true;
    _inputController.value = const TextEditingValue(
      text: terminalSoftInputSentinel,
      selection: TextSelection.collapsed(
        offset: terminalSoftInputSentinel.length,
      ),
    );
    _clearingInput = false;
  }

  KeyEventResult _handleHardwareKey(FocusNode node, KeyEvent event) {
    if (!widget.active) return KeyEventResult.ignored;
    if (event is KeyUpEvent) return KeyEventResult.ignored;
    if (_isPhysicalModifier(event.logicalKey)) return KeyEventResult.ignored;
    final keyboard = HardwareKeyboard.instance;
    var physicalModifiers = 0;
    if (keyboard.isShiftPressed) physicalModifiers |= terminalModifierShiftBit;
    if (keyboard.isControlPressed) {
      physicalModifiers |= terminalModifierControlBit;
    }
    if (keyboard.isAltPressed) physicalModifiers |= terminalModifierAltBit;
    if (keyboard.isMetaPressed) physicalModifiers |= terminalModifierSuperBit;
    final modifiers = _modifiers.bits | physicalModifiers;
    final hasCommandModifier =
        modifiers &
            (terminalModifierControlBit |
                terminalModifierAltBit |
                terminalModifierSuperBit) !=
        0;
    if (event.character != null && !hasCommandModifier) {
      return KeyEventResult.ignored;
    }
    final character = event.character ?? '';
    _dispatchModifiedInput(
      (combinedModifiers) => widget.onInput(
        _TerminalInputOperation.key(
          hidUsage: event.physicalKey.usbHidUsage,
          modifiers: combinedModifiers,
          unshiftedCodepoint: character.runes.firstOrNull ?? 0,
          text: character,
        ),
      ),
      additionalModifiers: physicalModifiers,
    );
    return KeyEventResult.handled;
  }

  void _sendHid(
    int usage, {
    int additionalModifiers = 0,
    int unshiftedCodepoint = 0,
    String text = '',
  }) {
    _dispatchModifiedInput(
      (modifiers) => widget.onInput(
        _TerminalInputOperation.key(
          hidUsage: usage,
          modifiers: modifiers,
          unshiftedCodepoint: unshiftedCodepoint,
          text: text,
        ),
      ),
      additionalModifiers: additionalModifiers,
    );
  }

  void _sendEnter() {
    _resetSoftKeyboardBuffer();
    _sendHid(_usbKeyboardPage | 0x28);
  }

  void _sendText(String text) {
    _dispatchModifiedInput(
      (modifiers) =>
          widget.onInput(_TerminalInputOperation.text(text, modifiers)),
    );
  }

  void _dispatchModifiedInput(
    Future<void> Function(int modifiers) send, {
    int additionalModifiers = 0,
  }) {
    if (_readOnly) return;
    final sent = _modifiers;
    final expectedConnection = widget.connection;
    if (sent.hasOnce) {
      setState(() => _modifiers = sent.consumeOnce());
      _notifyControls();
    }
    try {
      _reportInput(
        send(sent.bits | additionalModifiers),
        rejectedModifiers: sent,
        expectedConnection: expectedConnection,
      );
    } catch (error) {
      if (mounted && widget.connection == expectedConnection) {
        setState(() => _modifiers = _modifiers.restoreRejected(sent));
        _notifyControls();
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.toString())));
      }
    }
  }

  void _reportInput(
    Future<void> pending, {
    TerminalModifierState? rejectedModifiers,
    TerminalConnection? expectedConnection,
  }) {
    unawaited(
      pending.catchError((Object error) {
        if (!mounted) return;
        if (rejectedModifiers != null &&
            widget.connection == expectedConnection) {
          setState(() {
            _modifiers = _modifiers.restoreRejected(rejectedModifiers);
          });
          _notifyControls();
        }
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.toString())));
      }),
    );
  }
}

bool _isPhysicalModifier(LogicalKeyboardKey key) {
  return key == LogicalKeyboardKey.shift ||
      key == LogicalKeyboardKey.shiftLeft ||
      key == LogicalKeyboardKey.shiftRight ||
      key == LogicalKeyboardKey.control ||
      key == LogicalKeyboardKey.controlLeft ||
      key == LogicalKeyboardKey.controlRight ||
      key == LogicalKeyboardKey.alt ||
      key == LogicalKeyboardKey.altLeft ||
      key == LogicalKeyboardKey.altRight ||
      key == LogicalKeyboardKey.meta ||
      key == LogicalKeyboardKey.metaLeft ||
      key == LogicalKeyboardKey.metaRight;
}

final class _HistoryContextBar extends StatelessWidget {
  const _HistoryContextBar({
    required this.liveEnabled,
    required this.searchActive,
    required this.selectionActive,
    required this.status,
    required this.onLive,
    required this.onSearch,
    required this.onSelection,
  });

  final bool liveEnabled;
  final bool searchActive;
  final bool selectionActive;
  final String status;
  final VoidCallback onLive;
  final VoidCallback onSearch;
  final VoidCallback onSelection;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 32,
      child: Row(
        children: [
          Material(
            color: palette.surface.withValues(alpha: 0.94),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(4),
              side: BorderSide(color: palette.border),
            ),
            clipBehavior: Clip.antiAlias,
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                _HistoryContextAction(
                  icon: LucideIcons.radio,
                  label: 'Live',
                  enabled: liveEnabled,
                  onPressed: onLive,
                ),
                SizedBox(
                  height: 18,
                  child: VerticalDivider(width: 1, color: palette.border),
                ),
                _HistoryContextAction(
                  icon: LucideIcons.search,
                  tooltip: searchActive ? 'Close search' : 'Search history',
                  active: searchActive,
                  onPressed: onSearch,
                ),
                _HistoryContextAction(
                  icon: LucideIcons.scanText,
                  tooltip: selectionActive
                      ? 'Selection active'
                      : 'Select history',
                  active: selectionActive,
                  onPressed: onSelection,
                ),
              ],
            ),
          ),
          const Spacer(),
          Container(
            constraints: const BoxConstraints(minHeight: 28),
            padding: const EdgeInsets.symmetric(horizontal: 8),
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: palette.surface.withValues(alpha: 0.94),
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(4),
            ),
            child: Text(
              status,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.muted,
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 9,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _HistoryContextAction extends StatelessWidget {
  const _HistoryContextAction({
    required this.icon,
    required this.onPressed,
    this.label,
    this.tooltip,
    this.enabled = true,
    this.active = false,
  });

  final IconData icon;
  final VoidCallback onPressed;
  final String? label;
  final String? tooltip;
  final bool enabled;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final foreground = !enabled
        ? palette.faint
        : active
        ? palette.accent
        : palette.text;
    final button = InkWell(
      onTap: enabled ? onPressed : null,
      child: Container(
        height: 30,
        padding: EdgeInsets.symmetric(horizontal: label == null ? 8 : 9),
        color: active
            ? Color.lerp(palette.surface, palette.accent, 0.16)
            : Colors.transparent,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 14, color: foreground),
            if (label case final text?) ...[
              const SizedBox(width: 5),
              Text(
                text,
                style: TextStyle(
                  color: foreground,
                  fontSize: 9,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ],
          ],
        ),
      ),
    );
    final message = tooltip ?? label;
    return message == null ? button : Tooltip(message: message, child: button);
  }
}

@visibleForTesting
const terminalHistorySearchInputDecoration = InputDecoration(
  isDense: true,
  filled: false,
  border: InputBorder.none,
  hintText: 'Search history',
  hintStyle: TextStyle(color: Color(0xff71717a)),
);

final class _HistorySearchBar extends StatelessWidget {
  const _HistorySearchBar({
    required this.controller,
    required this.focusNode,
    required this.mode,
    required this.searching,
    required this.scanning,
    required this.matchCount,
    required this.wrapped,
    required this.onQueryChanged,
    required this.onModeChanged,
    required this.onPrevious,
    required this.onNext,
    required this.onClose,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final HistorySearchMode mode;
  final bool searching;
  final bool scanning;
  final int matchCount;
  final bool wrapped;
  final ValueChanged<String> onQueryChanged;
  final ValueChanged<HistorySearchMode> onModeChanged;
  final VoidCallback onPrevious;
  final VoidCallback onNext;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = searching && matchCount == 0
        ? 'Searching'
        : scanning
        ? '$matchCount+ matches'
        : matchCount == 0
        ? 'No matches'
        : '$matchCount matches'
              '${wrapped ? ' / wrapped' : ''}';
    return Container(
      height: 44,
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.borderStrong),
        borderRadius: const BorderRadius.all(Radius.circular(8)),
        boxShadow: const [
          BoxShadow(
            color: Color(0x50000000),
            blurRadius: 8,
            offset: Offset(0, -2),
          ),
        ],
      ),
      padding: const EdgeInsets.only(left: 8, right: 2),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: controller,
              focusNode: focusNode,
              autofocus: true,
              textInputAction: TextInputAction.search,
              onChanged: onQueryChanged,
              onSubmitted: (_) => onNext(),
              style: TextStyle(
                color: palette.text,
                fontFamily: 'monospace',
                fontSize: 11,
              ),
              decoration: terminalHistorySearchInputDecoration.copyWith(
                hintStyle: TextStyle(color: palette.faint),
                suffix: Text(
                  status,
                  style: TextStyle(color: palette.muted, fontSize: 8),
                ),
              ),
            ),
          ),
          SizedBox.square(
            dimension: 38,
            child: PopupMenuButton<HistorySearchMode>(
              tooltip: 'Search mode: ${_searchModeLabel(mode)}',
              initialValue: mode,
              padding: EdgeInsets.zero,
              onSelected: onModeChanged,
              icon: Icon(
                LucideIcons.slidersHorizontal,
                color: palette.text,
                size: 15,
              ),
              itemBuilder: (context) => [
                for (final value in const [
                  HistorySearchMode.HISTORY_SEARCH_MODE_TEXT,
                  HistorySearchMode.HISTORY_SEARCH_MODE_GLOB,
                  HistorySearchMode.HISTORY_SEARCH_MODE_REGEX,
                ])
                  PopupMenuItem(
                    value: value,
                    child: Text(_searchModeLabel(value)),
                  ),
              ],
            ),
          ),
          _SearchBarIconButton(
            tooltip: 'Previous match',
            onPressed: searching ? null : onPrevious,
            icon: LucideIcons.chevronUp,
          ),
          _SearchBarIconButton(
            tooltip: 'Next match',
            onPressed: searching ? null : onNext,
            icon: LucideIcons.chevronDown,
          ),
          _SearchBarIconButton(
            tooltip: 'Close search',
            onPressed: onClose,
            icon: LucideIcons.x,
          ),
        ],
      ),
    );
  }
}

final class _SearchBarIconButton extends StatelessWidget {
  const _SearchBarIconButton({
    required this.tooltip,
    required this.onPressed,
    required this.icon,
  });

  final String tooltip;
  final VoidCallback? onPressed;
  final IconData icon;

  @override
  Widget build(BuildContext context) => IconButton(
    tooltip: tooltip,
    onPressed: onPressed,
    constraints: const BoxConstraints.tightFor(width: 38, height: 38),
    padding: const EdgeInsets.all(11),
    icon: Icon(icon, size: 15),
    color: AnyttyPalette.of(context).text,
  );
}

String _searchModeLabel(HistorySearchMode mode) {
  return switch (mode) {
    HistorySearchMode.HISTORY_SEARCH_MODE_GLOB => 'Glob',
    HistorySearchMode.HISTORY_SEARCH_MODE_REGEX => 'Regex',
    _ => 'Text',
  };
}

final class _WaitingForSnapshot extends StatelessWidget {
  const _WaitingForSnapshot({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox.square(
              dimension: 18,
              child: CircularProgressIndicator(
                color: palette.accent,
                strokeWidth: 2,
              ),
            ),
            const SizedBox(height: 14),
            Text(
              label,
              style: TextStyle(
                color: palette.text,
                fontFamily: 'monospace',
                fontSize: 13,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _TerminalDeliveryBanner extends StatelessWidget {
  const _TerminalDeliveryBanner({
    required this.message,
    required this.error,
    this.onRetry,
  });

  final String message;
  final VoidCallback? onRetry;
  final bool error;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final background = error ? palette.danger : palette.warning;
    final foreground =
        ThemeData.estimateBrightnessForColor(background) == Brightness.dark
        ? Colors.white
        : Colors.black;
    return Positioned(
      top: 8,
      left: 8,
      right: 8,
      child: Semantics(
        container: true,
        liveRegion: true,
        child: Material(
          color: background,
          borderRadius: BorderRadius.circular(4),
          child: Row(
            children: [
              Padding(
                padding: const EdgeInsets.only(left: 12),
                child: Icon(
                  error ? Icons.error_outline_rounded : Icons.sync_rounded,
                  size: 18,
                  color: foreground,
                ),
              ),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 9,
                  ),
                  child: Text(
                    message,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: foreground, fontSize: 12),
                  ),
                ),
              ),
              if (onRetry case final retry?)
                IconButton(
                  tooltip: 'Reconnect terminal',
                  color: foreground,
                  onPressed: retry,
                  icon: const Icon(Icons.refresh_rounded),
                )
              else
                const SizedBox(width: 12),
            ],
          ),
        ),
      ),
    );
  }
}

final class _HistoryLoadingStatus extends StatelessWidget {
  const _HistoryLoadingStatus({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Positioned(
      right: 8,
      bottom: 8,
      child: Semantics(
        container: true,
        liveRegion: true,
        label: label,
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            borderRadius: BorderRadius.circular(4),
          ),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox.square(
                  dimension: 13,
                  child: CircularProgressIndicator(
                    color: palette.accent,
                    strokeWidth: 1.5,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: TextStyle(
                    color: palette.text,
                    fontFamily: 'monospace',
                    fontSize: 11,
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

final class _TerminalFailure extends StatelessWidget {
  const _TerminalFailure({
    required this.message,
    required this.onRetry,
    this.dark = false,
  });

  final String message;
  final VoidCallback onRetry;
  final bool dark;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: Padding(
          padding: const EdgeInsets.all(28),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.error_outline_rounded,
                color: palette.danger,
                size: 34,
              ),
              const SizedBox(height: 12),
              Text(
                message,
                maxLines: 4,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: TextStyle(color: palette.text, fontSize: 13),
              ),
              const SizedBox(height: 16),
              OutlinedButton.icon(
                onPressed: onRetry,
                icon: const Icon(Icons.refresh_rounded),
                label: const Text('Retry'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _TerminalCompositionOverlay extends StatelessWidget {
  const _TerminalCompositionOverlay({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        final maximumHeight = math.min(
          constraints.maxHeight,
          math.max(40.0, constraints.maxHeight * 0.4),
        );
        return Align(
          alignment: Alignment.bottomCenter,
          child: ConstrainedBox(
            constraints: BoxConstraints(maxHeight: maximumHeight),
            child: SizedBox(
              width: constraints.maxWidth,
              child: Semantics(
                container: true,
                liveRegion: true,
                label: text.isEmpty ? 'Composing input' : text,
                excludeSemantics: true,
                child: DecoratedBox(
                  decoration: BoxDecoration(
                    color: palette.surface,
                    border: Border(top: BorderSide(color: palette.border)),
                  ),
                  child: SingleChildScrollView(
                    reverse: true,
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 6,
                    ),
                    child: text.isEmpty
                        ? Align(
                            alignment: Alignment.centerLeft,
                            child: SizedBox.square(
                              dimension: 13,
                              child: CircularProgressIndicator(
                                strokeWidth: 1.5,
                                color: palette.accent,
                              ),
                            ),
                          )
                        : Text(
                            text,
                            style: TextStyle(
                              color: palette.text,
                              fontFamily: 'JetBrainsMonoNerd',
                              fontSize: 12,
                              height: 1.6,
                            ),
                          ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );
  }
}

final class _TerminalSelectionToolbar extends StatelessWidget {
  const _TerminalSelectionToolbar({
    required this.selectionAvailable,
    required this.copyPending,
    required this.onSelectAll,
    required this.onSelectVisible,
    required this.onCopy,
    required this.onClose,
  });

  final bool selectionAvailable;
  final bool copyPending;
  final VoidCallback onSelectAll;
  final VoidCallback onSelectVisible;
  final VoidCallback onCopy;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Material(
      color: palette.surface,
      elevation: 8,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
        side: BorderSide(color: palette.borderStrong),
      ),
      child: SizedBox(
        height: 48,
        child: Row(
          children: [
            TextButton(onPressed: onSelectAll, child: const Text('ALL')),
            TextButton(
              onPressed: onSelectVisible,
              child: const Text('VISIBLE'),
            ),
            const SizedBox(height: 20, child: VerticalDivider()),
            IconButton(
              tooltip: 'Copy selection',
              onPressed: selectionAvailable && !copyPending ? onCopy : null,
              icon: const Icon(Icons.copy_rounded, size: 18),
            ),
            const Spacer(),
            IconButton(
              tooltip: 'Cancel selection',
              onPressed: copyPending ? null : onClose,
              icon: const Icon(Icons.close_rounded, size: 18),
            ),
          ],
        ),
      ),
    );
  }
}

final class _TerminalToolsSheet extends StatelessWidget {
  const _TerminalToolsSheet({
    required this.terminalState,
    required this.terminalSize,
    required this.resizeControl,
    required this.inputEnabled,
    required this.historyActive,
    required this.historyActionEnabled,
    required this.keyboardMode,
    required this.searchActive,
    required this.resizeAvailable,
    required this.splitOpen,
    required this.syncInput,
    required this.canSplit,
    required this.onSplit,
    required this.onToggleSync,
    required this.onCloseSplit,
    required this.onKeyboardModeChanged,
    required this.onClose,
    required this.onSelected,
  });

  final TerminalState terminalState;
  final TerminalSize terminalSize;
  final ResizeControl resizeControl;
  final bool inputEnabled;
  final bool historyActive;
  final bool historyActionEnabled;
  final TerminalKeyboardMode keyboardMode;
  final bool searchActive;
  final bool resizeAvailable;
  final bool splitOpen;
  final bool syncInput;
  final bool canSplit;
  final ValueChanged<_TerminalSplitTarget> onSplit;
  final VoidCallback onToggleSync;
  final VoidCallback onCloseSplit;
  final ValueChanged<TerminalKeyboardMode> onKeyboardModeChanged;
  final VoidCallback onClose;
  final ValueChanged<_TerminalToolAction> onSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return TweenAnimationBuilder<double>(
      duration: AnyttyMotion.resolve(
        context,
        const Duration(milliseconds: 150),
      ),
      curve: Curves.easeOutCubic,
      tween: Tween(begin: 0, end: 1),
      builder: (context, value, child) => ClipRect(
        child: Opacity(
          opacity: value,
          child: FractionalTranslation(
            translation: Offset(0, 1 - value),
            child: child,
          ),
        ),
      ),
      child: Material(
        color: palette.surface,
        elevation: 12,
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(6),
          side: BorderSide(color: palette.borderStrong),
        ),
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: math.min(
              440,
              math.max(220, MediaQuery.sizeOf(context).height - 160),
            ),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              SizedBox(
                height: 42,
                child: Row(
                  children: [
                    const SizedBox(width: 10),
                    Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: _terminalStateColor(context, terminalState),
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 7),
                    Text(
                      'Terminal',
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 12,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        '${_terminalStateLabel(terminalState)}  ${terminalSize.cols}x${terminalSize.rows}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: palette.muted,
                          fontFamily: 'JetBrainsMonoNerd',
                          fontSize: 9,
                        ),
                      ),
                    ),
                    Text(
                      resizeControl.canResize ? 'OWNER' : 'FOLLOWER',
                      style: TextStyle(
                        color: palette.muted,
                        fontSize: 8,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    IconButton(
                      tooltip: 'Close terminal tools',
                      onPressed: onClose,
                      constraints: const BoxConstraints.tightFor(
                        width: 40,
                        height: 40,
                      ),
                      icon: const Icon(LucideIcons.x, size: 16),
                    ),
                  ],
                ),
              ),
              Container(
                decoration: BoxDecoration(
                  color: palette.surfaceRaised,
                  border: Border.symmetric(
                    horizontal: BorderSide(color: palette.border),
                  ),
                ),
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                child: Row(
                  children: [
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.clipboardPaste,
                        label: 'Paste',
                        enabled: inputEnabled,
                        onPressed: () => onSelected(_TerminalToolAction.paste),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.history,
                        label: historyActive ? 'Live' : 'History',
                        enabled: historyActionEnabled,
                        active: historyActive,
                        onPressed: () =>
                            onSelected(_TerminalToolAction.history),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.search,
                        label: 'Search',
                        active: searchActive,
                        onPressed: () => onSelected(_TerminalToolAction.search),
                      ),
                    ),
                    Expanded(
                      child: _CompactToolAction(
                        icon: LucideIcons.scanText,
                        label: 'Select',
                        onPressed: () =>
                            onSelected(_TerminalToolAction.selection),
                      ),
                    ),
                  ],
                ),
              ),
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.fromLTRB(8, 8, 8, 10),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const _ToolSectionLabel('TERMINAL'),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.slidersHorizontal,
                              label: 'Shortcut bar',
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.commandBar),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.maximize2,
                              label: 'Resize',
                              enabled: resizeAvailable,
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.resize),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.settings,
                              label: 'Settings',
                              onPressed: () =>
                                  onSelected(_TerminalToolAction.settings),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 9),
                      const _ToolSectionLabel('SPLIT'),
                      const SizedBox(height: 4),
                      Row(
                        children: [
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowLeftToLine,
                              label: 'Left',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.left),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowRightToLine,
                              label: 'Right',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.right),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowUpToLine,
                              label: 'Above',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.above),
                            ),
                          ),
                          Expanded(
                            child: _CompactToolAction(
                              icon: LucideIcons.arrowDownToLine,
                              label: 'Below',
                              enabled: canSplit,
                              onPressed: () =>
                                  onSplit(_TerminalSplitTarget.below),
                            ),
                          ),
                          if (splitOpen)
                            Expanded(
                              child: _CompactToolAction(
                                icon: syncInput
                                    ? LucideIcons.link2
                                    : LucideIcons.unlink2,
                                label: 'Sync',
                                active: syncInput,
                                onPressed: onToggleSync,
                              ),
                            ),
                          if (splitOpen)
                            Expanded(
                              child: _CompactToolAction(
                                icon: LucideIcons.x,
                                label: 'Close',
                                onPressed: onCloseSplit,
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(height: 9),
                      const _ToolSectionLabel('KEYBOARD LAYOUT'),
                      const SizedBox(height: 4),
                      _TerminalKeyboardModeControl(
                        value: keyboardMode,
                        onChanged: onKeyboardModeChanged,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _TerminalKeyboardModeControl extends StatelessWidget {
  const _TerminalKeyboardModeControl({
    required this.value,
    required this.onChanged,
  });

  final TerminalKeyboardMode value;
  final ValueChanged<TerminalKeyboardMode> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    const options = <({TerminalKeyboardMode mode, String label})>[
      (mode: TerminalKeyboardMode.automatic, label: 'Auto'),
      (mode: TerminalKeyboardMode.resize, label: 'Resize'),
      (mode: TerminalKeyboardMode.shift, label: 'Shift'),
    ];
    return Semantics(
      label: 'Keyboard',
      container: true,
      child: Container(
        height: 38,
        padding: const EdgeInsets.all(2),
        decoration: BoxDecoration(
          color: palette.surfaceRaised,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(4),
        ),
        child: Row(
          children: [
            for (final option in options)
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 1),
                  child: InkWell(
                    onTap: () {
                      unawaited(HapticFeedback.selectionClick());
                      onChanged(option.mode);
                    },
                    borderRadius: BorderRadius.circular(3),
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        color: value == option.mode
                            ? Color.lerp(palette.surface, palette.accent, 0.18)
                            : Colors.transparent,
                        borderRadius: BorderRadius.circular(3),
                      ),
                      child: Center(
                        child: Text(
                          option.label,
                          style: TextStyle(
                            color: value == option.mode
                                ? palette.accent
                                : palette.muted,
                            fontSize: 10,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

final class _CompactToolAction extends StatelessWidget {
  const _CompactToolAction({
    required this.icon,
    required this.label,
    required this.onPressed,
    this.enabled = true,
    this.active = false,
  });

  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  final bool enabled;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final foreground = enabled
        ? active
              ? palette.accent
              : palette.text
        : palette.faint;
    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: label,
      child: SizedBox(
        height: 48,
        child: InkWell(
          onTap: enabled
              ? () {
                  unawaited(HapticFeedback.selectionClick());
                  onPressed();
                }
              : null,
          borderRadius: BorderRadius.circular(4),
          child: AnimatedContainer(
            duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
            margin: const EdgeInsets.symmetric(horizontal: 1),
            decoration: BoxDecoration(
              color: active
                  ? Color.lerp(palette.surface, palette.accent, 0.16)
                  : Colors.transparent,
              border: active
                  ? Border.all(
                      color: Color.lerp(palette.border, palette.accent, 0.48)!,
                    )
                  : null,
              borderRadius: BorderRadius.circular(4),
            ),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, size: 15, color: foreground),
                const SizedBox(height: 3),
                Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: foreground,
                    fontSize: 8.5,
                    fontWeight: FontWeight.w600,
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

final class _ToolSectionLabel extends StatelessWidget {
  const _ToolSectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) => Text(
    label,
    style: TextStyle(
      color: AnyttyPalette.of(context).faint,
      fontSize: 8,
      fontWeight: FontWeight.w700,
    ),
  );
}

final class _TerminalSettingsSheet extends StatefulWidget {
  const _TerminalSettingsSheet({
    required this.initialSettings,
    required this.onChanged,
  });

  final TerminalSettings initialSettings;
  final Future<void> Function(TerminalSettings) onChanged;

  @override
  State<_TerminalSettingsSheet> createState() => _TerminalSettingsSheetState();
}

final class _TerminalSettingsSheetState extends State<_TerminalSettingsSheet> {
  late TerminalSettings _settings;

  @override
  void initState() {
    super.initState();
    _settings = widget.initialSettings.normalized();
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 0, 8, 8),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    'Terminal settings',
                    style: TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
                  ),
                ),
                IconButton(
                  tooltip: 'Close terminal settings',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close_rounded),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.only(bottom: 24),
              children: [
                const _SettingsBandTitle(label: 'Appearance'),
                Padding(
                  padding: const EdgeInsets.fromLTRB(12, 12, 12, 6),
                  child: _TerminalSettingsPreview(settings: _settings),
                ),
                _SettingsRow(
                  label: 'Font size',
                  child: _FontSizeStepper(
                    value: _settings.fontSize,
                    onChanged: (value) =>
                        _commit(_settings.copyWith(fontSize: value)),
                  ),
                ),
                _SettingsRow(
                  label: 'Font',
                  child: DropdownButton<String>(
                    value: _settings.fontFamily,
                    underline: const SizedBox.shrink(),
                    borderRadius: BorderRadius.circular(6),
                    onChanged: (value) {
                      if (value != null) {
                        unawaited(
                          _commit(_settings.copyWith(fontFamily: value)),
                        );
                      }
                    },
                    items: [
                      for (final family in terminalFontFamilies)
                        DropdownMenuItem(
                          value: family,
                          child: Text(
                            terminalFontLabel(family),
                            style: TextStyle(
                              fontFamily: family,
                              fontFamilyFallback: terminalFontFamilyFallback,
                              fontSize: 13,
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
                SwitchListTile.adaptive(
                  title: const Text('Cursor blink'),
                  value: _settings.cursorBlink,
                  onChanged: (value) =>
                      _commit(_settings.copyWith(cursorBlink: value)),
                ),
                SwitchListTile.adaptive(
                  title: const Text('Automatically own terminal size'),
                  value: _settings.autoAcquireResizeOwner,
                  onChanged: (value) => _commit(
                    _settings.copyWith(autoAcquireResizeOwner: value),
                  ),
                ),
                const _SettingsBandTitle(label: 'Touch and history'),
                _SettingsSlider(
                  label: 'Scroll inertia',
                  value: _settings.scrollInertia,
                  maximum: 100,
                  onChanged: (value) => setState(
                    () => _settings = _settings.copyWith(scrollInertia: value),
                  ),
                  onChangeEnd: (value) =>
                      _commit(_settings.copyWith(scrollInertia: value)),
                ),
                _SettingsSlider(
                  label: 'API page prefetch distance',
                  value: _settings.historyPrefetchThresholdRows,
                  maximum: 200,
                  onChanged: (value) => setState(
                    () => _settings = _settings.copyWith(
                      historyPrefetchThresholdRows: value,
                    ),
                  ),
                  onChangeEnd: (value) => _commit(
                    _settings.copyWith(historyPrefetchThresholdRows: value),
                  ),
                ),
                const _SettingsBandTitle(label: 'Terminal theme'),
                Padding(
                  padding: const EdgeInsets.all(12),
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      final columns = constraints.maxWidth >= 560 ? 3 : 2;
                      return GridView.builder(
                        shrinkWrap: true,
                        physics: const NeverScrollableScrollPhysics(),
                        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                          crossAxisCount: columns,
                          mainAxisExtent: 56,
                          crossAxisSpacing: 8,
                          mainAxisSpacing: 8,
                        ),
                        itemCount: terminalThemes.length,
                        itemBuilder: (context, index) {
                          final theme = terminalThemes[index];
                          final selected = theme.id == _settings.themeId;
                          void selectTheme() => unawaited(
                            _commit(_settings.copyWith(themeId: theme.id)),
                          );

                          return Semantics(
                            button: true,
                            selected: selected,
                            label: '${theme.label} theme palette preview',
                            onTap: selectTheme,
                            child: ExcludeSemantics(
                              child: OutlinedButton(
                                onPressed: selectTheme,
                                style: OutlinedButton.styleFrom(
                                  alignment: Alignment.centerLeft,
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 10,
                                  ),
                                  side: BorderSide(
                                    color: selected
                                        ? Theme.of(context).colorScheme.primary
                                        : Theme.of(context).dividerColor,
                                    width: selected ? 2 : 1,
                                  ),
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(4),
                                  ),
                                ),
                                child: Row(
                                  children: [
                                    _TerminalThemeSwatch(theme: theme),
                                    const SizedBox(width: 8),
                                    Expanded(
                                      child: Text(
                                        theme.label,
                                        maxLines: 2,
                                        overflow: TextOverflow.ellipsis,
                                        style: const TextStyle(fontSize: 11),
                                      ),
                                    ),
                                    if (selected)
                                      const Icon(Icons.check_rounded, size: 16),
                                  ],
                                ),
                              ),
                            ),
                          );
                        },
                      );
                    },
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _commit(TerminalSettings next) async {
    final previous = _settings;
    setState(() => _settings = next);
    try {
      await widget.onChanged(next);
    } catch (error) {
      if (!mounted) return;
      setState(() => _settings = previous);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not save terminal settings: $error')),
      );
    }
  }
}

final class _TerminalSettingsPreview extends StatelessWidget {
  const _TerminalSettingsPreview({required this.settings});

  final TerminalSettings settings;

  @override
  Widget build(BuildContext context) {
    final theme = settings.theme;
    final background = _terminalThemeColor(theme.background);
    final foreground = _terminalThemeColor(theme.foreground);
    final muted = _terminalThemeColor(theme.ansi[8]);
    final accent = _terminalThemeColor(theme.ansi[2]);
    final prompt = _terminalThemeColor(theme.ansi[3]);
    final fontSize = settings.fontSize.toDouble().clamp(10, 20).toDouble();
    final lineStyle = TextStyle(
      color: foreground,
      fontFamily: settings.fontFamily,
      fontFamilyFallback: terminalFontFamilyFallback,
      fontSize: fontSize,
      height: 1.25,
    );
    return Semantics(
      readOnly: true,
      label:
          'Terminal settings preview, ${theme.label}, '
          '${terminalFontLabel(settings.fontFamily)}, '
          '${settings.fontSize} point',
      child: ExcludeSemantics(
        child: Container(
          height: 104,
          clipBehavior: Clip.antiAlias,
          decoration: BoxDecoration(
            color: background,
            border: Border.all(color: muted.withValues(alpha: 0.5)),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 6,
                ),
                child: Row(
                  children: [
                    Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: accent,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 7),
                    Expanded(
                      child: Text(
                        '${theme.label}  ·  ${terminalFontLabel(settings.fontFamily)}',
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          color: muted,
                          fontFamily: settings.fontFamily,
                          fontFamilyFallback: terminalFontFamilyFallback,
                          fontSize: 10,
                        ),
                      ),
                    ),
                    Text(
                      '${settings.fontSize} pt',
                      style: TextStyle(
                        color: muted,
                        fontFamily: settings.fontFamily,
                        fontFamilyFallback: terminalFontFamilyFallback,
                        fontSize: 10,
                      ),
                    ),
                  ],
                ),
              ),
              Divider(height: 1, color: muted.withValues(alpha: 0.35)),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(10, 7, 10, 6),
                  child: RichText(
                    maxLines: 2,
                    overflow: TextOverflow.clip,
                    text: TextSpan(
                      style: lineStyle,
                      children: [
                        TextSpan(
                          text: 'anytty',
                          style: lineStyle.copyWith(color: accent),
                        ),
                        const TextSpan(text: ' in '),
                        TextSpan(
                          text: '~/workspace\n',
                          style: lineStyle.copyWith(
                            color: _terminalThemeColor(theme.ansi[6]),
                          ),
                        ),
                        TextSpan(
                          text: r'$ ',
                          style: lineStyle.copyWith(color: prompt),
                        ),
                        const TextSpan(text: 'git status --short'),
                        if (settings.cursorBlink)
                          TextSpan(
                            text: ' ',
                            style: lineStyle.copyWith(
                              backgroundColor: _terminalThemeColor(
                                theme.cursor,
                              ),
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _SettingsBandTitle extends StatelessWidget {
  const _SettingsBandTitle({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) => Container(
    color: Theme.of(context).colorScheme.surfaceContainerLow,
    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
    child: Text(
      label.toUpperCase(),
      style: TextStyle(
        color: Theme.of(context).colorScheme.onSurfaceVariant,
        fontSize: 11,
        fontWeight: FontWeight.w600,
      ),
    ),
  );
}

final class _SettingsRow extends StatelessWidget {
  const _SettingsRow({required this.label, required this.child});

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) => Container(
    constraints: const BoxConstraints(minHeight: 56),
    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 6),
    decoration: BoxDecoration(
      border: Border(bottom: BorderSide(color: Theme.of(context).dividerColor)),
    ),
    child: Row(
      children: [
        Expanded(child: Text(label)),
        const SizedBox(width: 12),
        child,
      ],
    ),
  );
}

final class _FontSizeStepper extends StatelessWidget {
  const _FontSizeStepper({required this.value, required this.onChanged});

  final int value;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      IconButton(
        tooltip: 'Decrease font size',
        onPressed: value > 8 ? () => onChanged(value - 1) : null,
        icon: const Icon(Icons.remove_rounded),
      ),
      SizedBox(
        width: 42,
        child: Text(
          '$value',
          textAlign: TextAlign.center,
          style: const TextStyle(fontFamily: 'JetBrainsMonoNerd', fontSize: 12),
        ),
      ),
      IconButton(
        tooltip: 'Increase font size',
        onPressed: value < 32 ? () => onChanged(value + 1) : null,
        icon: const Icon(Icons.add_rounded),
      ),
    ],
  );
}

final class _SettingsSlider extends StatelessWidget {
  const _SettingsSlider({
    required this.label,
    required this.value,
    required this.maximum,
    required this.onChanged,
    required this.onChangeEnd,
  });

  final String label;
  final int value;
  final int maximum;
  final ValueChanged<int> onChanged;
  final ValueChanged<int> onChangeEnd;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(20, 10, 12, 8),
    child: Column(
      children: [
        Row(
          children: [
            Expanded(child: Text(label)),
            Text(
              '$value',
              style: const TextStyle(
                fontFamily: 'JetBrainsMonoNerd',
                fontSize: 12,
              ),
            ),
          ],
        ),
        Slider(
          value: value.toDouble(),
          max: maximum.toDouble(),
          divisions: maximum,
          onChanged: (value) => onChanged(value.round()),
          onChangeEnd: (value) => onChangeEnd(value.round()),
        ),
      ],
    ),
  );
}

final class _TerminalThemeSwatch extends StatelessWidget {
  const _TerminalThemeSwatch({required this.theme});

  final TerminalTheme theme;

  @override
  Widget build(BuildContext context) {
    final colors = [
      theme.background,
      theme.foreground,
      theme.ansi[1],
      theme.ansi[2],
      theme.ansi[4],
    ];
    return Container(
      width: 32,
      height: 32,
      padding: const EdgeInsets.all(1),
      clipBehavior: Clip.antiAlias,
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Row(
        children: [
          for (final color in colors)
            SizedBox(
              width: 6,
              height: 30,
              child: ColoredBox(color: _terminalThemeColor(color)),
            ),
        ],
      ),
    );
  }
}

Color _terminalThemeColor(int rgb) => Color(0xff000000 | rgb);

final class _ResizeControlSheet extends StatefulWidget {
  const _ResizeControlSheet({
    required this.connection,
    required this.initialControl,
    required this.viewportCells,
  });

  final TerminalConnection connection;
  final ResizeControl initialControl;
  final ({int cols, int rows})? viewportCells;

  @override
  State<_ResizeControlSheet> createState() => _ResizeControlSheetState();
}

final class _ResizeControlSheetState extends State<_ResizeControlSheet> {
  late ResizeControl _control;
  StreamSubscription<ResizeControl>? _subscription;
  bool _busy = false;
  String? _error;

  bool get _ownsResize =>
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_OWNER ||
      _control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_SIZE_LOCKED;

  @override
  void initState() {
    super.initState();
    _control = widget.initialControl.deepCopy();
    _subscription = widget.connection.watchResizeControl().listen((control) {
      if (mounted) setState(() => _control = control);
    });
  }

  @override
  void dispose() {
    unawaited(_subscription?.cancel());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final size = _control.hasOwnership() && _control.ownership.hasSize()
        ? _control.ownership.size
        : widget.connection.terminalSize;
    final owner = _control.ownerSurfaceId.isNotEmpty
        ? _control.ownerSurfaceId
        : _control.hasOwnership()
        ? _control.ownership.ownerSurfaceId
        : '';
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'Terminal size',
              style: TextStyle(
                color: palette.text,
                fontSize: 17,
                fontWeight: FontWeight.w600,
              ),
            ),
            const SizedBox(height: 14),
            _ResizeStatusRow(
              icon: _control.sizeLocked
                  ? Icons.lock_outline_rounded
                  : _ownsResize
                  ? Icons.aspect_ratio_rounded
                  : Icons.visibility_outlined,
              label: _resizeControlStatus(_control),
              value: '${size.cols} x ${size.rows}',
            ),
            if (owner.isNotEmpty) ...[
              const SizedBox(height: 8),
              _ResizeStatusRow(
                icon: Icons.devices_other_rounded,
                label: 'Owner surface',
                value: owner,
              ),
            ],
            if (_error case final error?) ...[
              const SizedBox(height: 12),
              Text(
                error,
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: palette.danger, fontSize: 12),
              ),
            ],
            const SizedBox(height: 18),
            if (_ownsResize) ...[
              FilledButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(
                        () => widget.connection.setResizeLock(
                          !_control.sizeLocked,
                        ),
                      ),
                icon: Icon(
                  _control.sizeLocked
                      ? Icons.lock_open_rounded
                      : Icons.lock_outline_rounded,
                ),
                label: Text(_control.sizeLocked ? 'Unlock size' : 'Lock size'),
              ),
              const SizedBox(height: 8),
              OutlinedButton.icon(
                onPressed: _busy
                    ? null
                    : () => _run(widget.connection.releaseResizeOwnership),
                icon: const Icon(Icons.logout_rounded),
                label: const Text('Release ownership'),
              ),
            ] else
              FilledButton.icon(
                onPressed: _busy ? null : _takeOwnership,
                icon: const Icon(Icons.aspect_ratio_rounded),
                label: const Text('Take resize ownership'),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _takeOwnership() async {
    final cells = widget.viewportCells;
    final size = widget.connection.terminalSize;
    await _run(
      () => widget.connection.requestResizeOwnership(
        cols: cells?.cols ?? size.cols,
        rows: cells?.rows ?? size.rows,
      ),
    );
  }

  Future<void> _run(Future<ResizeControl> Function() operation) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final control = await operation();
      if (mounted) setState(() => _control = control);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

final class _ResizeStatusRow extends StatelessWidget {
  const _ResizeStatusRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Icon(icon, color: palette.muted, size: 20),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            label,
            style: TextStyle(color: palette.text, fontSize: 13),
          ),
        ),
        const SizedBox(width: 12),
        Flexible(
          child: Text(
            value,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: palette.text,
              fontFamily: 'monospace',
              fontSize: 12,
            ),
          ),
        ),
      ],
    );
  }
}

String _resizeControlStatus(ResizeControl control) {
  if (control.sizeLocked) {
    return control.reason == ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER
        ? 'Follower / locked'
        : 'Owner / locked';
  }
  return switch (control.reason) {
    ResizeControlReason.RESIZE_CONTROL_REASON_OWNER => 'Resize owner',
    ResizeControlReason.RESIZE_CONTROL_REASON_OBSERVER => 'Observer',
    ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER => 'Follower',
    _ => control.canResize ? 'Resize owner' : 'Unavailable',
  };
}

final class _KeyboardKeyButton extends StatefulWidget {
  const _KeyboardKeyButton({
    required this.keyboardVisible,
    required this.focusLocked,
    required this.onTap,
    required this.onLongPress,
  });

  final bool keyboardVisible;
  final bool focusLocked;
  final VoidCallback? onTap;
  final VoidCallback? onLongPress;

  @override
  State<_KeyboardKeyButton> createState() => _KeyboardKeyButtonState();
}

final class _KeyboardKeyButtonState extends State<_KeyboardKeyButton> {
  static const _longPressDelay = Duration(milliseconds: 400);

  Timer? _longPressTimer;
  int? _pointer;
  bool _longPressTriggered = false;

  @override
  void dispose() {
    _cancelLongPress();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final enabled = widget.onTap != null && widget.onLongPress != null;
    final active = widget.focusLocked || widget.keyboardVisible;
    final tooltip = widget.focusLocked
        ? 'Allow the system keyboard to open'
        : widget.keyboardVisible
        ? 'Hide system keyboard'
        : 'Show system keyboard';
    final background = widget.focusLocked
        ? palette.warning
        : widget.keyboardVisible
        ? palette.text
        : palette.surfaceRaised;
    final foreground = active
        ? palette.background
        : enabled
        ? palette.muted
        : palette.faint;

    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: tooltip,
      onTap: enabled ? _handleSemanticTap : null,
      child: Tooltip(
        message: tooltip,
        child: Listener(
          behavior: HitTestBehavior.opaque,
          onPointerDown: enabled ? _handlePointerDown : null,
          onPointerUp: enabled ? _handlePointerUp : null,
          onPointerCancel: enabled ? _handlePointerCancel : null,
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: background,
              borderRadius: BorderRadius.circular(6),
            ),
            child: Center(
              child: Icon(LucideIcons.keyboard, size: 18, color: foreground),
            ),
          ),
        ),
      ),
    );
  }

  void _handlePointerDown(PointerDownEvent event) {
    if (_pointer != null) return;
    _pointer = event.pointer;
    _longPressTriggered = false;
    _longPressTimer = Timer(_longPressDelay, () {
      _longPressTimer = null;
      _longPressTriggered = true;
      HapticFeedback.mediumImpact();
      widget.onLongPress?.call();
    });
  }

  void _handlePointerUp(PointerUpEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _cancelLongPress();
    if (_longPressTriggered) {
      _longPressTriggered = false;
      return;
    }
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _handlePointerCancel(PointerCancelEvent event) {
    if (_pointer != event.pointer) return;
    _pointer = null;
    _longPressTriggered = false;
    _cancelLongPress();
  }

  void _handleSemanticTap() {
    HapticFeedback.selectionClick();
    widget.onTap?.call();
  }

  void _cancelLongPress() {
    _longPressTimer?.cancel();
    _longPressTimer = null;
  }
}

bool _mouseTrackingEnabled(TerminalModes? modes) =>
    modes?.mouseTracking == true ||
    modes?.mouseX10 == true ||
    modes?.mouseNormal == true ||
    modes?.mouseButtonEvent == true ||
    modes?.mouseAnyEvent == true;

const _usbKeyboardPage = 0x00070000;
