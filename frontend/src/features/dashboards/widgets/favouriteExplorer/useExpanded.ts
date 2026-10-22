import { useCallback, useState } from "react";

export function storageKey(widgetId: string) {
	return `famodash.explorer.${widgetId}`;
}

function read(widgetId: string): Set<string> {
	try {
		const ids: unknown = JSON.parse(
			localStorage.getItem(storageKey(widgetId)) ?? "[]",
		);
		return new Set(
			Array.isArray(ids)
				? ids.filter((id): id is string => typeof id === "string")
				: [],
		);
	} catch {
		return new Set();
	}
}

/**
 * The folders the viewer expanded in an explorer, remembered in this browser
 * per widget. Everything starts collapsed, also when storage is blocked.
 */
export function useExpanded(widgetId: string) {
	const [expanded, setExpanded] = useState(() => read(widgetId));
	const update = useCallback(
		(next: Set<string>) => {
			setExpanded(next);
			try {
				localStorage.setItem(storageKey(widgetId), JSON.stringify([...next]));
			} catch {
				// Blocked storage: the state lasts until the page is left.
			}
		},
		[widgetId],
	);
	return [expanded, update] as const;
}
