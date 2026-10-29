import { Combobox } from "@base-ui/react/combobox";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { menuItem, menuPopup } from "#/components/menuStyles";
import type { Note, NoteCategory } from "#/features/notes/api";
import { displayTitle } from "#/features/notes/document/text";
import { NoteChoiceLabel } from "../NoteChoiceLabel";
import { noteChoices, notePath } from "../selectedNotes/selected";

/**
 * A search over all of the owner's notes to choose one. Until something is
 * typed, the field shows the chosen note like a result, with its icon,
 * category path and archived mark, and all notes are offered. A value
 * without a note in notes shows as nothing chosen.
 */
export function NotePicker({
	id,
	notes,
	categories,
	value,
	onChange,
}: {
	id: string;
	notes: Note[];
	categories: NoteCategory[];
	value: string;
	onChange: (noteId: string) => void;
}) {
	const { t } = useTranslation();
	const byID = new Map(notes.map((n) => [n.id, n]));
	const label = (noteId: string) => {
		const note = byID.get(noteId);
		return note ? (displayTitle(note) ?? t("notes.untitled")) : "";
	};
	const chosen = byID.get(value);
	const [input, setInput] = useState(chosen ? label(value) : "");
	const showsChosen = chosen !== undefined && input === label(value);
	const choices = noteChoices(notes, categories, [], showsChosen ? "" : input);
	const choiceByID = new Map(choices.map((c) => [c.note.id, c]));

	return (
		<Combobox.Root
			items={choices.map((c) => c.note.id)}
			filter={null}
			value={chosen ? value : null}
			onValueChange={(noteId: string | null) => {
				if (noteId !== null) onChange(noteId);
			}}
			itemToStringLabel={label}
			inputValue={input}
			onInputValueChange={setInput}
			onOpenChange={(open) => {
				// Leaving without choosing shows the chosen note again.
				if (!open) setInput(chosen ? label(value) : "");
			}}
		>
			<div className="relative">
				<Combobox.Input
					id={id}
					placeholder={t("dashboards.widgets.note.notePlaceholder")}
					className={`${textInput} ${showsChosen ? "text-transparent caret-ink" : ""}`}
				/>
				{showsChosen && (
					// Over the input's text, inside its border and padding.
					<span
						aria-hidden
						className={`pointer-events-none absolute inset-x-0 top-2 bottom-0 flex items-center gap-2 border border-transparent px-4 ${chosen.archived ? "opacity-70" : ""}`}
					>
						<NoteChoiceLabel
							note={chosen}
							title={displayTitle(chosen)}
							path={notePath(chosen, categories)}
						/>
					</span>
				)}
			</div>
			<Combobox.Portal>
				<Combobox.Positioner sideOffset={4} className="z-50">
					<Combobox.Popup
						className={`${menuPopup} max-h-64 w-[var(--anchor-width)] overflow-y-auto`}
					>
						<Combobox.Empty className="px-3 py-2 text-muted empty:hidden">
							{t("dashboards.widgets.note.noMatches")}
						</Combobox.Empty>
						<Combobox.List>
							{(noteId: string) => {
								const choice = choiceByID.get(noteId);
								if (!choice) return null;
								return (
									<Combobox.Item
										key={noteId}
										value={noteId}
										className={`${menuItem} ${choice.note.archived ? "opacity-70" : ""}`}
									>
										<NoteChoiceLabel
											note={choice.note}
											title={choice.title}
											path={choice.path}
										/>
									</Combobox.Item>
								);
							}}
						</Combobox.List>
					</Combobox.Popup>
				</Combobox.Positioner>
			</Combobox.Portal>
		</Combobox.Root>
	);
}
