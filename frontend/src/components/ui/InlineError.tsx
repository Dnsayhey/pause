type InlineErrorProps = {
  message: string;
};

export function InlineError({ message }: InlineErrorProps) {
  return (
    <div className="mt-3 rounded-[10px] border border-[#f4bec4] bg-[#ffe8ea] px-[10px] py-2 text-[#8f1d2c]">
      {message}
    </div>
  );
}
