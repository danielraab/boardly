import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";

export const Route = createFileRoute("/notes")({
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: Outlet,
});
