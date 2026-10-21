import { useCallback, useSyncExternalStore } from "react";
import type { Columns } from "../api";

/**
 * The column count of a dashboard's grid for the current viewport: the desktop
 * count from the mobile breakpoint up, the mobile count below it. The
 * breakpoint is a setting, so it can't be a build-time Tailwind breakpoint.
 */
export function useColumns(columns: Columns) {
	const query = `(min-width: ${columns.mobileBreakpoint}px)`;
	const subscribe = useCallback(
		(onChange: () => void) => {
			const list = window.matchMedia(query);
			list.addEventListener("change", onChange);
			return () => list.removeEventListener("change", onChange);
		},
		[query],
	);
	const wide = useSyncExternalStore(
		subscribe,
		() => window.matchMedia(query).matches,
		() => true,
	);
	return wide ? columns.desktopColumns : columns.mobileColumns;
}
