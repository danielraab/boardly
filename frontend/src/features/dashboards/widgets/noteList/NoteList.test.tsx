import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import { section, viewData } from "#/test/dashboards";
import { category, fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { NoteListConfig } from ".";
import type { ListedNote, NoteListCategory, NoteListData } from "./list";
import { NoteListView } from "./NoteListView";

const rosesRow: ListedNote = {
	id: "roses",
	title: "Roses",
	icon: { type: "symbol", name: "shopping-cart" },
	category: { title: "Garden", path: ["Home", "Garden"] },
};
const plumberRow: ListedNote = {
	id: "plumber",
	title: null,
	icon: null,
	category: null,
};

function view(
	data: NoteListData,
	fields: {
		category?: NoteListCategory;
		title?: string;
		size?: "S" | "M" | "L";
		onOpen?: (note: ListedNote) => void;
	} = {},
) {
	return render(
		<NoteListView
			data={data}
			category={fields.category ?? "category"}
			size={fields.size ?? "M"}
			title={fields.title}
			headingLevel={4}
			onOpen={fields.onOpen}
		/>,
	);
}

describe("NoteListView", () => {
	it("shows icon, display title and a second line only for other categories, and no time", () => {
		view({ notes: [rosesRow, plumberRow] });
		const rows = screen.getAllByRole("listitem");
		expect(rows[0]).toHaveTextContent(/^RosesGarden$/);
		expect(rows[0].querySelector("[data-icon='shopping-cart']")).not.toBeNull();
		expect(within(rows[0]).getByText("Garden")).toHaveAttribute(
			"title",
			"Home › Garden",
		);
		expect(rows[1]).toHaveTextContent(/^Untitled note$/);
		expect(rows[1].querySelector("svg")).toBeNull();
		expect(document.querySelector("time")).toBeNull();
		expect(screen.queryByRole("button")).toBeNull();
	});

	it("calls onOpen with the activated note", async () => {
		const onOpen = vi.fn();
		view({ notes: [rosesRow] }, { onOpen });
		await userEvent.click(screen.getByRole("button", { name: /Roses/ }));
		expect(onOpen).toHaveBeenCalledWith(rosesRow);
	});

	it("shows the empty and missing states", () => {
		const { rerender } = view({ notes: [] });
		expect(screen.getByText("No notes in this category")).toBeVisible();
		rerender(<NoteListView data={{ notes: [] }} category="all" size="M" />);
		expect(screen.getByText("No notes")).toBeVisible();
		rerender(
			<NoteListView data={{ missing: true }} category="category" size="M" />,
		);
		expect(screen.getByText("This category no longer exists")).toBeVisible();
	});

	it("shows a title and scales with the size", () => {
		view({ notes: [rosesRow] }, { title: "Work notes", size: "L" });
		const heading = screen.getByRole("heading", {
			level: 4,
			name: "Work notes",
		});
		expect(heading.parentElement).toHaveClass("text-base");
	});
});

const roses = note("roses", {
	title: "Roses",
	categoryId: "garden",
	updatedAt: "2026-10-20T12:00:00.000Z",
});
const plumber = note("plumber", {
	title: null,
	content: paragraphs("call the plumber"),
	categoryId: "home",
	updatedAt: "2026-10-10T12:00:00.000Z",
});
const draft = note("draft", {
	title: "Draft",
	updatedAt: "2026-10-01T12:00:00.000Z",
});
const home = category("home", "Home");
const garden = category("garden", "Garden", "home");

function listWidget(
	config: Partial<NoteListConfig> = {},
	data?: NoteListData,
): ContentWidget {
	return {
		id: "w1",
		kind: "note_list",
		width: 2,
		height: "auto",
		config: {
			category: "category",
			categoryId: "home",
			subcategories: true,
			sort: "updated",
			limit: 10,
			title: null,
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
	};
}

function dashboard(sections: () => Widget[], extra = {}, isOwner = true) {
	return fakeNotesServer([roses, plumber, draft], [home, garden], {
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

describe("the note list widget", () => {
	it("opens a note's editor in a modal for the owner", async () => {
		dashboard(() => [
			section("Work", {}, listWidget({}, { notes: [rosesRow] })),
		]);
		const { user, router } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: /Roses/ }));
		const dialog = await screen.findByRole("dialog", { name: "Roses" });
		expect(
			await within(dialog).findByRole("textbox", { name: "Title" }),
		).toHaveValue("Roses");
		expect(router.state.location.pathname).toBe("/ann/home");
	});

	it("can't be activated by visitors", async () => {
		dashboard(
			() => [
				section(
					"Work",
					{},
					listWidget(
						{ showIfPublic: true },
						{ notes: [{ title: "Roses", icon: null, category: null }] },
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

	it("shows the owner's current notes of the category in edit mode, without opening them", async () => {
		dashboard(() => [
			section("Work", {}, listWidget({ sort: "title" }, { notes: [] })),
		]);
		const { user } = renderApp("/ann/home?edit=true");
		const first = await screen.findByText("call the plumber");
		expect(rowTitles()).toEqual(["call the plumber", "Roses"]);
		const rows = screen.getAllByRole("listitem");
		expect(within(rows[0]).queryByText("Home")).toBeNull();
		expect(within(rows[1]).getByText("Garden")).toHaveAttribute(
			"title",
			"Home › Garden",
		);
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
			within(picker).getByRole("button", { name: /^Note list/ }),
		);

		const dialog = await screen.findByRole("dialog", {
			name: "Add note list",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		const subcategories = within(dialog).getByLabelText(
			"Include subcategories",
		);
		expect(subcategories).toBeChecked();
		expect(subcategories).toBeDisabled();
		expect(within(dialog).getByLabelText("Sort")).toHaveValue("updated");
		expect(within(dialog).getByLabelText("Number of notes")).toHaveValue(10);
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();

		await user.click(
			await within(dialog).findByRole("radio", { name: "Uncategorized" }),
		);
		expect(add).toBeEnabled();
		expect(subcategories).toBeDisabled();
		await user.click(within(dialog).getByRole("radio", { name: "Home" }));
		expect(subcategories).toBeEnabled();
		await user.click(subcategories);

		const limit = within(dialog).getByLabelText("Number of notes");
		await user.clear(limit);
		await user.type(limit, "0");
		expect(add).toBeDisabled();
		await user.clear(limit);
		await user.type(limit, "5");
		await user.selectOptions(
			within(dialog).getByLabelText("Sort"),
			"Alphabetical",
		);
		await user.click(add);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(rowTitles()).toEqual(["call the plumber"]);
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "note_list",
					config: {
						category: "category",
						categoryId: "home",
						subcategories: false,
						sort: "title",
						limit: 5,
						title: null,
						size: "M",
						showIfPublic: false,
					},
				},
			],
		});
	});
});
