import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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
	skipped: { duplicates: 1, invalidUrls: 0, notLinks: 2, smartLists: 2 },
	shortenedDescriptions: 0,
};

async function openTab() {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/favourites": { body: tree },
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
		expect(result).toHaveTextContent("2 entries aren't links");
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
			name: "Skip links that are already saved",
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
