part of 'settings_screen.dart';


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
        actions: [
          if (MediaQuery.textScalerOf(context).scale(18) < 30)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 20),
              child: AnyttyBrandMark(height: 32),
            ),
        ],
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
