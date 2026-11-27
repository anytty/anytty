part of 'terminal_workspace_screen.dart';

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

final class _TerminalFailure extends StatelessWidget {
  const _TerminalFailure({
    required this.message,
    required this.onRetry,
    this.dark = false,
    this.collapseDetails = false,
  });

  final bool collapseDetails;
  final String message;
  final VoidCallback onRetry;
  final bool dark;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: SafeArea(
        child: Center(
          child: SingleChildScrollView(
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
                if (collapseDetails) ...[
                  Text(
                    anyttyText(
                      context,
                      en: 'Could not load terminals',
                      zh: '无法加载终端列表',
                    ),
                    style: TextStyle(color: palette.text, fontSize: 15),
                  ),
                  ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 360),
                    child: _ConnectionErrorDetails(message: message),
                  ),
                ] else
                  SelectableText(
                    message,
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
