import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import type { Note, NoteCategory } from "./api";
import { CategoryBadge } from "./categories/CategoryBadge";
import { noteBackground } from "./colours";
import { displayTitle } from "./document/text";
import { NoteRenderer } from "./NoteRenderer";
import { iconProps } from "./noteIcon";

/**
 * A note in the grid of /notes: icon, title and the beginning of its content,
 * on the note's colour, with its category badge. The whole card opens the
 * note; only the badge reacts on its own.
 */
export function NoteCard({
	note,
	categories,
	onCategoryChange,
}: {
	note: Note;
	categories: NoteCategory[];
	onCategoryChange: (categoryId: string | null) => void;
}) {
	const { t } = useTranslation();
	const name = displayTitle(note) ?? t("notes.untitled");
	return (
		<article
			className={`relative flex max-h-80 flex-col overflow-hidden rounded-xl border border-line p-4 shadow-[0_1px_2px_var(--color-shadow)] focus-within:ring-2 focus-within:ring-accent hover:shadow-[0_4px_12px_-4px_var(--color-shadow)] ${note.color ? "" : "bg-surface"}`}
			style={{ background: noteBackground(note.color) }}
		>
			{(note.icon || note.title) && (
				<h2 className="mb-2 flex items-center gap-2 font-semibold">
					{note.icon && (
						<ChosenIcon {...iconProps(note.icon)} className="size-5" />
					)}
					{note.title && <span className="min-w-0 truncate">{note.title}</span>}
				</h2>
			)}
			<div className="relative min-h-0 flex-1 overflow-hidden [mask-image:linear-gradient(to_bottom,black_70%,transparent)]">
				<NoteRenderer doc={note.content} />
			</div>
			{/* Covers the card, so a click anywhere but on the badge opens the note. */}
			<Link
				to="/notes/$noteId"
				params={{ noteId: note.id }}
				aria-label={t("notes.open", { title: name })}
				className="absolute inset-0 rounded-xl outline-none"
			/>
			{note.categoryId && (
				<div className="relative mt-3 flex">
					<CategoryBadge
						categoryId={note.categoryId}
						categories={categories}
						onChange={onCategoryChange}
					/>
				</div>
			)}
		</article>
	);
}
