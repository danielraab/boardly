import { Maximize2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import type { NoteIcon } from "#/features/notes/api";
import {
	ReadOnlyCategoryBadge,
	type ShownCategory,
} from "#/features/notes/categories/CategoryBadge";
import { type NoteColour, noteBackground } from "#/features/notes/colours";
import { displayTitle } from "#/features/notes/document/text";
import type { NoteDoc } from "#/features/notes/document/types";
import { NoteMarks } from "#/features/notes/NoteMarks";
import { NoteRenderer } from "#/features/notes/NoteRenderer";
import { iconProps } from "#/features/notes/noteIcon";
import { sizeStyles, type WidgetSize } from "../sizes";

/** A shown note as the server resolves it; id, version and path are the owner's. */
export type ShownNote = {
	id?: string;
	version?: number;
	title: string | null;
	icon: NoteIcon | null;
	color: NoteColour | null;
	content: NoteDoc;
	pinned: boolean;
	archived: boolean;
	category: (ShownCategory & { path?: string[] }) | null;
};

export type NoteData = ShownNote | { missing: true };

/** The title's text size for each size: one step above the content's. */
const titleText: Record<WidgetSize, string> = {
	S: "text-sm",
	M: "text-base",
	L: "text-lg",
};

/**
 * A note as a card that fills its widget: colour, a header with icon, title,
 * category badge and the pin and archive marks, and the content. Checkboxes only toggle with
 * onToggleTask, and "Open note" is only shown with onOpen.
 */
export function NoteWidgetView({
	data,
	size,
	headingLevel,
	onOpen,
	onToggleTask,
	message,
}: {
	data: NoteData;
	size: WidgetSize;
	headingLevel: number;
	onOpen?: () => void;
	onToggleTask?: (id: string, checked: boolean) => void;
	/** Why the last change wasn't saved. */
	message?: string | null;
}) {
	const { t } = useTranslation();
	if ("missing" in data) {
		return (
			<p className={`text-muted ${sizeStyles[size].text}`}>
				{t("dashboards.widgets.note.missing")}
			</p>
		);
	}
	const Heading = `h${headingLevel}` as "h2";
	const name = displayTitle(data) ?? t("notes.untitled");
	const marked = data.pinned || data.archived;
	return (
		<article
			className={`relative flex min-h-full flex-col rounded-xl border border-line p-4 shadow-[0_1px_2px_var(--color-shadow)] ${data.color ? "" : "bg-surface"}`}
			style={{ background: noteBackground(data.color) }}
		>
			{(data.icon || data.title || data.category || marked) && (
				<div className={`mb-2 flex items-center gap-2 ${onOpen ? "pr-8" : ""}`}>
					{(data.icon || data.title) && (
						<Heading
							className={`flex min-w-0 items-center gap-2 font-semibold ${titleText[size]}`}
						>
							{data.icon && (
								<ChosenIcon
									{...iconProps(data.icon)}
									className={`${sizeStyles[size].icon} shrink-0`}
								/>
							)}
							{data.title && (
								<span className="min-w-0 truncate">{data.title}</span>
							)}
						</Heading>
					)}
					{data.category && (
						<div className="flex shrink-0">
							<ReadOnlyCategoryBadge
								category={data.category}
								tooltip={data.category.path?.join(" › ") ?? data.category.title}
							/>
						</div>
					)}
					<NoteMarks
						pinned={data.pinned}
						archived={data.archived}
						className="ml-auto"
					/>
				</div>
			)}
			<div
				className={
					onOpen && !(data.icon || data.title || data.category || marked)
						? "pr-6"
						: ""
				}
			>
				<NoteRenderer
					doc={data.content}
					size={size}
					onToggleTask={onToggleTask}
				/>
			</div>
			{message && (
				<p role="alert" className="mt-2 text-sm text-red-700">
					{message}
				</p>
			)}
			{onOpen && (
				<button
					type="button"
					onClick={onOpen}
					aria-label={t("notes.open", { title: name })}
					title={t("notes.openNote")}
					className="absolute top-2 right-2 rounded-md p-1.5 text-muted hover:bg-ink/5 hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
				>
					<Maximize2 className="size-4" aria-hidden />
				</button>
			)}
		</article>
	);
}
