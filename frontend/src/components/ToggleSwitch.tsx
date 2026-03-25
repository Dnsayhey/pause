export type ToggleSwitchProps = {
  ariaLabel: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
};

export type ToggleSwitchRowProps = {
  label: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
};

export function ToggleSwitch({ ariaLabel, checked, disabled = false, onChange }: ToggleSwitchProps) {
  return (
    <label
      className={[
        'relative inline-block h-5 w-[34px] rounded-full p-[2px] transition-colors duration-200',
        checked ? 'bg-[linear-gradient(130deg,var(--toggle-on),var(--toggle-on-strong))]' : 'bg-[var(--toggle-off-bg)]',
        disabled ? 'opacity-60' : ''
      ].join(' ')}
    >
      <input
        type="checkbox"
        aria-label={ariaLabel}
        className="absolute inset-0 z-[1] m-0 cursor-pointer opacity-0 disabled:cursor-not-allowed"
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span
        className={[
          'pointer-events-none block h-4 w-4 rounded-full bg-[var(--surface-bg)] shadow-[var(--shadow-knob)] transition-transform duration-200',
          checked ? 'translate-x-[14px]' : 'translate-x-0'
        ].join(' ')}
      />
    </label>
  );
}

export function ToggleSwitchRow({ label, checked, disabled = false, onChange }: ToggleSwitchRowProps) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm font-normal leading-[1.35]">
      <span>{label}</span>
      <ToggleSwitch ariaLabel={label} checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}
