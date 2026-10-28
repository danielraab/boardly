import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { NoteRows } from "../NoteRows";
import { sizeStyles, type WidgetSize } from "../sizes";
import type { RecentNote, RecentNotesData } from "./recent";

/**
 * The notes of a recent notes widget, each with its time, below its title as
 * a heading of headingLevel when it has one. With onOpen, each row is a
 * button that calls it with the note. now is the time the times are relative
 * to.
 */
export function RecentNotesView({
	data,
	days,
	size,
	title = null,
	headingLevel = 2,
	onOpen,
	now = new Date(),
}: {
	data: RecentNotesData;
	days: number | null;
	size: WidgetSize;
	title?: string | null;
	headingLevel?: number;
	onOpen?: (note: RecentNote) => void;
	now?: Date;
}) {
	const { t } = useTranslation();
	const style = sizeStyles[size];

	let body: ReactNode;
	if (data.notes.length === 0) {
		body = (
			<p className="text-muted">
				{days === null
					? t("dashboards.widgets.recentNotes.empty")
					: t("dashboards.widgets.recentNotes.emptyDays", { count: days })}
			</p>
		);
	} else {
		body = (
			<NoteRows notes={data.notes} size={size} onOpen={onOpen} now={now} />
		);
	}

	const Heading = `h${headingLevel}` as "h2";
	return (
		<div className={style.text}>
			{title && <Heading className="truncate font-medium">{title}</Heading>}
			{body}
		</div>
	);
}
