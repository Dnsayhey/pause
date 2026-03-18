import type { CSSProperties, SelectHTMLAttributes } from 'react';

type PillSelectOption = {
  value: string;
  label: string;
};

type PillSelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, 'children'> & {
  options: PillSelectOption[];
};

const classNameBase =
  'h-[22px] min-w-[82px] w-[82px] appearance-none rounded-full border border-[var(--glass-control-border)] bg-[var(--glass-control-bg)] bg-no-repeat bg-[position:right_7px_center] bg-[length:10px_6px] px-[10px] pr-[22px] text-xs leading-[1.2] font-medium text-[var(--control-text)] shadow-[var(--glass-control-shadow)] outline-offset-1 focus-visible:outline-2 focus-visible:outline-[rgba(21,123,209,0.28)] [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]';

export function PillSelect({ className = '', options, style, ...props }: PillSelectProps) {
  const mergedClassName = `${classNameBase} ${className}`.trim();
  const mergedStyle: CSSProperties = { backgroundImage: 'var(--select-arrow-image)', ...style };

  return (
    <select className={mergedClassName} style={mergedStyle} {...props}>
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );
}
