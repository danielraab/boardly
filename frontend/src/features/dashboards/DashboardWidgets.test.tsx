import { act, screen, waitFor, within } from "@testing-library/react";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import { textKind, textWidget } from "#/test/widgets";
import type { Widget } from "./api";
import { widgetKinds } from "./widgets/registry";

const frame = (id: string) => screen.getByTestId(`section-${id}`);
const widget = (id: string) => screen.getByTestId(`widget-${id}`);

beforeAll(() => {
	widgetKinds.push(textKind);
});
afterAll(() => {
	widgetKinds.splice(widgetKinds.indexOf(textKind), 1);
});

async function openEditor(sections: Widget[], more = {}) {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections) },
		...more,
	});
	const app = renderApp("/ann/home?edit=true");
	await screen.findByRole("button", { name: "Save and exit" });
	return { api, ...app };
}

describe("content widgets in view mode", () => {
	it("render through their kind, with width, height and scrolling", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[
						section(
							"Work",
							{ width: 4 },
							textWidget("n1", { width: 2, height: 2, data: "resolved" }),
							textWidget("n2", { width: 6 }),
						),
					],
					{ isOwner: false, id: undefined },
				),
			},
		});
		renderApp("/ann/home");
		expect(await screen.findByText("n1 [resolved]")).toBeVisible();
		expect(widget("n1")).toHaveAttribute("data-span", "2");
		expect(widget("n1").style.height).toBe(
			"calc(2 * var(--dashboard-row) + 1 * var(--dashboard-gap))",
		);
		expect(screen.getByText("n1 [resolved]").parentElement).toHaveClass(
			"overflow-auto",
		);
		// Wider than its section: clamped to the section's width.
		expect(widget("n2")).toHaveAttribute("data-span", "4");
		expect(screen.queryByRole("button", { name: /Edit/ })).toBeNull();
		expect(screen.queryByRole("button", { name: "Add widget" })).toBeNull();
	});
});

describe("editing controls for widgets", () => {
	it("offers Add widget in sections, but not at the top level", async () => {
		await openEditor([section("Work", {}, textWidget("n1"))]);
		expect(
			within(frame("Work")).getByRole("button", { name: "Add widget" }),
		).toBeVisible();
		expect(screen.getAllByRole("button", { name: "Add widget" })).toHaveLength(
			1,
		);
		expect(widget("n1")).toHaveClass("outline-dashed");
		expect(
			screen.getByRole("button", { name: "Move Text widget" }),
		).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Edit Text widget" }),
		).toBeVisible();
		expect(screen.getByText("n1 (editing)")).toBeVisible();
	});

	it("offers only Add widget in a section at level 5", async () => {
		const chain = (n: number): Widget[] =>
			n > 5 ? [] : [section(`L${n}`, {}, ...chain(n + 1))];
		await openEditor(chain(1));
		expect(
			within(frame("L5")).getByRole("button", { name: "Add widget" }),
		).toBeVisible();
		expect(
			within(frame("L5")).queryByRole("button", { name: "Add section" }),
		).toBeNull();
	});
});

describe("adding, editing and deleting widgets", () => {
	it("adds a widget at the end of its section through the picker", async () => {
		const { user } = await openEditor([
			section("Work", { width: 4 }, section("Docs", { width: 2 })),
		]);
		// Docs has its own button; Work's comes after it.
		await user.click(
			within(frame("Work")).getAllByRole("button", { name: "Add widget" })[1],
		);
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(picker).getByRole("button", { name: /Text widget/ }),
		);

		const dialog = await screen.findByRole("dialog", { name: "New text" });
		expect(within(dialog).getByLabelText("Width")).toHaveValue("4");
		expect(within(dialog).getByLabelText("Fit the content")).toBeChecked();
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		await user.type(within(dialog).getByLabelText("Text"), "Hello");
		await user.click(add);

		expect(await screen.findByText("Hello (editing)")).toBeVisible();
		const children = [
			...frame("Work").querySelectorAll(
				"[data-testid^='section-'], [data-testid^='widget-']",
			),
		].map((el) => el.getAttribute("data-testid")?.split("-")[0]);
		expect(children).toEqual(["section", "widget"]);
	});

	it("groups the kinds in the picker", async () => {
		const { user } = await openEditor([section("Work")]);
		await user.click(screen.getByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		const groups = within(picker).getAllByRole("region");
		expect(groups.map((g) => g.getAttribute("aria-labelledby"))).toEqual([
			"widget-group-favourites",
			"widget-group-notes",
		]);
		const names = (group: HTMLElement) =>
			within(group)
				.getAllByRole("button")
				.map((b) => b.querySelector(".font-medium")?.textContent);
		expect(
			names(within(picker).getByRole("region", { name: "Favourites" })),
		).toEqual([
			"Favourite explorer",
			"Link",
			"Link list",
			"Recent favourites",
			"Text widget",
		]);
		expect(
			names(within(picker).getByRole("region", { name: "Notes" })),
		).toEqual(["Note", "Recent notes", "Note list"]);
	});

	it("shows the picker's group headings in German", async () => {
		const { user } = await openEditor([section("Work")]);
		await act(() => i18n.changeLanguage("de"));
		await user.click(screen.getByRole("button", { name: "Widget hinzufügen" }));
		const picker = await screen.findByRole("dialog", {
			name: "Widget hinzufügen",
		});
		expect(
			within(picker)
				.getAllByRole("heading", { level: 3 })
				.map((h) => h.textContent),
		).toEqual(["Favoriten", "Notizen"]);
	});

	it("adds nothing when the settings are cancelled", async () => {
		const { user } = await openEditor([section("Work")]);
		await user.click(screen.getByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(picker).getByRole("button", { name: /Text widget/ }),
		);
		const dialog = await screen.findByRole("dialog", { name: "New text" });
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(screen.queryByTestId(/^widget-/)).toBeNull();
		// Nothing changed, so leaving needs no confirmation.
		await user.click(screen.getByRole("button", { name: "Cancel editing" }));
		expect(screen.queryByRole("alertdialog")).toBeNull();
	});

	it("edits a widget's settings and height", async () => {
		const { user } = await openEditor([section("Work", {}, textWidget("n1"))]);
		await user.click(screen.getByRole("button", { name: "Edit Text widget" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit text" });
		const text = within(dialog).getByLabelText("Text");
		await user.clear(text);
		await user.type(text, "Changed");
		await user.click(within(dialog).getByLabelText("Fixed"));
		const rows = within(dialog).getByRole("spinbutton", { name: "Rows" });
		await user.clear(rows);
		await user.type(rows, "3");
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));

		expect(await screen.findByText("Changed (editing)")).toBeVisible();
		expect(widget("n1").style.height).toBe(
			"calc(3 * var(--dashboard-row) + 2 * var(--dashboard-gap))",
		);
	});

	it("deletes a widget after confirming", async () => {
		const { user } = await openEditor([section("Work", {}, textWidget("n1"))]);
		await user.click(screen.getByRole("button", { name: "Edit Text widget" }));
		await user.click(
			await screen.findByRole("button", { name: "Delete widget" }),
		);
		const confirm = await screen.findByRole("alertdialog", {
			name: "Delete widget?",
		});
		expect(confirm).toHaveTextContent("“Text widget” will be deleted.");
		await user.click(within(confirm).getByRole("button", { name: "Delete" }));

		await waitFor(() => expect(screen.queryByTestId("widget-n1")).toBeNull());
		expect(frame("Work")).toBeVisible();
	});
});

describe("moving and saving widgets", () => {
	it("moves a widget into another section with the keyboard", async () => {
		const { user } = await openEditor([
			section("Work", {}, textWidget("n1")),
			section("Home"),
		]);
		screen.getByRole("button", { name: "Move Text widget" }).focus();
		// The only places are inside Work and inside Home, none at the top level.
		await user.keyboard(" {ArrowDown} ");
		await waitFor(() =>
			expect(within(frame("Home")).getByTestId("widget-n1")).toBeVisible(),
		);
		await user.keyboard(" {ArrowDown}{ArrowDown} ");
		expect(within(frame("Home")).getByTestId("widget-n1")).toBeVisible();
	});

	it("saves widgets with kind and config, without resolved data", async () => {
		const { api, user, router } = await openEditor(
			[section("Work", {}, textWidget("n1", { data: "resolved" }))],
			{
				"PUT /api/dashboards/d1/layout": ({ body }: { body: unknown }) => ({
					body: {
						version: 4,
						sections: (body as { sections: Widget[] }).sections,
					},
				}),
			},
		);
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));

		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		const sent = (call.body as { sections: Widget[] }).sections;
		expect(sent[0]).toMatchObject({
			kind: "section",
			children: [{ id: "n1", kind: "text", config: { text: "n1" } }],
		});
		expect(JSON.stringify(sent)).not.toContain("resolved");
	});
});
