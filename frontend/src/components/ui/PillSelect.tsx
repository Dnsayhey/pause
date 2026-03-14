import type { CSSProperties, SelectHTMLAttributes } from 'react';

type PillSelectOption = {
  value: string;
  label: string;
};

type PillSelectProps = Omit<SelectHTMLAttributes<HTMLSelectElement>, 'children'> & {
  options: PillSelectOption[];
};

const classNameBase =
  'h-[22px] min-w-[82px] w-[82px] appearance-none rounded-full border border-[rgba(255,255,255,0.74)] bg-[rgba(255,255,255,0.14)] bg-no-repeat bg-[position:right_7px_center] bg-[length:10px_6px] px-[10px] pr-[22px] text-xs leading-[1.2] font-medium text-[#122236] shadow-[inset_0_1px_0_rgba(255,255,255,0.34)] outline-offset-1 focus-visible:outline-2 focus-visible:outline-[rgba(21,123,209,0.28)] [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]';

const arrowStyle: CSSProperties = {
  backgroundImage:
    "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6' viewBox='0 0 10 6'%3E%3Cpath d='M1 1l4 4 4-4' stroke='%2333445C' stroke-width='1.4' fill='none' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E\")"
};

export function PillSelect({ className = '', options, style, ...props }: PillSelectProps) {
  const mergedClassName = `${classNameBase} ${className}`.trim();
  const mergedStyle = { ...arrowStyle, ...style };

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
