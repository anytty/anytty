import { anyttyI18n } from '../i18n'

/** EnvironmentVariable 是 TerminalCreateSpec.env 单项在表单内的可编辑投影，不是持久领域模型。 */
export interface EnvironmentVariable {
  key: string
  value: string
}

/** formatCommandLine 把 generated Proto command 参数无损投影到单行输入框。 */
export function formatCommandLine(arguments_: readonly string[]): string {
  return arguments_.map((argument) => {
    if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(argument)) return argument
    return `'${argument.replaceAll("'", "'\\''")}'`
  }).join(' ')
}

/** parseCommandLine 把用户输入解析回参数数组；未闭合引用会显式失败而不会改变命令语义。 */
export function parseCommandLine(input: string): string[] {
  const arguments_: string[] = []
  let value = ''
  let quote: "'" | '"' | null = null
  let escaped = false
  let active = false
  for (const character of input.trim()) {
    if (escaped) {
      value += character
      escaped = false
      active = true
      continue
    }
    if (character === '\\' && quote !== "'") {
      escaped = true
      active = true
      continue
    }
    if (quote) {
      if (character === quote) quote = null
      else value += character
      active = true
      continue
    }
    if (character === "'" || character === '"') {
      quote = character
      active = true
      continue
    }
    if (/\s/.test(character)) {
      if (active) {
        arguments_.push(value)
        value = ''
        active = false
      }
      continue
    }
    value += character
    active = true
  }
  if (escaped || quote) throw new Error(anyttyI18n.t('workspace.terminalForm.unfinishedCommand'))
  if (active) arguments_.push(value)
  return arguments_
}

/** parseEnvironmentEntry 按第一个等号拆分 Proto env 单项，值中的等号保持不变。 */
export function parseEnvironmentEntry(entry: string): EnvironmentVariable {
  const separator = entry.indexOf('=')
  return separator < 0
    ? { key: entry.trim(), value: '' }
    : { key: entry.slice(0, separator).trim(), value: entry.slice(separator + 1) }
}

/** environmentEntry 把 KV 编辑态投影回 TerminalCreateSpec.env 使用的 KEY=VALUE。 */
export function environmentEntry(variable: EnvironmentVariable): string {
  return `${variable.key.trim()}=${variable.value}`
}

/** validateEnvironmentVariables 在跨 API 边界前校验变量名和重复键。 */
export function validateEnvironmentVariables(entries: readonly string[]): string[] {
  const result: string[] = []
  const keys = new Set<string>()
  for (const entry of entries) {
    const variable = parseEnvironmentEntry(entry)
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(variable.key)) {
      throw new Error(anyttyI18n.t('workspace.terminalForm.invalidEnvironmentName', {
        name: variable.key || anyttyI18n.t('workspace.terminalForm.emptyEnvironmentName'),
      }))
    }
    if (keys.has(variable.key)) {
      throw new Error(anyttyI18n.t('workspace.terminalForm.duplicateEnvironment', { name: variable.key }))
    }
    keys.add(variable.key)
    result.push(environmentEntry(variable))
  }
  return result
}

/** parseEnvironmentBlock 解析多行粘贴内容，并接受常见的 export KEY=VALUE 形式。 */
export function parseEnvironmentBlock(input: string): string[] {
  const entries = input.split(/\r?\n/).flatMap((rawLine) => {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) return []
    return [line.startsWith('export ') ? line.slice('export '.length).trim() : line]
  })
  return validateEnvironmentVariables(entries)
}
