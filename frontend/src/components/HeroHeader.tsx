import type { ReactNode, Ref } from 'react';
import type { Locale } from '../i18n';
import { t } from '../i18n';

type HeroHeaderProps = {
  locale: Locale;
  titleRef?: Ref<HTMLHeadingElement>;
  actions?: ReactNode;
};

export function HeroHeader({ locale, titleRef, actions }: HeroHeaderProps) {
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
        <p className="m-0 text-[var(--text-secondary)]">{t(locale, 'appSubtitle')}</p>
      </div>
      {actions ? <div className="flex shrink-0 items-center justify-center">{actions}</div> : null}
    </header>
  );
}
