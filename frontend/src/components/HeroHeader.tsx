import type { ReactNode, Ref } from 'react';
import type { Locale } from '../i18n';
import { t } from '../i18n';

type HeroHeaderProps = {
  locale: Locale;
  titleRef?: Ref<HTMLHeadingElement>;
  actions?: ReactNode;
};

export function HeroHeader({ locale, titleRef, actions }: HeroHeaderProps) {
  const subtitleClassName =
    locale === 'en-US'
      ? 'm-0 text-[9px] leading-none tracking-[0.14em] uppercase text-[var(--text-tertiary)]'
      : 'm-0 text-[9px] leading-none tracking-[0.06em] text-[var(--text-tertiary)]';

  return (
    <header className="flex items-start justify-between gap-3 max-sm:flex-col max-sm:items-start">
      <div className="flex min-w-0 flex-col gap-1.5">
        <h1
          ref={titleRef}
          tabIndex={-1}
          className="m-0 text-[36px] font-semibold leading-[0.98] tracking-[-0.02em] outline-none max-sm:text-[28px]"
        >
          {t(locale, 'appTitle')}
        </h1>
        <div className="flex items-center gap-2">
          <span className="h-px w-5 bg-[var(--surface-border-strong)]" aria-hidden="true" />
          <p className={subtitleClassName}>{t(locale, 'appSubtitle')}</p>
        </div>
      </div>
      {actions ? <div className="flex shrink-0 items-center justify-center self-center max-sm:self-start">{actions}</div> : null}
    </header>
  );
}
