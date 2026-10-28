import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { type Dashboard, type DashboardEntry, defaultColumns } from "./api";

const home: Dashboard = {
	id: "d1",
	slug: "home",
	title: "Home",
	isDefault: true,
	isPublic: true,
	...defaultColumns,
};
const work: Dashboard = {
	id: "d2",
	slug: "work",
	title: "Work",
	isDefault: false,
	isPublic: false,
	...defaultColumns,
};

function view(
	current: DashboardEntry,
	dashboards: DashboardEntry[],
	isOwner = true,
) {
	return {
		body: {
			dashboard: current,
			ownerSlug: "ann",
			isOwner,
			dashboards,
			columns: defaultColumns,
			layoutVersion: 0,
			sections: [],
		},
	};
}

describe("dashboards page", () => {
	it("sends signed-out visitors to /login", async () => {
		const { router } = renderApp("/dashboards");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("shows an empty state", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/dashboards");
		expect(await screen.findByText("No dashboards yet")).toBeVisible();
		expect(screen.getByRole("button", { name: "New dashboard" })).toBeVisible();
	});

	it("lists dashboards with address and badges", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": { body: [home, work] },
		});
		renderApp("/dashboards");
		const homeRow = (await screen.findByText("Home")).closest("li");
		if (!homeRow) throw new Error("no row");
		expect(within(homeRow).getByText("Default")).toBeVisible();
		expect(within(homeRow).getByText("Public")).toBeVisible();
		expect(
			within(homeRow).getByRole("link", { name: "/ann/home" }),
		).toHaveAttribute("href", "/ann/home");

		const workRow = screen.getByText("Work").closest("li");
		if (!workRow) throw new Error("no row");
		expect(within(workRow).queryByText("Default")).toBeNull();
		expect(within(workRow).queryByText("Public")).toBeNull();
	});

	it("opens the create dialog from ?create=true and drops the param", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { router } = renderApp("/dashboards?create=true");
		expect(
			await screen.findByRole("dialog", { name: "New dashboard" }),
		).toBeVisible();
		await waitFor(() => expect(router.state.location.href).toBe("/dashboards"));
	});

	it("creates a dashboard with a slug that follows the title", async () => {
		let list: Dashboard[] = [];
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": () => ({ body: list }),
			"POST /api/dashboards": ({ body }) => {
				list = [{ id: "d9", ...(body as Omit<Dashboard, "id">) }];
				return { status: 201, body: list[0] };
			},
		});
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "New dashboard" }),
		);
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Title"), "Küche & Haus");
		expect(within(dialog).getByLabelText("Address")).toHaveValue("kueche-haus");
		await user.click(within(dialog).getByLabelText("Public"));
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		expect(await screen.findByText("Küche & Haus")).toBeVisible();
		expect(api.callsTo("POST", "/api/dashboards")[0].body).toEqual({
			title: "Küche & Haus",
			slug: "kueche-haus",
			isDefault: false,
			isPublic: true,
			...defaultColumns,
		});
	});

	it("keeps a manually edited slug and shows a taken slug at the field", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"POST /api/dashboards": apiError(409, "slug_taken"),
		});
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "New dashboard" }),
		);
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Address"), "Kitchen");
		await user.type(within(dialog).getByLabelText("Title"), "Küche");
		expect(within(dialog).getByLabelText("Address")).toHaveValue("kitchen");

		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(
			await within(dialog).findByText("This address is already taken."),
		).toBeVisible();
		expect(screen.getByRole("dialog", { name: "New dashboard" })).toBeVisible();
	});

	it("asks before renaming a slug and saves only when confirmed", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": { body: [home] },
			"PATCH /api/dashboards/d1": ({ body }) => ({
				body: { ...home, ...(body as object) },
			}),
		});
		const { user } = renderApp("/dashboards");
		await user.click(await screen.findByRole("button", { name: "Edit Home" }));
		const dialog = await screen.findByRole("dialog", {
			name: "Edit dashboard",
		});
		const slug = within(dialog).getByLabelText("Address");
		await user.clear(slug);
		await user.type(slug, "start");
		expect(
			within(dialog).getByText(
				"Links that use the old address will stop working.",
			),
		).toBeVisible();

		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		let confirm = await screen.findByRole("alertdialog", {
			name: "Change address?",
		});
		expect(within(confirm).getByText("/ann/home")).toBeVisible();
		expect(within(confirm).getByText("/ann/start")).toBeVisible();
		await user.click(within(confirm).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(api.callsTo("PATCH", "/api/dashboards/d1")).toHaveLength(0);
		expect(
			screen.getByRole("dialog", { name: "Edit dashboard" }),
		).toBeVisible();

		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		confirm = await screen.findByRole("alertdialog");
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(api.callsTo("PATCH", "/api/dashboards/d1")[0].body).toMatchObject({
			slug: "start",
		});
	});

	it("deletes a dashboard after confirmation", async () => {
		let list = [home, work];
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": () => ({ body: list }),
			"DELETE /api/dashboards/d2": () => {
				list = [home];
				return { status: 204 };
			},
		});
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "Delete Work" }),
		);
		const confirm = await screen.findByRole("alertdialog");
		expect(within(confirm).getByText(/“Work” will be deleted/)).toBeVisible();
		await user.click(within(confirm).getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(screen.queryByText("Work")).toBeNull());
		expect(api.callsTo("DELETE", "/api/dashboards/d2")).toHaveLength(1);
	});
});

describe("dashboard address", () => {
	const entries = [
		{ slug: "home", title: "Home" },
		{ slug: "work", title: "Work" },
	];

	it("shows the owner's dashboard with a switcher that changes the URL", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/u/ann/home": view(entries[0], entries),
			"GET /api/u/ann/work": view(entries[1], entries),
		});
		const { user, router } = renderApp("/ann/home");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Home" }),
		).toBeVisible();

		await user.click(
			screen.getByRole("button", { name: "Switch dashboard: Home" }),
		);
		const menu = await screen.findByRole("menu");
		expect(
			within(menu).getByRole("menuitem", { name: "Home" }),
		).toHaveAttribute("aria-current", "page");
		await user.click(within(menu).getByRole("menuitem", { name: "Work" }));

		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/ann/work"),
		);
		expect(
			await screen.findByRole("heading", { level: 1, name: "Work" }),
		).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Switch dashboard: Work" }),
		).toBeVisible();
	});

	it("shows a public dashboard to visitors with only public entries", async () => {
		const api = mockApi({
			"GET /api/u/ann/home": view(
				entries[0],
				[entries[0], { slug: "links", title: "Links" }],
				false,
			),
		});
		const { user } = renderApp("/Ann/Home");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Home" }),
		).toBeVisible();
		await user.click(
			screen.getByRole("button", { name: "Switch dashboard: Home" }),
		);
		const items = await screen.findAllByRole("menuitem");
		expect(items.map((i) => i.textContent)).toEqual(["Home", "Links"]);
		expect(api.callsTo("GET", "/api/u/ann/home")).toHaveLength(1);
	});

	it("hides the switcher with only one entry", async () => {
		mockApi({ "GET /api/u/ann/home": view(entries[0], [entries[0]], false) });
		renderApp("/ann/home");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Home" }),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: /Switch dashboard/ }),
		).toBeNull();
	});

	it("uses the full width of the shell", async () => {
		mockApi({ "GET /api/u/ann/home": view(entries[0], [entries[0]], false) });
		renderApp("/ann/home");
		await screen.findByRole("heading", { level: 1, name: "Home" });
		const page = screen.getByRole("main").firstElementChild;
		expect(page).toHaveClass("w-full");
		expect(page?.className).not.toMatch(/max-w-/);
	});

	it("shows not found when the dashboard can't be shown", async () => {
		renderApp("/ann/work");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeVisible();
	});

	it("redirects a user address alone to /", async () => {
		const { router } = renderApp("/ann");
		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
	});

	it("keeps the app's own routes ahead of dashboard addresses", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		expect(
			await screen.findByRole("heading", { name: "Settings" }),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: /Switch dashboard/ }),
		).toBeNull();
	});

	it("doesn't show the switcher on other pages", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/u/ann/home": view(entries[0], entries),
		});
		const { user } = renderApp("/ann/home");
		await screen.findByRole("button", { name: "Switch dashboard: Home" });
		await user.click(screen.getByRole("button", { name: "User menu" }));
		await user.click(await screen.findByRole("menuitem", { name: "Settings" }));
		expect(
			await screen.findByRole("heading", { name: "Settings" }),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: /Switch dashboard/ }),
		).toBeNull();
	});
});

describe("column settings", () => {
	it("starts a new dashboard with the default columns", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "New dashboard" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New dashboard" });
		expect(within(dialog).getByLabelText("Columns")).toHaveValue("6");
		expect(within(dialog).getByLabelText("Columns (mobile)")).toHaveValue("2");
		expect(
			within(dialog).getByLabelText("Small screens are narrower than"),
		).toHaveValue("768");
	});

	it("blocks more mobile than desktop columns", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "New dashboard" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New dashboard" });
		await user.type(within(dialog).getByLabelText("Title"), "Narrow");
		await user.selectOptions(within(dialog).getByLabelText("Columns"), "3");
		await user.selectOptions(
			within(dialog).getByLabelText("Columns (mobile)"),
			"4",
		);
		expect(
			within(dialog).getByText(
				"Use at most as many columns as on large screens.",
			),
		).toBeVisible();
		expect(within(dialog).getByRole("button", { name: "Save" })).toBeDisabled();
	});

	it("sends changed column settings when editing", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": { body: [home] },
			"PATCH /api/dashboards/d1": ({ body }) => ({
				body: { ...home, ...(body as object) },
			}),
		});
		const { user } = renderApp("/dashboards");
		await user.click(await screen.findByRole("button", { name: "Edit Home" }));
		const dialog = await screen.findByRole("dialog", {
			name: "Edit dashboard",
		});
		await user.selectOptions(within(dialog).getByLabelText("Columns"), "8");
		await user.selectOptions(
			within(dialog).getByLabelText("Small screens are narrower than"),
			"1024",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(api.callsTo("PATCH", "/api/dashboards/d1")[0].body).toMatchObject({
			desktopColumns: 8,
			mobileColumns: 2,
			mobileBreakpoint: 1024,
		});
	});

	it("shows a server column error in the dialog", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"POST /api/dashboards": apiError(400, "invalid_columns"),
		});
		const { user } = renderApp("/dashboards");
		await user.click(
			await screen.findByRole("button", { name: "New dashboard" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New dashboard" });
		await user.type(within(dialog).getByLabelText("Title"), "Home");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(await within(dialog).findByRole("alert")).toHaveTextContent(
			"These column settings aren't possible.",
		);
	});
});
