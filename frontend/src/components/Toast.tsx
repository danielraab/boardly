import { X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

/** A short message; a new id starts a new toast with the full duration. */
export type ToastMessage = {
	id: number;
	text: string;
	action?: { label: string; onClick: () => void };
};

/**
 * One message at a time at the bottom of the screen, announced politely. It
 * closes after `duration` ms; the time stops while the pointer or focus is on
 * it, so its action can be reached with the keyboard.
 */
export function Toast({
	toast,
	onClose,
	duration = 8000,
}: {
	toast: ToastMessage | null;
	onClose: () => void;
	duration?: number;
}) {
	const { t } = useTranslation();
	const [hovered, setHovered] = useState(false);
	const [focused, setFocused] = useState(false);
	const remaining = useRef(duration);
	const close = useRef(onClose);
	close.current = onClose;
	const id = toast?.id;
	const paused = hovered || focused;

	// A new toast restarts the time. A toast removed under the pointer or
	// focus (by its own action) never gets pointerleave or blur, so the pause
	// is reset too.
	// biome-ignore lint/correctness/useExhaustiveDependencies: runs per toast
	useEffect(() => {
		remaining.current = duration;
		setHovered(false);
		setFocused(false);
	}, [id, duration]);
	useEffect(() => {
		if (id === undefined || paused) return;
		const started = Date.now();
		const timer = setTimeout(() => close.current(), remaining.current);
		return () => {
			clearTimeout(timer);
			remaining.current -= Date.now() - started;
		};
	}, [id, paused]);

	return (
		<div
			aria-live="polite"
			className="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex justify-center px-4"
		>
			{toast && (
				<div
					role="status"
					className="pointer-events-auto flex max-w-full items-center gap-3 rounded-lg bg-ink py-2 pr-2 pl-4 text-sm text-paper shadow-[0_8px_24px_-12px_var(--color-shadow)]"
					onPointerEnter={() => setHovered(true)}
					onPointerLeave={() => setHovered(false)}
					onFocus={() => setFocused(true)}
					onBlur={() => setFocused(false)}
				>
					<span className="min-w-0">{toast.text}</span>
					{toast.action && (
						<button
							type="button"
							className="shrink-0 rounded-md px-2 py-1 font-semibold underline-offset-2 hover:underline"
							onClick={toast.action.onClick}
						>
							{toast.action.label}
						</button>
					)}
					<button
						type="button"
						className="inline-flex size-7 shrink-0 items-center justify-center rounded-md hover:bg-paper/15"
						aria-label={t("common.close")}
						onClick={onClose}
					>
						<X className="size-4" aria-hidden />
					</button>
				</div>
			)}
		</div>
	);
}
