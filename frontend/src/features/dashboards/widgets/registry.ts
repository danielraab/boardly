import type { ParseKeys, TFunction } from "i18next";
import type { LucideIcon } from "lucide-react";
import type { ComponentType } from "react";
import { type ContentWidget, isSection, type Widget } from "../api";
import { favouriteExplorerKind } from "./favouriteExplorer";
import { linkKind } from "./link";
import { linkListKind } from "./linkList";
import { noteKind } from "./note";
import { noteListKind } from "./noteList";
import { recentFavouritesKind } from "./recentFavourites";
import { recentNotesKind } from "./recentNotes";

export type WidgetViewProps = {
	widget: ContentWidget;
	/** In edit mode the widget shows the draft and must not navigate away. */
	editing: boolean;
	/** The level of a heading in the widget, as a section's title beside it has. */
	headingLevel: number;
};

export type WidgetSettingsProps = {
	/** The widget to edit; a new one comes with its kind's defaults. */
	widget: ContentWidget;
	isNew: boolean;
	desktopColumns: number;
	onApply: (widget: ContentWidget) => void;
	onDelete: () => void;
};

/** The groups of the widget picker, in the order it shows them. */
export const widgetGroups = ["favourites", "notes"] as const;

export type WidgetGroup = (typeof widgetGroups)[number];

/** A content widget kind, as the web app knows it. */
export type WidgetKind = {
	kind: string;
	group: WidgetGroup;
	icon: LucideIcon;
	/** Translation keys of the kind's name and a short description. */
	labelKey: ParseKeys;
	descriptionKey: ParseKeys;
	defaultConfig: () => unknown;
	View: ComponentType<WidgetViewProps>;
	/** The contents of the settings dialog, usually a WidgetSettingsForm. */
	Settings: ComponentType<WidgetSettingsProps>;
};

/** Every kind that can be added, in the order the picker lists them within their group. */
export const widgetKinds: WidgetKind[] = [
	favouriteExplorerKind,
	linkKind,
	linkListKind,
	recentFavouritesKind,
	noteKind,
	recentNotesKind,
	noteListKind,
];

export function findKind(kind: string): WidgetKind | undefined {
	return widgetKinds.find((k) => k.kind === kind);
}

/** How a widget is named in the editor: a section's title, or its kind's name. */
export function widgetLabel(widget: Widget, t: TFunction): string {
	if (isSection(widget)) {
		return widget.title ?? t("dashboards.editor.untitled");
	}
	const kind = findKind(widget.kind);
	return kind ? t(kind.labelKey) : widget.kind;
}
