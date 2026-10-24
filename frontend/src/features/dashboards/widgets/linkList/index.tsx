import { useQuery, useQueryClient } from "@tanstack/react-query";
import { List } from "lucide-react";
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
import { flattenLinks, type LinkListData } from "./flatten";
import {
	type LinkListLayout,
	LinkListView,
	linkListLayouts,
} from "./LinkListView";

export type LinkListConfig = {
	folderId: string;
	title: string | null;
	includeSubfolders: boolean;
	layout: LinkListLayout;
	size: WidgetSize;
	showIfPublic: boolean;
};

/**
 * In view mode the server's data; in edit mode the links of the owner's
 * current favourites, since a draft widget has no data yet and its folder or
 * settings may have changed. In view mode the owner can open the details of
 * the links.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const queryClient = useQueryClient();
	const config = widget.config as LinkListConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [detailsId, setDetailsId] = useState<string | null>(null);
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: editing || isOwner,
	});
	let data = widget.data as LinkListData | undefined;
	if (editing) {
		if (!favourites.data) return null;
		const folder = findNode(favourites.data, config.folderId);
		data =
			folder?.kind === "folder"
				? {
						links: flattenLinks(
							folder.children ?? [],
							config.includeSubfolders,
						),
					}
				: { missing: true };
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
	const [folderId, setFolderId] = useState(config.folderId);
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
		layout: "list",
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
