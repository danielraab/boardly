import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import {
	FavouritesPage,
	type FavouritesSearch,
} from "#/features/favourites/FavouritesPage";
import { normaliseDomain } from "#/features/favourites/tree";

// Search params are parsed as JSON, so a search for "123" arrives as a number.
const text = (value: unknown) =>
	typeof value === "string" || typeof value === "number"
		? String(value) || undefined
		: undefined;

export const Route = createFileRoute("/favourites")({
	// Invalid values must be set to undefined, not left out: the raw search
	// params are merged under the validated ones.
	validateSearch: (search: Record<string, unknown>): FavouritesSearch => {
		const tags = Array.isArray(search.tags)
			? [...new Set(search.tags.map(text).filter((t) => t !== undefined))]
			: [];
		return {
			q: text(search.q),
			tags: tags.length > 0 ? tags : undefined,
			domain: normaliseDomain(text(search.domain)),
			folder: text(search.folder),
		};
	},
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: FavouritesRoute,
});

function FavouritesRoute() {
	const search = Route.useSearch();
	const navigate = Route.useNavigate();
	return (
		<FavouritesPage
			search={search}
			onSearchChange={(patch, replace) =>
				void navigate({
					search: (prev) => ({ ...prev, ...patch }),
					replace,
				})
			}
		/>
	);
}
