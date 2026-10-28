import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import { section, viewData } from "#/test/dashboards";
import { category, fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { RecentNotesConfig } from ".";
import { RecentNotesView } from "./RecentNotesView";
import type { RecentNote, RecentNotesData } from "./recent";

const now = new Date("2026-10-27T12:00:00Z");
const day = 24 * 60 * 60 * 1000;
const daysAgo = (n: number) => new Date(now.getTime() - n * day).toISOString();

const groceriesRow: RecentNote = {
	id: "groceries",
	title: "Groceries",
	icon: { type: "symbol", name: "shopping-cart" },
	time: daysAgo(2),
	category: { title: "Garden", path: ["Home", "Garden"] },
};
const untitledRow: RecentNote = {
	id: "untitled",
	title: null,
	icon: null,
	time: daysAgo(1),
	category: null,
};

function view(
	data: RecentNotesData,
	fields: {
		days?: number | null;
		title?: string;
		size?: "S" | "M" | "L";
		onOpen?: (note: RecentNote) => void;
	} = {},
) {
	return render(
		<RecentNotesView
			data={data}
			days={fields.days ?? null}
			size={fields.size ?? "M"}
			title={fields.title}
			headingLevel={4}
			onOpen={fields.onOpen}
			now={now}
		/>,
	);
}

describe("RecentNotesView", () => {
	it("shows icon, display title, category and time of each note", () => {
		view({ notes: [groceriesRow, untitledRow] });
		const rows = screen.getAllByRole("listitem");
		expect(rows[0]).toHaveTextContent("GroceriesGarden2 days ago");
		expect(rows[0].querySelector("[data-icon='shopping-cart']")).not.toBeNull();
		expect(within(rows[0]).getByText("Garden")).toHaveAttribute(
			"title",
			"Home › Garden",
		);
		const time = within(rows[0]).getByText("2 days ago");
		expect(time.tagName).toBe("TIME");
		expect(time).toHaveAttribute("dateTime", groceriesRow.time);
		expect(time.getAttribute("title")).toMatch(/2026/);

		// Without a title, category or icon.
		expect(rows[1]).toHaveTextContent("Untitled noteyesterday");
		expect(rows[1].querySelector("svg")).toBeNull();
		expect(screen.queryByRole("button")).toBeNull();
	});

	it("shows the category's title as tooltip without a path", () => {
		view({
			notes: [{ ...groceriesRow, category: { title: "Garden" } }],
		});
		expect(screen.getByText("Garden")).toHaveAttribute("title", "Garden");
	});

	it("calls onOpen with the activated note", async () => {
		const onOpen = vi.fn();
		view({ notes: [groceriesRow] }, { onOpen });
		await userEvent.click(screen.getByRole("button", { name: /Groceries/ }));
		expect(onOpen).toHaveBeenCalledWith(groceriesRow);
	});

	it("shows the empty states", () => {
		const { rerender } = view({ notes: [] });
		expect(screen.getByText("No notes")).toBeVisible();
		rerender(
			<RecentNotesView data={{ notes: [] }} days={7} size="M" now={now} />,
		);
		expect(screen.getByText("No notes in the last 7 days")).toBeVisible();
		rerender(
			<RecentNotesView data={{ notes: [] }} days={1} size="M" now={now} />,
		);
		expect(screen.getByText("No notes in the last day")).toBeVisible();
	});

	it("shows a title and scales with the size", () => {
		view({ notes: [groceriesRow] }, { title: "Latest notes", size: "L" });
		const heading = screen.getByRole("heading", {
			level: 4,
			name: "Latest notes",
		});
		expect(heading.parentElement).toHaveClass("text-base");
		expect(
			screen.getByRole("listitem").querySelector("[data-icon='shopping-cart']"),
		).toHaveClass("size-5");
	});
});

const groceries = note("groceries", {
	title: "Groceries",
	categoryId: "garden",
	createdAt: "2026-09-01T12:00:00.000Z",
	updatedAt: "2026-10-20T12:00:00.000Z",
});
const plumber = note("plumber", {
	title: null,
	content: paragraphs("call the plumber"),
	createdAt: "2026-10-10T12:00:00.000Z",
	updatedAt: "2026-10-10T12:00:00.000Z",
});
const home = category("home", "Home");
const garden = category("garden", "Garden", "home");

function recentWidget(
	config: Partial<RecentNotesConfig> = {},
	data?: RecentNotesData,
): ContentWidget {
	return {
		id: "w1",
		kind: "recent_notes",
		width: 2,
		height: "auto",
		config: {
			sortBy: "created",
			limit: 10,
			days: null,
			title: null,
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
	};
}

function dashboard(sections: () => Widget[], extra = {}, isOwner = true) {
	return fakeNotesServer([groceries, plumber], [home, garden], {
		"GET /api/u/ann/home": () => ({
			body: viewData(
				sections(),
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
		...extra,
	});
}

describe("the recent notes widget", () => {
	it("opens a note's editor in a modal for the owner", async () => {
		dashboard(() => [
			section(
				"Work",
				{},
				recentWidget(
					{},
					{ notes: [{ ...groceriesRow, time: groceries.createdAt }] },
				),
			),
		]);
		const { user, router } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: /Groceries/ }));
		const dialog = await screen.findByRole("dialog", { name: "Groceries" });
		expect(
			await within(dialog).findByRole("textbox", { name: "Title" }),
		).toHaveValue("Groceries");
		expect(router.state.location.pathname).toBe("/ann/home");
		await user.click(within(dialog).getByRole("button", { name: "Close" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	});

	it("can't be activated by visitors", async () => {
		dashboard(
			() => [
				section(
					"Work",
					{},
					recentWidget(
						{ showIfPublic: true },
						{
							notes: [
								{
									title: "Groceries",
									icon: null,
									time: groceries.createdAt,
									category: { title: "Garden" },
								},
							],
						},
					),
				),
			],
			{},
			false,
		);
		renderApp("/ann/home");
		expect(await screen.findByText("Groceries")).toBeVisible();
		expect(screen.queryByRole("button", { name: /Groceries/ })).toBeNull();
	});

	it("shows the owner's current notes by the chosen time in edit mode, without opening them", async () => {
		const { api } = dashboard(() => [
			section("Work", {}, recentWidget({ sortBy: "updated" }, { notes: [] })),
		]);
		const { user } = renderApp("/ann/home?edit=true");
		const first = await screen.findByText("Groceries");
		const rows = screen.getAllByRole("listitem");
		expect(
			rows.map((r) => r.querySelector(".font-medium")?.textContent),
		).toEqual(["Groceries", "call the plumber"]);
		expect(within(rows[0]).getByText("Garden")).toHaveAttribute(
			"title",
			"Home › Garden",
		);
		await user.click(first);
		expect(screen.queryByRole("dialog", { name: "Groceries" })).toBeNull();
		expect(api.callsTo("GET", "/api/notes")).toHaveLength(1);
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
			within(picker).getByRole("button", { name: /^Recent notes/ }),
		);

		const dialog = await screen.findByRole("dialog", {
			name: "Add recent notes",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeEnabled();
		expect(within(dialog).getByLabelText("Sort by")).toHaveValue("created");
		expect(within(dialog).getByLabelText("Number of notes")).toHaveValue(10);
		expect(within(dialog).getByLabelText("Only the last … days")).toHaveValue(
			null,
		);
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();

		const limit = within(dialog).getByLabelText("Number of notes");
		await user.clear(limit);
		await user.type(limit, "101");
		expect(add).toBeDisabled();
		await user.clear(limit);
		await user.type(limit, "5");
		const days = within(dialog).getByLabelText("Only the last … days");
		await user.type(days, "0");
		expect(add).toBeDisabled();
		await user.clear(days);
		await user.type(days, "7");
		await user.selectOptions(
			within(dialog).getByLabelText("Sort by"),
			"Last changed",
		);
		await user.click(add);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "recent_notes",
					config: {
						sortBy: "updated",
						limit: 5,
						days: 7,
						title: null,
						size: "M",
						showIfPublic: false,
					},
				},
			],
		});
	});
});
