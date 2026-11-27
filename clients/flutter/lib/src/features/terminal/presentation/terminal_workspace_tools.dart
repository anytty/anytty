part of 'terminal_workspace_screen.dart';

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
