import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import {
	createNote,
	isVersionConflict,
	type Note,
	type NoteFields,
	noteQueryOptions,
	notesQueryKey,
	updateNote,
} from "./api";

const hasFields = (fields: object) => Object.keys(fields).length > 0;

/** How long after the last change a note is saved. */
export const autosaveDelay = 1000;

export type SaveStatus = "idle" | "saving" | "saved" | "failed" | "conflict";

/**
 * Saves a note's changes 1 second after the last one, and right away when
 * the page is left. Without a note, the first save creates it with all of
 * draft and calls onCreated. Only one request runs at a time; changes made
 * meanwhile are sent after it. A version conflict stops all saving.
 */
export function useAutosave({
	note,
	draft,
	onCreated,
}: {
	note: Note | null;
	/** All fields of the note as edited, for creating it. */
	draft: NoteFields;
	onCreated: (note: Note) => void;
}) {
	const queryClient = useQueryClient();
	const [status, setStatus] = useState<SaveStatus>("idle");
	const saved = useRef(note);
	const pending = useRef<Partial<NoteFields>>({});
	const running = useRef(false);
	const stopped = useRef(false);
	const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
	const draftRef = useRef(draft);
	draftRef.current = draft;
	const onCreatedRef = useRef(onCreated);
	onCreatedRef.current = onCreated;

	const flush = useCallback(
		async (keepalive = false) => {
			clearTimeout(timer.current);
			if (running.current || stopped.current || !hasFields(pending.current))
				return;
			const fields = pending.current;
			pending.current = {};
			running.current = true;
			setStatus("saving");
			try {
				const current = saved.current;
				const result = current
					? await updateNote(current.id, current.version, fields, { keepalive })
					: await createNote({ ...draftRef.current, ...fields });
				saved.current = result;
				queryClient.setQueryData(noteQueryOptions(result.id).queryKey, result);
				void queryClient.invalidateQueries({
					queryKey: notesQueryKey,
					exact: true,
				});
				if (!current) onCreatedRef.current(result);
				setStatus("saved");
			} catch (err) {
				if (isVersionConflict(err)) {
					stopped.current = true;
					setStatus("conflict");
				} else {
					pending.current = { ...fields, ...pending.current };
					setStatus("failed");
				}
				return;
			} finally {
				running.current = false;
			}
			if (hasFields(pending.current)) void flush(keepalive);
		},
		[queryClient],
	);

	/** Records changed fields and saves them after the delay. */
	const change = useCallback(
		(fields: Partial<NoteFields>) => {
			if (stopped.current) return;
			Object.assign(pending.current, fields);
			clearTimeout(timer.current);
			timer.current = setTimeout(() => void flush(), autosaveDelay);
		},
		[flush],
	);

	// Leaving the page or closing the tab sends what is left right away.
	useEffect(() => {
		const onPageHide = () => void flush(true);
		const onBeforeUnload = (event: BeforeUnloadEvent) => {
			if (!stopped.current && (hasFields(pending.current) || running.current)) {
				event.preventDefault();
			}
		};
		window.addEventListener("pagehide", onPageHide);
		window.addEventListener("beforeunload", onBeforeUnload);
		return () => {
			window.removeEventListener("pagehide", onPageHide);
			window.removeEventListener("beforeunload", onBeforeUnload);
			void flush(true);
		};
	}, [flush]);

	/** Drops unsaved changes and stops saving, e.g. once the note is deleted. */
	const discard = useCallback(() => {
		clearTimeout(timer.current);
		pending.current = {};
		stopped.current = true;
	}, []);

	return { status, change, retry: () => void flush(), discard };
}
