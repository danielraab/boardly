import {
	LoaderCircle,
	Pause,
	Play,
	Podcast,
	Radio,
	RotateCcw,
	Volume2,
	VolumeX,
	X,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { iconProps } from "#/features/notes/noteIcon";
import { useMediaPlayer, usePlayerProgress } from "./player";

/** The height the bar takes, which the page keeps free at its end. */
export const miniPlayerSpace = "h-20";

/** The same with an episode's position slider. */
const episodePlayerSpace = "h-28";

const controlButton =
	"inline-flex size-10 shrink-0 items-center justify-center rounded-full text-ink hover:bg-paper";

/** A time in seconds as m:ss, or h:mm:ss from an hour on. */
function clock(seconds: number) {
	const total = Math.max(0, Math.floor(seconds));
	const h = Math.floor(total / 3600);
	const m = Math.floor((total % 3600) / 60);
	const s = String(total % 60).padStart(2, "0");
	return h > 0 ? `${h}:${String(m).padStart(2, "0")}:${s}` : `${m}:${s}`;
}

/** An episode's position slider, with its times where there is room. */
function EpisodeProgress({ onSeek }: { onSeek: (seconds: number) => void }) {
	const { t } = useTranslation();
	const { position, duration } = usePlayerProgress();
	const time =
		"hidden shrink-0 text-xs text-muted tabular-nums min-[480px]:inline";

	return (
		<div className="mx-auto flex w-full max-w-screen-2xl items-center gap-2 px-4 pb-2 sm:px-6">
			<span className={time}>{clock(position)}</span>
			<input
				type="range"
				min={0}
				max={Math.max(1, Math.floor(duration))}
				step={1}
				value={Math.floor(position)}
				onChange={(e) => onSeek(Number(e.target.value))}
				aria-label={t("player.position")}
				aria-valuetext={`${clock(position)} / ${clock(duration)}`}
				className="min-w-0 flex-1 accent-ink"
			/>
			<span className={time}>{clock(duration)}</span>
		</div>
	);
}

/**
 * The bar at the bottom of every page while a stream is loaded: what plays,
 * play/pause, volume and close, or an error with "Try again".
 */
export function MiniPlayer() {
	const { t } = useTranslation();
	const player = useMediaPlayer();
	const { current, status } = player;
	if (!current) return null;
	const busy = status === "loading";
	const playing = status === "playing" || busy;
	const episode = current.kind === "episode";

	return (
		<>
			<div
				aria-hidden
				className={`${episode ? episodePlayerSpace : miniPlayerSpace} shrink-0`}
			/>
			<section
				aria-label={t("player.label")}
				className="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface/95 shadow-[0_-8px_24px_-16px_var(--color-shadow)] backdrop-blur"
			>
				<div className="mx-auto flex w-full max-w-screen-2xl items-center gap-3 px-4 py-3 sm:px-6">
					<span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-line bg-paper">
						<ChosenIcon
							{...iconProps(current.icon)}
							fallback={episode ? Podcast : Radio}
							className="size-6"
						/>
					</span>
					<div className="min-w-0 flex-1">
						<p className="truncate font-medium">{current.title}</p>
						{status === "error" ? (
							<p role="alert" className="truncate text-sm text-danger">
								{t(episode ? "player.episodeError" : "player.error")}
							</p>
						) : (
							<p className="truncate text-sm text-muted" aria-live="polite">
								{busy
									? t("player.loading")
									: status === "paused"
										? t("player.paused")
										: status === "finished"
											? t("player.finished")
											: episode
												? current.subtitle
												: t("player.live")}
							</p>
						)}
					</div>
					{status === "error" ? (
						<button
							type="button"
							onClick={() => player.play(current)}
							className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-line px-3 py-2 text-sm font-medium hover:bg-paper"
						>
							<RotateCcw className="size-4" aria-hidden />
							{t("player.retry")}
						</button>
					) : (
						<button
							type="button"
							onClick={player.toggle}
							aria-label={playing ? t("player.pause") : t("player.play")}
							className={controlButton}
						>
							{busy ? (
								<LoaderCircle
									className="size-6 animate-spin motion-reduce:animate-none"
									aria-hidden
									data-icon="loading"
								/>
							) : playing ? (
								<Pause className="size-6" aria-hidden />
							) : (
								<Play className="size-6" aria-hidden />
							)}
						</button>
					)}
					<div className="hidden items-center gap-1 min-[480px]:flex">
						<button
							type="button"
							onClick={() => player.setMuted(!player.muted)}
							aria-label={player.muted ? t("player.unmute") : t("player.mute")}
							aria-pressed={player.muted}
							className={iconButton}
						>
							{player.muted ? (
								<VolumeX className="size-4" aria-hidden />
							) : (
								<Volume2 className="size-4" aria-hidden />
							)}
						</button>
						<input
							type="range"
							min={0}
							max={100}
							step={5}
							value={Math.round(player.volume * 100)}
							onChange={(e) => player.setVolume(Number(e.target.value) / 100)}
							aria-label={t("player.volume")}
							className="w-24 accent-ink"
						/>
					</div>
					<button
						type="button"
						onClick={player.stop}
						aria-label={t("player.close")}
						className={iconButton}
					>
						<X className="size-4" aria-hidden />
					</button>
				</div>
				{episode && <EpisodeProgress onSeek={player.seek} />}
			</section>
		</>
	);
}
