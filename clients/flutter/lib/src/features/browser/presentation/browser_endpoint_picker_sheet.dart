import 'package:flutter/material.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/anytty_ui.dart';

final class BrowserEndpointOption {
  const BrowserEndpointOption({
    required this.endpointId,
    required this.label,
    required this.current,
  });

  final String endpointId;
  final String label;
  final bool current;
}

Future<String?> showAnyttyBrowserEndpointPicker({
  required BuildContext context,
  required List<BrowserEndpointOption> endpoints,
}) => showModalBottomSheet<String>(
  context: context,
  showDragHandle: true,
  isScrollControlled: true,
  backgroundColor: AnyttyPalette.of(context).surface,
  barrierColor: AnyttyPalette.of(context).overlay,
  builder: (context) => _BrowserEndpointPicker(endpoints: endpoints),
);

final class _BrowserEndpointPicker extends StatelessWidget {
  const _BrowserEndpointPicker({required this.endpoints});

  final List<BrowserEndpointOption> endpoints;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SafeArea(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxHeight: 520),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
              child: Text(
                anyttyText(context, en: 'Web sessions', zh: 'Web 会话'),
                style: AnyttyUi.title(context),
              ),
            ),
            Flexible(
              child: ListView.separated(
                shrinkWrap: true,
                padding: const EdgeInsets.only(bottom: 16),
                itemCount: endpoints.length,
                separatorBuilder: (_, _) =>
                    Divider(height: 1, indent: 68, color: palette.track),
                itemBuilder: (context, index) {
                  final endpoint = endpoints[index];
                  return ListTile(
                    selected: endpoint.current,
                    minTileHeight: 76,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 20),
                    leading: Container(
                      width: 36,
                      height: 36,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: endpoint.current
                            ? palette.accent.withValues(alpha: 0.12)
                            : palette.surfaceRaised,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Icon(
                        Icons.language_rounded,
                        size: 19,
                        color: endpoint.current
                            ? palette.accent
                            : palette.strong,
                      ),
                    ),
                    title: Text(
                      endpoint.label,
                      style: AnyttyUi.body(context).copyWith(
                        color: palette.text,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    subtitle: Text(
                      endpoint.endpointId,
                      style: AnyttyUi.muted(context),
                    ),
                    trailing: endpoint.current
                        ? Icon(Icons.check_rounded, color: palette.accent)
                        : null,
                    onTap: () => Navigator.of(context).pop(endpoint.endpointId),
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
