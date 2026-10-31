import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { MediaPage } from "#/features/media/MediaPage";

export const Route = createFileRoute("/media")({
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: MediaPage,
});
