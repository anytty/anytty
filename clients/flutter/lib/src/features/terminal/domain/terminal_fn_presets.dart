final class TerminalFnItem {
  const TerminalFnItem({
    required this.label,
    required this.data,
    this.description,
  });

  final String label;
  final String data;
  final String? description;
}

final class TerminalFnGroup {
  const TerminalFnGroup({required this.name, required this.items});

  final String name;
  final List<TerminalFnItem> items;
}

final class TerminalFnPreset {
  const TerminalFnPreset({
    required this.id,
    required this.name,
    required this.matches,
    required this.groups,
  });

  final String id;
  final String name;
  final List<String> matches;
  final List<TerminalFnGroup> groups;
}

const systemTerminalFnGroups = <TerminalFnGroup>[
  TerminalFnGroup(
    name: 'Process',
    items: [
      TerminalFnItem(label: 'Ctrl+C', data: '\x03', description: 'Interrupt'),
      TerminalFnItem(label: 'Ctrl+D', data: '\x04', description: 'EOF'),
      TerminalFnItem(label: 'Ctrl+Z', data: '\x1a', description: 'Suspend'),
      TerminalFnItem(label: 'Ctrl+L', data: '\x0c', description: 'Clear'),
      TerminalFnItem(label: 'Ctrl+R', data: '\x12', description: 'History'),
      TerminalFnItem(label: r'Ctrl+\', data: '\x1c', description: 'Quit'),
    ],
  ),
  TerminalFnGroup(
    name: 'Line editing',
    items: [
      TerminalFnItem(label: 'Ctrl+A', data: '\x01', description: 'Line start'),
      TerminalFnItem(label: 'Ctrl+E', data: '\x05', description: 'Line end'),
      TerminalFnItem(label: 'Ctrl+W', data: '\x17', description: 'Delete word'),
      TerminalFnItem(label: 'Ctrl+U', data: '\x15', description: 'Delete line'),
      TerminalFnItem(label: 'Ctrl+K', data: '\x0b', description: 'Delete tail'),
      TerminalFnItem(
        label: 'Delete',
        data: '\x1b[3~',
        description: 'Forward delete',
      ),
      TerminalFnItem(label: 'Tab', data: '\t', description: 'Complete'),
    ],
  ),
];

const _programTerminalFnPresets = <TerminalFnPreset>[
  TerminalFnPreset(
    id: 'claude',
    name: 'Claude Code',
    matches: ['claude'],
    groups: [
      TerminalFnGroup(
        name: 'Commands',
        items: [
          TerminalFnItem(
            label: '/clear',
            data: '/clear\n',
            description: 'Reset context',
          ),
          TerminalFnItem(
            label: '/compact',
            data: '/compact\n',
            description: 'Compact',
          ),
          TerminalFnItem(label: '/cost', data: '/cost\n', description: 'Usage'),
          TerminalFnItem(label: '/help', data: '/help\n', description: 'Help'),
          TerminalFnItem(
            label: '/review',
            data: '/review\n',
            description: 'Review',
          ),
          TerminalFnItem(label: '/init', data: '/init\n', description: 'Init'),
        ],
      ),
      TerminalFnGroup(
        name: 'Replies',
        items: [
          TerminalFnItem(label: 'yes', data: 'yes\n', description: 'Confirm'),
          TerminalFnItem(label: 'no', data: 'no\n', description: 'Decline'),
          TerminalFnItem(label: 'exit', data: 'exit\n', description: 'Exit'),
        ],
      ),
    ],
  ),
  TerminalFnPreset(
    id: 'opencode',
    name: 'OpenCode',
    matches: ['opencode'],
    groups: [
      TerminalFnGroup(
        name: 'Commands',
        items: [
          TerminalFnItem(
            label: '/clear',
            data: '/clear\n',
            description: 'Reset context',
          ),
          TerminalFnItem(
            label: '/compact',
            data: '/compact\n',
            description: 'Compact',
          ),
          TerminalFnItem(label: '/cost', data: '/cost\n', description: 'Usage'),
          TerminalFnItem(label: '/help', data: '/help\n', description: 'Help'),
        ],
      ),
      TerminalFnGroup(
        name: 'Replies',
        items: [
          TerminalFnItem(label: 'yes', data: 'yes\n', description: 'Confirm'),
          TerminalFnItem(label: 'no', data: 'no\n', description: 'Decline'),
        ],
      ),
    ],
  ),
];

TerminalFnPreset? matchTerminalFnPreset(String? command) {
  if (command == null || command.trim().isEmpty) return null;
  final normalized = command.toLowerCase();
  for (final preset in _programTerminalFnPresets) {
    if (preset.matches.any(normalized.contains)) return preset;
  }
  return null;
}
