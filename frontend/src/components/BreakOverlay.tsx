import { localizeReason, t, type Locale } from '../i18n';
import type { RuntimeState } from '../types';

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
    <div className="overlay">
      <div className="overlay-card">
        <h2>{t(locale, 'breakTime')}</h2>
        <p>
          {t(locale, 'reason')}: {runtime.currentSession.reasons.map((reason) => localizeReason(reason, locale)).join(' + ')}
        </p>
        <p>
          {t(locale, 'remaining')}: {formatSec(runtime.currentSession.remainingSec, t(locale, 'off'))}
        </p>
        {runtime.overlaySkipAllowed && runtime.currentSession.canSkip && (
          <button onClick={onSkip}>{t(locale, 'emergencySkip')}</button>
        )}
      </div>
    </div>
  );
}
