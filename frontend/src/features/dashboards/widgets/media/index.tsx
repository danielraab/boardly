import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Radio } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { mediaQueryOptions, type PlayableSource } from "#/features/media/api";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { ChosenSources, SourceSearch } from "./ChosenSources";
import {
	type MediaLayout,
	MediaWidgetView,
	mediaLayouts,
} from "./MediaWidgetView";

export type MediaConfig = {
	sourceIds: string[];
	title: string | null;
	layout: MediaLayout;
	size: WidgetSize;
	showIfPublic: boolean;
};

type MediaData = { sources: PlayableSource[] };

/**
 * In view mode the server's data; in edit mode the owner's current library,
 * since a draft widget has no data yet and its settings may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const config = widget.config as MediaConfig;
	const library = useQuery({ ...mediaQueryOptions, enabled: editing });
	let sources = (widget.data as MediaData | undefined)?.sources;
	if (editing) {
		if (!library.data) return null;
		const byID = new Map(library.data.map((s) => [s.id, s]));
		sources = config.sourceIds.flatMap((id) => {
			const s = byID.get(id);
			return s ? [s] : [];
		});
	}
	if (!sources) return null;
	return (
		<MediaWidgetView
			sources={sources}
			title={config.title}
			layout={config.layout}
			size={config.size}
			headingLevel={headingLevel}
			editing={editing}
		/>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as MediaConfig;
	const [sourceIds, setSourceIds] = useState(config.sourceIds);
	const [title, setTitle] = useState(config.title ?? "");
	const [layout, setLayout] = useState(config.layout);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const library = useQuery(mediaQueryOptions);

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.media.addTitle"
					: "dashboards.widgets.media.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={sourceIds.length > 0}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						sourceIds,
						title: title.trim() || null,
						layout,
						size,
						showIfPublic,
					} satisfies MediaConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div className="flex flex-col gap-2">
				<label htmlFor="media-search" className="block text-sm font-medium">
					{t("dashboards.widgets.media.sourcesLabel")}
				</label>
				{library.data?.length === 0 && sourceIds.length === 0 ? (
					<p className="text-sm text-muted">
						{t("dashboards.widgets.media.emptyLibrary")}{" "}
						<Link to="/media" className="text-accent underline">
							{t("dashboards.widgets.media.openLibrary")}
						</Link>
					</p>
				) : (
					library.data && (
						<>
							<ChosenSources
								sourceIds={sourceIds}
								sources={library.data}
								onChange={setSourceIds}
							/>
							<SourceSearch
								id="media-search"
								sources={library.data}
								chosen={sourceIds}
								onAdd={(id) => setSourceIds((ids) => [...ids, id])}
							/>
						</>
					)
				)}
			</div>
			<div>
				<label htmlFor="media-title" className="block text-sm font-medium">
					{t("dashboards.widgets.media.titleLabel")}
				</label>
				<input
					id="media-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="media-title-hint"
					className={textInput}
				/>
				<p id="media-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.media.titleHint")}
				</p>
			</div>
			<div>
				<label htmlFor="media-layout" className="block text-sm font-medium">
					{t("dashboards.widgets.media.layoutLabel")}
				</label>
				<select
					id="media-layout"
					value={layout}
					onChange={(e) => setLayout(e.target.value as MediaLayout)}
					className={textInput}
				>
					{mediaLayouts.map((l) => (
						<option key={l} value={l}>
							{t(`dashboards.widgets.media.layout.${l}`)}
						</option>
					))}
				</select>
			</div>
			<div>
				<label htmlFor="media-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="media-size"
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
					id="media-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="media-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="media-public" className="text-sm font-medium">
						{t("dashboards.widgets.media.publicLabel")}
					</label>
					<p id="media-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.media.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const mediaKind: WidgetKind = {
	kind: "media",
	icon: Radio,
	group: "media",
	labelKey: "dashboards.widgets.media.name",
	descriptionKey: "dashboards.widgets.media.description",
	defaultConfig: (): MediaConfig => ({
		sourceIds: [],
		title: null,
		layout: "list",
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
