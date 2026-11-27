part of 'browser_session_screen.dart';

final class _PreparedBrowserSession {
  _PreparedBrowserSession({
    required this.endpointSession,
    required this.httpProxy,
    required this.endpointSubscription,
    required this.snapshot,
  });

  final EndpointSessionClient endpointSession;
  final BrowserHttpProxy httpProxy;
  final ProviderSubscription<AsyncValue<EndpointSessionClient>>?
  endpointSubscription;
  final BrowserSessionSnapshot snapshot;

  Future<void> dispose() async {
    endpointSubscription?.close();
    await httpProxy.close();
  }
}

final class _BrowserToolbarTitle extends StatelessWidget {
  const _BrowserToolbarTitle({
    required this.addressController,
    required this.addressFocusNode,
    required this.controller,
    required this.onNavigate,
    required this.onBack,
    required this.onForward,
    required this.history,
    required this.onDismiss,
  });

  final TextEditingController addressController;
  final FocusNode addressFocusNode;
  final WebViewController? controller;
  final Future<void> Function([String?]) onNavigate;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final List<BrowserHistoryEntry> history;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: addressFocusNode,
      builder: (context, _) => LayoutBuilder(
        builder: (context, constraints) {
          final showInlineNavigation = constraints.maxWidth >= 560;
          final addressFocused = addressFocusNode.hasFocus;
          return Row(
            children: [
              if (showInlineNavigation && !addressFocused) ...[
                _BrowserIconButton(
                  tooltip: anyttyText(context, en: 'Back', zh: '后退'),
                  enabled: controller != null,
                  onPressed: onBack,
                  icon: Icons.arrow_back_rounded,
                ),
                _BrowserIconButton(
                  tooltip: anyttyText(context, en: 'Forward', zh: '前进'),
                  enabled: controller != null,
                  onPressed: onForward,
                  icon: Icons.arrow_forward_rounded,
                ),
                const SizedBox(width: 4),
              ],
              Expanded(
                child: BrowserAddressBar(
                  controller: addressController,
                  focusNode: addressFocusNode,
                  enabled: controller != null,
                  onNavigate: (value) => onNavigate(value),
                  history: history,
                  onDismiss: onDismiss,
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}

final class _BrowserOverflowMenu extends StatelessWidget {
  const _BrowserOverflowMenu({
    required this.bookmarked,
    required this.onToggleBookmark,
    required this.state,
    required this.hasProxy,
    required this.dnsProxied,
    required this.controller,
    required this.onBack,
    required this.onForward,
    required this.onReload,
    required this.onSwitchSession,
    required this.onOpenTabs,
    required this.onOpenHistory,
    required this.onOpenSettings,
    required this.readerMode,
    required this.desktopMode,
    required this.onToggleReaderMode,
    required this.onToggleDesktopMode,
  });

  final BrowserSessionState state;
  final bool bookmarked;
  final VoidCallback? onToggleBookmark;
  final bool hasProxy;
  final bool dnsProxied;
  final WebViewController? controller;
  final VoidCallback onBack;
  final VoidCallback onForward;
  final VoidCallback onReload;
  final VoidCallback onSwitchSession;
  final VoidCallback onOpenTabs;
  final VoidCallback onOpenHistory;
  final VoidCallback onOpenSettings;
  final bool readerMode;
  final bool desktopMode;
  final VoidCallback onToggleReaderMode;
  final VoidCallback onToggleDesktopMode;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = _browserStatus(
      context,
      state: state,
      hasProxy: hasProxy,
      dnsProxied: dnsProxied,
    );
    return MenuAnchor(
      style: MenuStyle(
        minimumSize: const WidgetStatePropertyAll(Size(220, 0)),
        padding: const WidgetStatePropertyAll(
          EdgeInsets.symmetric(vertical: 6),
        ),
      ),
      menuChildren: [
        MenuItemButton(
          leadingIcon: Icon(
            bookmarked ? Icons.star_rounded : Icons.star_border_rounded,
            size: 18,
          ),
          onPressed: onToggleBookmark,
          child: Text(
            bookmarked
                ? anyttyText(context, en: 'Remove bookmark', zh: '取消收藏')
                : anyttyText(context, en: 'Save page', zh: '收藏网页'),
          ),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.arrow_back_rounded, size: 18),
          onPressed: controller != null ? onBack : null,
          child: Text(anyttyText(context, en: 'Back', zh: '后退')),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.arrow_forward_rounded, size: 18),
          onPressed: controller != null ? onForward : null,
          child: Text(anyttyText(context, en: 'Forward', zh: '前进')),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.refresh_rounded, size: 18),
          onPressed: controller != null ? onReload : null,
          child: Text(anyttyText(context, en: 'Reload', zh: '重新加载')),
        ),
        const Divider(height: 1),
        MenuItemButton(
          leadingIcon: const Icon(Icons.tab_rounded, size: 18),
          onPressed: onOpenTabs,
          child: Text(anyttyText(context, en: 'Tabs', zh: '标签页')),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.history_rounded, size: 18),
          onPressed: onOpenHistory,
          child: Text(anyttyText(context, en: 'History', zh: '历史记录')),
        ),
        MenuItemButton(
          leadingIcon: Icon(
            Icons.menu_book_outlined,
            color: readerMode ? palette.accent : null,
            size: 18,
          ),
          onPressed: onToggleReaderMode,
          child: Row(
            children: [
              Expanded(
                child: Text(anyttyText(context, en: 'Reader mode', zh: '阅读模式')),
              ),
              Text(
                readerMode
                    ? anyttyText(context, en: 'On', zh: '已开启')
                    : anyttyText(context, en: 'Off', zh: '关闭'),
                style: TextStyle(color: palette.muted, fontSize: 11),
              ),
            ],
          ),
        ),
        MenuItemButton(
          leadingIcon: Icon(
            Icons.desktop_windows_outlined,
            color: desktopMode ? palette.accent : null,
            size: 18,
          ),
          onPressed: onToggleDesktopMode,
          child: Row(
            children: [
              Expanded(
                child: Text(
                  anyttyText(context, en: 'Desktop site', zh: '电脑模式'),
                ),
              ),
              Text(
                desktopMode
                    ? anyttyText(context, en: 'On', zh: '已开启')
                    : anyttyText(context, en: 'Off', zh: '关闭'),
                style: TextStyle(color: palette.muted, fontSize: 11),
              ),
            ],
          ),
        ),
        MenuItemButton(
          leadingIcon: const Icon(Icons.settings_outlined, size: 18),
          onPressed: onOpenSettings,
          child: Text(anyttyText(context, en: 'Browser settings', zh: '浏览器设置')),
        ),
        const Divider(height: 1),
        MenuItemButton(
          leadingIcon: const Icon(Icons.devices_rounded, size: 18),
          onPressed: onSwitchSession,
          child: Text(anyttyText(context, en: 'Switch session', zh: '切换会话')),
        ),
        SizedBox(
          width: 220,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 6),
            child: Row(
              children: [
                Icon(status.icon, size: 18, color: status.color),
                const SizedBox(width: 12),
                Expanded(
                  child: Text(
                    status.label,
                    style: TextStyle(color: palette.muted, fontSize: 12),
                  ),
                ),
              ],
            ),
          ),
        ),
      ],
      builder: (context, menuController, child) => IconButton(
        tooltip: anyttyText(context, en: 'More browser actions', zh: '更多浏览器操作'),
        onPressed: () {
          if (menuController.isOpen) {
            menuController.close();
          } else {
            menuController.open();
          }
        },
        icon: const Icon(Icons.more_vert_rounded, size: 21),
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints.tightFor(width: 44, height: 48),
      ),
    );
  }
}

final class _BrowserTabCountButton extends StatelessWidget {
  const _BrowserTabCountButton({required this.count, required this.onPressed});

  final int count;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => IconButton(
    tooltip: anyttyText(context, en: 'Tabs', zh: '标签页'),
    onPressed: onPressed,
    icon: Container(
      width: 22,
      height: 22,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        border: Border.all(color: AnyttyPalette.of(context).muted),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        '$count',
        style: TextStyle(
          color: AnyttyPalette.of(context).text,
          fontSize: 11,
          fontWeight: FontWeight.w700,
        ),
      ),
    ),
  );
}

enum _BrowserTabActionType { newTab, select, close }

final class _BrowserTabAction {
  const _BrowserTabAction._(this.type, [this.tabId]);

  const _BrowserTabAction.newTab() : this._(_BrowserTabActionType.newTab);

  const _BrowserTabAction.select(String tabId)
    : this._(_BrowserTabActionType.select, tabId);

  const _BrowserTabAction.close(String tabId)
    : this._(_BrowserTabActionType.close, tabId);

  final _BrowserTabActionType type;
  final String? tabId;
}

final class _BrowserTabsSheet extends StatelessWidget {
  const _BrowserTabsSheet({required this.tabs, required this.activeTabId});

  final List<BrowserTabSnapshot> tabs;
  final String? activeTabId;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 620),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 4, 12, 10),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      anyttyText(context, en: 'Tabs', zh: '标签页'),
                      style: TextStyle(
                        color: palette.text,
                        fontSize: 18,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  IconButton(
                    tooltip: anyttyText(context, en: 'New tab', zh: '新建标签页'),
                    onPressed: () =>
                        Navigator.of(context)
                            .pop(const _BrowserTabAction.newTab()),
                    icon: const Icon(Icons.add_rounded),
                  ),
                ],
              ),
            ),
            Flexible(
              child: ListView.separated(
                shrinkWrap: true,
                padding: const EdgeInsets.only(bottom: 12),
                itemCount: tabs.length,
                separatorBuilder: (_, _) =>
                    Divider(height: 1, indent: 72, color: palette.border),
                itemBuilder: (context, index) {
                  final tab = tabs[index];
                  final active = tab.id == activeTabId;
                  return ListTile(
                    minTileHeight: 68,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 20),
                    leading: Container(
                      width: 36,
                      height: 36,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: active
                            ? palette.accent.withValues(alpha: 0.12)
                            : palette.surfaceRaised,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        '${index + 1}',
                        style: TextStyle(
                          color: active ? palette.accent : palette.muted,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    title: Text(
                      tab.title.isEmpty
                          ? anyttyText(context, en: 'New tab', zh: '新标签页')
                          : tab.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    subtitle: Text(
                      tab.url.isEmpty ? 'about:blank' : tab.url,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                    trailing: IconButton(
                      tooltip: anyttyText(
                        context,
                        en: 'Close tab',
                        zh: '关闭标签页',
                      ),
                      onPressed: () =>
                          Navigator.of(context)
                              .pop(_BrowserTabAction.close(tab.id)),
                      icon: const Icon(Icons.close_rounded, size: 18),
                    ),
                    onTap: () =>
                        Navigator.of(context)
                            .pop(_BrowserTabAction.select(tab.id)),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _BrowserHistorySheet extends StatelessWidget {
  const _BrowserHistorySheet({required this.entries, required this.onClear});

  final List<BrowserHistoryEntry> entries;
  final Future<void> Function() onClear;

  @override
  Widget build(BuildContext context) {
    var currentEntries = entries;
    return StatefulBuilder(
      builder: (context, setState) {
        final palette = AnyttyPalette.of(context);
        return SafeArea(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 620),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 4, 12, 10),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          anyttyText(context, en: 'History', zh: '历史记录'),
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 18,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      TextButton(
                        onPressed: currentEntries.isEmpty
                            ? null
                            : () {
                                setState(() => currentEntries = const []);
                                unawaited(onClear());
                              },
                        child: Text(anyttyText(context, en: 'Clear', zh: '清空')),
                      ),
                    ],
                  ),
                ),
                if (currentEntries.isEmpty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(20, 20, 20, 34),
                    child: Center(
                      child: Text(
                        anyttyText(
                          context,
                          en: 'No browsing history',
                          zh: '还没有浏览记录',
                        ),
                        style: TextStyle(color: palette.muted),
                      ),
                    ),
                  )
                else
                  Flexible(
                    child: ListView.separated(
                      shrinkWrap: true,
                      padding: const EdgeInsets.only(bottom: 12),
                      itemCount: currentEntries.length,
                      separatorBuilder: (_, _) =>
                          Divider(height: 1, indent: 68, color: palette.border),
                      itemBuilder: (context, index) {
                        final entry = currentEntries[index];
                        return ListTile(
                          minTileHeight: 62,
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 20,
                          ),
                          leading: Icon(
                            Icons.history_rounded,
                            color: palette.muted,
                          ),
                          title: Text(
                            entry.title.isEmpty ? entry.url : entry.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.text,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                          subtitle: Text(
                            entry.url,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              color: palette.muted,
                              fontSize: 12,
                            ),
                          ),
                          onTap: () => Navigator.of(context).pop(entry),
                        );
                      },
                    ),
                  ),
              ],
            ),
          ),
        );
      },
    );
  }
}
