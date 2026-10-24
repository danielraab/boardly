import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { mockApi } from "#/test/api";
import { type FavouriteNode, favouritesQueryKey } from "./api";
import { DetailsButton, FavouriteDetailsDialog } from "./FavouriteDetails";

const go: FavouriteNode = {
	id: "go",
	kind: "link",
	title: "Go tour",
	url: "https://go.dev/tour",
	tags: ["go", "learning"],
	iconImageId: "icon1",
	description: "A tour of Go",
	previewImageId: "preview1",
};
const bare: FavouriteNode = {
	id: "bare",
	kind: "link",
	title: null,
	url: "https://bare.example.org/",
};

function treeOf(...children: FavouriteNode[]): FavouriteNode {
	return { id: "root", kind: "folder", title: null, children };
}

function renderDialog(id: string, links: FavouriteNode[] = [go, bare]) {
	let tree = treeOf(...links);
	const api = mockApi({
		"GET /api/favourites": () => ({ body: tree }),
		"GET /api/tags": { body: [] },
		"PATCH /api/favourites/go": ({ body }) => {
			const node = { ...go, ...(body as object) };
			tree = treeOf(node, bare);
			return { body: node };
		},
	});
	const client = new QueryClient();
	client.setQueryData(favouritesQueryKey, tree);
	const onClose = vi.fn();
	const onSaved = vi.fn();
	render(
		<QueryClientProvider client={client}>
			<FavouriteDetailsDialog
				favouriteId={id}
				onClose={onClose}
				onSaved={onSaved}
			/>
		</QueryClientProvider>,
	);
	return { api, onClose, onSaved, user: userEvent.setup() };
}

describe("DetailsButton", () => {
	it("is named after the link, or its URL", async () => {
		const user = userEvent.setup();
		const onClick = vi.fn();
		const { rerender } = render(
			<DetailsButton
				link={{ title: "Mail", url: "https://m" }}
				onClick={onClick}
			/>,
		);
		await user.click(screen.getByRole("button", { name: "Details for Mail" }));
		expect(onClick).toHaveBeenCalledOnce();
		rerender(
			<DetailsButton
				link={{ title: null, url: "https://m" }}
				onClick={onClick}
			/>,
		);
		expect(
			screen.getByRole("button", { name: "Details for https://m" }),
		).toBeVisible();
	});
});

describe("FavouriteDetailsDialog", () => {
	it("shows all details of a link", () => {
		renderDialog("go");
		const dialog = screen.getByRole("dialog", { name: "Go tour" });
		expect(
			within(dialog).getByRole("img", { name: "Preview image" }),
		).toHaveAttribute("src", "/api/images/preview1");
		const url = within(dialog).getByRole("link", {
			name: "https://go.dev/tour",
		});
		expect(url).toHaveAttribute("href", "https://go.dev/tour");
		expect(url).toHaveAttribute("target", "_blank");
		expect(url.querySelector("img")).toBeNull();
		// The website icon comes before the title.
		const title = within(dialog).getByRole("heading", { name: "Go tour" });
		expect(title.previousElementSibling).toHaveAttribute(
			"src",
			"/api/images/icon1",
		);
		expect(within(dialog).getByText("A tour of Go")).toBeVisible();
		const tags = within(dialog).getByRole("list", { name: "Tags" });
		expect(
			within(tags)
				.getAllByRole("listitem")
				.map((li) => li.textContent),
		).toEqual(["go", "learning"]);
		const open = within(dialog).getByRole("link", { name: "Open" });
		expect(open).toHaveAttribute("href", "https://go.dev/tour");
		expect(open).toHaveAttribute("target", "_blank");
		expect(open).toHaveAttribute("rel", "noopener noreferrer");
	});

	it("says there is no description, and leaves out preview and tags", () => {
		renderDialog("bare");
		const dialog = screen.getByRole("dialog", {
			name: "https://bare.example.org/",
		});
		expect(within(dialog).getByText("No description")).toHaveClass(
			"text-muted",
		);
		expect(
			within(dialog).queryByRole("img", { name: "Preview image" }),
		).toBeNull();
		expect(within(dialog).queryByRole("list", { name: "Tags" })).toBeNull();
	});

	it("closes with Escape and the close button", async () => {
		const { onClose, user } = renderDialog("go");
		await user.keyboard("{Escape}");
		expect(onClose).toHaveBeenCalledOnce();
		await user.click(screen.getByRole("button", { name: "Close" }));
		expect(onClose).toHaveBeenCalledTimes(2);
	});

	it("is closed for a missing link", () => {
		renderDialog("gone");
		expect(screen.queryByRole("dialog")).toBeNull();
	});

	it("edits the link and shows the saved values", async () => {
		const { api, onSaved, user } = renderDialog("go");
		await user.click(screen.getByRole("button", { name: "Edit" }));
		const edit = await screen.findByRole("dialog", { name: "Edit link" });
		const description = within(edit).getByLabelText("Description");
		await user.clear(description);
		await user.type(description, "Learn Go");
		await user.click(within(edit).getByRole("button", { name: "Save" }));

		const details = await screen.findByRole("dialog", { name: "Go tour" });
		await waitFor(() =>
			expect(within(details).getByText("Learn Go")).toBeVisible(),
		);
		expect(api.callsTo("PATCH", "/api/favourites/go")[0].body).toMatchObject({
			description: "Learn Go",
		});
		expect(onSaved).toHaveBeenCalledOnce();
	});

	it("returns to the unchanged details on cancel", async () => {
		const { api, user } = renderDialog("go");
		await user.click(screen.getByRole("button", { name: "Edit" }));
		const edit = await screen.findByRole("dialog", { name: "Edit link" });
		await user.type(
			within(edit).getByLabelText("Title (optional)"),
			" changed",
		);
		await user.click(within(edit).getByRole("button", { name: "Cancel" }));

		const details = await screen.findByRole("dialog", { name: "Go tour" });
		expect(within(details).getByText("A tour of Go")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/favourites/go")).toHaveLength(0);
	});
});
