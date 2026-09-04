import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'app_color_preferences.dart';

@immutable
final class AnyttyPalette extends ThemeExtension<AnyttyPalette> {
  const AnyttyPalette({
    required this.background,
    required this.surface,
    required this.surfaceRaised,
    required this.border,
    required this.borderStrong,
    required this.text,
    required this.muted,
    required this.faint,
    required this.accent,
    required this.accentText,
    required this.success,
    required this.warning,
    required this.danger,
    required this.overlay,
  });

  final Color background;
  final Color surface;
  final Color surfaceRaised;
  final Color border;
  final Color borderStrong;
  final Color text;
  final Color muted;
  final Color faint;
  final Color accent;
  final Color accentText;
  final Color success;
  final Color warning;
  final Color danger;
  final Color overlay;

  // Semantic aliases used by the product UI. Keep the legacy border names
  // for Terminal and older feature code while new surfaces use these tokens.
  Color get track => border;
  Color get strong => text;

  static final light = _paletteFor(
    Brightness.light,
    AppColorPreferences.defaults,
  );

  static final dark = _paletteFor(
    Brightness.dark,
    AppColorPreferences.defaults,
  );

  static AnyttyPalette of(BuildContext context) =>
      Theme.of(context).extension<AnyttyPalette>() ?? light;

  @override
  AnyttyPalette copyWith({
    Color? background,
    Color? surface,
    Color? surfaceRaised,
    Color? border,
    Color? borderStrong,
    Color? text,
    Color? muted,
    Color? faint,
    Color? accent,
    Color? accentText,
    Color? success,
    Color? warning,
    Color? danger,
    Color? overlay,
  }) => AnyttyPalette(
    background: background ?? this.background,
    surface: surface ?? this.surface,
    surfaceRaised: surfaceRaised ?? this.surfaceRaised,
    border: border ?? this.border,
    borderStrong: borderStrong ?? this.borderStrong,
    text: text ?? this.text,
    muted: muted ?? this.muted,
    faint: faint ?? this.faint,
    accent: accent ?? this.accent,
    accentText: accentText ?? this.accentText,
    success: success ?? this.success,
    warning: warning ?? this.warning,
    danger: danger ?? this.danger,
    overlay: overlay ?? this.overlay,
  );

  @override
  AnyttyPalette lerp(covariant AnyttyPalette? other, double t) {
    if (other == null) return this;
    return AnyttyPalette(
      background: Color.lerp(background, other.background, t)!,
      surface: Color.lerp(surface, other.surface, t)!,
      surfaceRaised: Color.lerp(surfaceRaised, other.surfaceRaised, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
      text: Color.lerp(text, other.text, t)!,
      muted: Color.lerp(muted, other.muted, t)!,
      faint: Color.lerp(faint, other.faint, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      accentText: Color.lerp(accentText, other.accentText, t)!,
      success: Color.lerp(success, other.success, t)!,
      warning: Color.lerp(warning, other.warning, t)!,
      danger: Color.lerp(danger, other.danger, t)!,
      overlay: Color.lerp(overlay, other.overlay, t)!,
    );
  }
}

abstract final class AnyttyMotion {
  static const quick = Duration(milliseconds: 120);
  static const standard = Duration(milliseconds: 200);
  static const routeEnter = Duration(milliseconds: 200);
  static const routeExit = Duration(milliseconds: 200);
  static const emphasized = Cubic(0.16, 1, 0.3, 1);

  static bool disabled(BuildContext context) =>
      MediaQuery.maybeOf(context)?.disableAnimations ?? false;

  static Duration resolve(BuildContext context, Duration duration) =>
      disabled(context) ? Duration.zero : duration;
}

final class AnyttyPageTransitionsBuilder extends PageTransitionsBuilder {
  const AnyttyPageTransitionsBuilder();

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) => child;
}

ThemeData anyttyTheme(
  Brightness brightness, [
  AppColorPreferences colors = AppColorPreferences.defaults,
]) {
  final dark = brightness == Brightness.dark;
  final palette = _paletteFor(brightness, colors);
  final accentContainer = _mix(
    palette.background,
    palette.accent,
    dark ? 0.22 : 0.14,
  );
  final onAccentContainer = _bestForeground(accentContainer);
  final scheme = dark
      ? ColorScheme.dark(
          primary: palette.accent,
          onPrimary: palette.accentText,
          primaryContainer: accentContainer,
          onPrimaryContainer: onAccentContainer,
          secondary: palette.muted,
          onSecondary: _bestForeground(palette.muted),
          secondaryContainer: palette.surfaceRaised,
          onSecondaryContainer: palette.text,
          error: palette.danger,
          onError: _bestForeground(palette.danger),
          surface: palette.surface,
          onSurface: palette.text,
          outline: palette.borderStrong,
          outlineVariant: palette.border,
          surfaceContainerLowest: palette.background,
          surfaceContainerLow: palette.surface,
          surfaceContainer: palette.surfaceRaised,
        )
      : ColorScheme.light(
          primary: palette.accent,
          onPrimary: palette.accentText,
          primaryContainer: accentContainer,
          onPrimaryContainer: onAccentContainer,
          secondary: palette.muted,
          onSecondary: _bestForeground(palette.muted),
          secondaryContainer: palette.surfaceRaised,
          onSecondaryContainer: palette.text,
          error: palette.danger,
          onError: _bestForeground(palette.danger),
          surface: palette.surface,
          onSurface: palette.text,
          outline: palette.borderStrong,
          outlineVariant: palette.border,
          surfaceContainerLowest: palette.background,
          surfaceContainerLow: palette.surface,
          surfaceContainer: palette.surfaceRaised,
        );
  const radius = BorderRadius.all(Radius.circular(14));
  final seed = ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    scaffoldBackgroundColor: palette.background,
    fontFamily: anyttyUiFontFamily,
    extensions: [palette],
    visualDensity: VisualDensity.standard,
    splashFactory: NoSplash.splashFactory,
    pageTransitionsTheme: const PageTransitionsTheme(
      builders: {
        TargetPlatform.android: AnyttyPageTransitionsBuilder(),
        TargetPlatform.iOS: AnyttyPageTransitionsBuilder(),
        TargetPlatform.macOS: AnyttyPageTransitionsBuilder(),
        TargetPlatform.linux: AnyttyPageTransitionsBuilder(),
        TargetPlatform.windows: AnyttyPageTransitionsBuilder(),
        TargetPlatform.fuchsia: AnyttyPageTransitionsBuilder(),
      },
    ),
  );
  final base = seed.copyWith(textTheme: _appTextTheme(seed.textTheme, palette));
  return base.copyWith(
    appBarTheme: AppBarTheme(
      toolbarHeight: 56,
      backgroundColor: palette.background,
      foregroundColor: palette.text,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      scrolledUnderElevation: 0,
      systemOverlayStyle: anyttySystemUiOverlayStyle(
        brightness,
        palette: palette,
      ),
      titleTextStyle: base.textTheme.titleLarge?.copyWith(
        color: palette.text,
        fontSize: 22,
        height: 28 / 22,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.7,
      ),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: palette.surface,
      surfaceTintColor: Colors.transparent,
      modalBackgroundColor: palette.surface,
      modalBarrierColor: palette.overlay,
      showDragHandle: true,
      dragHandleColor: palette.borderStrong,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
    ),
    cardTheme: CardThemeData(
      color: palette.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 2,
      shadowColor: Colors.black.withValues(alpha: dark ? 0.30 : 0.10),
      margin: EdgeInsets.zero,
      shape: RoundedRectangleBorder(borderRadius: radius),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: palette.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 2,
      shadowColor: Colors.black.withValues(alpha: dark ? 0.30 : 0.10),
      shape: RoundedRectangleBorder(borderRadius: radius),
      titleTextStyle: base.textTheme.titleMedium?.copyWith(color: palette.text),
      contentTextStyle: base.textTheme.bodyMedium?.copyWith(
        color: palette.text,
      ),
    ),
    dividerTheme: DividerThemeData(color: palette.track, thickness: 1),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: palette.surfaceRaised,
      border: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: Colors.transparent),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: Colors.transparent),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: radius,
        borderSide: BorderSide(color: Colors.transparent),
      ),
      labelStyle: TextStyle(
        color: palette.muted,
        fontSize: 14.5,
        height: 18 / 14.5,
        letterSpacing: 0.3,
      ),
      floatingLabelStyle: TextStyle(
        color: palette.muted,
        fontSize: 14.5,
        height: 18 / 14.5,
        letterSpacing: 0.3,
      ),
      hintStyle: TextStyle(
        color: palette.muted,
        fontSize: 14.5,
        height: 18 / 14.5,
        letterSpacing: 0.3,
      ),
      helperStyle: TextStyle(
        color: palette.muted,
        fontSize: 14.5,
        height: 18 / 14.5,
        letterSpacing: 0.3,
      ),
    ),
    searchBarTheme: SearchBarThemeData(
      elevation: const WidgetStatePropertyAll(0),
      backgroundColor: WidgetStatePropertyAll(palette.surfaceRaised),
      side: const WidgetStatePropertyAll(BorderSide(color: Colors.transparent)),
      shape: const WidgetStatePropertyAll(
        RoundedRectangleBorder(borderRadius: radius),
      ),
      textStyle: WidgetStatePropertyAll(
        base.textTheme.bodyMedium?.copyWith(color: palette.text),
      ),
      hintStyle: WidgetStatePropertyAll(
        base.textTheme.bodyMedium?.copyWith(color: palette.muted),
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: ButtonStyle(
        minimumSize: const WidgetStatePropertyAll(Size.square(44)),
        padding: const WidgetStatePropertyAll(EdgeInsets.all(10)),
        shape: const WidgetStatePropertyAll(CircleBorder()),
        foregroundColor: WidgetStatePropertyAll(palette.strong),
        overlayColor: WidgetStatePropertyAll(Color(0x16000000)),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: ButtonStyle(
        elevation: const WidgetStatePropertyAll(0),
        minimumSize: const WidgetStatePropertyAll(Size(44, 44)),
        shape: WidgetStatePropertyAll(StadiumBorder()),
        textStyle: WidgetStatePropertyAll(
          base.textTheme.labelLarge?.copyWith(fontWeight: FontWeight.w600),
        ),
      ),
    ),
    outlinedButtonTheme: const OutlinedButtonThemeData(
      style: ButtonStyle(
        minimumSize: WidgetStatePropertyAll(Size(44, 44)),
        shape: WidgetStatePropertyAll(StadiumBorder()),
      ),
    ),
    textButtonTheme: const TextButtonThemeData(
      style: ButtonStyle(
        minimumSize: WidgetStatePropertyAll(Size(44, 44)),
        shape: WidgetStatePropertyAll(StadiumBorder()),
      ),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: palette.surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(borderRadius: radius),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        minimumSize: const WidgetStatePropertyAll(Size(44, 44)),
        elevation: const WidgetStatePropertyAll(1),
        shadowColor: WidgetStatePropertyAll(
          Colors.black.withValues(alpha: dark ? 0.22 : 0.10),
        ),
        side: const WidgetStatePropertyAll(
          BorderSide(color: Colors.transparent),
        ),
        shape: const WidgetStatePropertyAll(StadiumBorder()),
      ),
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: palette.surfaceRaised,
      contentTextStyle: base.textTheme.bodyMedium?.copyWith(
        color: palette.text,
      ),
      behavior: SnackBarBehavior.floating,
      elevation: 1,
      insetPadding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
      shape: RoundedRectangleBorder(
        borderRadius: const BorderRadius.all(Radius.circular(14)),
      ),
    ),
  );
}

const anyttyUiFontFamily = 'InterTight';

TextTheme _appTextTheme(TextTheme theme, AnyttyPalette palette) =>
    theme.copyWith(
      displayLarge: _uiTitle(theme.displayLarge, palette, 22, 28, -0.7),
      displayMedium: _uiTitle(theme.displayMedium, palette, 22, 28, -0.7),
      displaySmall: _uiTitle(theme.displaySmall, palette, 22, 28, -0.7),
      headlineLarge: _uiTitle(theme.headlineLarge, palette, 22, 28, -0.7),
      headlineMedium: _uiTitle(theme.headlineMedium, palette, 22, 28, -0.7),
      headlineSmall: _uiSection(theme.headlineSmall, palette),
      titleLarge: _uiTitle(theme.titleLarge, palette, 22, 28, -0.7),
      titleMedium: _uiSection(theme.titleMedium, palette),
      titleSmall: _uiSection(theme.titleSmall, palette),
      bodyLarge: _uiBody(theme.bodyLarge, palette),
      bodyMedium: _uiBody(theme.bodyMedium, palette),
      bodySmall: _uiBody(theme.bodySmall, palette),
      labelLarge: _uiBody(theme.labelLarge, palette, weight: FontWeight.w600),
      labelMedium: _uiBody(theme.labelMedium, palette, weight: FontWeight.w600),
      labelSmall: _uiBody(theme.labelSmall, palette, weight: FontWeight.w600),
    );

TextStyle _uiTitle(
  TextStyle? base,
  AnyttyPalette palette,
  double fontSize,
  double lineHeight,
  double letterSpacing,
) => (base ?? const TextStyle()).copyWith(
  color: palette.text,
  fontFamily: anyttyUiFontFamily,
  fontSize: fontSize,
  height: lineHeight / fontSize,
  fontWeight: FontWeight.w600,
  letterSpacing: letterSpacing,
);

TextStyle _uiSection(TextStyle? base, AnyttyPalette palette) =>
    (base ?? const TextStyle()).copyWith(
      color: palette.text,
      fontFamily: anyttyUiFontFamily,
      fontSize: 19,
      height: 24 / 19,
      fontWeight: FontWeight.w600,
      letterSpacing: 0.6,
    );

TextStyle _uiBody(
  TextStyle? base,
  AnyttyPalette palette, {
  FontWeight weight = FontWeight.w400,
}) => (base ?? const TextStyle()).copyWith(
  color: palette.text,
  fontFamily: anyttyUiFontFamily,
  fontSize: 14.5,
  height: 18 / 14.5,
  fontWeight: weight,
  letterSpacing: 0.3,
);

SystemUiOverlayStyle anyttySystemUiOverlayStyle(
  Brightness brightness, {
  AnyttyPalette? palette,
}) {
  final background =
      palette?.background ??
      (brightness == Brightness.dark
          ? AnyttyPalette.dark.background
          : AnyttyPalette.light.background);
  final surfaceBrightness = ThemeData.estimateBrightnessForColor(background);
  final iconBrightness = surfaceBrightness == Brightness.dark
      ? Brightness.light
      : Brightness.dark;
  return SystemUiOverlayStyle(
    statusBarColor: background,
    statusBarIconBrightness: iconBrightness,
    statusBarBrightness: surfaceBrightness,
    systemNavigationBarColor: background,
    systemNavigationBarIconBrightness: iconBrightness,
    systemNavigationBarDividerColor: background,
    systemNavigationBarContrastEnforced: false,
  );
}

AnyttyPalette _paletteFor(Brightness brightness, AppColorPreferences colors) {
  final dark = brightness == Brightness.dark;
  final background = dark ? colors.darkBackground : const Color(0xfff2f5f3);
  final surface = dark ? colors.darkSurface : const Color(0xfffbfdfc);
  final text = _bestForeground(surface);
  final accent = _ensureContrast(colors.accent, background, 3);
  return AnyttyPalette(
    background: background,
    surface: surface,
    surfaceRaised: _mix(surface, text, dark ? 0.06 : 0.05),
    border: _mix(surface, text, dark ? 0.11 : 0.10),
    borderStrong: _mix(surface, text, dark ? 0.20 : 0.18),
    text: text,
    muted: _mix(text, surface, dark ? 0.40 : 0.34),
    faint: _mix(text, surface, dark ? 0.58 : 0.52),
    accent: accent,
    accentText: _bestForeground(accent),
    success: dark ? const Color(0xff4ade80) : const Color(0xff15803d),
    warning: dark ? const Color(0xfffbbf24) : const Color(0xffb45309),
    danger: dark ? const Color(0xfff87171) : const Color(0xffb91c1c),
    overlay: const Color(0x3d000000),
  );
}

Color _bestForeground(Color background) {
  const dark = Color(0xff081916);
  const light = Color(0xfff4f8f7);
  return _contrastRatio(background, dark) >= _contrastRatio(background, light)
      ? dark
      : light;
}

Color _ensureContrast(Color color, Color background, double minimum) {
  if (_contrastRatio(color, background) >= minimum) return color;
  final target =
      ThemeData.estimateBrightnessForColor(background) == Brightness.light
      ? const Color(0xff000000)
      : const Color(0xffffffff);
  for (var step = 1; step <= 20; step += 1) {
    final candidate = _mix(color, target, step / 20);
    if (_contrastRatio(candidate, background) >= minimum) return candidate;
  }
  return target;
}

double _contrastRatio(Color first, Color second) {
  final lighter = mathMax(first.computeLuminance(), second.computeLuminance());
  final darker = mathMin(first.computeLuminance(), second.computeLuminance());
  return (lighter + 0.05) / (darker + 0.05);
}

double mathMax(double first, double second) => first > second ? first : second;

double mathMin(double first, double second) => first < second ? first : second;

Color _mix(Color first, Color second, double amount) =>
    Color.lerp(first, second, amount)!;
