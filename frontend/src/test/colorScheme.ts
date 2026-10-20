import { vi } from "vitest";

/**
 * Replaces window.matchMedia (missing in jsdom) with a stub that reports the
 * given OS color scheme. setDark() changes it and notifies listeners.
 */
export function stubColorScheme(dark = false) {
	const listeners = new Set<EventListener>();
	let prefersDark = dark;

	vi.stubGlobal(
		"matchMedia",
		(query: string) =>
			({
				media: query,
				get matches() {
					return query === "(prefers-color-scheme: dark)" && prefersDark;
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
	};
}
