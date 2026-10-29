import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import type { NoteIcon } from "#/features/notes/api";
import { type NoteColour, noteBackground } from "#/features/notes/colours";
import { NoteMarks } from "#/features/notes/NoteMarks";
import { iconProps } from "#/features/notes/noteIcon";
import { fullDate, relativeTime } from "./relativeTime";
import { sizeStyles, type WidgetSize } from "./sizes";

/** A note as a row of a note widget list; id and path are the owner's. */
export type NoteRow = {
	id?: string;
	/** The display title, or null for "Untitled note". */
	title: string | null;
	icon: NoteIcon | null;
	color: NoteColour | null;
	pinned: boolean;
	archived: boolean;
	category: { title: string; path?: string[] } | null;
	/** A time to show at the row's end, as RFC 3339. */
	time?: string;
};

/** The category and the time are one step smaller than the title. */
const smallText: Record<WidgetSize, string> = {
	S: "text-xs",
	M: "text-xs",
	L: "text-sm",
};

/** The gap between rows grows with the size. */
const rowGap: Record<WidgetSize, string> = {
	S: "gap-1",
	M: "gap-1",
	L: "gap-1.5",
};

/**
 * Notes as rows with their icon, display title, category as a second line
 * and, when they have one, a time relative to now, followed by the pin and
 * archive marks. Each row looks like a small note card in the note's colour;
 * archived notes are dimmed. With onOpen, each row is a button that calls
 * it with the note.
 */
export function NoteRows<T extends NoteRow>({
	notes,
	size,
	onOpen,
	now = new Date(),
}: {
	notes: T[];
	size: WidgetSize;
	onOpen?: (note: T) => void;
	now?: Date;
}) {
	const { t, i18n } = useTranslation();
	const style = sizeStyles[size];
	return (
		<ul className={`flex flex-col ${rowGap[size]}`}>
			{notes.map((note, i) => {
				const name = note.title ?? t("notes.untitled");
				const row = (
					<>
						{note.icon && (
							<ChosenIcon {...iconProps(note.icon)} className={style.icon} />
						)}
						<span className="flex min-w-0 flex-col">
							<span className="truncate font-medium">{name}</span>
							{note.category && (
								<span
									title={note.category.path?.join(" › ") ?? note.category.title}
									className={`truncate text-muted ${smallText[size]}`}
								>
									{note.category.title}
								</span>
							)}
						</span>
						{note.time && (
							<time
								dateTime={note.time}
								title={fullDate(new Date(note.time), i18n.language)}
								className={`ml-auto shrink-0 pl-2 whitespace-nowrap text-muted ${smallText[size]}`}
							>
								{relativeTime(new Date(note.time), now, i18n.language)}
							</time>
						)}
						<NoteMarks
							pinned={note.pinned}
							archived={note.archived}
							compact
							className={note.time ? "" : "ml-auto pl-2"}
						/>
					</>
				);
				// Like a note card: a border and the note's colour, or surface.
				const rowClass = `flex w-full min-w-0 items-center gap-2 rounded-lg border border-line px-2 text-left ${style.row} ${note.color ? "" : "bg-surface"} ${note.archived ? "opacity-70" : ""}`;
				const background = { background: noteBackground(note.color) };
				return (
					<li key={note.id ?? i}>
						{onOpen ? (
							<button
								type="button"
								title={name}
								onClick={() => onOpen(note)}
								style={background}
								className={`${rowClass} hover:border-ink/30 focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none`}
							>
								{row}
							</button>
						) : (
							<div title={name} style={background} className={rowClass}>
								{row}
							</div>
						)}
					</li>
				);
			})}
		</ul>
	);
}
