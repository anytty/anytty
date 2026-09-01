/**
 * NativeForegroundBarrier 是 Android WebView 前后台 generation 交接的 UI 侧栅栏。
 *
 * Native plugin 负责关闭旧 engine 并创建新 bridge；这里不拥有 generation 真值，只阻止
 * session、resource 和 SAF 等业务结果在新 bridge 和 runtime reset 完成前进入 binding 调用。
 */
export class NativeForegroundBarrier {
  private ready: Promise<Error | undefined> = Promise.resolve(undefined)
  private resolveReady: ((failure?: Error) => void) | null = null
  private generation = 0

  /** markBackground 在可能冻结 WebView 的平台动作开始前建立栅栏；重复调用保持同一轮等待。 */
  markBackground(): number {
    this.generation += 1
    if (!this.resolveReady) {
      this.ready = new Promise<Error | undefined>((resolve) => { this.resolveReady = resolve })
    }
    return this.generation
  }

  /** cancelBackground only releases the barrier if no later lifecycle event reinforced it. */
  cancelBackground(generation: number): void {
    if (generation === this.generation) this.finishForeground()
  }

  /** finishForeground 只由 native generation 替换完成链路解除栅栏，并保留恢复失败原因。 */
  finishForeground(failure?: unknown): void {
    const resolve = this.resolveReady
    this.resolveReady = null
    resolve?.(failure instanceof Error ? failure : failure ? new Error(String(failure)) : undefined)
  }

  /** wait 等待当前 foreground generation 可供新的 binding operation 使用。 */
  async wait(signal?: AbortSignal): Promise<void> {
    const failure = await waitForForegroundResult(this.ready, signal)
    if (failure) throw failure
  }
}

async function waitForForegroundResult(
  ready: Promise<Error | undefined>,
  signal?: AbortSignal,
): Promise<Error | undefined> {
  if (!signal) return await ready
  if (signal.aborted) throw abortError(signal)
  return await new Promise<Error | undefined>((resolve, reject) => {
    let settled = false
    const abort = () => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      reject(abortError(signal))
    }
    signal.addEventListener('abort', abort, { once: true })
    void ready.then((failure) => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      resolve(failure)
    })
  })
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}

/**
 * runAcrossNativePicker 在启动 SAF 前建立 generation fence。
 * picker promise 可能早于 Capacitor appStateChange 回调完成，结果必须继续等待 foreground barrier。
 */
export async function runAcrossNativePicker<T>(
  barrier: NativeForegroundBarrier,
  pick: () => Promise<T>,
  validateContinuation?: (result: T) => void | Promise<void>,
): Promise<T> {
  const generation = barrier.markBackground()
  let result: T
  try {
    result = await pick()
  } catch (failure) {
    barrier.cancelBackground(generation)
    throw failure
  }
  await barrier.wait()
  await validateContinuation?.(result)
  return result
}
