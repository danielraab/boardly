// Whether maps may be loaded. A map loads its style and tiles from the map
// provider, which sees the viewer's IP address, so unless the style is on the
// page's own host the viewer has to agree first: for the page ("Load map") or
// for this browser ("Always load maps"), per provider host.
import { useSyncExternalStore } from "react";

const defaultStyleUrl = "https://tiles.openfreemap.org/styles/liberty";

/** The style maps are drawn from; the server sets it in index.html. */
export function mapStyleUrl(): string {
	return (
		document
			.querySelector<HTMLMetaElement>('meta[name="boardly-map-style"]')
			?.content.trim() || defaultStyleUrl
	);
}

/** The host name maps are loaded from. */
export function mapHost(): string {
	try {
		return new URL(mapStyleUrl(), location.href).hostname;
	} catch {
		return new URL(defaultStyleUrl).hostname;
	}
}

/** Maps from the page's own host add no recipient and need no consent. */
export function needsConsent(host = mapHost()): boolean {
	return host !== location.hostname;
}

const storageKey = "boardly.maps.allowedHosts";

function storedHosts(): string[] {
	try {
		const hosts: unknown = JSON.parse(localStorage.getItem(storageKey) ?? "[]");
		return Array.isArray(hosts)
			? hosts.filter((h): h is string => typeof h === "string")
			: [];
	} catch {
		return [];
	}
}

function storeHosts(hosts: string[]) {
	try {
		if (hosts.length) localStorage.setItem(storageKey, JSON.stringify(hosts));
		else localStorage.removeItem(storageKey);
	} catch {
		// Blocked storage: the consent lasts until the page is reloaded.
	}
}

/** Hosts allowed until the page is reloaded. */
const pageHosts = new Set<string>();
const listeners = new Set<() => void>();
const notify = () => {
	for (const listener of listeners) listener();
};

/** Whether this browser stores consent for host. */
export function isStored(host: string): boolean {
	return storedHosts().includes(host);
}

/** Whether maps from host may be loaded now. */
export function isAllowed(host: string): boolean {
	return !needsConsent(host) || pageHosts.has(host) || isStored(host);
}

/** Allows maps from host for the page and, with always, in this browser. */
export function allowMaps(host: string, always: boolean) {
	pageHosts.add(host);
	if (always && !isStored(host)) storeHosts([...storedHosts(), host]);
	notify();
}

/**
 * Removes the consent stored for host. Maps stay allowed for the page until it
 * is reloaded, so the map shown doesn't disappear.
 */
export function withdrawMaps(host: string) {
	pageHosts.add(host);
	storeHosts(storedHosts().filter((h) => h !== host));
	notify();
}

/** Forgets the consent given for the page; for tests. */
export function resetMapConsent() {
	pageHosts.clear();
	notify();
}

function subscribe(listener: () => void) {
	listeners.add(listener);
	return () => {
		listeners.delete(listener);
	};
}

/** Whether maps from host may load, kept up to date across all maps. */
export function useMapsAllowed(host: string): boolean {
	return useSyncExternalStore(subscribe, () => isAllowed(host));
}

/** Whether consent for host is stored, kept up to date across all maps. */
export function useConsentStored(host: string): boolean {
	return useSyncExternalStore(subscribe, () => isStored(host));
}
