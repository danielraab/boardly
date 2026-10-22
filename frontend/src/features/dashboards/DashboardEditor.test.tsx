import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import { type Dashboard, defaultColumns, type Section } from "./api";

const frame = (id: string) => screen.getByTestId(`section-${id}`);

/** Ann signed in, looking at her dashboard "Home" with these sections. */
function owner(sections: Section[], more = {}) {
	return mockApi({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections) },
		...more,
	});
}

/** Opens the dashboard actions menu and picks "Edit layout". */
async function editLayout(user: ReturnType<typeof renderApp>["user"]) {
	await user.click(
		await screen.findByRole("button", { name: "Dashboard actions" }),
	);
	await user.click(
		await screen.findByRole("menuitem", { name: "Edit layout" }),
	);
}

async function openEditor(sections: Section[], more = {}) {
	const api = owner(sections, more);
	const app = renderApp("/ann/home?edit=true");
	await screen.findByRole("button", { name: "Save and exit" });
	return { api, ...app };
}

describe("edit mode", () => {
	it("lets the owner enter edit mode", async () => {
		owner([section("Links")]);
		const { user, router } = renderApp("/ann/home");
		await editLayout(user);

		expect(router.state.location.search).toEqual({ edit: true });
		expect(
			await screen.findByRole("button", { name: "Save and exit" }),
		).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Cancel editing" }),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Dashboard actions" }),
		).toBeNull();
	});

	it("ignores the parameter for visitors", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([section("Links")], { isOwner: false, id: undefined }),
			},
		});
		renderApp("/ann/home?edit=true");
		expect(
			await screen.findByRole("heading", { level: 2, name: "Links" }),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Dashboard actions" }),
		).toBeNull();
		expect(screen.queryByRole("button", { name: "Save and exit" })).toBeNull();
		expect(screen.queryByRole("button", { name: /Move/ })).toBeNull();
	});

	it("offers editing on / for the default dashboard", async () => {
		const home: Dashboard = {
			id: "d1",
			slug: "home",
			title: "Home",
			isDefault: true,
			isPublic: false,
			...defaultColumns,
		};
		owner([], { "GET /api/dashboards": { body: [home] } });
		const { user, router } = renderApp("/");
		await editLayout(user);
		expect(
			await screen.findByRole("button", { name: "Save and exit" }),
		).toBeVisible();
		expect(router.state.location.pathname).toBe("/");
		expect(router.state.location.search).toEqual({ edit: true });
	});
});

describe("editing controls", () => {
	it("shows the controls on every section, also a borderless one", async () => {
		await openEditor([section("Plain", { border: false })]);
		expect(frame("Plain")).toHaveClass("outline-dashed");
		expect(screen.getByRole("button", { name: "Move Plain" })).toBeVisible();
		expect(screen.getByRole("button", { name: "Edit Plain" })).toBeVisible();
		expect(
			within(frame("Plain")).getByRole("button", { name: "Add section" }),
		).toBeVisible();
		expect(screen.getAllByRole("button", { name: "Add section" })).toHaveLength(
			2,
		);
	});

	it("keeps the normal border of a section that has one", async () => {
		await openEditor([section("Boxed")]);
		expect(frame("Boxed")).toHaveClass("border");
		expect(frame("Boxed")).not.toHaveClass("outline-dashed");
	});

	it("has no add button inside a section at level 5", async () => {
		const chain = (n: number): Section[] =>
			n > 5 ? [] : [section(`L${n}`, {}, ...chain(n + 1))];
		await openEditor(chain(1));
		// The top level and levels 1 to 4.
		expect(screen.getAllByRole("button", { name: "Add section" })).toHaveLength(
			5,
		);
		expect(
			within(frame("L5")).queryByRole("button", { name: "Add section" }),
		).toBeNull();
	});

	it("starts an empty dashboard with only the top-level add button", async () => {
		await openEditor([]);
		expect(screen.getAllByRole("button", { name: "Add section" })).toHaveLength(
			1,
		);
	});
});

describe("section dialog", () => {
	it("adds a nested section at the end of its parent", async () => {
		const { user } = await openEditor([
			section("Work", {}, section("Docs", { width: 3 })),
		]);
		await user.click(
			within(frame("Work")).getAllByRole("button", { name: "Add section" })[1],
		);
		const dialog = await screen.findByRole("dialog", { name: "New section" });
		expect(within(dialog).getByLabelText("Width")).toHaveValue("6");
		expect(within(dialog).getByLabelText("Border")).toBeChecked();
		expect(within(dialog).getByLabelText("Fit the content")).toBeChecked();
		await user.type(within(dialog).getByLabelText("Title"), "Tools");
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));

		const tools = await within(frame("Work")).findByRole("heading", {
			name: "Tools",
		});
		const headings = within(frame("Work"))
			.getAllByRole("heading")
			.map((h) => h.textContent);
		expect(headings).toEqual(["Work", "Docs", "Tools"]);
		expect(tools).toBeVisible();
	});

	it("edits border and height", async () => {
		const { user } = await openEditor([section("Links")]);
		await user.click(screen.getByRole("button", { name: "Edit Links" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit section" });
		await user.click(within(dialog).getByLabelText("Border"));
		await user.click(within(dialog).getByLabelText("Fixed"));
		const rows = within(dialog).getByRole("spinbutton", { name: "Rows" });
		await user.clear(rows);
		await user.type(rows, "4");
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));

		await waitFor(() => expect(frame("Links")).not.toHaveClass("border"));
		expect(frame("Links").style.height).toBe(
			"calc(4 * var(--dashboard-row) + 3 * var(--dashboard-gap))",
		);
	});

	it("deletes a section with everything inside after confirming", async () => {
		const { user } = await openEditor([
			section("Work", {}, section("A"), section("B")),
			section("Links"),
		]);
		await user.click(screen.getByRole("button", { name: "Edit Work" }));
		await user.click(
			await screen.findByRole("button", { name: "Delete section" }),
		);
		const confirm = await screen.findByRole("alertdialog");
		expect(confirm).toHaveTextContent(
			"“Work” and the 2 widgets inside it will be deleted.",
		);
		await user.click(within(confirm).getByRole("button", { name: "Delete" }));

		await waitFor(() =>
			expect(screen.queryByTestId("section-Work")).toBeNull(),
		);
		expect(screen.queryByTestId("section-A")).toBeNull();
		expect(frame("Links")).toBeVisible();
	});

	it("keeps the section when deleting is cancelled", async () => {
		const { user } = await openEditor([section("Work")]);
		await user.click(screen.getByRole("button", { name: "Edit Work" }));
		await user.click(
			await screen.findByRole("button", { name: "Delete section" }),
		);
		const confirm = await screen.findByRole("alertdialog");
		expect(confirm).toHaveTextContent("“Work” will be deleted.");
		await user.click(within(confirm).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(frame("Work")).toBeVisible();
	});
});

async function addTopLevel(
	user: ReturnType<typeof renderApp>["user"],
	title: string,
) {
	const buttons = screen.getAllByRole("button", { name: "Add section" });
	await user.click(buttons[buttons.length - 1]);
	const dialog = await screen.findByRole("dialog", { name: "New section" });
	await user.type(within(dialog).getByLabelText("Title"), title);
	await user.click(within(dialog).getByRole("button", { name: "Apply" }));
	await screen.findByRole("heading", { name: title });
}

describe("saving", () => {
	it("saves the draft and leaves edit mode", async () => {
		const { api, user, router } = await openEditor([section("Links")], {
			"PUT /api/dashboards/d1/layout": ({ body }: { body: unknown }) => ({
				body: {
					version: 4,
					sections: (body as { sections: Section[] }).sections,
				},
			}),
		});
		await addTopLevel(user, "News");
		await user.click(screen.getByRole("button", { name: "Save and exit" }));

		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		const body = call.body as { version: number; sections: Section[] };
		expect(body.version).toBe(3);
		expect(body.sections.map((s) => s.title)).toEqual(["Links", "News"]);
		expect(
			await screen.findByRole("heading", { level: 2, name: "News" }),
		).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Dashboard actions" }),
		).toBeVisible();
	});

	it("keeps the draft when saving fails", async () => {
		const { user, router } = await openEditor([section("Links")], {
			"PUT /api/dashboards/d1/layout": apiError(500, "internal"),
		});
		await addTopLevel(user, "News");
		await user.click(screen.getByRole("button", { name: "Save and exit" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The dashboard couldn't be saved.",
		);
		expect(router.state.location.search).toEqual({ edit: true });
		expect(screen.getByRole("heading", { name: "News" })).toBeVisible();
	});

	it("reports a conflict and reloads the current layout", async () => {
		const { api, user } = await openEditor([section("Links")], {
			"PUT /api/dashboards/d1/layout": apiError(409, "layout_conflict"),
		});
		await addTopLevel(user, "News");
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		const alert = await screen.findByRole("alert");
		expect(alert).toHaveTextContent("This dashboard was changed elsewhere.");

		api.set({
			"GET /api/u/ann/home": {
				body: viewData([section("Theirs")], { layoutVersion: 5 }),
			},
		});
		await user.click(within(alert).getByRole("button", { name: "Reload" }));
		expect(
			await screen.findByRole("heading", { name: "Theirs" }),
		).toBeVisible();
		expect(screen.queryByRole("heading", { name: "News" })).toBeNull();
		expect(screen.getByRole("button", { name: "Save and exit" })).toBeVisible();
		expect(screen.queryByRole("alert")).toBeNull();
	});
});

describe("unsaved changes", () => {
	it("leaves at once without changes", async () => {
		const { user, router } = await openEditor([section("Links")]);
		await user.click(screen.getByRole("button", { name: "Cancel editing" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		expect(screen.queryByRole("alertdialog")).toBeNull();
	});

	it("asks before discarding changes", async () => {
		const { user, router } = await openEditor([section("Links")]);
		await addTopLevel(user, "News");

		await user.click(screen.getByRole("button", { name: "Cancel editing" }));
		let confirm = await screen.findByRole("alertdialog", {
			name: "Discard changes?",
		});
		await user.click(
			within(confirm).getByRole("button", { name: "Keep editing" }),
		);
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(router.state.location.search).toEqual({ edit: true });
		expect(screen.getByRole("heading", { name: "News" })).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Cancel editing" }));
		confirm = await screen.findByRole("alertdialog", {
			name: "Discard changes?",
		});
		await user.click(within(confirm).getByRole("button", { name: "Discard" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		expect(screen.queryByRole("heading", { name: "News" })).toBeNull();
		expect(screen.getByRole("heading", { name: "Links" })).toBeVisible();
	});

	it("asks before switching to another dashboard", async () => {
		const entries = [
			{ slug: "home", title: "Home" },
			{ slug: "work", title: "Work" },
		];
		const { user, router } = await openEditor([section("Links")], {
			"GET /api/u/ann/home": {
				body: viewData([section("Links")], { dashboards: entries }),
			},
			"GET /api/u/ann/work": {
				body: viewData([], {
					id: "d2",
					dashboard: entries[1],
					dashboards: entries,
				}),
			},
		});
		await addTopLevel(user, "News");

		await user.click(
			screen.getByRole("button", { name: "Switch dashboard: Home" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Work" }));
		const confirm = await screen.findByRole("alertdialog", {
			name: "Discard changes?",
		});
		expect(router.state.location.pathname).toBe("/ann/home");

		await user.click(within(confirm).getByRole("button", { name: "Discard" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/ann/work"),
		);
		expect(
			await screen.findByRole("heading", { level: 1, name: "Work" }),
		).toBeVisible();
	});
});

describe("moving sections", () => {
	const handle = (title: string) =>
		screen.getByRole("button", { name: `Move ${title}` });
	const topLevel = () =>
		[...document.querySelectorAll("[data-testid^='section-']")]
			.filter(
				(el) => el.parentElement?.closest("[data-testid^='section-']") === null,
			)
			.map((el) => el.getAttribute("data-testid")?.slice("section-".length));

	it("reorders top-level sections with the keyboard", async () => {
		const { user } = await openEditor([section("A"), section("B")]);
		handle("B").focus();
		// From the end: inside A, then before A.
		await user.keyboard(" {ArrowUp}{ArrowUp} ");
		expect(topLevel()).toEqual(["B", "A"]);
		expect(handle("B")).toHaveFocus();
	});

	it("moves a section with its children into an empty section", async () => {
		const { user } = await openEditor([
			section("X", {}, section("Y")),
			section("Empty"),
		]);
		handle("X").focus();
		await user.keyboard(" ");
		expect(screen.getByText("Picked up X.")).toBeInTheDocument();
		await user.keyboard("{ArrowDown}");
		expect(screen.getByText("Position 1 in Empty.")).toBeInTheDocument();
		await user.keyboard(" ");

		expect(topLevel()).toEqual(["Empty"]);
		expect(within(frame("Empty")).getByTestId("section-X")).toBeVisible();
		expect(within(frame("X")).getByTestId("section-Y")).toBeVisible();
		expect(screen.getByText("X moved.")).toBeInTheDocument();
	});

	it("puts a section back when the move is cancelled", async () => {
		const { user } = await openEditor([section("A"), section("B")]);
		handle("B").focus();
		await user.keyboard(" {ArrowUp}{ArrowUp}{Escape}");
		expect(topLevel()).toEqual(["A", "B"]);
		expect(screen.getByText("Moving B cancelled.")).toBeInTheDocument();
	});

	it("blocks a move that nests too deep", async () => {
		const { user } = await openEditor([
			section("L1", {}, section("L2", {}, section("L3", {}, section("L4")))),
			section("M", {}, section("M2", {}, section("M3"))),
		]);
		handle("M").focus();
		// From the end: end of L1, end of L2, end of L3 (too deep), inside L4 (too deep).
		await user.keyboard(" {ArrowUp}{ArrowUp}");
		expect(within(frame("L2")).getByTestId("section-M")).toBeVisible();
		await user.keyboard("{ArrowUp}{ArrowUp}");
		expect(
			screen.getByText(
				"Position 1 in L4. Not possible: sections can be nested at most 5 levels deep.",
			),
		).toBeInTheDocument();
		await user.keyboard(" ");

		expect(topLevel()).toEqual(["L1", "M"]);
		expect(within(frame("L4")).queryByTestId("section-M")).toBeNull();
		expect(
			screen.getAllByText("Sections can be nested at most 5 levels deep.")
				.length,
		).toBeGreaterThan(0);
	});
});

describe("dashboard settings", () => {
	/**
	 * Ann's "Home" on the server: the list, its view at its current slug
	 * (`home` or `start`) and PATCH, which changes it.
	 */
	function settingsApi(fields: Partial<Dashboard> = {}) {
		let saved: Dashboard = {
			id: "d1",
			slug: "home",
			title: "Home",
			isDefault: false,
			isPublic: false,
			...defaultColumns,
			...fields,
		};
		const view = (slug: string) => () =>
			saved.slug === slug
				? {
						body: viewData([section("Links")], {
							dashboard: { slug: saved.slug, title: saved.title },
							isDefault: saved.isDefault,
							isPublic: saved.isPublic,
						}),
					}
				: apiError(404, "not_found");
		return mockApi({
			"GET /api/me": signedIn(),
			"GET /api/dashboards": () => ({ body: [saved] }),
			"GET /api/u/ann/home": view("home"),
			"GET /api/u/ann/start": view("start"),
			"PATCH /api/dashboards/d1": ({ body }) => {
				saved = { ...saved, ...(body as Partial<Dashboard>) };
				return { body: saved };
			},
		});
	}

	async function openSettings(user: ReturnType<typeof renderApp>["user"]) {
		await user.click(
			await screen.findByRole("button", { name: "Dashboard actions" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Settings" }));
		return screen.findByRole("dialog", { name: "Edit dashboard" });
	}

	async function changeSlug(
		user: ReturnType<typeof renderApp>["user"],
		dialog: HTMLElement,
	) {
		const slug = within(dialog).getByLabelText("Address");
		await user.clear(slug);
		await user.type(slug, "start");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		const confirm = await screen.findByRole("alertdialog");
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	}

	it("opens the dialog with the dashboard's values", async () => {
		settingsApi({ isDefault: true, isPublic: true });
		const { user } = renderApp("/ann/home");
		const dialog = await openSettings(user);
		expect(within(dialog).getByLabelText("Title")).toHaveValue("Home");
		expect(within(dialog).getByLabelText("Address")).toHaveValue("home");
		expect(within(dialog).getByLabelText("Default dashboard")).toBeChecked();
		expect(within(dialog).getByLabelText("Public")).toBeChecked();
	});

	it("stays on the page after a title change", async () => {
		const api = settingsApi();
		const { user, router } = renderApp("/ann/home");
		const dialog = await openSettings(user);
		const title = within(dialog).getByLabelText("Title");
		await user.clear(title);
		await user.type(title, "Start");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		expect(
			await screen.findByRole("heading", { level: 1, name: "Start" }),
		).toBeVisible();
		expect(router.state.location.pathname).toBe("/ann/home");
		expect(api.callsTo("PATCH", "/api/dashboards/d1")[0].body).toMatchObject({
			title: "Start",
		});
	});

	it("follows a changed slug to the new address", async () => {
		settingsApi();
		const { user, router } = renderApp("/ann/home");
		await changeSlug(user, await openSettings(user));

		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/ann/start"),
		);
		expect(
			await screen.findByRole("heading", { level: 2, name: "Links" }),
		).toBeVisible();
	});

	it("leaves / for the dashboard's address when the default is removed", async () => {
		settingsApi({ isDefault: true });
		const { user, router } = renderApp("/");
		const dialog = await openSettings(user);
		await user.click(within(dialog).getByLabelText("Default dashboard"));
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/ann/home"),
		);
		expect(
			await screen.findByRole("heading", { level: 1, name: "Home" }),
		).toBeVisible();
	});

	it("stays on / when only the slug of the default changes", async () => {
		settingsApi({ isDefault: true });
		const { user, router } = renderApp("/");
		await changeSlug(user, await openSettings(user));

		expect(
			await screen.findByRole("heading", { level: 2, name: "Links" }),
		).toBeVisible();
		expect(router.state.location.pathname).toBe("/");
	});
});
