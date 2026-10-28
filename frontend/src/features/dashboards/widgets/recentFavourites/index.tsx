import { useQuery, useQueryClient } from "@tanstack/react-query";
import { History } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { FavouriteDetailsDialog } from "#/features/favourites/FavouriteDetails";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { findNode } from "#/features/favourites/tree";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { RecentFavouritesView } from "./RecentFavouritesView";
import {
	maxRecentDays,
	maxRecentLimit,
	type RecentData,
	recentLinks,
} from "./recent";

export type RecentFavouritesConfig = {
	folderId: string;
	limit: number;
	days: number | null;
	title: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

/**
 * In view mode the server's data; in edit mode the newest links of the
 * owner's current favourites, since a draft widget has no data yet and its
 * folder or settings may have changed. In view mode the owner can open the
 * details of the links.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const queryClient = useQueryClient();
	const config = widget.config as RecentFavouritesConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [detailsId, setDetailsId] = useState<string | null>(null);
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: editing || isOwner,
	});
	let data = widget.data as RecentData | undefined;
	if (editing) {
		if (!favourites.data) return null;
		const folder = findNode(favourites.data, config.folderId);
		data =
			folder?.kind === "folder"
				? {
						links: recentLinks(
							folder.children ?? [],
							config.limit,
							config.days,
							new Date(),
						),
					}
				: { missing: true };
	}
	if (!data) return null;
	const canShowDetails = isOwner && !editing && !!favourites.data;
	return (
		<>
			<RecentFavouritesView
				data={data}
				days={config.days}
				size={config.size}
				editing={editing}
				title={config.title}
				headingLevel={headingLevel}
				onDetails={canShowDetails ? setDetailsId : undefined}
			/>
			{canShowDetails && (
				<FavouriteDetailsDialog
					favouriteId={detailsId}
					onSaved={() =>
						void queryClient.invalidateQueries({
							queryKey: dashboardViewQueryKey,
						})
					}
					onClose={() => setDetailsId(null)}
				/>
			)}
		</>
	);
}

/** The whole number in value if it is from 1 to max, otherwise null. */
function parseCount(value: string, max: number): number | null {
	const n = Number(value);
	return value.trim() !== "" && Number.isInteger(n) && n >= 1 && n <= max
		? n
		: null;
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as RecentFavouritesConfig;
	const favourites = useQuery(favouritesQueryOptions);
	const [folderId, setFolderId] = useState(config.folderId);
	const [title, setTitle] = useState(config.title ?? "");
	const [limit, setLimit] = useState(String(config.limit));
	const [days, setDays] = useState(
		config.days === null ? "" : String(config.days),
	);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	// A deleted folder is no choice; another one has to be picked.
	const chosen =
		favourites.data && findNode(favourites.data, folderId)?.kind === "folder"
			? folderId
			: "";
	const limitValue = parseCount(limit, maxRecentLimit);
	const daysValue = days.trim() === "" ? null : parseCount(days, maxRecentDays);
	const daysValid = days.trim() === "" || daysValue !== null;

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.recentFavourites.addTitle"
					: "dashboards.widgets.recentFavourites.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={chosen !== "" && limitValue !== null && daysValid}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						folderId: chosen,
						limit: limitValue ?? config.limit,
						days: daysValue,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies RecentFavouritesConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label htmlFor="recent-title" className="block text-sm font-medium">
					{t("dashboards.widgets.recentFavourites.titleLabel")}
				</label>
				<input
					id="recent-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="recent-title-hint"
					className={textInput}
				/>
				<p id="recent-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.recentFavourites.titleHint")}
				</p>
			</div>
			<div>
				<FavouritePicker
					id="recent-folder"
					label={t("dashboards.widgets.recentFavourites.folderLabel")}
					mode="folder"
					value={chosen}
					onChange={setFolderId}
				/>
				<p className="mt-1 text-sm text-muted">
					{config.folderId !== "" && favourites.data && chosen === ""
						? t("dashboards.widgets.recentFavourites.missing")
						: t("dashboards.widgets.recentFavourites.folderHint")}
				</p>
			</div>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label htmlFor="recent-limit" className="block text-sm font-medium">
						{t("dashboards.widgets.recentFavourites.limitLabel")}
					</label>
					<input
						id="recent-limit"
						type="number"
						inputMode="numeric"
						min={1}
						max={maxRecentLimit}
						step={1}
						required
						value={limit}
						onChange={(e) => setLimit(e.target.value)}
						aria-invalid={limitValue === null || undefined}
						aria-describedby="recent-limit-hint"
						className={textInput}
					/>
					<p id="recent-limit-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.recentFavourites.limitHint", {
							max: maxRecentLimit,
						})}
					</p>
				</div>
				<div>
					<label htmlFor="recent-days" className="block text-sm font-medium">
						{t("dashboards.widgets.recentFavourites.daysLabel")}
					</label>
					<input
						id="recent-days"
						type="number"
						inputMode="numeric"
						min={1}
						max={maxRecentDays}
						step={1}
						value={days}
						onChange={(e) => setDays(e.target.value)}
						aria-invalid={!daysValid || undefined}
						aria-describedby="recent-days-hint"
						className={textInput}
					/>
					<p id="recent-days-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.recentFavourites.daysHint", {
							max: maxRecentDays,
						})}
					</p>
				</div>
			</div>
			<div>
				<label htmlFor="recent-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="recent-size"
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
					id="recent-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="recent-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="recent-public" className="text-sm font-medium">
						{t("dashboards.widgets.recentFavourites.publicLabel")}
					</label>
					<p id="recent-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.recentFavourites.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const recentFavouritesKind: WidgetKind = {
	kind: "recent_favourites",
	icon: History,
	group: "favourites",
	labelKey: "dashboards.widgets.recentFavourites.name",
	descriptionKey: "dashboards.widgets.recentFavourites.description",
	defaultConfig: (): RecentFavouritesConfig => ({
		folderId: "",
		limit: 10,
		days: null,
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
