import { localizeReason, t, type Locale } from '../i18n';
import type { RuntimeState } from '../types';
import { PrimaryButton } from './ui';

function formatSec(totalSec: number, offLabel: string): string {
  if (totalSec < 0) return offLabel;
  const m = Math.floor(totalSec / 60);
  const s = totalSec % 60;
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
}

type BreakOverlayProps = {
  locale: Locale;
  runtime: RuntimeState;
  onSkip: () => void;
};

export function BreakOverlay({ locale, runtime, onSkip }: BreakOverlayProps) {
  if (!runtime.currentSession) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-[9999] grid place-items-center bg-[rgba(11,23,44,0.85)]">
      <div className="w-[min(440px,calc(100%-40px))] rounded-[18px] bg-[#f8fff7] p-6 text-center">
        <h2 className="m-0 text-[28px] leading-tight">{t(locale, 'breakTime')}</h2>
        <p className="mb-0 mt-3 text-[15px] text-[#243649]">
          {t(locale, 'reason')}: {runtime.currentSession.reasons.map((reason) => localizeReason(reason, locale)).join(' + ')}
        </p>
        <p className="mb-0 mt-2 text-[15px] text-[#243649]">
          {t(locale, 'remaining')}: {formatSec(runtime.currentSession.remainingSec, t(locale, 'off'))}
        </p>
        {runtime.overlaySkipAllowed && runtime.currentSession.canSkip && (
          <PrimaryButton className="mt-4" onClick={onSkip}>
            {t(locale, 'emergencySkip')}
          </PrimaryButton>
        )}
      </div>
    </div>
  );
}
