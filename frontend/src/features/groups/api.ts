import { queryOptions } from "@tanstack/react-query";
import { api } from "#/lib/api";

export type GroupRole = "admin" | "member";

export type Group = {
	id: string;
	name: string;
	/** The user's own role. */
	role: GroupRole;
	memberCount: number;
};

export type GroupMember = {
	id: string;
	slug: string;
	name: string | null;
	role: GroupRole;
};

/** An open invitation as the inviting side sees it. */
export type SentInvitation = {
	id: string;
	email: string;
	level?: string;
	role?: GroupRole;
	createdAt: string;
	expiresAt: string;
};

export type GroupDetails = Group & {
	members: GroupMember[];
	/** Only there for group admins. */
	invitations?: SentInvitation[];
};

export const MAX_GROUP_NAME = 100;

export const groupsQueryKey = ["groups"] as const;

export const groupsQueryOptions = queryOptions({
	queryKey: groupsQueryKey,
	queryFn: () => api<Group[]>("/api/groups"),
});

export function groupQueryOptions(id: string) {
	return queryOptions({
		queryKey: [...groupsQueryKey, id],
		queryFn: () => api<GroupDetails>(`/api/groups/${encodeURIComponent(id)}`),
		retry: false,
	});
}

export function createGroup(name: string) {
	return api<Group>("/api/groups", { method: "POST", json: { name } });
}

export function renameGroup(id: string, name: string) {
	return api<Group>(`/api/groups/${encodeURIComponent(id)}`, {
		method: "PATCH",
		json: { name },
	});
}

export function deleteGroup(id: string) {
	return api<void>(`/api/groups/${encodeURIComponent(id)}`, {
		method: "DELETE",
	});
}

function memberPath(groupId: string, userId: string) {
	return `/api/groups/${encodeURIComponent(groupId)}/members/${encodeURIComponent(userId)}`;
}

export function setMemberRole(
	groupId: string,
	userId: string,
	role: GroupRole,
) {
	return api<GroupDetails>(memberPath(groupId, userId), {
		method: "PATCH",
		json: { role },
	});
}

export function removeMember(groupId: string, userId: string) {
	return api<void>(memberPath(groupId, userId), { method: "DELETE" });
}

export function inviteToGroup(
	groupId: string,
	email: string,
	role: GroupRole,
	locale: string | undefined,
) {
	return api<SentInvitation>(
		`/api/groups/${encodeURIComponent(groupId)}/invitations`,
		{ method: "POST", json: { email, role, locale } },
	);
}

export function revokeGroupInvitation(groupId: string, invitationId: string) {
	return api<void>(
		`/api/groups/${encodeURIComponent(groupId)}/invitations/${encodeURIComponent(invitationId)}`,
		{ method: "DELETE" },
	);
}
