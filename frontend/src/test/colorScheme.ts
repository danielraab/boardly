import { vi } from "vitest";

/**
 * Replaces window.matchMedia (missing in jsdom) with a stub that reports the
 * given OS color scheme and answers `(min-width: Npx)` for a viewport that is
 * 1280px wide. setDark() and setWidth() change them and notify listeners.
 */
export function stubColorScheme(dark = false) {
	const listeners = new Set<EventListener>();
	let prefersDark = dark;
	let width = 1280;

	const matches = (query: string) => {
		if (query === "(prefers-color-scheme: dark)") return prefersDark;
		const minWidth = /^\(min-width: (\d+)px\)$/.exec(query);
		return minWidth !== null && width >= Number(minWidth[1]);
	};

	vi.stubGlobal(
		"matchMedia",
		(query: string) =>
			({
				media: query,
				get matches() {
					return matches(query);
				},
				addEventListener: (_type: string, listener: EventListener) =>
					listeners.add(listener),
				removeEventListener: (_type: string, listener: EventListener) =>
					listeners.delete(listener),
			}) as unknown as MediaQueryList,
	);

	return {
		setDark(value: boolean) {
			prefersDark = value;
			for (const listener of listeners) listener(new Event("change"));
		},
		setWidth(value: number) {
			width = value;
			for (const listener of listeners) listener(new Event("change"));
		},
	};
}
