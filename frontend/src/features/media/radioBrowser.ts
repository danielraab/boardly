// Searches radio-browser.info, a free community directory of internet radio
// stations, through Boardly's server, so that the browser sends the user's IP
// address neither to the directory nor to the hosts of station logos.
import { api } from "#/lib/api";

/** The directory's home page, linked from the search's notice. */
export const radioBrowserHome = "https://www.radio-browser.info";

/** A station as the search shows it. */
export type StationResult = {
	id: string;
	name: string;
	/** The stream URL, resolved by the directory where it could. */
	url: string;
	/** The directory's logo URL, or null; sent when the station is added. */
	logoUrl: string | null;
	/** The logo through Boardly's server, or null; this is what is shown. */
	logo: string | null;
	country: string | null;
	tags: string[];
	/** In kbit/s, or null when unknown. */
	bitrate: number | null;
	/** Only https streams can be played on an https page. */
	playable: boolean;
};

/**
 * Searches stations by name, most popular first. Throws an ApiError when the
 * directory can't be reached (`directory_unreachable`) or there were too many
 * searches (`rate_limited`), and an AbortError when signal aborts.
 */
export function searchStations(
	query: string,
	signal?: AbortSignal,
): Promise<StationResult[]> {
	return api<StationResult[]>(
		`/api/media/search?${new URLSearchParams({ q: query })}`,
		{ signal },
	);
}
