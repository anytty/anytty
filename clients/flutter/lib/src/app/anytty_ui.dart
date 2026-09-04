import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'anytty_theme.dart';

abstract final class AnyttyUi {
  static const cardRadius = 16.0;
  static const controlSize = 44.0;

  static double controlHeight(BuildContext context) {
    final scaler = MediaQuery.textScalerOf(context);
    final bodyLineHeight = scaler.scale(14.5) * (18 / 14.5);
    return math.max(controlSize, bodyLineHeight + 12);
  }

  static double appBarHeight(
    BuildContext context, {
    double minimum = kToolbarHeight,
    int subtitleLines = 0,
    bool eyebrow = false,
  }) {
    final scaler = MediaQuery.textScalerOf(context);
    final titleHeight = scaler.scale(22) * (28 / 22);
    final subtitleHeight = scaler.scale(14.5) * (18 / 14.5) * subtitleLines;
    final eyebrowHeight = eyebrow ? scaler.scale(14.5) * (18 / 14.5) + 4 : 0;
    final requiredHeight =
        titleHeight +
        eyebrowHeight +
        subtitleHeight +
        (subtitleLines > 0 ? 4 : 0) +
        12;
    return math.max(minimum, requiredHeight);
  }

  static TextStyle title(BuildContext context) =>
      Theme.of(context).textTheme.titleLarge!;

  static TextStyle sectionTitle(BuildContext context) =>
      Theme.of(context).textTheme.titleMedium!;

  static TextStyle body(BuildContext context) =>
      Theme.of(context).textTheme.bodyMedium!;

  static TextStyle muted(BuildContext context) =>
      body(context).copyWith(color: AnyttyPalette.of(context).muted);

  static BoxDecoration cardDecoration(
    BuildContext context, {
    double radius = cardRadius,
    int depth = 1,
    Color? color,
  }) {
    final palette = AnyttyPalette.of(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final surface = color ?? palette.surface;
    return BoxDecoration(
      color: surface,
      borderRadius: BorderRadius.circular(radius),
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(alpha: dark ? 0.28 : 0.11),
          blurRadius: 12 + (depth * 4),
          offset: Offset(0, 4 + depth.toDouble()),
        ),
        BoxShadow(
          color: Colors.white.withValues(alpha: dark ? 0.035 : 0.72),
          blurRadius: 7,
          offset: const Offset(0, -2),
        ),
      ],
    );
  }

  static BoxDecoration controlDecoration(
    BuildContext context, {
    bool selected = false,
    bool enabled = true,
    Color? color,
  }) {
    final palette = AnyttyPalette.of(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final base = color ?? (selected ? palette.accent : palette.surfaceRaised);
    return BoxDecoration(
      gradient: LinearGradient(
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
        colors: [
          Color.alphaBlend(
            Colors.white.withValues(
              alpha: enabled ? (dark ? 0.10 : 0.58) : (dark ? 0.04 : 0.24),
            ),
            base,
          ),
          base,
        ],
      ),
      shape: BoxShape.circle,
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(
            alpha: enabled ? (dark ? 0.26 : 0.12) : (dark ? 0.10 : 0.04),
          ),
          blurRadius: 9,
          offset: const Offset(0, 4),
        ),
        BoxShadow(
          color: Colors.white.withValues(
            alpha: enabled ? (dark ? 0.05 : 0.78) : (dark ? 0.02 : 0.30),
          ),
          blurRadius: 6,
          offset: const Offset(0, -2),
        ),
      ],
    );
  }

  static BoxDecoration pillDecoration(
    BuildContext context, {
    bool selected = false,
    bool enabled = true,
    Color? color,
    double radius = 22,
  }) {
    final palette = AnyttyPalette.of(context);
    final dark = Theme.of(context).brightness == Brightness.dark;
    final base = color ?? (selected ? palette.accent : palette.surfaceRaised);
    return BoxDecoration(
      gradient: LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [
          Color.alphaBlend(
            Colors.white.withValues(
              alpha: enabled ? (dark ? 0.10 : 0.58) : (dark ? 0.04 : 0.24),
            ),
            base,
          ),
          base,
        ],
      ),
      borderRadius: BorderRadius.circular(radius),
      boxShadow: [
        BoxShadow(
          color: Colors.black.withValues(
            alpha: enabled ? (dark ? 0.22 : 0.10) : (dark ? 0.08 : 0.04),
          ),
          blurRadius: 9,
          offset: const Offset(0, 4),
        ),
        BoxShadow(
          color: Colors.white.withValues(
            alpha: enabled ? (dark ? 0.04 : 0.72) : (dark ? 0.02 : 0.30),
          ),
          blurRadius: 6,
          offset: const Offset(0, -2),
        ),
      ],
    );
  }
}

final class AnyttyCard extends StatelessWidget {
  const AnyttyCard({
    super.key,
    required this.child,
    this.padding = EdgeInsets.zero,
    this.radius = AnyttyUi.cardRadius,
    this.depth = 1,
    this.color,
    this.onTap,
  });

  final Widget child;
  final EdgeInsetsGeometry padding;
  final double radius;
  final int depth;
  final Color? color;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final shape = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radius),
    );
    final content = Padding(padding: padding, child: child);
    final material = Material(color: Colors.transparent, child: content);
    return DecoratedBox(
      decoration: AnyttyUi.cardDecoration(
        context,
        radius: radius,
        depth: depth,
        color: color,
      ),
      child: onTap == null
          ? material
          : Material(
              color: Colors.transparent,
              child: InkWell(customBorder: shape, onTap: onTap, child: content),
            ),
    );
  }
}

final class AnyttyIconButton extends StatelessWidget {
  const AnyttyIconButton({
    super.key,
    required this.icon,
    required this.tooltip,
    required this.onPressed,
    this.size = AnyttyUi.controlSize,
    this.iconSize = 20,
    this.selected = false,
    this.iconColor,
    this.child,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback? onPressed;
  final double size;
  final double iconSize;
  final bool selected;
  final Color? iconColor;
  final Widget? child;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final button = SizedBox.square(
      dimension: size,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onPressed,
          child: DecoratedBox(
            decoration: AnyttyUi.controlDecoration(
              context,
              selected: selected,
              enabled: onPressed != null,
            ),
            child:
                child ??
                Icon(
                  icon,
                  size: iconSize,
                  color: onPressed == null
                      ? palette.muted.withValues(alpha: 0.45)
                      : iconColor ??
                            (selected ? palette.accentText : palette.strong),
                ),
          ),
        ),
      ),
    );
    return Tooltip(message: tooltip, child: button);
  }
}

final class AnyttyPillButton extends StatelessWidget {
  const AnyttyPillButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
    this.color,
    this.foregroundColor,
    this.outlined = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final Color? color;
  final Color? foregroundColor;
  final bool outlined;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final base = color ?? palette.surfaceRaised;
    final height = AnyttyUi.controlHeight(context);
    final style = outlined
        ? OutlinedButton.styleFrom(
            backgroundColor: Colors.transparent,
            foregroundColor: foregroundColor ?? palette.strong,
            disabledForegroundColor: palette.muted.withValues(alpha: 0.45),
            shadowColor: Colors.transparent,
            side: BorderSide.none,
            minimumSize: const Size(0, AnyttyUi.controlSize),
          )
        : FilledButton.styleFrom(
            backgroundColor: Colors.transparent,
            foregroundColor: foregroundColor ?? palette.strong,
            disabledBackgroundColor: Colors.transparent,
            disabledForegroundColor: palette.muted.withValues(alpha: 0.45),
            shadowColor: Colors.transparent,
            minimumSize: const Size(0, AnyttyUi.controlSize),
          );
    final button = icon == null
        ? (outlined
              ? OutlinedButton(
                  onPressed: onPressed,
                  style: style,
                  child: Text(label),
                )
              : FilledButton(
                  onPressed: onPressed,
                  style: style,
                  child: Text(label),
                ))
        : (outlined
              ? OutlinedButton.icon(
                  onPressed: onPressed,
                  style: style,
                  icon: Icon(icon, size: 18),
                  label: Text(label),
                )
              : FilledButton.icon(
                  onPressed: onPressed,
                  style: style,
                  icon: Icon(icon, size: 18),
                  label: Text(label),
                ));
    return DecoratedBox(
      decoration: AnyttyUi.pillDecoration(
        context,
        color: base,
        enabled: onPressed != null,
      ),
      child: SizedBox(height: height, child: button),
    );
  }
}

final class AnyttyDivider extends StatelessWidget {
  const AnyttyDivider({super.key, this.indent = 0, this.endIndent = 0});

  final double indent;
  final double endIndent;

  @override
  Widget build(BuildContext context) => Divider(
    height: 1,
    thickness: 1,
    indent: indent,
    endIndent: endIndent,
    color: AnyttyPalette.of(context).track,
  );
}

final class AnyttyDialog extends StatelessWidget {
  const AnyttyDialog({
    super.key,
    required this.title,
    required this.content,
    required this.actions,
  });

  final Widget title;
  final Widget content;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) => Dialog(
    backgroundColor: Colors.transparent,
    insetPadding: const EdgeInsets.all(24),
    child: AnyttyCard(
      radius: 16,
      depth: 2,
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          DefaultTextStyle(style: AnyttyUi.sectionTitle(context), child: title),
          const SizedBox(height: 10),
          DefaultTextStyle(style: AnyttyUi.body(context), child: content),
          const SizedBox(height: 18),
          Wrap(
            alignment: WrapAlignment.end,
            spacing: 8,
            runSpacing: 8,
            children: actions,
          ),
        ],
      ),
    ),
  );
}
