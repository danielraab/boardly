import { useCallback, useState } from "react";

function read(key: string): Set<string> {
	try {
		const ids: unknown = JSON.parse(localStorage.getItem(key) ?? "[]");
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
 * A set of IDs remembered in this browser under key. It starts empty when
 * nothing is stored or storage is blocked; then it lasts until the page is left.
 */
export function useStoredIdSet(key: string) {
	const [state, setState] = useState(() => ({ key, ids: read(key) }));
	// Another key (another widget or user) reads its own set.
	const ids = state.key === key ? state.ids : read(key);
	if (state.key !== key) setState({ key, ids });
	const update = useCallback(
		(next: ReadonlySet<string>) => {
			setState({ key, ids: new Set(next) });
			try {
				localStorage.setItem(key, JSON.stringify([...next]));
			} catch {
				// Blocked storage: the state lasts until the page is left.
			}
		},
		[key],
	);
	return [ids as ReadonlySet<string>, update] as const;
}
