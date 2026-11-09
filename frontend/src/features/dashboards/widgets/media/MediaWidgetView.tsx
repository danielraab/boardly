import { AudioLines, Pause, Play } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import type { PlayableSource } from "#/features/media/api";
import { useOpenEmbed } from "#/features/media/embedNotice";
import { useMediaPlayer } from "#/features/media/player";
import { sourceFallbackIcon } from "#/features/media/sourceIcon";
import { iconProps } from "#/features/notes/noteIcon";
import { sizeStyles, type WidgetSize } from "../sizes";

export type MediaLayout = "list" | "tiles";

export const mediaLayouts: MediaLayout[] = ["list", "tiles"];

/** Icon size, minimum tile width and label size of the tiles of each size. */
const tileStyles: Record<
	WidgetSize,
	{ icon: string; grid: string; text: string }
> = {
	S: {
		icon: "size-8",
		grid: "grid-cols-[repeat(auto-fill,minmax(5rem,1fr))]",
		text: "text-xs",
	},
	M: {
		icon: "size-12",
		grid: "grid-cols-[repeat(auto-fill,minmax(7rem,1fr))]",
		text: "text-sm",
	},
	L: {
		icon: "size-16",
		grid: "grid-cols-[repeat(auto-fill,minmax(9rem,1fr))]",
		text: "text-base",
	},
};

/**
 * The sources of a media widget as buttons below its title. Activating one
 * plays it in the app's player, or pauses it when it is playing; an embed
 * opens in the dock instead. In edit mode the buttons do nothing.
 */
export function MediaWidgetView({
	sources,
	title,
	layout,
	size,
	headingLevel = 2,
	editing,
}: {
	sources: PlayableSource[];
	title: string | null;
	layout: MediaLayout;
	size: WidgetSize;
	headingLevel?: number;
	editing: boolean;
}) {
	const { t } = useTranslation();
	const player = useMediaPlayer();
	const openEmbed = useOpenEmbed();
	const style = sizeStyles[size];
	const tile = tileStyles[size];
	const Heading = `h${headingLevel}` as "h2";

	const stateOf = (source: PlayableSource) => {
		if (player.current?.id !== source.id) return null;
		if (source.kind === "embed") return "open";
		return player.status === "playing" || player.status === "loading"
			? "playing"
			: "paused";
	};
	const activate = (source: PlayableSource) => {
		if (editing) return;
		if (source.kind === "embed") openEmbed(source);
		else if (player.current?.id === source.id && player.status !== "error")
			player.toggle();
		else player.play(source);
	};
	const label = (source: PlayableSource) => {
		if (source.kind === "embed")
			return t("media.openNamed", { name: source.title });
		return stateOf(source) === "playing"
			? t("media.pauseNamed", { name: source.title })
			: t("media.playNamed", { name: source.title });
	};

	let body: React.ReactNode;
	if (sources.length === 0) {
		body = (
			<p className="text-muted">{t("dashboards.widgets.media.missing")}</p>
		);
	} else if (layout === "tiles") {
		body = (
			<ul className={`grid gap-1 ${tile.grid}`}>
				{sources.map((source) => {
					const state = stateOf(source);
					return (
						<li key={source.id} className="min-w-0">
							<button
								type="button"
								onClick={() => activate(source)}
								aria-label={label(source)}
								aria-pressed={state === "playing" || state === "open"}
								data-state={state ?? undefined}
								className={`relative flex w-full min-w-0 flex-col items-center gap-1 rounded-lg p-2 ${tile.text} hover:bg-paper data-[state]:bg-paper`}
							>
								<span className="relative flex items-center justify-center">
									<ChosenIcon
										{...iconProps(source.icon)}
										fallback={sourceFallbackIcon(source)}
										className={tile.icon}
									/>
									{state && (
										<span className="absolute -right-1 -bottom-1 flex size-5 items-center justify-center rounded-full bg-ink text-paper">
											{state === "open" ? (
												<AudioLines className="size-3" aria-hidden />
											) : state === "playing" ? (
												<Pause className="size-3" aria-hidden />
											) : (
												<Play className="size-3" aria-hidden />
											)}
										</span>
									)}
								</span>
								<span className="w-full truncate text-center">
									{source.title}
								</span>
							</button>
						</li>
					);
				})}
			</ul>
		);
	} else {
		body = (
			<ul>
				{sources.map((source) => {
					const state = stateOf(source);
					return (
						<li key={source.id}>
							<button
								type="button"
								onClick={() => activate(source)}
								aria-label={label(source)}
								aria-pressed={state === "playing" || state === "open"}
								data-state={state ?? undefined}
								className={`flex w-full min-w-0 items-center gap-2 rounded-md px-1 ${style.row} text-left hover:bg-paper data-[state]:font-medium`}
							>
								<ChosenIcon
									{...iconProps(source.icon)}
									fallback={sourceFallbackIcon(source)}
									className={style.icon}
								/>
								<span className="min-w-0 flex-1 truncate">{source.title}</span>
								{state === "open" ? (
									<AudioLines
										className={`${style.icon} shrink-0`}
										aria-hidden
									/>
								) : state === "playing" ? (
									<Pause className={`${style.icon} shrink-0`} aria-hidden />
								) : (
									<Play
										className={`${style.icon} shrink-0 ${state ? "" : "text-muted"}`}
										aria-hidden
									/>
								)}
							</button>
						</li>
					);
				})}
			</ul>
		);
	}

	return (
		<div className={style.text}>
			{title && <Heading className="truncate font-medium">{title}</Heading>}
			{body}
		</div>
	);
}
