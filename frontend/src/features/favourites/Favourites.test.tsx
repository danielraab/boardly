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
				Object.assign(target ?? {}, body);
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

describe("refresh icon", () => {
	it("refreshes a link's favicon and shows the new one", async () => {
		const server = fakeServer([folder("Work"), link("GitHub")]);
		server.api.set({
			"POST /api/favicons/refresh": () => {
				const target = findNode(server.tree(), "GitHub");
				if (target) target.iconImageId = "img-new";
				return { body: { id: "img-new" } };
			},
		});
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Refresh icon of GitHub" }),
		);
		expect(server.api.callsTo("POST", "/api/favicons/refresh")[0].body).toEqual(
			{
				url: "https://example.org/GitHub",
			},
		);
		const row = screen.getByRole("link", { name: "GitHub" }).parentElement
			?.parentElement as HTMLElement;
		await waitFor(() =>
			expect(row.querySelector("img")).toHaveAttribute(
				"src",
				"/api/images/img-new",
			),
		);
		// Folders have no icon to refresh.
		expect(
			screen.queryByRole("button", { name: "Refresh icon of Work" }),
		).toBeNull();
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
		const button = await screen.findByRole("button", {
			name: "Refresh icon of GitHub",
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
	])("explains %s and keeps the icon", async (code, status, message) => {
		const server = fakeServer([link("GitHub", { iconImageId: "img-1" })]);
		server.api.set({ "POST /api/favicons/refresh": apiError(status, code) });
		const { user } = renderApp("/favourites");
		await user.click(
			await screen.findByRole("button", { name: "Refresh icon of GitHub" }),
		);
		expect(await screen.findByRole("alert")).toHaveTextContent(message);
		const row = screen.getByRole("link", { name: "GitHub" }).parentElement
			?.parentElement as HTMLElement;
		expect(row.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img-1",
		);
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
			{ title: "Documentation", url: "https://example.org/doc", tags: [] },
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
