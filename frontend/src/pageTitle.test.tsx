import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type Dashboard, defaultColumns } from "#/features/dashboards/api";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const home: Dashboard = {
	id: "d1",
	slug: "home",
	title: "Home",
	isDefault: true,
	isPublic: false,
	...defaultColumns,
};
const work: Dashboard = {
	...home,
	id: "d2",
	slug: "work",
	title: "Work",
	isDefault: false,
};

const view = (d: Dashboard) => ({
	body: {
		dashboard: { slug: d.slug, title: d.title },
		ownerSlug: "ann",
		isOwner: true,
		dashboards: [home, work].map(({ slug, title }) => ({ slug, title })),
		columns: defaultColumns,
		layoutVersion: 0,
		sections: [],
	},
});

const signedInRoutes = {
	"GET /api/me": signedIn(),
	"GET /api/dashboards": { body: [home, work] },
	"GET /api/u/ann/home": view(home),
	"GET /api/u/ann/work": view(work),
	"GET /api/favourites": { body: { id: "root", kind: "folder", title: null } },
};

const titleIs = (title: string) =>
	waitFor(() => expect(document.title).toBe(title));

describe("document title", () => {
	it("is the app name alone on the landing page", async () => {
		const { router } = renderApp("/login");
		await titleIs("Sign in - Boardly");
		await act(() => router.navigate({ to: "/" }));
		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		await titleIs("Boardly");
	});

	it.each([
		["/settings", "Settings - Boardly"],
		["/dashboards", "Dashboards - Boardly"],
		["/favourites", "Favourites - Boardly"],
		["/ann/work", "Work - Boardly"],
		["/", "Home - Boardly"],
	])("shows the page title on %s", async (path, title) => {
		mockApi(signedInRoutes);
		renderApp(path);
		await titleIs(title);
	});

	it("shows the sign-in and not-found titles", async () => {
		const { router } = renderApp("/login");
		await titleIs("Sign in - Boardly");
		await act(() => router.navigate({ href: "/no/such/page" }));
		await titleIs("Page not found - Boardly");
	});

	it("follows navigation and the language", async () => {
		mockApi(signedInRoutes);
		const { router } = renderApp("/favourites");
		await titleIs("Favourites - Boardly");
		await act(() => router.navigate({ to: "/dashboards" }));
		await titleIs("Dashboards - Boardly");
		await act(() => router.navigate({ to: "/favourites" }));
		await act(() => i18n.changeLanguage("de"));
		await titleIs("Favoriten - Boardly");
	});
});
