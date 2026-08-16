/**
 * settleBindingGeneration 只允许当前 binding generation 提交异步结果。
 *
 * Android 后台恢复或网络切换会关闭旧 Go engine 并创建新 generation。旧请求即使随后成功或失败，
 * 也不能覆盖新 generation 的 UI projection；只有仍属于 current owner 的失败才向调用方传播。
 */
export async function settleBindingGeneration<Client, Value>(
  candidate: Client,
  current: () => Client,
  operation: () => Promise<Value>,
): Promise<{ current: true; value: Value } | { current: false }> {
  try {
    const value = await operation()
    if (candidate !== current()) return { current: false }
    return { current: true, value }
  } catch (error) {
    if (candidate !== current()) return { current: false }
    throw error
  }
}
