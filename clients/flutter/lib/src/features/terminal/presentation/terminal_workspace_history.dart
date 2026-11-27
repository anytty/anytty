part of 'terminal_workspace_screen.dart';

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
          Flexible(
            flex: 3,
            child: Container(
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
