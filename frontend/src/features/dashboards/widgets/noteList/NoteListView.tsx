import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { NoteRows } from "../NoteRows";
import { sizeStyles, type WidgetSize } from "../sizes";
import type { ListedNote, NoteListCategory, NoteListData } from "./list";

/**
 * The notes of a note list below its title as a heading of headingLevel when
 * it has one, or that its category is gone. With onOpen, each row is a button
 * that calls it with the note.
 */
export function NoteListView({
	data,
	category,
	size,
	title = null,
	headingLevel = 2,
	onOpen,
}: {
	data: NoteListData;
	category: NoteListCategory;
	size: WidgetSize;
	title?: string | null;
	headingLevel?: number;
	onOpen?: (note: ListedNote) => void;
}) {
	const { t } = useTranslation();
	const style = sizeStyles[size];

	let body: ReactNode;
	if ("missing" in data) {
		body = (
			<p className="text-muted">{t("dashboards.widgets.noteList.missing")}</p>
		);
	} else if (data.notes.length === 0) {
		body = (
			<p className="text-muted">
				{t(
					category === "all"
						? "dashboards.widgets.noteList.empty"
						: "dashboards.widgets.noteList.emptyCategory",
				)}
			</p>
		);
	} else {
		body = <NoteRows notes={data.notes} size={size} onOpen={onOpen} />;
	}

	const Heading = `h${headingLevel}` as "h2";
	return (
		<div className={style.text}>
			{title && <Heading className="truncate font-medium">{title}</Heading>}
			{body}
		</div>
	);
}
