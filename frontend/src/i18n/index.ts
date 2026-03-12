import { enUS } from './en-US';
import { zhCN } from './zh-CN';

const dictionaries = {
  'en-US': enUS,
  'zh-CN': zhCN
} as const;

export type Locale = keyof typeof dictionaries;
export type LanguageSetting = 'auto' | Locale;
export type TranslationKey = keyof typeof enUS;

export function resolveLocale(setting: LanguageSetting): Locale {
  if (setting === 'zh-CN' || setting === 'en-US') {
    return setting;
  }
  const browserLang = typeof navigator !== 'undefined' ? navigator.language.toLowerCase() : '';
  return browserLang.startsWith('zh') ? 'zh-CN' : 'en-US';
}

export function t(locale: Locale, key: TranslationKey): string {
  return dictionaries[locale][key];
}

export function localizeReason(reason: string, locale: Locale): string {
  if (reason === 'eye') {
    return t(locale, 'reasonEye');
  }
  if (reason === 'stand') {
    return t(locale, 'reasonStand');
  }
  return reason;
}

