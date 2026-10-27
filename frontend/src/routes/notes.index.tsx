import { createFileRoute } from "@tanstack/react-router";
import { NotesPage, type NotesSearch } from "#/features/notes/NotesPage";

// Search params are parsed as JSON, so an ID like "123" arrives as a number.
const text = (value: unknown) =>
	typeof value === "string" || typeof value === "number"
		? String(value) || undefined
		: undefined;

export const Route = createFileRoute("/notes/")({
	// Invalid values must be set to undefined, not left out: the raw search
	// params are merged under the validated ones.
	validateSearch: (search: Record<string, unknown>): NotesSearch => ({
		category: text(search.category),
	}),
	component: NotesRoute,
});

function NotesRoute() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	return (
		<NotesPage
			search={search}
			onSearchChange={(next) => void navigate({ search: next })}
		/>
	);
}
