import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";

const preview = {
	type: "dashboard",
	title: "Work",
	inviterName: "Ann",
	level: "read",
};

describe("invitation page", () => {
	it("sends signed-out visitors to /login and back", async () => {
		const { router } = renderApp("/invitation?token=abc");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		expect(router.state.location.search).toEqual({
			redirect: "/invitation?token=abc",
		});
	});

	it("shows who invited the user and goes to the dashboard after accepting", async () => {
		const api = mockApi({
			"GET /api/me": signedIn({ id: "u2", slug: "bob" }),
			"POST /api/invitations/preview": { body: preview },
			"POST /api/invitations/accept": {
				body: { type: "dashboard", ownerSlug: "ann", slug: "work" },
			},
			"GET /api/u/ann/work": {
				body: viewData([], {
					dashboard: { slug: "work", title: "Work" },
					isOwner: false,
				}),
			},
		});
		const { user, router } = renderApp("/invitation?token=abc");
		expect(
			await screen.findByText('Ann invited you to view the dashboard "Work".'),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/invitations/accept")).toHaveLength(0);
		await user.click(screen.getByRole("button", { name: "Accept" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/ann/work"),
		);
		expect(api.callsTo("POST", "/api/invitations/preview")[0].body).toEqual({
			token: "abc",
		});
		expect(api.callsTo("POST", "/api/invitations/accept")[0].body).toEqual({
			token: "abc",
		});
	});

	it("goes to the group after accepting a group invitation", async () => {
		mockApi({
			"GET /api/me": signedIn({ id: "u2" }),
			"POST /api/invitations/preview": {
				body: {
					type: "group",
					title: "Family",
					inviterName: "Ann",
					role: "member",
				},
			},
			"POST /api/invitations/accept": {
				body: { type: "group", groupId: "g1" },
			},
			"GET /api/groups/g1": {
				body: {
					id: "g1",
					name: "Family",
					role: "member",
					memberCount: 1,
					members: [],
				},
			},
		});
		const { user, router } = renderApp("/invitation?token=abc");
		expect(
			await screen.findByText('Ann invited you to join the group "Family".'),
		).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Accept" }));
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ group: "g1" }),
		);
		expect(router.state.location.pathname).toBe("/groups");
	});

	it("says when the link is no longer valid", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"POST /api/invitations/preview": apiError(404, "invitation_not_found"),
		});
		renderApp("/invitation?token=old");
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This invitation is no longer valid.",
		);
		expect(screen.queryByRole("button", { name: "Accept" })).toBeNull();
	});

	it("says to use the invited address without showing it", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"POST /api/invitations/preview": apiError(
				403,
				"invitation_wrong_account",
			),
		});
		renderApp("/invitation?token=abc");
		const alert = await screen.findByRole("alert");
		expect(alert).toHaveTextContent("Sign in with the address it was sent to.");
		expect(alert.textContent).not.toContain("@");
	});

	it("treats a missing token as invalid", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/invitation");
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This invitation is no longer valid.",
		);
	});
});
