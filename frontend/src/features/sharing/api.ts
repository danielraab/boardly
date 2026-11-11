import { queryOptions } from "@tanstack/react-query";
import type { SentInvitation } from "#/features/groups/api";
import { api } from "#/lib/api";

/** An entry of a dashboard's access list: `user` or `group` is set. */
export type Share = {
	id: string;
	level: string;
	user?: { id: string; slug: string; name: string | null };
	group?: { id: string; name: string };
};

export type ShareList = { shares: Share[]; invitations: SentInvitation[] };

export type SharedDashboard = {
	ownerSlug: string;
	ownerName: string;
	slug: string;
	title: string;
	level: string;
};

export const sharesQueryKey = ["shares"] as const;
export const sharedWithMeQueryKey = ["shared-with-me"] as const;

export function sharesQueryOptions(dashboardId: string) {
	return queryOptions({
		queryKey: [...sharesQueryKey, dashboardId],
		queryFn: () =>
			api<ShareList>(
				`/api/shares/dashboard/${encodeURIComponent(dashboardId)}`,
			),
	});
}

export const sharedWithMeQueryOptions = queryOptions({
	queryKey: sharedWithMeQueryKey,
	queryFn: () => api<{ dashboards: SharedDashboard[] }>("/api/shared-with-me"),
});

export function shareWithGroup(dashboardId: string, groupId: string) {
	return api<Share>(
		`/api/shares/dashboard/${encodeURIComponent(dashboardId)}`,
		{
			method: "POST",
			json: { groupId, level: "read" },
		},
	);
}

export function removeShare(dashboardId: string, shareId: string) {
	return api<void>(
		`/api/shares/dashboard/${encodeURIComponent(dashboardId)}/${encodeURIComponent(shareId)}`,
		{ method: "DELETE" },
	);
}

export function inviteToDashboard(
	dashboardId: string,
	email: string,
	locale: string | undefined,
) {
	return api<SentInvitation>(
		`/api/shares/dashboard/${encodeURIComponent(dashboardId)}/invitations`,
		{ method: "POST", json: { email, level: "read", locale } },
	);
}

export function revokeDashboardInvitation(
	dashboardId: string,
	invitationId: string,
) {
	return api<void>(
		`/api/shares/dashboard/${encodeURIComponent(dashboardId)}/invitations/${encodeURIComponent(invitationId)}`,
		{ method: "DELETE" },
	);
}
