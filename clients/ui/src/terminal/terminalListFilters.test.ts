import { describe, expect, it } from 'vitest'
import type { Terminal } from '../core/model'
import { filterTerminals, publicTerminalTags, terminalTagId, terminalTagOptions } from './terminalListFilters'

describe('terminal list filters', () => {
  const terminals: Terminal[] = [
    terminal('running-api', 'running', { project: 'api', owner: 'platform', tag1: 'deploy=blue', cwd: '/srv/api' }),
    terminal('running-web', 'running', { project: 'web', tag1: 'frontend', 'anytty.size_lock': 'lock' }),
    terminal('exited-api', 'exited', { project: 'api', tag2: 'deploy=blue' }),
  ]

  it('filters status and exact tag values together', () => {
    expect(filterTerminals(terminals, 'running').map((terminal) => terminal.terminalId)).toEqual(['running-api', 'running-web'])
    expect(filterTerminals(terminals, 'exited').map((terminal) => terminal.terminalId)).toEqual(['exited-api'])
    expect(filterTerminals(terminals, 'all', [terminalTagId('project=api')]).map((terminal) => terminal.terminalId)).toEqual(['running-api', 'exited-api'])
    expect(filterTerminals(terminals, 'running', [terminalTagId('project=api'), terminalTagId('deploy=blue')]).map((terminal) => terminal.terminalId)).toEqual(['running-api'])
  })

  it('counts public tag options and excludes daemon-owned metadata', () => {
    expect(terminalTagOptions(terminals)).toEqual([
      { id: terminalTagId('deploy=blue'), label: 'deploy=blue', count: 2 },
      { id: terminalTagId('frontend'), label: 'frontend', count: 1 },
      { id: terminalTagId('owner=platform'), label: 'owner=platform', count: 1 },
      { id: terminalTagId('project=api'), label: 'project=api', count: 2 },
      { id: terminalTagId('project=web'), label: 'project=web', count: 1 },
    ])
    expect(publicTerminalTags(terminals[1]!)).toEqual([
      { id: 'frontend', label: 'frontend' },
      { id: 'project=web', label: 'project=web' },
    ])
  })
})

function terminal(terminalId: string, state: Terminal['state'], tags: Record<string, string>): Terminal {
  return { terminalId, machineId: 'studio', title: terminalId, state, tags }
}
