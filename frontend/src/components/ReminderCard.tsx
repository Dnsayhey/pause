import { type KeyboardEvent } from 'react';
import { ToggleSwitch } from './ToggleSwitch';
import { GlassCard } from './ui';

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

const rowLabelClassName =
  'flex flex-col items-start justify-between gap-3 text-sm leading-[1.35] sm:flex-row sm:items-center';
const numberInputClassName =
  'number-input w-[3.6ch] min-w-[3.6ch] cursor-text appearance-none border-0 border-b border-b-transparent bg-transparent px-0 py-[2px] text-right text-sm leading-[1.2] font-normal text-[var(--text-primary)] caret-[rgba(15,130,107,0.9)] shadow-none outline-none transition-colors duration-150 hover:border-b-[var(--field-border-muted)] focus:border-b-[rgba(15,130,107,0.88)]';

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
    <GlassCard as="article">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h3 className="m-0 text-[18px]">{title}</h3>
        <ToggleSwitch ariaLabel={enabledLabel} checked={enabled} onChange={onEnabledChange} />
      </div>
      <div className="grid gap-2.5">
        <label className={rowLabelClassName}>
          {intervalLabel}
          <input
            className={numberInputClassName}
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
        <label className={rowLabelClassName}>
          {breakLabel}
          <input
            className={numberInputClassName}
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
    </GlassCard>
  );
}
