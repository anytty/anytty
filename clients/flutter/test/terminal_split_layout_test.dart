import 'package:anytty_native/src/features/terminal/domain/terminal_split_layout.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('splits the targeted pane and resolves terminal ids', () {
    final root = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );

    expect(terminalPaneKeys(root), ['primary', 'terminal:logs']);
    expect(terminalIdForPane('primary', 'shell'), 'shell');
    expect(terminalIdForPane('terminal:logs', 'shell'), 'logs');
    final branch = root as TerminalSplitBranch;
    expect(branch.direction, TerminalSplitDirection.rows);
    expect(branch.ratio, 50);
  });

  test('supports before placement for left and above splits', () {
    final root = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.columns,
      splitId: 'split-1',
      placement: TerminalSplitPlacement.before,
    ) as TerminalSplitBranch;

    expect(terminalPaneKeys(root), ['terminal:logs', 'primary']);
  });

  test('creates an empty pane and assigns a terminal only after selection', () {
    final empty = splitTerminalPaneEmpty(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );

    expect(terminalPaneKeys(empty), ['primary', 'empty:split-1']);
    expect(terminalIdForPane('empty:split-1', 'shell'), isNull);
    expect(isEmptyTerminalPaneKey('empty:split-1'), isTrue);

    final assigned = assignTerminalToPane(
      root: empty,
      targetPaneKey: 'empty:split-1',
      terminalId: 'logs',
    );
    expect(terminalPaneKeys(assigned), ['primary', 'terminal:logs']);
  });

  test('moves an existing pane instead of duplicating it', () {
    final first = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );
    final second = splitTerminalPane(
      root: first,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'server',
      direction: TerminalSplitDirection.columns,
      splitId: 'split-2',
    );
    final moved = splitTerminalPane(
      root: second,
      targetPaneKey: 'terminal:server',
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-3',
      placement: TerminalSplitPlacement.before,
    );

    expect(terminalPaneKeys(moved), [
      'primary',
      'terminal:logs',
      'terminal:server',
    ]);
  });

  test('removing a leaf collapses its parent split', () {
    final root = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );

    final removed = removeTerminalPane(root, 'terminal:logs');
    expect(terminalPaneKeys(removed!), [primaryTerminalPaneKey]);
    expect(removeTerminalPane(root, primaryTerminalPaneKey), same(root));
  });

  test('updates nested ratios and clamps them to the Web limits', () {
    final first = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );
    final nested = splitTerminalPane(
      root: first,
      targetPaneKey: 'terminal:logs',
      terminalId: 'server',
      direction: TerminalSplitDirection.columns,
      splitId: 'split-2',
    );

    final low = updateTerminalSplitRatio(nested, 'split-2', 2);
    final high = updateTerminalSplitRatio(low, 'split-1', 97);
    final outer = high as TerminalSplitBranch;
    final inner = outer.second as TerminalSplitBranch;
    expect(outer.ratio, 80);
    expect(inner.ratio, 20);
  });

  test('keeps the tree unchanged for invalid or self targets', () {
    final invalid = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: 'terminal:missing',
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-1',
    );
    final withLogs = splitTerminalPane(
      root: primaryTerminalPane,
      targetPaneKey: primaryTerminalPaneKey,
      terminalId: 'logs',
      direction: TerminalSplitDirection.rows,
      splitId: 'split-2',
    );
    final self = splitTerminalPane(
      root: withLogs,
      targetPaneKey: 'terminal:logs',
      terminalId: 'logs',
      direction: TerminalSplitDirection.columns,
      splitId: 'split-3',
    );

    expect(invalid, same(primaryTerminalPane));
    expect(self, same(withLogs));
  });
}
