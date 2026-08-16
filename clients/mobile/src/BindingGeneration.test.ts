import { describe, expect, it } from 'vitest'
import { settleBindingGeneration } from './BindingGeneration'

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

describe('settleBindingGeneration', () => {
  it('returns a result from the current generation', async () => {
    const client = {}

    await expect(settleBindingGeneration(client, () => client, async () => 'current registry')).resolves.toEqual({
      current: true,
      value: 'current registry',
    })
  })

  it('drops a result from a stale generation', async () => {
    const oldClient = {}
    const newClient = {}
    let currentClient = oldClient
    const pending = deferred<string>()
    const resultPromise = settleBindingGeneration(oldClient, () => currentClient, () => pending.promise)

    currentClient = newClient
    pending.resolve('stale registry')

    await expect(resultPromise).resolves.toEqual({ current: false })
  })

  it('drops a closed-backend error from a stale generation', async () => {
    const oldClient = {}
    const newClient = {}
    let currentClient = oldClient
    const pending = deferred<string>()
    const resultPromise = settleBindingGeneration(oldClient, () => currentClient, () => pending.promise)

    currentClient = newClient
    pending.reject(new Error('Go binding backend is closed'))

    await expect(resultPromise).resolves.toEqual({ current: false })
  })

  it('propagates an error from the current generation', async () => {
    const client = {}

    await expect(settleBindingGeneration(
      client,
      () => client,
      async () => { throw new Error('registry unavailable') },
    )).rejects.toThrow('registry unavailable')
  })
})
