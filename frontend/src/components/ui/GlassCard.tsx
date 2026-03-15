import type { ElementType, ReactNode } from 'react';

type GlassCardProps = {
  as?: ElementType;
  className?: string;
  children: ReactNode;
};

const baseClassName =
  'mt-3 rounded-[18px] border border-[var(--surface-border)] bg-[var(--surface-bg)] p-[18px] shadow-[var(--surface-shadow),var(--surface-inner-highlight)] [backdrop-filter:blur(var(--surface-blur))_saturate(var(--surface-sat))]';

export function GlassCard({ as: Component = 'section', className = '', children }: GlassCardProps) {
  const mergedClassName = `${baseClassName} ${className}`.trim();
  return <Component className={mergedClassName}>{children}</Component>;
}
