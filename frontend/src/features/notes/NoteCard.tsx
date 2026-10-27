import { Link } from "@tanstack/react-router";
import { Maximize2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import {
	isVersionConflict,
	type Note,
	type NoteCategory,
	type NoteFields,
} from "./api";
import { CategoryBadge } from "./categories/CategoryBadge";
import { noteBackground } from "./colours";
import { displayTitle } from "./document/text";
import { setTaskChecked } from "./document/types";
import { NoteEditor } from "./editor/NoteEditor";
import { NoteRenderer } from "./NoteRenderer";
import { iconProps } from "./noteIcon";
import { SaveIndicator } from "./SaveIndicator";
import { useAutosave } from "./useAutosave";

/** What activating a card puts the cursor into. */
export type EditFocus = "title" | "content";

/** Why a change made on a card wasn't saved. */
export type CardSaveError = "conflict" | "failed";

type Draft = Pick<NoteFields, "title" | "content" | "categoryId">;

const draftOf = (note: Note): Draft => ({
	title: note.title,
	content: note.content,
	categoryId: note.categoryId,
});

type NoteCardProps = {
	note: Note;
	categories: NoteCategory[];
	/** How the card is being edited inline, or null when it isn't. */
	editing: EditFocus | null;
	onEditStart: (focus: EditFocus) => void;
	onEditEnd: () => void;
	onSaveError: (error: CardSaveError) => void;
};

/**
 * A note in the grid of /notes: icon, title and the beginning of its content,
 * on the note's colour, with its category badge. Activating the card edits
 * its title and content in place; checkboxes toggle their task right away.
 * "Open note" opens the editor page.
 */
export function NoteCard(props: NoteCardProps) {
	// A conflict stops the card's saver for good; a new one starts from the
	// note as the list has it.
	const [generation, setGeneration] = useState(0);
	return (
		<CardBody
			key={generation}
			{...props}
			onConflict={() => setGeneration((g) => g + 1)}
		/>
	);
}

function CardBody({
	note,
	categories,
	editing,
	onEditStart,
	onEditEnd,
	onSaveError,
	onConflict,
}: NoteCardProps & { onConflict: () => void }) {
	const { t } = useTranslation();
	const [draft, setDraft] = useState(() => draftOf(note));
	const editingRef = useRef(editing);
	editingRef.current = editing;
	const titleRef = useRef<HTMLHeadingElement>(null);
	const editButton = useRef<HTMLButtonElement>(null);
	const refocus = useRef(false);
	const saver = useAutosave({
		note,
		draft: { ...note, ...draft },
		onError: (err) => {
			if (isVersionConflict(err)) {
				onConflict();
				onSaveError("conflict");
			} else if (!editingRef.current) {
				// While editing, the save indicator offers Retry instead.
				saver.drop();
				setDraft(draftOf(note));
				onSaveError("failed");
			}
		},
	});
	const { idle, saveNow } = saver;

	// Shows the note as the list has it, unless changes of its own are
	// still on their way.
	useEffect(() => {
		if (!editingRef.current && idle()) setDraft(draftOf(note));
	}, [note, idle]);

	// Ending inline editing saves what is left right away.
	useEffect(() => {
		if (editing) return;
		saveNow();
		if (refocus.current) {
			refocus.current = false;
			editButton.current?.focus();
		}
	}, [editing, saveNow]);

	const update = (fields: Partial<Draft>) => {
		setDraft((d) => ({ ...d, ...fields }));
		saver.change(fields);
	};

	const shown = { ...note, ...draft };
	const name = displayTitle(shown) ?? t("notes.untitled");
	const openLink = (
		<Link
			to="/notes/$noteId"
			params={{ noteId: note.id }}
			aria-label={t("notes.open", { title: name })}
			title={t("notes.openNote")}
			className="absolute top-2 right-2 z-[2] rounded-md p-1.5 text-muted hover:bg-ink/5 hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
		>
			<Maximize2 className="size-4" aria-hidden />
		</Link>
	);
	const badge = draft.categoryId && (
		<div className="relative z-[2] mt-3 flex">
			<CategoryBadge
				categoryId={draft.categoryId}
				categories={categories}
				onChange={(categoryId) => {
					setDraft((d) => ({ ...d, categoryId }));
					saver.saveNow({ categoryId });
				}}
			/>
		</div>
	);
	const frame = `relative flex flex-col rounded-xl border border-line p-4 shadow-[0_1px_2px_var(--color-shadow)] ${note.color ? "" : "bg-surface"}`;

	if (editing) {
		return (
			<article
				className={`${frame} ring-2 ring-accent`}
				style={{ background: noteBackground(note.color) }}
				// Before the editor, which blurs itself on Escape.
				onKeyDownCapture={(e) => {
					if (e.key !== "Escape") return;
					e.preventDefault();
					refocus.current = true;
					onEditEnd();
				}}
				onBlur={(e) => {
					if (!e.currentTarget.contains(e.relatedTarget)) onEditEnd();
				}}
			>
				<div className="mb-2 flex items-center gap-2 pr-8">
					{note.icon && (
						<ChosenIcon {...iconProps(note.icon)} className="size-5 shrink-0" />
					)}
					<input
						defaultValue={draft.title ?? ""}
						onChange={(e) => update({ title: e.target.value.trim() || null })}
						placeholder={t("notes.editor.title")}
						aria-label={t("notes.editor.title")}
						maxLength={200}
						// Only mounted when the user activated the title.
						// biome-ignore lint/a11y/noAutofocus: focus follows the activation
						autoFocus={editing === "title"}
						className="min-w-0 flex-1 bg-transparent font-semibold outline-none placeholder:text-muted"
					/>
				</div>
				<NoteEditor
					initial={draft.content}
					toolbar={false}
					autoFocus={editing === "content"}
					onChange={(content) => update({ content })}
				/>
				<div className="mt-2 flex min-h-6 justify-end">
					<SaveIndicator status={saver.status} onRetry={saver.retry} />
				</div>
				{openLink}
				{badge}
			</article>
		);
	}

	return (
		<article
			className={`${frame} max-h-80 overflow-hidden focus-within:ring-2 focus-within:ring-accent hover:shadow-[0_4px_12px_-4px_var(--color-shadow)]`}
			style={{ background: noteBackground(note.color) }}
		>
			{(note.icon || draft.title) && (
				<h2
					ref={titleRef}
					className="mb-2 flex items-center gap-2 pr-8 font-semibold"
				>
					{note.icon && (
						<ChosenIcon {...iconProps(note.icon)} className="size-5" />
					)}
					{draft.title && (
						<span className="min-w-0 truncate">{draft.title}</span>
					)}
				</h2>
			)}
			{/* Above the edit button for the checkboxes; clicks on the rest pass through. */}
			<div className="pointer-events-none relative z-[1] min-h-0 flex-1 overflow-hidden pr-6 [mask-image:linear-gradient(to_bottom,black_70%,transparent)]">
				<NoteRenderer
					doc={draft.content}
					onToggleTask={(id, checked) => {
						const content = setTaskChecked(draft.content, id, checked);
						setDraft((d) => ({ ...d, content }));
						saver.saveNow({ content });
					}}
				/>
			</div>
			{/* Covers the card, so a click anywhere but on a checkbox, the badge or "Open note" edits the note. */}
			<button
				ref={editButton}
				type="button"
				aria-label={t("notes.edit", { title: name })}
				className="absolute inset-0 cursor-text rounded-xl outline-none"
				onClick={(e) => {
					// A click on the title edits it; the keyboard edits the content.
					const titleBottom = titleRef.current?.getBoundingClientRect().bottom;
					onEditStart(
						e.detail > 0 && titleBottom !== undefined && e.clientY < titleBottom
							? "title"
							: "content",
					);
				}}
			/>
			{openLink}
			{badge}
		</article>
	);
}
