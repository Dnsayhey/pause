import type { ButtonHTMLAttributes, ReactNode } from 'react';

type PrimaryButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  children: ReactNode;
};

const baseClassName =
  'min-h-[var(--control-height)] cursor-pointer rounded-[11px] border border-transparent bg-[linear-gradient(130deg,var(--accent),var(--accent-strong))] px-[14px] py-2 font-semibold leading-[1.2] text-white hover:brightness-[1.04] disabled:cursor-not-allowed disabled:opacity-60';

export function PrimaryButton({ className = '', children, ...props }: PrimaryButtonProps) {
  const mergedClassName = `${baseClassName} ${className}`.trim();
  return (
    <button className={mergedClassName} {...props}>
      {children}
    </button>
  );
}
