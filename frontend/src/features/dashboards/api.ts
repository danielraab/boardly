import { queryOptions } from "@tanstack/react-query";
import { ApiError, api } from "#/lib/api";

/** A dashboard's grid: column counts and the width below which mobile applies. */
export type Columns = {
	desktopColumns: number;
	mobileColumns: number;
	mobileBreakpoint: number;
};

export const defaultColumns: Columns = {
	desktopColumns: 6,
	mobileColumns: 2,
	mobileBreakpoint: 768,
};
export const maxColumns = 12;
export const breakpoints = [640, 768, 1024] as const;

/** A dashboard as its owner sees it. */
export type Dashboard = {
	id: string;
	slug: string;
	title: string;
	isDefault: boolean;
	isPublic: boolean;
} & Columns;

/** A widget's height: "auto" or a number of rows. */
export type SectionHeight = "auto" | number;

/** A section widget with its children, in order. */
export type Section = {
	id: string;
	kind: "section";
	width: number;
	title: string | null;
	border: boolean;
	height: SectionHeight;
	children: Widget[];
};

/** A content widget: its config belongs to its kind, data comes from the server. */
export type ContentWidget = {
	id: string;
	kind: string;
	width: number;
	height: SectionHeight;
	config: unknown;
	/** Resolved by the server for a lookup; never saved. */
	data?: unknown;
};

/** A node of a layout. Only sections have children. */
export type Widget = Section | ContentWidget;

export function isSection(widget: Widget): widget is Section {
	return widget.kind === "section";
}

/** A dashboard as anyone allowed to see it sees it. */
export type DashboardEntry = { slug: string; title: string };

/** A resolved dashboard address with the switcher entries and the layout. */
export type DashboardViewData = {
	/** Only set for the owner. */
	id?: string;
	/** Only set for the owner. */
	isDefault?: boolean;
	/** Only set for the owner. */
	isPublic?: boolean;
	dashboard: DashboardEntry;
	ownerSlug: string;
	isOwner: boolean;
	dashboards: DashboardEntry[];
	columns: Columns;
	layoutVersion: number;
	sections: Widget[];
};

/** A saved layout with its version. */
export type Layout = { version: number; sections: Widget[] };

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

/** A copy of a tree without the server-resolved data of its widgets. */
export function withoutData(tree: Widget[]): Widget[] {
	return tree.map((w) => {
		if (isSection(w)) return { ...w, children: withoutData(w.children) };
		const { data: _, ...rest } = w;
		return rest;
	});
}

/** Replaces a dashboard's layout; fails with 409 when version is outdated. */
export function saveLayout(id: string, version: number, sections: Widget[]) {
	return api<Layout>(`/api/dashboards/${encodeURIComponent(id)}/layout`, {
		method: "PUT",
		json: { version, sections: withoutData(sections) },
	});
}

/** Whether a layout save failed because the layout was changed elsewhere. */
export function isLayoutConflict(error: unknown) {
	return error instanceof ApiError && error.status === 409;
}

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
