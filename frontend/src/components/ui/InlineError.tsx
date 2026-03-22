type InlineErrorProps = {
  message: string;
};

export function InlineError({ message }: InlineErrorProps) {
  return (
    <div className="mt-3 rounded-[10px] border border-[var(--error-border)] bg-[var(--error-bg)] px-[10px] py-2 text-[var(--error-text)]">
      {message}
    </div>
  );
}
