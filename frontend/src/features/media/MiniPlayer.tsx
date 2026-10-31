import {
	LoaderCircle,
	Pause,
	Play,
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
import { useMediaPlayer } from "./player";

/** The height the bar takes, which the page keeps free at its end. */
export const miniPlayerSpace = "h-20";

const controlButton =
	"inline-flex size-10 shrink-0 items-center justify-center rounded-full text-ink hover:bg-paper";

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

	return (
		<>
			<div aria-hidden className={`${miniPlayerSpace} shrink-0`} />
			<section
				aria-label={t("player.label")}
				className="fixed inset-x-0 bottom-0 z-30 border-t border-line bg-surface/95 shadow-[0_-8px_24px_-16px_var(--color-shadow)] backdrop-blur"
			>
				<div className="mx-auto flex w-full max-w-screen-2xl items-center gap-3 px-4 py-3 sm:px-6">
					<span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-line bg-paper">
						<ChosenIcon
							{...iconProps(current.icon)}
							fallback={Radio}
							className="size-6"
						/>
					</span>
					<div className="min-w-0 flex-1">
						<p className="truncate font-medium">{current.title}</p>
						{status === "error" ? (
							<p role="alert" className="truncate text-sm text-danger">
								{t("player.error")}
							</p>
						) : (
							<p className="truncate text-sm text-muted" aria-live="polite">
								{busy
									? t("player.loading")
									: status === "paused"
										? t("player.paused")
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
			</section>
		</>
	);
}
