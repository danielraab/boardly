const key = "boardly.player.positions";

/** The most episodes whose position is remembered. */
export const maxRememberedEpisodes = 200;

type Entry = [id: string, seconds: number];

/** The remembered positions, most recently played first. */
function read(): Entry[] {
	try {
		const parsed: unknown = JSON.parse(localStorage.getItem(key) ?? "[]");
		return Array.isArray(parsed) ? (parsed as Entry[]) : [];
	} catch {
		return [];
	}
}

function write(entries: Entry[]) {
	try {
		localStorage.setItem(key, JSON.stringify(entries));
	} catch {
		// Without storage the player still works; episodes start at the beginning.
	}
}

/** Where to continue an episode, in seconds; 0 when it isn't known. */
export function readPosition(id: string) {
	const seconds = read().find(([entry]) => entry === id)?.[1];
	return typeof seconds === "number" && seconds > 0 ? seconds : 0;
}

export function savePosition(id: string, seconds: number) {
	write(
		[[id, seconds] as Entry, ...read().filter(([entry]) => entry !== id)].slice(
			0,
			maxRememberedEpisodes,
		),
	);
}

export function forgetPosition(id: string) {
	write(read().filter(([entry]) => entry !== id));
}
