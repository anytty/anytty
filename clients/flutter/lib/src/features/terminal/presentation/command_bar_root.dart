part of 'terminal_command_bar.dart';


typedef TerminalQuickActionSaver = Future<void> Function(
  List<TerminalQuickAction> actions,
);

typedef TerminalQuickActionColors = ({
  Color foreground,
  Color background,
  Color border,
});

TerminalQuickActionColors terminalQuickActionColors(
  TerminalQuickAction action, {
  bool active = false,
  bool enabled = true,
  AnyttyPalette? palette,
}) {
  if (palette != null) {
    if (!enabled) {
      return (
        foreground: palette.faint,
        background: palette.surface,
        border: palette.border,
      );
    }
    if (active) {
      return (
        foreground: palette.accent,
        background: Color.lerp(palette.surface, palette.accent, 0.16)!,
        border: Color.lerp(palette.border, palette.accent, 0.48)!,
      );
    }
    final tone = switch (action.kind) {
      TerminalQuickActionKind.key => palette.text,
      TerminalQuickActionKind.chord => palette.accent,
      TerminalQuickActionKind.text => palette.warning,
    };
    return (
      foreground: tone,
      background: Color.lerp(palette.surface, tone, 0.06)!,
      border: Color.lerp(palette.border, tone, 0.24)!,
    );
  }
  if (!enabled) {
    return (
      foreground: const Color(0xff626a67),
      background: const Color(0xff1b1f1e),
      border: const Color(0xff292e2c),
    );
  }
  if (active) {
    return (
      foreground: const Color(0xff8fe3d5),
      background: const Color(0xff263633),
      border: const Color(0xff41615b),
    );
  }
  return switch (action.kind) {
    TerminalQuickActionKind.key => (
      foreground: const Color(0xffd3d9d7),
      background: const Color(0xff202523),
      border: const Color(0xff303633),
    ),
    TerminalQuickActionKind.chord => (
      foreground: const Color(0xff9fe6d8),
      background: const Color(0xff1c2725),
      border: const Color(0xff35514b),
    ),
    TerminalQuickActionKind.text => (
      foreground: const Color(0xffe8d39a),
      background: const Color(0xff27231c),
      border: const Color(0xff51472e),
    ),
  };
}

final class TerminalCommandBar extends StatefulWidget {
  const TerminalCommandBar({
    super.key,
    required this.actions,
    required this.inputEnabled,
    required this.modifiers,
    required this.keyboardControl,
    required this.onAction,
    required this.onConfigure,
    required this.functionKeysActive,
    required this.onFunctionKeys,
  });

  final List<TerminalQuickAction> actions;
  final bool inputEnabled;
  final TerminalModifierState modifiers;
  final Widget keyboardControl;
  final ValueChanged<TerminalQuickAction> onAction;
  final VoidCallback onConfigure;
  final bool functionKeysActive;
  final VoidCallback onFunctionKeys;

  @override
  State<TerminalCommandBar> createState() => _TerminalCommandBarState();
}

final class _TerminalCommandBarState extends State<TerminalCommandBar> {
  static const _itemExtent = 52.0;

  late ScrollController _controller;

  @override
  void initState() {
    super.initState();
    _resetController();
  }

  @override
  void didUpdateWidget(TerminalCommandBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    final oldIds = oldWidget.actions.map((action) => action.id).join('\n');
    final nextIds = widget.actions.map((action) => action.id).join('\n');
    if (oldIds != nextIds) {
      _controller.dispose();
      _resetController();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _resetController() {
    _controller = ScrollController();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return RepaintBoundary(
      child: SafeArea(
        top: false,
        minimum: const EdgeInsets.only(bottom: 1),
        child: Container(
          height: 38,
          padding: const EdgeInsets.symmetric(horizontal: 3, vertical: 2),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border(top: BorderSide(color: palette.border)),
          ),
          child: Row(
            children: [
              _BarAnchorButton(
                tooltip: widget.functionKeysActive
                    ? 'Close function keys'
                    : 'Show function keys',
                label: 'Fn',
                active: widget.functionKeysActive,
                onPressed: widget.onFunctionKeys,
              ),
              const SizedBox(width: 2),
              Expanded(child: _buildReel(context)),
              const SizedBox(width: 2),
              SizedBox.square(dimension: 34, child: widget.keyboardControl),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildReel(BuildContext context) {
    if (widget.actions.isEmpty) {
      return Semantics(
        button: true,
        label: 'Add command bar action',
        child: InkWell(
          key: const ValueKey('command-bar-empty'),
          onTap: widget.onConfigure,
          borderRadius: BorderRadius.circular(6),
          child: Center(
            child: Text(
              'Add command',
              style: TextStyle(
                color: AnyttyPalette.of(context).muted,
                fontSize: 12,
              ),
            ),
          ),
        ),
      );
    }
    return Semantics(
      container: true,
      enabled: widget.inputEnabled,
      label: 'Command bar',
      child: ListView.builder(
        key: const ValueKey('command-reel'),
        controller: _controller,
        scrollDirection: Axis.horizontal,
        physics: const ClampingScrollPhysics(),
        itemExtent: _itemExtent,
        itemCount: widget.actions.length,
        itemBuilder: (context, itemIndex) {
          final action = widget.actions[itemIndex];
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 1),
            child: _CommandReelItem(
              key: ValueKey('${action.id}-$itemIndex'),
              action: action,
              active: _modifierActive(action),
              enabled: widget.inputEnabled,
              onPressed: () {
                HapticFeedback.selectionClick();
                widget.onAction(action);
              },
              onConfigure: widget.onConfigure,
            ),
          );
        },
      ),
    );
  }

  bool _modifierActive(TerminalQuickAction action) {
    final modifier = action.kind == TerminalQuickActionKind.key
        ? action.key?.modifier
        : null;
    return modifier != null &&
        widget.modifiers.stateOf(modifier) != TerminalModifierLatch.off;
  }
}

final class _CommandReelItem extends StatelessWidget {
  const _CommandReelItem({
    super.key,
    required this.action,
    required this.active,
    required this.enabled,
    required this.onPressed,
    required this.onConfigure,
  });

  final TerminalQuickAction action;
  final bool active;
  final bool enabled;
  final VoidCallback onPressed;
  final VoidCallback onConfigure;

  @override
  Widget build(BuildContext context) {
    final colors = terminalQuickActionColors(
      action,
      active: active,
      enabled: enabled,
      palette: AnyttyPalette.of(context),
    );
    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: 'Run ${action.displayLabel}',
      hint: 'Long press to edit command bar',
      child: AnimatedContainer(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        curve: Curves.easeOutCubic,
        margin: const EdgeInsets.symmetric(vertical: 1),
        decoration: BoxDecoration(
          color: colors.background,
          borderRadius: BorderRadius.circular(4),
          border: Border.all(color: colors.border),
        ),
        child: InkWell(
          onTap: enabled ? onPressed : null,
          onLongPress: onConfigure,
          borderRadius: BorderRadius.circular(4),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Center(
              child: Text(
                action.displayLabel,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: TextStyle(
                  color: colors.foreground,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 9,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _BarAnchorButton extends StatelessWidget {
  const _BarAnchorButton({
    required this.tooltip,
    required this.label,
    required this.active,
    required this.onPressed,
  });

  final String tooltip;
  final String label;
  final bool active;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox.square(
      dimension: 34,
      child: Tooltip(
        message: tooltip,
        child: Semantics(
          button: true,
          label: tooltip,
          child: InkWell(
            onTap: onPressed,
            borderRadius: BorderRadius.circular(4),
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: active
                    ? Color.lerp(palette.surface, palette.accent, 0.16)
                    : palette.surfaceRaised,
                borderRadius: const BorderRadius.all(Radius.circular(4)),
              ),
              child: Center(
                child: Text(
                  label,
                  style: TextStyle(
                    color: active ? palette.accent : palette.text,
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<bool?> showAnyttyCommandBarEditor({
  required BuildContext context,
  required List<TerminalQuickAction> actions,
  required TerminalQuickActionSaver onSave,
}) {
  final palette = AnyttyPalette.of(context);
  return showGeneralDialog<bool>(
    context: context,
    barrierDismissible: false,
    barrierLabel: 'Command bar editor',
    barrierColor: palette.overlay,
    transitionDuration: Duration.zero,
    pageBuilder: (context, _, _) =>
        _CommandEditorRoute(actions: actions, onSave: onSave),
    transitionBuilder: (context, animation, _, child) => child,
  );
}

final class _CommandEditorRoute extends StatelessWidget {
  const _CommandEditorRoute({required this.actions, required this.onSave});

  final List<TerminalQuickAction> actions;
  final TerminalQuickActionSaver onSave;

  @override
  Widget build(BuildContext context) {
    final media = MediaQuery.of(context);
    final available = math.max(
      280.0,
      media.size.height - media.viewInsets.bottom - media.padding.top - 12,
    );
    return SafeArea(
      top: false,
      child: AnimatedPadding(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        curve: Curves.easeOutCubic,
        padding: EdgeInsets.only(bottom: media.viewInsets.bottom),
        child: Align(
          alignment: Alignment.bottomCenter,
          child: SizedBox(
            width: double.infinity,
            height: math.min(available, 680),
            child: TerminalCommandEditorTray(actions: actions, onSave: onSave),
          ),
        ),
      ),
    );
  }
}
