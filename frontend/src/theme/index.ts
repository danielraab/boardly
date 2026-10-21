import { useSyncExternalStore } from "react";

/** All themes in menu order. Keep in sync with the inline script in index.html. */
export const themes = [
	{ id: "white", mode: "light" },
	{ id: "black", mode: "dark" },
	{ id: "sand", mode: "light" },
	{ id: "sage", mode: "light" },
	{ id: "fjord", mode: "dark" },
	{ id: "dusk", mode: "dark" },
	{ id: "ember", mode: "dark" },
] as const;

export type ThemeId = (typeof themes)[number]["id"];
export type ThemeChoice = "system" | ThemeId;
export type ThemeMode = (typeof themes)[number]["mode"];

export const THEME_STORAGE_KEY = "famodash.theme";

const DARK_QUERY = "(prefers-color-scheme: dark)";

let current: ThemeChoice = "system";
const listeners = new Set<() => void>();

export function isThemeChoice(value: unknown): value is ThemeChoice {
	return value === "system" || themes.some((t) => t.id === value);
}

/** Reads the stored choice; anything missing or unknown means "system". */
export function readChoice(): ThemeChoice {
	try {
		const stored = localStorage.getItem(THEME_STORAGE_KEY);
		return isThemeChoice(stored) ? stored : "system";
	} catch {
		return "system";
	}
}

function prefersDark(): boolean {
	return typeof window.matchMedia === "function"
		? window.matchMedia(DARK_QUERY).matches
		: false;
}

/** Maps a choice to the theme that is shown; "system" follows the OS color scheme. */
export function resolveTheme(choice: ThemeChoice): ThemeId {
	if (choice !== "system") return choice;
	return prefersDark() ? "black" : "white";
}

export function applyTheme(choice: ThemeChoice = current): void {
	document.documentElement.dataset.theme = resolveTheme(choice);
}

export function getTheme(): ThemeChoice {
	return current;
}

/** Whether the shown theme is light or dark. */
export function getThemeMode(): ThemeMode {
	const id = resolveTheme(current);
	return themes.find((t) => t.id === id)?.mode ?? "light";
}

function subscribe(listener: () => void): () => void {
	listeners.add(listener);
	return () => listeners.delete(listener);
}

function notify(): void {
	for (const listener of listeners) listener();
}

/** Switches the theme and remembers the explicit choice in this browser. */
export function setTheme(choice: ThemeChoice): void {
	try {
		localStorage.setItem(THEME_STORAGE_KEY, choice);
	} catch {
		// Storage can be unavailable (private mode); the switch still applies to this visit.
	}
	current = choice;
	applyTheme();
	notify();
}

/**
 * Applies the stored theme and follows OS color scheme changes while the
 * choice is "system". Returns a function that stops following them.
 */
export function initTheme(): () => void {
	current = readChoice();
	applyTheme();
	if (typeof window.matchMedia !== "function") return () => {};
	const query = window.matchMedia(DARK_QUERY);
	const onChange = () => {
		if (current !== "system") return;
		applyTheme();
		notify();
	};
	query.addEventListener("change", onChange);
	return () => query.removeEventListener("change", onChange);
}

export function useTheme(): ThemeChoice {
	return useSyncExternalStore(subscribe, getTheme);
}

/** Re-renders when the shown theme switches between light and dark, including OS changes under System. */
export function useThemeMode(): ThemeMode {
	return useSyncExternalStore(subscribe, getThemeMode);
}
