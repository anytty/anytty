part of 'device_list_screen.dart';

final class _RaisedHeaderAction extends StatelessWidget {
  const _RaisedHeaderAction({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) =>
      Transform.translate(offset: const Offset(0, -18), child: child);
}

AsyncValue<EndpointCloudPresenceGetResult>? _cloudPresenceFor(
  WidgetRef ref,
  EndpointConfigV1 endpoint,
) {
  if (!endpoint.enabled ||
      _authorizationRequired(endpoint) ||
      !_hasEnabledCloudRoute(endpoint)) {
    return null;
  }
  return ref.watch(endpointCloudPresenceProvider(endpoint.endpointId));
}

final class _DeviceHeaderStatus extends StatelessWidget {
  const _DeviceHeaderStatus({
    required this.onlineCount,
    required this.totalCount,
    required this.refreshing,
    required this.onRefresh,
  });

  final int onlineCount;
  final int totalCount;
  final bool refreshing;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final status = anyttyText(
      context,
      en: '$onlineCount of $totalCount devices online. Refresh',
      zh: '$onlineCount/$totalCount 台在线，刷新',
    );
    return Tooltip(
      message: status,
      child: Semantics(
        button: true,
        liveRegion: true,
        label: status,
        child: InkResponse(
          key: const ValueKey('device-header-status'),
          onTap: refreshing ? null : onRefresh,
          radius: 24,
          child: SizedBox.square(
            dimension: 48,
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                SizedBox.square(
                  dimension: 19,
                  child: refreshing
                      ? CircularProgressIndicator(
                          color: palette.accent,
                          strokeWidth: 2,
                        )
                      : Stack(
                          alignment: Alignment.center,
                          clipBehavior: Clip.none,
                          children: [
                            Icon(
                              Icons.refresh_rounded,
                              color: palette.muted,
                              size: 19,
                            ),
                            Positioned(
                              right: -1,
                              top: -1,
                              child: Container(
                                width: 7,
                                height: 7,
                                decoration: BoxDecoration(
                                  color: onlineCount > 0
                                      ? palette.success
                                      : palette.faint,
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: palette.background,
                                    width: 1.5,
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                ),
                const SizedBox(height: 1),
                Text(
                  '$onlineCount/$totalCount',
                  textScaler: TextScaler.noScaling,
                  style: TextStyle(
                    color: palette.muted,
                    fontSize: 9,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

final class _DeviceSearchField extends StatelessWidget {
  const _DeviceSearchField({
    required this.controller,
    required this.focusNode,
    required this.onClose,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: 44,
      child: Focus(
        onKeyEvent: (node, event) {
          if (event is KeyDownEvent &&
              event.logicalKey == LogicalKeyboardKey.escape) {
            onClose();
            return KeyEventResult.handled;
          }
          return KeyEventResult.ignored;
        },
        child: TextField(
          key: const ValueKey('device-search-field'),
          controller: controller,
          focusNode: focusNode,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: anyttyText(context, en: 'Search devices', zh: '搜索设备'),
            prefixIcon: Icon(
              Icons.search_rounded,
              color: palette.muted,
              size: 19,
            ),
            suffixIcon: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (controller.text.isNotEmpty)
                  IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Clear search',
                      zh: '清除搜索',
                    ),
                    onPressed: controller.clear,
                    icon: const Icon(Icons.backspace_outlined, size: 17),
                  ),
                IconButton(
                  tooltip: anyttyText(context, en: 'Close search', zh: '关闭搜索'),
                  onPressed: onClose,
                  icon: const Icon(Icons.close_rounded, size: 18),
                ),
              ],
            ),
            filled: true,
            fillColor: palette.surface,
            contentPadding: const EdgeInsets.symmetric(vertical: 10),
            enabledBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(14),
              borderSide: BorderSide(color: palette.border),
            ),
            focusedBorder: OutlineInputBorder(
              borderRadius: BorderRadius.circular(14),
              borderSide: BorderSide(color: palette.accent, width: 1.5),
            ),
          ),
        ),
      ),
    );
  }
}

final class _NoMatchingDevices extends StatelessWidget {
  const _NoMatchingDevices();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 36),
      child: Column(
        children: [
          Icon(Icons.search_off_rounded, color: palette.muted, size: 30),
          const SizedBox(height: 10),
          Text(
            anyttyText(context, en: 'No matching devices', zh: '没有匹配的设备'),
            style: TextStyle(color: palette.muted, fontWeight: FontWeight.w600),
          ),
        ],
      ),
    );
  }
}

final class _DeviceSectionHeading extends StatelessWidget {
  const _DeviceSectionHeading({
    required this.count,
    required this.searchOpen,
    required this.searchController,
    required this.searchFocusNode,
    required this.onOpenSearch,
    required this.onCloseSearch,
  });

  final int count;
  final bool searchOpen;
  final TextEditingController searchController;
  final FocusNode searchFocusNode;
  final VoidCallback onOpenSearch;
  final VoidCallback onCloseSearch;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: searchOpen
          ? _DeviceSearchField(
              controller: searchController,
              focusNode: searchFocusNode,
              onClose: onCloseSearch,
            )
          : Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                Text(
                  anyttyText(context, en: 'My devices', zh: '我的设备'),
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(width: 8),
                Text(
                  '$count',
                  style: TextStyle(
                    color: palette.faint,
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const Spacer(),
                SizedBox.square(
                  dimension: 40,
                  child: IconButton(
                    tooltip: anyttyText(
                      context,
                      en: 'Search devices',
                      zh: '搜索设备',
                    ),
                    onPressed: onOpenSearch,
                    padding: EdgeInsets.zero,
                    icon: const Icon(Icons.search_rounded, size: 20),
                  ),
                ),
              ],
            ),
    );
  }
}

final class _DeviceListPanel extends StatelessWidget {
  const _DeviceListPanel({
    required this.endpoints,
    required this.defaultEndpointId,
    required this.searchQuery,
    required this.onActions,
  });

  final List<EndpointConfigV1> endpoints;
  final String defaultEndpointId;
  final String searchQuery;
  final ValueChanged<EndpointConfigV1> onActions;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        for (var index = 0; index < endpoints.length; index++) ...[
          _DeviceRow(
            endpoint: endpoints[index],
            isDefault: endpoints[index].endpointId == defaultEndpointId,
            searchQuery: searchQuery,
            onActions: () => onActions(endpoints[index]),
          ),
          if (index != endpoints.length - 1) const SizedBox(height: 10),
        ],
      ],
    );
  }
}
