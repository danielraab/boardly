import type { ParseKeys, TFunction } from "i18next";
import type { LucideIcon } from "lucide-react";
import type { ComponentType } from "react";
import { type ContentWidget, isSection, type Widget } from "../api";
import { favouriteExplorerKind } from "./favouriteExplorer";
import { linkKind } from "./link";

export type WidgetViewProps = {
	widget: ContentWidget;
	/** In edit mode the widget shows the draft and must not navigate away. */
	editing: boolean;
};

export type WidgetSettingsProps = {
	/** The widget to edit; a new one comes with its kind's defaults. */
	widget: ContentWidget;
	isNew: boolean;
	desktopColumns: number;
	onApply: (widget: ContentWidget) => void;
	onDelete: () => void;
};

/** A content widget kind, as the web app knows it. */
export type WidgetKind = {
	kind: string;
	icon: LucideIcon;
	/** Translation keys of the kind's name and a short description. */
	labelKey: ParseKeys;
	descriptionKey: ParseKeys;
	defaultConfig: () => unknown;
	View: ComponentType<WidgetViewProps>;
	/** The contents of the settings dialog, usually a WidgetSettingsForm. */
	Settings: ComponentType<WidgetSettingsProps>;
};

/** Every kind that can be added, in the order the picker lists them. */
export const widgetKinds: WidgetKind[] = [favouriteExplorerKind, linkKind];

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
