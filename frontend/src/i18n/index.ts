import { enUS } from './en-US';
import { zhCN } from './zh-CN';

const dictionaries = {
  'en-US': enUS,
  'zh-CN': zhCN
} as const;

export type Locale = keyof typeof dictionaries;
export type LanguageSetting = 'auto' | Locale;
export type TranslationKey = keyof typeof enUS;

export function resolveLocale(effectiveLanguage?: string): Locale {
  if (effectiveLanguage === 'zh-CN' || effectiveLanguage === 'en-US') {
    return effectiveLanguage;
  }
  return 'en-US';
}

export function t(locale: Locale, key: TranslationKey): string {
  return dictionaries[locale][key];
}
