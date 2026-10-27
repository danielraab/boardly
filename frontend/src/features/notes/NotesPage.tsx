import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { NotebookPen, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { Toast, type ToastMessage } from "#/components/Toast";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	isVersionConflict,
	type Note,
	noteCategoriesQueryOptions,
	notesQueryKey,
	notesQueryOptions,
	updateNote,
} from "./api";
import { SidePanel } from "./categories/SidePanel";
import { type CategoryFilter, filterNotes } from "./categories/tree";
import { NoteCard } from "./NoteCard";

/** The category search param of /notes: a category ID, or "none". */
export type NotesSearch = { category?: string };

export function filterOf(search: NotesSearch): CategoryFilter {
	if (search.category === "none") return "none";
	return search.category ? { id: search.category } : "all";
}

export function searchOf(filter: CategoryFilter): NotesSearch {
	if (filter === "all") return { category: undefined };
	return { category: filter === "none" ? "none" : filter.id };
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
	const shown = filterNotes(notes.data ?? [], allCategories, filter);

	const changeCategory = useMutation({
		mutationFn: ({
			note,
			categoryId,
		}: {
			note: Note;
			categoryId: string | null;
		}) => updateNote(note.id, note.version, { categoryId }),
		onError: (err) =>
			setToast({
				id: Date.now(),
				text: isVersionConflict(err)
					? t("notes.changedElsewhere")
					: t("notes.changeFailed"),
			}),
		onSettled: () =>
			void queryClient.invalidateQueries({ queryKey: notesQueryKey }),
	});

	const newNoteSearch =
		typeof filter === "object" ? { category: filter.id } : {};

	return (
		<section className="pt-12 pb-16">
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
							{t("notes.emptyCategory")}
						</p>
					)}
					{shown.length > 0 && (
						<ul className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
							{shown.map((note) => (
								<li key={note.id} className="min-w-0">
									<NoteCard
										note={note}
										categories={allCategories}
										onCategoryChange={(categoryId) =>
											changeCategory.mutate({ note, categoryId })
										}
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
