import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{ id: "work", kind: "folder", title: "Work", children: [] },
		{ id: "home", kind: "folder", title: "Home", children: [] },
	],
};

function mockShareApi(shareFolderId: string | null = null) {
	return mockApi({
		"GET /api/me": signedIn({ shareFolderId }),
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

	it("shows the title before the URL and the background hint", async () => {
		mockShareApi();
		renderApp("/share");
		const title = await screen.findByLabelText("Title (optional)");
		const [first, second] = screen.getAllByRole("textbox");
		expect(first).toBe(title);
		expect(second).toBe(screen.getByLabelText("URL"));
		expect(
			screen.getByText(
				"The description and preview image are loaded from the page in the background. You can edit them afterwards.",
			),
		).toBeVisible();
	});

	it("saves into the chosen folder with tags", async () => {
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
	});

	it("preselects the default folder", async () => {
		mockShareApi("home");
		renderApp("/share");
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("doesn't remember another folder over the default", async () => {
		mockShareApi("home");
		const { user, router } = renderApp(share({ url: "https://example.org" }));
		await user.click(await folderRadio("Work"));
		await user.click(screen.getByRole("button", { name: "Save" }));
		await screen.findByRole("heading", { name: "Link saved" });

		await router.navigate({ to: "/" });
		await router.navigate({ to: "/share" });
		expect(await folderRadio("Home")).toBeChecked();
	});

	it("falls back to the root when the default folder is gone", async () => {
		mockShareApi("deleted");
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
