import { describe, expect, it, vi } from 'vitest'
import { NativeSessionDemandCoordinator, type NativeSessionDemandInput } from './NativeSessionDemand'

describe('NativeSessionDemandCoordinator', () => {
  it('replaces native ownership with the complete sorted renderer demand', async () => {
    const replaceDemand = vi.fn(async (_input: NativeSessionDemandInput) => {})
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    await demand.setActive('machine-b', true)
    await demand.setActive('machine-a', true)
    await demand.setActive('machine-b', false)

    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-b'],
      ['machine-a', 'machine-b'],
      ['machine-a'],
    ])
  })

  it('can reconcile an empty renderer after the previous WebView disappeared', async () => {
    const replaceDemand = vi.fn(async (_input: NativeSessionDemandInput) => {})
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    await demand.reconcileRenderer()

    expect(replaceDemand).toHaveBeenCalledWith({ endpointIds: [] })
  })

  it('adopts the native per-endpoint takeover projection', async () => {
    const replaceDemand = vi.fn(async ({ endpointIds }: NativeSessionDemandInput) => ({
      goManagedEndpointIds: endpointIds.filter((endpointId) => endpointId === 'machine-a'),
    }))
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    await demand.setActive('machine-b', true)
    await demand.setActive('machine-a', true)

    expect(demand.isGoManaged('machine-a')).toBe(true)
    expect(demand.isGoManaged('machine-b')).toBe(false)
  })

  it('clears the takeover projection synchronously on native user stop', async () => {
    const demand = new NativeSessionDemandCoordinator(async ({ endpointIds }) => ({
      goManagedEndpointIds: endpointIds,
    }))
    await demand.setActive('machine-a', true)
    expect(demand.isGoManaged('machine-a')).toBe(true)

    const stopped = demand.clearForUserStop()
    expect(demand.isGoManaged('machine-a')).toBe(false)
    await stopped
  })

  it('rolls back the latest local intent when native rejects its replacement', async () => {
    const replaceDemand = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('stale revision'))
      .mockResolvedValueOnce(undefined)
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    await demand.setActive('machine-a', true)
    await expect(demand.setActive('machine-a', false)).rejects.toThrow('stale revision')
    await demand.reconcileRenderer()

    expect(replaceDemand).toHaveBeenLastCalledWith({ endpointIds: ['machine-a'] })
  })

  it('does not let an older failed update roll back a newer full snapshot', async () => {
    let rejectFirst!: (failure: Error) => void
    const first = new Promise<void>((_resolve, reject) => { rejectFirst = reject })
    const replaceDemand = vi.fn()
      .mockReturnValueOnce(first)
      .mockResolvedValueOnce(undefined)
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    const firstUpdate = demand.setActive('machine-a', true)
    const secondUpdate = demand.setActive('machine-b', true)
    rejectFirst(new Error('old attachment'))

    await expect(firstUpdate).rejects.toThrow('old attachment')
    await secondUpdate
    expect(replaceDemand).toHaveBeenLastCalledWith({ endpointIds: ['machine-a', 'machine-b'] })
  })

  it('fences updates that were queued before a native user stop', async () => {
    let finishFirst!: () => void
    const first = new Promise<void>((resolve) => { finishFirst = resolve })
    const replaceDemand = vi.fn()
      .mockReturnValueOnce(first)
      .mockResolvedValue(undefined)
    const demand = new NativeSessionDemandCoordinator(replaceDemand)

    const activeA = demand.setActive('machine-a', true)
    await Promise.resolve()
    const staleActiveB = demand.setActive('machine-b', true)
    const stopped = demand.clearForUserStop()
    finishFirst()

    await activeA
    await staleActiveB
    await stopped
    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-a'],
      [],
    ])
  })
})
