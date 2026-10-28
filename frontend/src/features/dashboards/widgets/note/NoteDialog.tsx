import { Dialog } from "@base-ui/react/dialog";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { dialogBackdrop } from "#/components/dialogStyles";
import { noteQueryOptions } from "#/features/notes/api";
import { NoteEditorPage } from "#/features/notes/NoteEditorPage";

const popup =
	"fixed top-1/2 left-1/2 z-40 max-h-[calc(100dvh-2rem)] w-[48rem] max-w-[calc(100vw-2rem)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] transition-[opacity,scale] duration-150 data-[ending-style]:scale-95 data-[ending-style]:opacity-0 data-[starting-style]:scale-95 data-[starting-style]:opacity-0 motion-reduce:transition-none";

/**
 * The editor of a note in a modal over the dashboard. onChanged is called
 * after every save, also one that lands after closing, and onDeleted once the
 * note is deleted.
 */
export function NoteDialog({
	noteId,
	name,
	open,
	onClose,
	onChanged,
	onDeleted,
}: {
	noteId: string;
	name: string;
	open: boolean;
	onClose: () => void;
	onChanged: () => void;
	onDeleted: () => void;
}) {
	return (
		<Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={popup} aria-label={name}>
					<NoteDialogEditor
						noteId={noteId}
						onClose={onClose}
						onChanged={onChanged}
						onDeleted={onDeleted}
					/>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/** Loads the note fresh on every opening, so the editor starts from the server's version. */
function NoteDialogEditor({
	noteId,
	onClose,
	onChanged,
	onDeleted,
}: {
	noteId: string;
	onClose: () => void;
	onChanged: () => void;
	onDeleted: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	// A reload after a conflict starts a fresh editor.
	const [reloads, setReloads] = useState(0);
	const note = useQuery({
		...noteQueryOptions(noteId),
		staleTime: Number.POSITIVE_INFINITY,
		refetchOnMount: "always",
	});

	if (note.isError) {
		return (
			<p role="alert" className="rounded-2xl bg-surface p-6 text-sm">
				{t("common.genericError")}
			</p>
		);
	}
	if (!note.data || !note.isFetchedAfterMount) return null;
	return (
		<NoteEditorPage
			key={reloads}
			note={note.data}
			onCreated={() => {}}
			onReload={async () => {
				await queryClient.fetchQuery({
					...noteQueryOptions(noteId),
					staleTime: 0,
				});
				setReloads((n) => n + 1);
			}}
			onSaved={onChanged}
			onClose={onClose}
			onDeleted={onDeleted}
		/>
	);
}
