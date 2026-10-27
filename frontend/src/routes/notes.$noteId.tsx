import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { NotFound } from "#/components/NotFound";
import { noteQueryOptions } from "#/features/notes/api";
import { NoteEditorPage } from "#/features/notes/NoteEditorPage";

/** The address of a note that doesn't exist yet. */
const newNote = "new";

/**
 * Notes created on /notes/new, by ID: their editor keeps the key of the new
 * note, so replacing the address with the note's own doesn't remount it.
 */
const createdKeys = new Map<string, string>();

export const Route = createFileRoute("/notes/$noteId")({
	validateSearch: (search: Record<string, unknown>): { category?: string } => ({
		category:
			typeof search.category === "string" || typeof search.category === "number"
				? String(search.category) || undefined
				: undefined,
	}),
	component: NoteRoute,
});

function NoteRoute() {
	const { noteId } = Route.useParams();
	const { category } = Route.useSearch();
	const navigate = Route.useNavigate();
	const queryClient = useQueryClient();
	// A reload after a conflict starts a fresh editor.
	const [reloads, setReloads] = useState(0);
	const isNew = noteId === newNote;
	const note = useQuery({
		...noteQueryOptions(noteId),
		enabled: !isNew,
		staleTime: Number.POSITIVE_INFINITY,
	});

	if (!isNew && note.isError) return <NotFound />;
	if (!isNew && !note.data) return null;

	const key = `${createdKeys.get(noteId) ?? noteId}-${reloads}`;
	return (
		<NoteEditorPage
			key={key}
			note={isNew ? null : (note.data ?? null)}
			initialCategory={isNew ? (category ?? null) : null}
			onCreated={(created) => {
				createdKeys.set(created.id, noteId);
				void navigate({
					to: "/notes/$noteId",
					params: { noteId: created.id },
					search: {},
					replace: true,
				});
			}}
			onReload={async () => {
				await queryClient.fetchQuery({
					...noteQueryOptions(noteId),
					staleTime: 0,
				});
				setReloads((n) => n + 1);
			}}
		/>
	);
}
