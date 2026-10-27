import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { $getRoot, $isTextNode } from "lexical";
import { describe, expect, it, vi } from "vitest";
import { apiError } from "#/test/api";
import { editorOf, selectAll, typeInEditor } from "#/test/lexical";
import { category, fakeNotesServer, note } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { Block, Inline, NoteDoc } from "./document/types";

// Typing goes through the editor one character at a time, and saves wait for
// the autosave delay.
const slow = { timeout: 20_000 };

const empty = (): NoteDoc => ({ v: 1, blocks: [] });

async function openNote(
	fields: Parameters<typeof note>[1] = {},
	categories = [category("work", "Work")],
) {
	const server = fakeNotesServer(
		[note("n1", { content: empty(), ...fields })],
		categories,
	);
	const app = renderApp("/notes/n1");
	const content = await screen.findByRole("textbox", { name: "Content" });
	return { ...server, ...app, content };
}

/** Waits for the next save and returns the note as the server has it. */
async function saved(
	server: { notes: { version: number }[] },
	version: number,
) {
	await waitFor(() => expect(server.notes[0].version).toBe(version), {
		timeout: 3000,
	});
	return server.notes[0] as ReturnType<typeof note>;
}

const blocks = (n: { content: NoteDoc }): Block[] => n.content.blocks;

describe("Markdown shortcuts", () => {
	it(
		"makes tasks from - [ ] and [] and continues them with Enter",
		slow,
		async () => {
			const server = await openNote();
			await typeInEditor(server.content, "- [ ] milk\neggs\n\nthen\n[] bread");
			const n = await saved(server, 2);
			const [first, between, second] = blocks(n);
			expect(between).toMatchObject({
				type: "paragraph",
				content: [{ text: "then" }],
			});
			expect(first).toMatchObject({
				type: "list",
				style: "task",
				items: [
					{ checked: false, content: [{ text: "milk" }] },
					{ checked: false, content: [{ text: "eggs" }] },
				],
			});
			expect(second).toMatchObject({
				type: "list",
				style: "task",
				items: [{ checked: false, content: [{ text: "bread" }] }],
			});
			const ids = [first, second].flatMap((b) =>
				b.type === "list" ? b.items.map((i) => i.id) : [],
			);
			expect(new Set(ids).size).toBe(3);
			for (const id of ids) expect(id).toMatch(/^[A-Za-z0-9_-]{12}$/);
		},
	);

	it("makes a checked task from - [x]", slow, async () => {
		const server = await openNote();
		await typeInEditor(server.content, "- [x] done");
		const n = await saved(server, 2);
		expect(blocks(n)[0]).toMatchObject({
			style: "task",
			items: [{ checked: true, content: [{ text: "done" }] }],
		});
	});

	it(
		"makes headings of three levels, quotes, lists and dividers",
		slow,
		async () => {
			const server = await openNote();
			await typeInEditor(
				server.content,
				"# One\n### Three\n#### Four\n> quoted\n\n1. first\n\n* bullet\n\n---\n",
			);
			const n = await saved(server, 2);
			expect(
				blocks(n).map((b) => [
					b.type,
					"level" in b ? b.level : "style" in b ? b.style : null,
				]),
			).toEqual([
				["heading", 1],
				["heading", 3],
				["paragraph", null],
				["quote", null],
				["paragraph", null],
				["list", "ordered"],
				["list", "bullet"],
				["divider", null],
				["paragraph", null],
			]);
			expect(blocks(n)[2]).toMatchObject({ content: [{ text: "#### Four" }] });
		},
	);

	it("formats text and makes links", slow, async () => {
		const server = await openNote();
		await typeInEditor(
			server.content,
			"**urgent** _soft_ ~~gone~~ `code` [docs](https://example.org) [bad](javascript:alert(1)) ",
		);
		const n = await saved(server, 2);
		const content = (blocks(n)[0] as { content: Inline[] }).content;
		expect(
			content.flatMap((i) =>
				i.type === "text" && i.marks ? [[i.text.trim(), i.marks]] : [],
			),
		).toEqual([
			["urgent", ["bold"]],
			["soft", ["italic"]],
			["gone", ["strike"]],
			["code", ["code"]],
		]);
		expect(content).toContainEqual({
			type: "link",
			url: "https://example.org",
			content: [{ type: "text", text: "docs" }],
		});
		expect(JSON.stringify(content)).toContain("[bad](javascript:alert(1))");
		expect(server.content.textContent).not.toContain("**");
	});
});

describe("task items", () => {
	const tasks: NoteDoc = {
		v: 1,
		blocks: [
			{
				type: "list",
				style: "task",
				items: [
					{
						id: "a1",
						checked: false,
						content: [{ type: "text", text: "milkeggs" }],
					},
				],
			},
		],
	};

	it("checks a task with a click and saves it", slow, async () => {
		// jsdom has no layout: give the checkbox (the item's ::before) a width,
		// so Lexical sees the click on it.
		const style = window.getComputedStyle;
		vi.spyOn(window, "getComputedStyle").mockImplementation((el, pseudo) =>
			pseudo === "::before"
				? ({ width: "16px" } as CSSStyleDeclaration)
				: style(el, pseudo),
		);
		const server = await openNote({ content: tasks });
		const item = within(server.content).getByRole("checkbox");
		expect(item).toHaveAttribute("aria-checked", "false");
		fireEvent.click(item, { clientX: 1 });
		await waitFor(() => expect(item).toHaveAttribute("aria-checked", "true"));
		const n = await saved(server, 2);
		expect(blocks(n)[0]).toMatchObject({
			items: [{ id: "a1", checked: true }],
		});
	});

	it("gives the second half of a split task a new ID", slow, async () => {
		const server = await openNote({ content: tasks });
		const editor = editorOf(server.content);
		await act(async () => {
			editor.update(
				() => {
					const text = $getRoot().getFirstDescendant();
					if ($isTextNode(text)) text.select(4, 4);
				},
				{ discrete: true },
			);
		});
		await typeInEditor(server.content, "\n");
		const n = await saved(server, 2);
		const list = blocks(n)[0];
		expect(
			list.type === "list" && list.items.map((i) => [i.id === "a1", i.content]),
		).toEqual([
			[true, [{ type: "text", text: "milk" }]],
			[false, [{ type: "text", text: "eggs" }]],
		]);
	});
});

describe("toolbar", () => {
	it("makes a task list and bold text from the toolbar", slow, async () => {
		const server = await openNote();
		await typeInEditor(server.content, "milk");
		await selectAll(server.content);
		fireEvent.click(screen.getByRole("button", { name: "Bold" }));
		fireEvent.click(screen.getByRole("button", { name: "Task list" }));
		const n = await saved(server, 2);
		expect(blocks(n)[0]).toMatchObject({
			type: "list",
			style: "task",
			items: [{ content: [{ type: "text", text: "milk", marks: ["bold"] }] }],
		});
		expect(screen.getByRole("button", { name: "Task list" })).toHaveAttribute(
			"aria-pressed",
			"true",
		);
	});

	it("rejects links other than http, https and mailto", slow, async () => {
		const server = await openNote();
		await typeInEditor(server.content, "docs");
		await selectAll(server.content);
		const { user } = server;
		await user.click(screen.getByRole("button", { name: "Link" }));
		const input = await screen.findByRole("textbox", { name: "Link address" });
		await user.clear(input);
		await user.type(input, "javascript:alert(1)");
		await user.click(screen.getByRole("button", { name: "Apply" }));
		expect(screen.getByRole("alert")).toHaveTextContent(
			"Enter an http, https or mailto address.",
		);
		await user.clear(input);
		await user.type(input, "https://example.org");
		await user.click(screen.getByRole("button", { name: "Apply" }));
		expect(
			within(server.content).getByRole("link", { name: "docs" }),
		).toHaveAttribute("href", "https://example.org");
		await waitFor(
			() =>
				expect(blocks(server.notes[0])[0]).toMatchObject({
					content: [{ type: "link", url: "https://example.org" }],
				}),
			{ timeout: 3000 },
		);
	});
});

describe("links in the editor", () => {
	it("opens a link only with Ctrl or Cmd held", async () => {
		const open = vi.spyOn(window, "open").mockReturnValue(null);
		const server = await openNote({
			content: {
				v: 1,
				blocks: [
					{
						type: "paragraph",
						content: [
							{
								type: "link",
								url: "https://example.org/docs",
								content: [{ type: "text", text: "docs" }],
							},
						],
					},
				],
			},
		});
		const link = within(server.content).getByRole("link", { name: "docs" });
		fireEvent.click(link);
		expect(open).not.toHaveBeenCalled();
		fireEvent.click(link, { ctrlKey: true });
		expect(open).toHaveBeenCalledWith(
			"https://example.org/docs",
			"_blank",
			"noopener,noreferrer",
		);
	});
});

describe("note editor page", () => {
	it("shows the note's fields and its display title in the tab", async () => {
		await openNote({ title: "Groceries", color: "yellow", categoryId: "work" });
		expect(screen.getByRole("textbox", { name: "Title" })).toHaveValue(
			"Groceries",
		);
		expect(
			await screen.findByRole("button", {
				name: "Category: Work. Change category",
			}),
		).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "Colour: Yellow" }),
		).toBeInTheDocument();
		expect(screen.getByRole("link", { name: "Notes" })).toHaveAttribute(
			"href",
			"/notes",
		);
		await waitFor(() => expect(document.title).toBe("Groceries - Boardly"));
	});

	it("saves title, colour, icon and category changes", slow, async () => {
		const server = await openNote({ title: null });
		const { user } = server;
		await user.type(
			screen.getByRole("textbox", { name: "Title" }),
			"Groceries",
		);
		await user.click(screen.getByRole("button", { name: "Colour: No colour" }));
		await user.click(await screen.findByRole("radio", { name: "Yellow" }));
		await user.click(screen.getByRole("button", { name: /^Icon:/ }));
		await user.click(await screen.findByRole("radio", { name: "Home" }));
		await user.click(screen.getByRole("button", { name: "Done" }));
		await user.click(screen.getByRole("button", { name: "Add category" }));
		await user.click(
			await screen.findByRole("menuitemradio", { name: "Work" }),
		);
		await waitFor(
			() =>
				expect(server.notes[0]).toMatchObject({
					title: "Groceries",
					color: "yellow",
					icon: { type: "symbol", name: "home" },
					categoryId: "work",
				}),
			{ timeout: 3000 },
		);
		expect(
			screen.getByRole("textbox", { name: "Title" }).closest("section")?.style
				.background,
		).toBe("var(--color-note-yellow)");
		await waitFor(() => expect(document.title).toBe("Groceries - Boardly"));
	});

	it("shows the not-found page for an unknown note", async () => {
		fakeNotesServer();
		renderApp("/notes/doesnotexist");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeInTheDocument();
		expect(screen.queryByRole("textbox", { name: "Content" })).toBeNull();
	});

	it("deletes a note after confirming", async () => {
		const server = await openNote({ title: "Groceries" });
		const { user, router } = server;
		await user.click(screen.getByRole("button", { name: "More actions" }));
		await user.click(
			await screen.findByRole("menuitem", { name: "Delete note" }),
		);
		const dialog = await screen.findByRole("alertdialog", {
			name: "Delete note?",
		});
		expect(dialog).toHaveTextContent("“Groceries” will be deleted.");
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		expect(server.notes).toHaveLength(1);
		await user.click(screen.getByRole("button", { name: "More actions" }));
		await user.click(
			await screen.findByRole("menuitem", { name: "Delete note" }),
		);
		await user.click(
			within(await screen.findByRole("alertdialog")).getByRole("button", {
				name: "Delete",
			}),
		);
		await waitFor(() => expect(router.state.location.pathname).toBe("/notes"));
		expect(server.notes).toHaveLength(0);
	});
});

describe("autosave", () => {
	it(
		"shows Saving… and then Saved, and sends only the changed fields",
		slow,
		async () => {
			const server = await openNote({ title: "Groceries" });
			await typeInEditor(server.content, "eggs");
			expect(server.api.callsTo("PATCH", "/api/notes/n1")).toHaveLength(0);
			expect(
				await screen.findByText("Saved", {}, { timeout: 3000 }),
			).toBeInTheDocument();
			const patches = server.api.callsTo("PATCH", "/api/notes/n1");
			expect(patches).toHaveLength(1);
			expect(Object.keys(patches[0].body as object).sort()).toEqual([
				"content",
				"version",
			]);
		},
	);

	it("keeps the text and offers Retry when a save fails", slow, async () => {
		const server = await openNote();
		server.api.set({ "PATCH /api/notes/n1": apiError(500, "error") });
		await typeInEditor(server.content, "eggs");
		const retry = await screen.findByRole(
			"button",
			{ name: "Retry" },
			{ timeout: 3000 },
		);
		expect(screen.getByText("Not saved")).toBeInTheDocument();
		expect(server.content).toHaveTextContent("eggs");
		server.api.set({
			"PATCH /api/notes/n1": ({ body }) => ({
				body: { ...server.notes[0], ...(body as object), version: 2 },
			}),
		});
		fireEvent.click(retry);
		expect(await screen.findByText("Saved")).toBeInTheDocument();
	});

	it(
		"stops and offers a reload when the note was changed elsewhere",
		slow,
		async () => {
			const server = await openNote();
			// Another tab saves first.
			server.notes[0].version = 5;
			server.notes[0].content = {
				v: 1,
				blocks: [
					{
						type: "paragraph",
						content: [{ type: "text", text: "from the other tab" }],
					},
				],
			};
			await typeInEditor(server.content, "mine");
			const alert = await screen.findByRole("alert", {}, { timeout: 3000 });
			expect(alert).toHaveTextContent("This note was changed elsewhere.");
			await waitFor(() =>
				expect(server.content).toHaveAttribute("contenteditable", "false"),
			);
			expect(server.content).toHaveTextContent("mine");
			fireEvent.click(within(alert).getByRole("button", { name: "Reload" }));
			await waitFor(() =>
				expect(
					screen.getByRole("textbox", { name: "Content" }),
				).toHaveTextContent("from the other tab"),
			);
			expect(screen.queryByRole("alert")).toBeNull();
		},
	);

	it(
		"asks before closing the tab while changes are unsaved",
		slow,
		async () => {
			const server = await openNote();
			await typeInEditor(server.content, "x");
			const event = new Event("beforeunload", { cancelable: true });
			window.dispatchEvent(event);
			expect(event.defaultPrevented).toBe(true);
			await screen.findByText("Saved", {}, { timeout: 3000 });
			const later = new Event("beforeunload", { cancelable: true });
			window.dispatchEvent(later);
			expect(later.defaultPrevented).toBe(false);
		},
	);

	it("saves right away when the page is left", slow, async () => {
		const server = await openNote();
		await typeInEditor(server.content, "quick");
		await server.user.click(screen.getByRole("link", { name: "Notes" }));
		await waitFor(() =>
			expect(server.api.callsTo("PATCH", "/api/notes/n1")).toHaveLength(1),
		);
		expect(blocks(server.notes[0])[0]).toMatchObject({
			content: [{ text: "quick" }],
		});
	});
});

describe("new notes", () => {
	it(
		"creates the note on the first change and keeps the editor",
		slow,
		async () => {
			const { api, notes } = fakeNotesServer([], [category("work", "Work")]);
			const { router } = renderApp("/notes/new?category=work");
			const content = await screen.findByRole("textbox", { name: "Content" });
			await typeInEditor(content, "milk");
			await waitFor(() => expect(notes).toHaveLength(1), { timeout: 3000 });
			expect(api.callsTo("POST", "/api/notes")[0].body).toMatchObject({
				title: null,
				categoryId: "work",
				content: {
					v: 1,
					blocks: [{ type: "paragraph", content: [{ text: "milk" }] }],
				},
			});
			await waitFor(() =>
				expect(router.state.location.pathname).toBe(`/notes/${notes[0].id}`),
			);
			expect(screen.getByRole("textbox", { name: "Content" })).toBe(content);
			expect(router.history.length).toBe(1);
			await typeInEditor(content, " eggs");
			await waitFor(() => expect(notes[0].version).toBe(2), { timeout: 3000 });
			expect(api.callsTo("POST", "/api/notes")).toHaveLength(1);
		},
	);

	it("doesn't create a note that was left unchanged", async () => {
		const { api } = fakeNotesServer();
		const { user, router } = renderApp("/notes/new");
		await screen.findByRole("textbox", { name: "Content" });
		expect(
			screen.getByRole("button", { name: "Add category" }),
		).toBeInTheDocument();
		await user.click(screen.getByRole("link", { name: "Notes" }));
		await waitFor(() => expect(router.state.location.pathname).toBe("/notes"));
		expect(api.callsTo("POST", "/api/notes")).toHaveLength(0);
	});
});
