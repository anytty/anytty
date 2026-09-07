import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../data/browser_history_store.dart';

final class _AddressSuggestion {
  const _AddressSuggestion(this.value, this.title, {this.isQuery = false});
  final String value;
  final String title;
  final bool isQuery;
}

/// The field and its autocomplete menu share Flutter's text-field tap region.
/// Outside touches dismiss editing on mobile as well as desktop.
final class BrowserAddressBar extends StatelessWidget {
  const BrowserAddressBar({
    super.key,
    required this.controller,
    required this.focusNode,
    required this.enabled,
    required this.history,
    required this.onNavigate,
    required this.onDismiss,
  });
  final TextEditingController controller;
  final FocusNode focusNode;
  final bool enabled;
  final List<BrowserHistoryEntry> history;
  final Future<void> Function(String) onNavigate;
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: focusNode,
    builder: (context, _) => _build(context),
  );

  Widget _build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    var selectedOnSubmit = false;
    return CallbackShortcuts(
      bindings: {const SingleActivator(LogicalKeyboardKey.escape): onDismiss},
      child: LayoutBuilder(
        builder: (context, constraints) => RawAutocomplete<_AddressSuggestion>(
          textEditingController: controller,
          focusNode: focusNode,
          displayStringForOption: (entry) => entry.value,
          optionsBuilder: (value) {
            final text = value.text.trim(), query = text.toLowerCase();
            return [
              if (text.isNotEmpty)
                _AddressSuggestion(text, text, isQuery: true),
              ...history
                  .where(
                    (entry) =>
                        entry.url != text &&
                        (query.isEmpty ||
                            entry.url.toLowerCase().contains(query) ||
                            entry.title.toLowerCase().contains(query)),
                  )
                  .take(5)
                  .map((entry) => _AddressSuggestion(entry.url, entry.title)),
            ];
          },
          onSelected: (entry) {
            selectedOnSubmit = true;
            focusNode.unfocus();
            unawaited(onNavigate(entry.value));
          },
          optionsViewBuilder: (context, select, options) {
            final entries = options.toList(growable: false);
            return Align(
              alignment: Alignment.topLeft,
              child: SizedBox(
                width: constraints.maxWidth,
                child: Material(
                  key: const ValueKey('browser-address-suggestions'),
                  color: palette.surface,
                  elevation: 4,
                  clipBehavior: Clip.antiAlias,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(8),
                    side: BorderSide(color: palette.border),
                  ),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxHeight: 280),
                    child: ListView.builder(
                      shrinkWrap: true,
                      keyboardDismissBehavior:
                          ScrollViewKeyboardDismissBehavior.onDrag,
                      padding: const EdgeInsets.symmetric(vertical: 4),
                      itemCount: entries.length,
                      itemBuilder: (context, index) {
                        final entry = entries[index];
                        return ListTile(
                          selected:
                              AutocompleteHighlightedOption.of(context) ==
                              index,
                          selectedTileColor: palette.surfaceRaised,
                          minTileHeight: 48,
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 12,
                          ),
                          leading: Icon(
                            entry.isQuery
                                ? Icons.arrow_forward_rounded
                                : Icons.history_rounded,
                            color: palette.muted,
                            size: 18,
                          ),
                          title: Text(
                            entry.title.isEmpty ? entry.value : entry.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(color: palette.text, fontSize: 14),
                          ),
                          subtitle: entry.isQuery
                              ? null
                              : Text(
                                  entry.value,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: TextStyle(
                                    color: palette.muted,
                                    fontSize: 12,
                                  ),
                                ),
                          onTap: () => select(entry),
                        );
                      },
                    ),
                  ),
                ),
              ),
            );
          },
          fieldViewBuilder: (context, text, focus, submit) => Focus(
            canRequestFocus: false,
            onKeyEvent: (_, event) {
              if (event is KeyDownEvent &&
                  event.logicalKey == LogicalKeyboardKey.escape) {
                onDismiss();
                return KeyEventResult.handled;
              }
              return KeyEventResult.ignored;
            },
            child: SizedBox(
              height: 44,
              child: TextField(
                key: const ValueKey('browser-address-field'),
                controller: text,
                focusNode: focus,
                enabled: enabled,
                onTapOutside: (_) => onDismiss(),
                onEditingComplete: () {},
                onSubmitted: (value) {
                  if (value.trim().isEmpty) return;
                  selectedOnSubmit = false;
                  submit();
                  if (!selectedOnSubmit) {
                    focus.unfocus();
                    unawaited(onNavigate(value));
                  }
                },
                textInputAction: TextInputAction.go,
                keyboardType: TextInputType.url,
                autocorrect: false,
                enableSuggestions: false,
                maxLines: 1,
                style: TextStyle(color: palette.text, fontSize: 14),
                decoration: InputDecoration(
                  hintText: anyttyText(
                    context,
                    en: 'Search or enter address',
                    zh: '搜索或输入网址',
                  ),
                  hintStyle: TextStyle(color: palette.muted, fontSize: 14),
                  prefixIcon: Icon(
                    focus.hasFocus
                        ? Icons.search_rounded
                        : Icons.public_rounded,
                    size: 18,
                    color: palette.muted,
                  ),
                  prefixIconConstraints: const BoxConstraints(
                    minWidth: 34,
                    minHeight: 44,
                  ),
                  suffixIcon: focus.hasFocus
                      ? IconButton(
                          tooltip: anyttyText(
                            context,
                            en: 'Clear address',
                            zh: '清空地址',
                          ),
                          onPressed: text.clear,
                          icon: const Icon(Icons.close_rounded, size: 18),
                          color: palette.muted,
                        )
                      : null,
                  filled: true,
                  fillColor: palette.surfaceRaised,
                  isDense: true,
                  contentPadding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 10,
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
                    borderSide: BorderSide(color: palette.accent, width: 1.2),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
