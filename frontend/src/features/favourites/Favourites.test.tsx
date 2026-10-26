import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { AppToken } from "#/features/settings/appTokens/api";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import type { FavouriteNode } from "./api";
import { applyMove, findNode, findParent } from "./tree";

const link = (
	id: string,
	extra: Partial<FavouriteNode> = {},
): FavouriteNode => ({
	id,
	kind: "link",
	title: id,
	url: `https://example.org/${id}`,
	...extra,
});
const folder = (id: string, ...children: FavouriteNode[]): FavouriteNode => ({
	id,
	kind: "folder",
	title: id,
	children,
});

/**
 * An in-memory favourites API: it answers like the server and keeps the tree,
 * so the page refetches what the requests changed.
 */
function fakeServer(initial: FavouriteNode[], tags: string[] = []) {
	let tree: FavouriteNode = {
		id: "root",
		kind: "folder",
		title: null,
		children: initial,
	};
	let next = 1;
	const clone = () => structuredClone(tree);

	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/favourites": () => ({ body: clone() }),
		"POST /api/favourites": ({ body }) => {
			const b = body as FavouriteNode & { parentId: string };
			const node: FavouriteNode =
				b.kind === "folder"
					? { id: `n${next++}`, kind: "folder", title: b.title }
					: {
							id: `n${next++}`,
							kind: "link",
							title: b.title || null,
							url: b.url,
							tags: b.tags,
						};
			const parent = findNode(tree, b.parentId);
			if (parent) parent.children = [...(parent.children ?? []), node];
			return { status: 201, body: node };
		},
		"POST /api/favourites/bulk": ({ body }) => {
			const op = body as {
				ids: string[];
				action: string;
				tags?: string[];
				parentId?: string;
			};
			for (const id of op.ids) {
				const found = findParent(tree, id);
				if (!found)
					return { status: 404, body: { error: { code: "not_found" } } };
				const node = found.parent.children?.[found.index] as FavouriteNode;
				if (op.action === "delete" || op.action === "move") {
					found.parent.children?.splice(found.index, 1);
				}
				if (op.action === "move") {
					const target = findNode(tree, op.parentId ?? "");
					if (target) target.children = [...(target.children ?? []), node];
				}
				const tags = op.tags ?? [];
				if (op.action === "addTags") {
					node.tags = [...new Set([...(node.tags ?? []), ...tags])].sort();
				}
				if (op.action === "removeTags") {
					node.tags = (node.tags ?? []).filter((t) => !tags.includes(t));
				}
				if (op.action === "setTags") node.tags = tags;
			}
			return { status: 204 };
		},
		"GET /api/tags": ({ url }) => {
			const q = (url.searchParams.get("q") ?? "").toLowerCase();
			return {
				body: tags
					.filter((t) => t.toLowerCase().startsWith(q))
					.map((name) => ({ id: name, name })),
			};
		},
	});

	// Per-node routes are added for every node, since the mock matches exact paths.
	const route = (node: FavouriteNode) => {
		api.set({
			[`PATCH /api/favourites/${node.id}`]: ({ body }) => {
				const target = findNode(tree, node.id);
				const { removePreview, ...patch } = body as FavouriteNode & {
					removePreview?: boolean;
				};
				Object.assign(target ?? {}, patch);
				if (target && removePreview) delete target.previewImageId;
				if (target && patch.description === "") delete target.description;
				return { body: target };
			},
			[`POST /api/favourites/${node.id}/move`]: ({ body }) => {
				const { parentId, position } = body as {
					parentId: string;
					position: number;
				};
				tree = applyMove(tree, node.id, parentId, position);
				return { status: 204 };
			},
			[`DELETE /api/favourites/${node.id}`]: ({ url }) => {
				const found = findParent(tree, node.id);
				const siblings = found?.parent.children ?? [];
				const replacement =
					url.searchParams.get("children") === "move"
						? (node.children ?? [])
						: [];
				siblings.splice(found?.index ?? 0, 1, ...replacement);
				return { status: 204 };
			},
		});
		node.children?.forEach(route);
	};
	tree.children?.forEach(route);

	return {
		api,
		tree: () => tree,
		titles: (id = "root") =>
			(findNode(tree, id)?.children ?? []).map((c) => c.title ?? c.url),
	};
}

function rows() {
	return within(screen.getByRole("list"))
		.getAllByRole("listitem")
		.map((li) => li.querySelector("a, span.font-medium")?.textContent);
}

describe("favourites page", () => {
	it("sends signed-out visitors to /login", async () => {
		const { router } = renderApp("/favourites");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("shows a hint when the tree is empty", async () => {
		fakeServer([]);
		renderApp("/favourites");
		expect(
			await screen.findByRole("heading", { name: "No favourites yet" }),
		).toBeVisible();
	});

	it("shows folders and links, labelled by title or URL", async () => {
		fakeServer([
			folder("Work", link("Docs", { tags: ["reading"] })),
			link("bare", { title: null, url: "https://example.org" }),
		]);
		renderApp("/favourites");
		const docs = await screen.findByRole("link", { name: "Docs" });
		expect(docs).toHaveAttribute("href", "https://example.org/Docs");
		expect(docs).toHaveAttribute("target", "_blank");
		expect(docs).toHaveAttribute("rel", "noopener noreferrer");
		expect(
			screen.getByRole("link", { name: "https://example.org" }),
		).toBeVisible();
		expect(rows()).toEqual(["Work", "Docs", "https://example.org"]);
	});

	it("shows a link's favicon, or a globe without one", async () => {
		fakeServer([link("GitHub", { iconImageId: "img-1" }), link("Plain")]);
		renderApp("/favourites");
		const github = (await screen.findByRole("link", { name: "GitHub" }))
			.parentElement?.parentElement as HTMLElement;
		expect(github.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img-1",
		);
		const plain = screen.getByRole("link", { name: "Plain" }).parentElement
			?.parentElement as HTMLElement;
		expect(plain.querySelector("img")).toBeNull();
		expect(plain.querySelector("[data-icon='globe']")).not.toBeNull();
	});

	it("collapses and expands a folder", async () => {
		fakeServer([folder("Work", link("Docs"))]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Collapse Work" }),
		);
		expect(screen.queryByRole("link", { name: "Docs" })).toBeNull();
		await user.click(screen.getByRole("button", { name: "Expand Work" }));
		expect(screen.getByRole("link", { name: "Docs" })).toBeVisible();
	});
});

/** Opens the edit dialog of a node and returns it. */
async function openEdit(
	user: ReturnType<typeof renderApp>["user"],
	name: string,
) {
	await user.click(await screen.findByRole("button", { name: `Edit ${name}` }));
	return screen.findByRole("dialog");
}

describe("refresh icon", () => {
	it("refreshes a link's favicon from its edit dialog", async () => {
		const server = fakeServer([folder("Work"), link("GitHub")]);
		server.api.set({
			"POST /api/favicons/refresh": () => {
				const target = findNode(server.tree(), "GitHub");
				if (target) target.iconImageId = "img-new";
				return { body: { id: "img-new" } };
			},
		});
		const { user } = renderApp("/favourites");
		// The tree rows have no refresh action.
		await screen.findByRole("link", { name: "GitHub" });
		expect(screen.queryByRole("button", { name: /Refresh icon/ })).toBeNull();

		const dialog = await openEdit(user, "GitHub");
		await user.click(
			within(dialog).getByRole("button", { name: "Refresh icon" }),
		);
		expect(server.api.callsTo("POST", "/api/favicons/refresh")[0].body).toEqual(
			{ url: "https://example.org/GitHub" },
		);
		await waitFor(() =>
			expect(
				within(dialog).getByRole("button", { name: "Refresh icon" })
					.previousElementSibling,
			).toHaveAttribute(
				"src",
				expect.stringMatching(/^\/api\/images\/img-new\?v=/),
			),
		);
		const row = screen.getByRole("link", { name: "GitHub", hidden: true })
			.parentElement?.parentElement as HTMLElement;
		expect(row.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img-new",
		);
		expect(screen.getByRole("dialog")).toBeVisible();
		expect(server.api.callsTo("PATCH", "/api/favourites/GitHub")).toHaveLength(
			0,
		);
	});

	it("is disabled while the refresh runs", async () => {
		const server = fakeServer([link("GitHub")]);
		let finish: () => void = () => {};
		server.api.set({
			"POST /api/favicons/refresh": () =>
				new Promise((resolve) => {
					finish = () => resolve({ body: { id: "img-1" } });
				}),
		});
		const { user } = renderApp("/favourites");
		const dialog = await openEdit(user, "GitHub");
		const button = within(dialog).getByRole("button", {
			name: "Refresh icon",
		});
		await user.click(button);
		await waitFor(() => expect(button).toBeDisabled());
		finish();
		await waitFor(() => expect(button).toBeEnabled());
	});

	it.each([
		["favicon_not_found", 404, "No icon was found for GitHub."],
		[
			"blocked_address",
			400,
			"The icon of GitHub isn't loaded, because the site is on a private network address.",
		],
		[
			"rate_limited",
			429,
			"Too many icon refreshes. Try again in a few minutes.",
		],
		["internal", 500, "The icon of GitHub could not be refreshed."],
	])(
		"explains %s in the dialog and keeps the icon",
		async (code, status, message) => {
			const server = fakeServer([link("GitHub", { iconImageId: "img-1" })]);
			server.api.set({ "POST /api/favicons/refresh": apiError(status, code) });
			const { user } = renderApp("/favourites");
			const dialog = await openEdit(user, "GitHub");
			await user.click(
				within(dialog).getByRole("button", { name: "Refresh icon" }),
			);
			expect(await within(dialog).findByRole("alert")).toHaveTextContent(
				message,
			);
			const row = screen.getByRole("link", { name: "GitHub", hidden: true })
				.parentElement?.parentElement as HTMLElement;
			expect(row.querySelector("img")).toHaveAttribute(
				"src",
				"/api/images/img-1",
			);
		},
	);
});

describe("description and preview", () => {
	it("edits a link's description", async () => {
		const server = fakeServer([link("Docs", { description: "Old" })]);
		const { user } = renderApp("/favourites");
		let dialog = await openEdit(user, "Docs");
		const field = within(dialog).getByLabelText("Description");
		expect(field).toHaveValue("Old");
		await user.clear(field);
		await user.type(field, "Team docs");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("PATCH", "/api/favourites/Docs")[0].body).toEqual(
			{
				title: "Docs",
				url: "https://example.org/Docs",
				tags: [],
				description: "Team docs",
			},
		);

		dialog = await openEdit(user, "Docs");
		expect(within(dialog).getByLabelText("Description")).toHaveValue(
			"Team docs",
		);
	});

	it("rejects a too long description", async () => {
		const server = fakeServer([link("Docs")]);
		const { user } = renderApp("/favourites");
		const dialog = await openEdit(user, "Docs");
		within(dialog).getByLabelText("Description").focus();
		await user.paste("a".repeat(3000));
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"The description can be at most 2048 characters long.",
		);
		expect(server.api.callsTo("PATCH", "/api/favourites/Docs")).toHaveLength(0);
	});

	it("removes the preview when the dialog is saved", async () => {
		const server = fakeServer([link("Docs", { previewImageId: "prev-1" })]);
		const { user } = renderApp("/favourites");
		let dialog = await openEdit(user, "Docs");
		expect(
			within(dialog).getByRole("img", { name: "Preview image of the link" }),
		).toHaveAttribute("src", "/api/images/prev-1");
		await user.click(
			within(dialog).getByRole("button", { name: "Remove preview" }),
		);
		expect(within(dialog).getByText("No preview image")).toBeVisible();
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(
			server.api.callsTo("PATCH", "/api/favourites/Docs")[0].body,
		).toMatchObject({ removePreview: true });

		dialog = await openEdit(user, "Docs");
		expect(within(dialog).getByText("No preview image")).toBeVisible();
	});

	it("keeps the preview when the dialog is cancelled", async () => {
		const server = fakeServer([link("Docs", { previewImageId: "prev-1" })]);
		const { user } = renderApp("/favourites");
		let dialog = await openEdit(user, "Docs");
		await user.click(
			within(dialog).getByRole("button", { name: "Remove preview" }),
		);
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("PATCH", "/api/favourites/Docs")).toHaveLength(0);

		dialog = await openEdit(user, "Docs");
		expect(
			within(dialog).getByRole("img", { name: "Preview image of the link" }),
		).toBeVisible();
	});

	it("shows only the name when editing a folder", async () => {
		fakeServer([folder("Work")]);
		const { user } = renderApp("/favourites");
		const dialog = await openEdit(user, "Work");
		expect(within(dialog).getAllByRole("textbox")).toHaveLength(1);
		expect(within(dialog).queryByText("Preview image")).toBeNull();
		expect(
			within(dialog).queryByRole("button", { name: "Refresh icon" }),
		).toBeNull();
	});

	it("shows the title first and the background hint when adding a link", async () => {
		fakeServer([]);
		const { user } = renderApp("/favourites");
		await user.click(await screen.findByRole("button", { name: "Add link" }));
		const dialog = await screen.findByRole("dialog");
		const [first, second] = within(dialog).getAllByRole("textbox");
		expect(first).toBe(within(dialog).getByLabelText("Title (optional)"));
		expect(second).toBe(within(dialog).getByLabelText("URL"));
		expect(
			within(dialog).getByText(
				"The description and preview image are loaded from the page in the background. You can edit them afterwards.",
			),
		).toBeVisible();
		expect(within(dialog).queryByLabelText("Description")).toBeNull();
	});
});

describe("add and edit", () => {
	it("creates a folder inside another folder", async () => {
		const server = fakeServer([folder("Work")]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Add folder to Work" }),
		);
		const dialog = await screen.findByRole("dialog");
		expect(within(dialog).getByText("In “Work”")).toBeVisible();
		await user.type(within(dialog).getByLabelText("Name"), "Tools");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(() => expect(server.titles("Work")).toEqual(["Tools"]));
		expect(server.api.callsTo("POST", "/api/favourites")[0].body).toEqual({
			parentId: "Work",
			kind: "folder",
			title: "Tools",
		});
		expect(await screen.findByText("Tools")).toBeVisible();
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	});

	it("edits a link's title and URL", async () => {
		const server = fakeServer([link("Docs")]);
		const { user } = renderApp("/favourites");
		await user.click(await screen.findByRole("button", { name: "Edit Docs" }));
		const dialog = await screen.findByRole("dialog");
		const title = within(dialog).getByLabelText("Title (optional)");
		await user.clear(title);
		await user.type(title, "Documentation");
		const url = within(dialog).getByLabelText("URL");
		await user.clear(url);
		await user.type(url, "https://example.org/doc");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await screen.findByRole("link", { name: "Documentation" });
		expect(server.api.callsTo("PATCH", "/api/favourites/Docs")[0].body).toEqual(
			{
				title: "Documentation",
				url: "https://example.org/doc",
				tags: [],
				description: "",
			},
		);
	});

	it("validates names and URLs before sending", async () => {
		const server = fakeServer([]);
		const { user } = renderApp("/favourites");

		await user.click(await screen.findByRole("button", { name: "Add folder" }));
		let dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Name"), "   ");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"Enter a name.",
		);
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());

		await user.click(screen.getByRole("button", { name: "Add link" }));
		dialog = await screen.findByRole("dialog");
		await user.type(
			within(dialog).getByLabelText("URL"),
			"javascript:alert(1)",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"Enter a web address that starts with http:// or https://.",
		);
		expect(server.api.callsTo("POST", "/api/favourites")).toHaveLength(0);
	});
});

describe("tags", () => {
	it("suggests existing tags by prefix, without the link's own tags", async () => {
		fakeServer(
			[link("Docs", { tags: ["reading"] })],
			["reading", "recipes", "tools"],
		);
		const { user } = renderApp("/favourites");
		await user.click(await screen.findByRole("button", { name: "Edit Docs" }));
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Tags"), "re");

		const option = await screen.findByRole("option", { name: "recipes" });
		expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual([
			"recipes",
			"Add “re”",
		]);

		await user.click(option);
		expect(within(dialog).getByText("recipes")).toBeVisible();
	});

	it("adds a new tag with Enter and sends it on save", async () => {
		const server = fakeServer([link("Docs")]);
		const { user } = renderApp("/favourites");
		await user.click(await screen.findByRole("button", { name: "Edit Docs" }));
		const dialog = await screen.findByRole("dialog");
		await user.type(within(dialog).getByLabelText("Tags"), "news{Enter}");
		expect(
			within(dialog).getByRole("button", { name: "Remove tag news" }),
		).toBeVisible();

		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() =>
			expect(
				server.api.callsTo("PATCH", "/api/favourites/Docs")[0]?.body,
			).toMatchObject({ tags: ["news"] }),
		);
	});
});

describe("delete", () => {
	it("deletes a link after confirmation", async () => {
		const server = fakeServer([link("Docs"), link("Other")]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Delete Docs" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent("“Docs” will be removed");
		await user.click(within(dialog).getByRole("button", { name: "Delete" }));

		await waitFor(() => expect(server.titles()).toEqual(["Other"]));
		expect(server.api.callsTo("DELETE", "/api/favourites/Docs")[0].search).toBe(
			"",
		);
		await waitFor(() =>
			expect(screen.queryByRole("link", { name: "Docs" })).toBeNull(),
		);
	});

	it("keeps the tree when the dialog is cancelled", async () => {
		const server = fakeServer([link("Docs")]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Delete Docs" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(server.api.callsTo("DELETE", "/api/favourites/Docs")).toHaveLength(
			0,
		);
		expect(screen.getByRole("link", { name: "Docs" })).toBeVisible();
	});

	it("asks what happens to a folder's contents and can keep them", async () => {
		const server = fakeServer([
			link("A"),
			folder("Work", link("Tools"), link("Docs")),
			link("B"),
		]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Delete Work" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent("“Work” contains 2 items.");
		const confirm = within(dialog).getByRole("button", { name: "Delete" });
		expect(confirm).toBeDisabled();

		await user.click(
			within(dialog).getByRole("radio", {
				name: "Keep the contents and move them to “Favourites”",
			}),
		);
		await user.click(confirm);

		await waitFor(() =>
			expect(server.titles()).toEqual(["A", "Tools", "Docs", "B"]),
		);
		expect(server.api.callsTo("DELETE", "/api/favourites/Work")[0].search).toBe(
			"?children=move",
		);
	});

	it("can delete a folder with everything in it", async () => {
		const server = fakeServer([folder("Work", link("Docs")), link("B")]);
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Delete Work" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(
			within(dialog).getByRole("radio", {
				name: "Delete the folder and everything in it",
			}),
		);
		await user.click(within(dialog).getByRole("button", { name: "Delete" }));

		await waitFor(() => expect(server.titles()).toEqual(["B"]));
		expect(server.api.callsTo("DELETE", "/api/favourites/Work")[0].search).toBe(
			"?children=delete",
		);
	});
});

describe("moving with the keyboard", () => {
	it("reorders within a folder", async () => {
		const server = fakeServer([link("A"), link("B"), link("C")]);
		const { user } = renderApp("/favourites");
		const handle = await screen.findByRole("button", { name: "Move C" });
		handle.focus();
		await user.keyboard(" ");
		expect(handle).toHaveAttribute("aria-pressed", "true");
		await user.keyboard("{ArrowUp}{ArrowUp}");
		expect(screen.getByText("Position 1 in Favourites.")).toBeInTheDocument();
		await user.keyboard(" ");

		expect(
			server.api.callsTo("POST", "/api/favourites/C/move")[0].body,
		).toEqual({ parentId: "root", position: 0 });
		await waitFor(() => expect(rows()).toEqual(["C", "A", "B"]));
		expect(handle).toHaveFocus();
	});

	it("moves a link into a folder", async () => {
		const server = fakeServer([folder("Archive", link("Old")), link("Docs")]);
		const { user } = renderApp("/favourites");
		(await screen.findByRole("button", { name: "Move Docs" })).focus();
		// Slots above Docs: end of Archive, before Old, start of root.
		await user.keyboard(" {ArrowUp} ");

		expect(
			server.api.callsTo("POST", "/api/favourites/Docs/move")[0].body,
		).toEqual({ parentId: "Archive", position: 1 });
		await waitFor(() =>
			expect(server.titles("Archive")).toEqual(["Old", "Docs"]),
		);
	});

	it("cancels with Escape", async () => {
		const server = fakeServer([link("A"), link("B")]);
		const { user } = renderApp("/favourites");
		(await screen.findByRole("button", { name: "Move B" })).focus();
		await user.keyboard(" {ArrowUp}{Escape}");
		expect(screen.getByText("Moving B cancelled.")).toBeInTheDocument();
		expect(server.api.callsTo("POST", "/api/favourites/B/move")).toHaveLength(
			0,
		);
	});

	it("restores the order when the server rejects a move", async () => {
		const server = fakeServer([link("A"), link("B")]);
		server.api.set({
			"POST /api/favourites/B/move": apiError(400, "invalid_move"),
		});
		const { user } = renderApp("/favourites");
		(await screen.findByRole("button", { name: "Move B" })).focus();
		await user.keyboard(" {ArrowUp} ");

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The item could not be moved. The list has been restored.",
		);
		await waitFor(() => expect(rows()).toEqual(["A", "B"]));
	});
});

describe("delete with connected apps", () => {
	const token = (
		id: string,
		rootId: string,
		status: AppToken["status"] = "active",
	): AppToken => ({
		id,
		name: id,
		app: "karakeep",
		prefix: "bdly_ABCDEFG",
		rootId,
		createdAt: "2026-02-01T10:00:00Z",
		lastUsedAt: null,
		expiresAt: null,
		revokedAt: null,
		status,
	});

	async function openDelete(tokens: AppToken[], name: string) {
		const server = fakeServer([
			folder("Work", folder("Tools", link("Git")), link("Docs")),
			folder("Other"),
		]);
		server.api.set({ "GET /api/app-tokens": { body: tokens } });
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: `Delete ${name}` }),
		);
		const dialog = await screen.findByRole("alertdialog");
		return { user, dialog, server };
	}

	const warning = /connected apps? uses? this folder/;

	it("warns about the active tokens rooted at the folder", async () => {
		const { dialog } = await openDelete(
			[
				token("a", "Work"),
				token("b", "Work"),
				token("c", "Work", "expired"),
				token("d", "Work", "revoked"),
				token("e", "Other"),
			],
			"Work",
		);
		expect(
			await within(dialog).findByText(
				"2 connected apps use this folder. They will be revoked and stop working.",
			),
		).toBeVisible();
	});

	it("counts tokens in subfolders only when the contents are deleted", async () => {
		const { user, dialog } = await openDelete([token("a", "Tools")], "Work");
		await user.click(
			within(dialog).getByRole("radio", {
				name: "Delete the folder and everything in it",
			}),
		);
		expect(
			await within(dialog).findByText(
				"1 connected app uses this folder. It will be revoked and stop working.",
			),
		).toBeVisible();

		await user.click(
			within(dialog).getByRole("radio", {
				name: "Keep the contents and move them to “Favourites”",
			}),
		);
		await waitFor(() => expect(within(dialog).queryByText(warning)).toBeNull());
	});

	it("shows no warning when no token is affected", async () => {
		const { dialog, server } = await openDelete(
			[token("a", "Work", "revoked"), token("b", "Tools")],
			"Other",
		);
		// Wait until the tokens have been loaded before checking the absence.
		await waitFor(() =>
			expect(server.api.callsTo("GET", "/api/app-tokens")).not.toHaveLength(0),
		);
		await waitFor(() => expect(dialog).toHaveTextContent("Delete folder?"));
		expect(within(dialog).queryByText(warning)).toBeNull();
	});
});

// root
// ├─ Work
// │  ├─ Tools
// │  │  └─ Grafana (monitoring)
// │  └─ Mail (reading)
// └─ Long read (reading, later)
function searchTree() {
	return fakeServer([
		folder(
			"Work",
			folder(
				"Tools",
				link("Grafana", {
					url: "https://grafana.example.com",
					tags: ["monitoring"],
				}),
			),
			link("Mail", { tags: ["reading"] }),
		),
		link("Long read", { tags: ["reading", "later"] }),
	]);
}

const searchBox = () =>
	screen.findByRole("searchbox", { name: "Search favourites" });

describe("search and filter", () => {
	it("shows matching links with their folders", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await user.type(await searchBox(), "graf");
		expect(rows()).toEqual(["Work", "Tools", "Grafana"]);
		expect(screen.getByText("Graf", { selector: "mark" })).toBeVisible();
	});

	it("matches URLs, tags and folder names on the path", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		const box = await searchBox();
		await user.type(box, "grafana.example");
		expect(rows()).toEqual(["Work", "Tools", "Grafana"]);
		await user.clear(box);
		await user.type(box, "work read");
		expect(rows()).toEqual(["Work", "Mail"]);
	});

	it("keeps the filter in the URL across a reload", async () => {
		searchTree();
		const first = renderApp("/favourites");
		await first.user.type(await searchBox(), "mail");
		await waitFor(() =>
			expect(first.router.state.location.search).toEqual({ q: "mail" }),
		);
		first.unmount();
		renderApp("/favourites?q=mail");
		expect(await searchBox()).toHaveValue("mail");
		await waitFor(() => expect(rows()).toEqual(["Work", "Mail"]));
	});

	it("filters by every chosen tag", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await screen.findByRole("link", { name: "Mail" });
		await user.click(screen.getByRole("button", { name: "Tags" }));
		const reading = await screen.findByRole("menuitemcheckbox", {
			name: /reading/,
		});
		expect(reading).toHaveTextContent("2");
		await user.click(reading);
		await waitFor(() => expect(rows()).toEqual(["Work", "Mail", "Long read"]));
		await user.click(screen.getByRole("menuitemcheckbox", { name: /later/ }));
		await waitFor(() => expect(rows()).toEqual(["Long read"]));
	});

	it("adds a row's tag to the filter and removes it again", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Filter by tag monitoring" }),
		);
		await waitFor(() => expect(rows()).toEqual(["Work", "Tools", "Grafana"]));
		await user.click(
			screen.getByRole("button", { name: "Remove tag filter monitoring" }),
		);
		await waitFor(() => expect(rows()).toHaveLength(5));
	});

	it("shows when nothing matches and clears the filters", async () => {
		searchTree();
		const { user } = renderApp("/favourites?tags=%5B%22later%22%5D");
		await user.type(await searchBox(), "nothing");
		expect(
			screen.getByRole("heading", { name: "No favourites match" }),
		).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Clear filters" }));
		expect(await searchBox()).toHaveValue("");
		await waitFor(() => expect(rows()).toHaveLength(5));
	});

	it("can't move nodes while filtering", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		expect(
			await screen.findByRole("button", { name: "Move Mail" }),
		).toBeVisible();
		await user.type(await searchBox(), "mail");
		expect(screen.queryByRole("button", { name: /^Move / })).toBeNull();
		expect(screen.getByText("Clear the filters to reorder.")).toBeVisible();
	});

	it("keeps the filter after editing a shown link", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await user.type(await searchBox(), "mail");
		const dialog = await openEdit(user, "Mail");
		const title = within(dialog).getByLabelText("Title (optional)");
		await user.clear(title);
		await user.type(title, "Mailbox");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(await screen.findByRole("link", { name: "Mailbox" })).toBeVisible();
		expect(await searchBox()).toHaveValue("mail");
		expect(rows()).toEqual(["Work", "Mailbox"]);
	});

	it("focuses the search field with /", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		const box = await searchBox();
		await user.keyboard("/");
		expect(box).toHaveFocus();
		expect(box).toHaveValue("");
	});

	it("shows how many links a folder contains, also while filtering", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		const work = await screen.findByTestId("row-Work");
		expect(within(work).getByText("2 links")).toBeInTheDocument();
		expect(
			within(screen.getByTestId("row-Tools")).getByText("1 link"),
		).toBeInTheDocument();
		await user.type(await searchBox(), "mail");
		expect(
			within(screen.getByTestId("row-Work")).getByText("1 link"),
		).toBeInTheDocument();
	});
});

describe("expand and collapse", () => {
	it("collapses and expands all folders", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Collapse all" }),
		);
		expect(rows()).toEqual(["Work", "Long read"]);
		await user.click(screen.getByRole("button", { name: "Expand all" }));
		expect(rows()).toHaveLength(5);
		expect(screen.getByRole("button", { name: "Collapse all" })).toBeVisible();
	});

	it("remembers collapsed folders", async () => {
		searchTree();
		const first = renderApp("/favourites");
		await first.user.click(
			await screen.findByRole("button", { name: "Collapse Work" }),
		);
		first.unmount();
		renderApp("/favourites");
		await screen.findByRole("button", { name: "Expand Work" });
		expect(rows()).toEqual(["Work", "Long read"]);
	});

	it("doesn't change the remembered state while filtering", async () => {
		searchTree();
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Collapse Work" }),
		);
		await user.type(await searchBox(), "mail");
		expect(rows()).toEqual(["Work", "Mail"]);
		await user.click(screen.getByRole("button", { name: "Collapse all" }));
		await user.click(screen.getByRole("button", { name: "Clear filters" }));
		await waitFor(() => expect(rows()).toEqual(["Work", "Long read"]));
		await user.click(screen.getByRole("button", { name: "Expand Work" }));
		expect(
			screen.getByRole("button", { name: "Collapse Tools" }),
		).toBeVisible();
	});

	it("opens a folder from the URL", async () => {
		searchTree();
		const first = renderApp("/favourites");
		await first.user.click(
			await screen.findByRole("button", { name: "Collapse all" }),
		);
		first.unmount();
		const { router } = renderApp("/favourites?folder=Tools");
		const tools = await screen.findByTestId("row-Tools");
		expect(tools).toHaveAttribute("data-highlighted", "true");
		expect(rows()).toEqual(["Work", "Tools", "Grafana", "Mail", "Long read"]);
		await waitFor(() => expect(router.state.location.search).toEqual({}));
	});

	it("ignores an unknown folder or a link in the URL", async () => {
		searchTree();
		const { router } = renderApp("/favourites?folder=Mail");
		await screen.findByRole("link", { name: "Mail" });
		expect(rows()).toHaveLength(5);
		expect(document.querySelector("[data-highlighted]")).toBeNull();
		await waitFor(() => expect(router.state.location.search).toEqual({}));
	});
});

// root
// ├─ Work
// │  ├─ Tools
// │  │  ├─ Git (dev)
// │  │  └─ Wiki (dev, docs)
// │  └─ Mail
// ├─ Empty
// └─ News
function bulkTree() {
	return fakeServer([
		folder(
			"Work",
			folder(
				"Tools",
				link("Git", { tags: ["dev"] }),
				link("Wiki", { tags: ["dev", "docs"] }),
			),
			link("Mail"),
		),
		folder("Empty"),
		link("News"),
	]);
}

const checkbox = (name: string) =>
	screen.getByRole("checkbox", { name: `Select ${name}` });

async function startSelecting() {
	const app = renderApp("/favourites");
	await app.user.click(await screen.findByRole("button", { name: "Select" }));
	return app;
}

describe("selection mode", () => {
	it("shows checkboxes only in selection mode, in place of the handles", async () => {
		bulkTree();
		const { user } = renderApp("/favourites");
		await screen.findByRole("link", { name: "Mail" });
		expect(screen.queryByRole("checkbox")).toBeNull();
		await user.click(screen.getByRole("button", { name: "Select" }));
		expect(screen.getAllByRole("checkbox")).toHaveLength(7);
		expect(screen.queryByRole("button", { name: /^Move / })).toBeNull();
	});

	it("checks a folder with everything below it", async () => {
		bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Work"));
		for (const name of ["Work", "Tools", "Git", "Wiki", "Mail"]) {
			expect(checkbox(name)).toBeChecked();
		}
		expect(checkbox("News")).not.toBeChecked();
		await user.click(checkbox("Mail"));
		expect(checkbox("Work")).toBePartiallyChecked();
		expect(checkbox("Tools")).toBeChecked();
		await user.click(checkbox("Work"));
		expect(checkbox("Git")).toBeChecked();
		expect(checkbox("Mail")).toBeChecked();
		await user.click(checkbox("Work"));
		expect(checkbox("Git")).not.toBeChecked();
	});

	it("checks a range with shift-click and everything shown with Select all", async () => {
		bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Git"));
		await user.keyboard("{Shift>}");
		await user.click(checkbox("Empty"));
		await user.keyboard("{/Shift}");
		for (const name of ["Git", "Wiki", "Mail", "Empty"]) {
			expect(checkbox(name)).toBeChecked();
		}
		expect(checkbox("News")).not.toBeChecked();
		await user.click(screen.getByRole("button", { name: "Select all" }));
		expect(checkbox("News")).toBeChecked();
	});

	it("checks only the shown nodes of a folder while filtering", async () => {
		bulkTree();
		const { user } = await startSelecting();
		await user.type(await searchBox(), "git");
		await user.click(checkbox("Work"));
		await user.click(screen.getByRole("button", { name: "Clear filters" }));
		await waitFor(() => expect(checkbox("Wiki")).not.toBeChecked());
		expect(checkbox("Git")).toBeChecked();
		// The selection survived the filter change; Work and Tools are only partly checked.
		expect(checkbox("Work")).toBePartiallyChecked();
		expect(screen.getByText("1 link selected")).toBeVisible();
	});

	it("ends with Escape and clears the selection", async () => {
		bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("News"));
		await user.keyboard("{Escape}");
		expect(screen.queryByRole("checkbox")).toBeNull();
		await user.click(screen.getByRole("button", { name: "Select" }));
		expect(checkbox("News")).not.toBeChecked();
	});
});

describe("selection bar", () => {
	it("counts links and folders, including hidden ones", async () => {
		bulkTree();
		const { user } = await startSelecting();
		expect(screen.queryByText(/selected$/)).toBeNull();
		await user.click(checkbox("Tools"));
		await user.click(checkbox("News"));
		expect(screen.getByText("3 links and 1 folder selected")).toBeVisible();
		await user.type(await searchBox(), "news");
		expect(screen.getByText("3 links and 1 folder selected")).toBeVisible();
	});

	it("only offers moving for empty folders", async () => {
		bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Empty"));
		expect(screen.getByText("1 folder selected")).toBeVisible();
		expect(screen.getByRole("button", { name: "Move to…" })).toBeEnabled();
		for (const name of [
			"Add tags",
			"Remove tags",
			"Set tags",
			"Delete 0 links",
		]) {
			expect(screen.getByRole("button", { name })).toBeDisabled();
		}
	});
});

describe("bulk actions", () => {
	it("adds tags to the selected links and keeps the selection", async () => {
		const server = bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Tools"));
		await user.click(checkbox("Empty"));
		await user.click(screen.getByRole("button", { name: "Add tags" }));
		const dialog = await screen.findByRole("dialog", { name: "Add tags" });
		expect(dialog).toHaveTextContent("Applies to 2 links.");
		await user.type(within(dialog).getByRole("combobox"), "later{Enter}");
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("POST", "/api/favourites/bulk")[0].body).toEqual({
			action: "addTags",
			ids: ["Git", "Wiki"],
			tags: ["later"],
		});
		expect(checkbox("Git")).toBeChecked();
		expect(findNode(server.tree(), "Wiki")?.tags).toEqual([
			"dev",
			"docs",
			"later",
		]);
	});

	it("removes chosen tags, listing how many links have each", async () => {
		const server = bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Tools"));
		await user.click(screen.getByRole("button", { name: "Remove tags" }));
		const dialog = await screen.findByRole("dialog", { name: "Remove tags" });
		const dev = within(dialog).getByRole("checkbox", { name: /dev/ });
		expect(dev.closest("label")).toHaveTextContent("dev2");
		await user.click(dev);
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(findNode(server.tree(), "Wiki")?.tags).toEqual(["docs"]);
	});

	it("warns that setting tags replaces them", async () => {
		const server = bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Tools"));
		await user.click(screen.getByRole("button", { name: "Set tags" }));
		const dialog = await screen.findByRole("dialog", { name: "Set tags" });
		expect(dialog).toHaveTextContent(
			"current tags of these links are replaced",
		);
		await user.type(within(dialog).getByRole("combobox"), "archive{Enter}");
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(findNode(server.tree(), "Git")?.tags).toEqual(["archive"]);
		expect(findNode(server.tree(), "Wiki")?.tags).toEqual(["archive"]);
	});

	it("moves full folders as a whole and can't move into them", async () => {
		const server = bulkTree();
		const { user } = await startSelecting();
		await user.click(checkbox("Tools"));
		await user.click(checkbox("News"));
		await user.click(screen.getByRole("button", { name: "Move to…" }));
		const dialog = await screen.findByRole("dialog", { name: "Move to…" });
		expect(dialog).toHaveTextContent("Moves 2 items");
		await user.click(
			within(dialog).getByRole("button", { name: "Expand Work" }),
		);
		expect(within(dialog).getByRole("radio", { name: "Tools" })).toBeDisabled();
		await user.click(within(dialog).getByRole("radio", { name: "Empty" }));
		await user.click(within(dialog).getByRole("button", { name: "Move" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("POST", "/api/favourites/bulk")[0].body).toEqual({
			action: "move",
			ids: ["Tools", "News"],
			parentId: "Empty",
		});
		await waitFor(() =>
			expect(server.titles("Empty")).toEqual(["Tools", "News"]),
		);
		expect(await screen.findByTestId("row-Tools")).toBeVisible();
	});
});

describe("bulk delete", () => {
	async function deleteGitAndMail() {
		const server = bulkTree();
		const app = await startSelecting();
		await app.user.click(checkbox("Git"));
		await app.user.click(checkbox("Mail"));
		await app.user.click(
			screen.getByRole("button", { name: "Delete 2 links" }),
		);
		return { server, ...app };
	}

	it("hides the links and can undo before anything is sent", async () => {
		const { server, user } = await deleteGitAndMail();
		expect(screen.getByText("2 links deleted")).toBeVisible();
		expect(screen.queryByRole("link", { name: "Git" })).toBeNull();
		expect(screen.queryByRole("link", { name: "Mail" })).toBeNull();
		await user.click(screen.getByRole("button", { name: "Undo" }));
		expect(screen.getByRole("link", { name: "Git" })).toBeVisible();
		expect(screen.getByRole("link", { name: "Mail" })).toBeVisible();
		expect(server.api.callsTo("POST", "/api/favourites/bulk")).toHaveLength(0);
	});

	it("sends the delete when the message closes and keeps the folders", async () => {
		const { server, user } = await deleteGitAndMail();
		await user.click(screen.getByRole("button", { name: "Close" }));
		await waitFor(() =>
			expect(server.api.callsTo("POST", "/api/favourites/bulk")).toHaveLength(
				1,
			),
		);
		expect(server.api.callsTo("POST", "/api/favourites/bulk")[0].body).toEqual({
			action: "delete",
			ids: ["Git", "Mail"],
		});
		await waitFor(() => expect(server.titles("Tools")).toEqual(["Wiki"]));
		expect(screen.queryByRole("link", { name: "Git" })).toBeNull();
		expect(screen.getByTestId("row-Work")).toBeVisible();
	});

	it("sends the delete when the page is left", async () => {
		const { server, router } = await deleteGitAndMail();
		await router.navigate({ to: "/" });
		await waitFor(() =>
			expect(server.api.callsTo("POST", "/api/favourites/bulk")).toHaveLength(
				1,
			),
		);
	});

	it("brings the links back when the delete fails", async () => {
		const { server, user } = await deleteGitAndMail();
		server.api.set({
			"POST /api/favourites/bulk": {
				status: 500,
				body: { error: { code: "x" } },
			},
		});
		await user.click(screen.getByRole("button", { name: "Close" }));
		expect(
			await screen.findByText(
				"The links couldn't be deleted. Please try again.",
			),
		).toBeVisible();
		expect(screen.getByRole("link", { name: "Git" })).toBeVisible();
	});
});

// root
// ├─ A (Reading, old)
// ├─ B (Reading, reading-list)
// └─ C (reading-list)
function tagServer() {
	const server = fakeServer([
		link("A", { tags: ["Reading", "old"] }),
		link("B", { tags: ["Reading", "reading-list"] }),
		link("C", { tags: ["reading-list"] }),
	]);
	const links = () => server.tree().children ?? [];
	const names = () => [...new Set(links().flatMap((l) => l.tags ?? []))];
	const usage = () =>
		names()
			.sort((a, b) => a.toLowerCase().localeCompare(b.toLowerCase()))
			.map((name) => ({
				id: `t-${name}`,
				name,
				linkCount: links().filter((l) => l.tags?.includes(name)).length,
			}));
	const replace = (from: string, to: string | null) => {
		for (const l of links()) {
			if (!l.tags?.includes(from)) continue;
			const rest = l.tags.filter((t) => t !== from);
			l.tags = to && !rest.includes(to) ? [...rest, to] : rest;
		}
	};
	const routes: Parameters<typeof server.api.set>[0] = {
		"GET /api/tags": ({ url }) =>
			url.searchParams.get("all") === "true" ? { body: usage() } : { body: [] },
	};
	for (const name of ["Reading", "old", "reading-list"]) {
		routes[`PATCH /api/tags/t-${name}`] = ({ body }) => {
			const b = body as { name: string; merge?: boolean };
			const other = names().find(
				(n) => n !== name && n.toLowerCase() === b.name.toLowerCase(),
			);
			if (other && !b.merge) {
				return { status: 409, body: { error: { code: "tag_exists" } } };
			}
			replace(name, other ?? b.name);
			const to = other ?? b.name;
			return { body: { id: `t-${to}`, name: to } };
		};
		routes[`DELETE /api/tags/t-${name}`] = () => {
			replace(name, null);
			return { status: 204 };
		};
	}
	server.api.set(routes);
	return server;
}

async function openManageTags(path = "/favourites") {
	const app = renderApp(path);
	await app.user.click(
		await screen.findByRole("button", { name: "Manage tags" }),
	);
	const dialog = await screen.findByRole("dialog", { name: "Manage tags" });
	await within(dialog).findByText("old");
	return { ...app, dialog };
}

describe("manage tags", () => {
	it("lists the tags with their link counts", async () => {
		tagServer();
		const { dialog } = await openManageTags();
		const items = within(dialog)
			.getAllByRole("listitem")
			.map((li) => li.textContent);
		expect(items).toEqual(["old1", "Reading2", "reading-list2"]);
	});

	it("says when there are no tags", async () => {
		const server = fakeServer([link("A")]);
		server.api.set({ "GET /api/tags": { body: [] } });
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Manage tags" }),
		);
		expect(await screen.findByText("No tags yet")).toBeVisible();
	});

	it("renames a tag on all its links", async () => {
		const server = tagServer();
		const { user, dialog } = await openManageTags();
		await user.click(
			within(dialog).getByRole("button", { name: "Rename old" }),
		);
		const input = within(dialog).getByRole("textbox", {
			name: "New name for old",
		});
		await user.clear(input);
		await user.type(input, "archive{Enter}");
		expect(await within(dialog).findByText("archive")).toBeVisible();
		expect(server.api.callsTo("PATCH", "/api/tags/t-old")).toHaveLength(1);
		expect(findNode(server.tree(), "A")?.tags).toEqual(["Reading", "archive"]);
	});

	it("changes only the case without asking to merge", async () => {
		tagServer();
		const { user, dialog } = await openManageTags();
		await user.click(
			within(dialog).getByRole("button", { name: "Rename old" }),
		);
		const input = within(dialog).getByRole("textbox", {
			name: "New name for old",
		});
		await user.clear(input);
		await user.type(input, "Old{Enter}");
		expect(await within(dialog).findByText("Old")).toBeVisible();
		expect(within(dialog).queryByText(/Merge/)).toBeNull();
	});

	it("asks before merging into an existing tag, and can cancel", async () => {
		const server = tagServer();
		const { user, dialog } = await openManageTags();
		await user.click(
			within(dialog).getByRole("button", { name: "Rename reading-list" }),
		);
		const input = within(dialog).getByRole("textbox", {
			name: "New name for reading-list",
		});
		await user.clear(input);
		await user.type(input, "reading{Enter}");
		expect(
			await within(dialog).findByText(
				"“Reading” already exists. Merge “reading-list” into “Reading”?",
			),
		).toBeVisible();
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		expect(within(dialog).getByText("reading-list")).toBeVisible();
		expect(findNode(server.tree(), "C")?.tags).toEqual(["reading-list"]);

		await user.click(
			within(dialog).getByRole("button", { name: "Rename reading-list" }),
		);
		const again = within(dialog).getByRole("textbox", {
			name: "New name for reading-list",
		});
		await user.clear(again);
		await user.type(again, "reading{Enter}");
		await user.click(
			await within(dialog).findByRole("button", { name: "Merge" }),
		);
		await waitFor(() =>
			expect(within(dialog).queryByText("reading-list")).toBeNull(),
		);
		expect(findNode(server.tree(), "B")?.tags).toEqual(["Reading"]);
		expect(findNode(server.tree(), "C")?.tags).toEqual(["Reading"]);
		expect(
			server.api.callsTo("PATCH", "/api/tags/t-reading-list").at(-1)?.body,
		).toEqual({ name: "reading", merge: true });
	});

	it("deletes a tag after confirming, keeping the links", async () => {
		const server = tagServer();
		const { user, dialog } = await openManageTags();
		await user.click(
			within(dialog).getByRole("button", { name: "Delete Reading" }),
		);
		expect(
			within(dialog).getByText("Remove “Reading” from 2 links?"),
		).toBeVisible();
		await user.click(within(dialog).getByRole("button", { name: "Remove" }));
		await waitFor(() =>
			expect(within(dialog).queryByText("Reading")).toBeNull(),
		);
		expect(findNode(server.tree(), "A")?.tags).toEqual(["old"]);
		expect(server.titles()).toEqual(["A", "B", "C"]);
	});

	it("filters by a tag from its count", async () => {
		tagServer();
		const { user, dialog, router } = await openManageTags();
		await user.click(
			within(dialog).getByRole("button", {
				name: "Show the 2 links with reading-list",
			}),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(router.state.location.search).toEqual({ tags: ["reading-list"] });
		await waitFor(() => expect(rows()).toEqual(["B", "C"]));
	});

	it("keeps the tag filter in sync after a rename or delete", async () => {
		tagServer();
		const { user, dialog, router } = await openManageTags(
			"/favourites?tags=%5B%22old%22%2C%22Reading%22%5D",
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Rename old" }),
		);
		const input = within(dialog).getByRole("textbox", {
			name: "New name for old",
		});
		await user.clear(input);
		await user.type(input, "archive{Enter}");
		await waitFor(() =>
			expect(router.state.location.search).toEqual({
				tags: ["Reading", "archive"],
			}),
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Delete Reading" }),
		);
		await user.click(within(dialog).getByRole("button", { name: "Remove" }));
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ tags: ["archive"] }),
		);
	});
});

describe("deep nesting", () => {
	it("indents with the shared variable and scrolls inside the frame", async () => {
		fakeServer([folder("A", folder("B", link("Deep")))]);
		renderApp("/favourites");
		const deep = await screen.findByTestId("row-Deep");
		expect(deep.style.paddingLeft).toBe("calc(var(--indent) * 2 + 0.25rem)");
		// Half the indent below sm.
		const list = deep.parentElement as HTMLElement;
		expect(list).toHaveClass(
			"w-max",
			"min-w-full",
			"[--indent:0.75rem]",
			"sm:[--indent:1.5rem]",
		);
		expect(list.parentElement).toHaveClass("@container", "overflow-x-auto");
		// At least as wide as the frame, wider when needed; names keep a minimum.
		expect(deep).toHaveClass("w-min", "min-w-[calc(100cqw-0.5rem)]");
		const name = screen.getByRole("link", { name: "Deep" }).parentElement;
		expect(name).toHaveClass("min-w-24", "sm:min-w-32");
	});

	it("caps long names with the full name as a tooltip", async () => {
		const long = "A very long title ".repeat(20).trim();
		fakeServer([folder("Work", link("L", { title: long }))]);
		renderApp("/favourites");
		const title = await screen.findByRole("link", { name: long });
		expect(title).toHaveClass("max-w-sm", "truncate");
		expect(title).toHaveAttribute("title", long);
		const work = within(screen.getByTestId("row-Work")).getByText("Work");
		expect(work.closest("[title]")).toHaveClass("max-w-sm", "truncate");
		expect(work.closest("[title]")).toHaveAttribute("title", "Work");
	});
});
