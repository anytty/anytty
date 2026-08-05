import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import zhCN from './locales/zh-CN.json'

export type AnyTTYLanguage = 'en' | 'zh-CN'

export const anyttyLanguages: { id: AnyTTYLanguage; label: string }[] = [
  { id: 'en', label: 'English' },
  { id: 'zh-CN', label: '简体中文' },
]

export function normalizeAnyTTYLanguage(value?: string | null): AnyTTYLanguage {
  return value?.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en'
}

function initialLanguage(): AnyTTYLanguage {
  if (typeof window === 'undefined') return 'en'
  try {
    return normalizeAnyTTYLanguage(window.localStorage?.getItem('anytty-language') ?? window.navigator.language)
  } catch {
    return normalizeAnyTTYLanguage(window.navigator.language)
  }
}

if (!i18n.isInitialized) {
  void i18n.use(initReactI18next).init({
    resources: { en: { translation: en }, 'zh-CN': { translation: zhCN } },
    lng: initialLanguage(),
    fallbackLng: 'en',
    interpolation: { escapeValue: false },
  })
}

if (typeof document !== 'undefined') {
  document.documentElement.lang = normalizeAnyTTYLanguage(i18n.resolvedLanguage)
  i18n.on('languageChanged', (language) => {
    const normalized = normalizeAnyTTYLanguage(language)
    try {
      window.localStorage?.setItem('anytty-language', normalized)
    } catch {
      // 某些 WebView/测试环境禁用 localStorage；语言仍在当前进程内生效。
    }
    document.documentElement.lang = normalized
  })
}

export function anyttyIntlLocale(): string {
  return normalizeAnyTTYLanguage(i18n.resolvedLanguage)
}

export { i18n as anyttyI18n }
