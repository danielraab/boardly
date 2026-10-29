import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import type { Note } from "#/features/notes/api";
import { NoteMarks } from "#/features/notes/NoteMarks";
import { iconProps } from "#/features/notes/noteIcon";

/**
 * A note as a choice in a picker: its icon, its display title (or
 * "Untitled"), the archived mark and its category path at the end. The title
 * takes the room, the path at most half of it; both are cut off with "…".
 * Dimming archived notes is up to the row it's in.
 */
export function NoteChoiceLabel({
	note,
	title,
	path,
}: {
	note: Pick<Note, "icon" | "archived">;
	title: string | null;
	path: string[];
}) {
	const { t } = useTranslation();
	return (
		<>
			{note.icon && (
				<ChosenIcon {...iconProps(note.icon)} className="size-4 shrink-0" />
			)}
			<span className="min-w-0 flex-1 truncate">
				{title ?? t("notes.untitled")}
			</span>
			<NoteMarks pinned={false} archived={note.archived} compact />
			{path.length > 0 && (
				<span className="max-w-1/2 min-w-0 shrink-0 truncate pl-1 text-xs text-muted">
					{path.join(" › ")}
				</span>
			)}
		</>
	);
}
