import { Menu } from "@base-ui/react/menu";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
	Bookmark,
	Check,
	EllipsisVertical,
	Heart,
	Info,
	Pause,
	Play,
	RotateCcw,
	Rss,
	Trash2,
} from "lucide-react";
import {
	type ReactNode,
	useId,
	useLayoutEffect,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import {
	fullDate,
	relativeTime,
} from "#/features/dashboards/widgets/relativeTime";
import {
	sizeStyles,
	type WidgetSize,
} from "#/features/dashboards/widgets/sizes";
import { useMediaPlayer, usePlayerProgress } from "#/features/media/player";
import { iconProps } from "#/features/notes/noteIcon";
import {
	deleteListen,
	episodeSource,
	type FeedItem,
	type FlagPatch,
	feedIcon,
	feedItemsQueryKey,
	saveListen,
	setFeedItemFlags,
} from "./api";
import { inOrder, showListen } from "./episodeProgress";
import { observeResize } from "./observeResize";

/**
 * A feed item: its feed's icon, title, feed and time. A news item opens its
 * link in a new tab, and an episode has a play button. With editing the row
 * does nothing, showFlags adds the read later and like buttons, and
 * showEpisodeMenu the listen history's actions for an episode. With summary
 * "preview" the summary is shown cut off after two lines; with "onDemand" an
 * info button shows the whole summary.
 *
 * The row is a container: narrower than 480 px it puts the feed and time
 * first, the title below over the full width, and the buttons under the text.
 * The DOM order (icon, title, meta, summary, actions) is the same in both.
 */
export function FeedItemRow({
	item,
	size = "M",
	editing = false,
	showFlags,
	summary,
	showEpisodeMenu = false,
	now = new Date(),
}: {
	item: FeedItem;
	size?: WidgetSize;
	editing?: boolean;
	showFlags: boolean;
	summary: "preview" | "onDemand";
	showEpisodeMenu?: boolean;
	now?: Date;
}) {
	const { t, i18n } = useTranslation();
	const style = sizeStyles[size];
	const date = new Date(item.publishedAt);
	const newsLink = !item.audio && !editing ? item.link : null;
	const titleLink = item.audio && !editing ? item.link : null;
	const flags = showFlags && !editing;
	const summaryId = useId();
	const [summaryOpen, setSummaryOpen] = useState(false);
	const summaryButton = summary === "onDemand" && !editing && !!item.summary;
	// Two lines when narrow, one truncated line when wide.
	const titleClass = `${rowTitle} font-medium`;

	return (
		// The row is relative so that the news link covers it, and its bottom
		// edge is left free for a progress bar.
		<li
			className={`@container relative min-w-0 rounded-md px-1 ${style.row} ${newsLink ? "hover:bg-paper" : ""}`}
		>
			<div className={rowGrid}>
				<ChosenIcon
					{...iconProps(feedIcon(item.feed))}
					fallback={Rss}
					className={`${style.icon} shrink-0 self-center [grid-area:icon] @min-[480px]:mt-0.5 @min-[480px]:self-start`}
				/>
				{newsLink ? (
					<a
						href={newsLink}
						target="_blank"
						rel="noopener noreferrer"
						title={item.title}
						className={`${titleClass} after:absolute after:inset-0`}
					>
						{item.title}
					</a>
				) : titleLink ? (
					<a
						href={titleLink}
						target="_blank"
						rel="noopener noreferrer"
						title={item.title}
						className={`${titleClass} hover:underline`}
					>
						{item.title}
					</a>
				) : (
					<p title={item.title} className={titleClass}>
						{item.title}
					</p>
				)}
				<p className="flex min-w-0 gap-2 text-[0.85em] text-muted [grid-area:meta]">
					<span className="min-w-0 truncate">{item.feed.title}</span>
					<time
						dateTime={item.publishedAt}
						title={fullDate(date, i18n.language)}
						className="shrink-0"
					>
						{relativeTime(date, now, i18n.language)}
					</time>
					<ListenProgress item={item} render={progressText} />
				</p>
				{summary === "preview" && item.summary && (
					<Summary text={item.summary} editing={editing} />
				)}
				{summaryButton && summaryOpen && (
					<p
						id={summaryId}
						className="min-w-0 break-words text-[0.9em] text-muted [grid-area:summary]"
					>
						{item.summary}
					</p>
				)}
				{(item.audio || flags || summaryButton) && (
					<div className="flex items-center gap-2 justify-self-end [grid-area:actions] @min-[480px]:self-start">
						{summaryButton && (
							<button
								type="button"
								aria-label={t(
									summaryOpen ? "feeds.hideSummary" : "feeds.showSummary",
								)}
								aria-expanded={summaryOpen}
								aria-controls={summaryOpen ? summaryId : undefined}
								onClick={() => setSummaryOpen(!summaryOpen)}
								className={`${iconButton} relative z-10`}
							>
								<Info className="size-4" aria-hidden />
							</button>
						)}
						{item.audio && (
							<EpisodeButton
								item={
									item as FeedItem & { audio: NonNullable<FeedItem["audio"]> }
								}
								editing={editing}
								iconClass={style.icon}
							/>
						)}
						{item.audio && showEpisodeMenu && !editing && (
							<EpisodeMenu item={item} />
						)}
						{flags && (
							<FlagButtons
								item={item}
								label={t("feeds.flagsOf", { name: item.title })}
							/>
						)}
					</div>
				)}
			</div>
			<ListenProgress item={item} render={progressBar} />
		</li>
	);
}

/** Where an episode is, as far as a row shows it. */
type Progress = { fraction: number; minutesLeft: number; finished: boolean };

/**
 * The progress of an episode with a listen record: from the player while it
 * has the episode loaded, else from the record. Only the loaded episode's
 * row follows the player, so other rows don't re-render while it plays.
 */
function ListenProgress({
	item,
	render,
}: {
	item: FeedItem;
	render: (progress: Progress) => ReactNode;
}) {
	const player = useMediaPlayer();
	const listen = item.listen;
	if (!item.audio || !listen) return null;
	const duration = listen.durationSeconds ?? item.audio.durationSeconds ?? 0;
	if (player.current?.id === item.id)
		return (
			<LiveProgress
				duration={duration}
				finished={player.status === "finished"}
				render={render}
			/>
		);
	return progressOf(
		listen.positionSeconds,
		duration,
		listen.finishedAt !== null,
		render,
	);
}

function LiveProgress({
	duration,
	finished,
	render,
}: {
	duration: number;
	finished: boolean;
	render: (progress: Progress) => ReactNode;
}) {
	const live = usePlayerProgress();
	return progressOf(live.position, live.duration || duration, finished, render);
}

function progressOf(
	position: number,
	duration: number,
	finished: boolean,
	render: (progress: Progress) => ReactNode,
) {
	if (finished) return render({ fraction: 1, minutesLeft: 0, finished });
	// Without a duration, an episode in progress shows nothing.
	if (!(duration > 0)) return null;
	const fraction = Math.min(1, Math.max(0, position / duration));
	const minutesLeft = Math.ceil(Math.max(0, duration - position) / 60);
	return render({ fraction, minutesLeft, finished });
}

/** The bar along the row's bottom edge; the text says the same. */
function progressBar({ fraction }: Progress) {
	return (
		<span
			aria-hidden
			data-testid="listen-progress"
			className="pointer-events-none absolute inset-x-1 bottom-0 h-[3px] overflow-hidden rounded-full bg-line"
		>
			<span
				className="block h-full bg-ink"
				style={{ width: `${fraction * 100}%` }}
			/>
		</span>
	);
}

function progressText({ fraction, minutesLeft, finished }: Progress) {
	return (
		<ProgressText
			finished={finished}
			started={fraction > 0}
			minutesLeft={minutesLeft}
		/>
	);
}

function ProgressText({
	finished,
	started,
	minutesLeft,
}: {
	finished: boolean;
	started: boolean;
	minutesLeft: number;
}) {
	const { t } = useTranslation();
	if (!finished && !started) return null;
	return (
		<span className="shrink-0">
			{finished
				? t("feeds.played")
				: t("feeds.minutesLeft", { count: minutesLeft })}
		</span>
	);
}

/** The grid of a row, narrow by default and wide from 480 px. */
const rowGrid =
	"grid grid-cols-[auto_minmax(0,1fr)] gap-x-2 [grid-template-areas:'icon_meta'_'title_title'_'summary_summary'_'actions_actions'] @min-[480px]:grid-cols-[auto_minmax(0,1fr)_auto] @min-[480px]:[grid-template-areas:'icon_title_actions'_'icon_meta_actions'_'icon_summary_actions']";

/** The title: up to two lines when narrow, one truncated line when wide. */
const rowTitle =
	"min-w-0 [grid-area:title] line-clamp-2 break-words @min-[480px]:block @min-[480px]:truncate";

/**
 * The summary, cut off after two lines. When it doesn't fit, a toggle shows
 * the rest; it sits above the news link, so it never opens the link.
 */
function Summary({ text, editing }: { text: string; editing: boolean }) {
	const { t } = useTranslation();
	const id = useId();
	const ref = useRef<HTMLParagraphElement>(null);
	const [expanded, setExpanded] = useState(false);
	const [cut, setCut] = useState(false);

	useLayoutEffect(() => {
		const el = ref.current;
		if (!el || expanded) return;
		// Fractional line heights can make scrollHeight a pixel larger
		// than clientHeight for a text that fits.
		const measure = () => setCut(el.scrollHeight > el.clientHeight + 1);
		measure();
		return observeResize(el, measure);
	}, [expanded]);

	return (
		<div className="min-w-0 text-[0.9em] text-muted [grid-area:summary]">
			<p
				id={id}
				ref={ref}
				className={`break-words ${expanded ? "" : "line-clamp-2"}`}
			>
				{text}
			</p>
			{!editing && (cut || expanded) && (
				<button
					type="button"
					aria-expanded={expanded}
					aria-controls={id}
					onClick={() => setExpanded(!expanded)}
					className="relative z-10 text-ink underline-offset-2 hover:underline"
				>
					{t(expanded ? "feeds.showLess" : "feeds.showMore")}
				</button>
			)}
		</div>
	);
}

/**
 * Marks an episode as played or unplayed, or removes it from the history.
 * The loaded episode is stopped first, so that the player's last save comes
 * before the action and doesn't undo it.
 */
function EpisodeMenu({ item }: { item: FeedItem }) {
	const { t } = useTranslation();
	const player = useMediaPlayer();
	const queryClient = useQueryClient();
	const listen = item.listen ?? null;
	const finished = listen?.finishedAt != null;
	const started = listen !== null && listen.positionSeconds > 0;
	const action = useMutation({
		mutationFn: async (kind: "played" | "unplayed" | "remove") => {
			if (player.current?.id === item.id) player.stop();
			return inOrder(item.id, async () => {
				if (kind === "remove") {
					await deleteListen(item.id);
					return null;
				}
				const saved = await saveListen(item.id, {
					positionSeconds: 0,
					...(kind === "played" ? { finished: true } : {}),
				});
				return saved.listen ?? null;
			});
		},
		onSuccess: (next) =>
			showListen(queryClient, item.id, next, { refetch: true }),
	});

	return (
		<Menu.Root>
			<Menu.Trigger
				aria-label={t("feeds.episodeActions", { name: item.title })}
				className={`${iconButton} relative z-10`}
			>
				<EllipsisVertical className="size-4" aria-hidden />
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={4} className="z-50">
					<Menu.Popup className={menuPopup}>
						{!finished && (
							<Menu.Item
								className={menuItem}
								onClick={() => action.mutate("played")}
							>
								<Check className="size-4 text-muted" aria-hidden />
								{t("feeds.markPlayed")}
							</Menu.Item>
						)}
						{(finished || started) && (
							<Menu.Item
								className={menuItem}
								onClick={() => action.mutate("unplayed")}
							>
								<RotateCcw className="size-4 text-muted" aria-hidden />
								{t("feeds.markUnplayed")}
							</Menu.Item>
						)}
						{listen && (
							<Menu.Item
								className={menuItem}
								onClick={() => action.mutate("remove")}
							>
								<Trash2 className="size-4 text-muted" aria-hidden />
								{t("feeds.removeFromHistory")}
							</Menu.Item>
						)}
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}

function EpisodeButton({
	item,
	editing,
	iconClass,
}: {
	item: FeedItem & { audio: NonNullable<FeedItem["audio"]> };
	editing: boolean;
	iconClass: string;
}) {
	const { t } = useTranslation();
	const player = useMediaPlayer();
	const current = player.current?.id === item.id;
	const playing =
		current && (player.status === "playing" || player.status === "loading");

	return (
		<button
			type="button"
			disabled={editing}
			onClick={() => {
				if (current && player.status !== "error") player.toggle();
				else player.play(episodeSource(item));
			}}
			aria-label={t(playing ? "media.pauseNamed" : "media.playNamed", {
				name: item.title,
			})}
			aria-pressed={playing}
			data-state={current ? (playing ? "playing" : "paused") : undefined}
			className="relative z-10 inline-flex size-8 shrink-0 items-center justify-center rounded-full border border-line text-ink hover:bg-paper aria-pressed:bg-ink aria-pressed:text-paper"
		>
			{playing ? (
				<Pause className={iconClass} aria-hidden />
			) : (
				<Play className={iconClass} aria-hidden />
			)}
		</button>
	);
}

function FlagButtons({ item, label }: { item: FeedItem; label: string }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	// Waiting for the refetch keeps the optimistic state until the new data.
	const flags = useMutation({
		mutationFn: (patch: FlagPatch) => setFeedItemFlags(item.id, patch),
		onSettled: () =>
			Promise.all([
				queryClient.invalidateQueries({ queryKey: feedItemsQueryKey }),
				queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey }),
			]),
	});
	const pending = flags.isPending ? flags.variables : undefined;
	const readLater = pending?.readLater ?? Boolean(item.readLaterAt);
	const liked = pending?.liked ?? Boolean(item.likedAt);

	return (
		// biome-ignore lint/a11y/useSemanticElements: a group of two toggles, not a form
		<div role="group" aria-label={label} className="relative z-10 flex gap-0.5">
			<button
				type="button"
				onClick={() => flags.mutate({ readLater: !readLater })}
				aria-label={t(readLater ? "feeds.removeReadLater" : "feeds.readLater")}
				aria-pressed={readLater}
				className={iconButton}
			>
				<Bookmark
					className={`size-4 ${readLater ? "fill-current" : ""}`}
					aria-hidden
				/>
			</button>
			<button
				type="button"
				onClick={() => flags.mutate({ liked: !liked })}
				aria-label={t(liked ? "feeds.removeLike" : "feeds.like")}
				aria-pressed={liked}
				className={iconButton}
			>
				<Heart
					className={`size-4 ${liked ? "fill-current" : ""}`}
					aria-hidden
				/>
			</button>
		</div>
	);
}
