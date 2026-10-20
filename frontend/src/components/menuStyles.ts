// Shared Tailwind classes for Base UI menu parts.

export const menuPopup =
	"min-w-48 rounded-lg border border-line bg-surface p-1 text-sm text-ink shadow-[0_8px_24px_-12px_rgb(29_39_51/0.35)] outline-none origin-[var(--transform-origin)] transition-[opacity,scale] duration-100 data-[starting-style]:scale-95 data-[starting-style]:opacity-0 data-[ending-style]:opacity-0 motion-reduce:transition-none";

export const menuItem =
	"flex w-full cursor-default items-center gap-2 rounded-md px-3 py-2 select-none outline-none data-[highlighted]:bg-paper";

export const headerButton =
	"inline-flex size-10 items-center justify-center rounded-full text-muted hover:bg-surface hover:text-ink data-[popup-open]:bg-surface data-[popup-open]:text-ink";
