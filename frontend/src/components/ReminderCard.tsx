import { useEffect, useRef, useState, type KeyboardEvent } from 'react';
import type { Locale } from '../i18n';
import { ToggleSwitch } from './ToggleSwitch';
import { GlassCard } from './ui';

function suppressEnter(event: KeyboardEvent<HTMLInputElement>) {
  if (event.key !== 'Enter') return;
  event.preventDefault();
  event.currentTarget.blur();
}

type ReminderCardProps = {
  locale: Locale;
  title: string;
  enabledLabel: string;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  titleAnchorClassName?: string;
  editLabel: string;
  doneLabel: string;
  intervalLabel: string;
  intervalValue: string;
  intervalUnitSec: number;
  intervalMin: number;
  intervalMax?: number;
  onIntervalChange: (value: string) => void;
  onIntervalNormalize: (value: string) => void;
  breakLabel: string;
  breakValue: string;
  breakUnitSec: number;
  breakMin: number;
  breakMax?: number;
  onBreakChange: (value: string) => void;
  onBreakNormalize: (value: string) => void;
  onDoneEdit: () => Promise<void> | void;
  onCancelEdit: () => void;
};

const inlineNumberInputClassName =
  'number-input w-[3.8ch] min-w-[3.8ch] cursor-text appearance-none border-0 border-b border-[var(--surface-border-strong)] bg-transparent px-0 py-[1px] text-right text-[15px] leading-[1.2] font-medium text-[var(--text-primary)] caret-[var(--accent)] shadow-none outline-none transition-colors duration-150 hover:border-[var(--field-border-muted)] focus:border-[var(--accent)]';

function parsePositiveInteger(value: string): number | null {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return null;
  const rounded = Math.round(numeric);
  return rounded > 0 ? rounded : null;
}

function unitLabel(locale: Locale, unitSec: number, value: string): string {
  const n = parsePositiveInteger(value) ?? 2;
  if (locale === 'zh-CN') {
    if (unitSec === 3600) return '小时';
    if (unitSec === 60) return '分钟';
    return '秒';
  }
  if (unitSec === 3600) return n === 1 ? 'hour' : 'hours';
  if (unitSec === 60) return n === 1 ? 'minute' : 'minutes';
  return n === 1 ? 'second' : 'seconds';
}

function summaryText(locale: Locale, intervalValue: string, intervalUnitSec: number, breakValue: string, breakUnitSec: number): string {
  const intervalUnit = unitLabel(locale, intervalUnitSec, intervalValue);
  const breakUnit = unitLabel(locale, breakUnitSec, breakValue);
  if (locale === 'zh-CN') {
    return `每隔 ${intervalValue || '--'} ${intervalUnit} 休息 ${breakValue || '--'} ${breakUnit}`;
  }
  return `Take a ${breakValue || '--'} ${breakUnit} break every ${intervalValue || '--'} ${intervalUnit}`;
}

export function ReminderCard({
  locale,
  title,
  enabledLabel,
  enabled,
  onEnabledChange,
  titleAnchorClassName = 'bg-[var(--toggle-on)]',
  editLabel,
  doneLabel,
  intervalLabel,
  intervalValue,
  intervalUnitSec,
  intervalMin,
  intervalMax,
  onIntervalChange,
  onIntervalNormalize,
  breakLabel,
  breakValue,
  breakUnitSec,
  breakMin,
  breakMax,
  onBreakChange,
  onBreakNormalize,
  onDoneEdit,
  onCancelEdit
}: ReminderCardProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const suppressBlurNormalizeRef = useRef(false);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    if (!isEditing || isSaving) return;

    const handlePointerDown = (event: PointerEvent) => {
      const root = rootRef.current;
      if (!root) return;
      const target = event.target as Node | null;
      if (target && root.contains(target)) {
        return;
      }
      suppressBlurNormalizeRef.current = true;
      onCancelEdit();
      setIsEditing(false);
      window.setTimeout(() => {
        suppressBlurNormalizeRef.current = false;
      }, 0);
    };

    document.addEventListener('pointerdown', handlePointerDown, true);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown, true);
    };
  }, [breakValue, intervalValue, isEditing, isSaving, onBreakNormalize, onIntervalNormalize]);

  const handleDone = async () => {
    if (isSaving) return;
    onIntervalNormalize(intervalValue);
    onBreakNormalize(breakValue);
    setIsSaving(true);
    try {
      await onDoneEdit();
      setIsEditing(false);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div ref={rootRef}>
      <GlassCard
        as="article"
        className="group/reminder border border-[var(--surface-border)] bg-[var(--app-bg)] shadow-none transition-colors hover:border-[var(--surface-border-strong)]"
      >
        <div className="mb-3 flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2.5">
            <span aria-hidden="true" className={`h-4 w-[3px] rounded-full ${titleAnchorClassName}`} />
            <h3 className="m-0 text-[18px]">{title}</h3>
          </div>
          <ToggleSwitch ariaLabel={enabledLabel} checked={enabled} onChange={onEnabledChange} />
        </div>
        <div className="flex items-start justify-between gap-2">
          {!isEditing ? (
            <p className="m-0 text-[15px] leading-[1.45] text-[var(--text-primary)]">
              {summaryText(locale, intervalValue, intervalUnitSec, breakValue, breakUnitSec)}
            </p>
          ) : (
            <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[15px] leading-[1.45] text-[var(--text-primary)]">
              {locale === 'zh-CN' ? (
                <>
                  <span>每隔</span>
                  <input
                    aria-label={intervalLabel}
                    className={inlineNumberInputClassName}
                    type="number"
                    min={intervalMin}
                    max={intervalMax}
                    step={1}
                    value={intervalValue}
                    onChange={(e) => onIntervalChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onIntervalNormalize(e.currentTarget.value);
                    }}
                  />
                  <span>{unitLabel(locale, intervalUnitSec, intervalValue)}</span>
                  <span>休息</span>
                  <input
                    aria-label={breakLabel}
                    className={inlineNumberInputClassName}
                    type="number"
                    min={breakMin}
                    max={breakMax}
                    step={1}
                    value={breakValue}
                    onChange={(e) => onBreakChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onBreakNormalize(e.currentTarget.value);
                    }}
                  />
                  <span>{unitLabel(locale, breakUnitSec, breakValue)}</span>
                </>
              ) : (
                <>
                  <span>Take a</span>
                  <input
                    aria-label={breakLabel}
                    className={inlineNumberInputClassName}
                    type="number"
                    min={breakMin}
                    max={breakMax}
                    step={1}
                    value={breakValue}
                    onChange={(e) => onBreakChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onBreakNormalize(e.currentTarget.value);
                    }}
                  />
                  <span>{unitLabel(locale, breakUnitSec, breakValue)}</span>
                  <span>break every</span>
                  <input
                    aria-label={intervalLabel}
                    className={inlineNumberInputClassName}
                    type="number"
                    min={intervalMin}
                    max={intervalMax}
                    step={1}
                    value={intervalValue}
                    onChange={(e) => onIntervalChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onIntervalNormalize(e.currentTarget.value);
                    }}
                  />
                  <span>{unitLabel(locale, intervalUnitSec, intervalValue)}</span>
                </>
              )}
            </div>
          )}
          <button
            type="button"
            disabled={isSaving}
            aria-label={isEditing ? doneLabel : editLabel}
            title={isEditing ? doneLabel : editLabel}
            className={[
              'mt-[1px] inline-flex h-6 w-6 flex-none items-center justify-center rounded-md border-0 bg-transparent text-[var(--text-tertiary)] transition-colors',
              isEditing ? 'opacity-100' : 'opacity-0 group-hover/reminder:opacity-100 focus-visible:opacity-100',
              'hover:bg-[var(--control-hover-bg)] hover:text-[var(--text-primary)] focus-visible:outline-none disabled:opacity-40'
            ].join(' ')}
            onClick={() => {
              if (isEditing) {
                void handleDone();
                return;
              }
              setIsEditing(true);
            }}
          >
            {isEditing ? (
              <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4" fill="none">
                <path
                  d="M5 10.5l3.2 3.2L15 7"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            ) : (
              <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4" fill="none">
                <path
                  d="M4.5 15.5h3l7.8-7.8a1.4 1.4 0 0 0 0-2l-1-1a1.4 1.4 0 0 0-2 0L4.5 12.5v3Z"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            )}
          </button>
        </div>
      </GlassCard>
    </div>
  );
}
