import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../generated/proto/apipb/terminal.pb.dart';
import '../../../shared/domain/fuzzy_search.dart';
import '../../../shared/presentation/fuzzy_highlight_text.dart';
import '../domain/terminal_inventory.dart';
import 'terminal_program_icon.dart';

typedef TerminalInventoryLoader = Future<List<TerminalInfo>> Function(
  String endpointId,
);

final class TerminalSwitcherEndpoint {
  const TerminalSwitcherEndpoint({
    required this.endpointId,
    required this.label,
    required this.current,
    this.terminals,
    this.activeTerminalId,
  });

  final String endpointId;
  final String label;
  final bool current;
  final List<TerminalInfo>? terminals;
  final String? activeTerminalId;
}

final class TerminalSwitcherSelection {
  const TerminalSwitcherSelection({
    required this.endpointId,
    required this.endpointLabel,
    required this.terminalId,
  });

  final String endpointId;
  final String endpointLabel;
  final String terminalId;
}

Future<TerminalSwitcherSelection?> showAnyttyTerminalSwitcher({
  required BuildContext context,
  required List<TerminalSwitcherEndpoint> endpoints,
  required TerminalInventoryLoader loadTerminals,
}) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<TerminalSwitcherSelection>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    showDragHandle: false,
    backgroundColor: Colors.transparent,
    barrierColor: palette.overlay,
    sheetAnimationStyle: AnimationStyle(
      duration: AnyttyMotion.resolve(context, AnyttyMotion.standard),
      reverseDuration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
    ),
    builder: (sheetContext) {
      final media = MediaQuery.of(sheetContext);
      return AnimatedPadding(
        duration: AnyttyMotion.resolve(sheetContext, AnyttyMotion.quick),
        curve: Curves.easeOutCubic,
        padding: EdgeInsets.only(bottom: media.viewInsets.bottom),
        child: TerminalSwitcherSheet(
          endpoints: endpoints,
          loadTerminals: loadTerminals,
        ),
      );
    },
  );
}

final class TerminalSwitcherSheet extends StatefulWidget {
  const TerminalSwitcherSheet({
    super.key,
    required this.endpoints,
    required this.loadTerminals,
  });

  final List<TerminalSwitcherEndpoint> endpoints;
  final TerminalInventoryLoader loadTerminals;

  @override
  State<TerminalSwitcherSheet> createState() => _TerminalSwitcherSheetState();
}

final class _TerminalSwitcherSheetState extends State<TerminalSwitcherSheet> {
  final _searchController = TextEditingController();
  TerminalStatusFilter _status = TerminalStatusFilter.running;
  late final Map<String, bool> _expanded = {
    for (final endpoint in widget.endpoints)
      endpoint.endpointId: endpoint.current,
  };
  late final Map<String, List<TerminalInfo>> _inventories = {
    for (final endpoint in widget.endpoints)
      if (endpoint.terminals != null) endpoint.endpointId: endpoint.terminals!,
  };
  final Set<String> _loading = {};
  final Map<String, Object> _errors = {};

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
    super.dispose();
  }

  void _handleSearchChanged() {
    setState(() {});
    if (_searchController.text.trim().isNotEmpty) {
      for (final endpoint in widget.endpoints) {
        if (!_inventories.containsKey(endpoint.endpointId)) {
          unawaited(_load(endpoint));
        }
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final media = MediaQuery.of(context);
    final availableHeight = math.max(
      280.0,
      media.size.height - media.viewInsets.bottom - media.padding.top,
    );
    final preferredHeight = math.min(media.size.height * 0.78, 680.0);
    final visibleEndpoints = widget.endpoints
        .where(_endpointMatchesFilters)
        .toList(growable: false);
    return SizedBox(
      width: double.infinity,
      height: math.min(availableHeight, math.max(360, preferredHeight)),
      child: Material(
        color: palette.surface,
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(
          borderRadius: const BorderRadius.vertical(top: Radius.circular(18)),
          side: BorderSide(color: palette.border),
        ),
        child: Column(
          children: [
            const _SheetHandle(),
            _SheetHeader(endpointCount: widget.endpoints.length),
            Divider(height: 1, color: palette.border),
            _SwitcherControls(
              searchController: _searchController,
              status: _status,
              onStatusChanged: (status) => setState(() => _status = status),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: visibleEndpoints.isEmpty
                  ? const _SwitcherEmpty()
                  : ListView.builder(
                      padding: const EdgeInsets.only(bottom: 24),
                      itemCount: visibleEndpoints.length,
                      itemBuilder: (context, index) =>
                          _buildEndpoint(context, visibleEndpoints[index]),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildEndpoint(
    BuildContext context,
    TerminalSwitcherEndpoint endpoint,
  ) {
    final palette = AnyttyPalette.of(context);
    final searching = _searchController.text.trim().isNotEmpty;
    final expanded = searching || (_expanded[endpoint.endpointId] ?? false);
    final inventory = _inventories[endpoint.endpointId];
    final visibleInventory = inventory == null
        ? null
        : _filterInventory(endpoint, inventory);
    final loading = _loading.contains(endpoint.endpointId);
    final error = _errors[endpoint.endpointId];
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        InkWell(
          key: ValueKey('endpoint-${endpoint.endpointId}'),
          onTap: () => _toggle(endpoint),
          child: SizedBox(
            height: 54,
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                children: [
                  Container(
                    width: 32,
                    height: 32,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: endpoint.current
                          ? palette.accent.withValues(alpha: 0.12)
                          : palette.surfaceRaised,
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Icon(
                      Icons.monitor_rounded,
                      size: 17,
                      color: endpoint.current ? palette.accent : palette.muted,
                    ),
                  ),
                  const SizedBox(width: 11),
                  Expanded(
                    child: FuzzyHighlightText(
                      endpoint.label,
                      query: _searchController.text,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  if (endpoint.current)
                    _EndpointBadge(label: 'Current', color: palette.accent)
                  else if (inventory != null)
                    Text(
                      '${inventory.length}',
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  const SizedBox(width: 8),
                  AnimatedRotation(
                    turns: expanded ? 0.25 : 0,
                    duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
                    curve: Curves.easeOutCubic,
                    child: Icon(
                      Icons.chevron_right_rounded,
                      size: 20,
                      color: palette.faint,
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        AnimatedSize(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.standard),
          curve: AnyttyMotion.emphasized,
          alignment: Alignment.topCenter,
          child: expanded
              ? _EndpointInventory(
                  endpoint: endpoint,
                  terminals: visibleInventory,
                  loading: loading,
                  error: error,
                  searchQuery: _searchController.text,
                  onRetry: () => _load(endpoint, force: true),
                  onSelected: (terminal) {
                    HapticFeedback.selectionClick();
                    Navigator.pop(
                      context,
                      TerminalSwitcherSelection(
                        endpointId: endpoint.endpointId,
                        endpointLabel: endpoint.label,
                        terminalId: terminal.ref.terminalId,
                      ),
                    );
                  },
                )
              : const SizedBox.shrink(),
        ),
        Divider(height: 1, color: palette.border),
      ],
    );
  }

  bool _endpointMatchesFilters(TerminalSwitcherEndpoint endpoint) {
    final query = _searchController.text.trim();
    final inventory = _inventories[endpoint.endpointId];
    if (query.isEmpty) return true;
    final endpointMatches = fuzzyMatchesAny([
      endpoint.label,
      endpoint.endpointId,
    ], query);
    if (endpointMatches) return true;
    if (inventory == null) {
      return _loading.contains(endpoint.endpointId) ||
          _errors.containsKey(endpoint.endpointId);
    }
    return _filterInventory(endpoint, inventory).isNotEmpty;
  }

  List<TerminalInfo> _filterInventory(
    TerminalSwitcherEndpoint endpoint,
    List<TerminalInfo> inventory,
  ) {
    final query = _searchController.text.trim();
    final endpointMatches =
        query.isNotEmpty &&
        fuzzyMatchesAny([endpoint.label, endpoint.endpointId], query);
    return filterTerminals(
      terminals: inventory,
      status: _status,
      query: endpointMatches ? '' : query,
    );
  }

  void _toggle(TerminalSwitcherEndpoint endpoint) {
    HapticFeedback.selectionClick();
    final next = !(_expanded[endpoint.endpointId] ?? false);
    setState(() => _expanded[endpoint.endpointId] = next);
    if (next && !_inventories.containsKey(endpoint.endpointId)) {
      _load(endpoint);
    }
  }

  Future<void> _load(
    TerminalSwitcherEndpoint endpoint, {
    bool force = false,
  }) async {
    final endpointId = endpoint.endpointId;
    if (_loading.contains(endpointId) ||
        (!force && _inventories.containsKey(endpointId))) {
      return;
    }
    setState(() {
      _loading.add(endpointId);
      _errors.remove(endpointId);
    });
    try {
      final terminals = await widget.loadTerminals(endpointId);
      if (!mounted) return;
      setState(() => _inventories[endpointId] = terminals);
    } catch (error) {
      if (!mounted) return;
      setState(() => _errors[endpointId] = error);
    } finally {
      if (mounted) setState(() => _loading.remove(endpointId));
    }
  }
}

final class _SwitcherControls extends StatelessWidget {
  const _SwitcherControls({
    required this.searchController,
    required this.status,
    required this.onStatusChanged,
  });

  final TextEditingController searchController;
  final TerminalStatusFilter status;
  final ValueChanged<TerminalStatusFilter> onStatusChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
      child: Column(
        children: [
          SizedBox(
            height: 44,
            child: TextField(
              key: const ValueKey('terminal-switcher-search-field'),
              controller: searchController,
              autofocus: false,
              textInputAction: TextInputAction.search,
              decoration: InputDecoration(
                hintText: anyttyText(
                  context,
                  en: 'Search terminals or devices',
                  zh: '搜索终端或设备',
                ),
                prefixIcon: Icon(
                  Icons.search_rounded,
                  color: palette.muted,
                  size: 18,
                ),
                suffixIcon: searchController.text.isEmpty
                    ? null
                    : IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Clear search',
                          zh: '清除搜索',
                        ),
                        onPressed: searchController.clear,
                        icon: const Icon(Icons.close_rounded, size: 17),
                      ),
                filled: true,
                fillColor: palette.background,
                contentPadding: const EdgeInsets.symmetric(vertical: 10),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                  borderSide: BorderSide(color: palette.border),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(12),
                  borderSide: BorderSide(color: palette.accent, width: 1.5),
                ),
              ),
            ),
          ),
          const SizedBox(height: 8),
          Container(
            height: 44,
            padding: const EdgeInsets.all(3),
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              children: [
                _SwitcherStatusOption(
                  key: const ValueKey('switcher-filter-running'),
                  label: anyttyText(context, en: 'Running', zh: '运行中'),
                  selected: status == TerminalStatusFilter.running,
                  onPressed: () =>
                      onStatusChanged(TerminalStatusFilter.running),
                ),
                _SwitcherStatusOption(
                  key: const ValueKey('switcher-filter-all'),
                  label: anyttyText(context, en: 'All', zh: '全部'),
                  selected: status == TerminalStatusFilter.all,
                  onPressed: () => onStatusChanged(TerminalStatusFilter.all),
                ),
                _SwitcherStatusOption(
                  key: const ValueKey('switcher-filter-exited'),
                  label: anyttyText(context, en: 'Exited', zh: '已退出'),
                  selected: status == TerminalStatusFilter.exited,
                  onPressed: () => onStatusChanged(TerminalStatusFilter.exited),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _SwitcherStatusOption extends StatelessWidget {
  const _SwitcherStatusOption({
    super.key,
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Expanded(
      child: Semantics(
        button: true,
        selected: selected,
        child: Material(
          color: selected ? palette.accent : Colors.transparent,
          borderRadius: BorderRadius.circular(9),
          child: InkWell(
            onTap: onPressed,
            borderRadius: BorderRadius.circular(9),
            child: Center(
              child: Text(
                label,
                style: TextStyle(
                  color: selected ? palette.accentText : palette.muted,
                  fontSize: 12,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _SwitcherEmpty extends StatelessWidget {
  const _SwitcherEmpty();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.search_off_rounded, color: palette.muted, size: 30),
          const SizedBox(height: 10),
          Text(
            anyttyText(context, en: 'No matching terminals', zh: '没有匹配的终端'),
            style: TextStyle(color: palette.muted, fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

final class _SheetHandle extends StatelessWidget {
  const _SheetHandle();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 18,
      child: Center(
        child: Container(
          width: 34,
          height: 4,
          decoration: BoxDecoration(
            color: palette.borderStrong,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
      ),
    );
  }
}

final class _SheetHeader extends StatelessWidget {
  const _SheetHeader({required this.endpointCount});

  final int endpointCount;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 52,
      child: Padding(
        padding: const EdgeInsets.only(left: 16, right: 4),
        child: Row(
          children: [
            Expanded(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Switch terminal',
                    style: TextStyle(
                      color: palette.text,
                      fontSize: 16,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  Text(
                    '$endpointCount device${endpointCount == 1 ? '' : 's'}',
                    style: TextStyle(color: palette.muted, fontSize: 11),
                  ),
                ],
              ),
            ),
            IconButton(
              tooltip: 'Close terminal switcher',
              onPressed: () => Navigator.pop(context),
              icon: const Icon(Icons.close_rounded, size: 19),
            ),
          ],
        ),
      ),
    );
  }
}

final class _EndpointBadge extends StatelessWidget {
  const _EndpointBadge({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
    decoration: BoxDecoration(
      color: color.withValues(alpha: 0.1),
      borderRadius: BorderRadius.circular(4),
    ),
    child: Text(
      label,
      style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w600),
    ),
  );
}

final class _EndpointInventory extends StatelessWidget {
  const _EndpointInventory({
    required this.endpoint,
    required this.terminals,
    required this.loading,
    required this.error,
    required this.searchQuery,
    required this.onRetry,
    required this.onSelected,
  });

  final TerminalSwitcherEndpoint endpoint;
  final List<TerminalInfo>? terminals;
  final bool loading;
  final Object? error;
  final String searchQuery;
  final VoidCallback onRetry;
  final ValueChanged<TerminalInfo> onSelected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    if (loading) {
      return const SizedBox(
        height: 64,
        child: Center(
          child: SizedBox.square(
            dimension: 18,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }
    if (error != null) {
      return SizedBox(
        height: 64,
        child: Padding(
          padding: const EdgeInsets.only(left: 59, right: 8),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  'Could not load terminals',
                  style: TextStyle(color: palette.danger, fontSize: 12),
                ),
              ),
              TextButton(onPressed: onRetry, child: const Text('Retry')),
            ],
          ),
        ),
      );
    }
    final values = terminals ?? const <TerminalInfo>[];
    if (values.isEmpty) {
      return SizedBox(
        height: 54,
        child: Padding(
          padding: const EdgeInsets.only(left: 59),
          child: Align(
            alignment: Alignment.centerLeft,
            child: Text(
              'No terminals',
              style: TextStyle(color: palette.muted, fontSize: 12),
            ),
          ),
        ),
      );
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (final terminal in values)
          _TerminalRow(
            terminal: terminal,
            active:
                endpoint.current &&
                terminal.ref.terminalId == endpoint.activeTerminalId,
            searchQuery: searchQuery,
            onTap: () => onSelected(terminal),
          ),
      ],
    );
  }
}

final class _TerminalRow extends StatelessWidget {
  const _TerminalRow({
    required this.terminal,
    required this.active,
    required this.searchQuery,
    required this.onTap,
  });

  final TerminalInfo terminal;
  final bool active;
  final String searchQuery;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final cwd = terminal.liveCwd.isNotEmpty ? terminal.liveCwd : terminal.cwd;
    final subtitle = cwd.isNotEmpty
        ? cwd
        : terminal.command.isNotEmpty
        ? terminal.command.join(' ')
        : _stateLabel(terminal.state);
    final hasSize =
        terminal.hasSize() && terminal.size.cols > 0 && terminal.size.rows > 0;
    return InkWell(
      key: ValueKey(
        'terminal-${terminal.ref.endpointId}-${terminal.ref.terminalId}',
      ),
      onTap: onTap,
      child: Container(
        height: 58,
        color: active ? palette.accent.withValues(alpha: 0.08) : null,
        padding: const EdgeInsets.fromLTRB(24, 7, 16, 7),
        child: Row(
          children: [
            SizedBox(
              width: 30,
              child: Stack(
                alignment: Alignment.center,
                clipBehavior: Clip.none,
                children: [
                  Icon(
                    terminalProgramIcon(terminal),
                    size: 18,
                    color: palette.text,
                  ),
                  Positioned(
                    right: 2,
                    bottom: 3,
                    child: Container(
                      width: 7,
                      height: 7,
                      decoration: BoxDecoration(
                        color: _stateColor(palette, terminal.state),
                        shape: BoxShape.circle,
                        border: Border.all(color: palette.surface, width: 1.5),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 9),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  FuzzyHighlightText(
                    _displayName(terminal),
                    query: searchQuery,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.text,
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 2),
                  FuzzyHighlightText(
                    subtitle,
                    query: searchQuery,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.muted,
                      fontFamily: cwd.isNotEmpty ? 'JetBrainsMonoNerd' : null,
                      fontSize: 11,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            if (active)
              Icon(Icons.check_rounded, color: palette.accent, size: 18)
            else if (hasSize)
              Text(
                '${terminal.size.cols}x${terminal.size.rows}',
                style: TextStyle(
                  color: palette.faint,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 10,
                ),
              ),
          ],
        ),
      ),
    );
  }
}

String _displayName(TerminalInfo terminal) {
  if (terminal.name.trim().isNotEmpty) return terminal.name.trim();
  if (terminal.foregroundProcess.trim().isNotEmpty) {
    return terminal.foregroundProcess.trim();
  }
  if (terminal.command.isNotEmpty && terminal.command.first.trim().isNotEmpty) {
    return terminal.command.first.trim();
  }
  return terminal.ref.terminalId;
}

String _stateLabel(TerminalState state) => switch (state) {
  TerminalState.TERMINAL_STATE_CREATED => 'Starting',
  TerminalState.TERMINAL_STATE_RUNNING => 'Running',
  TerminalState.TERMINAL_STATE_EXITED => 'Exited',
  TerminalState.TERMINAL_STATE_REMOVED => 'Removed',
  _ => 'Unknown',
};

Color _stateColor(AnyttyPalette palette, TerminalState state) =>
    switch (state) {
      TerminalState.TERMINAL_STATE_CREATED => palette.warning,
      TerminalState.TERMINAL_STATE_RUNNING => palette.success,
      TerminalState.TERMINAL_STATE_EXITED => palette.muted,
      TerminalState.TERMINAL_STATE_REMOVED => palette.danger,
      _ => palette.muted,
    };
