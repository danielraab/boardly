import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { mockApi } from "#/test/api";
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
