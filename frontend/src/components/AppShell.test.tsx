import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { THEME_STORAGE_KEY } from "#/theme";

describe("app shell", () => {
	it.each(["/", "/login", "/no/such/page"])(
		"shows the header on %s",
		async (path) => {
			renderApp(path);
			const home = await screen.findByRole("link", { name: "Famodash" });
			expect(home).toHaveAttribute("href", "/");
			expect(home.firstElementChild).toHaveAttribute("data-icon", "app");
			expect(screen.getByRole("button", { name: "Theme" })).toBeVisible();
			expect(screen.getByRole("button", { name: "Language" })).toBeVisible();
			expect(screen.getByRole("button", { name: "User menu" })).toBeVisible();
		},
	);

	it("places the theme switcher directly before the language switcher", async () => {
		renderApp("/");
		const theme = await screen.findByRole("button", { name: "Theme" });
		expect(theme.nextElementSibling).toBe(
			screen.getByRole("button", { name: "Language" }),
		);
	});

	it("shows a not-found page with a link home for unknown paths", async () => {
		renderApp("/no/such/page");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeVisible();
		expect(
			screen.getByRole("link", { name: "Back to the start page" }),
		).toHaveAttribute("href", "/");
	});
});

describe("user menu", () => {
	it("opens with the keyboard, closes on Escape and returns focus", async () => {
		const { user } = renderApp("/");
		const avatar = await screen.findByRole("button", { name: "User menu" });

		avatar.focus();
		await user.keyboard("{Enter}");
		expect(await screen.findByRole("menu")).toBeVisible();

		await user.keyboard("{Escape}");
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
		expect(avatar).toHaveFocus();
	});

	it("closes on an outside click", async () => {
		const { user } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "User menu" }));
		expect(await screen.findByRole("menu")).toBeVisible();

		await user.click(document.body);
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});

	it("offers only 'Sign in' when signed out, which goes to /login", async () => {
		const { user, router } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "User menu" }));

		const items = await screen.findAllByRole("menuitem");
		expect(items.map((i) => i.textContent)).toEqual(["Sign in"]);

		await user.click(items[0]);
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});
});

describe("signed-in user menu", () => {
	async function openMenu(user: ReturnType<typeof renderApp>["user"]) {
		await screen.findByRole("heading", { name: /Welcome/ });
		await user.click(screen.getByRole("button", { name: "User menu" }));
		return screen.findByRole("menu");
	}

	it("shows name, email, Dashboards, Favourites, Settings and Sign out in this order", async () => {
		mockApi({ "GET /api/me": signedIn({ name: "Ann" }) });
		const { user } = renderApp("/");
		const menu = await openMenu(user);
		expect(menu.textContent).toBe(
			"Annann@example.orgDashboardsFavouritesSettingsSign out",
		);
		expect(
			within(menu)
				.getAllByRole("menuitem")
				.map((i) => i.textContent),
		).toEqual(["Dashboards", "Favourites", "Settings", "Sign out"]);
	});

	it("shows the email without an empty name line when there is no name", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/");
		const menu = await openMenu(user);
		expect(menu.textContent).toBe(
			"ann@example.orgDashboardsFavouritesSettingsSign out",
		);
		expect(
			within(menu).getByText("ann@example.org").previousSibling,
		).toBeNull();
	});

	it("shows only a loading state while the session is unknown", async () => {
		mockApi({ "GET /api/me": () => new Promise<never>(() => {}) });
		const { user } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "User menu" }));
		const menu = await screen.findByRole("menu");
		expect(within(menu).getByText("Loading…")).toBeVisible();
		expect(within(menu).queryAllByRole("menuitem")).toHaveLength(0);
	});

	it("goes to /dashboards from the Dashboards item and closes the menu", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user, router } = renderApp("/");
		const menu = await openMenu(user);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Dashboards" }),
		);
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/dashboards"),
		);
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});

	it("goes to /favourites from the Favourites item and closes the menu", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/favourites": {
				body: { id: "root", kind: "folder", title: null },
			},
		});
		const { user, router } = renderApp("/");
		const menu = await openMenu(user);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Favourites" }),
		);
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/favourites"),
		);
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});

	it("goes to /settings from the Settings item and closes the menu", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user, router } = renderApp("/");
		const menu = await openMenu(user);
		await user.click(within(menu).getByRole("menuitem", { name: "Settings" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/settings"),
		);
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});

	it("signs out to the landing page with 'Sign in' in the menu", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"POST /api/auth/logout": { status: 204 },
		});
		const { user, router } = renderApp("/settings");
		await screen.findByRole("heading", { name: "Settings" });
		await user.click(screen.getByRole("button", { name: "User menu" }));
		await user.click(await screen.findByRole("menuitem", { name: "Sign out" }));

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { level: 1, name: "Famodash" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/auth/logout")).toHaveLength(1);

		await user.click(screen.getByRole("button", { name: "User menu" }));
		const items = await screen.findAllByRole("menuitem");
		expect(items.map((i) => i.textContent)).toEqual(["Sign in"]);
	});
});

describe("language switcher", () => {
	it("switches all texts and the lang attribute without reload", async () => {
		const { user } = renderApp("/");
		expect(await screen.findByRole("link", { name: "Sign in" })).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Language" }));
		await user.click(
			await screen.findByRole("menuitemradio", { name: "Deutsch" }),
		);

		expect(await screen.findByRole("link", { name: "Anmelden" })).toBeVisible();
		expect(screen.getByRole("button", { name: "Benutzermenü" })).toBeVisible();
		expect(document.documentElement.lang).toBe("de");
	});
});

describe("theme switcher", () => {
	const root = document.documentElement;

	async function openMenu(user: ReturnType<typeof renderApp>["user"]) {
		await user.click(await screen.findByRole("button", { name: "Theme" }));
		return screen.findByRole("menu");
	}

	it("lists System and the 7 themes with System selected by default", async () => {
		const { user } = renderApp("/");
		const menu = await openMenu(user);
		const items = within(menu).getAllByRole("menuitemradio");
		expect(items.map((i) => i.textContent)).toEqual([
			"System",
			"White",
			"Black",
			"Sand",
			"Sage",
			"Fjord",
			"Dusk",
			"Ember",
		]);
		expect(items[0]).toHaveAttribute("aria-checked", "true");
	});

	it("applies and stores the chosen theme and closes the menu", async () => {
		const { user } = renderApp("/");
		const menu = await openMenu(user);
		await user.click(within(menu).getByRole("menuitemradio", { name: "Sage" }));

		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
		expect(root.dataset.theme).toBe("sage");
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("sage");

		const reopened = await openMenu(user);
		expect(
			within(reopened).getByRole("menuitemradio", { name: "Sage" }),
		).toHaveAttribute("aria-checked", "true");
	});

	it("opens with the keyboard, closes on Escape and returns focus", async () => {
		const { user } = renderApp("/");
		const button = await screen.findByRole("button", { name: "Theme" });

		button.focus();
		await user.keyboard("{Enter}");
		expect(await screen.findByRole("menu")).toBeVisible();

		await user.keyboard("{Escape}");
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
		expect(button).toHaveFocus();
	});

	it("shows the theme names in German", async () => {
		await i18n.changeLanguage("de");
		const { user } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "Farbschema" }));
		expect(
			await screen.findByRole("menuitemradio", { name: "Dämmerung" }),
		).toBeVisible();
	});
});
