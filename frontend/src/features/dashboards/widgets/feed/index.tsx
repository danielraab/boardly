import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Rss } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import {
	type FeedItem,
	type FeedView,
	feedIcon,
	feedPreviewQueryOptions,
	feedsQueryOptions,
} from "#/features/feeds/api";
import { FeedItemRow } from "#/features/feeds/FeedItemRow";
import { ChosenSources, SourceSearch } from "../media/ChosenSources";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { sizeStyles, type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";

export const maxFeedWidgetItems = 100;

/** What a widget can show; in_progress is "Continue listening". */
const widgetViews = [
	"latest",
	"read_later",
	"liked",
	"in_progress",
] as const satisfies FeedView[];

export type FeedConfig = {
	/** Subscription IDs; empty means all of the owner's feeds. */
	feedIds: string[];
	show: FeedView;
	count: number;
	showSummary: boolean;
	/** Missing on widgets saved before the setting existed. */
	audioOnly?: boolean;
	title: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

/** Only the owner's data says whether the chosen feeds still exist. */
type FeedData = { items: FeedItem[]; missing?: boolean };

/**
 * In view mode the server's data; in edit mode the owner's current items,
 * since a draft widget has no data yet and its settings may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const config = widget.config as FeedConfig;
	const preview = useQuery({
		...feedPreviewQueryOptions(config.show),
		enabled: editing,
	});
	const data = widget.data as FeedData | undefined;
	let items = data?.items;
	if (editing) {
		if (!preview.data) return null;
		items = preview.data.items
			.filter(
				(it) =>
					(config.feedIds.length === 0 ||
						config.feedIds.includes(it.feed.subscriptionId ?? "")) &&
					(!config.audioOnly || it.audio),
			)
			.slice(0, config.count);
	}
	if (!items) return null;
	const style = sizeStyles[config.size];
	const Heading = `h${headingLevel}` as "h2";
	const owner = data?.missing !== undefined;

	return (
		<div className={style.text}>
			{config.title && (
				<Heading className="truncate font-medium">{config.title}</Heading>
			)}
			{data?.missing && !editing ? (
				<p className="text-muted">{t("dashboards.widgets.feed.missing")}</p>
			) : items.length === 0 ? (
				<p className="text-muted">{t("dashboards.widgets.feed.empty")}</p>
			) : (
				<ul className="flex flex-col">
					{items.map((item) => (
						<FeedItemRow
							key={item.id}
							item={item}
							size={config.size}
							editing={editing}
							showFlags={owner}
							showSummary={config.showSummary}
						/>
					))}
				</ul>
			)}
		</div>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as FeedConfig;
	const [all, setAll] = useState(config.feedIds.length === 0);
	const [feedIds, setFeedIds] = useState(config.feedIds);
	const [show, setShow] = useState(config.show);
	const [count, setCount] = useState(String(config.count));
	const [showSummary, setShowSummary] = useState(config.showSummary);
	const [audioOnly, setAudioOnly] = useState(config.audioOnly ?? false);
	const [title, setTitle] = useState(config.title ?? "");
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const library = useQuery(feedsQueryOptions);
	const choices = library.data?.map((f) => ({
		id: f.id,
		title: f.title,
		icon: feedIcon(f),
	}));
	const number = Number(count);
	const countValid =
		Number.isInteger(number) && number >= 1 && number <= maxFeedWidgetItems;

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.feed.addTitle"
					: "dashboards.widgets.feed.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={countValid && (all || feedIds.length > 0)}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						feedIds: all ? [] : feedIds,
						show,
						count: number,
						showSummary,
						audioOnly,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies FeedConfig,
				})
			}
			onDelete={props.onDelete}
		>
			{library.data?.length === 0 ? (
				<p className="text-sm text-muted">
					{t("dashboards.widgets.feed.emptyLibrary")}{" "}
					<Link to="/feeds" className="text-accent underline">
						{t("dashboards.widgets.feed.openLibrary")}
					</Link>
				</p>
			) : (
				<div className="flex flex-col gap-2">
					<div className="flex items-center gap-3">
						<input
							id="feed-all"
							type="checkbox"
							checked={all}
							onChange={(e) => setAll(e.target.checked)}
							className="size-4 accent-ink"
						/>
						<label htmlFor="feed-all" className="text-sm font-medium">
							{t("dashboards.widgets.feed.allFeeds")}
						</label>
					</div>
					{!all && choices && (
						<>
							<ChosenSources
								sourceIds={feedIds}
								sources={choices}
								onChange={setFeedIds}
								labels="dashboards.widgets.feed"
							/>
							<SourceSearch
								id="feed-search"
								sources={choices}
								chosen={feedIds}
								onAdd={(id) => setFeedIds((ids) => [...ids, id])}
								labels="dashboards.widgets.feed"
							/>
						</>
					)}
				</div>
			)}
			<div>
				<label htmlFor="feed-show" className="block text-sm font-medium">
					{t("dashboards.widgets.feed.showLabel")}
				</label>
				<select
					id="feed-show"
					value={show}
					onChange={(e) => setShow(e.target.value as FeedView)}
					className={textInput}
				>
					{widgetViews.map((v) => (
						<option key={v} value={v}>
							{v === "in_progress"
								? t("dashboards.widgets.feed.continueListening")
								: t(`feeds.views.${v}`)}
						</option>
					))}
				</select>
			</div>
			<div>
				<label htmlFor="feed-count" className="block text-sm font-medium">
					{t("dashboards.widgets.feed.countLabel")}
				</label>
				<input
					id="feed-count"
					type="number"
					min={1}
					max={maxFeedWidgetItems}
					required
					value={count}
					onChange={(e) => setCount(e.target.value)}
					className={textInput}
				/>
			</div>
			<div className="flex items-center gap-3">
				<input
					id="feed-summary"
					type="checkbox"
					checked={showSummary}
					onChange={(e) => setShowSummary(e.target.checked)}
					className="size-4 accent-ink"
				/>
				<label htmlFor="feed-summary" className="text-sm font-medium">
					{t("dashboards.widgets.feed.summaryLabel")}
				</label>
			</div>
			<div className="flex items-center gap-3">
				<input
					id="feed-audio-only"
					type="checkbox"
					checked={audioOnly}
					onChange={(e) => setAudioOnly(e.target.checked)}
					className="size-4 accent-ink"
				/>
				<label htmlFor="feed-audio-only" className="text-sm font-medium">
					{t("dashboards.widgets.feed.audioOnlyLabel")}
				</label>
			</div>
			<div>
				<label htmlFor="feed-title" className="block text-sm font-medium">
					{t("dashboards.widgets.feed.titleLabel")}
				</label>
				<input
					id="feed-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					className={textInput}
				/>
			</div>
			<div>
				<label htmlFor="feed-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="feed-size"
					value={size}
					onChange={(e) => setSize(e.target.value as WidgetSize)}
					className={textInput}
				>
					{widgetSizes.map((s) => (
						<option key={s} value={s}>
							{t(`dashboards.widgets.size.${s}`)}
						</option>
					))}
				</select>
			</div>
			<div className="flex items-start gap-3">
				<input
					id="feed-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="feed-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="feed-public" className="text-sm font-medium">
						{t("dashboards.widgets.feed.publicLabel")}
					</label>
					<p id="feed-public-hint" className="text-sm text-muted">
						{t(
							all
								? "dashboards.widgets.feed.publicHintAll"
								: "dashboards.widgets.feed.publicHint",
						)}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const feedKind: WidgetKind = {
	kind: "feed",
	icon: Rss,
	group: "feeds",
	labelKey: "dashboards.widgets.feed.name",
	descriptionKey: "dashboards.widgets.feed.description",
	defaultConfig: (): FeedConfig => ({
		feedIds: [],
		show: "latest",
		count: 10,
		showSummary: false,
		audioOnly: false,
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
