import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../app/anytty_theme.dart';

/// 终端里所有"从花瓣菜单打开的面板"统一走这里。
///
/// 之前 settings / 尺寸 / 资源 / 快捷栏各用一套 showModalBottomSheet 或
/// showGeneralDialog，圆角、背景、手柄、关闭方式都不一样。这个基座把它们收成
/// 同一个视觉：底部升起、大圆角、毛玻璃、统一标题栏和关闭按钮。
///
/// 面板内部只负责内容，不再关心弹层样式；新增面板只要调用
/// [showAnyttyPanel] 就自动和现有面板一致。
Future<T?> showAnyttyPanel<T>({
  required BuildContext context,
  required Widget child,
  String? title,
  IconData? icon,
  double heightFactor = 0.86,
  bool blurred = true,
}) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<T>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    barrierColor: palette.overlay,
    backgroundColor: Colors.transparent,
    builder: (context) => AnyttyPanel(
      title: title,
      icon: icon,
      heightFactor: heightFactor,
      blurred: blurred,
      child: child,
    ),
  );
}

/// 面板容器：统一的圆角、毛玻璃和标题栏。
final class AnyttyPanel extends StatelessWidget {
  const AnyttyPanel({
    super.key,
    required this.child,
    this.title,
    this.icon,
    this.heightFactor = 0.86,
    this.blurred = true,
  });

  final Widget child;
  final String? title;
  final IconData? icon;
  final double heightFactor;
  final bool blurred;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final media = MediaQuery.of(context);
    // 键盘弹起时面板要跟着收缩，否则输入框会被挡住。
    final available = media.size.height - media.viewInsets.bottom;
    final height = (available * heightFactor).clamp(160.0, available);
    final surface = ColoredBox(
      color: blurred
          ? palette.surface.withValues(alpha: 0.94)
          : palette.surface,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _Grabber(color: palette.border),
          if (title != null)
            _PanelHeader(
              title: title!,
              icon: icon,
              onClose: () => Navigator.of(context).maybePop(),
            ),
          Expanded(child: child),
        ],
      ),
    );
    return SizedBox(
      height: height,
      child: ClipRRect(
        borderRadius: const BorderRadius.vertical(top: Radius.circular(18)),
        child: blurred
            ? BackdropFilter(
                filter: ui.ImageFilter.blur(sigmaX: 24, sigmaY: 24),
                child: surface,
              )
            : surface,
      ),
    );
  }
}

final class _Grabber extends StatelessWidget {
  const _Grabber({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: 8, bottom: 4),
      child: Center(
        child: Container(
          width: 36,
          height: 4,
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
      ),
    );
  }
}

final class _PanelHeader extends StatelessWidget {
  const _PanelHeader({
    required this.title,
    required this.onClose,
    this.icon,
  });

  final String title;
  final IconData? icon;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 4, 8, 8),
      child: Row(
        children: [
          if (icon != null) ...[
            Icon(icon, size: 18, color: palette.accent),
            const SizedBox(width: 8),
          ],
          Expanded(
            child: Text(
              title,
              style: text.titleSmall?.copyWith(
                color: palette.text,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          IconButton(
            onPressed: onClose,
            icon: const Icon(LucideIcons.x, size: 18),
            color: palette.muted,
            tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
          ),
        ],
      ),
    );
  }
}
