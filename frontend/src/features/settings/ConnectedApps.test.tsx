import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { setTheme, type ThemeId } from "#/theme";
import { appLogos } from "./appTokens/AppLogo";
import type { AppToken } from "./appTokens/api";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{
			id: "reading",
			kind: "folder",
			title: "Reading",
			children: [{ id: "later", kind: "folder", title: "Later" }],
		},
	],
};

const active: AppToken = {
	id: "t1",
	name: "Phone",
	app: "karakeep",
	prefix: "bdly_ABCDEFG",
	rootId: "later",
	noteScope: "all",
	rootCategoryId: null,
	createdAt: "2026-02-01T10:00:00Z",
	lastUsedAt: null,
	expiresAt: null,
	revokedAt: null,
	status: "active",
};

const revoked: AppToken = {
	...active,
	id: "t2",
	name: "Laptop",
	rootId: null,
	revokedAt: "2026-02-20T10:00:00Z",
	status: "revoked",
};

const categories = [
	{ id: "inbox", title: "Inbox", icon: null, color: null, parentId: null },
	{ id: "ideas", title: "Ideas", icon: null, color: null, parentId: "inbox" },
];

function routes(tokens: AppToken[]) {
	return {
		"GET /api/me": signedIn(),
		"GET /api/favourites": { body: tree },
		"GET /api/app-tokens": { body: tokens },
		"GET /api/note-categories": { body: categories },
	};
}

async function openTab(tokens: AppToken[]) {
	const api = mockApi(routes(tokens));
	const app = renderApp("/settings?tab=apps");
	const panel = await screen.findByRole("tabpanel");
	return { api, panel, ...app };
}

describe("connected apps tab", () => {
	it("is the third tab and survives a reload", async () => {
		mockApi(routes([]));
		const { user, router, unmount } = renderApp("/settings");
		const tabs = await screen.findAllByRole("tab");
		expect(tabs.map((tab) => tab.textContent)).toEqual([
			"Profile",
			"Passkeys",
			"Connected apps",
			"Import / Export",
		]);
		await user.click(tabs[2]);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ tab: "apps" }),
		);

		// Reload: a fresh app at the same URL selects the tab again.
		unmount();
		await openTab([]);
		await waitFor(() =>
			expect(
				screen.getByRole("tab", { name: "Connected apps" }),
			).toHaveAttribute("aria-selected", "true"),
		);
	});

	it("explains connected apps when there are none", async () => {
		const { panel } = await openTab([]);
		expect(
			await within(panel).findByText("No connected apps yet"),
		).toBeVisible();
		expect(
			within(panel).getByRole("button", { name: "Connect app" }),
		).toBeVisible();
	});

	it("lists tokens with root path, status and the action for their status", async () => {
		const { panel } = await openTab([active, revoked]);
		const items = await within(panel).findAllByRole("listitem");
		expect(items).toHaveLength(2);

		const [phone, laptop] = items;
		expect(phone).toHaveTextContent("Phone");
		expect(phone).toHaveTextContent("Active");
		expect(phone).toHaveTextContent("Karakeep · Favourites / Reading / Later");
		expect(phone).toHaveTextContent("bdly_ABCDEFG…");
		expect(phone).toHaveTextContent("never used");
		expect(phone).toHaveTextContent("no expiry");
		expect(
			within(phone).getByRole("button", { name: "Revoke Phone" }),
		).toBeVisible();
		expect(
			within(phone).getByRole("button", { name: "Edit Phone" }),
		).toBeVisible();
		expect(within(phone).queryByRole("button", { name: /Delete/ })).toBeNull();

		expect(laptop).toHaveTextContent("Revoked");
		expect(laptop).toHaveTextContent("Karakeep · Deleted folder");
		expect(
			within(laptop).getByRole("button", { name: "Delete Laptop" }),
		).toBeVisible();
		expect(within(laptop).queryByRole("button", { name: /Revoke/ })).toBeNull();
		expect(within(laptop).queryByRole("button", { name: /Edit/ })).toBeNull();
	});

	it("shows the notes each token sees", async () => {
		const { panel } = await openTab([
			active,
			{
				...active,
				id: "t4",
				name: "Inbox",
				noteScope: "category",
				rootCategoryId: "ideas",
			},
			{ ...revoked, noteScope: "category", rootCategoryId: null },
			{
				...active,
				id: "t3",
				name: "Tablet",
				app: "linkwarden",
				noteScope: "none",
			},
		]);
		const [phone, inbox, laptop, tablet] =
			await within(panel).findAllByRole("listitem");
		expect(phone).toHaveTextContent("Later · All notes");
		expect(await within(inbox).findByText(/Inbox › Ideas/)).toBeVisible();
		expect(laptop).toHaveTextContent("Deleted folder · Deleted category");
		expect(tablet).not.toHaveTextContent("notes");
	});

	it.each<[ThemeId, string, string]>([
		["white", appLogos.karakeep.light, appLogos.linkwarden.light],
		["black", appLogos.karakeep.dark as string, appLogos.linkwarden.light],
	])(
		"in theme %s shows the logo of each token's app",
		async (theme, karakeepSrc, linkwardenSrc) => {
			setTheme(theme);
			const linkwarden: AppToken = {
				...active,
				id: "t3",
				name: "Tablet",
				app: "linkwarden",
			};
			const { panel } = await openTab([active, linkwarden]);
			const [phone, tablet] = await within(panel).findAllByRole("listitem");

			expect(phone.querySelector("img")).toHaveAttribute("src", karakeepSrc);
			expect(phone.querySelector("img")).toHaveAttribute("alt", "");
			expect(phone).toHaveTextContent("Karakeep · Favourites");
			expect(tablet.querySelector("img")).toHaveAttribute("src", linkwardenSrc);
			expect(tablet.querySelector("img")).toHaveAttribute("alt", "");
			expect(tablet).toHaveTextContent("Linkwarden · Favourites");
		},
	);

	it("revokes only after confirmation", async () => {
		const { panel, user, api } = await openTab([active]);
		await user.click(
			await within(panel).findByRole("button", { name: "Revoke Phone" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(api.callsTo("POST", "/api/app-tokens/t1/revoke")).toHaveLength(0);

		api.set({
			"POST /api/app-tokens/t1/revoke": {
				body: { ...active, status: "revoked" },
			},
			"GET /api/app-tokens": {
				body: [{ ...active, status: "revoked", revokedAt: active.createdAt }],
			},
		});
		await user.click(
			within(panel).getByRole("button", { name: "Revoke Phone" }),
		);
		await user.click(
			within(await screen.findByRole("alertdialog")).getByRole("button", {
				name: "Revoke",
			}),
		);
		expect(
			await within(panel).findByRole("button", { name: "Delete Phone" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/app-tokens/t1/revoke")).toHaveLength(1);
	});

	it("deletes a revoked token after confirmation", async () => {
		const { panel, user, api } = await openTab([revoked]);
		api.set({
			"DELETE /api/app-tokens/t2": { status: 204 },
			"GET /api/app-tokens": { body: [] },
		});
		await user.click(
			await within(panel).findByRole("button", { name: "Delete Laptop" }),
		);
		await user.click(
			within(await screen.findByRole("alertdialog")).getByRole("button", {
				name: "Delete",
			}),
		);
		expect(
			await within(panel).findByText("No connected apps yet"),
		).toBeVisible();
		expect(api.callsTo("DELETE", "/api/app-tokens/t2")).toHaveLength(1);
	});
});

describe("create dialog", () => {
	it("creates a token and shows its secret once", async () => {
		const { panel, user, api } = await openTab([]);
		const secret = "bdly_ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
		api.set({
			"POST /api/app-tokens": ({ body }) => ({
				status: 201,
				body: {
					...active,
					...(body as object),
					status: "active",
					expiresAt: null,
					secret,
				},
			}),
		});
		await user.click(
			await within(panel).findByRole("button", { name: "Connect app" }),
		);
		const dialog = await screen.findByRole("dialog");
		const create = within(dialog).getByRole("button", { name: "Create key" });

		// 90 days is the default expiry; Karakeep is preselected.
		expect(
			within(dialog).getByRole("radio", { name: "In 90 days" }),
		).toBeChecked();
		expect(within(dialog).getByLabelText("App")).toHaveValue("karakeep");
		expect(create).toBeDisabled();

		await user.type(within(dialog).getByLabelText("Name"), "Tablet");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(dialog).getByRole("radio", { name: "Never" }));
		await user.click(create);

		expect(
			await within(dialog).findByRole("heading", { name: "App connected" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/app-tokens")[0].body).toEqual({
			name: "Tablet",
			app: "karakeep",
			rootId: "reading",
			rootCategoryId: null,
			expiresAt: null,
		});
		expect(within(dialog).getByLabelText("API key")).toHaveValue(secret);
		expect(within(dialog).getByLabelText("Server address")).toHaveValue(
			`${window.location.origin}/api/karakeep`,
		);
		expect(within(dialog).getByText(/won't be shown again/)).toBeVisible();

		await user.click(
			within(dialog).getByRole("button", { name: "Copy API key" }),
		);
		expect(await navigator.clipboard.readText()).toBe(secret);
		expect(
			within(dialog).getByRole("button", { name: "Copied" }),
		).toBeVisible();
		// The check mark turns back into the copy icon after 3 seconds.
		expect(
			await within(dialog).findByRole(
				"button",
				{ name: "Copy API key" },
				{ timeout: 4000 },
			),
		).toBeVisible();

		await user.click(within(dialog).getByRole("button", { name: "Done" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(screen.queryByDisplayValue(secret)).toBeNull();

		// Opening the dialog again starts with an empty form, not the secret.
		await user.click(
			within(panel).getByRole("button", { name: "Connect app" }),
		);
		const again = await screen.findByRole("dialog");
		expect(within(again).getByLabelText("Name")).toHaveValue("");
		expect(screen.queryByDisplayValue(secret)).toBeNull();
		// Waiting for the copy button's 3 seconds needs more than the default timeout.
	}, 15_000);

	it("offers Linkwarden and shows its server address", async () => {
		const { panel, user, api } = await openTab([]);
		api.set({
			"POST /api/app-tokens": ({ body }) => ({
				status: 201,
				body: {
					...active,
					...(body as object),
					status: "active",
					expiresAt: null,
					secret: "bdly_ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",
				},
			}),
		});
		await user.click(
			await within(panel).findByRole("button", { name: "Connect app" }),
		);
		const dialog = await screen.findByRole("dialog");
		const appSelect = within(dialog).getByLabelText("App");
		expect(
			within(appSelect)
				.getAllByRole("option")
				.map((o) => o.textContent),
		).toEqual(["Karakeep", "Linkwarden"]);

		await user.selectOptions(appSelect, "linkwarden");
		await user.type(within(dialog).getByLabelText("Name"), "Phone");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Create key" }),
		);

		expect(
			await within(dialog).findByRole("heading", { name: "App connected" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/app-tokens")[0].body).toMatchObject({
			app: "linkwarden",
		});
		expect(within(dialog).getByLabelText("Server address")).toHaveValue(
			`${window.location.origin}/api/linkwarden`,
		);
	});

	it("sends a preset expiry as a time in the future", async () => {
		const { panel, user, api } = await openTab([]);
		api.set({
			"POST /api/app-tokens": {
				status: 201,
				body: { ...active, secret: "bdly_X" },
			},
		});
		await user.click(
			await within(panel).findByRole("button", { name: "Connect app" }),
		);
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "Phone");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(dialog).getByRole("radio", { name: "In 30 days" }));
		const before = Date.now();
		await user.click(
			within(dialog).getByRole("button", { name: "Create key" }),
		);
		await within(dialog).findByRole("heading", { name: "App connected" });

		const sent = api.callsTo("POST", "/api/app-tokens")[0].body as {
			expiresAt: string;
		};
		const days = (Date.parse(sent.expiresAt) - before) / 86_400_000;
		expect(days).toBeGreaterThan(29.99);
		expect(days).toBeLessThan(30.01);
	});

	it("requires a custom date in the future", async () => {
		const { panel, user } = await openTab([]);
		await user.click(
			await within(panel).findByRole("button", { name: "Connect app" }),
		);
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "Phone");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(dialog).getByRole("radio", { name: "On a date" }));
		const create = within(dialog).getByRole("button", { name: "Create key" });
		expect(create).toBeDisabled();

		await user.type(within(dialog).getByLabelText("Expiry date"), "2000-01-01");
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"Choose a date in the future.",
		);
		expect(create).toBeDisabled();

		const field = within(dialog).getByLabelText("Expiry date");
		await user.clear(field);
		await user.type(field, `${new Date().getFullYear() + 1}-06-30`);
		expect(within(dialog).queryByRole("alert")).toBeNull();
		expect(create).toBeEnabled();
	});
});

describe("root category in the create dialog", () => {
	async function openDialog() {
		const tab = await openTab([]);
		tab.api.set({
			"POST /api/app-tokens": ({ body }) => ({
				status: 201,
				body: { ...active, ...(body as object), secret: "bdly_X" },
			}),
		});
		await tab.user.click(
			await within(tab.panel).findByRole("button", { name: "Connect app" }),
		);
		return { ...tab, dialog: await screen.findByRole("dialog") };
	}

	it("sends the chosen category for Karakeep", async () => {
		const { dialog, user, api } = await openDialog();
		const picker = within(dialog).getByRole("radiogroup", {
			name: "Category for notes",
		});
		expect(
			await within(picker).findByRole("radio", { name: "All notes" }),
		).toBeChecked();
		await user.click(within(picker).getByRole("radio", { name: "Ideas" }));
		await user.type(within(dialog).getByLabelText("Name"), "Phone");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Create key" }),
		);

		await within(dialog).findByRole("heading", { name: "App connected" });
		expect(api.callsTo("POST", "/api/app-tokens")[0].body).toMatchObject({
			rootCategoryId: "ideas",
		});
	});

	it("has no category for Linkwarden", async () => {
		const { dialog, user, api } = await openDialog();
		await user.selectOptions(
			within(dialog).getByLabelText("App"),
			"linkwarden",
		);
		expect(
			within(dialog).queryByRole("radiogroup", { name: "Category for notes" }),
		).toBeNull();
		await user.type(within(dialog).getByLabelText("Name"), "Tablet");
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Create key" }),
		);

		await within(dialog).findByRole("heading", { name: "App connected" });
		expect(api.callsTo("POST", "/api/app-tokens")[0].body).not.toHaveProperty(
			"rootCategoryId",
		);
	});
});

describe("edit dialog", () => {
	async function openEdit(token: AppToken) {
		const tab = await openTab([token]);
		await tab.user.click(
			await within(tab.panel).findByRole("button", {
				name: `Edit ${token.name}`,
			}),
		);
		return { ...tab, dialog: await screen.findByRole("dialog") };
	}

	it("changes the folder and sends only that", async () => {
		const { dialog, user, api } = await openEdit(active);
		api.set({
			"PATCH /api/app-tokens/t1": ({ body }) => ({
				body: { ...active, ...(body as object) },
			}),
		});
		const save = within(dialog).getByRole("button", { name: "Save" });
		expect(
			await within(dialog).findByRole("radio", { name: "Later" }),
		).toBeChecked();
		expect(save).toBeDisabled();
		await user.click(within(dialog).getByRole("radio", { name: "Reading" }));
		await user.click(save);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(api.callsTo("PATCH", "/api/app-tokens/t1")[0].body).toEqual({
			rootId: "reading",
		});
	});

	it("changes the category of a Karakeep token", async () => {
		const { dialog, user, api } = await openEdit(active);
		api.set({ "PATCH /api/app-tokens/t1": { body: active } });
		const picker = within(dialog).getByRole("radiogroup", {
			name: "Category for notes",
		});
		await user.click(
			await within(picker).findByRole("radio", { name: "Inbox" }),
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(api.callsTo("PATCH", "/api/app-tokens/t1")[0].body).toEqual({
			rootCategoryId: "inbox",
		});
	});

	it("shows only the folder for Linkwarden", async () => {
		const { dialog } = await openEdit({
			...active,
			app: "linkwarden",
			noteScope: "none",
		});
		expect(
			await within(dialog).findByRole("radio", { name: "Later" }),
		).toBeChecked();
		expect(
			within(dialog).queryByRole("radiogroup", { name: "Category for notes" }),
		).toBeNull();
	});

	it("keeps the dialog open with an error", async () => {
		const { dialog, user, api } = await openEdit(active);
		api.set({ "PATCH /api/app-tokens/t1": apiError(409, "token_inactive") });
		await user.click(
			await within(dialog).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		expect(await within(dialog).findByRole("alert")).toHaveTextContent(
			"This app is no longer active.",
		);
	});
});
