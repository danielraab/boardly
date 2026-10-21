// Shared Tailwind classes for buttons.

const textButton =
	"inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium disabled:cursor-not-allowed";

export const primaryButton = `${textButton} bg-ink text-paper hover:bg-ink/85 disabled:bg-ink/40`;

export const secondaryButton = `${textButton} border border-line text-ink hover:bg-surface disabled:opacity-50`;

export const dangerButton = `${textButton} bg-danger text-on-danger hover:bg-danger/90 disabled:opacity-50`;

export const iconButton =
	"inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted hover:bg-paper hover:text-ink";

export const headerButton =
	"inline-flex size-9 items-center justify-center rounded-lg text-muted hover:bg-surface hover:text-ink data-[popup-open]:bg-surface data-[popup-open]:text-ink";
