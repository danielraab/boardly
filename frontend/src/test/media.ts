import { vi } from "vitest";
import type { MediaSource } from "#/features/media/api";

/** A stream of the media library, at position. */
export function station(
	id: string,
	title: string,
	position = 0,
	fields: Partial<MediaSource> = {},
): MediaSource {
	return {
		id,
		kind: "stream",
		title,
		url: `https://stream.example/${id}.mp3`,
		icon: null,
		position,
		...fields,
	};
}

/**
 * Replaces the media methods jsdom doesn't implement. play resolves unless
 * playResult is changed; the "playing" event must be fired by the test.
 */
export function stubAudio() {
	const state = { playResult: () => Promise.resolve() };
	const play = vi
		.spyOn(HTMLMediaElement.prototype, "play")
		.mockImplementation(() => state.playResult());
	const pause = vi
		.spyOn(HTMLMediaElement.prototype, "pause")
		.mockImplementation(() => {});
	const load = vi
		.spyOn(HTMLMediaElement.prototype, "load")
		.mockImplementation(() => {});
	return {
		play,
		pause,
		load,
		state,
		/** The app's one audio element. */
		audio: () => {
			const all = document.querySelectorAll("audio");
			if (all.length !== 1) throw new Error(`${all.length} audio elements`);
			return all[0];
		},
	};
}
