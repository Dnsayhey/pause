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
  variant?: 'rest' | 'notify';
  title: string;
  titleStatusText?: string;
  titleStatusLabel?: string;
  titleStatusTone?: 'pending' | 'unavailable';
  onTitleStatusClick?: () => void;
  enabledLabel: string;
  enabled: boolean;
  onEnabledChange: (enabled: boolean) => void;
  editLabel: string;
  doneLabel: string;
  cancelLabel: string;
  deleteLabel: string;
  metaText?: string;
  intervalLabel: string;
  intervalValue: string;
  intervalUnitSec: number;
  intervalMin: number;
  intervalMax?: number;
  canToggleIntervalUnit?: boolean;
  onIntervalUnitToggle?: () => void;
  onIntervalChange: (value: string) => void;
  onIntervalNormalize: (value: string) => void;
  breakLabel: string;
  breakValue: string;
  breakUnitSec: number;
  breakMin: number;
  breakMax?: number;
  canToggleBreakUnit?: boolean;
  onBreakUnitToggle?: () => void;
  onBreakChange: (value: string) => void;
  onBreakNormalize: (value: string) => void;
  onDoneEdit: () => Promise<void> | void;
  onCancelEdit: () => void;
  onDelete: () => Promise<void> | void;
};

const inlineNumberInputClassName =
  'number-input w-[3.8ch] min-w-[3.8ch] cursor-text appearance-none border-0 border-b border-[var(--surface-border-strong)] bg-transparent px-0 py-[1px] text-right text-[15px] leading-[1.2] font-medium text-[var(--text-primary)] caret-[var(--accent)] shadow-none outline-none transition-colors duration-150 hover:border-[var(--field-border-muted)] focus:border-[var(--accent)]';

const inlineUnitToggleClassName =
  'inline-flex items-center gap-0.5 rounded-md border border-transparent bg-transparent px-1 py-[1px] text-[var(--text-secondary)] underline decoration-dotted underline-offset-[3px] transition-colors hover:border-[var(--surface-border)] hover:text-[var(--text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)]';

function parsePositiveInteger(value: string): number | null {
  const trimmed = value.trim();
  if (!/^[0-9]+$/.test(trimmed)) return null;
  const numeric = Number(trimmed);
  if (!Number.isSafeInteger(numeric)) return null;
  return numeric > 0 ? numeric : null;
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

function summaryText(
  locale: Locale,
  variant: 'rest' | 'notify',
  intervalValue: string,
  intervalUnitSec: number,
  breakValue: string,
  breakUnitSec: number
): string {
  const intervalUnit = unitLabel(locale, intervalUnitSec, intervalValue);
  if (variant === 'notify') {
    if (locale === 'zh-CN') {
      return `每隔 ${intervalValue || '--'} ${intervalUnit} 提醒一次`;
    }
    return `Notify me every ${intervalValue || '--'} ${intervalUnit}`;
  }
  const breakUnit = unitLabel(locale, breakUnitSec, breakValue);
  if (locale === 'zh-CN') {
    return `每隔 ${intervalValue || '--'} ${intervalUnit} 休息 ${breakValue || '--'} ${breakUnit}`;
  }
  return `Take a ${breakValue || '--'} ${breakUnit} break every ${intervalValue || '--'} ${intervalUnit}`;
}

export function ReminderCard({
  locale,
  variant = 'rest',
  title,
  titleStatusText,
  titleStatusLabel,
  titleStatusTone = 'unavailable',
  onTitleStatusClick,
  enabledLabel,
  enabled,
  onEnabledChange,
  editLabel,
  doneLabel,
  cancelLabel,
  deleteLabel,
  metaText,
  intervalLabel,
  intervalValue,
  intervalUnitSec,
  canToggleIntervalUnit = false,
  onIntervalUnitToggle,
  onIntervalChange,
  onIntervalNormalize,
  breakLabel,
  breakValue,
  breakUnitSec,
  canToggleBreakUnit = false,
  onBreakUnitToggle,
  onBreakChange,
  onBreakNormalize,
  onDoneEdit,
  onCancelEdit,
  onDelete
}: ReminderCardProps) {
  const rootRef = useRef<HTMLDivElement | null>(null);
  const suppressBlurNormalizeRef = useRef(false);
  const [isEditing, setIsEditing] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isConfirmingDelete, setIsConfirmingDelete] = useState(false);
  const unitSwitchHint = locale === 'zh-CN' ? '点击切换单位' : 'Click to switch unit';
  const titleStatusClassName =
    titleStatusTone === 'pending'
      ? 'border-[var(--surface-border-strong)] bg-[var(--surface-muted)] text-[var(--text-secondary)] hover:border-[var(--surface-border-strong)] hover:bg-[var(--seg-hover-bg)] hover:text-[var(--text-primary)]'
      : 'border-[var(--error-border)] bg-[var(--error-bg)] text-[var(--error-text)] hover:opacity-95';

  useEffect(() => {
    if ((!isEditing && !isConfirmingDelete) || isSaving || isDeleting) return;

    const handlePointerDown = (event: PointerEvent) => {
      const root = rootRef.current;
      if (!root) return;
      const target = event.target as Node | null;
      if (target && root.contains(target)) {
        return;
      }
      if (isEditing) {
        suppressBlurNormalizeRef.current = true;
        onCancelEdit();
        setIsEditing(false);
      }
      if (isConfirmingDelete) {
        setIsConfirmingDelete(false);
      }
      window.setTimeout(() => {
        suppressBlurNormalizeRef.current = false;
      }, 0);
    };

    document.addEventListener('pointerdown', handlePointerDown, true);
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown, true);
    };
  }, [isConfirmingDelete, isDeleting, isEditing, isSaving, onCancelEdit]);

  const handleDone = async () => {
    if (isSaving) return;
    onIntervalNormalize(intervalValue);
    if (variant === 'rest') {
      onBreakNormalize(breakValue);
    }
    setIsSaving(true);
    try {
      await onDoneEdit();
      setIsEditing(false);
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (isDeleting || isSaving || isEditing || isConfirmingDelete) return;
    setIsConfirmingDelete(true);
  };

  const handleDeleteConfirm = async () => {
    if (isDeleting || isSaving || isEditing || !isConfirmingDelete) return;
    setIsDeleting(true);
    try {
      await onDelete();
    } finally {
      setIsDeleting(false);
      setIsConfirmingDelete(false);
    }
  };

  const renderUnitToken = (value: string, unitSec: number, canToggle: boolean, onToggle?: () => void) => {
    const label = unitLabel(locale, unitSec, value);
    if (!canToggle || !onToggle) {
      return <span>{label}</span>;
    }
    return (
      <button
        type="button"
        className={inlineUnitToggleClassName}
        title={unitSwitchHint}
        aria-label={`${label} · ${unitSwitchHint}`}
        onClick={onToggle}
      >
        <span>{label}</span>
        <svg aria-hidden="true" viewBox="0 0 12 12" className="h-2.5 w-2.5" fill="none">
          <path d="M3 4.5 6 7.5 9 4.5" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>
    );
  };

  return (
    <div ref={rootRef}>
      <GlassCard
        as="article"
        className="group/reminder border border-[var(--surface-border)] bg-[var(--app-bg)] shadow-none transition-colors hover:border-[var(--surface-border-strong)]"
      >
        <div className="mb-3 flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2.5">
            <span
              aria-hidden="true"
              className={`h-4 w-[3px] rounded-full transition-colors ${
                enabled ? 'bg-[var(--text-primary)] opacity-90' : 'bg-[var(--text-tertiary)] opacity-50'
              }`}
            />
            <h3 className="m-0 flex min-w-0 items-center gap-2 text-[18px]">
              <span className="truncate">{title}</span>
              {titleStatusText ? (
                <button
                  type="button"
                  className={`inline-flex h-5 shrink-0 items-center justify-center rounded-full border px-2 text-[10px] font-medium leading-none transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--control-focus-ring)] ${titleStatusClassName}`}
                  title={titleStatusLabel}
                  aria-label={titleStatusLabel}
                  onClick={onTitleStatusClick}
                >
                  {titleStatusText}
                </button>
              ) : null}
            </h3>
          </div>
          <ToggleSwitch ariaLabel={enabledLabel} checked={enabled} onChange={onEnabledChange} />
        </div>
        <div className="flex items-start justify-between gap-2">
          {!isEditing ? (
            <p className="m-0 text-[15px] leading-[1.45] text-[var(--text-primary)]">
              {summaryText(locale, variant, intervalValue, intervalUnitSec, breakValue, breakUnitSec)}
            </p>
          ) : (
            <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[15px] leading-[1.45] text-[var(--text-primary)]">
              {variant === 'notify' ? (
                locale === 'zh-CN' ? (
                  <>
                    <span>每隔</span>
                    <input
                      aria-label={intervalLabel}
                      className={inlineNumberInputClassName}
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      value={intervalValue}
                      onChange={(e) => onIntervalChange(e.target.value)}
                      onKeyDown={suppressEnter}
                      onBlur={(e) => {
                        if (suppressBlurNormalizeRef.current) return;
                        onIntervalNormalize(e.currentTarget.value);
                      }}
                    />
                    {renderUnitToken(intervalValue, intervalUnitSec, canToggleIntervalUnit, onIntervalUnitToggle)}
                    <span>提醒一次</span>
                  </>
                ) : (
                  <>
                    <span>Notify me every</span>
                    <input
                      aria-label={intervalLabel}
                      className={inlineNumberInputClassName}
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      value={intervalValue}
                      onChange={(e) => onIntervalChange(e.target.value)}
                      onKeyDown={suppressEnter}
                      onBlur={(e) => {
                        if (suppressBlurNormalizeRef.current) return;
                        onIntervalNormalize(e.currentTarget.value);
                      }}
                    />
                    {renderUnitToken(intervalValue, intervalUnitSec, canToggleIntervalUnit, onIntervalUnitToggle)}
                  </>
                )
              ) : locale === 'zh-CN' ? (
                <>
                  <span>每隔</span>
                  <input
                    aria-label={intervalLabel}
                    className={inlineNumberInputClassName}
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={intervalValue}
                    onChange={(e) => onIntervalChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onIntervalNormalize(e.currentTarget.value);
                    }}
                  />
                  {renderUnitToken(intervalValue, intervalUnitSec, canToggleIntervalUnit, onIntervalUnitToggle)}
                  <span>休息</span>
                  <input
                    aria-label={breakLabel}
                    className={inlineNumberInputClassName}
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={breakValue}
                    onChange={(e) => onBreakChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onBreakNormalize(e.currentTarget.value);
                    }}
                  />
                  {renderUnitToken(breakValue, breakUnitSec, canToggleBreakUnit, onBreakUnitToggle)}
                </>
              ) : (
                <>
                  <span>Take a</span>
                  <input
                    aria-label={breakLabel}
                    className={inlineNumberInputClassName}
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={breakValue}
                    onChange={(e) => onBreakChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onBreakNormalize(e.currentTarget.value);
                    }}
                  />
                  {renderUnitToken(breakValue, breakUnitSec, canToggleBreakUnit, onBreakUnitToggle)}
                  <span>break every</span>
                  <input
                    aria-label={intervalLabel}
                    className={inlineNumberInputClassName}
                    type="text"
                    inputMode="numeric"
                    pattern="[0-9]*"
                    value={intervalValue}
                    onChange={(e) => onIntervalChange(e.target.value)}
                    onKeyDown={suppressEnter}
                    onBlur={(e) => {
                      if (suppressBlurNormalizeRef.current) return;
                      onIntervalNormalize(e.currentTarget.value);
                    }}
                  />
                  {renderUnitToken(intervalValue, intervalUnitSec, canToggleIntervalUnit, onIntervalUnitToggle)}
                </>
              )}
            </div>
          )}
          <div className="mt-[1px] inline-flex items-center gap-1">
            <button
              type="button"
              disabled={isSaving || isDeleting || isConfirmingDelete}
              aria-label={isEditing ? doneLabel : editLabel}
              title={isEditing ? doneLabel : editLabel}
              className={[
                'inline-flex h-6 w-6 flex-none items-center justify-center rounded-md border-0 bg-transparent text-[var(--text-tertiary)] transition-colors',
                isEditing
                  ? 'opacity-100'
                  : isConfirmingDelete
                    ? 'pointer-events-none opacity-0'
                    : 'opacity-0 group-hover/reminder:opacity-100 focus-visible:opacity-100',
                'hover:bg-[var(--control-hover-bg)] hover:text-[var(--text-primary)] focus-visible:outline-none disabled:opacity-40'
              ].join(' ')}
              onClick={() => {
                if (isEditing) {
                  void handleDone();
                  return;
                }
                setIsConfirmingDelete(false);
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

            <button
              type="button"
              disabled={isDeleting || isSaving || isEditing || isConfirmingDelete}
              aria-label={deleteLabel}
              title={deleteLabel}
              className={[
                'inline-flex h-6 w-6 flex-none items-center justify-center rounded-md border-0 bg-transparent text-[var(--text-tertiary)] transition-colors',
                isEditing || isConfirmingDelete
                  ? 'pointer-events-none opacity-0'
                  : 'opacity-0 group-hover/reminder:opacity-100 focus-visible:opacity-100',
                'hover:bg-[var(--control-hover-bg)] hover:text-[var(--negative-text)] focus-visible:outline-none disabled:opacity-40'
              ].join(' ')}
              onClick={() => {
                void handleDelete();
              }}
            >
              <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4" fill="none">
                <path
                  d="M4.5 5.5h11M8 5.5V4.2c0-.5.4-.9.9-.9h2.2c.5 0 .9.4.9.9v1.3M7.2 8.5v6M10 8.5v6M12.8 8.5v6M6.4 16.7h7.2c.5 0 .9-.4.9-.9l.6-10.3H4.9l.6 10.3c0 .5.4.9.9.9Z"
                  stroke="currentColor"
                  strokeWidth="1.4"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </button>

            {isConfirmingDelete ? (
              <>
                <button
                  type="button"
                  disabled={isDeleting}
                  aria-label={cancelLabel}
                  title={cancelLabel}
                  className="inline-flex h-6 w-6 flex-none items-center justify-center rounded-md border-0 bg-transparent text-[var(--text-tertiary)] transition-colors hover:bg-[var(--control-hover-bg)] hover:text-[var(--text-primary)] focus-visible:outline-none disabled:opacity-40"
                  onClick={() => {
                    setIsConfirmingDelete(false);
                  }}
                >
                  <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4" fill="none">
                    <path
                      d="M5 5l10 10M15 5L5 15"
                      stroke="currentColor"
                      strokeWidth="1.7"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                </button>

                <button
                  type="button"
                  disabled={isDeleting}
                  aria-label={deleteLabel}
                  title={deleteLabel}
                  className="inline-flex h-6 w-6 flex-none items-center justify-center rounded-md border-0 bg-transparent text-[var(--negative-text)] transition-colors hover:bg-[var(--control-hover-bg)] focus-visible:outline-none disabled:opacity-40"
                  onClick={() => {
                    void handleDeleteConfirm();
                  }}
                >
                  <svg aria-hidden="true" viewBox="0 0 20 20" className="h-4 w-4" fill="none">
                    <path
                      d="M5 10.5l3.2 3.2L15 7"
                      stroke="currentColor"
                      strokeWidth="1.8"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                </button>
              </>
            ) : null}
          </div>
        </div>
        {metaText ? <p className="mt-2 m-0 text-xs leading-[1.35] text-[var(--text-tertiary)]">{metaText}</p> : null}
      </GlassCard>
    </div>
  );
}
