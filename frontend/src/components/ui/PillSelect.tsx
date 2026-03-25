import { useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent as ReactKeyboardEvent } from 'react';

type PillSelectOption = {
  value: string;
  label: string;
};

type PillSelectProps = {
  value: string;
  onChange?: (event: { target: { value: string } }) => void;
  options: PillSelectOption[];
  size?: 'default' | 'compact';
  variant?: 'default' | 'minimal';
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
};

const classNameBySize: Record<NonNullable<PillSelectProps['size']>, string> = {
  default:
    'h-7 min-w-[124px] w-[124px] rounded-[10px] bg-[position:right_10px_center] px-3 pr-8 text-xs font-medium',
  compact:
    'h-6 min-w-[96px] w-[96px] rounded-full bg-[position:right_8px_center] px-2.5 pr-7 text-[11px] font-medium'
};

const classNameBase =
  'appearance-none border border-[var(--control-border)] bg-[var(--surface-bg)] bg-no-repeat bg-[length:10px_6px] leading-[1.2] text-[var(--control-text)] shadow-[var(--shadow-soft)] outline-offset-1 transition-colors hover:border-[var(--surface-border-strong)] hover:bg-[var(--surface-muted)] focus-visible:border-[var(--toggle-on)] focus-visible:outline-2 focus-visible:outline-[var(--control-focus-ring)]';

const minimalClassName =
  'w-auto min-w-0 appearance-none border-0 bg-transparent bg-no-repeat bg-[position:right_0_center] bg-[length:10px_12px] px-0 pr-4 text-sm font-normal leading-[1.25] text-[var(--control-text)] shadow-none outline-none transition-colors hover:text-[var(--text-primary)] focus-visible:outline-none';

export function PillSelect({
  className = '',
  options,
  size = 'default',
  style,
  variant = 'default',
  value,
  onChange,
  disabled = false
}: PillSelectProps) {
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);
  const rootRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  const selectedIndex = useMemo(() => options.findIndex((option) => option.value === value), [options, value]);
  const selectedLabel = selectedIndex >= 0 ? options[selectedIndex]?.label : options[0]?.label ?? '';

  useEffect(() => {
    if (!open) return;
    const handlePointerDown = (event: PointerEvent) => {
      if (!rootRef.current) return;
      const target = event.target as Node | null;
      if (target && rootRef.current.contains(target)) return;
      setOpen(false);
      setActiveIndex(-1);
    };
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      setOpen(false);
      setActiveIndex(-1);
      triggerRef.current?.focus();
    };
    window.addEventListener('pointerdown', handlePointerDown);
    window.addEventListener('keydown', handleEscape);
    return () => {
      window.removeEventListener('pointerdown', handlePointerDown);
      window.removeEventListener('keydown', handleEscape);
    };
  }, [open]);

  const selectValue = (nextValue: string) => {
    if (!disabled && nextValue !== value) {
      onChange?.({ target: { value: nextValue } });
    }
    setOpen(false);
    setActiveIndex(-1);
    triggerRef.current?.focus();
  };

  const onTriggerKeyDown = (event: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (disabled) return;
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault();
      if (!open) {
        setOpen(true);
        setActiveIndex(selectedIndex >= 0 ? selectedIndex : 0);
        return;
      }
      const direction = event.key === 'ArrowDown' ? 1 : -1;
      const base = activeIndex >= 0 ? activeIndex : selectedIndex >= 0 ? selectedIndex : 0;
      const next = (base + direction + options.length) % options.length;
      setActiveIndex(next);
      return;
    }
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      if (!open) {
        setOpen(true);
        setActiveIndex(-1);
        return;
      }
      const option = options[activeIndex >= 0 ? activeIndex : selectedIndex >= 0 ? selectedIndex : 0];
      if (option) {
        selectValue(option.value);
      }
      return;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      setOpen(false);
      setActiveIndex(-1);
    }
  };

  const menuClassName =
    variant === 'minimal'
      ? 'absolute right-0 top-[calc(100%+6px)] z-40 max-h-60 w-max min-w-[96px] max-w-[180px] overflow-auto rounded-[8px] border border-[var(--dropdown-border)] bg-[var(--dropdown-bg)] p-0 shadow-[var(--dropdown-shadow)]'
      : 'absolute left-0 top-[calc(100%+6px)] z-40 max-h-60 w-max min-w-[96px] max-w-[180px] overflow-auto rounded-[8px] border border-[var(--dropdown-border)] bg-[var(--dropdown-bg)] p-0 shadow-[var(--dropdown-shadow)]';

  const mergedClassName =
    variant === 'minimal'
      ? `${minimalClassName} ${className}`.trim()
      : `${classNameBase} ${classNameBySize[size]} ${className}`.trim();
  const mergedStyle: CSSProperties = {
    backgroundImage: variant === 'minimal' ? 'var(--select-arrow-image-minimal)' : 'var(--select-arrow-image)',
    ...style
  };

  return (
    <div className="relative inline-flex" ref={rootRef}>
      <button
        ref={triggerRef}
        type="button"
        disabled={disabled}
        className={`${mergedClassName} text-left disabled:cursor-not-allowed disabled:opacity-45`}
        style={mergedStyle}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => {
          if (disabled) return;
          setOpen((prev) => !prev);
        }}
        onKeyDown={onTriggerKeyDown}
      >
        {selectedLabel}
      </button>

      {open && (
        <div className={menuClassName} role="listbox">
          {options.map((option, index) => {
            const isSelected = option.value === value;
            const isActive = index === activeIndex;
            return (
              <button
                key={option.value}
                type="button"
                role="option"
                aria-selected={isSelected}
                className={[
                  'flex w-full appearance-none border-0 bg-transparent cursor-pointer items-center justify-between whitespace-nowrap px-2.5 py-[1px] text-left text-[13px] leading-[1.2] transition-colors focus-visible:outline-none',
                  isSelected
                    ? 'text-[var(--dropdown-selected-text)] font-medium'
                    : 'text-[var(--dropdown-item-text)] hover:bg-[var(--dropdown-hover-bg)]',
                  isActive && !isSelected ? 'bg-[var(--dropdown-hover-bg)]' : ''
                ].join(' ')}
                onMouseEnter={() => {
                  setActiveIndex(index);
                }}
                onClick={() => {
                  selectValue(option.value);
                }}
              >
                <span>{option.label}</span>
                <span className={isSelected ? 'ml-3 text-[var(--dropdown-selected-text)]' : 'ml-3 opacity-0'} aria-hidden="true">
                  ✓
                </span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
