import { queryOptions } from "@tanstack/react-query";
import type { GroupRole } from "#/features/groups/api";
import { api } from "#/lib/api";

export type InvitationPreview = {
	type: "dashboard" | "group";
	title: string;
	inviterName: string;
	level?: string;
	role?: GroupRole;
};

export type OpenInvitation = InvitationPreview & {
	id: string;
	expiresAt: string;
};

export type AcceptedInvitation = {
	type: "dashboard" | "group";
	ownerSlug?: string;
	slug?: string;
	groupId?: string;
};

export const openInvitationsQueryKey = ["invitations"] as const;

export const openInvitationsQueryOptions = queryOptions({
	queryKey: openInvitationsQueryKey,
	queryFn: () => api<OpenInvitation[]>("/api/invitations"),
});

export function invitationPreviewQueryOptions(token: string) {
	return queryOptions({
		queryKey: ["invitation-preview", token],
		queryFn: () =>
			api<InvitationPreview>("/api/invitations/preview", {
				method: "POST",
				json: { token },
			}),
		retry: false,
		// The token is used up by accepting; never reload it behind the page.
		staleTime: Number.POSITIVE_INFINITY,
		refetchOnWindowFocus: false,
	});
}

export function acceptInvitation(
	from: { token: string } | { invitationId: string },
) {
	return api<AcceptedInvitation>("/api/invitations/accept", {
		method: "POST",
		json: from,
	});
}
