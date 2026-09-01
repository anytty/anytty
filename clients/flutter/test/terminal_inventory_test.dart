import 'package:anytty_native/src/features/terminal/domain/terminal_inventory.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final terminals = [
    _terminal('running-api', TerminalState.TERMINAL_STATE_RUNNING, {
      'project': 'api',
      'owner': 'platform',
      'tag1': 'deploy=blue',
    }),
    _terminal('created-web', TerminalState.TERMINAL_STATE_CREATED, {
      'project': 'web',
      'tag1': 'frontend',
      'anytty.size_lock': 'lock',
    }),
    _terminal('exited-api', TerminalState.TERMINAL_STATE_EXITED, {
      'project': 'api',
      'tag2': 'deploy=blue',
      'cwd': '/srv/api',
    }),
  ];

  test('filters running and exited state with exact public tags', () {
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.running,
      ).map((terminal) => terminal.ref.terminalId),
      ['running-api', 'created-web'],
    );
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.exited,
      ).map((terminal) => terminal.ref.terminalId),
      ['exited-api'],
    );
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.all,
        tagIds: {'project=api', 'deploy=blue'},
      ).map((terminal) => terminal.ref.terminalId),
      ['running-api', 'exited-api'],
    );
  });

  test('counts public tags and hides daemon-owned metadata', () {
    expect(
      terminalTagOptions(terminals)
          .map((option) => '${option.label}:${option.count}'),
      [
        'deploy=blue:2',
        'frontend:1',
        'owner=platform:1',
        'project=api:2',
        'project=web:1',
      ],
    );
  });

  test('searches terminal names, ids, commands, paths, and tags', () {
    terminals.first
      ..name = 'API worker'
      ..foregroundProcess = 'go'
      ..liveCwd = '/workspace/backend';
    terminals[1].command.addAll(['npm', 'run', 'dev']);

    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.all,
        query: 'BACKEND',
      ),
      [terminals.first],
    );
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.all,
        query: 'npm run',
      ),
      [terminals[1]],
    );
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.all,
        query: 'frontend',
      ),
      [terminals[1]],
    );
    expect(
      filterTerminals(
        terminals: terminals,
        status: TerminalStatusFilter.all,
        query: 'APwrkr',
      ),
      [terminals.first],
    );
  });

  test('pins sort first and can be reordered locally', () {
    final pinned = toggleTerminalPin(
      toggleTerminalPin(const [], 'exited-api'),
      'running-api',
    );
    expect(
      sortPinnedTerminals(
        terminals,
        pinned,
      ).map((terminal) => terminal.ref.terminalId),
      ['exited-api', 'running-api', 'created-web'],
    );
    expect(movePinnedTerminal(pinned, 'running-api', -1), [
      'running-api',
      'exited-api',
    ]);
    expect(toggleTerminalPin(pinned, 'exited-api'), ['running-api']);
  });

  test('exited and removed terminals open without a live attachment', () {
    expect(
      terminalUsesHistoryOnly(TerminalState.TERMINAL_STATE_EXITED),
      isTrue,
    );
    expect(
      terminalUsesHistoryOnly(TerminalState.TERMINAL_STATE_REMOVED),
      isTrue,
    );
    expect(
      terminalUsesHistoryOnly(TerminalState.TERMINAL_STATE_RUNNING),
      isFalse,
    );
  });
}

TerminalInfo _terminal(
  String id,
  TerminalState state,
  Map<String, String> tags,
) {
  return TerminalInfo(
    ref: TerminalRef(endpointId: 'endpoint-1', terminalId: id),
    state: state,
    tags: tags.entries,
  );
}
