// Whether embeds of a provider may be opened. Its player connects the browser
// to the provider, so the user has to agree first, like for maps: for the page
// or for this browser, per provider.
import { useSyncExternalStore } from "react";

/** The name to show for each provider of the server's registry. */
const providerNames: Record<string, string> = { spotify: "Spotify" };

export function providerName(provider: string): string {
	return providerNames[provider] ?? provider;
}

const storageKey = "boardly.embeds.allowedProviders";

function storedProviders(): string[] {
	try {
		const providers: unknown = JSON.parse(
			localStorage.getItem(storageKey) ?? "[]",
		);
		return Array.isArray(providers)
			? providers.filter((p): p is string => typeof p === "string")
			: [];
	} catch {
		return [];
	}
}

function storeProviders(providers: string[]) {
	try {
		if (providers.length)
			localStorage.setItem(storageKey, JSON.stringify(providers));
		else localStorage.removeItem(storageKey);
	} catch {
		// Blocked storage: the consent lasts until the page is reloaded.
	}
}

/** Providers allowed until the page is reloaded. */
const pageProviders = new Set<string>();
const listeners = new Set<() => void>();
const notify = () => {
	for (const listener of listeners) listener();
};

function isStored(provider: string): boolean {
	return storedProviders().includes(provider);
}

/** Whether embeds of provider may be opened now. */
export function isEmbedAllowed(provider: string): boolean {
	return pageProviders.has(provider) || isStored(provider);
}

/** Allows provider for the page and, with always, in this browser. */
export function allowEmbeds(provider: string, always: boolean) {
	pageProviders.add(provider);
	if (always && !isStored(provider))
		storeProviders([...storedProviders(), provider]);
	notify();
}

/**
 * Removes the consent stored for provider. It stays allowed for the page until
 * it is reloaded, so the open player doesn't disappear.
 */
export function withdrawEmbeds(provider: string) {
	pageProviders.add(provider);
	storeProviders(storedProviders().filter((p) => p !== provider));
	notify();
}

/** Forgets the consent given for the page; for tests. */
export function resetEmbedConsent() {
	pageProviders.clear();
	notify();
}

function subscribe(listener: () => void) {
	listeners.add(listener);
	return () => {
		listeners.delete(listener);
	};
}

/** Whether consent for provider is stored, kept up to date. */
export function useEmbedConsentStored(provider: string): boolean {
	return useSyncExternalStore(subscribe, () => isStored(provider));
}
