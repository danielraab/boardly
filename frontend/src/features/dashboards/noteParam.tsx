import { useQueryClient } from "@tanstack/react-query";
import {
	type HistoryState,
	useLocation,
	useNavigate,
	useRouter,
} from "@tanstack/react-router";
import { useCallback, useState } from "react";
import { dashboardViewQueryKey } from "./api";
import type { DashboardSearch } from "./DashboardView";
import { NoteDialog } from "./widgets/note/NoteDialog";

/** Marks the history entry that opening a note pushed. */
type NoteHistoryState = HistoryState & { noteOpened?: boolean };

/**
 * Opens a note's editor modal on the current dashboard by adding `note` to
 * its address as a new history entry, so back closes it.
 */
export function useOpenNote() {
	const navigate = useNavigate();
	return (noteId: string) =>
		void navigate({
			to: ".",
			search: (prev: DashboardSearch) => ({ ...prev, note: noteId }),
			state: (prev) => ({ ...prev, noteOpened: true }) as NoteHistoryState,
		});
}

/** Removes `note` from the address without a new history entry. */
export function useDropNote() {
	const navigate = useNavigate();
	return useCallback(
		() =>
			void navigate({
				to: ".",
				search: (prev: DashboardSearch) => ({ ...prev, note: undefined }),
				replace: true,
			}),
		[navigate],
	);
}

/**
 * The editor modal of the note in the address. Closing goes back when opening
 * pushed the entry, so no entry stays behind, and replaces the address
 * otherwise (after a reload or from a link).
 */
export function DashboardNoteDialog({ noteId }: { noteId?: string }) {
	const router = useRouter();
	const location = useLocation();
	const queryClient = useQueryClient();
	const dropNote = useDropNote();
	// The last note stays while the modal plays its closing animation.
	const [shown, setShown] = useState(noteId);
	if (noteId !== undefined && noteId !== shown) setShown(noteId);
	if (shown === undefined) return null;

	const close = () => {
		if ((location.state as NoteHistoryState).noteOpened) router.history.back();
		else dropNote();
	};
	const refresh = () =>
		void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
	return (
		<NoteDialog
			noteId={shown}
			open={noteId !== undefined}
			onClose={close}
			onMissing={dropNote}
			onChanged={refresh}
			onDeleted={() => {
				close();
				refresh();
			}}
		/>
	);
}
