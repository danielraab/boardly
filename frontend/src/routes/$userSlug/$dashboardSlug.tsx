import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { NotFound } from "#/components/NotFound";
import { useMe } from "#/features/auth/session";
import { dashboardViewQueryOptions } from "#/features/dashboards/api";
import { DashboardView } from "#/features/dashboards/DashboardView";

// Static routes such as /login/verify rank above this one, so the app's own
// pages always win over a user slug.
export const Route = createFileRoute("/$userSlug/$dashboardSlug")({
	component: DashboardRoute,
});

function DashboardRoute() {
	const { userSlug, dashboardSlug } = Route.useParams();
	const { data: me, isPending: mePending } = useMe();
	const view = useQuery({
		...dashboardViewQueryOptions(userSlug, dashboardSlug, me?.id ?? null),
		enabled: !mePending,
	});

	if (view.isError) return <NotFound />;
	if (!view.data) return null;
	return <DashboardView view={view.data} />;
}
