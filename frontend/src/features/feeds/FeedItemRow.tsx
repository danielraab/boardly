import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Bookmark, Heart, Pause, Play, Rss } from "lucide-react";
import { useTranslation } from "react-i18next";
import { iconButton } from "#/components/buttonStyles";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import {
	fullDate,
	relativeTime,
} from "#/features/dashboards/widgets/relativeTime";
import {
	sizeStyles,
	type WidgetSize,
} from "#/features/dashboards/widgets/sizes";
import { useMediaPlayer } from "#/features/media/player";
import { iconProps } from "#/features/notes/noteIcon";
import {
	episodeSource,
	type FeedItem,
	type FlagPatch,
	feedIcon,
	feedItemsQueryKey,
	setFeedItemFlags,
} from "./api";

/**
 * A feed item: its feed's icon, title, feed and time. A news item opens its
 * link in a new tab, and an episode has a play button. With editing the row
 * does nothing, and showFlags adds the read later and like buttons.
 */
export function FeedItemRow({
	item,
	size = "M",
	editing = false,
	showFlags,
	showSummary,
	now = new Date(),
}: {
	item: FeedItem;
	size?: WidgetSize;
	editing?: boolean;
	showFlags: boolean;
	showSummary: boolean;
	now?: Date;
}) {
	const { t, i18n } = useTranslation();
	const style = sizeStyles[size];
	const date = new Date(item.publishedAt);
	const newsLink = !item.audio && !editing ? item.link : null;
	const titleLink = item.audio && !editing ? item.link : null;

	return (
		<li
			className={`relative flex min-w-0 items-start gap-2 rounded-md px-1 ${style.row} ${newsLink ? "hover:bg-paper" : ""}`}
		>
			<ChosenIcon
				{...iconProps(feedIcon(item.feed))}
				fallback={Rss}
				className={`${style.icon} mt-0.5 shrink-0`}
			/>
			<div className="min-w-0 flex-1">
				{newsLink ? (
					<a
						href={newsLink}
						target="_blank"
						rel="noopener noreferrer"
						title={item.title}
						className="block truncate font-medium after:absolute after:inset-0"
					>
						{item.title}
					</a>
				) : titleLink ? (
					<a
						href={titleLink}
						target="_blank"
						rel="noopener noreferrer"
						title={item.title}
						className="block truncate font-medium hover:underline"
					>
						{item.title}
					</a>
				) : (
					<p title={item.title} className="truncate font-medium">
						{item.title}
					</p>
				)}
				<p className="flex min-w-0 gap-2 text-[0.85em] text-muted">
					<span className="min-w-0 truncate">{item.feed.title}</span>
					<time
						dateTime={item.publishedAt}
						title={fullDate(date, i18n.language)}
						className="shrink-0"
					>
						{relativeTime(date, now, i18n.language)}
					</time>
				</p>
				{showSummary && item.summary && (
					<p className="line-clamp-2 text-[0.9em] text-muted">{item.summary}</p>
				)}
			</div>
			{item.audio && (
				<EpisodeButton
					item={item as FeedItem & { audio: NonNullable<FeedItem["audio"]> }}
					editing={editing}
					iconClass={style.icon}
				/>
			)}
			{showFlags && !editing && (
				<FlagButtons
					item={item}
					label={t("feeds.flagsOf", { name: item.title })}
				/>
			)}
		</li>
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
