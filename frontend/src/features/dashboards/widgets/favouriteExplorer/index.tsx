import { useQuery } from "@tanstack/react-query";
import { FolderTree } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { findNode } from "#/features/favourites/tree";
import { folderOptions } from "../favouriteOptions";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { WidgetSettingsForm } from "../WidgetSettingsForm";
import { type ExplorerData, ExplorerTree } from "./ExplorerTree";

export type ExplorerConfig = {
	rootFolderId: string;
	size: WidgetSize;
	showIfPublic: boolean;
};

/**
 * In view mode the server's data; in edit mode the owner's current favourites,
 * since a draft widget has no data yet and its folder may have changed.
 */
function View({ widget, editing }: WidgetViewProps) {
	const config = widget.config as ExplorerConfig;
	const favourites = useQuery({ ...favouritesQueryOptions, enabled: editing });
	let data = widget.data as ExplorerData | undefined;
	if (editing) {
		if (!favourites.data) return null;
		const folder = findNode(favourites.data, config.rootFolderId);
		data =
			folder?.kind === "folder"
				? { children: folder.children ?? [] }
				: { missing: true };
	}
	if (!data) return null;
	return (
		<ExplorerTree
			widgetId={widget.id}
			size={config.size}
			data={data}
			editing={editing}
		/>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as ExplorerConfig;
	const favourites = useQuery(favouritesQueryOptions);
	const folders = favourites.data
		? folderOptions(favourites.data, t("favourites.title")).map(
				({ node, path }) => ({ id: node.id, path }),
			)
		: [];
	const [folderId, setFolderId] = useState(config.rootFolderId);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	// A deleted folder is no choice; another one has to be picked.
	const chosen = folders.some((f) => f.id === folderId) ? folderId : "";

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
						size,
						showIfPublic,
					} satisfies ExplorerConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label htmlFor="explorer-folder" className="block text-sm font-medium">
					{t("dashboards.widgets.favouriteExplorer.folderLabel")}
				</label>
				<select
					id="explorer-folder"
					value={chosen}
					onChange={(e) => setFolderId(e.target.value)}
					disabled={!favourites.data}
					className={textInput}
				>
					<option value="" disabled>
						{t("dashboards.widgets.favouriteExplorer.folderPlaceholder")}
					</option>
					{folders.map((f) => (
						<option key={f.id} value={f.id}>
							{f.path}
						</option>
					))}
				</select>
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
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
