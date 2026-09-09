import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../generated/proto/apipb/file.pb.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import '../domain/file_web_preview.dart';

final class FileWebPreviewScreen extends StatefulWidget {
  const FileWebPreviewScreen({
    super.key,
    required this.endpointId,
    required this.path,
    required this.preview,
  });

  final String endpointId;
  final String path;
  final FilePreviewResult preview;

  @override
  State<FileWebPreviewScreen> createState() => _FileWebPreviewScreenState();
}

final class _FileWebPreviewScreenState extends State<FileWebPreviewScreen> {
  WebViewController? _controller;
  Timer? _deadline;
  bool _started = false;
  bool _loading = true;
  String? _error;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (!_started) {
      _started = true;
      unawaited(_load());
    }
  }

  Future<void> _load() async {
    final dark = Theme.of(context).brightness == Brightness.dark;
    final chinese = AnyttyLocalizations.of(context).isChinese;
    setState(() {
      _loading = true;
      _error = null;
    });
    _deadline?.cancel();
    _deadline = Timer(const Duration(seconds: 20), () {
      if (mounted && _loading) {
        setState(() {
          _loading = false;
          _error = 'Preview timed out';
        });
      }
    });
    try {
      final template = await rootBundle.loadString(
        'assets/file-viewer/viewer.html',
      );
      if (!mounted) return;
      final document = buildFileWebPreview(
        template: template,
        path: widget.path,
        preview: widget.preview,
        dark: dark,
        chinese: chinese,
      );
      final controller = WebViewController();
      await controller.setJavaScriptMode(JavaScriptMode.unrestricted);
      await controller.setNavigationDelegate(
        NavigationDelegate(
          onNavigationRequest: (request) =>
              const {'about:blank', 'about:srcdoc'}.contains(request.url)
              ? NavigationDecision.navigate
              : NavigationDecision.prevent,
          onPageFinished: (_) {
            _deadline?.cancel();
            if (mounted) setState(() => _loading = false);
          },
          onWebResourceError: (error) {
            if (error.isForMainFrame != true) return;
            _deadline?.cancel();
            if (mounted) {
              setState(() {
                _loading = false;
                _error = error.description;
              });
            }
          },
        ),
      );
      // Only the bundled renderer runs scripts. HTML files are nested in a
      // sandbox without allow-scripts. No JavaScript channels are registered.
      await controller.loadHtmlString(document);
      if (mounted) setState(() => _controller = controller);
    } catch (error) {
      _deadline?.cancel();
      if (mounted) {
        setState(() {
          _loading = false;
          _error = error.toString();
        });
      }
    }
  }

  @override
  void dispose() {
    _deadline?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.path.split('/').last,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        actions: [
          IconButton(
            tooltip: anyttyText(context, en: 'Back to source', zh: '返回源码'),
            onPressed: () => Navigator.pop(context),
            icon: const Icon(Icons.code_rounded),
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Stack(
          children: [
            if (_controller case final controller?)
              Positioned.fill(child: WebViewWidget(controller: controller)),
            if (_loading || _error != null)
              Positioned.fill(
                child: ColoredBox(
                  color: palette.background,
                  child: Center(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.all(24),
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          if (_error == null)
                            const AnyttyBrandLoader()
                          else
                            const AnyttyBrandLoader(scene: AnyttyMascotScene.failure),
                          const SizedBox(height: 16),
                          Text(
                            _error ??
                                anyttyText(
                                  context,
                                  en: 'Opening preview',
                                  zh: '正在打开预览',
                                ),
                            textAlign: TextAlign.center,
                          ),
                          if (_error != null)
                            TextButton.icon(
                              onPressed: _load,
                              icon: const Icon(Icons.refresh_rounded),
                              label: Text(
                                anyttyText(context, en: 'Retry', zh: '重试'),
                              ),
                            ),
                        ],
                      ),
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
