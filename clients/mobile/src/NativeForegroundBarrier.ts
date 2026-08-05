/**
 * NativeForegroundBarrier 是 Android WebView 前后台 generation 交接的 UI 侧栅栏。
 *
 * Native plugin 负责关闭旧 engine 并创建新 bridge；这里不拥有 generation 真值，只阻止
 * SAF 等外部 Activity 返回的结果在新 bridge 和 runtime reset 完成前进入业务调用。
 */
export class NativeForegroundBarrier {
  private ready: Promise<Error | undefined> = Promise.resolve(undefined)
  private resolveReady: ((failure?: Error) => void) | null = null

  /** markBackground 在可能冻结 WebView 的平台动作开始前建立栅栏；重复调用保持同一轮等待。 */
  markBackground(): void {
    if (this.resolveReady) return
    this.ready = new Promise<Error | undefined>((resolve) => { this.resolveReady = resolve })
  }

  /** finishForeground 只由 native generation 替换完成链路解除栅栏，并保留恢复失败原因。 */
  finishForeground(failure?: unknown): void {
    const resolve = this.resolveReady
    this.resolveReady = null
    resolve?.(failure instanceof Error ? failure : failure ? new Error(String(failure)) : undefined)
  }

  /** wait 等待当前 foreground generation 可供新的 binding operation 使用。 */
  async wait(): Promise<void> {
    const failure = await this.ready
    if (failure) throw failure
  }
}

/**
 * runAcrossNativePicker 在启动 SAF 前建立 generation fence。
 * picker promise 可能早于 Capacitor appStateChange 回调完成，结果必须继续等待 foreground barrier。
 */
export async function runAcrossNativePicker<T>(
  barrier: NativeForegroundBarrier,
  pick: () => Promise<T>,
): Promise<T> {
  barrier.markBackground()
  const result = await pick()
  await barrier.wait()
  return result
}
