import 'dart:async';

import 'package:flutter/material.dart';

import '../../../app/anytty_theme.dart';
import '../data/path_bookmark_store.dart';

Future<String?> showPathBookmarks({
  required BuildContext context,
  required String endpointId,
  required String currentPath,
  PathBookmarkStore store = const PathBookmarkStore(),
}) => showModalBottomSheet<String>(
  context: context,
  sheetAnimationStyle: AnimationStyle.noAnimation,
  isScrollControlled: true,
  useSafeArea: true,
  showDragHandle: true,
  builder: (context) => _PathBookmarksSheet(
    endpointId: endpointId,
    currentPath: currentPath,
    store: store,
  ),
);

final class _PathBookmarksSheet extends StatefulWidget {
  const _PathBookmarksSheet({
    required this.endpointId,
    required this.currentPath,
    required this.store,
  });

  final String endpointId;
  final String currentPath;
  final PathBookmarkStore store;

  @override
  State<_PathBookmarksSheet> createState() => _PathBookmarksSheetState();
}

final class _PathBookmarksSheetState extends State<_PathBookmarksSheet> {
  List<PathBookmark> _bookmarks = const [];
  String? _error;
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final bookmarks = await widget.store.load(widget.endpointId);
      if (!mounted) return;
      setState(() {
        _bookmarks = bookmarks;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = error.toString();
        _loading = false;
      });
    }
  }

  Future<void> _saveCurrent() async {
    if (_saving) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.store.add(widget.endpointId, widget.currentPath);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Bookmark saved')));
      await _load();
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _edit(PathBookmark bookmark) async {
    final changed = await showModalBottomSheet<bool>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (context) => _PathBookmarkEditor(
        endpointId: widget.endpointId,
        bookmark: bookmark,
        store: widget.store,
      ),
    );
    if (changed == true && mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox(
      height: MediaQuery.sizeOf(context).height * 0.72,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 4, 12, 12),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Bookmarks',
                        style: TextStyle(
                          color: palette.text,
                          fontSize: 20,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        _loading ? 'Loading' : '${_bookmarks.length} saved',
                        style: TextStyle(color: palette.muted, fontSize: 13),
                      ),
                    ],
                  ),
                ),
                IconButton(
                  tooltip: 'Close bookmarks',
                  onPressed: () => Navigator.pop(context),
                  icon: const Icon(Icons.close_rounded),
                ),
              ],
            ),
          ),
          Divider(height: 1, color: palette.border),
          if (_error case final error?)
            MaterialBanner(
              content: Text(
                error,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              actions: [
                TextButton(onPressed: _load, child: const Text('Retry')),
              ],
            ),
          ListTile(
            leading: const Icon(Icons.bookmark_add_outlined),
            title: const Text('Save current folder'),
            subtitle: Text(
              widget.currentPath,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            enabled: !_saving,
            onTap: _saveCurrent,
          ),
          Divider(height: 1, color: palette.border),
          Expanded(
            child: _loading && _bookmarks.isEmpty
                ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
                : _bookmarks.isEmpty
                ? Center(
                    child: Text(
                      'No saved bookmarks',
                      style: TextStyle(color: palette.muted),
                    ),
                  )
                : ListView.separated(
                    itemCount: _bookmarks.length,
                    separatorBuilder: (_, _) =>
                        Divider(height: 1, color: palette.border),
                    itemBuilder: (context, index) {
                      final bookmark = _bookmarks[index];
                      return ListTile(
                        leading: const Icon(Icons.bookmark_border_rounded),
                        title: Text(bookmark.label),
                        subtitle: Text(
                          bookmark.path,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        trailing: IconButton(
                          tooltip: 'Edit bookmark ${bookmark.label}',
                          onPressed: () => _edit(bookmark),
                          icon: const Icon(Icons.edit_outlined),
                        ),
                        onTap: () => Navigator.pop(context, bookmark.path),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

final class _PathBookmarkEditor extends StatefulWidget {
  const _PathBookmarkEditor({
    required this.endpointId,
    required this.bookmark,
    required this.store,
  });

  final String endpointId;
  final PathBookmark bookmark;
  final PathBookmarkStore store;

  @override
  State<_PathBookmarkEditor> createState() => _PathBookmarkEditorState();
}

final class _PathBookmarkEditorState extends State<_PathBookmarkEditor> {
  late final TextEditingController _labelController;
  String? _error;
  bool _pending = false;

  @override
  void initState() {
    super.initState();
    _labelController = TextEditingController(text: widget.bookmark.label);
  }

  @override
  void dispose() {
    _labelController.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (_pending) return;
    setState(() => _pending = true);
    try {
      await widget.store.rename(
        widget.endpointId,
        widget.bookmark.id,
        _labelController.text,
      );
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _pending = false);
    }
  }

  Future<void> _remove() async {
    if (_pending) return;
    setState(() => _pending = true);
    try {
      await widget.store.remove(widget.endpointId, widget.bookmark.id);
      if (mounted) Navigator.pop(context, true);
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _pending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return AnimatedPadding(
      duration: AnyttyMotion.resolve(
        context,
        const Duration(milliseconds: 180),
      ),
      curve: Curves.easeOutCubic,
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              'Edit bookmark',
              style: TextStyle(
                color: palette.text,
                fontSize: 20,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              widget.bookmark.path,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(color: palette.muted, fontSize: 13),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _labelController,
              autofocus: true,
              textInputAction: TextInputAction.done,
              onSubmitted: (_) => _save(),
              decoration: InputDecoration(
                labelText: 'Alias',
                errorText: _error,
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _pending ? null : () => Navigator.pop(context),
                    child: const Text('Cancel'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: _pending ? null : _save,
                    child: const Text('Save'),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                style: OutlinedButton.styleFrom(
                  foregroundColor: palette.danger,
                  side: BorderSide(
                    color: palette.danger.withValues(alpha: 0.35),
                  ),
                ),
                onPressed: _pending ? null : _remove,
                icon: const Icon(Icons.delete_outline_rounded),
                label: const Text('Remove bookmark'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
