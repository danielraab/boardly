// Shared Tailwind classes for Base UI dialog parts.

export const dialogBackdrop =
	"fixed inset-0 bg-ink/30 transition-opacity duration-150 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0 motion-reduce:transition-none";

export const dialogPopup =
	"fixed top-1/2 left-1/2 w-[28rem] max-w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 rounded-xl border border-line bg-surface p-6 text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] transition-[opacity,scale] duration-150 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 data-[starting-style]:scale-95 data-[starting-style]:opacity-0 motion-reduce:transition-none";

export const dangerButton =
	"inline-flex items-center justify-center rounded-full bg-danger px-6 py-2.5 font-medium text-on-danger hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-50";

export const textInput =
	"mt-2 w-full rounded-lg border border-line bg-paper/40 px-4 py-2.5 text-ink outline-none focus-visible:border-ink";
