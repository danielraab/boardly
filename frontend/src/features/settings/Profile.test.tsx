import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

async function openMenuText(user: ReturnType<typeof renderApp>["user"]) {
	await user.click(screen.getByRole("button", { name: "User menu" }));
	const menu = await screen.findByRole("menu");
	const text = menu.textContent ?? "";
	await user.keyboard("{Escape}");
	await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	return text;
}

describe("settings profile tab", () => {
	it("limits the page to the narrow width", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		await screen.findByRole("heading", { name: "Settings" });
		expect(screen.getByRole("main").firstElementChild).toHaveClass(
			"mx-auto",
			"max-w-3xl",
		);
	});

	it("shows the email and an empty name field", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		expect(
			await screen.findByRole("heading", { name: "Settings" }),
		).toBeVisible();
		expect(screen.getByText("ann@example.org")).toBeVisible();
		expect(screen.getByLabelText("Display name")).toHaveValue("");
		expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
	});

	it("sets the name and updates the menu without a reload", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": ({ body }) =>
				signedIn({ name: (body as { name: string }).name.trim() }),
		});
		const { user } = renderApp("/settings");

		await user.type(await screen.findByLabelText("Display name"), "Ann Smith");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(screen.getByLabelText("Display name")).toHaveValue("Ann Smith");
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			name: "Ann Smith",
		});
		expect(await openMenuText(user)).toContain("Ann Smith");
		expect(api.callsTo("GET", "/api/me")).toHaveLength(1);
	});

	it("clears the name with null", async () => {
		const api = mockApi({
			"GET /api/me": signedIn({ name: "Ann" }),
			"PATCH /api/me": signedIn({ name: null }),
		});
		const { user } = renderApp("/settings");

		const input = await screen.findByLabelText("Display name");
		expect(input).toHaveValue("Ann");
		await user.clear(input);
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({ name: null });
		const menu = await openMenuText(user);
		expect(menu).not.toContain("Ann ");
		expect(menu.startsWith("ann@example.org")).toBe(true);
	});

	it("explains a name that is too long", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": apiError(400, "invalid_name"),
		});
		const { user } = renderApp("/settings");
		await user.type(
			await screen.findByLabelText("Display name"),
			"x".repeat(101),
		);
		await user.click(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The name can be at most 100 characters long.",
		);
		expect(within(document.body).queryByText("Saved.")).toBeNull();
	});
});

describe("settings profile slug", () => {
	it("changes the slug after confirmation and updates the cache", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": ({ body }) =>
				signedIn({ slug: (body as { slug: string }).slug }),
		});
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		expect(input).toHaveValue("ann");
		await user.clear(input);
		await user.type(input, "AnnSmith");
		expect(input).toHaveValue("annsmith");
		expect(
			screen.getByText("Links that use the old address will stop working."),
		).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog", {
			name: "Change address?",
		});
		expect(within(confirm).getByText("/ann/…")).toBeVisible();
		expect(within(confirm).getByText("/annsmith/…")).toBeVisible();
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);

		expect(
			await screen.findByText("Your dashboards are at /annsmith/…"),
		).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			slug: "annsmith",
		});
	});

	it("saves nothing when the confirmation is cancelled", async () => {
		const api = mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.type(input, "2");
		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog");
		await user.click(within(confirm).getByRole("button", { name: "Cancel" }));

		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(input).toHaveValue("ann2");
		expect(api.callsTo("PATCH", "/api/me")).toHaveLength(0);
	});

	it("blocks reserved words", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.clear(input);
		await user.type(input, "login");
		expect(screen.getByRole("alert")).toHaveTextContent(
			"This address is reserved by Boardly.",
		);
		expect(screen.getByRole("button", { name: "Save address" })).toBeDisabled();
	});

	it("shows a taken slug at the field", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": apiError(409, "slug_taken"),
		});
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.clear(input);
		await user.type(input, "bob");
		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog");
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);
		expect(
			await screen.findByText("This address is already taken."),
		).toBeVisible();
		expect(input).toHaveValue("bob");
	});
});

describe("settings default folder", () => {
	const tree = {
		id: "root",
		kind: "folder",
		title: null,
		children: [
			{ id: "inbox", kind: "folder", title: "Inbox", children: [] },
			{ id: "work", kind: "folder", title: "Work", children: [] },
		],
	};

	it("saves the chosen folder", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/favourites": { body: tree },
			"PATCH /api/me": signedIn({ shareFolderId: "inbox" }),
		});
		const { user } = renderApp("/settings");

		expect(
			await screen.findByRole("radio", { name: "Favourites" }),
		).toBeChecked();
		const save = screen.getByRole("button", { name: "Save default folder" });
		expect(save).toBeDisabled();
		await user.click(screen.getByRole("radio", { name: "Inbox" }));
		await user.click(save);

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			shareFolderId: "inbox",
		});
		expect(screen.getByRole("radio", { name: "Inbox" })).toBeChecked();
	});

	it("saves the root as null", async () => {
		const api = mockApi({
			"GET /api/me": signedIn({ shareFolderId: "inbox" }),
			"GET /api/favourites": { body: tree },
			"PATCH /api/me": signedIn({ shareFolderId: null }),
		});
		const { user } = renderApp("/settings");

		expect(await screen.findByRole("radio", { name: "Inbox" })).toBeChecked();
		await user.click(screen.getByRole("radio", { name: "Favourites" }));
		await user.click(
			screen.getByRole("button", { name: "Save default folder" }),
		);

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			shareFolderId: null,
		});
	});

	it("shows when the folder is gone", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/favourites": { body: tree },
			"PATCH /api/me": apiError(400, "invalid_folder"),
		});
		const { user } = renderApp("/settings");

		await user.click(await screen.findByRole("radio", { name: "Work" }));
		await user.click(
			screen.getByRole("button", { name: "Save default folder" }),
		);

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This folder no longer exists. Choose another one.",
		);
	});
});

describe("default category for new notes", () => {
	const categories = [
		{ id: "home", title: "Home", icon: null, color: null, parentId: null },
		{
			id: "garden",
			title: "Garden",
			icon: null,
			color: null,
			parentId: "home",
		},
		{ id: "inbox", title: "Inbox", icon: null, color: null, parentId: null },
	];
	const picker = () =>
		screen.findByRole("radiogroup", {
			name: "Default category for new notes",
		});

	it("saves the chosen category", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/note-categories": { body: categories },
			"PATCH /api/me": signedIn({ noteCategoryId: "garden" }),
		});
		const { user } = renderApp("/settings");

		const group = await picker();
		expect(
			await within(group).findByRole("radio", { name: "No category" }),
		).toBeChecked();
		const save = screen.getByRole("button", { name: "Save default category" });
		expect(save).toBeDisabled();
		await user.click(within(group).getByRole("radio", { name: "Garden" }));
		await user.click(save);

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			noteCategoryId: "garden",
		});
		expect(within(group).getByRole("radio", { name: "Garden" })).toBeChecked();
	});

	it("saves no category as null", async () => {
		const api = mockApi({
			"GET /api/me": signedIn({ noteCategoryId: "inbox" }),
			"GET /api/note-categories": { body: categories },
			"PATCH /api/me": signedIn({ noteCategoryId: null }),
		});
		const { user } = renderApp("/settings");

		const group = await picker();
		expect(
			await within(group).findByRole("radio", { name: "Inbox" }),
		).toBeChecked();
		await user.click(within(group).getByRole("radio", { name: "No category" }));
		await user.click(
			screen.getByRole("button", { name: "Save default category" }),
		);

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			noteCategoryId: null,
		});
	});

	it("shows when the category is gone", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/note-categories": { body: categories },
			"PATCH /api/me": apiError(400, "invalid_category"),
		});
		const { user } = renderApp("/settings");

		const group = await picker();
		await user.click(
			await within(group).findByRole("radio", { name: "Inbox" }),
		);
		await user.click(
			screen.getByRole("button", { name: "Save default category" }),
		);

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This category no longer exists. Choose another one.",
		);
	});
});
