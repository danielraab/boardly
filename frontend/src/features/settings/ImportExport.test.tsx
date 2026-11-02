import { screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import type { ImportReport } from "./import/api";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [{ id: "reading", kind: "folder", title: "Reading" }],
};

const report: ImportReport = {
	links: 165,
	folders: 17,
	notes: 0,
	categories: 0,
	skipped: {
		duplicates: 1,
		duplicateNotes: 0,
		invalidUrls: 0,
		notLinks: 2,
		smartLists: 2,
		tooLargeNotes: 0,
	},
	shortenedDescriptions: 0,
};

const categories = [
	{ id: "recipes", title: "Recipes", icon: null, color: null, parentId: null },
];

async function openTab() {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/favourites": { body: tree },
		"GET /api/note-categories": { body: categories },
	});
	const app = renderApp("/settings?tab=data");
	const panel = await screen.findByRole("tabpanel");
	return { api, panel, ...app };
}

const file = () =>
	new File(['{"bookmarks": [], "lists": []}'], "karakeep.json", {
		type: "application/json",
	});

describe("import / export tab", () => {
	it("is the fourth tab and survives a reload", async () => {
		const { unmount } = await openTab();
		expect(
			screen.getByRole("tab", { name: "Import / Export" }),
		).toHaveAttribute("aria-selected", "true");
		unmount();

		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/favourites": { body: tree },
		});
		const { user, router } = renderApp("/settings");
		await user.click(
			await screen.findByRole("tab", { name: "Import / Export" }),
		);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ tab: "data" }),
		);
	});

	it("needs a file and an import folder", async () => {
		const { panel, user } = await openTab();
		const submit = within(panel).getByRole("button", { name: "Import" });
		expect(
			within(panel).getByRole("radio", { name: "Karakeep (JSON)" }),
		).toBeChecked();
		expect(submit).toBeDisabled();

		await user.upload(within(panel).getByLabelText("File"), file());
		expect(submit).toBeDisabled();

		await user.click(
			await within(panel).findByRole("radio", { name: "Reading" }),
		);
		expect(submit).toBeEnabled();
	});

	it("imports the file into the folder and shows the report", async () => {
		const { panel, user, api } = await openTab();
		let sent: { body: unknown; search: URLSearchParams } | null = null;
		api.set({
			"POST /api/favourites/import": ({ init, url }) => {
				sent = { body: init.body, search: url.searchParams };
				return { body: report };
			},
		});
		await user.click(
			within(panel).getByRole("radio", { name: "Linkwarden (JSON)" }),
		);
		await user.upload(within(panel).getByLabelText("File"), file());
		await user.click(
			await within(panel).findByRole("radio", { name: "Favourites" }),
		);
		const reloads = api.callsTo("GET", "/api/favourites").length;
		await user.click(within(panel).getByRole("button", { name: "Import" }));

		const result = await within(panel).findByRole("status");
		expect(result).toHaveTextContent(
			"Imported 165 links and created 17 folders.",
		);
		expect(result).toHaveTextContent("1 link was already in your favourites");
		expect(result).toHaveTextContent("2 entries are neither links nor notes");
		expect(result).not.toHaveTextContent("note was");
		expect(result).toHaveTextContent("2 smart lists were left out");
		expect(result).not.toHaveTextContent("invalid address");
		expect(result).not.toHaveTextContent("shortened");

		expect(sent).not.toBeNull();
		const { body, search } = sent as unknown as {
			body: unknown;
			search: URLSearchParams;
		};
		expect(body).toBeInstanceOf(File);
		expect(search.get("format")).toBe("linkwarden");
		expect(search.get("folderId")).toBe("root");
		expect(search.get("skipDuplicates")).toBe("true");
		expect(search.has("categoryId")).toBe(false);
		await waitFor(() =>
			expect(api.callsTo("GET", "/api/favourites").length).toBeGreaterThan(
				reloads,
			),
		);
	});

	it("can import links that are already saved", async () => {
		const { panel, user, api } = await openTab();
		let search: URLSearchParams | null = null;
		api.set({
			"POST /api/favourites/import": ({ url }) => {
				search = url.searchParams;
				return { body: report };
			},
		});
		const skip = within(panel).getByRole("checkbox", {
			name: "Skip links and notes that are already saved",
		});
		expect(skip).toBeChecked();
		await user.click(skip);
		await user.upload(within(panel).getByLabelText("File"), file());
		await user.click(
			await within(panel).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(panel).getByRole("button", { name: "Import" }));

		await within(panel).findByRole("status");
		expect((search as URLSearchParams | null)?.get("skipDuplicates")).toBe(
			"false",
		);
	});

	it("imports Karakeep text bookmarks into the chosen category", async () => {
		const { panel, user, api } = await openTab();
		let search: URLSearchParams | null = null;
		api.set({
			"POST /api/favourites/import": ({ url }) => {
				search = url.searchParams;
				return {
					body: {
						...report,
						notes: 3,
						categories: 1,
						skipped: { ...report.skipped, duplicateNotes: 2, tooLargeNotes: 1 },
					},
				};
			},
		});
		const category = await within(panel).findByRole("radiogroup", {
			name: "Import notes into",
		});
		expect(
			await within(category).findByRole("radio", { name: "Top level" }),
		).toBeChecked();
		await user.click(within(category).getByRole("radio", { name: "Recipes" }));
		await user.upload(within(panel).getByLabelText("File"), file());
		await user.click(
			await within(panel).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(panel).getByRole("button", { name: "Import" }));

		const result = await within(panel).findByRole("status");
		expect(result).toHaveTextContent(
			"Imported 3 notes and created 1 category.",
		);
		expect(result).toHaveTextContent("2 notes were already saved");
		expect(result).toHaveTextContent("1 note was too large to import");
		expect((search as URLSearchParams | null)?.get("categoryId")).toBe(
			"recipes",
		);
		await waitFor(() =>
			expect(api.callsTo("GET", "/api/note-categories").length).toBeGreaterThan(
				1,
			),
		);
	});

	it("asks for the category only for Karakeep", async () => {
		const { panel, user } = await openTab();
		expect(
			await within(panel).findByRole("radiogroup", {
				name: "Import notes into",
			}),
		).toBeVisible();
		await user.click(
			within(panel).getByRole("radio", { name: "Linkwarden (JSON)" }),
		);
		expect(
			within(panel).queryByRole("radiogroup", { name: "Import notes into" }),
		).toBeNull();
	});

	it("explains an invalid file", async () => {
		const { panel, user, api } = await openTab();
		api.set({
			"POST /api/favourites/import": apiError(400, "invalid_import_file"),
		});
		await user.upload(within(panel).getByLabelText("File"), file());
		await user.click(
			await within(panel).findByRole("radio", { name: "Reading" }),
		);
		await user.click(within(panel).getByRole("button", { name: "Import" }));
		expect(await within(panel).findByRole("alert")).toHaveTextContent(
			"This file isn't a valid Karakeep (JSON) export.",
		);
		expect(within(panel).queryByRole("status")).toBeNull();
	});
});

describe("export section", () => {
	const createObjectURL = vi.fn(() => "blob:export");
	const clicks: string[] = [];

	beforeEach(() => {
		clicks.length = 0;
		// jsdom has no object URLs.
		URL.createObjectURL = createObjectURL;
		URL.revokeObjectURL = vi.fn();
		vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (
			this: HTMLAnchorElement,
		) {
			clicks.push(this.download);
		});
	});
	afterEach(() => {
		vi.restoreAllMocks();
		vi.useRealTimers();
	});

	function exportRoute(fileName: string) {
		return {
			body: { format: "boardly" },
			headers: { "Content-Disposition": `attachment; filename=${fileName}` },
		};
	}

	it("exports without files by default and saves the JSON file", async () => {
		const { api, panel, user } = await openTab();
		api.set({
			"GET /api/me/export": exportRoute("boardly-export-ann-2026-11-02.json"),
		});
		expect(
			within(panel).getByRole("checkbox", {
				name: "Include files (images and attachments)",
			}),
		).not.toBeChecked();

		await user.click(
			within(panel).getByRole("button", { name: "Export everything" }),
		);
		await waitFor(() =>
			expect(clicks).toEqual(["boardly-export-ann-2026-11-02.json"]),
		);
		expect(api.callsTo("GET", "/api/me/export")[0].search).toBe("?files=false");
	});

	it("exports with files when checked", async () => {
		const { api, panel, user } = await openTab();
		api.set({
			"GET /api/me/export": exportRoute("boardly-export-ann-2026-11-02.zip"),
		});
		await user.click(
			within(panel).getByRole("checkbox", {
				name: "Include files (images and attachments)",
			}),
		);
		await user.click(
			within(panel).getByRole("button", { name: "Export everything" }),
		);
		await waitFor(() =>
			expect(clicks).toEqual(["boardly-export-ann-2026-11-02.zip"]),
		);
		expect(api.callsTo("GET", "/api/me/export")[0].search).toBe("?files=true");
	});

	it("disables the button while the export runs", async () => {
		const { api, panel, user } = await openTab();
		let finish: (r: ReturnType<typeof exportRoute>) => void = () => {};
		api.set({
			"GET /api/me/export": () =>
				new Promise((resolve) => {
					finish = resolve;
				}),
		});
		await user.click(
			within(panel).getByRole("button", { name: "Export everything" }),
		);
		expect(
			await within(panel).findByRole("button", { name: "Exporting…" }),
		).toBeDisabled();
		finish(exportRoute("x.json"));
		expect(
			await within(panel).findByRole("button", { name: "Export everything" }),
		).toBeEnabled();
	});

	it("shows when the next export is possible after the limit", async () => {
		vi.useFakeTimers({ toFake: ["Date"] });
		vi.setSystemTime(new Date(2026, 10, 2, 14, 0));
		const { api, panel, user } = await openTab();
		api.set({
			"GET /api/me/export": apiError(429, "rate_limited", {
				"Retry-After": "1200",
			}),
		});
		await user.click(
			within(panel).getByRole("button", { name: "Export everything" }),
		);
		const time = new Intl.DateTimeFormat("en", { timeStyle: "short" }).format(
			new Date(2026, 10, 2, 14, 20),
		);
		expect(await within(panel).findByRole("alert")).toHaveTextContent(
			`You've reached the export limit. You can export again at ${time}.`,
		);
		expect(clicks).toEqual([]);
	});
});
