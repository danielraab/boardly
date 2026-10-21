import { createFileRoute, redirect } from "@tanstack/react-router";
import { useCallback } from "react";
import { meQueryOptions } from "#/features/auth/session";
import { DashboardsPage } from "#/features/dashboards/DashboardsPage";

type DashboardsSearch = { create?: boolean };

export const Route = createFileRoute("/dashboards")({
	// Anything but `true` must be set to undefined, not left out: the raw search
	// params are merged under the validated ones.
	validateSearch: (search: Record<string, unknown>): DashboardsSearch => ({
		create: search.create === true ? true : undefined,
	}),
	beforeLoad: async ({ context }) => {
		if (!(await context.queryClient.ensureQueryData(meQueryOptions))) {
			throw redirect({ to: "/login" });
		}
	},
	component: DashboardsRoute,
});

function DashboardsRoute() {
	const { create } = Route.useSearch();
	const navigate = Route.useNavigate();
	// Drop the param once the dialog is open, so a reload doesn't reopen it.
	const onCreateOpened = useCallback(
		() => void navigate({ search: {}, replace: true }),
		[navigate],
	);
	return (
		<DashboardsPage
			openCreate={create === true}
			onCreateOpened={onCreateOpened}
		/>
	);
}
