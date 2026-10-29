import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { NoteDoc } from "#/features/notes/document/types";
import { apiError } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { category, fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { NoteConfig } from ".";
import { NotePicker } from "./NotePicker";
import type { NoteData } from "./NoteWidgetView";

const todos: NoteDoc = {
	v: 1,
	blocks: [
		{
			type: "list",
			style: "task",
			items: [
				{ id: "t1", checked: false, content: [{ type: "text", text: "milk" }] },
				{ id: "t2", checked: true, content: [{ type: "text", text: "bread" }] },
			],
		},
	],
};

const groceries = note("groceries", {
	title: "Groceries",
	icon: { type: "symbol", name: "shopping-cart" },
	color: "green",
	categoryId: "garden",
	content: todos,
	version: 3,
});
const untitled = note("untitled", {
	title: null,
	content: paragraphs("call the plumber tomorrow"),
});
const home = category("home", "Home");
const garden = category("garden", "Garden", "home", { color: "teal" });

/** The data the server resolves for groceries, for the owner or a visitor. */
function resolved(owner: boolean, fields: Partial<NoteData> = {}): NoteData {
	return {
		...(owner ? { id: groceries.id, version: groceries.version } : {}),
		title: groceries.title,
		icon: groceries.icon,
		color: groceries.color,
		content: groceries.content,
		category: {
			title: "Garden",
			icon: null,
			color: "teal",
			...(owner ? { path: ["Home", "Garden"] } : {}),
		},
		...fields,
	} as NoteData;
}

function noteWidget(
	id: string,
	config: Partial<NoteConfig> = {},
	data?: NoteData,
	fields: Partial<ContentWidget> = {},
): ContentWidget {
	return {
		id,
		kind: "note",
		width: 2,
		height: "auto",
		config: {
			noteId: groceries.id,
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
		...fields,
	};
}

/** A dashboard of the owner with a notes server behind it. */
function dashboard(sections: () => Widget[], extra = {}, isOwner = true) {
	return fakeNotesServer([groceries, untitled], [home, garden], {
		"GET /api/u/ann/home": () => ({
			body: viewData(
				sections(),
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
		...extra,
	});
}

describe("the note widget", () => {
	it("shows the note like a card, with a badge showing the category's path", async () => {
		dashboard(() => [
			section("Work", {}, noteWidget("w1", {}, resolved(true))),
		]);
		renderApp("/ann/home");
		const open = await screen.findByRole("button", { name: "Open Groceries" });
		const heading = screen.getByRole("heading", { name: "Groceries" });
		const card = heading.closest("article") as HTMLElement;
		expect(card.tagName).toBe("ARTICLE");
		expect(card).toContainElement(open);
		expect(card.style.background).toContain("--color-note-green");
		expect(card.querySelector("[data-icon='shopping-cart']")).not.toBeNull();
		expect(
			within(card).getByRole("checkbox", { name: "milk" }),
		).not.toBeChecked();
		expect(within(card).getByRole("checkbox", { name: "bread" })).toBeChecked();
		const badge = within(card).getByText("Garden").parentElement as HTMLElement;
		expect(badge).toHaveAttribute("title", "Home › Garden");
		// The badge is in the header, next to the title and before the content.
		expect(heading.parentElement).toContainElement(badge);
		expect(
			badge.compareDocumentPosition(
				within(card).getByRole("checkbox", { name: "milk" }),
			) & Node.DOCUMENT_POSITION_FOLLOWING,
		).toBeTruthy();
	});

	it("shows no title line for a note without a title, and scales its text", async () => {
		dashboard(() => [
			section(
				"Work",
				{},
				noteWidget(
					"w1",
					{ size: "L" },
					resolved(true, { title: null, icon: null, category: null }),
				),
			),
		]);
		renderApp("/ann/home");
		await screen.findByRole("checkbox", { name: "milk" });
		expect(screen.queryByRole("heading", { name: /Groceries/ })).toBeNull();
		expect(screen.getByText("milk").closest(".space-y-1\\.5")).toHaveClass(
			"text-base",
		);
	});

	it("shows the pin and archive marks, also to visitors", async () => {
		dashboard(
			() => [
				section(
					"Work",
					{},
					noteWidget(
						"w1",
						{ showIfPublic: true },
						resolved(false, { pinned: true, archived: true }),
					),
				),
			],
			{},
			false,
		);
		renderApp("/ann/home");
		const heading = await screen.findByRole("heading", { name: "Groceries" });
		const card = heading.closest("article") as HTMLElement;
		expect(within(card).getByRole("img", { name: "Pinned" })).toBeVisible();
		expect(within(card).getByText("Archived")).toBeVisible();
	});

	it("says when the note no longer exists", async () => {
		dashboard(() => [
			section("Work", {}, noteWidget("w1", {}, { missing: true })),
		]);
		renderApp("/ann/home");
		expect(await screen.findByText("This note no longer exists")).toBeVisible();
	});

	it("is read-only for visitors", async () => {
		dashboard(
			() => [
				section(
					"Work",
					{},
					noteWidget("w1", { showIfPublic: true }, resolved(false)),
				),
			],
			{},
			false,
		);
		renderApp("/ann/home");
		const milk = await screen.findByRole("checkbox", { name: "milk" });
		expect(milk).toBeDisabled();
		expect(screen.queryByRole("button", { name: /^Open/ })).toBeNull();
		expect(screen.getByText("Garden").parentElement).toHaveAttribute(
			"title",
			"Garden",
		);
	});

	it("saves a ticked todo right away for the owner", async () => {
		const { api, notes } = dashboard(() => [
			section("Work", {}, noteWidget("w1", {}, resolved(true))),
		]);
		const { user, router } = renderApp("/ann/home");
		await screen.findByRole("button", { name: "Open Groceries" });
		await user.click(screen.getByRole("checkbox", { name: "milk" }));

		expect(screen.getByRole("checkbox", { name: "milk" })).toBeChecked();
		await waitFor(() =>
			expect(api.callsTo("PATCH", "/api/notes/groceries")).toHaveLength(1),
		);
		const [call] = api.callsTo("PATCH", "/api/notes/groceries");
		expect(call.body).toMatchObject({ version: 3 });
		const saved = notes.find((n) => n.id === "groceries");
		expect(saved?.content.blocks[0]).toMatchObject({
			items: [
				{ id: "t1", checked: true },
				{ id: "t2", checked: true },
			],
		});
		expect(router.state.location.pathname).toBe("/ann/home");

		// The next toggle builds on the saved version.
		await user.click(screen.getByRole("checkbox", { name: "bread" }));
		await waitFor(() =>
			expect(api.callsTo("PATCH", "/api/notes/groceries")).toHaveLength(2),
		);
		expect(api.callsTo("PATCH", "/api/notes/groceries")[1].body).toMatchObject({
			version: 4,
		});
	});

	it("shows the note as it was when saving fails", async () => {
		dashboard(
			() => [section("Work", {}, noteWidget("w1", {}, resolved(true)))],
			{
				"PATCH /api/notes/groceries": apiError(500, "internal"),
			},
		);
		const { user } = renderApp("/ann/home");
		await screen.findByRole("button", { name: "Open Groceries" });
		await user.click(screen.getByRole("checkbox", { name: "milk" }));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The note couldn't be changed. Please try again.",
		);
		expect(screen.getByRole("checkbox", { name: "milk" })).not.toBeChecked();
	});

	it("shows the current note when it was changed elsewhere", async () => {
		let version = 3;
		let content = todos;
		dashboard(
			() => [
				section(
					"Work",
					{},
					noteWidget(
						"w1",
						{},
						resolved(true, { version, content } as Partial<NoteData>),
					),
				),
			],
			{
				"PATCH /api/notes/groceries": () => {
					version = 4;
					content = paragraphs("changed in another tab");
					return apiError(409, "version_conflict");
				},
			},
		);
		const { user } = renderApp("/ann/home");
		await screen.findByRole("button", { name: "Open Groceries" });
		await user.click(screen.getByRole("checkbox", { name: "milk" }));
		expect(await screen.findByText("changed in another tab")).toBeVisible();
		expect(screen.getByRole("alert")).toHaveTextContent(
			"The note was changed elsewhere. The widget shows the current state.",
		);
	});

	it("opens the note in a modal and shows its changes on the widget", async () => {
		const server = dashboard(() => {
			const n = server.notes.find((x) => x.id === "groceries");
			const data = n
				? resolved(true, { title: n.title, version: n.version })
				: { missing: true as const };
			return [section("Work", {}, noteWidget("w1", {}, data))];
		});
		const { user, router } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Open Groceries" }),
		);

		const dialog = await screen.findByRole("dialog", { name: "Groceries" });
		const title = await within(dialog).findByRole("textbox", { name: "Title" });
		expect(title).toHaveValue("Groceries");
		expect(router.state.location.pathname).toBe("/ann/home");
		await user.clear(title);
		await user.type(title, "Shopping");
		await user.click(within(dialog).getByRole("button", { name: "Close" }));

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		// Closing saves and then reloads the dashboard, which can take a while.
		expect(
			await screen.findByRole(
				"heading",
				{ name: "Shopping" },
				{ timeout: 3000 },
			),
		).toBeVisible();
		expect(server.notes.find((n) => n.id === "groceries")?.title).toBe(
			"Shopping",
		);
		expect(router.state.location.pathname).toBe("/ann/home");
	});

	it("shows a note deleted in the modal as missing", async () => {
		const server = dashboard(() => {
			const n = server.notes.find((x) => x.id === "groceries");
			return [
				section(
					"Work",
					{},
					noteWidget("w1", {}, n ? resolved(true) : { missing: true }),
				),
			];
		});
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Open Groceries" }),
		);
		const dialog = await screen.findByRole("dialog", { name: "Groceries" });
		await user.click(
			await within(dialog).findByRole("button", { name: "More actions" }),
		);
		await user.click(
			await screen.findByRole("menuitem", { name: "Delete note" }),
		);
		const confirm = await screen.findByRole("alertdialog", {
			name: "Delete note?",
		});
		await user.click(within(confirm).getByRole("button", { name: "Delete" }));

		expect(await screen.findByText("This note no longer exists")).toBeVisible();
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	});

	it("shows the owner's current note in edit mode, without toggling", async () => {
		const { api } = dashboard(() => [
			section("Work", {}, noteWidget("w1", {}, resolved(true))),
		]);
		renderApp("/ann/home?edit=true");
		const milk = await screen.findByRole("checkbox", { name: "milk" });
		expect(milk).toBeDisabled();
		expect(screen.getByText("Garden").parentElement).toHaveAttribute(
			"title",
			"Home › Garden",
		);
		expect(screen.queryByRole("button", { name: /^Open/ })).toBeNull();
		expect(api.callsTo("GET", "/api/notes")).toHaveLength(1);
	});

	it("is added through the picker, needs a note, and saves its settings", async () => {
		const { api } = dashboard(() => [section("Work")], {
			"PUT /api/dashboards/d1/layout": ({ body }: { body: unknown }) => ({
				body: {
					version: 4,
					sections: (body as { sections: Widget[] }).sections,
				},
			}),
		});
		const { user, router } = renderApp("/ann/home?edit=true");
		await user.click(await screen.findByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(picker).getByRole("button", { name: /^Note(?! list)/ }),
		);

		const dialog = await screen.findByRole("dialog", { name: "Add note" });
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();
		expect(within(dialog).getByLabelText("Fit the content")).toBeChecked();

		const search = await within(dialog).findByRole("combobox", {
			name: "Note",
		});
		expect(search).toHaveValue("");
		expect(search).toHaveAttribute("placeholder", "Choose a note");
		await user.click(search);
		expect(options()).toEqual([
			"call the plumber tomorrow",
			"GroceriesHome › Garden",
		]);
		await user.click(screen.getByRole("option", { name: /Groceries/ }));
		expect(search).toHaveValue("Groceries");
		await user.selectOptions(within(dialog).getByLabelText("Size"), "L");
		await user.click(within(dialog).getByLabelText("Show on public dashboard"));
		await user.click(add);

		expect(await screen.findByRole("checkbox", { name: "milk" })).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "note",
					config: { noteId: "groceries", size: "L", showIfPublic: true },
				},
			],
		});
	});

	it("shows a deleted note in the settings and lets another one be chosen", async () => {
		const { user } = (() => {
			dashboard(() => [
				section("Work", {}, noteWidget("w1", { noteId: "gone" })),
			]);
			return renderApp("/ann/home?edit=true");
		})();
		expect(await screen.findByText("This note no longer exists")).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Edit Note" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit note" });
		const apply = within(dialog).getByRole("button", { name: "Apply" });
		expect(
			within(dialog).getByText("This note no longer exists"),
		).toBeVisible();
		expect(apply).toBeDisabled();
		const search = await within(dialog).findByRole("combobox", {
			name: "Note",
		});
		expect(search).toHaveValue("");
		await user.type(search, "groc");
		await user.click(screen.getByRole("option", { name: /Groceries/ }));
		expect(apply).toBeEnabled();
	});
});

const options = () => screen.getAllByRole("option").map((o) => o.textContent);

const office = category("office", "Office", "work");
const work = category("work", "Work");
const workGroceries = note("work-groceries", {
	title: "Groceries",
	categoryId: "office",
});
const oldPlan = note("old-plan", { title: "Old plan", archived: true });
const pickerNotes = [groceries, workGroceries, untitled, oldPlan];

function Picker({ initial = "" }: { initial?: string }) {
	const [value, setValue] = useState(initial);
	return (
		<>
			<label htmlFor="picker">Note</label>
			<NotePicker
				id="picker"
				notes={pickerNotes}
				categories={[home, garden, work, office]}
				value={value}
				onChange={setValue}
			/>
			<output>{value}</output>
			<button type="button">Next</button>
		</>
	);
}

describe("the note picker", () => {
	it("tells notes with the same title apart by their category", async () => {
		const user = userEvent.setup();
		render(<Picker />);
		await user.type(screen.getByRole("combobox"), "groc");
		expect(options()).toEqual([
			"GroceriesHome › Garden",
			"GroceriesWork › Office",
		]);
		await user.clear(screen.getByRole("combobox"));
		await user.type(screen.getByRole("combobox"), "office");
		expect(options()).toEqual(["GroceriesWork › Office"]);
		await user.click(screen.getByRole("option", { name: /Groceries/ }));
		expect(screen.getByRole("status")).toHaveTextContent("work-groceries");
	});

	it("marks and dims archived notes, which can still be chosen", async () => {
		const user = userEvent.setup();
		render(<Picker />);
		await user.type(screen.getByRole("combobox"), "plan");
		const option = screen.getByRole("option", { name: /Old plan/ });
		expect(within(option).getByRole("img", { name: "Archived" })).toBeVisible();
		expect(option).toHaveClass("opacity-70");
		await user.click(option);
		expect(screen.getByRole("status")).toHaveTextContent("old-plan");
	});

	it("shows the chosen note with its category path and offers all notes", async () => {
		const user = userEvent.setup();
		render(<Picker initial="work-groceries" />);
		const search = screen.getByRole("combobox");
		expect(search).toHaveValue("Groceries");
		expect(screen.getByText("Work › Office")).toBeVisible();
		await user.click(search);
		expect(options()).toHaveLength(pickerNotes.length);
		await user.type(search, "x");
		expect(screen.queryByText("Work › Office")).toBeNull();
		await user.tab();
		expect(search).toHaveValue("Groceries");
		expect(screen.getByText("Work › Office")).toBeVisible();
	});

	it("says when nothing matches", async () => {
		const user = userEvent.setup();
		render(<Picker />);
		await user.type(screen.getByRole("combobox"), "nothing like it");
		expect(await screen.findByText("No matching notes")).toBeVisible();
	});
});
