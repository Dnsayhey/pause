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
    <label className={`pill-switch ${checked ? 'is-on' : ''} ${disabled ? 'is-disabled' : ''}`}>
      <input
        type="checkbox"
        aria-label={ariaLabel}
        checked={checked}
        disabled={disabled}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className="pill-thumb" />
    </label>
  );
}

export function ToggleSwitchRow({ label, checked, disabled = false, onChange }: ToggleSwitchRowProps) {
  return (
    <div className="switch-row">
      <span>{label}</span>
      <ToggleSwitch ariaLabel={label} checked={checked} disabled={disabled} onChange={onChange} />
    </div>
  );
}
