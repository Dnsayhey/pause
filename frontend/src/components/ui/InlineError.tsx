type InlineErrorProps = {
  message: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function InlineError({ message, actionLabel, onAction }: InlineErrorProps) {
  const showAction = typeof onAction === 'function' && actionLabel && actionLabel.trim() !== '';
  return (
    <div className="mt-3 rounded-[10px] border border-[var(--error-border)] bg-[var(--error-bg)] px-[10px] py-2 text-[var(--error-text)]">
      <div className="flex items-center justify-between gap-2">
        <span>{message}</span>
        {showAction && (
          <button
            type="button"
            className="shrink-0 rounded-[8px] border border-[var(--error-border)] bg-[var(--surface-bg)] px-2 py-1 text-xs font-medium text-[var(--error-text)] hover:opacity-90"
            onClick={() => onAction?.()}
          >
            {actionLabel}
          </button>
        )}
      </div>
    </div>
  );
}
