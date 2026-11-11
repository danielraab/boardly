import { createFileRoute, redirect } from "@tanstack/react-router";
import { meQueryOptions } from "#/features/auth/session";
import { GroupsPage } from "#/features/groups/GroupsPage";

type GroupsSearch = { group?: string };

export const Route = createFileRoute("/groups")({
	// The open group is a search parameter: a path below /groups would be the
	// address of a dashboard of a user who already has that slug.
	validateSearch: (search: Record<string, unknown>): GroupsSearch => ({
		group: typeof search.group === "string" ? search.group : undefined,
	}),
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: GroupsRoute,
});

function GroupsRoute() {
	const { group } = Route.useSearch();
	return <GroupsPage groupId={group} />;
}
