import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { InvitationPage } from "#/features/invitations/InvitationPage";

type InvitationSearch = { token?: string };

export const Route = createFileRoute("/invitation")({
	validateSearch: (search: Record<string, unknown>): InvitationSearch => ({
		token: typeof search.token === "string" ? search.token : undefined,
	}),
	beforeLoad: async ({ context, location }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login", search: { redirect: location.href } });
		}
	},
	component: InvitationRoute,
});

function InvitationRoute() {
	const { token = "" } = Route.useSearch();
	return <InvitationPage token={token} />;
}
