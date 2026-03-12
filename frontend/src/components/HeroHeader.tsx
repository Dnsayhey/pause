import type { RefObject } from 'react';
import type { Settings } from '../types';
import type { Locale } from '../i18n';
import { t } from '../i18n';

type HeroHeaderProps = {
  locale: Locale;
  language: Settings['ui']['language'];
  titleRef?: RefObject<HTMLHeadingElement | null>;
  onLanguageChange: (language: Settings['ui']['language']) => void;
};

export function HeroHeader({ locale, language, titleRef, onLanguageChange }: HeroHeaderProps) {
  return (
    <header className="hero">
      <div className="hero-main">
        <h1 ref={titleRef} tabIndex={-1}>{t(locale, 'appTitle')}</h1>
        <p>{t(locale, 'appSubtitle')}</p>
      </div>
      <div className="hero-lang">
        <select
          className="lang-select"
          value={language}
          onChange={(e) => {
            onLanguageChange(e.target.value as Settings['ui']['language']);
            e.currentTarget.blur();
          }}
        >
          <option value="auto">{t(locale, 'languageAuto')}</option>
          <option value="zh-CN">{t(locale, 'languageZhCN')}</option>
          <option value="en-US">{t(locale, 'languageEnUS')}</option>
        </select>
      </div>
    </header>
  );
}
