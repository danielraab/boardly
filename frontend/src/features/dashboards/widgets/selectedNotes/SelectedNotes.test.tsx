import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import { section, viewData } from "#/test/dashboards";
import { category, fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { ListedNote, NoteListData } from "../noteList/list";
import type { SelectedNotesConfig } from ".";
import { ChosenNotes } from "./ChosenNotes";
import { NoteSearch } from "./NoteSearch";
import { maxSelectedNotes } from "./selected";

const roses = note("roses", { title: "Roses", categoryId: "garden" });
const plumber = note("plumber", {
	title: null,
	content: paragraphs("call the plumber"),
});
const draft = note("draft", { title: "Draft", categoryId: "home" });
const arzte = note("arzte", { title: "Ärzte" });
const home = category("home", "Home");
const garden = category("garden", "Garden", "home");
const allNotes = [roses, plumber, draft, arzte];

function Chosen({ initial }: { initial: string[] }) {
	const [ids, setIds] = useState(initial);
	return (
		<>
			<ChosenNotes noteIds={ids} notes={allNotes} onChange={setIds} />
			<NoteSearch
				id="search"
				notes={allNotes}
				categories={[home, garden]}
				chosen={ids}
				onAdd={(id) => setIds((all) => [...all, id])}
			/>
		</>
	);
}

const chosenNames = () =>
	screen
		.getAllByRole("listitem")
		.map((li) => li.querySelector("span[title]")?.textContent);

const options = () => screen.getAllByRole("option").map((o) => o.textContent);

describe("the note search", () => {
	it("filters by title and category, ignoring accents, and hides chosen notes", async () => {
		const user = userEvent.setup();
		render(<Chosen initial={["roses"]} />);
		const search = screen.getByPlaceholderText("Add a note…");
		await user.click(search);
		await user.type(search, "a");
		expect(options()).toEqual(["Ärzte", "call the plumber", "DraftHome"]);
		await user.clear(search);
		await user.type(search, "arz");
		expect(options()).toEqual(["Ärzte"]);
		await user.clear(search);
		await user.type(search, "home");
		expect(options()).toEqual(["DraftHome"]);
		await user.clear(search);
		await user.type(search, "nothing like it");
		expect(await screen.findByText("No matching notes")).toBeVisible();
	});

	it("adds a chosen note at the end and offers it no more", async () => {
		const user = userEvent.setup();
		render(<Chosen initial={["roses"]} />);
		const search = screen.getByPlaceholderText("Add a note…");
		await user.type(search, "draft");
		await user.click(screen.getByRole("option", { name: /Draft/ }));
		expect(chosenNames()).toEqual(["Roses", "Draft"]);
		expect(search).toHaveValue("");
		await user.type(search, "dra");
		expect(screen.queryByRole("option", { name: /Draft/ })).toBeNull();
	});

	it("is disabled when the maximum is chosen", () => {
		const ids = Array.from({ length: maxSelectedNotes }, (_, i) => `n${i}`);
		render(
			<NoteSearch
				id="search"
				notes={allNotes}
				categories={[]}
				chosen={ids}
				onAdd={() => {}}
			/>,
		);
		expect(screen.getByPlaceholderText("Add a note…")).toBeDisabled();
		expect(
			screen.getByText("At most 100 notes can be selected."),
		).toBeVisible();
	});
});

describe("the chosen notes", () => {
	it("move up and down, with the ends disabled", async () => {
		const user = userEvent.setup();
		render(<Chosen initial={["draft", "plumber", "roses"]} />);
		expect(
			screen.getByRole("button", { name: "Move Draft up" }),
		).toBeDisabled();
		expect(
			screen.getByRole("button", { name: "Move Roses down" }),
		).toBeDisabled();
		await user.click(screen.getByRole("button", { name: "Move Roses up" }));
		expect(chosenNames()).toEqual(["Draft", "Roses", "call the plumber"]);
		await user.click(screen.getByRole("button", { name: "Move Draft down" }));
		expect(chosenNames()).toEqual(["Roses", "Draft", "call the plumber"]);
	});

	it("can be removed and are then offered again", async () => {
		const user = userEvent.setup();
		render(<Chosen initial={["draft", "roses"]} />);
		await user.click(screen.getByRole("button", { name: "Remove Draft" }));
		expect(chosenNames()).toEqual(["Roses"]);
		await user.type(screen.getByPlaceholderText("Add a note…"), "dra");
		expect(screen.getByRole("option", { name: /Draft/ })).toBeVisible();
	});

	it("shows notes that no longer exist so they can be removed", async () => {
		const user = userEvent.setup();
		render(<Chosen initial={["roses", "gone"]} />);
		expect(chosenNames()).toEqual(["Roses", "This note no longer exists"]);
		await user.click(
			screen.getByRole("button", {
				name: "Remove This note no longer exists",
			}),
		);
		expect(chosenNames()).toEqual(["Roses"]);
	});
});

const rosesRow: ListedNote = {
	id: "roses",
	title: "Roses",
	icon: null,
	color: "green",
	category: { title: "Garden", path: ["Home", "Garden"] },
};

function selectedWidget(
	config: Partial<SelectedNotesConfig> = {},
	data?: NoteListData,
): ContentWidget {
	return {
		id: "w1",
		kind: "selected_notes",
		width: 2,
		height: "auto",
		config: {
			noteIds: ["roses"],
			title: null,
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
	};
}

function dashboard(sections: () => Widget[], extra = {}, isOwner = true) {
	return fakeNotesServer(allNotes, [home, garden], {
		"GET /api/u/ann/home": () => ({
			body: viewData(
				sections(),
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
		...extra,
	});
}

const rowTitles = () =>
	screen
		.getAllByRole("listitem")
		.map((r) => r.querySelector(".font-medium")?.textContent);

describe("the selected notes widget", () => {
	it("shows its rows and opens a note's editor in a modal for the owner", async () => {
		dashboard(() => [
			section("Work", {}, selectedWidget({}, { notes: [rosesRow] })),
		]);
		const { user, router } = renderApp("/ann/home");
		const row = await screen.findByRole("button", { name: /Roses/ });
		expect(within(row).getByText("Garden")).toHaveAttribute(
			"title",
			"Home › Garden",
		);
		await user.click(row);
		const dialog = await screen.findByRole("dialog", { name: "Roses" });
		expect(
			await within(dialog).findByRole("textbox", { name: "Title" }),
		).toHaveValue("Roses");
		expect(router.state.location.pathname).toBe("/ann/home");
	});

	it("says when none of its notes exist", async () => {
		dashboard(() => [
			section("Work", {}, selectedWidget({}, { missing: true })),
		]);
		renderApp("/ann/home");
		expect(
			await screen.findByText("None of the selected notes exist anymore"),
		).toBeVisible();
	});

	it("can't be activated by visitors", async () => {
		dashboard(
			() => [
				section(
					"Work",
					{},
					selectedWidget(
						{ showIfPublic: true },
						{
							notes: [
								{ title: "Roses", icon: null, color: null, category: null },
							],
						},
					),
				),
			],
			{},
			false,
		);
		renderApp("/ann/home");
		expect(await screen.findByText("Roses")).toBeVisible();
		expect(screen.queryByRole("button", { name: /Roses/ })).toBeNull();
	});

	it("shows the owner's current notes in its order in edit mode, skipping deleted ones, without opening them", async () => {
		dashboard(() => [
			section(
				"Work",
				{},
				selectedWidget(
					{ noteIds: ["draft", "gone", "plumber"] },
					{ notes: [] },
				),
			),
		]);
		const { user } = renderApp("/ann/home?edit=true");
		const first = await screen.findByText("Draft");
		expect(rowTitles()).toEqual(["Draft", "call the plumber"]);
		await user.click(first);
		expect(screen.queryByRole("dialog")).toBeNull();
	});

	it("is added through the picker with its defaults and saves its settings", async () => {
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
			within(picker).getByRole("button", { name: /^Selected notes/ }),
		);

		const dialog = await screen.findByRole("dialog", {
			name: "Add selected notes",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();

		const search = await within(dialog).findByLabelText("Notes");
		await user.type(search, "roses");
		await user.click(await screen.findByRole("option", { name: /Roses/ }));
		await user.type(search, "draft");
		await user.click(await screen.findByRole("option", { name: /Draft/ }));
		expect(add).toBeEnabled();
		await user.click(
			within(dialog).getByRole("button", { name: "Move Draft up" }),
		);
		await user.click(add);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(rowTitles()).toEqual(["Draft", "Roses"]);
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "selected_notes",
					config: {
						noteIds: ["draft", "roses"],
						title: null,
						size: "M",
						showIfPublic: false,
					},
				},
			],
		});
	});
});
