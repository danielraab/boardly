import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import type { GroupDetails } from "./api";

const family: GroupDetails = {
	id: "g1",
	name: "Family",
	role: "admin",
	memberCount: 2,
	members: [
		{ id: "u1", slug: "ann", name: "Ann", role: "admin" },
		{ id: "u2", slug: "bob", name: null, role: "member" },
	],
	invitations: [
		{
			id: "i1",
			email: "new@example.org",
			role: "member",
			createdAt: "2026-03-01T12:00:00.000Z",
			expiresAt: "2026-03-15T12:00:00.000Z",
		},
	],
};
const asMember: GroupDetails = {
	...family,
	role: "member",
	invitations: undefined,
};

function groupApi(details: GroupDetails, extra = {}) {
	return mockApi({
		"GET /api/me": signedIn({ id: details.role === "admin" ? "u1" : "u2" }),
		"GET /api/groups/g1": { body: details },
		...extra,
	});
}

describe("groups page", () => {
	it("sends signed-out visitors to /login", async () => {
		const { router } = renderApp("/groups");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("shows an empty state with a create action", async () => {
		mockApi({ "GET /api/me": signedIn(), "GET /api/groups": { body: [] } });
		renderApp("/groups");
		expect(await screen.findByText("No groups yet")).toBeVisible();
		expect(screen.getByRole("button", { name: "New group" })).toBeVisible();
	});

	it("lists groups with member count and role", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/groups": {
				body: [{ id: "g1", name: "Family", role: "admin", memberCount: 2 }],
			},
		});
		renderApp("/groups");
		expect(await screen.findByRole("link", { name: "Family" })).toHaveAttribute(
			"href",
			"/groups?group=g1",
		);
		expect(screen.getByText("2 members · Group admin")).toBeVisible();
	});

	it("creates a group", async () => {
		let list: unknown[] = [];
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/groups": () => ({ body: list }),
			"POST /api/groups": ({ body }) => {
				list = [
					{
						id: "g2",
						name: (body as { name: string }).name,
						role: "admin",
						memberCount: 1,
					},
				];
				return { status: 201, body: list[0] };
			},
		});
		const { user } = renderApp("/groups");
		await user.click(await screen.findByRole("button", { name: "New group" }));
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "Family");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(await screen.findByRole("link", { name: "Family" })).toBeVisible();
		expect(api.callsTo("POST", "/api/groups")[0].body).toEqual({
			name: "Family",
		});
	});

	it("shows a group admin the management actions", async () => {
		groupApi(family);
		renderApp("/groups?group=g1");
		expect(
			await screen.findByRole("heading", { name: "Family" }),
		).toBeVisible();
		for (const name of ["Rename", "Delete group", "Leave group", "Invite"]) {
			expect(screen.getByRole("button", { name })).toBeVisible();
		}
		expect(screen.getByText("new@example.org")).toBeVisible();
		expect(screen.getByText("Invited")).toBeVisible();
		expect(screen.getByRole("button", { name: "Remove bob" })).toBeVisible();
		expect(screen.getByLabelText("Role of bob")).toHaveValue("member");
		expect(screen.queryByRole("button", { name: "Remove Ann" })).toBeNull();
	});

	it("shows a group member only the members and Leave", async () => {
		groupApi(asMember);
		renderApp("/groups?group=g1");
		expect(
			await screen.findByRole("button", { name: "Leave group" }),
		).toBeVisible();
		expect(screen.getByText("/ann")).toBeVisible();
		expect(screen.getByText("Group admin")).toBeVisible();
		for (const name of ["Rename", "Delete group", "Invite", "Remove bob"]) {
			expect(screen.queryByRole("button", { name })).toBeNull();
		}
		expect(screen.queryByText("new@example.org")).toBeNull();
		expect(screen.queryByLabelText("Role of bob")).toBeNull();
	});

	it("invites an address with a role", async () => {
		const api = groupApi(family, {
			"POST /api/groups/g1/invitations": { status: 201, body: {} },
		});
		const { user } = renderApp("/groups?group=g1");
		await user.type(
			await screen.findByLabelText("Invite by email address"),
			"cara@example.org",
		);
		await user.selectOptions(
			screen.getByLabelText("Role of the invited person"),
			"admin",
		);
		await user.click(screen.getByRole("button", { name: "Invite" }));
		await waitFor(() =>
			expect(api.callsTo("POST", "/api/groups/g1/invitations")[0].body).toEqual(
				{
					email: "cara@example.org",
					role: "admin",
					locale: "en",
				},
			),
		);
		await waitFor(() =>
			expect(screen.getByLabelText("Invite by email address")).toHaveValue(""),
		);
	});

	it("revokes an open invitation", async () => {
		const api = groupApi(family, {
			"DELETE /api/groups/g1/invitations/i1": { status: 204 },
		});
		const { user } = renderApp("/groups?group=g1");
		await user.click(
			await screen.findByRole("button", {
				name: "Revoke the invitation of new@example.org",
			}),
		);
		await waitFor(() =>
			expect(
				api.callsTo("DELETE", "/api/groups/g1/invitations/i1"),
			).toHaveLength(1),
		);
	});

	it("changes a member's role", async () => {
		const api = groupApi(family, {
			"PATCH /api/groups/g1/members/u2": { body: family },
		});
		const { user } = renderApp("/groups?group=g1");
		await user.selectOptions(await screen.findByLabelText("Role of bob"), [
			"admin",
		]);
		await waitFor(() =>
			expect(api.callsTo("PATCH", "/api/groups/g1/members/u2")[0].body).toEqual(
				{ role: "admin" },
			),
		);
	});

	it("says that the last admin can't leave and keeps them in the group", async () => {
		groupApi(family, {
			"DELETE /api/groups/g1/members/u1": apiError(409, "last_admin"),
		});
		const { user, router } = renderApp("/groups?group=g1");
		await user.click(
			await screen.findByRole("button", { name: "Leave group" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(
			within(dialog).getByRole("button", { name: "Leave group" }),
		);
		expect(await within(dialog).findByRole("alert")).toHaveTextContent(
			"This group needs another group admin first.",
		);
		expect(router.state.location.search).toEqual({ group: "g1" });
	});

	it("returns to the list after leaving", async () => {
		const api = groupApi(asMember, {
			"DELETE /api/groups/g1/members/u2": { status: 204 },
			"GET /api/groups": { body: [] },
		});
		const { user, router } = renderApp("/groups?group=g1");
		await user.click(
			await screen.findByRole("button", { name: "Leave group" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(
			within(dialog).getByRole("button", { name: "Leave group" }),
		);
		await waitFor(() => expect(router.state.location.href).toBe("/groups"));
		expect(api.callsTo("DELETE", "/api/groups/g1/members/u2")).toHaveLength(1);
	});

	it("deletes a group after confirmation", async () => {
		const api = groupApi(family, {
			"DELETE /api/groups/g1": { status: 204 },
			"GET /api/groups": { body: [] },
		});
		const { user, router } = renderApp("/groups?group=g1");
		await user.click(
			await screen.findByRole("button", { name: "Delete group" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(api.callsTo("DELETE", "/api/groups/g1")).toHaveLength(0);
		await user.click(
			within(dialog).getByRole("button", { name: "Delete group" }),
		);
		await waitFor(() => expect(router.state.location.href).toBe("/groups"));
	});

	it("says when a group doesn't exist for the user", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/groups/g1": apiError(404, "not_found"),
		});
		renderApp("/groups?group=g1");
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This group doesn't exist or you aren't a member of it.",
		);
	});
});
