part of 'terminal_command_bar.dart';


final class _CommandEditorRow extends StatelessWidget {
  const _CommandEditorRow({
    super.key,
    required this.action,
    required this.index,
    required this.onEdit,
    required this.onDelete,
  });

  final TerminalQuickAction action;
  final int index;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final kind = switch (action.kind) {
      TerminalQuickActionKind.key => 'Key',
      TerminalQuickActionKind.chord => 'Chord',
      TerminalQuickActionKind.text =>
        action.sendEnter ? 'Text + Enter' : 'Text',
    };
    return Material(
      color: palette.surface,
      child: InkWell(
        onTap: onEdit,
        child: Container(
          constraints: const BoxConstraints(minHeight: 58),
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: palette.border)),
          ),
          padding: const EdgeInsets.only(left: 16, right: 4),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      action.displayLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontFamily: 'JetBrainsMonoNerd',
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      kind,
                      style: TextStyle(color: palette.muted, fontSize: 11),
                    ),
                  ],
                ),
              ),
              _TrayIconButton(
                tooltip: 'Delete ${action.displayLabel}',
                icon: LucideIcons.trash2,
                danger: true,
                onPressed: onDelete,
              ),
              ReorderableDragStartListener(
                index: index,
                child: Semantics(
                  label: 'Reorder ${action.displayLabel}',
                  child: SizedBox.square(
                    dimension: 48,
                    child: Icon(
                      LucideIcons.gripVertical,
                      color: palette.muted,
                      size: 19,
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

final class _CommandKindPicker extends StatelessWidget {
  const _CommandKindPicker({required this.value, required this.onChanged});

  final TerminalQuickActionKind value;
  final ValueChanged<TerminalQuickActionKind> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 48,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        children: [
          for (final option in TerminalQuickActionKind.values)
            Expanded(
              child: _KindOption(
                label: switch (option) {
                  TerminalQuickActionKind.key => 'Key',
                  TerminalQuickActionKind.chord => 'Chord',
                  TerminalQuickActionKind.text => 'Text',
                },
                selected: option == value,
                onPressed: () => onChanged(option),
              ),
            ),
        ],
      ),
    );
  }
}

final class _KindOption extends StatelessWidget {
  const _KindOption({
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
    return Semantics(
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(4),
        child: AnimatedContainer(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: selected ? palette.surface : Colors.transparent,
            borderRadius: BorderRadius.circular(4),
            border: Border.all(
              color: selected ? palette.borderStrong : Colors.transparent,
            ),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: selected ? palette.text : palette.muted,
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

final class _KeySelectionButton extends StatelessWidget {
  const _KeySelectionButton({required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: 'Choose key, current value $label',
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(6),
        child: Container(
          constraints: const BoxConstraints(minHeight: 50),
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border.all(color: palette.borderStrong),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(
                    color: palette.text,
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              Icon(LucideIcons.chevronRight, color: palette.muted, size: 18),
            ],
          ),
        ),
      ),
    );
  }
}

final class _ModifierChoice extends StatelessWidget {
  const _ModifierChoice({
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
    return Semantics(
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(6),
        child: AnimatedContainer(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
          constraints: const BoxConstraints(minWidth: 68, minHeight: 44),
          alignment: Alignment.center,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: selected
                ? palette.accent.withValues(alpha: 0.12)
                : palette.surface,
            border: Border.all(
              color: selected ? palette.accent : palette.borderStrong,
            ),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: selected ? palette.accent : palette.text,
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

final class _InlineToggle extends StatelessWidget {
  const _InlineToggle({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      toggled: value,
      label: label,
      child: InkWell(
        onTap: () => onChanged(!value),
        borderRadius: BorderRadius.circular(6),
        child: Container(
          constraints: const BoxConstraints(minHeight: 50),
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border.all(color: palette.borderStrong),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(color: palette.text, fontSize: 13),
                ),
              ),
              AnimatedContainer(
                duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
                width: 24,
                height: 24,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: value ? palette.accent : Colors.transparent,
                  border: Border.all(
                    color: value ? palette.accent : palette.borderStrong,
                  ),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: value
                    ? Icon(
                        LucideIcons.check,
                        color: palette.accentText,
                        size: 16,
                      )
                    : null,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _KeyPickerRow extends StatelessWidget {
  const _KeyPickerRow({
    super.key,
    required this.value,
    required this.selected,
    required this.onPressed,
  });

  final TerminalQuickKey value;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return InkWell(
      onTap: onPressed,
      child: Container(
        constraints: const BoxConstraints(minHeight: 52),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        decoration: BoxDecoration(
          color: selected ? palette.accent.withValues(alpha: 0.08) : null,
          border: Border(bottom: BorderSide(color: palette.border)),
        ),
        child: Row(
          children: [
            Expanded(
              child: Text(
                value.label,
                style: TextStyle(
                  color: palette.text,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            if (selected)
              Icon(LucideIcons.check, color: palette.accent, size: 18),
          ],
        ),
      ),
    );
  }
}

final class _TrayIconButton extends StatelessWidget {
  const _TrayIconButton({
    required this.tooltip,
    required this.icon,
    required this.onPressed,
    this.danger = false,
  });

  final String tooltip;
  final IconData icon;
  final VoidCallback onPressed;
  final bool danger;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox.square(
      dimension: 48,
      child: Tooltip(
        message: tooltip,
        child: IconButton(
          onPressed: onPressed,
          icon: Icon(icon, size: 19),
          color: danger ? palette.danger : palette.muted,
        ),
      ),
    );
  }
}

final class _TrayTextButton extends StatelessWidget {
  const _TrayTextButton({
    required this.label,
    required this.enabled,
    required this.emphasized,
    required this.onPressed,
  });

  final String label;
  final bool enabled;
  final bool emphasized;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ConstrainedBox(
      constraints: const BoxConstraints(minWidth: 56, minHeight: 44),
      child: TextButton(
        onPressed: enabled ? onPressed : null,
        style: TextButton.styleFrom(
          foregroundColor: emphasized ? palette.accent : palette.muted,
          disabledForegroundColor: palette.faint,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
        ),
        child: Text(
          label,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
        ),
      ),
    );
  }
}

Future<bool> _confirmDiscard(BuildContext context) async {
  final palette = AnyttyPalette.of(context);
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: 'Discard changes',
    barrierColor: palette.overlay,
    transitionDuration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
    pageBuilder: (context, _, _) => Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 320),
        child: Material(
          color: palette.surface,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
            side: BorderSide(color: palette.border),
          ),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Discard changes?',
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Your command bar changes have not been saved.',
                  style: TextStyle(color: palette.muted, fontSize: 13),
                ),
                const SizedBox(height: 20),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    _TrayTextButton(
                      label: 'Keep editing',
                      enabled: true,
                      emphasized: false,
                      onPressed: () => Navigator.pop(context, false),
                    ),
                    const SizedBox(width: 4),
                    _TrayTextButton(
                      label: 'Discard',
                      enabled: true,
                      emphasized: true,
                      onPressed: () => Navigator.pop(context, true),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
  return result ?? false;
}
