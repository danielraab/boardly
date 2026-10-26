import { useStoredIdSet } from "#/lib/useStoredIdSet";

export function storageKey(widgetId: string) {
	return `boardly.explorer.${widgetId}`;
}

/**
 * The folders the viewer expanded in an explorer, remembered in this browser
 * per widget. Everything starts collapsed, also when storage is blocked.
 */
export function useExpanded(widgetId: string) {
	return useStoredIdSet(storageKey(widgetId));
}
