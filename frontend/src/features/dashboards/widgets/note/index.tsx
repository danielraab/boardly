import { useQuery, useQueryClient } from "@tanstack/react-query";
import { StickyNote } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import {
	isVersionConflict,
	type Note,
	noteCategoriesQueryOptions,
	notesQueryOptions,
} from "#/features/notes/api";
import { categoryPath } from "#/features/notes/categories/tree";
import { displayTitle } from "#/features/notes/document/text";
import { setTaskChecked } from "#/features/notes/document/types";
import { useAutosave } from "#/features/notes/useAutosave";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { WidgetSettingsForm } from "../WidgetSettingsForm";
import {
	type NoteData,
	NoteWidgetView,
	type ShownNote,
} from "./NoteWidgetView";

export type NoteConfig = {
	noteId: string;
	size: WidgetSize;
	showIfPublic: boolean;
};

/** Why a toggled checkbox wasn't saved. */
type SaveError = "conflict" | "failed";

/**
 * In view mode the server's data, with toggling checkboxes for the owner; in
 * edit mode the note from the owner's notes, since a draft widget has no data
 * yet and its note may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const config = widget.config as NoteConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [error, setError] = useState<SaveError | null>(null);
	const notes = useQuery({ ...notesQueryOptions, enabled: editing });
	const categories = useQuery({
		...noteCategoriesQueryOptions,
		enabled: editing,
	});

	if (editing) {
		if (!notes.data || !categories.data) return null;
		const note = notes.data.find((n) => n.id === config.noteId);
		const category =
			note?.categoryId && categories.data.find((c) => c.id === note.categoryId);
		const data: NoteData = note
			? {
					...note,
					category: category
						? {
								...category,
								path: categoryPath(categories.data, category.id),
							}
						: null,
				}
			: { missing: true };
		return (
			<NoteWidgetView
				data={data}
				size={config.size}
				headingLevel={headingLevel}
			/>
		);
	}

	const data = widget.data as NoteData | undefined;
	if (!data) return null;
	if (!isOwner || "missing" in data || !data.id) {
		return (
			<NoteWidgetView
				data={data}
				size={config.size}
				headingLevel={headingLevel}
			/>
		);
	}
	const message =
		error &&
		t(
			error === "conflict"
				? "dashboards.widgets.note.changedElsewhere"
				: "dashboards.widgets.note.changeFailed",
		);
	return (
		// A newer version from the server starts over with a new saver.
		<OwnerNote
			key={data.version}
			data={{ ...data, id: data.id }}
			size={config.size}
			headingLevel={headingLevel}
			message={message}
			onToggled={() => setError(null)}
			onError={setError}
		/>
	);
}

/** The owner's note: checkboxes toggle and are saved right away. */
function OwnerNote({
	data,
	size,
	headingLevel,
	message,
	onToggled,
	onError,
}: {
	data: ShownNote & { id: string };
	size: WidgetSize;
	headingLevel: number;
	message: string | null;
	onToggled: () => void;
	onError: (error: SaveError) => void;
}) {
	const queryClient = useQueryClient();
	const [content, setContent] = useState(data.content);
	// The saver only uses the ID and version of the note it updates.
	const note = {
		...data,
		version: data.version ?? 0,
		categoryId: null,
		createdAt: "",
		updatedAt: "",
	} satisfies Note;
	const saver = useAutosave({
		note,
		draft: { ...note, content },
		onError: (err) => {
			if (isVersionConflict(err)) {
				onError("conflict");
				// Reloads the dashboard, which shows the note as it is now.
				void queryClient.invalidateQueries({
					queryKey: dashboardViewQueryKey,
				});
			} else {
				saver.drop();
				setContent(data.content);
				onError("failed");
			}
		},
	});
	return (
		<NoteWidgetView
			data={{ ...data, content }}
			size={size}
			headingLevel={headingLevel}
			noteId={data.id}
			message={message}
			onToggleTask={(id, checked) => {
				const next = setTaskChecked(content, id, checked);
				setContent(next);
				onToggled();
				saver.saveNow({ content: next });
			}}
		/>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as NoteConfig;
	const notes = useQuery(notesQueryOptions);
	const [noteId, setNoteId] = useState(config.noteId);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	// A deleted note is no choice; another one has to be picked.
	const chosen = notes.data?.some((n) => n.id === noteId) ? noteId : "";

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.note.addTitle"
					: "dashboards.widgets.note.editTitle",
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
						noteId: chosen,
						size,
						showIfPublic,
					} satisfies NoteConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label htmlFor="note-widget-note" className="block text-sm font-medium">
					{t("dashboards.widgets.note.noteLabel")}
				</label>
				<select
					id="note-widget-note"
					value={chosen}
					onChange={(e) => setNoteId(e.target.value)}
					disabled={!notes.data}
					className={textInput}
				>
					<option value="" disabled>
						{t("dashboards.widgets.note.notePlaceholder")}
					</option>
					{notes.data?.map((n) => (
						<option key={n.id} value={n.id}>
							{displayTitle(n) ?? t("notes.untitled")}
						</option>
					))}
				</select>
				{config.noteId !== "" && notes.data && chosen === "" && (
					<p className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.note.missing")}
					</p>
				)}
			</div>
			<div>
				<label htmlFor="note-widget-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="note-widget-size"
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
					id="note-widget-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="note-widget-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="note-widget-public" className="text-sm font-medium">
						{t("dashboards.widgets.note.publicLabel")}
					</label>
					<p id="note-widget-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.note.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const noteKind: WidgetKind = {
	kind: "note",
	icon: StickyNote,
	group: "notes",
	labelKey: "dashboards.widgets.note.name",
	descriptionKey: "dashboards.widgets.note.description",
	defaultConfig: (): NoteConfig => ({
		noteId: "",
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
