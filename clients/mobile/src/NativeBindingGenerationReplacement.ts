type CloseableBindingClient = {
  close(): Promise<void>
}

type NativeGenerationResetEntry = {
  reset(signal?: AbortSignal): Promise<void>
  release?(signal?: AbortSignal): Promise<void>
}

/**
 * Serializes ownership transfer between renderer binding generations.
 *
 * The candidate becomes provisional current before managers are fenced so any concurrent
 * reconnect uses the new generation. Once published, ownership only moves forward: a failed or
 * cancelled reset never rolls back to a predecessor whose bridge may already be closed. The next
 * serialized round replaces the provisional candidate and closes it after that round succeeds.
 * Reset and refresh receive the attempt signal and must check it before publishing post-await state.
 */
export class NativeBindingGenerationReplacement<Client extends CloseableBindingClient> {
  private tail: Promise<void> = Promise.resolve()

  constructor(
    private readonly current: () => Client,
    private readonly commit: (client: Client) => void,
    private readonly create: () => Client,
  ) {}

  replace(
    resetRuntime: (signal?: AbortSignal) => Promise<void>,
    refresh: (client: Client, signal?: AbortSignal) => Promise<void>,
    signal?: AbortSignal,
  ): Promise<void> {
    const replacement = this.tail.catch(() => undefined).then(async () => {
      throwIfNativeGenerationAborted(signal)
      const staleClient = this.current()
      const currentClient = this.create()
      if (signal?.aborted) {
        await settleNativeGenerationCleanup(() => currentClient.close(), signal)
        throw nativeGenerationAbortFailure(signal)
      }
      const commitResult = settleNativeGenerationMutation(() => this.commit(currentClient))
      if (commitResult.status === 'rejected') {
        const closeResult = await settleNativeGenerationCleanup(() => currentClient.close(), signal)
        if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
        throwNativeGenerationFailures('Native binding generation replacement failed', [commitResult, closeResult])
      }

      const resetResult = await settleNativeGenerationStep(() => resetRuntime(signal), signal)
      if (resetResult.status === 'rejected' || signal?.aborted) {
        const closeResult = await settleNativeGenerationCleanup(() => staleClient.close(), signal)
        if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
        throwNativeGenerationFailures(
          'Native binding generation replacement failed',
          [resetResult, closeResult],
        )
      }

      const refreshResult = await settleNativeGenerationStep(() => refresh(currentClient, signal), signal)
      if (refreshResult.status === 'rejected' || signal?.aborted) {
        const closeResult = await settleNativeGenerationCleanup(() => staleClient.close(), signal)
        if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
        throwNativeGenerationFailures(
          'Native binding generation replacement failed',
          [refreshResult, closeResult],
        )
      }

      const closeResult = await settleNativeGenerationCleanup(() => staleClient.close(), signal)
      if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
      throwNativeGenerationFailures('Native binding generation replacement failed', [closeResult])
    })
    this.tail = replacement
    return replacement
  }
}

function throwIfNativeGenerationAborted(signal?: AbortSignal): void {
  if (!signal?.aborted) return
  throw nativeGenerationAbortFailure(signal)
}

function nativeGenerationAbortResult(signal: AbortSignal): PromiseRejectedResult {
  return { status: 'rejected', reason: nativeGenerationAbortFailure(signal) }
}

function nativeGenerationAbortFailure(signal: AbortSignal): unknown {
  return signal.reason ?? new DOMException('Native binding generation replacement was cancelled', 'AbortError')
}

export async function drainNativeGenerationReset(
  entries: readonly NativeGenerationResetEntry[],
  signal?: AbortSignal,
): Promise<void> {
  throwIfNativeGenerationAborted(signal)
  const results = await Promise.allSettled(entries.map(async (entry) => {
    const resetResult = await settleNativeGenerationStep(() => entry.reset(signal), signal)
    const releaseResult = entry.release
      ? await settleNativeGenerationCleanup(() => entry.release!(signal), signal)
      : { status: 'fulfilled', value: undefined } satisfies PromiseFulfilledResult<void>
    if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
    throwNativeGenerationFailures('Native manager generation reset failed', [resetResult, releaseResult])
  }))
  if (signal?.aborted) throw nativeGenerationAbortFailure(signal)
  throwNativeGenerationFailures('Native manager generation reset failed', results)
}

async function settleNativeGenerationStep(
  operation: () => Promise<void>,
  signal?: AbortSignal,
): Promise<PromiseSettledResult<void>> {
  if (signal?.aborted) return nativeGenerationAbortResult(signal)
  return await settleNativeGenerationPromise(startNativeGenerationOperation(operation), signal)
}

async function settleNativeGenerationCleanup(
  operation: () => Promise<void>,
  signal?: AbortSignal,
): Promise<PromiseSettledResult<void>> {
  return await settleNativeGenerationPromise(startNativeGenerationOperation(operation), signal)
}

function startNativeGenerationOperation(operation: () => Promise<void>): Promise<void> {
  try {
    return Promise.resolve(operation())
  } catch (reason) {
    return Promise.reject(reason)
  }
}

async function settleNativeGenerationPromise(
  operation: Promise<void>,
  signal?: AbortSignal,
): Promise<PromiseSettledResult<void>> {
  if (!signal) {
    try {
      await operation
      return { status: 'fulfilled', value: undefined }
    } catch (reason) {
      return { status: 'rejected', reason }
    }
  }
  return await new Promise<PromiseSettledResult<void>>((resolve) => {
    let settled = false
    const finish = (result: PromiseSettledResult<void>) => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      resolve(result)
    }
    const abort = () => finish(nativeGenerationAbortResult(signal))
    signal.addEventListener('abort', abort, { once: true })
    void operation.then(
      () => finish({ status: 'fulfilled', value: undefined }),
      (reason: unknown) => finish({ status: 'rejected', reason }),
    )
    if (signal.aborted) abort()
  })
}

function settleNativeGenerationMutation(operation: () => void): PromiseSettledResult<void> {
  try {
    operation()
    return { status: 'fulfilled', value: undefined }
  } catch (reason) {
    return { status: 'rejected', reason }
  }
}

function throwNativeGenerationFailures(label: string, results: readonly PromiseSettledResult<void>[]): void {
  const failures = results.flatMap((result) => result.status === 'rejected' ? [result.reason] : [])
  if (failures.length === 0) return
  if (failures.length === 1) throw failures[0]
  throw new AggregateError(failures, label)
}
