import 'dart:async';

import 'package:flutter/material.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_ui.dart';
import '../data/path_bookmark_store.dart';

Future<String?> showPathBookmarks({
  required BuildContext context,
  required String endpointId,
  required String currentPath,
  PathBookmarkStore store = const PathBookmarkStore(),
}) => showModalBottomSheet<String>(
  context: context,
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
    return FractionallySizedBox(
      heightFactor: 0.9,
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
                      Text('Bookmarks', style: AnyttyUi.title(context)),
                      const SizedBox(height: 2),
                      Text(
                        _loading ? 'Loading' : '${_bookmarks.length} saved',
                        style: AnyttyUi.muted(context),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ),
                ),
                AnyttyIconButton(
                  tooltip: 'Close bookmarks',
                  onPressed: () => Navigator.pop(context),
                  icon: Icons.close_rounded,
                ),
              ],
            ),
          ),
          Divider(height: 1, color: palette.track),
          if (_error case final error?)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 8, 12, 8),
              child: Row(
                children: [
                  Expanded(
                    child: Semantics(
                      container: true,
                      liveRegion: true,
                      child: Text(
                        error,
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: AnyttyUi.body(context)
                            .copyWith(color: palette.danger),
                      ),
                    ),
                  ),
                  AnyttyIconButton(
                    tooltip: 'Retry bookmarks',
                    onPressed: _load,
                    icon: Icons.refresh_rounded,
                  ),
                ],
              ),
            ),
          ListTile(
            leading: Icon(
              Icons.bookmark_add_outlined,
              color: _saving ? palette.muted : palette.strong,
            ),
            title: Text(
              'Save current folder',
              style: AnyttyUi.body(context).copyWith(
                color: _saving ? palette.muted : palette.text,
                fontWeight: FontWeight.w600,
              ),
            ),
            subtitle: Text(
              widget.currentPath,
              style: AnyttyUi.muted(context)
                  .copyWith(color: _saving ? palette.muted : palette.muted),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            enabled: !_saving,
            onTap: _saveCurrent,
          ),
          Divider(height: 1, color: palette.track),
          Expanded(
            child: _loading && _bookmarks.isEmpty
                ? const Center(child: CircularProgressIndicator(strokeWidth: 2))
                : _bookmarks.isEmpty
                ? Center(
                    child: Text(
                      'No saved bookmarks',
                      style: AnyttyUi.muted(context),
                    ),
                  )
                : ListView.separated(
                    itemCount: _bookmarks.length,
                    separatorBuilder: (_, _) =>
                        Divider(height: 1, color: palette.track),
                    itemBuilder: (context, index) {
                      final bookmark = _bookmarks[index];
                      return Padding(
                        padding: const EdgeInsets.only(left: 20, right: 8),
                        child: Row(
                          children: [
                            Icon(
                              Icons.bookmark_border_rounded,
                              color: palette.strong,
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: InkWell(
                                borderRadius: BorderRadius.circular(12),
                                onTap: () =>
                                    Navigator.pop(context, bookmark.path),
                                child: Padding(
                                  padding: const EdgeInsets.symmetric(
                                    vertical: 10,
                                  ),
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        bookmark.label,
                                        style: AnyttyUi.body(
                                          context,
                                        ).copyWith(fontWeight: FontWeight.w600),
                                        maxLines: 2,
                                        overflow: TextOverflow.ellipsis,
                                      ),
                                      const SizedBox(height: 2),
                                      Text(
                                        bookmark.path,
                                        style: AnyttyUi.muted(context),
                                        maxLines: 2,
                                        overflow: TextOverflow.ellipsis,
                                      ),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                            AnyttyIconButton(
                              tooltip: 'Edit bookmark ${bookmark.label}',
                              onPressed: () => _edit(bookmark),
                              icon: Icons.edit_outlined,
                            ),
                          ],
                        ),
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
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) {
        final palette = AnyttyPalette.of(context);
        return Dialog(
          backgroundColor: Colors.transparent,
          elevation: 0,
          child: AnyttyCard(
            radius: 16,
            depth: 2,
            padding: const EdgeInsets.fromLTRB(20, 20, 20, 12),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Remove bookmark?', style: AnyttyUi.sectionTitle(context)),
                const SizedBox(height: 10),
                ConstrainedBox(
                  constraints: BoxConstraints(
                    maxHeight: MediaQuery.sizeOf(context).height * 0.25,
                  ),
                  child: SingleChildScrollView(
                    child: Text(
                      widget.bookmark.path,
                      style: AnyttyUi.body(context),
                    ),
                  ),
                ),
                const SizedBox(height: 18),
                Wrap(
                  alignment: WrapAlignment.end,
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    AnyttyPillButton(
                      outlined: true,
                      label: 'Cancel',
                      onPressed: () => Navigator.pop(context),
                    ),
                    AnyttyPillButton(
                      label: 'Remove',
                      color: palette.danger.withValues(alpha: 0.14),
                      foregroundColor: palette.danger,
                      onPressed: () => Navigator.pop(context, true),
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
    if (confirmed != true || !mounted) return;
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
            Text('Edit bookmark', style: AnyttyUi.title(context)),
            const SizedBox(height: 4),
            SizedBox(
              width: double.infinity,
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Text(
                  widget.bookmark.path,
                  softWrap: false,
                  style: AnyttyUi.muted(context),
                ),
              ),
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
                  child: AnyttyPillButton(
                    outlined: true,
                    label: 'Cancel',
                    onPressed: _pending ? null : () => Navigator.pop(context),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: AnyttyPillButton(
                    label: 'Save',
                    onPressed: _pending ? null : _save,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            SizedBox(
              width: double.infinity,
              child: AnyttyPillButton(
                label: 'Remove bookmark',
                icon: Icons.delete_outline_rounded,
                color: palette.danger.withValues(alpha: 0.12),
                foregroundColor: palette.danger,
                onPressed: _pending ? null : _remove,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
