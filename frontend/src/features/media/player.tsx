import type Hls from "hls.js";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { imageUrl } from "#/components/ImageIcon";
import { useMe } from "#/features/auth/session";
import { appTitle } from "#/lib/appTitle";
import type { PlayableSource } from "./api";

export type PlayerStatus = "idle" | "loading" | "playing" | "paused" | "error";

export type MediaPlayer = {
	/** The loaded source, or null when the player is closed. */
	current: PlayableSource | null;
	status: PlayerStatus;
	/** Plays a source, replacing the current one. */
	play: (source: PlayableSource) => void;
	/** Pauses the current source, or plays it again from the live position. */
	toggle: () => void;
	/** Stops playback and closes the player. */
	stop: () => void;
	/**
	 * Shows a changed source, such as a new title, if it is the current one.
	 * A changed URL is played if the old one was playing.
	 */
	update: (source: PlayableSource) => void;
	volume: number;
	setVolume: (volume: number) => void;
	muted: boolean;
	setMuted: (muted: boolean) => void;
};

const PlayerContext = createContext<MediaPlayer | null>(null);

/** The app's one media player; it lives in the app shell. */
export function useMediaPlayer() {
	const player = useContext(PlayerContext);
	if (!player) throw new Error("useMediaPlayer outside MediaPlayerProvider");
	return player;
}

const volumeKey = "boardly.player.volume";
const mutedKey = "boardly.player.muted";

function readStored<T>(
	key: string,
	parse: (raw: string) => T | undefined,
	fallback: T,
) {
	try {
		const raw = localStorage.getItem(key);
		return (raw === null ? undefined : parse(raw)) ?? fallback;
	} catch {
		return fallback;
	}
}

function store(key: string, value: string) {
	try {
		localStorage.setItem(key, value);
	} catch {
		// Without storage the player still works; the value is just not kept.
	}
}

const parseVolume = (raw: string) => {
	const v = Number(raw);
	return Number.isFinite(v) && v >= 0 && v <= 1 ? v : undefined;
};

const isHlsUrl = (url: string) => {
	try {
		return new URL(url).pathname.toLowerCase().endsWith(".m3u8");
	} catch {
		return false;
	}
};

/**
 * Owns the single <audio> element that every stream plays in, so playback
 * survives navigation. Audio is loaded straight from the stream's URL.
 */
export function MediaPlayerProvider({ children }: { children: ReactNode }) {
	const audioRef = useRef<HTMLAudioElement>(null);
	const hlsRef = useRef<Hls | null>(null);
	const [current, setCurrent] = useState<PlayableSource | null>(null);
	const [status, setStatus] = useState<PlayerStatus>("idle");
	const [volume, setVolumeState] = useState(() =>
		readStored(volumeKey, parseVolume, 1),
	);
	const [muted, setMutedState] = useState(() =>
		readStored(mutedKey, (raw) => raw === "true", false),
	);
	// Ignores events of a stream that was replaced or stopped meanwhile.
	const loadId = useRef(0);

	const detach = useCallback(() => {
		hlsRef.current?.destroy();
		hlsRef.current = null;
		const audio = audioRef.current;
		if (!audio) return;
		audio.pause();
		audio.removeAttribute("src");
		audio.load();
	}, []);

	const start = useCallback(
		(source: PlayableSource) => {
			const audio = audioRef.current;
			if (!audio) return;
			detach();
			const id = ++loadId.current;
			setStatus("loading");
			const fail = (e: unknown) => {
				if (id !== loadId.current) return;
				if (e instanceof DOMException && e.name === "AbortError") return;
				setStatus("error");
			};
			const playNow = () => {
				try {
					void Promise.resolve(audio.play()).catch(fail);
				} catch (e) {
					fail(e);
				}
			};
			if (
				isHlsUrl(source.url) &&
				!audio.canPlayType("application/vnd.apple.mpegurl")
			) {
				import("hls.js")
					.then(({ default: HlsClass }) => {
						if (id !== loadId.current) return;
						if (!HlsClass.isSupported()) return fail(null);
						const hls = new HlsClass();
						hlsRef.current = hls;
						hls.on(HlsClass.Events.ERROR, (_event, data) => {
							if (data.fatal) fail(null);
						});
						hls.loadSource(source.url);
						hls.attachMedia(audio);
						playNow();
					})
					.catch(fail);
				return;
			}
			audio.src = source.url;
			playNow();
		},
		[detach],
	);

	const play = useCallback(
		(source: PlayableSource) => {
			setCurrent(source);
			start(source);
		},
		[start],
	);

	const stop = useCallback(() => {
		loadId.current++;
		detach();
		setCurrent(null);
		setStatus("idle");
	}, [detach]);

	const pause = useCallback(() => {
		loadId.current++;
		audioRef.current?.pause();
		setStatus((s) => (s === "idle" ? s : "paused"));
	}, []);

	// A live stream resumes at the live position, not a stale buffer.
	const resume = useCallback(() => {
		if (current) start(current);
	}, [current, start]);

	const update = useCallback(
		(source: PlayableSource) => {
			if (current?.id !== source.id) return;
			setCurrent(source);
			if (
				source.url !== current.url &&
				(status === "playing" || status === "loading")
			)
				start(source);
		},
		[current, status, start],
	);

	const toggle = useCallback(() => {
		if (status === "playing" || status === "loading") pause();
		else resume();
	}, [status, pause, resume]);

	const setVolume = useCallback((v: number) => {
		const clamped = Math.min(1, Math.max(0, v));
		setVolumeState(clamped);
		store(volumeKey, String(clamped));
	}, []);

	const setMuted = useCallback((m: boolean) => {
		setMutedState(m);
		store(mutedKey, String(m));
	}, []);

	useEffect(() => {
		const audio = audioRef.current;
		if (!audio) return;
		audio.volume = volume;
		audio.muted = muted;
	}, [volume, muted]);

	// Signing out, or another user signing in, stops what the last one played.
	const me = useMe();
	const userId = me.data?.id ?? null;
	const lastUser = useRef(userId);
	useEffect(() => {
		if (lastUser.current !== null && lastUser.current !== userId) stop();
		lastUser.current = userId;
	}, [userId, stop]);

	// The operating system's media controls: lock screen, headset, media keys.
	useEffect(() => {
		const session =
			typeof navigator !== "undefined" ? navigator.mediaSession : undefined;
		if (!session) return;
		if (!current) {
			session.metadata = null;
			for (const action of ["play", "pause", "stop"] as const)
				session.setActionHandler(action, null);
			return;
		}
		if (typeof MediaMetadata !== "undefined") {
			session.metadata = new MediaMetadata({
				title: current.title,
				artist: appTitle(),
				artwork:
					current.icon?.type === "image"
						? [{ src: imageUrl(current.icon.imageId) }]
						: [],
			});
		}
		session.setActionHandler("play", () => resume());
		session.setActionHandler("pause", () => pause());
		session.setActionHandler("stop", () => stop());
	}, [current, pause, resume, stop]);

	const value = useMemo<MediaPlayer>(
		() => ({
			current,
			status,
			play,
			toggle,
			stop,
			update,
			volume,
			setVolume,
			muted,
			setMuted,
		}),
		[
			current,
			status,
			play,
			toggle,
			stop,
			update,
			volume,
			setVolume,
			muted,
			setMuted,
		],
	);

	return (
		<PlayerContext.Provider value={value}>
			{children}
			{/* biome-ignore lint/a11y/useMediaCaption: live radio has no captions */}
			<audio
				ref={audioRef}
				preload="none"
				onPlaying={() => setStatus("playing")}
				onWaiting={() => setStatus((s) => (s === "playing" ? "loading" : s))}
				onError={() => {
					if (audioRef.current?.getAttribute("src") && !hlsRef.current)
						setStatus("error");
				}}
				onEnded={() => setStatus("error")}
			/>
		</PlayerContext.Provider>
	);
}
