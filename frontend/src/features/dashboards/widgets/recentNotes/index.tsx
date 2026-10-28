import { useQuery, useQueryClient } from "@tanstack/react-query";
import { FileClock } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import {
	noteCategoriesQueryOptions,
	notesQueryOptions,
} from "#/features/notes/api";
import { NoteDialog } from "../note/NoteDialog";
import {
	maxRecentDays,
	maxRecentLimit,
	parseCount,
} from "../recentFavourites/recent";
import type {
	WidgetKind,
	WidgetSettingsProps,
	WidgetViewProps,
} from "../registry";
import { type WidgetSize, widgetSizes } from "../sizes";
import { maxWidgetTitle, WidgetSettingsForm } from "../WidgetSettingsForm";
import { RecentNotesView } from "./RecentNotesView";
import {
	type RecentNotesData,
	type RecentNotesSort,
	recentNotes,
} from "./recent";

export type RecentNotesConfig = {
	sortBy: RecentNotesSort;
	limit: number;
	days: number | null;
	title: string | null;
	size: WidgetSize;
	showIfPublic: boolean;
};

const sorts: RecentNotesSort[] = ["created", "updated"];

/**
 * In view mode the server's data, where the owner opens a note's editor in a
 * modal; in edit mode the newest of the owner's current notes, since a draft
 * widget has no data yet and its settings may have changed.
 */
function View({ widget, editing, headingLevel }: WidgetViewProps) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const config = widget.config as RecentNotesConfig;
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

	let data = widget.data as RecentNotesData | undefined;
	if (editing) {
		if (!notes.data || !categories.data) return null;
		data = {
			notes: recentNotes(
				notes.data,
				categories.data,
				config.sortBy,
				config.limit,
				config.days,
				new Date(),
			),
		};
	}
	if (!data) return null;
	const canOpen = isOwner && !editing;
	const refresh = () =>
		void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
	return (
		<>
			<RecentNotesView
				data={data}
				days={config.days}
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
	const config = props.widget.config as RecentNotesConfig;
	const [sortBy, setSortBy] = useState(config.sortBy);
	const [title, setTitle] = useState(config.title ?? "");
	const [limit, setLimit] = useState(String(config.limit));
	const [days, setDays] = useState(
		config.days === null ? "" : String(config.days),
	);
	const [size, setSize] = useState(config.size);
	const [showIfPublic, setShowIfPublic] = useState(config.showIfPublic);
	const limitValue = parseCount(limit, maxRecentLimit);
	const daysValue = days.trim() === "" ? null : parseCount(days, maxRecentDays);
	const daysValid = days.trim() === "" || daysValue !== null;

	return (
		<WidgetSettingsForm
			title={t(
				props.isNew
					? "dashboards.widgets.recentNotes.addTitle"
					: "dashboards.widgets.recentNotes.editTitle",
			)}
			widget={props.widget}
			isNew={props.isNew}
			desktopColumns={props.desktopColumns}
			valid={limitValue !== null && daysValid}
			onApply={(dims) =>
				props.onApply({
					...props.widget,
					...dims,
					config: {
						sortBy,
						limit: limitValue ?? config.limit,
						days: daysValue,
						title: title.trim() || null,
						size,
						showIfPublic,
					} satisfies RecentNotesConfig,
				})
			}
			onDelete={props.onDelete}
		>
			<div>
				<label
					htmlFor="recent-notes-title"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.recentNotes.titleLabel")}
				</label>
				<input
					id="recent-notes-title"
					value={title}
					maxLength={maxWidgetTitle}
					onChange={(e) => setTitle(e.target.value)}
					aria-describedby="recent-notes-title-hint"
					className={textInput}
				/>
				<p id="recent-notes-title-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.recentNotes.titleHint")}
				</p>
			</div>
			<div>
				<label
					htmlFor="recent-notes-sort"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.recentNotes.sortLabel")}
				</label>
				<select
					id="recent-notes-sort"
					value={sortBy}
					onChange={(e) => setSortBy(e.target.value as RecentNotesSort)}
					aria-describedby="recent-notes-sort-hint"
					className={textInput}
				>
					{sorts.map((s) => (
						<option key={s} value={s}>
							{t(`dashboards.widgets.recentNotes.sort.${s}`)}
						</option>
					))}
				</select>
				<p id="recent-notes-sort-hint" className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.recentNotes.sortHint")}
				</p>
			</div>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label
						htmlFor="recent-notes-limit"
						className="block text-sm font-medium"
					>
						{t("dashboards.widgets.recentNotes.limitLabel")}
					</label>
					<input
						id="recent-notes-limit"
						type="number"
						inputMode="numeric"
						min={1}
						max={maxRecentLimit}
						step={1}
						required
						value={limit}
						onChange={(e) => setLimit(e.target.value)}
						aria-invalid={limitValue === null || undefined}
						aria-describedby="recent-notes-limit-hint"
						className={textInput}
					/>
					<p id="recent-notes-limit-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.recentNotes.limitHint", {
							max: maxRecentLimit,
						})}
					</p>
				</div>
				<div>
					<label
						htmlFor="recent-notes-days"
						className="block text-sm font-medium"
					>
						{t("dashboards.widgets.recentNotes.daysLabel")}
					</label>
					<input
						id="recent-notes-days"
						type="number"
						inputMode="numeric"
						min={1}
						max={maxRecentDays}
						step={1}
						value={days}
						onChange={(e) => setDays(e.target.value)}
						aria-invalid={!daysValid || undefined}
						aria-describedby="recent-notes-days-hint"
						className={textInput}
					/>
					<p id="recent-notes-days-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.widgets.recentNotes.daysHint", {
							max: maxRecentDays,
						})}
					</p>
				</div>
			</div>
			<div>
				<label
					htmlFor="recent-notes-size"
					className="block text-sm font-medium"
				>
					{t("dashboards.widgets.sizeLabel")}
				</label>
				<select
					id="recent-notes-size"
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
					id="recent-notes-public"
					type="checkbox"
					checked={showIfPublic}
					onChange={(e) => setShowIfPublic(e.target.checked)}
					aria-describedby="recent-notes-public-hint"
					className="mt-1 size-4 accent-ink"
				/>
				<div>
					<label htmlFor="recent-notes-public" className="text-sm font-medium">
						{t("dashboards.widgets.recentNotes.publicLabel")}
					</label>
					<p id="recent-notes-public-hint" className="text-sm text-muted">
						{t("dashboards.widgets.recentNotes.publicHint")}
					</p>
				</div>
			</div>
		</WidgetSettingsForm>
	);
}

export const recentNotesKind: WidgetKind = {
	kind: "recent_notes",
	icon: FileClock,
	group: "notes",
	labelKey: "dashboards.widgets.recentNotes.name",
	descriptionKey: "dashboards.widgets.recentNotes.description",
	defaultConfig: (): RecentNotesConfig => ({
		sortBy: "created",
		limit: 10,
		days: null,
		title: null,
		size: "M",
		showIfPublic: false,
	}),
	View,
	Settings,
};
