import { createFileRoute, redirect } from "@tanstack/react-router";
import { LoginPage } from "#/features/auth/LoginPage";
import { meQueryOptions } from "#/features/auth/session";

export const Route = createFileRoute("/login/")({
	beforeLoad: async ({ context }) => {
		if (await context.queryClient.ensureQueryData(meQueryOptions)) {
			throw redirect({ to: "/" });
		}
	},
	component: LoginPage,
});
