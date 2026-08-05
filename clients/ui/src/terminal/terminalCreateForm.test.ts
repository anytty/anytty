import { describe, expect, it } from 'vitest'
import { anyttyI18n } from '../i18n'
import { formatCommandLine, parseCommandLine, parseEnvironmentBlock } from './terminalCreateForm'

describe('terminal create form mapping', () => {
  it('round trips command arguments with spaces and quotes', () => {
    const command = ['/opt/My Shell/bin/zsh', '-lc', "printf '%s' ready"]
    expect(parseCommandLine(formatCommandLine(command))).toEqual(command)
  })

  it('parses pasted environment variables without losing equals signs', () => {
    expect(parseEnvironmentBlock('A=1\nexport TOKEN=a=b\n# ignored\n')).toEqual(['A=1', 'TOKEN=a=b'])
  })

  it('rejects invalid and duplicate environment keys', () => {
    expect(() => parseEnvironmentBlock('BAD-NAME=x')).toThrow(/Invalid/)
    expect(() => parseEnvironmentBlock('A=1\nA=2')).toThrow(/Duplicate/)
  })

  it('localizes user-facing parse failures', async () => {
    await anyttyI18n.changeLanguage('zh-CN')
    try {
      expect(() => parseCommandLine("echo 'unfinished")).toThrow('命令中有未结束的引号或转义符')
      expect(() => parseEnvironmentBlock('BAD-NAME=x')).toThrow('环境变量名称无效：BAD-NAME')
      expect(() => parseEnvironmentBlock('A=1\nA=2')).toThrow('环境变量重复：A')
    } finally {
      await anyttyI18n.changeLanguage('en')
    }
  })
})
