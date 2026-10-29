import { useQuery } from "@tanstack/react-query";
import { ListChecks } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import { useOpenNote } from "#/features/dashboards/noteParam";
import {
	noteCategoriesQueryOptions,
	notesQueryOptions,
} from "#/features/notes/api";
import type { NoteListData } from "../noteList/list";
import { NoteListView } from "../noteList/NoteListView";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { ChosenNotes } from "./ChosenNotes";
import { NoteSearch } from "./NoteSearch";
import { selectedNotes } from "./selected";

export type SelectedNotesConfig = {
	noteIds: string[];
	title: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

/**
 * In view mode the server's data, where the owner opens a note's editor in a
 * modal; in edit mode the owner's current notes, since a draft widget has no
 * data yet and its settings may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const config = widget.config as SelectedNotesConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const openNote = useOpenNote();
	const notes = useQuery({ ...notesQueryOptions, enabled: editing });
	const categories = useQuery({
		...noteCategoriesQueryOptions,
		enabled: editing,
	});

	let data = widget.data as NoteListData | undefined;
	if (editing) {
		if (!notes.data || !categories.data) return null;
		data = selectedNotes(notes.data, categories.data, config);
	}
	if (!data) return null;
	const canOpen = isOwner && !editing;
	// Selected notes are never empty: without notes they're missing.
	return (
		<NoteListView
			data={data}
			category="all"
			size={config.size}
			title={config.title}
			headingLevel={headingLevel}
			missingText={t("dashboards.widgets.selectedNotes.missing")}
			onOpen={
				canOpen
					? (note) => {
							if (note.id) openNote(note.id);
						}
					: undefined
			}
		/>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as SelectedNotesConfig;
	const [noteIds, setNoteIds] = useState(config.noteIds);
	const [title, setTitle] = useState(config.title ?? "");
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const notes = useQuery(notesQueryOptions);
	const categories = useQuery(noteCategoriesQueryOptions);

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.selectedNotes.addTitle"
					: "dashboards.widgets.selectedNotes.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={noteIds.length > 0}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						noteIds,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies SelectedNotesConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div className="flex flex-col gap-2">
				<label
					htmlFor="selected-notes-search"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.selectedNotes.notesLabel")}
				</label>
				{notes.data && categories.data && (
					<>
						<ChosenNotes
							noteIds={noteIds}
							notes={notes.data}
							categories={categories.data}
							onChange={setNoteIds}
						/>
						<NoteSearch
							id="selected-notes-search"
							notes={notes.data}
							categories={categories.data}
							chosen={noteIds}
							onAdd={(id) => setNoteIds((ids) => [...ids, id])}
						/>
					</>
				)}
			</div>
			<div>
				<label
					htmlFor="selected-notes-title"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.selectedNotes.titleLabel")}
				</label>
				<input
					id="selected-notes-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="selected-notes-title-hint"
					className={textInput}
				/>
				<p id="selected-notes-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.selectedNotes.titleHint")}
				</p>
			</div>
			<div>
				<label
					htmlFor="selected-notes-size"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="selected-notes-size"
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
					id="selected-notes-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="selected-notes-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label
						htmlFor="selected-notes-public"
						className="text-sm font-medium"
					>
						{t("dashboards.widgets.selectedNotes.publicLabel")}
					</label>
					<p id="selected-notes-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.selectedNotes.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const selectedNotesKind: WidgetKind = {
	kind: "selected_notes",
	icon: ListChecks,
	group: "notes",
	labelKey: "dashboards.widgets.selectedNotes.name",
	descriptionKey: "dashboards.widgets.selectedNotes.description",
	defaultConfig: (): SelectedNotesConfig => ({
		noteIds: [],
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
