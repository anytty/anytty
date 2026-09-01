import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/app_appearance.dart';
import '../../../app/app_color_preferences.dart';
import '../../../app/app_language.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/background_preferences.dart';
import '../../../app/providers.dart';
import '../../../native/background_platform.dart';
import '../../terminal/domain/terminal_petal_menu_preferences.dart';
import '../../terminal/domain/terminal_settings.dart';

final class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final palette = AnyttyPalette.of(context);
    final appearance =
        ref.watch(appAppearanceProvider).valueOrNull ?? AppAppearance.dark;
    final colorPreferences =
        ref.watch(appColorPreferencesProvider).valueOrNull ??
        AppColorPreferences.defaults;
    final language =
        ref.watch(appLanguageProvider).valueOrNull ?? AppLanguage.system;
    final terminalSettings =
        ref.watch(terminalSettingsProvider).valueOrNull ??
        defaultTerminalSettings;
    final petalMenuPreferences =
        ref.watch(terminalPetalMenuPreferencesProvider).valueOrNull ??
        TerminalPetalMenuPreferences.defaults;
    final backgroundPreferences =
        ref.watch(backgroundPreferencesProvider).valueOrNull ??
        BackgroundPreferences.defaults;
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: anyttyText(context, en: 'Back to devices', zh: '返回设备列表'),
          onPressed: context.pop,
          icon: const Icon(Icons.chevron_left_rounded),
        ),
        titleSpacing: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              anyttyText(context, en: 'Settings', zh: '设置'),
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            Text(
              anyttyText(context, en: 'Device access', zh: '设备与终端'),
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500),
            ),
          ],
        ),
      ),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 20, 16, 24),
          children: [
            ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 576),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    anyttyText(context, en: 'APPEARANCE', zh: '外观'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  _AppearancePicker(
                    value: appearance,
                    onChanged: (value) => _save(context, ref, value),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: _SettingsNavigationRow(
                      key: const ValueKey('theme-color-settings-link'),
                      label: anyttyText(
                        context,
                        en: 'Theme colors',
                        zh: '主题颜色',
                      ),
                      status: colorHex(colorPreferences.accent),
                      icon: LucideIcons.palette,
                      onTap: () => context.push('/settings/theme-colors'),
                    ),
                  ),
                  const SizedBox(height: 24),
                  Text(
                    anyttyText(context, en: 'LANGUAGE', zh: '语言'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: _SettingsRow(
                      label: anyttyText(
                        context,
                        en: 'App language',
                        zh: '应用语言',
                      ),
                      excludeLabelSemantics: true,
                      child: _SettingsDropdown<AppLanguage>(
                        semanticsLabel: anyttyText(
                          context,
                          en: 'App language',
                          zh: '应用语言',
                        ),
                        value: language,
                        values: AppLanguage.values,
                        labelFor: (value) => switch (value) {
                          AppLanguage.system => anyttyText(
                            context,
                            en: 'System',
                            zh: '跟随系统',
                          ),
                          AppLanguage.english => 'English',
                          AppLanguage.simplifiedChinese => '简体中文',
                        },
                        onChanged: (value) =>
                            _saveLanguage(context, ref, value),
                      ),
                    ),
                  ),
                  const SizedBox(height: 24),
                  Text(
                    anyttyText(context, en: 'TERMINAL', zh: '终端'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  _TerminalSettingsPreview(settings: terminalSettings),
                  const SizedBox(height: 12),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      children: [
                        _SettingsNavigationRow(
                          key: const ValueKey('petal-menu-settings-link'),
                          label: anyttyText(
                            context,
                            en: 'Petal menu',
                            zh: '花瓣菜单',
                          ),
                          status: petalMenuPreferences.enabled
                              ? anyttyText(
                                  context,
                                  en: '${petalMenuPreferences.visibleActionCount} active',
                                  zh: '已启用 ${petalMenuPreferences.visibleActionCount} 个',
                                )
                              : anyttyText(context, en: 'Off', zh: '已关闭'),
                          icon: LucideIcons.flower2,
                          onTap: () => context.push('/settings/petal-menu'),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Font size',
                            zh: '字体大小',
                          ),
                          child: _FontSizeStepper(
                            value: terminalSettings.fontSize,
                            onChanged: (value) => _saveTerminal(
                              context,
                              ref,
                              terminalSettings.copyWith(fontSize: value),
                            ),
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(context, en: 'Font', zh: '字体'),
                          child: _FontPreviewButton(
                            value: terminalSettings.fontFamily,
                            onPressed: () async {
                              final value = await _showFontPicker(
                                context,
                                terminalSettings.fontFamily,
                              );
                              if (value == null || !context.mounted) return;
                              await _saveTerminal(
                                context,
                                ref,
                                terminalSettings.copyWith(fontFamily: value),
                              );
                            },
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Terminal theme',
                            zh: '终端主题',
                          ),
                          child: _ThemePreviewButton(
                            value: terminalSettings.theme,
                            onPressed: () async {
                              final value = await _showThemePicker(
                                context,
                                terminalSettings.themeId,
                              );
                              if (value == null || !context.mounted) return;
                              await _saveTerminal(
                                context,
                                ref,
                                terminalSettings.copyWith(themeId: value),
                              );
                            },
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Keyboard',
                            zh: '键盘模式',
                          ),
                          excludeLabelSemantics: true,
                          child: _SettingsDropdown<TerminalKeyboardMode>(
                            semanticsLabel: anyttyText(
                              context,
                              en: 'Keyboard mode',
                              zh: '键盘模式',
                            ),
                            value: terminalSettings.keyboardMode,
                            values: TerminalKeyboardMode.values,
                            labelFor: (value) => switch (value) {
                              TerminalKeyboardMode.automatic => anyttyText(
                                context,
                                en: 'Auto',
                                zh: '自动',
                              ),
                              TerminalKeyboardMode.resize => anyttyText(
                                context,
                                en: 'Resize',
                                zh: '调整尺寸',
                              ),
                              TerminalKeyboardMode.shift => anyttyText(
                                context,
                                en: 'Shift',
                                zh: '整体上移',
                              ),
                            },
                            onChanged: (value) => _saveTerminal(
                              context,
                              ref,
                              terminalSettings.copyWith(keyboardMode: value),
                            ),
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Cursor blink',
                            zh: '光标闪烁',
                          ),
                          excludeLabelSemantics: true,
                          child: Semantics(
                            label: anyttyText(
                              context,
                              en: 'Cursor blink',
                              zh: '光标闪烁',
                            ),
                            child: Switch.adaptive(
                              value: terminalSettings.cursorBlink,
                              activeTrackColor: palette.accent,
                              onChanged: (value) => _saveTerminal(
                                context,
                                ref,
                                terminalSettings.copyWith(cursorBlink: value),
                              ),
                            ),
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Auto-acquire resize owner',
                            zh: '自动接管终端尺寸',
                          ),
                          excludeLabelSemantics: true,
                          child: Semantics(
                            label: anyttyText(
                              context,
                              en: 'Auto-acquire resize owner',
                              zh: '自动接管终端尺寸',
                            ),
                            child: Switch.adaptive(
                              value: terminalSettings.autoAcquireResizeOwner,
                              activeTrackColor: palette.accent,
                              onChanged: (value) => _saveTerminal(
                                context,
                                ref,
                                terminalSettings.copyWith(
                                  autoAcquireResizeOwner: value,
                                ),
                              ),
                            ),
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Scroll inertia',
                            zh: '滚动惯性',
                          ),
                          stacked: true,
                          excludeLabelSemantics: true,
                          child: Row(
                            children: [
                              Expanded(
                                child: MergeSemantics(
                                  child: Semantics(
                                    label: anyttyText(
                                      context,
                                      en: 'Scroll inertia',
                                      zh: '滚动惯性',
                                    ),
                                    child: Slider.adaptive(
                                      value: terminalSettings.scrollInertia
                                          .toDouble(),
                                      min: 0,
                                      max: 100,
                                      divisions: 20,
                                      onChanged: (value) => _saveTerminal(
                                        context,
                                        ref,
                                        terminalSettings.copyWith(
                                          scrollInertia: value.round(),
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                              SizedBox(
                                width: 32,
                                child: ExcludeSemantics(
                                  child: Text(
                                    '${terminalSettings.scrollInertia}',
                                    textAlign: TextAlign.right,
                                    style: const TextStyle(
                                      fontSize: 12,
                                      fontWeight: FontWeight.w600,
                                    ),
                                  ),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 24),
                  Text(
                    anyttyText(context, en: 'BACKGROUND', zh: '后台'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      children: [
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Background connections',
                            zh: '后台保持连接',
                          ),
                          excludeLabelSemantics: true,
                          child: Semantics(
                            label: anyttyText(
                              context,
                              en: 'Background connections',
                              zh: '后台保持连接',
                            ),
                            child: Switch.adaptive(
                              value: backgroundPreferences.keepConnections,
                              activeTrackColor: palette.accent,
                              onChanged: (value) => _saveBackground(
                                context,
                                ref,
                                backgroundPreferences.copyWith(
                                  keepConnections: value,
                                ),
                              ),
                            ),
                          ),
                        ),
                        _SettingsDivider(color: palette.border),
                        _SettingsRow(
                          label: anyttyText(
                            context,
                            en: 'Terminal notifications',
                            zh: '终端通知',
                          ),
                          excludeLabelSemantics: true,
                          child: Semantics(
                            label: anyttyText(
                              context,
                              en: 'Terminal notifications',
                              zh: '终端通知',
                            ),
                            child: Switch.adaptive(
                              value: backgroundPreferences.notifications,
                              activeTrackColor: palette.accent,
                              onChanged: (value) => _setNotifications(
                                context,
                                ref,
                                backgroundPreferences,
                                value,
                              ),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _save(
    BuildContext context,
    WidgetRef ref,
    AppAppearance value,
  ) async {
    try {
      await ref.read(appAppearanceProvider.notifier).save(value);
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${anyttyText(context, en: 'Could not save appearance', zh: '无法保存外观设置')}: $error',
          ),
        ),
      );
    }
  }

  Future<void> _saveTerminal(
    BuildContext context,
    WidgetRef ref,
    TerminalSettings value,
  ) async {
    try {
      await ref.read(terminalSettingsProvider.notifier).save(value);
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${anyttyText(context, en: 'Could not save terminal settings', zh: '无法保存终端设置')}: $error',
          ),
        ),
      );
    }
  }

  Future<void> _saveLanguage(
    BuildContext context,
    WidgetRef ref,
    AppLanguage value,
  ) async {
    try {
      await ref.read(appLanguageProvider.notifier).save(value);
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${anyttyText(context, en: 'Could not save app language', zh: '无法保存应用语言')}: $error',
          ),
        ),
      );
    }
  }

  Future<void> _setNotifications(
    BuildContext context,
    WidgetRef ref,
    BackgroundPreferences current,
    bool enabled,
  ) async {
    if (enabled) {
      try {
        final allowed = await MethodChannelBackgroundPlatform.instance
            .requestNotificationAuthorization();
        if (!allowed) {
          if (!context.mounted) return;
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                anyttyText(
                  context,
                  en: 'Notifications are not allowed',
                  zh: '系统未允许通知权限',
                ),
              ),
            ),
          );
          return;
        }
      } catch (error) {
        if (!context.mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              '${anyttyText(context, en: 'Could not enable notifications', zh: '无法启用通知')}: $error',
            ),
          ),
        );
        return;
      }
    }
    if (!context.mounted) return;
    await _saveBackground(
      context,
      ref,
      current.copyWith(notifications: enabled),
    );
  }

  Future<void> _saveBackground(
    BuildContext context,
    WidgetRef ref,
    BackgroundPreferences value,
  ) async {
    try {
      await ref.read(backgroundPreferencesProvider.notifier).save(value);
    } catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            '${anyttyText(context, en: 'Could not save background settings', zh: '无法保存后台设置')}: $error',
          ),
        ),
      );
    }
  }
}

final class _SettingsDivider extends StatelessWidget {
  const _SettingsDivider({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) => Divider(height: 1, color: color);
}

final class _SettingsNavigationRow extends StatelessWidget {
  const _SettingsNavigationRow({
    super.key,
    required this.label,
    required this.status,
    required this.icon,
    required this.onTap,
  });

  final String label;
  final String status;
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: '$label, $status',
      onTap: onTap,
      excludeSemantics: true,
      child: InkWell(
        onTap: onTap,
        child: ConstrainedBox(
          constraints: const BoxConstraints(minHeight: 64),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Row(
              children: [
                Icon(icon, size: 20, color: palette.muted),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        label,
                        maxLines: 2,
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w500,
                          height: 1.25,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        status,
                        maxLines: 2,
                        style: TextStyle(
                          color: palette.muted,
                          fontSize: 12,
                          fontWeight: FontWeight.w500,
                          height: 1.25,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                Icon(LucideIcons.chevronRight, size: 18, color: palette.muted),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _TerminalSettingsPreview extends StatefulWidget {
  const _TerminalSettingsPreview({required this.settings});

  final TerminalSettings settings;

  @override
  State<_TerminalSettingsPreview> createState() =>
      _TerminalSettingsPreviewState();
}

final class _TerminalSettingsPreviewState
    extends State<_TerminalSettingsPreview>
    with SingleTickerProviderStateMixin {
  late final AnimationController _cursor;

  @override
  void initState() {
    super.initState();
    _cursor = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 260),
      value: 1,
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _pulseCursor(motionDisabled: AnyttyMotion.disabled(context));
  }

  @override
  void didUpdateWidget(_TerminalSettingsPreview oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.settings != oldWidget.settings) {
      _pulseCursor(motionDisabled: AnyttyMotion.disabled(context));
    }
  }

  @override
  void dispose() {
    _cursor.dispose();
    super.dispose();
  }

  void _pulseCursor({required bool motionDisabled}) {
    if (motionDisabled || !widget.settings.cursorBlink) {
      _cursor.stop();
      _cursor.value = 1;
      return;
    }
    _cursor.repeat(reverse: true, count: 2).whenComplete(() {
      if (mounted) _cursor.value = 1;
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final terminalTheme = widget.settings.theme;
    final background = _terminalColor(terminalTheme.background);
    final foreground = _terminalColor(terminalTheme.foreground);
    final green = _terminalColor(terminalTheme.ansi[2]);
    final cyan = _terminalColor(terminalTheme.ansi[6]);
    final yellow = _terminalColor(terminalTheme.ansi[3]);
    final previewFontSize = widget.settings.fontSize
        .toDouble()
        .clamp(10, 18)
        .toDouble();
    final lineStyle = TextStyle(
      color: foreground,
      fontFamily: widget.settings.fontFamily,
      fontSize: previewFontSize,
      height: 1.35,
      letterSpacing: 0,
    );
    final motionDisabled = AnyttyMotion.disabled(context);
    return Semantics(
      label:
          'Terminal preview, ${terminalTheme.label}, ${widget.settings.fontSize} point ${widget.settings.fontFamily}',
      child: AnimatedContainer(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.standard),
        curve: AnyttyMotion.emphasized,
        height: 148,
        clipBehavior: Clip.antiAlias,
        decoration: BoxDecoration(
          color: background,
          border: Border.all(color: palette.borderStrong),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              height: 32,
              padding: const EdgeInsets.symmetric(horizontal: 10),
              decoration: BoxDecoration(
                color: Color.alphaBlend(
                  foreground.withValues(alpha: 0.06),
                  background,
                ),
                border: Border(
                  bottom: BorderSide(color: foreground.withValues(alpha: 0.12)),
                ),
              ),
              child: Row(
                children: [
                  Container(
                    width: 7,
                    height: 7,
                    decoration: BoxDecoration(
                      color: green,
                      shape: BoxShape.circle,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      terminalTheme.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: foreground.withValues(alpha: 0.72),
                        fontFamily: widget.settings.fontFamily,
                        fontSize: 10,
                        letterSpacing: 0,
                      ),
                    ),
                  ),
                  Text(
                    '${widget.settings.fontSize} pt',
                    style: TextStyle(
                      color: foreground.withValues(alpha: 0.56),
                      fontFamily: widget.settings.fontFamily,
                      fontSize: 9,
                      letterSpacing: 0,
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(12, 9, 12, 8),
                child: FittedBox(
                  alignment: Alignment.topLeft,
                  fit: BoxFit.scaleDown,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text.rich(
                        TextSpan(
                          style: lineStyle,
                          children: [
                            TextSpan(
                              text: 'anytty',
                              style: TextStyle(color: green),
                            ),
                            const TextSpan(text: ' in '),
                            TextSpan(
                              text: '~/workspace',
                              style: TextStyle(color: cyan),
                            ),
                          ],
                        ),
                      ),
                      Text.rich(
                        TextSpan(
                          style: lineStyle,
                          children: [
                            TextSpan(
                              text: r'$ ',
                              style: TextStyle(color: yellow),
                            ),
                            const TextSpan(text: 'git status --short'),
                          ],
                        ),
                      ),
                      Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(
                            ' M clients/flutter/lib/app.dart ',
                            style: lineStyle,
                          ),
                          AnimatedBuilder(
                            animation: _cursor,
                            builder: (context, child) => Opacity(
                              opacity:
                                  widget.settings.cursorBlink && !motionDisabled
                                  ? 0.28 + (_cursor.value * 0.72)
                                  : 1,
                              child: child,
                            ),
                            child: Container(
                              width: math.max(2, previewFontSize * 0.52),
                              height: previewFontSize * 1.05,
                              color: _terminalColor(terminalTheme.cursor),
                            ),
                          ),
                        ],
                      ),
                    ],
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

Color _terminalColor(int rgb) => Color(0xff000000 | rgb);

final class _SettingsRow extends StatelessWidget {
  const _SettingsRow({
    required this.label,
    required this.child,
    this.stacked = false,
    this.excludeLabelSemantics = false,
  });

  final String label;
  final Widget child;
  final bool stacked;
  final bool excludeLabelSemantics;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    child: stacked
        ? Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              ExcludeSemantics(
                excluding: excludeLabelSemantics,
                child: Text(
                  label,
                  style: const TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
              const SizedBox(height: 8),
              child,
            ],
          )
        : Row(
            children: [
              Expanded(
                child: ExcludeSemantics(
                  excluding: excludeLabelSemantics,
                  child: Text(
                    label,
                    maxLines: 2,
                    overflow: TextOverflow.visible,
                    softWrap: true,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                      height: 1.25,
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              child,
            ],
          ),
  );
}

final class _FontSizeStepper extends StatelessWidget {
  const _FontSizeStepper({required this.value, required this.onChanged});

  final int value;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 50,
      decoration: BoxDecoration(
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            tooltip: anyttyText(context, en: 'Decrease font size', zh: '减小字体'),
            onPressed: value > 8 ? () => onChanged(value - 1) : null,
            icon: const Icon(Icons.remove_rounded, size: 17),
          ),
          SizedBox(
            width: 42,
            child: Text(
              '$value',
              textAlign: TextAlign.center,
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
            ),
          ),
          IconButton(
            tooltip: anyttyText(context, en: 'Increase font size', zh: '增大字体'),
            onPressed: value < 32 ? () => onChanged(value + 1) : null,
            icon: const Icon(Icons.add_rounded, size: 17),
          ),
        ],
      ),
    );
  }
}

final class _SettingsDropdown<T> extends StatelessWidget {
  const _SettingsDropdown({
    required this.semanticsLabel,
    required this.value,
    required this.values,
    required this.labelFor,
    required this.onChanged,
  });

  final String semanticsLabel;
  final T value;
  final List<T> values;
  final String Function(T value) labelFor;
  final ValueChanged<T> onChanged;

  @override
  Widget build(BuildContext context) => MergeSemantics(
    child: Semantics(
      label: semanticsLabel,
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 190),
        child: DropdownButton<T>(
          value: value,
          isExpanded: true,
          underline: const SizedBox.shrink(),
          borderRadius: BorderRadius.circular(6),
          items: [
            for (final item in values)
              DropdownMenuItem<T>(
                value: item,
                child: Text(
                  labelFor(item),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
          ],
          onChanged: (next) {
            if (next != null) onChanged(next);
          },
        ),
      ),
    ),
  );
}

final class _FontPreviewButton extends StatelessWidget {
  const _FontPreviewButton({required this.value, required this.onPressed});

  final String value;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: anyttyText(
        context,
        en: 'Font preview, ${terminalFontLabel(value)}, choose font',
        zh: '字体预览，${terminalFontLabel(value)}，选择字体',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const ValueKey('terminal-font-picker'),
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(
              minWidth: 174,
              maxWidth: 174,
              minHeight: 48,
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(
                          'Aa 01 ~/',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            color: palette.text,
                            fontFamily: value,
                            fontSize: 14,
                            letterSpacing: 0,
                          ),
                        ),
                        Text(
                          terminalFontLabel(value),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(color: palette.muted, fontSize: 10),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    Icons.chevron_right_rounded,
                    size: 18,
                    color: palette.faint,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _ThemePreviewButton extends StatelessWidget {
  const _ThemePreviewButton({required this.value, required this.onPressed});

  final TerminalTheme value;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: anyttyText(
        context,
        en: 'Theme preview, ${value.label}, choose terminal theme',
        zh: '主题预览，${value.label}，选择终端主题',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          key: const ValueKey('terminal-theme-picker'),
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: ConstrainedBox(
            constraints: const BoxConstraints(
              minWidth: 174,
              maxWidth: 174,
              minHeight: 48,
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 6),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(
                          value.label,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 5),
                        _ThemeSwatches(theme: value, compact: true),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  Icon(
                    Icons.chevron_right_rounded,
                    size: 18,
                    color: palette.faint,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<String?> _showFontPicker(BuildContext context, String selected) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<String>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    useSafeArea: true,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: palette.surface,
    barrierColor: palette.overlay,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
    ),
    builder: (context) => _FontPickerSheet(selected: selected),
  );
}

final class _FontPickerSheet extends StatelessWidget {
  const _FontPickerSheet({required this.selected});

  final String selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      top: false,
      child: SizedBox(
        height: math.min(MediaQuery.sizeOf(context).height * 0.62, 480),
        child: Column(
          children: [
            _PickerHeader(
              title: anyttyText(context, en: 'Choose font', zh: '选择字体'),
              closeTooltip: anyttyText(
                context,
                en: 'Close font picker',
                zh: '关闭字体选择',
              ),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
                children: [
                  Text(
                    anyttyText(context, en: 'FONT PREVIEW', zh: '字体预览'),
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    clipBehavior: Clip.antiAlias,
                    decoration: BoxDecoration(
                      color: palette.surfaceRaised,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Column(
                      children: [
                        for (
                          var index = 0;
                          index < terminalFontFamilies.length;
                          index++
                        ) ...[
                          _FontChoiceRow(
                            value: terminalFontFamilies[index],
                            selected: terminalFontFamilies[index] == selected,
                          ),
                          if (index != terminalFontFamilies.length - 1)
                            Divider(height: 1, color: palette.border),
                        ],
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _FontChoiceRow extends StatelessWidget {
  const _FontChoiceRow({required this.value, required this.selected});

  final String value;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      label: anyttyText(
        context,
        en: '${terminalFontLabel(value)} font preview',
        zh: '${terminalFontLabel(value)} 字体预览',
      ),
      onTap: () => Navigator.pop(context, value),
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('terminal-font-$value'),
        onTap: () => Navigator.pop(context, value),
        child: SizedBox(
          height: 78,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        terminalFontLabel(value),
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 5),
                      Text(
                        r'anytty ~/workspace $ Aa 01 {}[]',
                        maxLines: 1,
                        overflow: TextOverflow.clip,
                        style: TextStyle(
                          color: palette.muted,
                          fontFamily: value,
                          fontSize: 13,
                          letterSpacing: 0,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                SizedBox.square(
                  dimension: 24,
                  child: selected
                      ? Icon(
                          Icons.check_rounded,
                          color: palette.accent,
                          size: 21,
                        )
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

Future<String?> _showThemePicker(BuildContext context, String selected) {
  final palette = AnyttyPalette.of(context);
  return showModalBottomSheet<String>(
    context: context,
    sheetAnimationStyle: AnimationStyle.noAnimation,
    useSafeArea: true,
    isScrollControlled: true,
    showDragHandle: true,
    backgroundColor: palette.surface,
    barrierColor: palette.overlay,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
    ),
    builder: (context) => _ThemePickerSheet(selected: selected),
  );
}

final class _ThemePickerSheet extends StatefulWidget {
  const _ThemePickerSheet({required this.selected});

  final String selected;

  @override
  State<_ThemePickerSheet> createState() => _ThemePickerSheetState();
}

final class _ThemePickerSheetState extends State<_ThemePickerSheet> {
  final GlobalKey _selectedKey = GlobalKey();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final selectedContext = _selectedKey.currentContext;
      if (!mounted || selectedContext == null) return;
      Scrollable.ensureVisible(
        selectedContext,
        alignment: 0.42,
        duration: AnyttyMotion.disabled(context)
            ? Duration.zero
            : AnyttyMotion.standard,
        curve: AnyttyMotion.emphasized,
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final dark = terminalThemes.where((theme) => theme.dark).toList();
    final light = terminalThemes.where((theme) => !theme.dark).toList();
    return SafeArea(
      top: false,
      child: SizedBox(
        height: math.min(MediaQuery.sizeOf(context).height * 0.78, 680),
        child: Column(
          children: [
            _PickerHeader(
              title: anyttyText(
                context,
                en: 'Choose terminal theme',
                zh: '选择终端主题',
              ),
              closeTooltip: anyttyText(
                context,
                en: 'Close theme picker',
                zh: '关闭主题选择',
              ),
            ),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
                children: [
                  _ThemeChoiceGroup(
                    label: anyttyText(context, en: 'DARK THEMES', zh: '深色主题'),
                    themes: dark,
                    selected: widget.selected,
                    selectedKey: _selectedKey,
                  ),
                  const SizedBox(height: 20),
                  _ThemeChoiceGroup(
                    label: anyttyText(context, en: 'LIGHT THEMES', zh: '浅色主题'),
                    themes: light,
                    selected: widget.selected,
                    selectedKey: _selectedKey,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _ThemeChoiceGroup extends StatelessWidget {
  const _ThemeChoiceGroup({
    required this.label,
    required this.themes,
    required this.selected,
    required this.selectedKey,
  });

  final String label;
  final List<TerminalTheme> themes;
  final String selected;
  final GlobalKey selectedKey;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          label,
          style: TextStyle(
            color: palette.muted,
            fontSize: 11,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        Container(
          clipBehavior: Clip.antiAlias,
          decoration: BoxDecoration(
            color: palette.surfaceRaised,
            border: Border.all(color: palette.border),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Column(
            children: [
              for (var index = 0; index < themes.length; index++) ...[
                _ThemeChoiceRow(
                  key: themes[index].id == selected ? selectedKey : null,
                  theme: themes[index],
                  selected: themes[index].id == selected,
                ),
                if (index != themes.length - 1)
                  Divider(height: 1, color: palette.border),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

final class _ThemeChoiceRow extends StatelessWidget {
  const _ThemeChoiceRow({
    super.key,
    required this.theme,
    required this.selected,
  });

  final TerminalTheme theme;
  final bool selected;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      label: anyttyText(
        context,
        en: '${theme.label} theme palette preview',
        zh: '${theme.label} 主题配色预览',
      ),
      onTap: () => Navigator.pop(context, theme.id),
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('terminal-theme-${theme.id}'),
        onTap: () => Navigator.pop(context, theme.id),
        child: SizedBox(
          height: 62,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    theme.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                _ThemeSwatches(theme: theme),
                const SizedBox(width: 10),
                SizedBox.square(
                  dimension: 24,
                  child: selected
                      ? Icon(
                          Icons.check_rounded,
                          color: palette.accent,
                          size: 21,
                        )
                      : null,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _ThemeSwatches extends StatelessWidget {
  const _ThemeSwatches({required this.theme, this.compact = false});

  final TerminalTheme theme;
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final colors = [
      theme.background,
      theme.foreground,
      theme.ansi[1],
      theme.ansi[2],
      theme.ansi[4],
    ];
    final size = compact ? 11.0 : 18.0;
    final gap = compact ? 3.0 : 4.0;
    return SizedBox(
      height: size,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (var index = 0; index < colors.length; index++) ...[
            Container(
              width: size,
              height: size,
              decoration: BoxDecoration(
                color: _terminalColor(colors[index]),
                borderRadius: BorderRadius.circular(compact ? 2 : 3),
                border: Border.all(
                  color: palette.borderStrong.withValues(alpha: 0.55),
                  width: 0.5,
                ),
              ),
            ),
            if (index != colors.length - 1) SizedBox(width: gap),
          ],
        ],
      ),
    );
  }
}

final class _PickerHeader extends StatelessWidget {
  const _PickerHeader({required this.title, required this.closeTooltip});

  final String title;
  final String closeTooltip;

  @override
  Widget build(BuildContext context) => SizedBox(
    height: 58,
    child: Padding(
      padding: const EdgeInsets.only(left: 16, right: 4),
      child: Row(
        children: [
          Expanded(
            child: Text(
              title,
              style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w600),
            ),
          ),
          IconButton(
            tooltip: closeTooltip,
            onPressed: () => Navigator.pop(context),
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
    ),
  );
}

final class _AppearancePicker extends StatelessWidget {
  const _AppearancePicker({required this.value, required this.onChanged});

  final AppAppearance value;
  final ValueChanged<AppAppearance> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 58,
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'Light', zh: '浅色'),
              icon: Icons.light_mode_outlined,
              selected: value == AppAppearance.light,
              onPressed: () => onChanged(AppAppearance.light),
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'Dark', zh: '深色'),
              icon: Icons.dark_mode_outlined,
              selected: value == AppAppearance.dark,
              onPressed: () => onChanged(AppAppearance.dark),
            ),
          ),
          const SizedBox(width: 4),
          Expanded(
            child: _AppearanceOption(
              label: anyttyText(context, en: 'System', zh: '跟随系统'),
              icon: Icons.brightness_auto_outlined,
              selected: value == AppAppearance.system,
              onPressed: () => onChanged(AppAppearance.system),
            ),
          ),
        ],
      ),
    );
  }
}

final class _AppearanceOption extends StatelessWidget {
  const _AppearanceOption({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      inMutuallyExclusiveGroup: true,
      label: anyttyText(
        context,
        en: '$label interface theme',
        zh: '$label界面主题',
      ),
      onTap: onPressed,
      excludeSemantics: true,
      child: Material(
        color: selected ? palette.accent : Colors.transparent,
        borderRadius: BorderRadius.circular(6),
        child: InkWell(
          borderRadius: BorderRadius.circular(6),
          onTap: onPressed,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                icon,
                size: 17,
                color: selected ? palette.accentText : palette.muted,
              ),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: selected ? palette.accentText : palette.text,
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
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
