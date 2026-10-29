import { Combobox } from "@base-ui/react/combobox";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { menuItem, menuPopup } from "#/components/menuStyles";
import type { Note, NoteCategory } from "#/features/notes/api";
import { NoteChoiceLabel } from "../NoteChoiceLabel";
import { maxSelectedNotes, noteChoices } from "./selected";

/**
 * A search over the owner's notes that aren't chosen yet. Choosing one calls
 * onAdd with its ID and clears the search. Disabled once maxSelectedNotes
 * are chosen.
 */
export function NoteSearch({
	id,
	notes,
	categories,
	chosen,
	onAdd,
}: {
	id: string;
	notes: Note[];
	categories: NoteCategory[];
	chosen: string[];
	onAdd: (noteId: string) => void;
}) {
	const { t } = useTranslation();
	const [input, setInput] = useState("");
	const full = chosen.length >= maxSelectedNotes;
	const choices = noteChoices(notes, categories, chosen, input);
	const byID = new Map(choices.map((c) => [c.note.id, c]));

	return (
		<Combobox.Root
			items={choices.map((c) => c.note.id)}
			filter={null}
			value={null}
			onValueChange={(noteId: string | null) => {
				if (noteId === null) return;
				onAdd(noteId);
				setInput("");
			}}
			inputValue={input}
			onInputValueChange={setInput}
			disabled={full}
		>
			<Combobox.Input
				id={id}
				placeholder={t("dashboards.widgets.selectedNotes.addPlaceholder")}
				aria-describedby={full ? `${id}-full` : undefined}
				className={textInput}
			/>
			{full && (
				<p id={`${id}-full`} className="mt-1 text-sm text-muted">
					{t("dashboards.widgets.selectedNotes.full", {
						max: maxSelectedNotes,
					})}
				</p>
			)}
			<Combobox.Portal>
				<Combobox.Positioner sideOffset={4} className="z-50">
					<Combobox.Popup
						className={`${menuPopup} max-h-64 w-[var(--anchor-width)] overflow-y-auto`}
					>
						<Combobox.Empty className="px-3 py-2 text-muted empty:hidden">
							{t("dashboards.widgets.selectedNotes.noMatches")}
						</Combobox.Empty>
						<Combobox.List>
							{(noteId: string) => {
								const choice = byID.get(noteId);
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
