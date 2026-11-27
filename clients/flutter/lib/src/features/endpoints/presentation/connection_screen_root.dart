part of 'connection_screen.dart';


final class ConnectionScreen extends ConsumerStatefulWidget {
  const ConnectionScreen({super.key, required this.endpointId, this.label});

  final String endpointId;
  final String? label;

  @override
  ConsumerState<ConnectionScreen> createState() => _ConnectionScreenState();
}

final class _ConnectionScreenState extends ConsumerState<ConnectionScreen> {
  ConnectionPolicy? _draft;
  bool _applying = false;
  bool _refreshing = false;
  String? _actionError;

  String get _label {
    final value = widget.label?.trim() ?? '';
    return value.isEmpty ? widget.endpointId : value;
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final policyAsync = ref.watch(connectionPolicyProvider(widget.endpointId));
    final diagnosticsAsync = ref.watch(
      connectionDiagnosticsProvider(widget.endpointId),
    );
    final savedPolicy = policyAsync.valueOrNull?.policy;
    final effectiveDraft = _draft ?? savedPolicy?.deepCopy();
    final changed =
        savedPolicy != null &&
        effectiveDraft != null &&
        !_samePolicy(savedPolicy, effectiveDraft);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back to devices',
          onPressed: () => Navigator.maybePop(context),
          icon: const Icon(Icons.arrow_back_rounded),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              anyttyText(context, en: 'Connection', zh: '网络连接'),
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
            ),
            Text(
              _label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: palette.muted,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: 'Refresh connection diagnostics',
            onPressed: _refreshing || _applying ? null : _refresh,
            icon: _refreshing
                ? const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.refresh_rounded),
          ),
          const SizedBox(width: 4),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
        children: [
          _SectionLabel(
            label: anyttyText(context, en: 'CURRENT PATH', zh: '当前连接'),
          ),
          const SizedBox(height: 8),
          _ConnectionSummary(diagnostics: diagnosticsAsync),
          if (_actionError != null) ...[
            const SizedBox(height: 12),
            _InlineMessage(
              message: _actionError!,
              icon: Icons.error_outline_rounded,
              color: palette.danger,
              onClose: () => setState(() => _actionError = null),
            ),
          ],
          const SizedBox(height: 24),
          _SectionLabel(
            label: anyttyText(context, en: 'ROUTE PREFERENCE', zh: '路由偏好'),
          ),
          const SizedBox(height: 6),
          Text(
            anyttyText(
              context,
              en: 'Auto chooses the best available route.',
              zh: '自动模式会选择当前最合适的可用路由。',
            ),
            style: TextStyle(color: palette.muted, fontSize: 12, height: 1.4),
          ),
          const SizedBox(height: 10),
          if (policyAsync.isLoading && policyAsync.valueOrNull == null)
            const _PolicyLoading()
          else if (policyAsync.hasError && policyAsync.valueOrNull == null)
            _PolicyError(
              message: policyAsync.error.toString(),
              onRetry: _refresh,
            )
          else if (policyAsync.valueOrNull case final state?)
            _RoutePreferenceList(
              state: state,
              value: effectiveDraft!.routePreference,
              enabled: !_applying,
              onChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.routePreference = value,
              ),
            ),
          if (policyAsync.valueOrNull case final state?) ...[
            const SizedBox(height: 24),
            _SectionLabel(
              label: anyttyText(context, en: 'CLOUD PATH', zh: '云端路径'),
            ),
            const SizedBox(height: 10),
            _CloudPolicyControls(
              state: state,
              policy: effectiveDraft!,
              enabled: !_applying,
              onCloudChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.cloudRelayMode = value,
              ),
              onTransportChanged: (value) => _updateDraft(
                effectiveDraft,
                (next) => next.relayTransport = value,
              ),
            ),
          ],
          const SizedBox(height: 24),
          _SectionLabel(
            label: anyttyText(context, en: 'CONNECTION ROUTES', zh: '连接路由'),
          ),
          const SizedBox(height: 8),
          Material(
            color: palette.surface,
            clipBehavior: Clip.antiAlias,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
              side: BorderSide(color: palette.border),
            ),
            child: ListTile(
              leading: const Icon(Icons.alt_route_rounded),
              title: Text(anyttyText(context, en: 'Manage routes', zh: '管理路由')),
              subtitle: Text(
                anyttyText(
                  context,
                  en: 'Add, order, and test routes',
                  zh: '添加、排序和测试连接路由',
                ),
              ),
              trailing: Icon(Icons.chevron_right_rounded, color: palette.faint),
              onTap: () => context.push(
                Uri(
                  path: '/routes/${Uri.encodeComponent(widget.endpointId)}',
                  queryParameters: {'label': _label},
                ).toString(),
              ),
            ),
          ),
          const SizedBox(height: 24),
          _DiagnosticsSection(
            diagnostics: diagnosticsAsync,
            onCopyRedacted: _copyRedactedDiagnostics,
          ),
        ],
      ),
      bottomNavigationBar: _ConnectionFooter(
        changed: changed,
        applying: _applying,
        enabled: savedPolicy != null,
        onApply: effectiveDraft == null ? null : () => _apply(effectiveDraft),
      ),
    );
  }

  void _updateDraft(
    ConnectionPolicy current,
    void Function(ConnectionPolicy next) update,
  ) {
    final next = current.deepCopy();
    update(next);
    setState(() {
      _draft = next;
      _actionError = null;
    });
  }

  Future<void> _refresh() async {
    if (_refreshing || _applying) return;
    setState(() {
      _refreshing = true;
      _actionError = null;
    });
    try {
      await Future.wait<Object>([
        ref.refresh(connectionPolicyProvider(widget.endpointId).future),
        ref.refresh(connectionDiagnosticsProvider(widget.endpointId).future),
      ]);
      if (mounted) setState(() => _draft = null);
    } catch (_) {
      // Each independent provider renders its own actionable error state.
    } finally {
      if (mounted) setState(() => _refreshing = false);
    }
  }

  Future<void> _apply(ConnectionPolicy draft) async {
    if (_applying) return;
    setState(() {
      _applying = true;
      _actionError = null;
    });
    var policySaved = false;
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      await ConnectionRepository(runtime).applyPolicy(widget.endpointId, draft);
      policySaved = true;
      if (mounted) setState(() => _draft = null);
      ref.invalidate(connectionPolicyProvider(widget.endpointId));

      await EndpointRepository(runtime).disconnectEndpoint(widget.endpointId);
      ref.invalidate(endpointSessionProvider(widget.endpointId));
      ref.invalidate(connectionDiagnosticsProvider(widget.endpointId));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Connection policy applied')),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _actionError = policySaved
            ? 'Policy was saved, but reconnect failed: $error'
            : error.toString();
      });
    } finally {
      if (mounted) setState(() => _applying = false);
    }
  }

  Future<void> _copyRedactedDiagnostics() async {
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final current = ref
          .read(connectionDiagnosticsProvider(widget.endpointId))
          .valueOrNull;
      final report = runtime.buildRedactedDiagnostics(
        endpointId: widget.endpointId,
        session: current?.session,
      );
      await Clipboard.setData(ClipboardData(text: report));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Redacted diagnostics copied')),
      );
    } catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Could not copy diagnostics: $error')),
      );
    }
  }
}
