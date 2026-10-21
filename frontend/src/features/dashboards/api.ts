import { queryOptions } from "@tanstack/react-query";
import { api } from "#/lib/api";

/** A dashboard as its owner sees it. */
export type Dashboard = {
	id: string;
	slug: string;
	title: string;
	isDefault: boolean;
	isPublic: boolean;
};

/** A dashboard as anyone allowed to see it sees it. */
export type DashboardEntry = { slug: string; title: string };

/** A resolved dashboard address with the switcher entries. */
export type DashboardViewData = {
	dashboard: DashboardEntry;
	ownerSlug: string;
	isOwner: boolean;
	dashboards: DashboardEntry[];
};

export const dashboardsQueryKey = ["dashboards"] as const;
export const dashboardViewQueryKey = ["dashboard-view"] as const;

export const dashboardsQueryOptions = queryOptions({
	queryKey: dashboardsQueryKey,
	queryFn: () => api<Dashboard[]>("/api/dashboards"),
});

/** The lookup of an address; viewerId keeps owner and visitor results apart. */
export function dashboardViewQueryOptions(
	userSlug: string,
	dashboardSlug: string,
	viewerId: string | null,
) {
	const user = userSlug.toLowerCase();
	const dashboard = dashboardSlug.toLowerCase();
	return queryOptions({
		queryKey: [...dashboardViewQueryKey, user, dashboard, viewerId],
		queryFn: () =>
			api<DashboardViewData>(
				`/api/u/${encodeURIComponent(user)}/${encodeURIComponent(dashboard)}`,
			),
		retry: false,
	});
}

export type DashboardFields = Omit<Dashboard, "id">;

export function createDashboard(fields: DashboardFields) {
	return api<Dashboard>("/api/dashboards", { method: "POST", json: fields });
}

export function updateDashboard(id: string, patch: Partial<DashboardFields>) {
	return api<Dashboard>(`/api/dashboards/${encodeURIComponent(id)}`, {
		method: "PATCH",
		json: patch,
	});
}

export function deleteDashboard(id: string) {
	return api<void>(`/api/dashboards/${encodeURIComponent(id)}`, {
		method: "DELETE",
	});
}
