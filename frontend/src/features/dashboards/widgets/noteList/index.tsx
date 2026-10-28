import { useQuery, useQueryClient } from "@tanstack/react-query";
import { NotebookText } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import {
	noteCategoriesQueryOptions,
	notesQueryOptions,
} from "#/features/notes/api";
import { CategoryPicker } from "#/features/notes/categories/CategoryPicker";
import { NoteDialog } from "../note/NoteDialog";
import { maxRecentLimit, parseCount } from "../recentFavourites/recent";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import {
	type NoteListCategory,
	type NoteListData,
	type NoteListSort,
	noteList,
} from "./list";
import { NoteListView } from "./NoteListView";

export type NoteListConfig = {
	category: NoteListCategory;
	categoryId: string | null;
	subcategories: boolean;
	sort: NoteListSort;
	limit: number;
	title: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

/** A new note list's config, before a category is chosen. */
type DraftNoteListConfig = Omit<NoteListConfig, "category"> & {
	category: NoteListCategory | null;
};

const sorts: NoteListSort[] = ["updated", "created", "title"];

// The picker's choices for all notes and the uncategorized ones; category IDs
// never start with a colon.
const allChoice = ":all";
const noneChoice = ":none";

/**
 * In view mode the server's data, where the owner opens a note's editor in a
 * modal; in edit mode the owner's current notes, since a draft widget has no
 * data yet and its settings may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const config = widget.config as NoteListConfig;
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	// The last opened note stays while the modal closes.
	const [opened, setOpened] = useState<{ id: string; name: string } | null>(
		null,
	);
	const [open, setOpen] = useState(false);
	const notes = useQuery({ ...notesQueryOptions, enabled: editing });
	const categories = useQuery({
		...noteCategoriesQueryOptions,
		enabled: editing,
	});

	let data = widget.data as NoteListData | undefined;
	if (editing) {
		if (!notes.data || !categories.data) return null;
		data = noteList(notes.data, categories.data, config);
	}
	if (!data) return null;
	const canOpen = isOwner && !editing;
	const refresh = () =>
		void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
	return (
		<>
			<NoteListView
				data={data}
				category={config.category}
				size={config.size}
				title={config.title}
				headingLevel={headingLevel}
				onOpen={
					canOpen
						? (note) => {
								if (!note.id) return;
								setOpened({
									id: note.id,
									name: note.title ?? t("notes.untitled"),
								});
								setOpen(true);
							}
						: undefined
				}
			/>
			{canOpen && opened && (
				<NoteDialog
					noteId={opened.id}
					name={opened.name}
					open={open}
					onClose={() => setOpen(false)}
					onChanged={refresh}
					onDeleted={() => {
						setOpen(false);
						refresh();
					}}
				/>
			)}
		</>
	);
}

function Settings(props: WidgetSettingsProps) {
	const { t } = useTranslation();
	const config = props.widget.config as DraftNoteListConfig;
	const [choice, setChoice] = useState<string | null>(
		config.category === "all"
			? allChoice
			: config.category === "none"
				? noneChoice
				: config.category === "category"
					? config.categoryId
					: null,
	);
	const [subcategories, setSubcategories] = useState(config.subcategories);
	const [sort, setSort] = useState(config.sort);
	const [title, setTitle] = useState(config.title ?? "");
	const [limit, setLimit] = useState(String(config.limit));
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const limitValue = parseCount(limit, maxRecentLimit);
	const isCategory =
		choice !== null && choice !== allChoice && choice !== noneChoice;

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.noteList.addTitle"
					: "dashboards.widgets.noteList.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={choice !== null && limitValue !== null}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						category:
							choice === allChoice
								? "all"
								: choice === noneChoice
									? "none"
									: "category",
						categoryId: isCategory ? choice : null,
						subcategories,
						sort,
						limit: limitValue ?? config.limit,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies NoteListConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<CategoryPicker
				id="note-list-category"
				label={t("dashboards.widgets.noteList.categoryLabel")}
				extraChoices={[
					{ id: allChoice, title: t("notes.categories.all") },
					{ id: noneChoice, title: t("notes.categories.none") },
				]}
				value={choice}
				onChange={setChoice}
			/>
			<div className="flex items-start gap-3">
				<input
					id="note-list-subcategories"
					type="checkbox"
					checked={subcategories}
					disabled={!isCategory}
					onChange={(e) => setSubcategories(e.target.checked)}
					className="mt-1 size-4 accent-ink disabled:opacity-50"
				/>
				<label
					htmlFor="note-list-subcategories"
					className={`text-sm font-medium ${isCategory ? "" : "text-muted"}`}
				>
					{t("dashboards.widgets.noteList.subcategoriesLabel")}
				</label>
			</div>
			<div>
				<label htmlFor="note-list-title" className="block text-sm font-medium">
					{t("dashboards.widgets.noteList.titleLabel")}
				</label>
				<input
					id="note-list-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="note-list-title-hint"
					className={textInput}
				/>
				<p id="note-list-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.noteList.titleHint")}
				</p>
			</div>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label htmlFor="note-list-sort" className="block text-sm font-medium">
						{t("dashboards.widgets.noteList.sortLabel")}
					</label>
					<select
						id="note-list-sort"
						value={sort}
						onChange={(e) => setSort(e.target.value as NoteListSort)}
						className={textInput}
					>
						{sorts.map((s) => (
							<option key={s} value={s}>
								{t(`dashboards.widgets.noteList.sort.${s}`)}
							</option>
						))}
					</select>
				</div>
				<div>
					<label
						htmlFor="note-list-limit"
						className="block text-sm font-medium"
					>
						{t("dashboards.widgets.noteList.limitLabel")}
					</label>
					<input
						id="note-list-limit"
						type="number"
						inputMode="numeric"
						min={1}
						max={maxRecentLimit}
						step={1}
						required
						value={limit}
						onChange={(e) => setLimit(e.target.value)}
						aria-invalid={limitValue === null || undefined}
						aria-describedby="note-list-limit-hint"
						className={textInput}
					/>
					<p id="note-list-limit-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.noteList.limitHint", {
							max: maxRecentLimit,
						})}
					</p>
				</div>
			</div>
			<div>
				<label htmlFor="note-list-size" className="block text-sm font-medium">
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="note-list-size"
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
					id="note-list-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="note-list-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="note-list-public" className="text-sm font-medium">
						{t("dashboards.widgets.noteList.publicLabel")}
					</label>
					<p id="note-list-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.noteList.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const noteListKind: WidgetKind = {
	kind: "note_list",
	icon: NotebookText,
	group: "notes",
	labelKey: "dashboards.widgets.noteList.name",
	descriptionKey: "dashboards.widgets.noteList.description",
	defaultConfig: (): DraftNoteListConfig => ({
		category: null,
		categoryId: null,
		subcategories: true,
		sort: "updated",
		limit: 10,
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
