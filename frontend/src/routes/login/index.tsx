import { createFileRoute, redirect } from "@tanstack/react-router";
import { LoginPage } from "#/features/auth/LoginPage";
import { safeRedirect } from "#/features/auth/redirect";
import { meQueryOptions } from "#/features/auth/session";

type LoginSearch = { redirect?: string };

export const Route = createFileRoute("/login/")({
	// Where to continue after signing in on this page; only local paths.
	validateSearch: (search: Record<string, unknown>): LoginSearch => ({
		redirect: safeRedirect(search.redirect),
	}),
	beforeLoad: async ({ context, search }) => {
		if (await context.queryClient.ensureQueryData(meQueryOptions)) {
			throw redirect({ href: search.redirect ?? "/" });
		}
	},
	component: LoginRoute,
});

function LoginRoute() {
	const { redirect: next = "/" } = Route.useSearch();
	return <LoginPage next={next} />;
}
