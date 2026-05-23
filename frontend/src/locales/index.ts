import { createI18n } from 'vue-i18n'
import en from './en'
import zh from './zh'

type SupportedLocale = 'zh' | 'en'
type LocaleMessages = Record<string, unknown>

const messageLookup: Record<SupportedLocale, LocaleMessages> = { en, zh }

function normalizeLocale(value: string | null | undefined): SupportedLocale {
  const raw = String(value || '').trim().toLowerCase()
  if (raw.startsWith('en')) return 'en'
  if (raw.startsWith('zh')) return 'zh'
  return 'zh'
}

function initialLocale(): SupportedLocale {
  const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('locale') : ''
  const browser = typeof navigator !== 'undefined' ? navigator.language : ''
  const locale = normalizeLocale(stored || browser)
  if (typeof localStorage !== 'undefined' && stored !== locale) {
    localStorage.setItem('locale', locale)
  }
  return locale
}

function humanizeMissingKey(key: string) {
  const last = key.split('.').filter(Boolean).pop() || key
  return last
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/[-_]+/g, ' ')
    .replace(/\b\w/g, (char) => char.toUpperCase())
}

function lookupMessage(locale: SupportedLocale, key: string): string | undefined {
  let current: unknown = messageLookup[locale]
  for (const part of key.split('.').filter(Boolean)) {
    if (!current || typeof current !== 'object' || !(part in current)) {
      return undefined
    }
    current = (current as Record<string, unknown>)[part]
  }
  return typeof current === 'string' ? current : undefined
}

function resolveMissingMessage(locale: string, key: string) {
  const primary = normalizeLocale(locale)
  return lookupMessage(primary, key)
    || lookupMessage(primary === 'zh' ? 'en' : 'zh', key)
    || lookupMessage('zh', key)
    || lookupMessage('en', key)
    || humanizeMissingKey(key)
}

const i18n = createI18n({
  legacy: false,
  locale: initialLocale(),
  fallbackLocale: {
    zh: ['en'],
    en: ['zh'],
    default: ['zh', 'en'],
  },
  missingWarn: false,
  fallbackWarn: false,
  messages: { en, zh },
  missing: (locale, key) => resolveMissingMessage(locale, key),
})

export default i18n
