import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { NotebookPen, Plus } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { fullPage } from "#/components/pageWidths";
import { Toast, type ToastMessage } from "#/components/Toast";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	type Note,
	noteCategoriesQueryOptions,
	notesQueryKey,
	notesQueryOptions,
} from "./api";
import { SidePanel } from "./categories/SidePanel";
import {
	type CategoryFilter,
	filterNotes,
	pinnedFirst,
} from "./categories/tree";
import { type EditFocus, NoteCard } from "./NoteCard";

/** The category search param of /notes: a category ID, "none" or "archived". */
export type NotesSearch = { category?: string };

export function filterOf(search: NotesSearch): CategoryFilter {
	if (search.category === "none" || search.category === "archived")
		return search.category;
	return search.category ? { id: search.category } : "all";
}

export function searchOf(filter: CategoryFilter): NotesSearch {
	if (filter === "all") return { category: undefined };
	return { category: typeof filter === "object" ? filter.id : filter };
}

/**
 * Keeps notes in the order they first had, so a card doesn't jump while it is
 * changed on the page. Notes that weren't there yet come first, in the order
 * given.
 */
function useStableOrder(notes: Note[]): Note[] {
	const order = useRef<string[]>([]);
	const position = new Map(order.current.map((id, i) => [id, i]));
	const fresh = notes.filter((n) => !position.has(n.id));
	const known = notes
		.filter((n) => position.has(n.id))
		.sort((a, b) => (position.get(a.id) ?? 0) - (position.get(b.id) ?? 0));
	const result = [...fresh, ...known];
	if (fresh.length > 0) order.current = result.map((n) => n.id);
	return result;
}

/** /notes: the card grid of the user's notes with the category side panel. */
export function NotesPage({
	search,
	onSearchChange,
}: {
	search: NotesSearch;
	onSearchChange: (search: NotesSearch) => void;
}) {
	const { t } = useTranslation();
	usePageTitle(t("notes.title"));
	const queryClient = useQueryClient();
	const notes = useQuery(notesQueryOptions);
	const categories = useQuery(noteCategoriesQueryOptions);
	const [toast, setToast] = useState<ToastMessage | null>(null);
	const [editing, setEditing] = useState<{
		id: string;
		focus: EditFocus;
	} | null>(null);
	const ordered = useStableOrder(pinnedFirst(notes.data ?? []));

	const allCategories = categories.data ?? [];
	let filter = filterOf(search);
	// An unknown category shows all notes.
	if (
		typeof filter === "object" &&
		categories.data &&
		!allCategories.some((c) => c.id === (filter as { id: string }).id)
	) {
		filter = "all";
	}
	const shown = filterNotes(ordered, allCategories, filter);

	const onSaveError = (id: string, error: "conflict" | "failed") => {
		if (error === "conflict") setEditing((e) => (e?.id === id ? null : e));
		void queryClient.invalidateQueries({ queryKey: notesQueryKey });
		setToast({
			id: Date.now(),
			text:
				error === "conflict"
					? t("notes.changedElsewhere")
					: t("notes.changeFailed"),
		});
	};

	const newNoteSearch =
		typeof filter === "object" ? { category: filter.id } : {};

	return (
		<section className={`${fullPage} pt-12 pb-16`}>
			<div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("notes.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">{t("notes.intro")}</p>
				</div>
				<Link
					to="/notes/$noteId"
					params={{ noteId: "new" }}
					search={newNoteSearch}
					className={primaryButton}
				>
					<Plus className="size-4" aria-hidden />
					{t("notes.newNote")}
				</Link>
			</div>
			<div className="mt-8 flex flex-col gap-6 md:flex-row md:items-start">
				<SidePanel
					notes={notes.data ?? []}
					categories={allCategories}
					filter={filter}
					onFilterChange={(f) => onSearchChange(searchOf(f))}
				/>
				<div className="min-w-0 flex-1">
					{notes.isError && (
						<p role="alert" className="text-sm text-danger">
							{t("common.genericError")}
						</p>
					)}
					{notes.data && notes.data.length === 0 && (
						<div className="rounded-xl border border-dashed border-line p-10 text-center">
							<NotebookPen className="mx-auto size-8 text-muted" aria-hidden />
							<p className="mt-3 font-medium">{t("notes.empty")}</p>
							<p className="mt-1 text-sm text-muted">{t("notes.emptyHint")}</p>
							<Link
								to="/notes/$noteId"
								params={{ noteId: "new" }}
								search={newNoteSearch}
								className={`${primaryButton} mt-5`}
							>
								<Plus className="size-4" aria-hidden />
								{t("notes.newNote")}
							</Link>
						</div>
					)}
					{notes.data && notes.data.length > 0 && shown.length === 0 && (
						<p className="rounded-xl border border-dashed border-line p-10 text-center text-muted">
							{filter === "archived"
								? t("notes.emptyArchived")
								: filter === "all"
									? t("notes.allArchived")
									: t("notes.emptyCategory")}
						</p>
					)}
					{shown.length > 0 && (
						<ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
							{shown.map((note) => (
								<li key={note.id} className="min-w-0">
									<NoteCard
										note={note}
										categories={allCategories}
										editing={editing?.id === note.id ? editing.focus : null}
										onEditStart={(focus) => setEditing({ id: note.id, focus })}
										onEditEnd={() =>
											setEditing((e) => (e?.id === note.id ? null : e))
										}
										onSaveError={(error) => onSaveError(note.id, error)}
									/>
								</li>
							))}
						</ul>
					)}
				</div>
			</div>
			<Toast toast={toast} onClose={() => setToast(null)} />
		</section>
	);
}
