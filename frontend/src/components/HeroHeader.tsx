import type { Ref } from 'react';
import type { Settings } from '../types';
import type { Locale } from '../i18n';
import { t } from '../i18n';

type HeroHeaderProps = {
  locale: Locale;
  language: Settings['ui']['language'];
  titleRef?: Ref<HTMLHeadingElement>;
  onLanguageChange: (language: Settings['ui']['language']) => void;
};

export function HeroHeader({ locale, language, titleRef, onLanguageChange }: HeroHeaderProps) {
  const languageSelectClassName =
    'h-[var(--control-height)] w-[108px] rounded-full border border-[var(--glass-control-border)] bg-[var(--glass-control-bg)] px-[10px] text-[13px] leading-[1.2] shadow-[var(--glass-control-shadow)] max-sm:w-[100px] [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]';

  return (
    <header className="flex items-center justify-between gap-3">
      <div className="flex items-baseline gap-3 max-sm:gap-2">
        <h1
          ref={titleRef}
          tabIndex={-1}
          className="m-0 text-[44px] leading-none tracking-[-0.02em] outline-none max-sm:text-[32px]"
        >
          {t(locale, 'appTitle')}
        </h1>
        <p className="m-0 text-[rgba(18,34,54,0.7)]">{t(locale, 'appSubtitle')}</p>
      </div>
      <div className="flex shrink-0 items-center justify-center">
        <select
          className={languageSelectClassName}
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
