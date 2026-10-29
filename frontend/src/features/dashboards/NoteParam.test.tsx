import { act, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ContentWidget } from "#/features/dashboards/api";
import { section, viewData } from "#/test/dashboards";
import { fakeNotesServer, note } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import { validateDashboardSearch } from "./DashboardView";

const roses = note("roses", { title: "Roses" });

const listWidget: ContentWidget = {
	id: "w1",
	kind: "note_list",
	width: 2,
	height: "auto",
	config: {
		category: "all",
		categoryId: null,
		subcategories: true,
		sort: "updated",
		limit: 10,
		title: null,
		size: "M",
		showIfPublic: true,
	},
	data: {
		notes: [
			{ id: "roses", title: "Roses", icon: null, color: null, category: null },
		],
	},
};

function dashboard(isOwner = true) {
	return fakeNotesServer([roses], [], {
		"GET /api/notes/gone": () => ({
			status: 404,
			body: { error: { code: "not_found", message: "not found" } },
		}),
		"GET /api/u/ann/home": () => ({
			body: viewData(
				[section("Work", {}, listWidget)],
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
	});
}

describe("validateDashboardSearch", () => {
	it("keeps a note ID and drops anything else", () => {
		expect(validateDashboardSearch({ note: "roses" }).note).toBe("roses");
		expect(validateDashboardSearch({ note: "" }).note).toBeUndefined();
		expect(validateDashboardSearch({ note: 3 }).note).toBeUndefined();
		expect(
			validateDashboardSearch({ note: "a".repeat(65) }).note,
		).toBeUndefined();
	});
});

describe("the note modal in the address", () => {
	it("opens with a new history entry and closes with back", async () => {
		dashboard();
		const { user, router } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: /Roses/ }));
		expect(await screen.findByRole("dialog", { name: "Roses" })).toBeVisible();
		expect(router.state.location.search).toEqual({ note: "roses" });
		expect(router.history.length).toBe(2);

		act(() => router.history.back());
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(router.state.location.pathname).toBe("/ann/home");
		expect(router.state.location.search).toEqual({});
	});

	it("goes back when closed with its button, leaving no extra entry", async () => {
		dashboard();
		const { user, router } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: /Roses/ }));
		const dialog = await screen.findByRole("dialog", { name: "Roses" });
		await user.click(
			await within(dialog).findByRole("button", { name: "Close" }),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(router.state.location.search).toEqual({});
		expect(router.history.location.state.__TSR_index).toBe(0);
	});

	it("opens from the address and replaces it when closed", async () => {
		dashboard();
		const { user, router } = renderApp("/ann/home?note=roses");
		const dialog = await screen.findByRole("dialog", { name: "Roses" });
		await user.click(
			await within(dialog).findByRole("button", { name: "Close" }),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(router.state.location.search).toEqual({});
		expect(router.history.length).toBe(1);
	});

	it("drops an unknown note", async () => {
		dashboard();
		const { router } = renderApp("/ann/home?note=gone");
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		expect(screen.queryByRole("dialog")).toBeNull();
		expect(await screen.findByText("Roses")).toBeVisible();
	});

	it("drops the note for visitors", async () => {
		dashboard(false);
		const { router } = renderApp("/ann/home?note=roses");
		expect(await screen.findByText("Roses")).toBeVisible();
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		expect(screen.queryByRole("dialog")).toBeNull();
	});

	it("drops the note in edit mode", async () => {
		dashboard();
		const { router } = renderApp("/ann/home?edit=true&note=roses");
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ edit: true }),
		);
		expect(screen.queryByRole("dialog", { name: "Roses" })).toBeNull();
	});
});
