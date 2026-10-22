import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { SHARE_FOLDER_STORAGE_KEY } from "./SharePage";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{ id: "work", kind: "folder", title: "Work", children: [] },
		{ id: "home", kind: "folder", title: "Home", children: [] },
	],
};

function mockShareApi() {
	return mockApi({
		"GET /api/me": signedIn(),
		"GET /api/favourites": { body: tree },
		"GET /api/tags": { body: [] },
		"POST /api/favourites": ({ body }) => ({
			status: 201,
			body: { id: "new", kind: "link", ...(body as object) },
		}),
	});
}

const share = (params: Record<string, string>) =>
	`/share?${new URLSearchParams(params)}`;

async function folderRadio(name: string) {
	return screen.findByRole("radio", { name });
}

describe("save link page", () => {
	it("prefills url and title", async () => {
		mockShareApi();
		renderApp(share({ title: "Docs", url: "https://example.org/docs" }));
		expect(await screen.findByLabelText("URL")).toHaveValue(
			"https://example.org/docs",
		);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Docs");
	});

	it("takes the URL out of the shared text", async () => {
		mockShareApi();
		renderApp(share({ text: "Read this https://example.org/a" }));
		expect(await screen.findByLabelText("URL")).toHaveValue(
			"https://example.org/a",
		);
		expect(screen.getByLabelText("Title (optional)")).toHaveValue("Read this");
	});

	it("starts empty in the root folder without a share", async () => {
		mockShareApi();
		renderApp("/share");
		expect(await screen.findByLabelText("URL")).toHaveValue("");
		expect(await folderRadio("Favourites")).toBeChecked();
	});

	it("saves into the chosen folder with tags and remembers the folder", async () => {
		const api = mockShareApi();
		const { user } = renderApp(share({ url: "https://example.org/docs" }));

		await user.click(await folderRadio("Work"));
		await user.type(screen.getByLabelText("Tags"), "docs{Enter}");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(
			await screen.findByRole("heading", { name: "Link saved" }),
		).toBeVisible();
		expect(screen.getByText("It's now in “Work”.")).toBeVisible();
		expect(api.callsTo("POST", "/api/favourites")[0].body).toEqual({
			parentId: "work",
			kind: "link",
			url: "https://example.org/docs",
			title: "",
			tags: ["docs"],
		});
		expect(localStorage.getItem(SHARE_FOLDER_STORAGE_KEY)).toBe("work");
	});

	it("preselects the remembered folder", async () => {
		localStorage.setItem(SHARE_FOLDER_STORAGE_KEY, "home");
		mockShareApi();
		renderApp("/share");
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("falls back to the root when the remembered folder is gone", async () => {
		localStorage.setItem(SHARE_FOLDER_STORAGE_KEY, "deleted");
		mockShareApi();
		renderApp("/share");
		expect(await folderRadio("Favourites")).toBeChecked();
	});

	it("rejects an invalid URL", async () => {
		const api = mockShareApi();
		const { user } = renderApp("/share");
		await user.type(await screen.findByLabelText("URL"), "javascript:alert(1)");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"Enter a web address that starts with http:// or https://.",
		);
		expect(api.callsTo("POST", "/api/favourites")).toHaveLength(0);
	});

	it("offers to save another link with an empty form", async () => {
		mockShareApi();
		const { user, router } = renderApp(share({ url: "https://example.org" }));
		await user.click(await screen.findByRole("button", { name: "Save" }));
		await user.click(
			await screen.findByRole("button", { name: "Save another link" }),
		);

		expect(await screen.findByLabelText("URL")).toHaveValue("");
		await waitFor(() => expect(router.state.location.href).toBe("/share"));
	});

	it("goes home on cancel", async () => {
		mockShareApi();
		const { user, router } = renderApp("/share");
		await user.click(await screen.findByRole("link", { name: "Cancel" }));
		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
	});
});

describe("signed-out share", () => {
	// Continuing after sign-in is covered by the passkey tests.
	it("sends the visitor to the login page with the share as redirect", async () => {
		const path = share({ url: "https://example.org" });
		const { router } = renderApp(path);

		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		expect(router.state.location.search).toEqual({ redirect: path });
	});
});
