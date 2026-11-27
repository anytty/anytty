part of 'browser_session_screen.dart';

final class _BrowserConnectionBanner extends StatelessWidget {
  const _BrowserConnectionBanner({
    required this.message,
    required this.onRetry,
  });

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      decoration: BoxDecoration(
        color: palette.warning.withValues(alpha: 0.94),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: palette.warning),
      ),
      child: Row(
        children: [
          Icon(Icons.cloud_off_rounded, size: 18, color: palette.background),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              message,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.background,
                fontSize: 12,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          TextButton.icon(
            onPressed: onRetry,
            icon: Icon(
              Icons.refresh_rounded,
              size: 16,
              color: palette.background,
            ),
            label: Text(
              anyttyText(context, en: 'Retry', zh: '重试'),
              style: TextStyle(color: palette.background, fontSize: 12),
            ),
            style: TextButton.styleFrom(
              minimumSize: const Size(0, 40),
              padding: const EdgeInsets.symmetric(horizontal: 8),
              tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            ),
          ),
        ],
      ),
    );
  }
}

final class _BrowserStatus {
  const _BrowserStatus({
    required this.label,
    required this.icon,
    required this.color,
  });

  final String label;
  final IconData icon;
  final Color color;
}

_BrowserStatus _browserStatus(
  BuildContext context, {
  required BrowserSessionState state,
  required bool hasProxy,
  required bool dnsProxied,
}) {
  final palette = AnyttyPalette.of(context);
  final active = state.phase == BrowserSessionPhase.active && hasProxy;
  final blocked = state.phase == BrowserSessionPhase.blocked;
  if (active) {
    return _BrowserStatus(
      label: dnsProxied
          ? anyttyText(context, en: 'Tunnel · DNS remote', zh: '隧道 · DNS 已代理')
          : anyttyText(context, en: 'Tunnel active', zh: '隧道已连接'),
      icon: Icons.shield_outlined,
      color: palette.success,
    );
  }
  return _BrowserStatus(
    label: blocked
        ? anyttyText(context, en: 'Route unavailable', zh: '路由不可用')
        : _browserPhaseText(context, state.phase),
    icon: blocked ? Icons.route_rounded : Icons.more_horiz_rounded,
    color: blocked ? palette.warning : palette.muted,
  );
}

String _browserPhaseText(BuildContext context, BrowserSessionPhase phase) {
  return switch (phase) {
    BrowserSessionPhase.parked => anyttyText(context, en: 'Paused', zh: '已暂停'),
    BrowserSessionPhase.restoring => anyttyText(
      context,
      en: 'Loading page',
      zh: '正在加载页面',
    ),
    BrowserSessionPhase.active => anyttyText(
      context,
      en: 'Starting',
      zh: '正在启动',
    ),
    BrowserSessionPhase.parking => anyttyText(
      context,
      en: 'Closing page',
      zh: '正在关闭页面',
    ),
    BrowserSessionPhase.blocked => anyttyText(
      context,
      en: 'Route unavailable',
      zh: '路由不可用',
    ),
    BrowserSessionPhase.failed => anyttyText(
      context,
      en: 'Browser error',
      zh: '浏览器错误',
    ),
  };
}

final class _BrowserIconButton extends StatelessWidget {
  const _BrowserIconButton({
    required this.tooltip,
    required this.enabled,
    required this.onPressed,
    required this.icon,
  });

  final String tooltip;
  final bool enabled;
  final VoidCallback onPressed;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Tooltip(
      message: tooltip,
      child: IconButton(
        constraints: const BoxConstraints.tightFor(width: 44, height: 44),
        padding: const EdgeInsets.all(11),
        onPressed: enabled ? onPressed : null,
        color: palette.muted,
        icon: Icon(icon, size: 18),
      ),
    );
  }
}

final class _BrowserLoadingSurface extends StatelessWidget {
  const _BrowserLoadingSurface({required this.value});

  final double value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ColoredBox(
      color: palette.background,
      child: Center(
        child: SizedBox(
          width: 220,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const AnyttyBrandLoader(),
              const SizedBox(height: 24),
              Semantics(
                label: anyttyText(context, en: 'Loading page', zh: '正在加载页面'),
                value: '${(value * 100).round()}%',
                child: _BrowserLoadProgress(value: value),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _BrowserLoadProgress extends StatelessWidget {
  const _BrowserLoadProgress({required this.value});

  final double value;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 2,
      child: LinearProgressIndicator(
        value: value <= 0 ? null : value.clamp(0, 1),
        backgroundColor: palette.border.withValues(alpha: 0.35),
        color: palette.accent,
        minHeight: 2,
      ),
    );
  }
}
