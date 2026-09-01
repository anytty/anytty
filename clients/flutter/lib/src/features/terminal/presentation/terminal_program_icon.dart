import 'package:flutter/widgets.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../generated/proto/apipb/terminal.pb.dart';

enum TerminalProgramKind {
  shell,
  editor,
  agent,
  runtime,
  container,
  database,
  remote,
  git,
  monitor,
  terminal,
}

TerminalProgramKind terminalProgramKind(TerminalInfo terminal) {
  final name = terminalProgramName(terminal).toLowerCase();
  if ({
    'bash',
    'zsh',
    'fish',
    'sh',
    'dash',
    'ksh',
    'tmux',
    'screen',
    'pwsh',
    'powershell',
  }.contains(name)) {
    return TerminalProgramKind.shell;
  }
  if ({'vi', 'vim', 'nvim', 'nano', 'emacs', 'helix', 'hx'}.contains(name)) {
    return TerminalProgramKind.editor;
  }
  if ({'codex', 'claude', 'opencode', 'aider', 'gemini'}.contains(name)) {
    return TerminalProgramKind.agent;
  }
  if ({
    'python',
    'python3',
    'node',
    'deno',
    'bun',
    'ruby',
    'php',
    'java',
    'go',
    'cargo',
    'rustc',
  }.contains(name)) {
    return TerminalProgramKind.runtime;
  }
  if ({'docker', 'podman', 'kubectl', 'k9s', 'nerdctl'}.contains(name)) {
    return TerminalProgramKind.container;
  }
  if ({
    'mysql',
    'psql',
    'postgres',
    'sqlite3',
    'redis-cli',
    'mongosh',
  }.contains(name)) {
    return TerminalProgramKind.database;
  }
  if ({'ssh', 'mosh', 'telnet'}.contains(name)) {
    return TerminalProgramKind.remote;
  }
  if ({'git', 'lazygit', 'tig', 'gh'}.contains(name)) {
    return TerminalProgramKind.git;
  }
  if ({'top', 'htop', 'btop', 'glances', 'watch'}.contains(name)) {
    return TerminalProgramKind.monitor;
  }
  return TerminalProgramKind.terminal;
}

String terminalProgramName(TerminalInfo terminal) {
  final source = terminal.foregroundProcess.trim().isNotEmpty
      ? terminal.foregroundProcess.trim()
      : terminal.command.isNotEmpty
      ? terminal.command.first.trim()
      : '';
  if (source.isEmpty) return '';
  final executable = source.split(RegExp(r'\s+')).first;
  final normalized = executable.replaceAll('\\', '/');
  return normalized.split('/').last;
}

IconData terminalProgramIcon(TerminalInfo terminal) =>
    switch (terminalProgramKind(terminal)) {
      TerminalProgramKind.shell => LucideIcons.squareTerminal,
      TerminalProgramKind.editor => LucideIcons.fileCode,
      TerminalProgramKind.agent => LucideIcons.bot,
      TerminalProgramKind.runtime => LucideIcons.codeXml,
      TerminalProgramKind.container => LucideIcons.container,
      TerminalProgramKind.database => LucideIcons.database,
      TerminalProgramKind.remote => LucideIcons.server,
      TerminalProgramKind.git => LucideIcons.gitBranch,
      TerminalProgramKind.monitor => LucideIcons.activity,
      TerminalProgramKind.terminal => LucideIcons.terminal,
    };
