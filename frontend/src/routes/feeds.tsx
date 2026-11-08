import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { FeedsPage } from "#/features/feeds/FeedsPage";

export const Route = createFileRoute("/feeds")({
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: FeedsPage,
});
