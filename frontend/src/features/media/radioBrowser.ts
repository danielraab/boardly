// Searches radio-browser.info, a free community directory of internet radio
// stations. The browser calls it directly; Boardly's server isn't involved.

/** The directory's home page, linked from the search's notice. */
export const radioBrowserHome = "https://www.radio-browser.info";

const serverList = "https://all.api.radio-browser.info/json/servers";
const fallbackServer = "de1.api.radio-browser.info";

/** How many results a search shows. */
export const searchLimit = 30;

/** A station as the search shows it. */
export type StationResult = {
	id: string;
	name: string;
	/** The stream URL, resolved by the directory where it could. */
	url: string;
	/** The station's logo, or null. */
	logoUrl: string | null;
	country: string | null;
	tags: string[];
	/** In kbit/s, or null when unknown. */
	bitrate: number | null;
	/** Only https streams can be played on an https page. */
	playable: boolean;
};

type RawStation = {
	stationuuid: string;
	name: string;
	url: string;
	url_resolved?: string;
	favicon?: string;
	country?: string;
	tags?: string;
	bitrate?: number;
};

let server: Promise<string> | null = null;

/** Picks a mirror once per page load, at random, or the fallback. */
function pickServer(signal?: AbortSignal): Promise<string> {
	server ??= fetch(serverList, { signal })
		.then((res) => (res.ok ? res.json() : Promise.reject(res.status)))
		.then((list: { name: string }[]) => {
			const names = list.map((s) => s.name).filter(Boolean);
			return names[Math.floor(Math.random() * names.length)] ?? fallbackServer;
		})
		.catch(() => {
			server = null; // try the list again next time
			return fallbackServer;
		});
	return server;
}

/** Forgets the chosen mirror; for tests. */
export function resetRadioBrowser() {
	server = null;
}

export function toResult(raw: RawStation): StationResult {
	const url = (raw.url_resolved || raw.url).trim();
	const name = raw.name.trim();
	return {
		id: raw.stationuuid,
		name: name || url,
		url,
		logoUrl: raw.favicon?.trim() || null,
		country: raw.country?.trim() || null,
		tags: (raw.tags ?? "")
			.split(",")
			.map((t) => t.trim())
			.filter(Boolean)
			.slice(0, 3),
		bitrate: raw.bitrate ? raw.bitrate : null,
		playable: url.toLowerCase().startsWith("https://"),
	};
}

/**
 * Searches stations by name, most popular first. Throws when the directory
 * can't be reached, and an AbortError when signal aborts.
 */
export async function searchStations(
	query: string,
	signal?: AbortSignal,
): Promise<StationResult[]> {
	const host = await pickServer(signal);
	const params = new URLSearchParams({
		name: query,
		limit: String(searchLimit),
		order: "clickcount",
		reverse: "true",
		hidebroken: "true",
	});
	const res = await fetch(`https://${host}/json/stations/search?${params}`, {
		signal,
	});
	if (!res.ok) throw new Error(`radio-browser: ${res.status}`);
	const list = (await res.json()) as RawStation[];
	return list.map(toResult);
}
