import { type KeyboardEvent } from 'react';
import { ToggleSwitch } from './ToggleSwitch';

function blurOnEnter(event: KeyboardEvent<HTMLInputElement>) {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  event.currentTarget.blur();
}

type ReminderCardProps = {
  title: string;
  enabledLabel: string;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  intervalLabel: string;
  intervalValue: string;
  intervalMin: number;
  intervalMax?: number;
  onIntervalChange: (value: string) => void;
  onIntervalCommit: (value: string) => Promise<void>;
  breakLabel: string;
  breakValue: string;
  breakMin: number;
  breakMax?: number;
  onBreakChange: (value: string) => void;
  onBreakCommit: (value: string) => Promise<void>;
};

export function ReminderCard({
  title,
  enabledLabel,
  enabled,
  onEnabledChange,
  intervalLabel,
  intervalValue,
  intervalMin,
  intervalMax,
  onIntervalChange,
  onIntervalCommit,
  breakLabel,
  breakValue,
  breakMin,
  breakMax,
  onBreakChange,
  onBreakCommit
}: ReminderCardProps) {
  return (
    <article className="card">
      <div className="card-title-row">
        <h3>{title}</h3>
        <ToggleSwitch ariaLabel={enabledLabel} checked={enabled} onChange={onEnabledChange} />
      </div>
      <div className="form-grid">
        <label>
          {intervalLabel}
          <input
            className="rule-number-input"
            type="number"
            min={intervalMin}
            max={intervalMax}
            step={1}
            value={intervalValue}
            onChange={(e) => onIntervalChange(e.target.value)}
            onKeyDown={blurOnEnter}
            onBlur={(e) => {
              void onIntervalCommit(e.currentTarget.value);
            }}
          />
        </label>
        <label>
          {breakLabel}
          <input
            className="rule-number-input"
            type="number"
            min={breakMin}
            max={breakMax}
            step={1}
            value={breakValue}
            onChange={(e) => onBreakChange(e.target.value)}
            onKeyDown={blurOnEnter}
            onBlur={(e) => {
              void onBreakCommit(e.currentTarget.value);
            }}
          />
        </label>
      </div>
    </article>
  );
}
