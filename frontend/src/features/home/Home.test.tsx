import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type Dashboard, defaultColumns } from "#/features/dashboards/api";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const home: Dashboard = {
	id: "d1",
	slug: "home",
	title: "Home",
	isDefault: false,
	isPublic: false,
	...defaultColumns,
};
const work: Dashboard = { ...home, id: "d2", slug: "work", title: "Work" };

const view = (d: Dashboard, all: Dashboard[]) => ({
	body: {
		dashboard: { slug: d.slug, title: d.title },
		ownerSlug: "ann",
		isOwner: true,
		dashboards: all.map(({ slug, title }) => ({ slug, title })),
		columns: defaultColumns,
		layoutVersion: 0,
		sections: [],
	},
});

describe("home page", () => {
	it("greets a user by name and offers the first dashboard", async () => {
		mockApi({ "GET /api/me": signedIn({ name: "Ann" }) });
		const { user, router } = renderApp("/");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Welcome, Ann" }),
		).toBeVisible();
		expect(screen.queryByRole("heading", { name: "Boardly" })).toBeNull();
		expect(screen.queryByText("Screenshot coming soon")).toBeNull();

		await user.click(
			screen.getByRole("link", { name: "Create your first dashboard" }),
		);
		expect(
			await screen.findByRole("dialog", { name: "New dashboard" }),
		).toBeVisible();
		await waitFor(() => expect(router.state.location.href).toBe("/dashboards"));
	});

	it("greets a user without a name by email", async () => {
		mockApi({ "GET /api/me": signedIn({ email: "bob@example.org" }) });
		renderApp("/");
		expect(
			await screen.findByRole("heading", {
				level: 1,
				name: "Welcome, bob@example.org",
			}),
		).toBeVisible();
	});

	it("shows the default dashboard and keeps the URL", async () => {
		const all = [{ ...home, isDefault: true }, work];
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": { body: all },
			"GET /api/u/ann/home": view(home, all),
		});
		const { router } = renderApp("/");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Home" }),
		).toBeVisible();
		expect(screen.getByText("This dashboard is empty")).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Switch dashboard: Home" }),
		).toBeVisible();
		expect(router.state.location.pathname).toBe("/");
	});

	it("lets the user pick a default dashboard and shows it", async () => {
		let all = [home, work];
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": () => ({ body: all }),
			"PATCH /api/dashboards/d2": () => {
				all = [home, { ...work, isDefault: true }];
				return { body: all[1] };
			},
			"GET /api/u/ann/work": () => view(work, all),
		});
		const { user } = renderApp("/");
		expect(
			await screen.findByText("Select your default dashboard"),
		).toBeVisible();

		await user.selectOptions(
			screen.getByLabelText("Default dashboard"),
			"Work",
		);
		expect(
			await screen.findByRole("heading", { level: 1, name: "Work" }),
		).toBeVisible();
		expect(api.callsTo("PATCH", "/api/dashboards/d2")[0].body).toEqual({
			isDefault: true,
		});
	});
});
