import { useQuery, useQueryClient } from "@tanstack/react-query";
import { List } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import {
	favouritesQueryOptions,
	listTags,
	tagUsageQueryKey,
} from "#/features/favourites/api";
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
import {
	flattenLinks,
	type LinkListData,
	type LinkListSort,
	linkListSorts,
} from "./flatten";
import {
	type LinkListLayout,
	LinkListView,
	linkListLayouts,
} from "./LinkListView";

export type LinkListConfig = {
	folderId: string;
	title: string | null;
	includeSubfolders: boolean;
	/** Absent in configs saved before tags and sorting: no tag, favourites order. */
	tagId?: string | null;
	sort?: LinkListSort;
	layout: LinkListLayout;
	size: WidgetSize;
	showIfPublic: boolean;
};

/** The owner's tags with their IDs, to choose a link list's tag. */
const tagsQueryOptions = { queryKey: tagUsageQueryKey, queryFn: listTags };

/**
 * In view mode the server's data; in edit mode the links of the owner's
 * current favourites, since a draft widget has no data yet and its folder or
 * settings may have changed. In view mode the owner can open the details of
 * the links.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const queryClient = useQueryClient();
	const config = widget.config as LinkListConfig;
	const tagId = config.tagId ?? null;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [detailsId, setDetailsId] = useState<string | null>(null);
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: editing || isOwner,
	});
	// The tree has tag names only, so the preview looks up the tag's name.
	const tags = useQuery({
		...tagsQueryOptions,
		enabled: editing && tagId !== null,
	});
	let data = widget.data as LinkListData | undefined;
	if (editing) {
		if (!favourites.data || (tagId !== null && !tags.data)) return null;
		const folder = findNode(favourites.data, config.folderId);
		const tag = tags.data?.find((t) => t.id === tagId);
		if (folder?.kind !== "folder") data = { missing: true };
		else if (tagId !== null && !tag) data = { missing: "tag" };
		else
			data = {
				links: flattenLinks(
					folder.children ?? [],
					config.includeSubfolders,
					tag?.name ?? null,
					config.sort,
				),
			};
	}
	if (!data) return null;
	const canShowDetails = isOwner && !editing && !!favourites.data;
	return (
		<>
			<LinkListView
				data={data}
				layout={config.layout}
				size={config.size}
				editing={editing}
				title={config.title}
				tagFilter={tagId !== null}
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

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as LinkListConfig;
	const favourites = useQuery(favouritesQueryOptions);
	const tags = useQuery(tagsQueryOptions);
	const [folderId, setFolderId] = useState(config.folderId);
	const [tagId, setTagId] = useState(config.tagId ?? "");
	const [sort, setSort] = useState<LinkListSort>(config.sort ?? "favourites");
	const [title, setTitle] = useState(config.title ?? "");
	const [includeSubfolders, setIncludeSubfolders] = useState(
		config.includeSubfolders,
	);
	const [layout, setLayout] = useState(config.layout);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	// A deleted folder is no choice; another one has to be picked.
	const chosen =
		favourites.data && findNode(favourites.data, folderId)?.kind === "folder"
			? folderId
			: "";
	// Likewise, a deleted tag falls back to no tag filter.
	const chosenTag = tags.data?.some((t) => t.id === tagId) ? tagId : "";

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.linkList.addTitle"
					: "dashboards.widgets.linkList.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={chosen !== ""}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						folderId: chosen,
						title: title.trim() || null,
						includeSubfolders,
						tagId: chosenTag || null,
						sort,
						layout,
						size,
						showIfPublic,
					} satisfies LinkListConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label htmlFor="link-list-title" className="block text-sm font-medium">
					{t("dashboards.widgets.linkList.titleLabel")}
				</label>
				<input
					id="link-list-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="link-list-title-hint"
					className={textInput}
				/>
				<p id="link-list-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.linkList.titleHint")}
				</p>
			</div>
			<div>
				<FavouritePicker
					id="link-list-folder"
					label={t("dashboards.widgets.linkList.folderLabel")}
					mode="folder"
					value={chosen}
					onChange={setFolderId}
				/>
				{config.folderId !== "" && favourites.data && chosen === "" && (
					<p className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.linkList.missing")}
					</p>
				)}
			</div>
			<div className="flex items-center gap-3">
				<input
					id="link-list-subfolders"
					type="checkbox"
					checked={includeSubfolders}
					onChange={(e) => setIncludeSubfolders(e.target.checked)}
					className="size-4 accent-ink"
				/>
				<label htmlFor="link-list-subfolders" className="text-sm font-medium">
					{t("dashboards.widgets.linkList.subfoldersLabel")}
				</label>
			</div>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label htmlFor="link-list-tag" className="block text-sm font-medium">
						{t("dashboards.widgets.linkList.tagLabel")}
					</label>
					<select
						id="link-list-tag"
						value={chosenTag}
						onChange={(e) => setTagId(e.target.value)}
						className={textInput}
					>
						<option value="">{t("dashboards.widgets.linkList.noTag")}</option>
						{tags.data?.map((tag) => (
							<option key={tag.id} value={tag.id}>
								{tag.name}
							</option>
						))}
					</select>
					{config.tagId && tags.data && chosenTag === "" && (
						<p className="mt-1 text-sm text-muted">
							{t("dashboards.widgets.linkList.missingTag")}
						</p>
					)}
				</div>
				<div>
					<label htmlFor="link-list-sort" className="block text-sm font-medium">
						{t("dashboards.widgets.linkList.sortLabel")}
					</label>
					<select
						id="link-list-sort"
						value={sort}
						onChange={(e) => setSort(e.target.value as LinkListSort)}
						className={textInput}
					>
						{linkListSorts.map((s) => (
							<option key={s} value={s}>
								{t(`dashboards.widgets.linkList.sort.${s}`)}
							</option>
						))}
					</select>
				</div>
			</div>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label
						htmlFor="link-list-layout"
						className="block text-sm font-medium"
					>
						{t("dashboards.widgets.linkList.layoutLabel")}
					</label>
					<select
						id="link-list-layout"
						value={layout}
						onChange={(e) => setLayout(e.target.value as LinkListLayout)}
						className={textInput}
					>
						{linkListLayouts.map((l) => (
							<option key={l} value={l}>
								{t(`dashboards.widgets.linkList.layout.${l}`)}
							</option>
						))}
					</select>
				</div>
				<div>
					<label htmlFor="link-list-size" className="block text-sm font-medium">
						{t("dashboards.widgets.sizeLabel")}
					</label>
					<select
						id="link-list-size"
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
			</div>
			<div className="flex items-start gap-3">
				<input
					id="link-list-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="link-list-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="link-list-public" className="text-sm font-medium">
						{t("dashboards.widgets.linkList.publicLabel")}
					</label>
					<p id="link-list-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.linkList.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const linkListKind: WidgetKind = {
	kind: "link_list",
	icon: List,
	labelKey: "dashboards.widgets.linkList.name",
	descriptionKey: "dashboards.widgets.linkList.description",
	defaultConfig: (): LinkListConfig => ({
		folderId: "",
		title: null,
		includeSubfolders: false,
		tagId: null,
		sort: "favourites",
		layout: "list",
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
