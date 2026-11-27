part of 'file_manager_screen.dart';


final class FilePreviewSheet extends ConsumerStatefulWidget {
  const FilePreviewSheet({
    super.key,
    required this.endpointId,
    required this.entry,
    required this.path,
    this.preview,
  });

  final String endpointId;
  final FileEntry entry;
  final String path;
  final Future<FilePreviewResult>? preview;

  @override
  ConsumerState<FilePreviewSheet> createState() => _FilePreviewSheetState();
}

final class _FilePreviewSheetState extends ConsumerState<FilePreviewSheet> {
  late Future<FilePreviewResult> _preview;
  Timer? _previewDeadline;

  @override
  void initState() {
    super.initState();
    _preview = _loadPreview();
  }

  Future<FilePreviewResult> _loadPreview() {
    _previewDeadline?.cancel();
    final result = Completer<FilePreviewResult>();
    final source =
        widget.preview ??
        ref
            .read(endpointSessionProvider(widget.endpointId).future)
            .then((session) => session.previewFile(widget.path));
    final timer = Timer(const Duration(seconds: 20), () {
      if (!result.isCompleted) {
        result.completeError(
          TimeoutException(
            'The file did not respond. Retry when the connection is available.',
          ),
        );
      }
    });
    _previewDeadline = timer;
    source.then(
      (value) {
        timer.cancel();
        if (!result.isCompleted) result.complete(value);
      },
      onError: (Object error, StackTrace stack) {
        timer.cancel();
        if (!result.isCompleted) result.completeError(error, stack);
      },
    );
    return result.future;
  }

  @override
  void dispose() {
    _previewDeadline?.cancel();
    super.dispose();
  }

  @override
  void didUpdateWidget(FilePreviewSheet oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.endpointId != oldWidget.endpointId ||
        widget.path != oldWidget.path ||
        widget.preview != oldWidget.preview) {
      _preview = _loadPreview();
    }
  }

  @override
  Widget build(BuildContext context) {
    return FractionallySizedBox(
      heightFactor: 0.9,
      child: Scaffold(
        appBar: AppBar(
          automaticallyImplyLeading: false,
          title: Text(
            widget.entry.name,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          actions: [
            FutureBuilder<FilePreviewResult>(
              future: _preview,
              builder: (context, snapshot) {
                final data = snapshot.data;
                if (data == null ||
                    !supportsWebFilePreview(widget.path, data)) {
                  return const SizedBox.shrink();
                }
                return IconButton(
                  tooltip: anyttyText(
                    context,
                    en: 'Browser preview',
                    zh: '浏览器预览',
                  ),
                  icon: const Icon(Icons.language_rounded),
                  onPressed: () => Navigator.of(context).push<void>(
                    MaterialPageRoute(
                      builder: (_) => FileWebPreviewScreen(
                        endpointId: widget.endpointId,
                        path: widget.path,
                        preview: data.deepCopy(),
                      ),
                    ),
                  ),
                );
              },
            ),
            IconButton(
              tooltip: 'Close preview',
              constraints: const BoxConstraints.tightFor(width: 48, height: 48),
              onPressed: () => Navigator.pop(context),
              icon: const Icon(Icons.close_rounded),
            ),
          ],
        ),
        body: FutureBuilder<FilePreviewResult>(
          future: _preview,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return Center(
                child: Semantics(
                  label: anyttyText(
                    context,
                    en: 'Loading preview',
                    zh: '正在读取预览',
                  ),
                  child: const AnyttyBrandLoader(),
                ),
              );
            }
            if (snapshot.hasError) {
              return Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        snapshot.error.toString(),
                        textAlign: TextAlign.center,
                      ),
                      const SizedBox(height: 16),
                      OutlinedButton.icon(
                        onPressed: () =>
                            setState(() => _preview = _loadPreview()),
                        icon: const Icon(Icons.refresh_rounded),
                        label: Text(anyttyText(context, en: 'Retry', zh: '重试')),
                      ),
                    ],
                  ),
                ),
              );
            }
            final preview = snapshot.requireData;
            final bytes = Uint8List.fromList(preview.content);
            final mimeType = normalizedPreviewMimeType(preview.mimeType);
            if (mimeType.startsWith('image/')) {
              return InteractiveViewer(
                minScale: 0.5,
                maxScale: 5,
                child: Center(
                  child: Image(
                    image: ResizeImage(
                      MemoryImage(bytes),
                      width: maximumPreviewImageDimension,
                      height: maximumPreviewImageDimension,
                      policy: ResizeImagePolicy.fit,
                    ),
                    fit: BoxFit.contain,
                    gaplessPlayback: true,
                    errorBuilder: (context, error, stackTrace) => const Padding(
                      padding: EdgeInsets.all(24),
                      child: Text('The image could not be decoded safely.'),
                    ),
                  ),
                ),
              );
            }
            final textLike =
                mimeType.startsWith('text/') ||
                const [
                  'application/json',
                  'application/xml',
                  'application/yaml',
                  'application/x-yaml',
                ].contains(mimeType);
            if (!textLike) {
              return Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text('Preview is not available for $mimeType.'),
                ),
              );
            }
            final text = utf8.decode(bytes, allowMalformed: true);
            return Column(
              children: [
                if (preview.truncated)
                  const MaterialBanner(
                    content: Text('Preview truncated'),
                    actions: [SizedBox.shrink()],
                  ),
                Expanded(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: SelectableText(
                      text,
                      style: const TextStyle(
                        fontFamily: 'JetBrainsMonoNerd',
                        fontSize: 13,
                        height: 1.45,
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}
