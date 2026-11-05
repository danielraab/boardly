import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{ id: "work", kind: "folder", title: "Work", children: [] },
		{ id: "home", kind: "folder", title: "Home", children: [] },
	],
};

const categories = [
	{ id: "inbox", title: "Inbox", icon: null, color: null, parentId: null },
	{ id: "ideas", title: "Ideas", icon: null, color: null, parentId: "inbox" },
];

function mockShareApi(
	shareFolderId: string | null = null,
	noteCategoryId: string | null = null,
) {
	return mockApi({
		"GET /api/me": signedIn({ shareFolderId, noteCategoryId }),
		"GET /api/note-categories": { body: categories },
		"POST /api/notes": ({ body }) => ({
			status: 201,
			body: { id: "n1", ...(body as { categoryId: string | null }) },
		}),
		"GET /api/favourites": { body: tree },
		"GET /api/tags": { body: [] },
		"POST /api/favourites": ({ body }) => ({
			status: 201,
			body: { id: "new", kind: "link", ...(body as object) },
		}),
	});
}

/**
 * Makes creating folders and categories stateful, so a created one is listed
 * when the page fetches again.
 */
function withCreates(api: ReturnType<typeof mockApi>) {
	const folders: FavouriteNode = structuredClone(tree);
	const cats = structuredClone(categories);
	api.set({
		"GET /api/favourites": () => ({ body: folders }),
		"GET /api/note-categories": () => ({ body: cats }),
		"POST /api/favourites": ({ body }) => {
			const fields = body as { parentId: string; kind: string; title: string };
			const node: FavouriteNode =
				fields.kind === "folder"
					? {
							id: `f-${fields.title}`,
							kind: "folder",
							title: fields.title,
							children: [],
						}
					: ({ id: "new", ...fields } as unknown as FavouriteNode);
			if (node.kind === "folder") {
				const parent =
					fields.parentId === "root"
						? folders
						: folders.children?.find((c) => c.id === fields.parentId);
				parent?.children?.push(node);
			}
			return { status: 201, body: node };
		},
		"POST /api/note-categories": ({ body }) => {
			const fields = body as (typeof categories)[number];
			const created = { ...fields, id: `c-${fields.title}` };
			cats.push(created);
			return { status: 201, body: created };
		},
	});
	return api;
}

const share = (params: Record<string, string>) =>
	`/share?${new URLSearchParams(params)}`;

async function folderRadio(name: string) {
	return screen.findByRole("radio", { name });
}

describe("save link page", () => {
	it("prefills url and title", async () => {
		mockShareApi();
		renderApp(share({ title: "Docs", url: "https://example.org/docs" }));
		expect(await screen.findByLabelText("URL")).toHaveValue(
			"https://example.org/docs",
		);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Docs");
	});

	it("takes the URL out of the shared text", async () => {
		mockShareApi();
		renderApp(share({ text: "Read this https://example.org/a" }));
		expect(await screen.findByLabelText("URL")).toHaveValue(
			"https://example.org/a",
		);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Read this");
	});

	it("starts empty in the root folder without a share", async () => {
		mockShareApi();
		renderApp("/share");
		expect(await screen.findByLabelText("URL")).toHaveValue("");
		expect(screen.getByRole("radio", { name: "Link" })).toBeChecked();
		expect(await folderRadio("Favourites")).toBeChecked();
	});

	it("shows the title before the URL and the background hint", async () => {
		mockShareApi();
		renderApp("/share");
		const title = await screen.findByLabelText("Title (optional)");
		const [first, second] = screen.getAllByRole("textbox");
		expect(first).toBe(title);
		expect(second).toBe(screen.getByLabelText("URL"));
		expect(
			screen.getByText(
				"The description and preview image are loaded from the page in the background. You can edit them afterwards.",
			),
		).toBeVisible();
	});

	it("saves into the chosen folder with tags", async () => {
		const api = mockShareApi();
		const { user } = renderApp(share({ url: "https://example.org/docs" }));

		await user.click(await folderRadio("Work"));
		await user.type(screen.getByLabelText("Tags"), "docs{Enter}");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(
			await screen.findByRole("heading", { name: "Link saved" }),
		).toBeVisible();
		expect(screen.getByText("It's now in “Work”.")).toBeVisible();
		expect(api.callsTo("POST", "/api/favourites")[0].body).toEqual({
			parentId: "work",
			kind: "link",
			url: "https://example.org/docs",
			title: "",
			tags: ["docs"],
		});
	});

	it("preselects the default folder", async () => {
		mockShareApi("home");
		renderApp("/share");
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("doesn't remember another folder over the default", async () => {
		mockShareApi("home");
		const { user, router } = renderApp(share({ url: "https://example.org" }));
		await user.click(await folderRadio("Work"));
		await user.click(screen.getByRole("button", { name: "Save" }));
		await screen.findByRole("heading", { name: "Link saved" });

		await router.navigate({ to: "/" });
		await router.navigate({ to: "/share" });
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("falls back to the root when the default folder is gone", async () => {
		mockShareApi("deleted");
		renderApp("/share");
		expect(await folderRadio("Favourites")).toBeChecked();
	});

	it("shows how many links are on the shared URL's domain", async () => {
		const api = mockShareApi();
		const onGithub = (id: string): FavouriteNode => ({
			id,
			kind: "link",
			title: id,
			url: `https://github.com/${id}`,
		});
		api.set({
			"GET /api/favourites": {
				body: {
					...tree,
					children: [
						{ ...tree.children?.[0], children: [onGithub("a")] },
						onGithub("b"),
					],
				},
			},
		});
		renderApp(share({ url: "https://github.com/x" }));
		const hint = await screen.findByRole("link", {
			name: "2 favourites on github.com",
		});
		expect(hint).toHaveAttribute("href", "/favourites?domain=github.com");
		expect(hint).toHaveAttribute("target", "_blank");
	});

	it("rejects an invalid URL", async () => {
		const api = mockShareApi();
		const { user } = renderApp("/share");
		await user.type(await screen.findByLabelText("URL"), "javascript:alert(1)");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"Enter a web address that starts with http:// or https://.",
		);
		expect(api.callsTo("POST", "/api/favourites")).toHaveLength(0);
	});

	it("offers to save another link with an empty form", async () => {
		mockShareApi();
		const { user, router } = renderApp(share({ url: "https://example.org" }));
		await user.click(await screen.findByRole("button", { name: "Save" }));
		await user.click(
			await screen.findByRole("button", { name: "Save another link" }),
		);

		expect(await screen.findByLabelText("URL")).toHaveValue("");
		await waitFor(() => expect(router.state.location.href).toBe("/share"));
	});

	it("creates a subfolder of the chosen folder and saves into it", async () => {
		const api = withCreates(mockShareApi());
		const { user } = renderApp(share({ url: "https://example.org/docs" }));
		await user.click(await folderRadio("Work"));
		await user.click(screen.getByRole("button", { name: "New folder" }));
		const dialog = await screen.findByRole("dialog");
		expect(within(dialog).getByText("In “Work”")).toBeVisible();
		await user.type(within(dialog).getByLabelText("Name"), "Reading");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

		expect(api.callsTo("POST", "/api/favourites")[0].body).toEqual({
			parentId: "work",
			kind: "folder",
			title: "Reading",
		});
		await waitFor(async () =>
			expect(await folderRadio("Reading")).toBeChecked(),
		);
		expect(screen.getByLabelText("URL")).toHaveValue(
			"https://example.org/docs",
		);

		await user.click(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findByText("It's now in “Reading”.")).toBeVisible();
		expect(api.callsTo("POST", "/api/favourites")[1].body).toMatchObject({
			parentId: "f-Reading",
			kind: "link",
		});
	});

	it("creates a top-level folder from the root", async () => {
		const api = withCreates(mockShareApi());
		const { user } = renderApp("/share");
		expect(await folderRadio("Favourites")).toBeChecked();
		await user.click(screen.getByRole("button", { name: "New folder" }));
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "Inbox");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(async () => expect(await folderRadio("Inbox")).toBeChecked());
		expect(api.callsTo("POST", "/api/favourites")[0].body).toMatchObject({
			parentId: "root",
			kind: "folder",
		});
	});

	it("keeps the form when the new folder dialog is cancelled", async () => {
		const api = withCreates(mockShareApi());
		const { user } = renderApp(
			share({ title: "Docs", url: "https://example.org/docs" }),
		);
		await user.click(await folderRadio("Home"));
		await user.type(screen.getByLabelText("Tags"), "docs{Enter}");
		await user.click(screen.getByRole("button", { name: "New folder" }));
		const dialog = await screen.findByRole("dialog");
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

		expect(api.callsTo("POST", "/api/favourites")).toHaveLength(0);
		expect(screen.getByLabelText("URL")).toHaveValue(
			"https://example.org/docs",
		);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Docs");
		expect(screen.getByText("docs")).toBeVisible();
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("doesn't make a new folder the default", async () => {
		withCreates(mockShareApi("home"));
		const { user, router } = renderApp(share({ url: "https://example.org" }));
		await folderRadio("Home");
		await user.click(screen.getByRole("button", { name: "New folder" }));
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "Reading");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(async () =>
			expect(await folderRadio("Reading")).toBeChecked(),
		);
		await user.click(screen.getByRole("button", { name: "Save" }));
		await screen.findByRole("heading", { name: "Link saved" });

		await router.navigate({ to: "/" });
		await router.navigate({ to: "/share" });
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("goes home on cancel", async () => {
		mockShareApi();
		const { user, router } = renderApp("/share");
		await user.click(await screen.findByRole("link", { name: "Cancel" }));
		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
	});
});

describe("save note from a share", () => {
	const categoryRadio = async (name: string) =>
		within(
			await screen.findByRole("radiogroup", { name: "Category" }),
		).findByRole("radio", { name });

	it("starts as a note for shared text without a URL", async () => {
		mockShareApi(null, "inbox");
		renderApp(share({ title: "Plan", text: "Meet at 8" }));

		expect(
			await screen.findByRole("heading", { name: "Save note" }),
		).toBeVisible();
		expect(screen.getByRole("radio", { name: "Note" })).toBeChecked();
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Plan");
		expect(screen.getByLabelText("Text")).toHaveValue("Meet at 8");
		expect(await categoryRadio("Inbox")).toBeChecked();
	});

	it("starts without a category when there is no default", async () => {
		mockShareApi();
		renderApp(share({ text: "Meet at 8" }));
		expect(await categoryRadio("No category")).toBeChecked();
	});

	it("adds the URL to the text and keeps both forms when switching", async () => {
		mockShareApi();
		const { user } = renderApp(
			share({ text: "Look at this", url: "https://example.org/a" }),
		);
		const linkTitle = await screen.findByLabelText("Title (optional)");
		await user.clear(linkTitle);
		await user.type(linkTitle, "Edited");

		await user.click(screen.getByRole("radio", { name: "Note" }));
		expect(screen.getByLabelText("Text")).toHaveValue(
			"Look at this\nhttps://example.org/a",
		);
		await user.click(screen.getByRole("radio", { name: "Link" }));
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Edited");
	});

	it("saves the note into the chosen category", async () => {
		const api = mockShareApi(null, "inbox");
		const { user } = renderApp(share({ title: "Plan", text: "Meet at 8" }));
		await user.click(await categoryRadio("Ideas"));
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(
			await screen.findByRole("heading", { name: "Note saved" }),
		).toBeVisible();
		expect(screen.getByText("It's now in “Inbox › Ideas”.")).toBeVisible();
		expect(screen.getByRole("link", { name: "Open note" })).toHaveAttribute(
			"href",
			"/notes/n1",
		);
		expect(api.callsTo("POST", "/api/notes")[0].body).toEqual({
			title: "Plan",
			text: "Meet at 8",
			categoryId: "ideas",
		});
	});

	const newCategory = async (user: ReturnType<typeof renderApp>["user"]) => {
		await user.click(screen.getByRole("button", { name: "New category" }));
		return screen.findByRole("dialog", { name: "New category" });
	};

	it("creates a subcategory of the chosen category and saves into it", async () => {
		const api = withCreates(mockShareApi(null, "inbox"));
		const { user } = renderApp(share({ title: "Plan", text: "Meet at 8" }));
		expect(await categoryRadio("Inbox")).toBeChecked();
		const dialog = await newCategory(user);
		expect(
			within(dialog).getByRole("combobox", { name: "Parent category" }),
		).toHaveValue("inbox");
		await user.type(
			within(dialog).getByRole("textbox", { name: "Title" }),
			"Garden",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

		expect(api.callsTo("POST", "/api/note-categories")[0].body).toMatchObject({
			title: "Garden",
			parentId: "inbox",
		});
		await waitFor(async () =>
			expect(await categoryRadio("Garden")).toBeChecked(),
		);
		expect(screen.getByLabelText("Text")).toHaveValue("Meet at 8");

		await user.click(screen.getByRole("button", { name: "Save" }));
		expect(
			await screen.findByText("It's now in “Inbox › Garden”."),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/notes")[0].body).toMatchObject({
			categoryId: "c-Garden",
		});
	});

	it("creates a top-level category from no category", async () => {
		const api = withCreates(mockShareApi());
		const { user } = renderApp(share({ text: "Meet at 8" }));
		expect(await categoryRadio("No category")).toBeChecked();
		const dialog = await newCategory(user);
		expect(
			within(dialog).getByRole("combobox", { name: "Parent category" }),
		).toHaveValue("");
		await user.type(
			within(dialog).getByRole("textbox", { name: "Title" }),
			"Travel",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(async () =>
			expect(await categoryRadio("Travel")).toBeChecked(),
		);
		expect(api.callsTo("POST", "/api/note-categories")[0].body).toMatchObject({
			parentId: null,
		});
	});

	it("keeps the form when the new category dialog is cancelled", async () => {
		const api = withCreates(mockShareApi());
		const { user } = renderApp(share({ title: "Plan", text: "Meet at 8" }));
		await user.click(await categoryRadio("Ideas"));
		const dialog = await newCategory(user);
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

		expect(api.callsTo("POST", "/api/note-categories")).toHaveLength(0);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Plan");
		expect(screen.getByLabelText("Text")).toHaveValue("Meet at 8");
		expect(await categoryRadio("Ideas")).toBeChecked();
	});

	it("doesn't make a new category the default", async () => {
		withCreates(mockShareApi(null, "inbox"));
		const { user, router } = renderApp(share({ text: "Meet at 8" }));
		await categoryRadio("Inbox");
		const dialog = await newCategory(user);
		await user.type(
			within(dialog).getByRole("textbox", { name: "Title" }),
			"Garden",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(async () =>
			expect(await categoryRadio("Garden")).toBeChecked(),
		);
		await user.click(screen.getByRole("button", { name: "Save" }));
		await screen.findByRole("heading", { name: "Note saved" });

		await router.navigate({ to: "/" });
		await router.navigate({ to: "/share", search: { text: "Again" } });
		expect(await categoryRadio("Inbox")).toBeChecked();
	});

	it("explains a category that is gone", async () => {
		const api = mockShareApi(null, "inbox");
		api.set({ "POST /api/notes": apiError(400, "invalid_note") });
		const { user } = renderApp(share({ text: "Meet at 8" }));
		await categoryRadio("Inbox");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This category no longer exists. Choose another one.",
		);
		expect(screen.getByLabelText("Text")).toHaveValue("Meet at 8");
	});
});

describe("signed-out share", () => {
	// Continuing after sign-in is covered by the passkey tests.
	it("sends the visitor to the login page with the share as redirect", async () => {
		const path = share({ url: "https://example.org" });
		const { router } = renderApp(path);

		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		expect(router.state.location.search).toEqual({ redirect: path });
	});
});
