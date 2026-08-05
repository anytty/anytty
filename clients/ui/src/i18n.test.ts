import { afterEach, describe, expect, it } from 'vitest'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'
import { anyttyI18n, normalizeAnyTTYLanguage } from './i18n'

function localeKeys(value: unknown, prefix = ''): string[] {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return [prefix]
  return Object.entries(value).flatMap(([key, child]) => localeKeys(child, prefix ? `${prefix}.${key}` : key))
}

describe('AnyTTY UI i18n', () => {
  afterEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('normalizes system locales to the two supported languages', () => {
    expect(normalizeAnyTTYLanguage('zh-CN')).toBe('zh-CN')
    expect(normalizeAnyTTYLanguage('zh-TW')).toBe('zh-CN')
    expect(normalizeAnyTTYLanguage('en-US')).toBe('en')
    expect(normalizeAnyTTYLanguage('ru-RU')).toBe('en')
  })

  it('switches the primary device and pairing vocabulary without changing domain state', async () => {
    await anyttyI18n.changeLanguage('zh-CN')
    expect(anyttyI18n.t('machines.title')).toBe('设备')
    expect(anyttyI18n.t('machines.scanService')).toBe('添加设备')

    await anyttyI18n.changeLanguage('en')
    expect(anyttyI18n.t('machines.title')).toBe('Devices')
    expect(anyttyI18n.t('machines.scanService')).toBe('Add device')
  })

  it('keeps English and Simplified Chinese locale keys symmetric', () => {
    expect(localeKeys(zhCN).sort()).toEqual(localeKeys(en).sort())
  })
})
