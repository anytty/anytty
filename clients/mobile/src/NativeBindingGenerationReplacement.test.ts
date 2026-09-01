import { describe, expect, it, vi } from 'vitest'
import { NativeBindingGenerationReplacement, drainNativeGenerationReset } from './NativeBindingGenerationReplacement'

describe('native binding generation replacement', () => {
  it('drains a failed reset before rejecting and still releases every entry', async () => {
    const delayedReset = deferred<void>()
    const firstRelease = vi.fn(async () => undefined)
    const secondRelease = vi.fn(async () => undefined)
    let settled = false

    const reset = drainNativeGenerationReset([
      {
        reset: async () => { throw new Error('manager reset failed') },
        release: firstRelease,
      },
      {
        reset: () => delayedReset.promise,
        release: secondRelease,
      },
    ])
    const outcome = reset.then(
      () => null,
      (failure: unknown) => failure,
    ).finally(() => { settled = true })

    await vi.waitFor(() => expect(firstRelease).toHaveBeenCalledOnce())
    expect(secondRelease).not.toHaveBeenCalled()
    expect(settled).toBe(false)

    delayedReset.resolve()
    expect(await outcome).toEqual(new Error('manager reset failed'))
    expect(secondRelease).toHaveBeenCalledOnce()
  })

  it('keeps ownership monotonic and serializes an immediate retry behind a failed reset', async () => {
    type FakeClient = {
      id: string
      closed: boolean
      close: ReturnType<typeof vi.fn<() => Promise<void>>>
    }
    const clients: FakeClient[] = []
    const makeClient = (id: string): FakeClient => {
      const client: FakeClient = {
        id,
        closed: false,
        close: vi.fn(async () => { client.closed = true }),
      }
      clients.push(client)
      return client
    }
    let current = makeClient('initial')
    const commits: string[] = []
    const createClient = vi.fn(() => makeClient(`candidate-${clients.length}`))
    const replacement = new NativeBindingGenerationReplacement(
      () => current,
      (client) => {
        current = client
        commits.push(client.id)
      },
      createClient,
    )
    const delayedReset = deferred<void>()
    let resetRound = 0
    const resetRuntime = vi.fn(async () => {
      resetRound += 1
      if (resetRound !== 1) return
      await drainNativeGenerationReset([
        { reset: async () => { throw new Error('first reset failed') } },
        { reset: () => delayedReset.promise },
      ])
    })
    const refreshed: string[] = []
    const refresh = vi.fn(async (client: FakeClient) => { refreshed.push(client.id) })

    const first = replacement.replace(resetRuntime, refresh)
    const firstOutcome = first.then(
      () => null,
      (failure: unknown) => failure,
    )
    const retry = replacement.replace(resetRuntime, refresh)

    await vi.waitFor(() => expect(current.id).toBe('candidate-1'))
    expect(createClient).toHaveBeenCalledOnce()
    expect(clients[0]?.close).not.toHaveBeenCalled()
    expect(refreshed).toEqual([])

    delayedReset.resolve()

    expect(await firstOutcome).toEqual(new Error('first reset failed'))
    expect(current.id).toBe('candidate-1')
    await expect(retry).resolves.toBeUndefined()
    expect(createClient).toHaveBeenCalledTimes(2)
    expect(commits).toEqual(['candidate-1', 'candidate-2'])
    expect(refreshed).toEqual(['candidate-2'])
    expect(clients[0]?.close).toHaveBeenCalledOnce()
    expect(clients[1]?.close).toHaveBeenCalledOnce()
    expect(clients[2]?.close).not.toHaveBeenCalled()
    expect(clients.filter((client) => !client.closed)).toEqual([current])
    expect(current.id).toBe('candidate-2')
  })

  it('never rolls a timed-out replacement back to an already closed stale client', async () => {
    type FakeClient = {
      id: string
      closed: boolean
      close: ReturnType<typeof vi.fn<() => Promise<void>>>
    }
    const clients: FakeClient[] = []
    const makeClient = (id: string, closed = false): FakeClient => {
      const client: FakeClient = {
        id,
        closed,
        close: vi.fn(async () => { client.closed = true }),
      }
      clients.push(client)
      return client
    }
    let current = makeClient('closed-initial', true)
    const commits: string[] = []
    const replacement = new NativeBindingGenerationReplacement(
      () => current,
      (client) => {
        current = client
        commits.push(client.id)
      },
      () => makeClient(`candidate-${clients.length}`),
    )
    const reset = deferred<void>()
    const controller = new AbortController()
    const timedOut = replacement.replace(
      () => reset.promise,
      async () => undefined,
      controller.signal,
    )
    await vi.waitFor(() => expect(current.id).toBe('candidate-1'))

    const timeout = Object.assign(new Error('binding replacement timed out'), {
      code: 'unavailable',
      retryable: true,
    })
    controller.abort(timeout)

    await expect(timedOut).rejects.toBe(timeout)
    expect(current.id).toBe('candidate-1')
    expect(current).not.toBe(clients[0])
    expect(clients[0]?.close).toHaveBeenCalledOnce()
    expect(clients[1]?.close).not.toHaveBeenCalled()

    await expect(replacement.replace(async () => undefined, async () => undefined)).resolves.toBeUndefined()
    reset.resolve()
    await Promise.resolve()
    expect(commits).toEqual(['candidate-1', 'candidate-2'])
    expect(current.id).toBe('candidate-2')
    expect(clients[1]?.close).toHaveBeenCalledOnce()
    expect(clients[2]?.close).not.toHaveBeenCalled()
  })

  it('releases serialization when an aborted reset and stale close never settle', async () => {
    type FakeClient = {
      id: string
      close: ReturnType<typeof vi.fn<() => Promise<void>>>
    }
    const never = new Promise<void>(() => {})
    const clients: FakeClient[] = []
    const makeClient = (id: string, close: () => Promise<void> = async () => undefined): FakeClient => {
      const client = { id, close: vi.fn(close) }
      clients.push(client)
      return client
    }
    let current = makeClient('initial', () => never)
    const commits: string[] = []
    const replacement = new NativeBindingGenerationReplacement(
      () => current,
      (client) => {
        current = client
        commits.push(client.id)
      },
      () => makeClient(`candidate-${clients.length}`),
    )
    const firstController = new AbortController()
    const first = replacement.replace(
      () => never,
      async () => { throw new Error('hung reset must never reach refresh') },
      firstController.signal,
    )
    await vi.waitFor(() => expect(current.id).toBe('candidate-1'))

    const timeout = Object.assign(new Error('binding replacement timed out'), {
      code: 'unavailable',
      retryable: true,
    })
    firstController.abort(timeout)

    await expect(first).rejects.toBe(timeout)
    await expect(replacement.replace(async () => undefined, async () => undefined)).resolves.toBeUndefined()
    expect(commits).toEqual(['candidate-1', 'candidate-2'])
    expect(current.id).toBe('candidate-2')
    expect(clients[0]?.close).toHaveBeenCalledOnce()
    expect(clients[1]?.close).toHaveBeenCalledOnce()
    expect(clients[2]?.close).not.toHaveBeenCalled()
  })
})

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((accept, decline) => {
    resolve = accept
    reject = decline
  })
  return { promise, resolve, reject }
}
