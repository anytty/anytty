part of 'terminal_workspace_screen.dart';

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
