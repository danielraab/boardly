import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { iconProps } from "#/features/notes/noteIcon";
import { fullDate, relativeTime } from "../relativeTime";
import { sizeStyles, type WidgetSize } from "../sizes";
import type { RecentNote, RecentNotesData } from "./recent";

/** The category and the time are one step smaller than the title. */
const smallText: Record<WidgetSize, string> = {
	S: "text-xs",
	M: "text-xs",
	L: "text-sm",
};

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
	const { t, i18n } = useTranslation();
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
			<ul>
				{data.notes.map((note, i) => {
					const time = new Date(note.time);
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
										title={
											note.category.path?.join(" › ") ?? note.category.title
										}
										className={`truncate text-muted ${smallText[size]}`}
									>
										{note.category.title}
									</span>
								)}
							</span>
							<time
								dateTime={note.time}
								title={fullDate(time, i18n.language)}
								className={`ml-auto shrink-0 pl-2 whitespace-nowrap text-muted ${smallText[size]}`}
							>
								{relativeTime(time, now, i18n.language)}
							</time>
						</>
					);
					const rowClass = `flex w-full min-w-0 items-center gap-2 rounded-lg px-1 text-left ${style.row}`;
					return (
						<li key={note.id ?? i}>
							{onOpen ? (
								<button
									type="button"
									title={name}
									onClick={() => onOpen(note)}
									className={`${rowClass} hover:bg-paper focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none`}
								>
									{row}
								</button>
							) : (
								<div title={name} className={rowClass}>
									{row}
								</div>
							)}
						</li>
					);
				})}
			</ul>
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
