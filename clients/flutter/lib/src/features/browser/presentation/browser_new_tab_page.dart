import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/anytty_ui.dart';
import '../data/browser_bookmark_store.dart';
import '../data/browser_history_store.dart';

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
  });

  final TextEditingController searchController;
  final FocusNode searchFocusNode;
  final Future<void> Function(String value) onSearch;
  final VoidCallback onFocusSearch;
  final List<BrowserBookmark> bookmarks;
  final List<BrowserHistoryEntry> history;
  final Future<void> Function(String url) onRemoveBookmark;
  final VoidCallback onOpenHistory;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final horizontalPadding = constraints.maxWidth >= 640 ? 32.0 : 20.0;
      return Material(
        color: AnyttyPalette.of(context).background,
        child: SingleChildScrollView(
          padding: EdgeInsets.fromLTRB(
            horizontalPadding,
            32,
            horizontalPadding,
            48,
          ),
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 760),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _NewTabIdentity(),
                  const SizedBox(height: 24),
                  _NewTabSearchField(
                    controller: searchController,
                    focusNode: searchFocusNode,
                    onSearch: onSearch,
                    onFocusSearch: onFocusSearch,
                  ),
                  const SizedBox(height: 34),
                  _NewTabSectionHeader(
                    icon: Icons.bookmark_outline_rounded,
                    title: anyttyText(context, en: 'Saved links', zh: '收藏链接'),
                    count: bookmarks.length,
                  ),
                  const SizedBox(height: 12),
                  if (bookmarks.isEmpty)
                    _NewTabEmptyMessage(
                      icon: Icons.bookmark_add_outlined,
                      title: anyttyText(
                        context,
                        en: 'No saved links yet',
                        zh: '还没有收藏链接',
                      ),
                      detail: anyttyText(
                        context,
                        en: 'Use the star in the toolbar to keep a page here.',
                        zh: '点击工具栏里的星标，就能把网页保存在这里。',
                      ),
                    )
                  else
                    _NewTabBookmarkGrid(
                      bookmarks: bookmarks,
                      onOpen: onSearch,
                      onRemove: onRemoveBookmark,
                    ),
                  const SizedBox(height: 32),
                  _NewTabSectionHeader(
                    icon: Icons.history_rounded,
                    title: anyttyText(context, en: 'Recent pages', zh: '最近访问'),
                    count: history.length,
                    trailing: history.isEmpty
                        ? null
                        : AnyttyPillButton(
                            label: anyttyText(
                              context,
                              en: 'View all',
                              zh: '查看全部',
                            ),
                            icon: Icons.arrow_forward_rounded,
                            onPressed: onOpenHistory,
                          ),
                  ),
                  const SizedBox(height: 8),
                  if (history.isEmpty)
                    _NewTabEmptyMessage(
                      icon: Icons.language_rounded,
                      title: anyttyText(
                        context,
                        en: 'Your recent pages will appear here',
                        zh: '最近打开的网页会显示在这里',
                      ),
                      detail: anyttyText(
                        context,
                        en: 'Search above or enter an address to get started.',
                        zh: '在上方搜索，或输入网址开始浏览。',
                      ),
                    )
                  else
                    _NewTabHistoryList(
                      entries: history.take(6).toList(),
                      onOpen: onSearch,
                    ),
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
  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        DecoratedBox(
          decoration: BoxDecoration(
            color: palette.accent.withValues(alpha: 0.14),
            borderRadius: BorderRadius.circular(10),
          ),
          child: SizedBox(
            width: 48,
            height: 48,
            child: Icon(
              Icons.explore_outlined,
              color: palette.accent,
              size: 28,
            ),
          ),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                anyttyText(context, en: 'AnyTTY browser', zh: 'AnyTTY 浏览器'),
                style: AnyttyUi.title(context),
              ),
              const SizedBox(height: 3),
              Text(
                anyttyText(
                  context,
                  en: 'A clean start for the pages you use remotely.',
                  zh: '从这里继续浏览远程设备上的网页。',
                ),
                style: AnyttyUi.muted(context),
              ),
            ],
          ),
        ),
      ],
    );
  }
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
        onSubmitted: (value) {
          if (value.trim().isNotEmpty) unawaited(onSearch(value));
        },
        textInputAction: TextInputAction.search,
        keyboardType: TextInputType.url,
        maxLines: 1,
        style: AnyttyUi.body(context).copyWith(color: palette.text),
        decoration: InputDecoration(
          hintText: anyttyText(
            context,
            en: 'Search or enter a web address',
            zh: '搜索或输入网址',
          ),
          hintStyle: AnyttyUi.body(context).copyWith(color: palette.muted),
          prefixIcon: Icon(
            Icons.search_rounded,
            color: palette.accent,
            size: 23,
          ),
          suffixIcon: AnyttyIconButton(
            tooltip: anyttyText(context, en: 'Search', zh: '搜索'),
            onPressed: () {
              if (controller.text.trim().isNotEmpty) {
                unawaited(onSearch(controller.text));
              }
            },
            icon: Icons.arrow_forward_rounded,
            iconColor: palette.accent,
            iconSize: 20,
          ),
          filled: true,
          fillColor: palette.surfaceRaised,
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 12,
            vertical: 17,
          ),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: BorderSide(color: Colors.transparent),
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: BorderSide(color: Colors.transparent),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: const BorderSide(color: Colors.transparent),
          ),
        ),
      ),
    );
  }
}

final class _NewTabSectionHeader extends StatelessWidget {
  const _NewTabSectionHeader({
    required this.icon,
    required this.title,
    required this.count,
    this.trailing,
  });

  final IconData icon;
  final String title;
  final int count;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Row(
      children: [
        Icon(icon, color: palette.strong, size: 20),
        const SizedBox(width: 9),
        Expanded(
          child: Text(
            title,
            style: AnyttyUi.sectionTitle(context),
            maxLines: 2,
            softWrap: true,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        const SizedBox(width: 8),
        Text('$count', style: AnyttyUi.muted(context)),
        if (trailing == null) const Spacer(),
        ?trailing,
      ],
    );
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
      final columns = constraints.maxWidth >= 520 ? 2 : 1;
      return GridView.builder(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        itemCount: bookmarks.length,
        gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
          crossAxisCount: columns,
          mainAxisExtent: math.max(
            72,
            MediaQuery.textScalerOf(context).scale(14.5) * (18 / 14.5) * 4 + 16,
          ),
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

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final title = bookmark.title.isEmpty ? bookmark.url : bookmark.title;
    return AnyttyCard(
      color: palette.surfaceRaised,
      radius: 14,
      depth: 1,
      padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: InkWell(
              customBorder: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(12),
              ),
              onTap: onOpen,
              child: Row(
                children: [
                  Icon(Icons.public_rounded, color: palette.strong, size: 20),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: AnyttyUi.body(context).copyWith(
                            color: palette.text,
                            fontWeight: FontWeight.w600,
                          ),
                          maxLines: 2,
                          softWrap: true,
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 2),
                        Text(
                          bookmark.url,
                          style: AnyttyUi.muted(context),
                          maxLines: 2,
                          softWrap: true,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
          AnyttyIconButton(
            tooltip: anyttyText(context, en: 'Remove saved link', zh: '移除收藏链接'),
            onPressed: onRemove,
            icon: Icons.close_rounded,
            iconColor: palette.strong,
            iconSize: 17,
          ),
        ],
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
      separatorBuilder: (_, _) => Divider(height: 1, color: palette.track),
      itemBuilder: (context, index) {
        final entry = entries[index];
        return ListTile(
          contentPadding: EdgeInsets.zero,
          minTileHeight: 58,
          leading: Icon(Icons.history_rounded, color: palette.strong, size: 21),
          title: Text(
            entry.title.isEmpty ? entry.url : entry.title,
            style: AnyttyUi.body(context)
                .copyWith(color: palette.text, fontWeight: FontWeight.w600),
          ),
          subtitle: Text(entry.url, style: AnyttyUi.muted(context)),
          onTap: () => unawaited(onOpen(entry.url)),
        );
      },
    );
  }
}

final class _NewTabEmptyMessage extends StatelessWidget {
  const _NewTabEmptyMessage({
    required this.icon,
    required this.title,
    required this.detail,
  });

  final IconData icon;
  final String title;
  final String detail;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 10),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: palette.strong, size: 21),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: AnyttyUi.body(
                    context,
                  ).copyWith(color: palette.text, fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 3),
                Text(detail, style: AnyttyUi.muted(context)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
