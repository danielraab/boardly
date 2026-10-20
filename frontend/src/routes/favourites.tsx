import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { FavouritesPage } from "#/features/favourites/FavouritesPage";

export const Route = createFileRoute("/favourites")({
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: FavouritesPage,
});
