import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FolderTree } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { FavouriteDetailsDialog } from "#/features/favourites/FavouriteDetails";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { NodeDialog } from "#/features/favourites/NodeDialog";
import { findNode, label } from "#/features/favourites/tree";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { type ExplorerData, ExplorerTree } from "./ExplorerTree";

export type ExplorerConfig = {
	rootFolderId: string;
	/** Shown as a heading above the tree; missing in configs saved before titles. */
	title?: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

/**
 * In view mode the server's data; in edit mode the owner's current favourites,
 * since a draft widget has no data yet and its folder may have changed. In
 * view mode the owner can add a link, to the explorer's folder by default, and
 * open the details of the links.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const config = widget.config as ExplorerConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [adding, setAdding] = useState(false);
	const [detailsId, setDetailsId] = useState<string | null>(null);
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: editing || isOwner,
	});
	const folder = favourites.data
		? findNode(favourites.data, config.rootFolderId)
		: undefined;
	let data = widget.data as ExplorerData | undefined;
	if (editing) {
		if (!favourites.data) return null;
		data =
			folder?.kind === "folder"
				? { children: folder.children ?? [] }
				: { missing: true };
	}
	if (!data) return null;
	const canAdd =
		isOwner && !editing && !("missing" in data) && folder?.kind === "folder";
	const canShowDetails = isOwner && !editing && !!favourites.data;
	const refreshDashboard = () =>
		void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
	return (
		<>
			<ExplorerTree
				widgetId={widget.id}
				size={config.size}
				data={data}
				editing={editing}
				title={config.title ?? null}
				headingLevel={headingLevel}
				onAdd={canAdd ? () => setAdding(true) : undefined}
				onDetails={canShowDetails ? setDetailsId : undefined}
				titleLink={
					canAdd
						? folder.id === favourites.data?.id
							? {}
							: { folder: folder.id }
						: undefined
				}
			/>
			{canShowDetails && (
				<FavouriteDetailsDialog
					favouriteId={detailsId}
					onSaved={refreshDashboard}
					onClose={() => setDetailsId(null)}
				/>
			)}
			{canAdd && (
				<NodeDialog
					target={
						adding
							? {
									kind: "link",
									parent: folder,
									parentLabel: label(folder) || t("favourites.title"),
								}
							: null
					}
					chooseFolder
					onSaved={refreshDashboard}
					onClose={() => setAdding(false)}
				/>
			)}
		</>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as ExplorerConfig;
	const favourites = useQuery(favouritesQueryOptions);
	const [folderId, setFolderId] = useState(config.rootFolderId);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const [title, setTitle] = useState(config.title ?? "");
	// A deleted folder is no choice; another one has to be picked.
	const chosen =
		favourites.data && findNode(favourites.data, folderId)?.kind === "folder"
			? folderId
			: "";

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.favouriteExplorer.addTitle"
					: "dashboards.widgets.favouriteExplorer.editTitle",
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
						rootFolderId: chosen,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies ExplorerConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label htmlFor="explorer-title" className="block text-sm font-medium">
					{t("dashboards.widgets.favouriteExplorer.titleLabel")}
				</label>
				<input
					id="explorer-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="explorer-title-hint"
					className={textInput}
				/>
				<p id="explorer-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.favouriteExplorer.titleHint")}
				</p>
			</div>
			<div>
				<FavouritePicker
					id="explorer-folder"
					label={t("dashboards.widgets.favouriteExplorer.folderLabel")}
					mode="folder"
					value={chosen}
					onChange={setFolderId}
				/>
				{config.rootFolderId !== "" && favourites.data && chosen === "" && (
					<p className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.favouriteExplorer.missing")}
					</p>
				)}
			</div>
			<div>
				<label htmlFor="explorer-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="explorer-size"
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
					id="explorer-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="explorer-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="explorer-public" className="text-sm font-medium">
						{t("dashboards.widgets.favouriteExplorer.publicLabel")}
					</label>
					<p id="explorer-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.favouriteExplorer.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const favouriteExplorerKind: WidgetKind = {
	kind: "favourite_explorer",
	icon: FolderTree,
	labelKey: "dashboards.widgets.favouriteExplorer.name",
	descriptionKey: "dashboards.widgets.favouriteExplorer.description",
	defaultConfig: (): ExplorerConfig => ({
		rootFolderId: "",
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
