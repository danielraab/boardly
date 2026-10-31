import { MapPin } from "lucide-react";
import { useTranslation } from "react-i18next";
import { formatPoint, type LocationValue } from "../document/types";

/** What a location is called: its label, else its coordinates. */
export const locationName = (location: LocationValue) =>
	location.label ?? formatPoint(location);

/**
 * Keeps Enter and Space on button from the editor around it, which would
 * otherwise take them for typing and keep the button from being activated.
 * React's handlers run too late for that, so it is a listener of its own.
 */
function keepActivationKeys(button: HTMLButtonElement | null) {
	if (!button) return;
	const stop = (e: KeyboardEvent) => {
		if (e.key === "Enter" || e.key === " ") e.stopPropagation();
	};
	button.addEventListener("keydown", stop);
	return () => button.removeEventListener("keydown", stop);
}

/**
 * A location block of a note: a pin with the location's label or
 * coordinates. Activating it calls onOpen with the chip, so the map can give
 * the focus back to it; it doesn't reach the card or editor around it.
 */
export function LocationChip({
	location,
	onOpen,
}: {
	location: LocationValue;
	onOpen: (chip: HTMLButtonElement) => void;
}) {
	const { t } = useTranslation();
	const name = locationName(location);
	return (
		<button
			ref={keepActivationKeys}
			type="button"
			aria-label={t("notes.locations.chip", { name })}
			className="pointer-events-auto inline-flex max-w-full items-center gap-1.5 rounded-full border border-line bg-paper/40 py-1 pr-3 pl-2 text-left hover:bg-ink/5 focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			onClick={(e) => {
				e.stopPropagation();
				onOpen(e.currentTarget);
			}}
		>
			<MapPin className="size-4 shrink-0 text-accent" aria-hidden />
			<span className="truncate">{name}</span>
		</button>
	);
}
