import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { $getSelection, $isRangeSelection } from "lexical";
import { describe, expect, it } from "vitest";
import type { NoteDoc } from "#/features/notes/document/types";
import type { AppToken } from "#/features/settings/appTokens/api";
import { apiError, mockApi } from "#/test/api";
import { editorOf, selectAll, typeInEditor } from "#/test/lexical";
import { category, fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";

/** The accessible names of the cards' links, in grid order. */
function cardNames() {
	return screen
		.queryAllByRole("link", { name: /^Open / })
		.map((l) => l.getAttribute("aria-label")?.replace(/^Open /, ""));
}

const panel = () => screen.getByRole("complementary", { name: "Categories" });

describe("notes page", () => {
	it("sends signed-out visitors to /login", async () => {
		mockApi();
		const { router } = renderApp("/notes");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("shows the empty state with a New note button", async () => {
		fakeNotesServer();
		renderApp("/notes");
		expect(await screen.findByText("No notes yet")).toBeInTheDocument();
		expect(screen.getAllByRole("link", { name: "New note" })).toHaveLength(2);
		await waitFor(() => expect(document.title).toBe("Notes - Boardly"));
	});

	it("lists notes as cards, last edited first, with untitled ones by their first line", async () => {
		fakeNotesServer([
			note("Groceries"),
			note("Sprint plan"),
			note("u", { title: null, content: paragraphs("call the plumber") }),
			note("e", { title: null, content: { v: 1, blocks: [] } }),
		]);
		renderApp("/notes");
		await screen.findByRole("link", { name: "Open Groceries" });
		await waitFor(() =>
			expect(cardNames()).toEqual([
				"Untitled note",
				"call the plumber",
				"Sprint plan",
				"Groceries",
			]),
		);
	});

	it("shows icon, title, content, colour and category on a card", async () => {
		fakeNotesServer(
			[
				note("Groceries", {
					color: "green",
					categoryId: "home",
					icon: { type: "symbol", name: "shopping-cart" },
					content: {
						v: 1,
						blocks: [
							{
								type: "list",
								style: "task",
								items: [
									{
										id: "t1",
										checked: true,
										content: [{ type: "text", text: "milk" }],
									},
								],
							},
						],
					},
				}),
			],
			[category("home", "Home")],
		);
		renderApp("/notes");
		const link = await screen.findByRole("link", { name: "Open Groceries" });
		const card = link.closest("article") as HTMLElement;
		expect(card.style.background).toBe("var(--color-note-green)");
		expect(
			within(card).getByRole("heading", { name: "Groceries" }),
		).toBeInTheDocument();
		expect(card.querySelector('[data-icon="shopping-cart"]')).not.toBeNull();
		expect(within(card).getByRole("checkbox", { name: "milk" })).toBeChecked();
		expect(
			within(card).getByRole("button", {
				name: "Category: Home. Change category",
			}),
		).toBeInTheDocument();
	});

	it("opens a note from its card", async () => {
		fakeNotesServer([note("Groceries")]);
		const { user, router } = renderApp("/notes");
		await user.click(
			await screen.findByRole("link", { name: "Open Groceries" }),
		);
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/notes/Groceries"),
		);
	});
});

describe("category side panel", () => {
	const data = () =>
		fakeNotesServer(
			[
				note("Lamp", { categoryId: "home" }),
				note("Seeds", { categoryId: "garden" }),
				note("Roses", { categoryId: "garden" }),
				note("Sprint", { categoryId: "work" }),
				note("Loose"),
			],
			[
				category("home", "Home"),
				category("garden", "Garden", "home"),
				category("work", "Work"),
				category("empty", "Empty"),
			],
		);

	it("filters by category including subcategories, with counts", async () => {
		data();
		const { user, router } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		const home = within(panel()).getByRole("button", { name: /^Home/ });
		expect(home).toHaveAccessibleName("Home, 3 notes");
		await user.click(home);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ category: "home" }),
		);
		await waitFor(() =>
			expect(cardNames()).toEqual(["Roses", "Seeds", "Lamp"]),
		);
		expect(home).toHaveAttribute("aria-current", "true");
	});

	it("shows uncategorized notes and an empty category", async () => {
		data();
		const { user, router } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		await user.click(
			within(panel()).getByRole("button", { name: /^Uncategorized/ }),
		);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ category: "none" }),
		);
		await waitFor(() => expect(cardNames()).toEqual(["Loose"]));
		await user.click(within(panel()).getByRole("button", { name: /^Empty/ }));
		expect(
			await screen.findByText("No notes in this category"),
		).toBeInTheDocument();
	});

	it("shows all notes for an unknown category in the address", async () => {
		data();
		renderApp("/notes?category=doesnotexist");
		await screen.findByRole("link", { name: "Open Lamp" });
		await waitFor(() => expect(cardNames()).toHaveLength(5));
		expect(
			within(panel()).getByRole("button", { name: /^All notes/ }),
		).toHaveAttribute("aria-current", "true");
	});

	it("collapses and expands subcategories", async () => {
		data();
		const { user } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		await user.click(
			within(panel()).getByRole("button", { name: "Collapse Home" }),
		);
		expect(
			within(panel()).queryByRole("button", { name: /^Garden/ }),
		).toBeNull();
		await user.click(
			within(panel()).getByRole("button", { name: "Expand Home" }),
		);
		expect(
			within(panel()).getByRole("button", { name: /^Garden/ }),
		).toBeInTheDocument();
	});

	it("chooses a category in the drawer and closes it", async () => {
		data();
		const { user } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		await user.click(screen.getByRole("button", { name: "Categories" }));
		const drawer = await screen.findByRole("dialog", { name: "Categories" });
		await user.click(within(drawer).getByRole("button", { name: /^Work/ }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		await waitFor(() => expect(cardNames()).toEqual(["Sprint"]));
	});

	it("starts a new note in the chosen category", async () => {
		data();
		const { user, router } = renderApp("/notes?category=work");
		await screen.findByRole("link", { name: "Open Sprint" });
		await user.click(screen.getByRole("link", { name: "New note" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/notes/new"),
		);
		expect(
			await screen.findByRole("button", {
				name: "Category: Work. Change category",
			}),
		).toBeInTheDocument();
	});
});

describe("pinned and archived notes", () => {
	const data = () =>
		fakeNotesServer(
			[
				note("Groceries", { categoryId: "home", pinned: true }),
				note("Old plan", { categoryId: "home", archived: true }),
				note("Lamp", { categoryId: "home" }),
				note("Sprint", { categoryId: "work" }),
				note("Loose", { archived: true }),
			],
			[category("home", "Home"), category("work", "Work")],
		);

	it("shows pinned notes first with a pin, within the chosen category", async () => {
		data();
		const { user } = renderApp("/notes");
		const open = await screen.findByRole("link", { name: "Open Groceries" });
		await waitFor(() =>
			expect(cardNames()).toEqual(["Groceries", "Sprint", "Lamp"]),
		);
		const card = open.closest("article") as HTMLElement;
		expect(within(card).getByRole("img", { name: "Pinned" })).toBeVisible();
		const lamp = screen
			.getByRole("link", { name: "Open Lamp" })
			.closest("article") as HTMLElement;
		expect(within(lamp).queryByRole("img", { name: "Pinned" })).toBeNull();
		await user.click(within(panel()).getByRole("button", { name: /^Work/ }));
		await waitFor(() => expect(cardNames()).toEqual(["Sprint"]));
	});

	it("leaves archived notes out and shows them under Archived", async () => {
		data();
		const { user, router } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		expect(cardNames()).not.toContain("Old plan");
		expect(
			within(panel()).getByRole("button", { name: /^Home/ }),
		).toHaveAccessibleName("Home, 2 notes");
		expect(
			within(panel()).getByRole("button", { name: /^Uncategorized/ }),
		).toHaveAccessibleName("Uncategorized, 0 notes");
		const archived = within(panel()).getByRole("button", {
			name: /^Archived/,
		});
		expect(archived).toHaveAccessibleName("Archived, 2 notes");
		await user.click(archived);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ category: "archived" }),
		);
		await waitFor(() => expect(cardNames()).toEqual(["Loose", "Old plan"]));
		const card = screen
			.getByRole("link", { name: "Open Old plan" })
			.closest("article") as HTMLElement;
		expect(card).toHaveClass("opacity-70");
		expect(within(card).getByText("Archived")).toBeVisible();
	});

	it("says when every note is archived, and shows it under Archived", async () => {
		fakeNotesServer([note("Old plan", { archived: true })]);
		const { user } = renderApp("/notes");
		expect(
			await screen.findByText("All notes are archived"),
		).toBeInTheDocument();
		await user.click(
			within(panel()).getByRole("button", { name: /^Archived/ }),
		);
		await waitFor(() => expect(cardNames()).toEqual(["Old plan"]));
	});

	it("says when no note is archived", async () => {
		fakeNotesServer([note("Lamp")]);
		renderApp("/notes?category=archived");
		expect(await screen.findByText("No archived notes")).toBeInTheDocument();
	});

	it("starts a new note from the archive without a category", async () => {
		data();
		const { user, router } = renderApp("/notes?category=archived");
		await screen.findByRole("link", { name: "Open Old plan" });
		await user.click(screen.getByRole("link", { name: "New note" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/notes/new"),
		);
		expect(router.state.location.search).toEqual({});
		expect(
			await screen.findByRole("button", { name: "Add category" }),
		).toBeInTheDocument();
	});
});

describe("managing categories", () => {
	it("creates a category with title, colour and parent", async () => {
		const { categories } = fakeNotesServer([], [category("home", "Home")]);
		const { user } = renderApp("/notes");
		await screen.findByText("No notes yet");
		await user.click(
			within(panel()).getByRole("button", { name: "New category" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New category" });
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(within(dialog).getByRole("alert")).toHaveTextContent(
			"Enter a title.",
		);
		await user.type(
			within(dialog).getByRole("textbox", { name: "Title" }),
			"Garden",
		);
		await user.click(within(dialog).getByRole("radio", { name: "Green" }));
		await user.selectOptions(
			within(dialog).getByRole("combobox", { name: "Parent category" }),
			"home",
		);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(categories.at(-1)).toMatchObject({
			title: "Garden",
			color: "green",
			parentId: "home",
			icon: null,
		});
		expect(
			await within(panel()).findByRole("button", { name: /^Garden/ }),
		).toBeInTheDocument();
	});

	it("renames a category and leaves out itself and its subcategories as parents", async () => {
		fakeNotesServer(
			[note("Lamp", { categoryId: "home" })],
			[
				category("home", "Home"),
				category("garden", "Garden", "home"),
				category("work", "Work"),
			],
		);
		const { user } = renderApp("/notes");
		await screen.findByRole("link", { name: "Open Lamp" });
		await user.click(
			within(panel()).getByRole("button", { name: "Actions for Home" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Edit" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit category" });
		const parent = within(dialog).getByRole("combobox", {
			name: "Parent category",
		});
		expect(
			within(parent)
				.getAllByRole("option")
				.map((o) => o.textContent),
		).toEqual(["None", "Work"]);
		const title = within(dialog).getByRole("textbox", { name: "Title" });
		await user.clear(title);
		await user.type(title, "House");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(
			await screen.findByRole("button", {
				name: "Category: House. Change category",
			}),
		).toBeInTheDocument();
	});

	it("preselects the parent of a new subcategory", async () => {
		fakeNotesServer([], [category("home", "Home")]);
		const { user } = renderApp("/notes");
		await screen.findByText("No notes yet");
		await user.click(
			within(panel()).getByRole("button", { name: "Actions for Home" }),
		);
		await user.click(
			await screen.findByRole("menuitem", { name: "New subcategory" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New category" });
		expect(
			within(dialog).getByRole("combobox", { name: "Parent category" }),
		).toHaveValue("home");
	});

	it("deletes a category after saying what happens to its notes and subcategories", async () => {
		const { notes } = fakeNotesServer(
			[
				note("Lamp", { categoryId: "home" }),
				note("Desk", { categoryId: "home" }),
				note("Seeds", { categoryId: "garden" }),
			],
			[category("home", "Home"), category("garden", "Garden", "home")],
		);
		const { user } = renderApp("/notes?category=home");
		await screen.findByRole("link", { name: "Open Lamp" });
		await user.click(
			within(panel()).getByRole("button", { name: "Actions for Home" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
		const dialog = await screen.findByRole("alertdialog", {
			name: "Delete category?",
		});
		expect(dialog).toHaveTextContent("“Home” will be deleted.");
		expect(dialog).toHaveTextContent("Its 2 notes become uncategorized.");
		expect(dialog).toHaveTextContent("Its subcategory moves up one level.");
		await user.click(within(dialog).getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(
			notes
				.filter((n) => n.categoryId === null)
				.map((n) => n.id)
				.sort(),
		).toEqual(["Desk", "Lamp"]);
		await waitFor(() =>
			expect(
				within(panel()).queryByRole("button", { name: /^Home/ }),
			).toBeNull(),
		);
		expect(
			within(panel()).getByRole("button", { name: /^Garden/ }),
		).toBeInTheDocument();
		expect(
			within(panel()).getByRole("button", { name: /^All notes/ }),
		).toHaveAttribute("aria-current", "true");
	});
});

describe("deleting a category used by connected apps", () => {
	const token = (id: string, fields: Partial<AppToken>): AppToken => ({
		id,
		name: id,
		app: "karakeep",
		prefix: "bdly_ABCDEFG",
		rootId: "root",
		noteScope: "category",
		rootCategoryId: "home",
		createdAt: "2026-02-01T10:00:00Z",
		lastUsedAt: null,
		expiresAt: null,
		revokedAt: null,
		status: "active",
		...fields,
	});

	it("says how many active apps will be revoked", async () => {
		fakeNotesServer([], [category("home", "Home"), category("work", "Work")], {
			"GET /api/app-tokens": {
				body: [
					token("Phone", {}),
					token("Tablet", {}),
					token("Old", { status: "revoked" }),
					token("All", { noteScope: "all", rootCategoryId: null }),
					token("Work", { rootCategoryId: "work" }),
				],
			},
		});
		const { user } = renderApp("/notes");
		await screen.findByText("No notes yet");
		await user.click(
			within(panel()).getByRole("button", { name: "Actions for Home" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
		const dialog = await screen.findByRole("alertdialog", {
			name: "Delete category?",
		});
		expect(
			await within(dialog).findByText(
				"2 connected apps use this category. They will be revoked and stop working.",
			),
		).toBeVisible();
	});
});

describe("category badge", () => {
	it("shows the path as its tooltip and changes the category from a card", async () => {
		const { notes } = fakeNotesServer(
			[note("Seeds", { categoryId: "garden" })],
			[category("home", "Home"), category("garden", "Garden", "home")],
		);
		const { user, router } = renderApp("/notes");
		const badge = await screen.findByRole("button", {
			name: "Category: Home › Garden. Change category",
		});
		expect(badge).toHaveAttribute("title", "Home › Garden");
		await user.click(badge);
		await user.click(
			await screen.findByRole("menuitemradio", { name: "Home" }),
		);
		expect(
			await screen.findByRole("button", {
				name: "Category: Home. Change category",
			}),
		).toBeInTheDocument();
		expect(notes[0]).toMatchObject({ categoryId: "home", version: 2 });
		expect(router.state.location.pathname).toBe("/notes");
	});

	it("removes the category with No category", async () => {
		const { notes } = fakeNotesServer(
			[note("Seeds", { categoryId: "home" })],
			[category("home", "Home")],
		);
		const { user } = renderApp("/notes");
		await user.click(
			await screen.findByRole("button", {
				name: "Category: Home. Change category",
			}),
		);
		await user.click(
			await screen.findByRole("menuitemradio", { name: "No category" }),
		);
		await waitFor(() => expect(notes[0].categoryId).toBeNull());
		await waitFor(() =>
			expect(screen.queryByRole("button", { name: /^Category:/ })).toBeNull(),
		);
	});
});

const tasks = (...items: [string, boolean][]): NoteDoc => ({
	v: 1,
	blocks: [
		{
			type: "list",
			style: "task",
			items: items.map(([text, checked], i) => ({
				id: `t${i + 1}`,
				checked,
				content: [{ type: "text", text }],
			})),
		},
	],
});

// Typing goes through the editor one character at a time, and inline edits
// wait for the autosave delay.
const slow = { timeout: 20_000 };

describe("checkboxes on cards", () => {
	it("toggles a task right away without opening the note", async () => {
		const { notes } = fakeNotesServer([
			note("Groceries", { content: tasks(["milk", false], ["eggs", false]) }),
		]);
		const { user, router } = renderApp("/notes");
		const milk = await screen.findByRole("checkbox", { name: "milk" });
		await user.click(milk);
		await user.click(screen.getByRole("checkbox", { name: "eggs" }));
		expect(milk).toBeChecked();
		await waitFor(() => expect(notes[0].version).toBe(3));
		expect(notes[0].content).toEqual(tasks(["milk", true], ["eggs", true]));
		expect(router.state.location.pathname).toBe("/notes");
		expect(screen.queryByRole("textbox", { name: "Content" })).toBeNull();
	});

	it("shows the saved state again when a toggle fails", async () => {
		const { api } = fakeNotesServer([
			note("Groceries", { content: tasks(["milk", false]) }),
		]);
		const { user } = renderApp("/notes");
		const milk = await screen.findByRole("checkbox", { name: "milk" });
		api.set({ "PATCH /api/notes/Groceries": apiError(500, "error") });
		await user.click(milk);
		expect(
			await screen.findByText(
				"The note couldn't be changed. Please try again.",
			),
		).toBeInTheDocument();
		await waitFor(() =>
			expect(screen.getByRole("checkbox", { name: "milk" })).not.toBeChecked(),
		);
	});
});

describe("inline editing", () => {
	it("edits the content in place without a toolbar", slow, async () => {
		const { notes } = fakeNotesServer([note("Groceries")]);
		const { user, router } = renderApp("/notes");
		await user.click(
			await screen.findByRole("button", { name: "Edit Groceries" }),
		);
		const content = await screen.findByRole("textbox", { name: "Content" });
		expect(screen.queryByRole("button", { name: "Bold" })).toBeNull();
		await waitFor(() => expect(content).toHaveFocus());
		await waitFor(() =>
			editorOf(content)
				.getEditorState()
				.read(() => {
					const selection = $getSelection();
					expect($isRangeSelection(selection) && selection.anchor.offset).toBe(
						"Groceries text".length,
					);
				}),
		);
		await selectAll(content);
		fireEvent.keyDown(content, { key: "b", code: "KeyB", ctrlKey: true });
		await waitFor(() => expect(notes[0].version).toBe(2), { timeout: 3000 });
		expect(notes[0].content).toEqual({
			v: 1,
			blocks: [
				{
					type: "paragraph",
					content: [{ type: "text", text: "Groceries text", marks: ["bold"] }],
				},
			],
		});
		fireEvent.keyDown(content, { key: "Escape" });
		await waitFor(() =>
			expect(screen.queryByRole("textbox", { name: "Content" })).toBeNull(),
		);
		expect(
			screen.getByRole("button", { name: "Edit Groceries" }),
		).toHaveFocus();
		expect(screen.getByText("Groceries text")).toHaveClass("font-semibold");
		expect(router.state.location.pathname).toBe("/notes");
	});

	it("makes tasks with Markdown shortcuts inline", slow, async () => {
		const { notes } = fakeNotesServer([
			note("Groceries", { content: { v: 1, blocks: [] } }),
		]);
		const { user } = renderApp("/notes");
		await user.click(
			await screen.findByRole("button", { name: "Edit Groceries" }),
		);
		await typeInEditor(
			await screen.findByRole("textbox", { name: "Content" }),
			"- [ ] bread",
		);
		await waitFor(() => expect(notes[0].version).toBe(2), { timeout: 3000 });
		expect(notes[0].content.blocks[0]).toMatchObject({
			type: "list",
			style: "task",
			items: [{ checked: false, content: [{ text: "bread" }] }],
		});
	});

	it("edits the title and saves it when editing ends", async () => {
		const { notes } = fakeNotesServer([
			note("u", { title: null, content: paragraphs("bread") }),
		]);
		const { user } = renderApp("/notes");
		await user.click(await screen.findByRole("button", { name: "Edit bread" }));
		await user.type(
			await screen.findByRole("textbox", { name: "Title" }),
			"Shopping",
		);
		await user.keyboard("{Escape}");
		await waitFor(() => expect(notes[0].title).toBe("Shopping"));
		expect(
			await screen.findByRole("heading", { name: "Shopping" }),
		).toBeInTheDocument();
	});

	it("edits one card at a time", async () => {
		fakeNotesServer([note("Groceries"), note("Sprint")]);
		const { user } = renderApp("/notes");
		await user.click(
			await screen.findByRole("button", { name: "Edit Groceries" }),
		);
		await screen.findByRole("textbox", { name: "Content" });
		await user.click(screen.getByRole("button", { name: "Edit Sprint" }));
		await waitFor(() =>
			expect(screen.getAllByRole("textbox", { name: "Content" })).toHaveLength(
				1,
			),
		);
		expect(
			screen.getByRole("button", { name: "Edit Groceries" }),
		).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Edit Sprint" })).toBeNull();
	});

	it("keeps the card's place until /notes is opened again", slow, async () => {
		const { notes } = fakeNotesServer([
			note("Groceries"),
			note("Sprint"),
			note("Ideas"),
		]);
		const { user, router } = renderApp("/notes");
		await waitFor(() =>
			expect(cardNames()).toEqual(["Ideas", "Sprint", "Groceries"]),
		);
		await user.click(screen.getByRole("button", { name: "Edit Groceries" }));
		await typeInEditor(
			await screen.findByRole("textbox", { name: "Content" }),
			"!",
		);
		await user.keyboard("{Escape}");
		await waitFor(() => expect(notes[0].version).toBe(2));
		await waitFor(() => expect(router.state.isLoading).toBe(false));
		expect(cardNames()).toEqual(["Ideas", "Sprint", "Groceries"]);
		await user.click(screen.getByRole("link", { name: "Open Ideas" }));
		await screen.findByRole("textbox", { name: "Content" });
		await user.click(screen.getByRole("link", { name: "Notes" }));
		await waitFor(() =>
			expect(cardNames()).toEqual(["Groceries", "Ideas", "Sprint"]),
		);
	});

	it("ends editing when the note was changed elsewhere", slow, async () => {
		const { notes } = fakeNotesServer([note("Groceries")]);
		const { user } = renderApp("/notes");
		await user.click(
			await screen.findByRole("button", { name: "Edit Groceries" }),
		);
		const content = await screen.findByRole("textbox", { name: "Content" });
		// Another tab saves first.
		notes[0].version = 5;
		notes[0].content = paragraphs("from the other tab");
		await typeInEditor(content, "mine");
		expect(
			await screen.findByText(
				"The note was changed elsewhere. Its card shows the current state.",
				{},
				{ timeout: 3000 },
			),
		).toBeInTheDocument();
		expect(screen.queryByRole("textbox", { name: "Content" })).toBeNull();
		expect(await screen.findByText("from the other tab")).toBeInTheDocument();
	});
});
