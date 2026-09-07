import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../data/browser_bookmark_store.dart';
import '../data/browser_history_store.dart';
import 'browser_perched_mascot.dart';

final class BrowserNewTabPage extends StatelessWidget {
  const BrowserNewTabPage({
    super.key,
    required this.searchController,
    required this.searchFocusNode,
    required this.onSearch,
    required this.onFocusSearch,
    required this.bookmarks,
    required this.history,
    required this.onRemoveBookmark,
    required this.onOpenHistory,
    this.endpointLabel,
    this.onSwitchEndpoint,
    this.addressFocusNode,
    this.addressController,
  });

  final TextEditingController searchController;
  final FocusNode searchFocusNode;
  final Future<void> Function(String value) onSearch;
  final VoidCallback onFocusSearch;
  final List<BrowserBookmark> bookmarks;
  final List<BrowserHistoryEntry> history;
  final Future<void> Function(String url) onRemoveBookmark;
  final VoidCallback onOpenHistory;
  final String? endpointLabel;
  final VoidCallback? onSwitchEndpoint;
  final FocusNode? addressFocusNode;
  final TextEditingController? addressController;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final horizontalPadding = constraints.maxWidth >= 640 ? 32.0 : 24.0;
      final compact = constraints.maxHeight < 480;
      return Material(
        color: AnyttyPalette.of(context).background,
        child: SingleChildScrollView(
          keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
          padding: EdgeInsets.fromLTRB(
            horizontalPadding,
            compact ? 16 : 32,
            horizontalPadding,
            48,
          ),
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Wrap(
                    alignment: WrapAlignment.spaceBetween,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    spacing: 16,
                    children: [
                      const _NewTabIdentity(),
                      if (endpointLabel?.trim().isNotEmpty ?? false)
                        _NewTabEndpoint(
                          label: endpointLabel!,
                          onSwitch: onSwitchEndpoint,
                        ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  LayoutBuilder(
                    builder: (context, fieldConstraints) {
                      return AnimatedBuilder(
                        animation: Listenable.merge([
                          searchController,
                          searchFocusNode,
                          addressController,
                          addressFocusNode,
                        ]),
                        builder: (context, _) {
                          final address = addressController?.text.trim() ?? '';
                          final resting =
                              searchController.text.trim().isEmpty &&
                              !searchFocusNode.hasFocus &&
                              !(addressFocusNode?.hasFocus ?? false) &&
                              (address.isEmpty || address == 'about:blank') &&
                              MediaQuery.viewInsetsOf(context).bottom == 0;
                          return BrowserPerchedMascot(
                            key: const ValueKey('browser-perched-mascot'),
                            visible: resting,
                            child: _NewTabSearchField(
                              controller: searchController,
                              focusNode: searchFocusNode,
                              onSearch: onSearch,
                              onFocusSearch: onFocusSearch,
                            ),
                          );
                        },
                      );
                    },
                  ),
                  const SizedBox(height: 16),
                  _NewTabSectionHeader(
                    title: anyttyText(context, en: 'Saved links', zh: '收藏链接'),
                  ),
                  const SizedBox(height: 12),
                  if (bookmarks.isEmpty)
                    _NewTabEmptyMessage(
                      title: anyttyText(
                        context,
                        en: 'No saved links yet',
                        zh: '还没有收藏链接',
                      ),
                    )
                  else
                    _NewTabBookmarkGrid(
                      bookmarks: bookmarks,
                      onOpen: onSearch,
                      onRemove: onRemoveBookmark,
                    ),
                  if (history.isNotEmpty) ...[
                    const SizedBox(height: 20),
                    _NewTabSectionHeader(
                      title: anyttyText(
                        context,
                        en: 'Recent pages',
                        zh: '最近访问',
                      ),
                      trailing: history.isEmpty
                          ? null
                          : TextButton(
                              onPressed: onOpenHistory,
                              child: Text(
                                anyttyText(context, en: 'View all', zh: '查看全部'),
                              ),
                            ),
                    ),
                    const SizedBox(height: 8),
                    _NewTabHistoryList(
                      entries: history.take(3).toList(),
                      onOpen: onSearch,
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      );
    },
  );
}

final class _NewTabIdentity extends StatelessWidget {
  const _NewTabIdentity();
  @override
  Widget build(BuildContext context) => Text(
    'AnyTTY',
    style: TextStyle(
      color: AnyttyPalette.of(context).text,
      fontFamily: 'JetBrainsMonoNerd',
      fontSize: 24,
      fontWeight: FontWeight.w700,
    ),
  );
}

final class _NewTabEndpoint extends StatelessWidget {
  const _NewTabEndpoint({required this.label, this.onSwitch});
  final String label;
  final VoidCallback? onSwitch;
  @override
  Widget build(BuildContext context) => Tooltip(
    message: anyttyText(context, en: 'Switch device', zh: '切换设备'),
    child: TextButton.icon(
      onPressed: onSwitch,
      icon: const Icon(Icons.computer_rounded, size: 16),
      label: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 170),
        child: Text(
          label,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            color: AnyttyPalette.of(context).muted,
            fontSize: 12,
          ),
        ),
      ),
    ),
  );
}

final class _NewTabSearchField extends StatelessWidget {
  const _NewTabSearchField({
    required this.controller,
    required this.focusNode,
    required this.onSearch,
    required this.onFocusSearch,
  });

  final TextEditingController controller;
  final FocusNode focusNode;
  final Future<void> Function(String value) onSearch;
  final VoidCallback onFocusSearch;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      textField: true,
      label: anyttyText(context, en: 'Search the web', zh: '搜索网页'),
      child: TextField(
        key: const ValueKey('browser-new-tab-search'),
        controller: controller,
        focusNode: focusNode,
        onTap: onFocusSearch,
        onTapOutside: (_) => focusNode.unfocus(),
        onSubmitted: (value) {
          if (value.trim().isNotEmpty) unawaited(onSearch(value));
        },
        textInputAction: TextInputAction.search,
        keyboardType: TextInputType.url,
        maxLines: 1,
        style: TextStyle(color: palette.text, fontSize: 16),
        decoration: InputDecoration(
          hintText: anyttyText(
            context,
            en: 'Search or enter a web address',
            zh: '搜索或输入网址',
          ),
          hintStyle: TextStyle(color: palette.muted, fontSize: 16),
          prefixIcon: Icon(
            Icons.search_rounded,
            color: palette.accent,
            size: 23,
          ),
          suffixIcon: IconButton(
            tooltip: anyttyText(context, en: 'Search', zh: '搜索'),
            onPressed: () {
              if (controller.text.trim().isNotEmpty) {
                unawaited(onSearch(controller.text));
              }
            },
            icon: const Icon(Icons.arrow_forward_rounded, size: 20),
            color: palette.accent,
          ),
          filled: true,
          fillColor: palette.surface,
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 12,
            vertical: 17,
          ),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: palette.border),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: palette.border),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
            borderSide: BorderSide(color: palette.accent, width: 1.4),
          ),
        ),
      ),
    );
  }
}

final class _NewTabSectionHeader extends StatelessWidget {
  const _NewTabSectionHeader({required this.title, this.trailing});

  final String title;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final splitActions = MediaQuery.textScalerOf(context).scale(14) > 21;
    final heading = Row(
      children: [
        Expanded(
          child: Text(
            title,
            style: TextStyle(
              color: palette.muted,
              fontSize: 14,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        const SizedBox(width: 8),
        if (!splitActions) ?trailing,
      ],
    );
    return splitActions && trailing != null
        ? Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              heading,
              Align(alignment: Alignment.centerRight, child: trailing),
            ],
          )
        : heading;
  }
}

final class _NewTabBookmarkGrid extends StatelessWidget {
  const _NewTabBookmarkGrid({
    required this.bookmarks,
    required this.onOpen,
    required this.onRemove,
  });

  final List<BrowserBookmark> bookmarks;
  final Future<void> Function(String url) onOpen;
  final Future<void> Function(String url) onRemove;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final textScaler = MediaQuery.textScalerOf(context);
      final columns = textScaler.scale(14) > 21
          ? (constraints.maxWidth >= 280 ? 2 : 1)
          : (constraints.maxWidth / 80).floor().clamp(2, 6);
      return GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        itemCount: bookmarks.length,
        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: columns,
          mainAxisExtent: 68 + textScaler.scale(13) * 1.3,
          crossAxisSpacing: 8,
          mainAxisSpacing: 8,
        ),
        itemBuilder: (context, index) {
          final bookmark = bookmarks[index];
          return _NewTabBookmarkTile(
            bookmark: bookmark,
            onOpen: () => unawaited(onOpen(bookmark.url)),
            onRemove: () => unawaited(onRemove(bookmark.url)),
          );
        },
      );
    },
  );
}

final class _NewTabBookmarkTile extends StatelessWidget {
  const _NewTabBookmarkTile({
    required this.bookmark,
    required this.onOpen,
    required this.onRemove,
  });
  final BrowserBookmark bookmark;
  final VoidCallback onOpen;
  final VoidCallback onRemove;

  Future<void> _showActions(BuildContext context) => showModalBottomSheet<void>(
    context: context,
    showDragHandle: true,
    builder: (sheetContext) => SafeArea(
      child: ListTile(
        leading: const Icon(Icons.delete_outline_rounded),
        title: Text(anyttyText(context, en: 'Remove saved link', zh: '移除收藏链接')),
        subtitle: Text(bookmark.title.isEmpty ? bookmark.url : bookmark.title),
        onTap: () {
          Navigator.of(sheetContext).pop();
          onRemove();
        },
      ),
    ),
  );

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final title = bookmark.title.isEmpty
        ? (Uri.tryParse(bookmark.url)?.host ?? bookmark.url)
        : bookmark.title;
    return Tooltip(
      message: bookmark.url,
      child: Semantics(
        customSemanticsActions: {
          CustomSemanticsAction(
            label: anyttyText(context, en: 'Remove saved link', zh: '移除收藏链接'),
          ): onRemove,
        },
        child: Material(
          color: Colors.transparent,
          child: InkWell(
            onTap: onOpen,
            onLongPress: () => _showActions(context),
            borderRadius: BorderRadius.circular(8),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Column(
                children: [
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      color: palette.surfaceRaised,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Icon(
                      Icons.public_rounded,
                      color: palette.accent,
                      size: 23,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    title,
                    maxLines: 1,
                    textAlign: TextAlign.center,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      color: palette.text,
                      fontSize: 13,
                      height: 1.3,
                    ),
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

final class _NewTabHistoryList extends StatelessWidget {
  const _NewTabHistoryList({required this.entries, required this.onOpen});

  final List<BrowserHistoryEntry> entries;
  final Future<void> Function(String url) onOpen;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: entries.length,
      separatorBuilder: (_, _) => Divider(height: 1, color: palette.border),
      itemBuilder: (context, index) {
        final entry = entries[index];
        return ListTile(
          contentPadding: EdgeInsets.zero,
          minTileHeight: 58,
          leading: Icon(Icons.history_rounded, color: palette.muted, size: 21),
          title: Text(
            entry.title.isEmpty ? entry.url : entry.title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: palette.text, fontWeight: FontWeight.w600),
          ),
          subtitle: Text(
            entry.url,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: palette.muted, fontSize: 12),
          ),
          onTap: () => unawaited(onOpen(entry.url)),
        );
      },
    );
  }
}

final class _NewTabEmptyMessage extends StatelessWidget {
  const _NewTabEmptyMessage({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Text(title, style: TextStyle(color: palette.muted, fontSize: 14)),
    );
  }
}
