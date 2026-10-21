import { createFileRoute, redirect } from "@tanstack/react-router";

/** A user address alone (/<user-slug>) has no page of its own. */
export const Route = createFileRoute("/$userSlug/")({
	beforeLoad: () => {
		throw redirect({ to: "/", replace: true });
	},
});
