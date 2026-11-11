import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type Dashboard, defaultColumns } from "#/features/dashboards/api";
import { apiError, mockApi, type Routes, signedIn } from "#/test/api";
import { viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import type { ShareList } from "./api";

const work: Dashboard = {
	id: "d1",
	slug: "work",
	title: "Work",
	isDefault: false,
	isPublic: false,
	...defaultColumns,
};
const empty: ShareList = { shares: [], invitations: [] };
const family = { id: "g1", name: "Family", role: "member", memberCount: 2 };

function setup(routes: Routes = {}) {
	return mockApi({
		"GET /api/me": signedIn(),
		"GET /api/dashboards": { body: [work] },
		"GET /api/shares/dashboard/d1": { body: empty },
		"GET /api/groups": { body: [family] },
		...routes,
	});
}

async function openDialog(user: ReturnType<typeof renderApp>["user"]) {
	await user.click(await screen.findByRole("button", { name: "Share Work" }));
	return screen.findByRole("dialog", { name: "Share Work" });
}

describe("share dialog", () => {
	it("invites an address and lists it as invited", async () => {
		let list = empty;
		const api = setup({
			"GET /api/shares/dashboard/d1": () => ({ body: list }),
			"POST /api/shares/dashboard/d1/invitations": ({ body }) => {
				const { email } = body as { email: string };
				list = {
					shares: [],
					invitations: [
						{
							id: "i1",
							email,
							level: "read",
							createdAt: "2026-03-01T12:00:00.000Z",
							expiresAt: "2026-03-15T12:00:00.000Z",
						},
					],
				};
				return { status: 201, body: list.invitations[0] };
			},
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		expect(
			within(dialog).getByText(/Registered users get a link to accept/),
		).toBeVisible();
		await user.type(
			within(dialog).getByLabelText("Invite by email address"),
			"bob@example.org",
		);
		await user.click(within(dialog).getByRole("button", { name: "Invite" }));

		expect(await within(dialog).findByText("bob@example.org")).toBeVisible();
		expect(within(dialog).getByText("Invited")).toBeVisible();
		expect(
			api.callsTo("POST", "/api/shares/dashboard/d1/invitations")[0].body,
		).toEqual({ email: "bob@example.org", level: "read", locale: "en" });
	});

	it.each([
		["invalid_email", 400, "This email address isn't valid."],
		["already_invited", 409, "This address is already invited."],
		["cannot_invite_self", 400, "You can't invite your own address."],
		["rate_limited", 429, "You sent too many invitations. Try again later."],
	])("shows the server error %s", async (code, status, text) => {
		setup({
			"POST /api/shares/dashboard/d1/invitations": apiError(status, code),
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		await user.type(
			within(dialog).getByLabelText("Invite by email address"),
			"x@example.org",
		);
		await user.click(within(dialog).getByRole("button", { name: "Invite" }));
		expect(await within(dialog).findByRole("alert")).toHaveTextContent(text);
	});

	it("saves Public right away", async () => {
		const api = setup({
			"PATCH /api/dashboards/d1": { body: { ...work, isPublic: true } },
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		await user.click(within(dialog).getByLabelText("Public"));
		await waitFor(() =>
			expect(api.callsTo("PATCH", "/api/dashboards/d1")[0]?.body).toEqual({
				isPublic: true,
			}),
		);
		expect(within(dialog).getByLabelText("Public")).toBeChecked();
	});

	it("shares with a group and shows the hint once one is selected", async () => {
		let list = empty;
		const api = setup({
			"GET /api/shares/dashboard/d1": () => ({ body: list }),
			"POST /api/shares/dashboard/d1": () => {
				list = {
					shares: [
						{
							id: "s1",
							level: "read",
							group: { id: "g1", name: "Family" },
						},
					],
					invitations: [],
				};
				return { status: 201, body: list.shares[0] };
			},
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		const hint = /Group admins decide who is in a group/;
		expect(within(dialog).queryByText(hint)).toBeNull();
		await within(dialog).findByRole("option", { name: "Family" });
		await user.selectOptions(
			within(dialog).getByLabelText("Share with a group"),
			"g1",
		);
		expect(within(dialog).getByText(hint)).toBeVisible();
		await user.click(within(dialog).getByRole("button", { name: "Share" }));

		expect(await within(dialog).findByText("Can view")).toBeVisible();
		expect(api.callsTo("POST", "/api/shares/dashboard/d1")[0].body).toEqual({
			groupId: "g1",
			level: "read",
		});
		expect(within(dialog).getByText(hint)).toBeVisible();
	});

	it("links to the groups page without groups", async () => {
		setup({ "GET /api/groups": { body: [] } });
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		expect(
			await within(dialog).findByRole("link", { name: "Go to groups" }),
		).toHaveAttribute("href", "/groups");
	});

	it("removes a person without showing an address", async () => {
		let list: ShareList = {
			shares: [
				{
					id: "s1",
					level: "read",
					user: { id: "u2", slug: "bob", name: "Bob" },
				},
			],
			invitations: [],
		};
		const api = setup({
			"GET /api/shares/dashboard/d1": () => ({ body: list }),
			"DELETE /api/shares/dashboard/d1/s1": () => {
				list = empty;
				return { status: 204 };
			},
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		expect(await within(dialog).findByText("/bob")).toBeVisible();
		expect(dialog.textContent).not.toContain("@");
		await user.click(
			within(dialog).getByRole("button", { name: "Remove Bob" }),
		);
		await waitFor(() => expect(within(dialog).queryByText("/bob")).toBeNull());
		expect(api.callsTo("DELETE", "/api/shares/dashboard/d1/s1")).toHaveLength(
			1,
		);
	});

	it("revokes an invitation", async () => {
		const api = setup({
			"GET /api/shares/dashboard/d1": {
				body: {
					shares: [],
					invitations: [
						{
							id: "i1",
							email: "bob@example.org",
							level: "read",
							createdAt: "2026-03-01T12:00:00.000Z",
							expiresAt: "2026-03-15T12:00:00.000Z",
						},
					],
				},
			},
			"DELETE /api/shares/dashboard/d1/invitations/i1": { status: 204 },
		});
		const { user } = renderApp("/dashboards");
		const dialog = await openDialog(user);
		await user.click(
			await within(dialog).findByRole("button", {
				name: "Revoke the invitation of bob@example.org",
			}),
		);
		await waitFor(() =>
			expect(
				api.callsTo("DELETE", "/api/shares/dashboard/d1/invitations/i1"),
			).toHaveLength(1),
		);
	});

	it("is in the actions menu for the owner", async () => {
		setup({
			"GET /api/u/ann/home": { body: viewData([]) },
			"GET /api/shares/dashboard/d1": { body: empty },
		});
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Dashboard actions" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Share" }));
		expect(
			await screen.findByRole("dialog", { name: "Share Home" }),
		).toBeVisible();
	});

	it("isn't offered to someone the dashboard is shared with", async () => {
		mockApi({
			"GET /api/me": signedIn({ id: "u2", slug: "bob" }),
			"GET /api/u/ann/home": {
				body: viewData([], {
					id: undefined,
					isDefault: undefined,
					isPublic: undefined,
					isOwner: false,
				}),
			},
		});
		renderApp("/ann/home");
		expect(await screen.findByRole("heading", { name: "Home" })).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Dashboard actions" }),
		).toBeNull();
	});
});

describe("shared with me page", () => {
	const shared = {
		dashboards: [
			{
				ownerSlug: "ann",
				ownerName: "Ann",
				slug: "work",
				title: "Work",
				level: "read",
			},
		],
	};
	const invitation = {
		id: "i1",
		type: "dashboard",
		title: "Trips",
		inviterName: "Carol",
		level: "read",
		expiresAt: "2026-03-15T12:00:00.000Z",
	};

	it("sends signed-out visitors to /login", async () => {
		const { router } = renderApp("/shared");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("shows the empty state", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/shared-with-me": { body: { dashboards: [] } },
			"GET /api/invitations": { body: [] },
		});
		renderApp("/shared");
		expect(
			await screen.findByText("Nothing shared with you yet"),
		).toBeVisible();
	});

	it("lists shared dashboards with their owner", async () => {
		mockApi({
			"GET /api/me": signedIn({ id: "u2" }),
			"GET /api/shared-with-me": { body: shared },
			"GET /api/invitations": { body: [] },
		});
		renderApp("/shared");
		expect(await screen.findByRole("link", { name: "Work" })).toHaveAttribute(
			"href",
			"/ann/work",
		);
		expect(screen.getByText("Ann")).toBeVisible();
	});

	it("accepts an open invitation on the page", async () => {
		let open = [invitation];
		let list = { dashboards: [] as typeof shared.dashboards };
		const api = mockApi({
			"GET /api/me": signedIn({ id: "u2" }),
			"GET /api/shared-with-me": () => ({ body: list }),
			"GET /api/invitations": () => ({ body: open }),
			"POST /api/invitations/accept": () => {
				open = [];
				list = {
					dashboards: [
						{ ...shared.dashboards[0], title: "Trips", slug: "trips" },
					],
				};
				return { body: { type: "dashboard", ownerSlug: "ann", slug: "trips" } };
			},
		});
		const { user } = renderApp("/shared");
		expect(
			await screen.findByText(
				'Carol invited you to view the dashboard "Trips".',
			),
		).toBeVisible();
		expect(screen.queryByRole("link", { name: "Trips" })).toBeNull();
		await user.click(
			screen.getByRole("button", { name: "Accept the invitation to Trips" }),
		);
		expect(await screen.findByRole("link", { name: "Trips" })).toBeVisible();
		expect(screen.queryByText(/invited you to view/)).toBeNull();
		expect(api.callsTo("POST", "/api/invitations/accept")[0].body).toEqual({
			invitationId: "i1",
		});
	});
});
