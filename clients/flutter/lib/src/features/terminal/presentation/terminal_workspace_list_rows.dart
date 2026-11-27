part of 'terminal_workspace_screen.dart';

final class _TerminalRow extends StatelessWidget {
  const _TerminalRow({
    required this.terminal,
    required this.pinned,
    required this.canMoveUp,
    required this.canMoveDown,
    required this.searchQuery,
    required this.selected,
    required this.selectionMode,
    required this.onTap,
    required this.onLongPress,
    required this.onToggleSelected,
    required this.onAction,
  });

  final TerminalInfo terminal;
  final bool pinned;
  final bool canMoveUp;
  final bool canMoveDown;
  final String searchQuery;
  final bool selected;
  final bool selectionMode;
  final VoidCallback onTap;
  final VoidCallback onLongPress;
  final VoidCallback onToggleSelected;
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
    final outputQuiet = terminalOutputQuietDuration(terminal);
    final outputTone = terminalOutputActivityTone(terminal);
    return SizedBox(
      height: 88,
      child: Card(
        key: ValueKey('terminal-row-${terminal.ref.terminalId}'),
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
            if (selectionMode) {
              onToggleSelected();
            } else {
              onTap();
            }
          },
          onLongPress: onLongPress,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(9, 7, 2, 7),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (selectionMode)
                  SizedBox(
                    width: 40,
                    height: 40,
                    child: Checkbox(
                      key: ValueKey(
                        'terminal-select-${terminal.ref.terminalId}',
                      ),
                      value: selected,
                      onChanged: (_) => onToggleSelected(),
                      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                    ),
                  ),
                Semantics(
                  container: true,
                  label: outputQuiet == null
                      ? anyttyText(
                          context,
                          en: 'No terminal output time available',
                          zh: '暂无终端输出时间',
                        )
                      : _terminalOutputActivitySemantics(context, outputQuiet),
                  child: SizedBox(
                    width: 40,
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
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
                                  color: _terminalStateColor(
                                    context,
                                    terminal.state,
                                  ),
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: palette.surface,
                                    width: 2,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                        if (outputQuiet != null) ...[
                          const SizedBox(height: 2),
                          Icon(
                            LucideIcons.clock3,
                            size: 10,
                            color: _terminalOutputActivityColor(
                              palette,
                              outputTone,
                            ),
                          ),
                          Text(
                            _terminalOutputActivityLabel(context, outputQuiet),
                            maxLines: 1,
                            overflow: TextOverflow.clip,
                            style: TextStyle(
                              color: _terminalOutputActivityColor(
                                palette,
                                outputTone,
                              ),
                              fontFamily: 'JetBrainsMonoNerd',
                              fontSize: 9,
                              fontWeight: FontWeight.w700,
                              height: 1,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
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

final class _TerminalSelectionBar extends StatelessWidget {
  const _TerminalSelectionBar({
    required this.selectedCount,
    required this.totalCount,
    required this.canEnd,
    required this.canDelete,
    required this.busy,
    required this.onSelectAll,
    required this.onEnd,
    required this.onDelete,
    required this.onClose,
  });

  final int selectedCount;
  final int totalCount;
  final bool canEnd;
  final bool canDelete;
  final bool busy;
  final VoidCallback onSelectAll;
  final VoidCallback onEnd;
  final VoidCallback onDelete;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final allSelected = totalCount > 0 && selectedCount == totalCount;
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 4, 14, 2),
      child: Container(
        height: 56,
        padding: const EdgeInsets.symmetric(horizontal: 4),
        decoration: BoxDecoration(
          color: palette.surfaceRaised,
          borderRadius: BorderRadius.circular(14),
          border: Border.all(color: palette.border),
        ),
        child: Row(
          children: [
            IconButton(
              tooltip: anyttyText(context, en: 'Close selection', zh: '关闭选择'),
              onPressed: busy ? null : onClose,
              icon: const Icon(Icons.close_rounded, size: 19),
            ),
            Expanded(
              child: Text(
                anyttyText(
                  context,
                  en: '$selectedCount selected',
                  zh: '已选择 $selectedCount 个',
                ),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            IconButton(
              tooltip: allSelected
                  ? anyttyText(context, en: 'Clear selection', zh: '清除选择')
                  : anyttyText(context, en: 'Select all', zh: '全选'),
              onPressed: busy ? null : onSelectAll,
              icon: Icon(
                allSelected
                    ? Icons.indeterminate_check_box_outlined
                    : Icons.select_all_rounded,
                size: 20,
              ),
            ),
            IconButton(
              tooltip: anyttyText(context, en: 'End processes', zh: '结束进程'),
              onPressed: busy || !canEnd ? null : onEnd,
              icon: const Icon(Icons.stop_circle_outlined, size: 20),
            ),
            IconButton(
              tooltip: anyttyText(context, en: 'Delete records', zh: '删除记录'),
              onPressed: busy || !canDelete ? null : onDelete,
              color: palette.danger,
              icon: const Icon(Icons.delete_outline_rounded, size: 20),
            ),
          ],
        ),
      ),
    );
  }
}

final class _TerminalSearchField extends StatelessWidget {
  const _TerminalSearchField({
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
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 6, 14, 4),
      child: SizedBox(
        height: 42,
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
            key: const ValueKey('terminal-list-search-field'),
            controller: controller,
            focusNode: focusNode,
            textInputAction: TextInputAction.search,
            decoration: InputDecoration(
              hintText: anyttyText(context, en: 'Search terminals', zh: '搜索终端'),
              prefixIcon: Icon(
                Icons.search_rounded,
                color: palette.muted,
                size: 18,
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
                    tooltip: anyttyText(
                      context,
                      en: 'Close search',
                      zh: '关闭搜索',
                    ),
                    onPressed: onClose,
                    icon: const Icon(Icons.close_rounded, size: 17),
                  ),
                ],
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
    required this.onSearch,
  });

  final TerminalStatusFilter status;
  final int runningCount;
  final int exitedCount;
  final int totalCount;
  final int selectedTagCount;
  final bool tagsAvailable;
  final ValueChanged<TerminalStatusFilter> onStatusChanged;
  final VoidCallback onTags;
  final VoidCallback onSearch;

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
            const SizedBox(width: 8),
            Material(
              color: palette.surfaceRaised,
              borderRadius: BorderRadius.circular(14),
              child: SizedBox.square(
                dimension: 56,
                child: IconButton(
                  tooltip: anyttyText(
                    context,
                    en: 'Search terminals',
                    zh: '搜索终端',
                  ),
                  onPressed: onSearch,
                  icon: const Icon(Icons.search_rounded, size: 20),
                ),
              ),
            ),
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

String _terminalOutputActivityLabel(BuildContext context, Duration quiet) {
  if (quiet < const Duration(seconds: 5)) {
    return anyttyText(context, en: 'now', zh: '刚刚');
  }
  if (quiet < const Duration(minutes: 1)) {
    final seconds = quiet.inSeconds;
    return anyttyText(
      context,
      en: _terminalOutputCompactDuration(seconds, 's'),
      zh: '$seconds秒',
    );
  }
  if (quiet < const Duration(hours: 1)) {
    final minutes = quiet.inMinutes;
    return anyttyText(
      context,
      en: _terminalOutputCompactDuration(minutes, 'm'),
      zh: '$minutes分',
    );
  }
  final hours = quiet.inHours;
  return anyttyText(
    context,
    en: _terminalOutputCompactDuration(hours, 'h'),
    zh: '$hours时',
  );
}

String _terminalOutputCompactDuration(int value, String unit) => '$value$unit';

String _terminalOutputActivitySemantics(BuildContext context, Duration quiet) {
  if (quiet < const Duration(seconds: 5)) {
    return anyttyText(context, en: 'Last output just now', zh: '最近刚有输出');
  }
  if (quiet < const Duration(minutes: 1)) {
    final seconds = quiet.inSeconds;
    return anyttyText(
      context,
      en: 'Last output $seconds seconds ago',
      zh: '最近输出于 $seconds 秒前',
    );
  }
  if (quiet < const Duration(hours: 1)) {
    final minutes = quiet.inMinutes;
    return anyttyText(
      context,
      en: 'Last output $minutes minutes ago',
      zh: '最近输出于 $minutes 分钟前',
    );
  }
  final hours = quiet.inHours;
  return anyttyText(
    context,
    en: 'Last output $hours hours ago',
    zh: '最近输出于 $hours 小时前',
  );
}

Color _terminalOutputActivityColor(
  AnyttyPalette palette,
  TerminalOutputActivityTone tone,
) {
  return switch (tone) {
    TerminalOutputActivityTone.fresh ||
    TerminalOutputActivityTone.recent => palette.accent,
    TerminalOutputActivityTone.idle ||
    TerminalOutputActivityTone.stale => palette.muted,
    TerminalOutputActivityTone.none => palette.faint,
  };
}
