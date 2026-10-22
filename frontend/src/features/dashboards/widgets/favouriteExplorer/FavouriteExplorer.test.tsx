import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { FavouriteNode } from "#/features/favourites/api";
import { mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import { type ExplorerData, ExplorerTree } from "./ExplorerTree";
import { storageKey } from "./useExpanded";

const gitlab: FavouriteNode = {
	id: "gitlab",
	kind: "link",
	title: "GitLab",
	url: "https://gitlab.example.org/",
	iconImageId: "img1",
};
const tools: FavouriteNode = {
	id: "tools",
	kind: "folder",
	title: "Tools",
	children: [
		gitlab,
		{ id: "inner", kind: "folder", title: "Inner", children: [] },
	],
};
const mail: FavouriteNode = {
	id: "mail",
	kind: "link",
	title: null,
	url: "https://mail.example.org/inbox",
};
const work: FavouriteNode = {
	id: "work",
	kind: "folder",
	title: "Work",
	children: [tools, mail],
};
const favourites: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		work,
		{
			id: "news",
			kind: "link",
			title: "News",
			url: "https://news.example.org/",
		},
	],
};

function tree(data: ExplorerData, fields: { editing?: boolean } = {}) {
	return render(
		<ExplorerTree
			widgetId="w1"
			size="M"
			data={data}
			editing={fields.editing ?? false}
		/>,
	);
}

describe("ExplorerTree", () => {
	it("shows folders and links, with favicons or a globe", async () => {
		const user = userEvent.setup();
		const { container } = tree({ children: [tools, mail] });

		const toolsButton = screen.getByRole("button", { name: "Tools" });
		expect(toolsButton).toHaveAttribute("aria-expanded", "false");
		expect(screen.queryByText("GitLab")).toBeNull();
		// A link without a title shows its URL and opens in a new tab.
		const mailLink = screen.getByRole("link", {
			name: "https://mail.example.org/inbox",
		});
		expect(mailLink).toHaveAttribute("href", "https://mail.example.org/inbox");
		expect(mailLink).toHaveAttribute("target", "_blank");
		expect(mailLink.querySelector("[data-icon='globe']")).not.toBeNull();

		await user.click(toolsButton);
		expect(toolsButton).toHaveAttribute("aria-expanded", "true");
		const gitlabLink = screen.getByRole("link", { name: "GitLab" });
		expect(gitlabLink.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img1",
		);
		expect(screen.getByText("GitLab")).toHaveClass("truncate");
		expect(container.querySelector(".text-sm")).not.toBeNull();
	});

	it("doesn't link anywhere in edit mode", () => {
		tree({ children: [mail] }, { editing: true });
		expect(screen.queryByRole("link")).toBeNull();
		expect(screen.getByText("https://mail.example.org/inbox")).toBeVisible();
	});

	it("shows an empty and a missing folder", () => {
		const { unmount } = tree({ children: [] });
		expect(screen.getByText("This folder is empty")).toBeVisible();
		expect(screen.queryByRole("button")).toBeNull();
		unmount();
		tree({ missing: true });
		expect(screen.getByText("This folder no longer exists")).toBeVisible();
	});

	it("sizes icons, text and rows", () => {
		const { container } = render(
			<ExplorerTree
				widgetId="w1"
				size="L"
				data={{ children: [mail] }}
				editing={false}
			/>,
		);
		expect(container.firstChild).toHaveClass("text-base");
		expect(screen.getByRole("link")).toHaveClass("py-1.5");
		expect(container.querySelector("[data-icon='globe']")).toHaveClass(
			"size-5",
		);
	});
});

describe("collapse state", () => {
	it("starts collapsed and remembers expanded folders per widget", async () => {
		const user = userEvent.setup();
		const { unmount } = tree({ children: [tools, mail] });
		await user.click(screen.getByRole("button", { name: "Tools" }));
		expect(JSON.parse(localStorage.getItem(storageKey("w1")) ?? "")).toEqual([
			"tools",
		]);
		unmount();

		tree({ children: [tools, mail] });
		expect(screen.getByRole("button", { name: "Tools" })).toHaveAttribute(
			"aria-expanded",
			"true",
		);
		expect(screen.getByRole("link", { name: "GitLab" })).toBeVisible();
	});

	it("expands and collapses all folders at every level", async () => {
		const user = userEvent.setup();
		tree({ children: [tools, mail] });
		await user.click(screen.getByRole("button", { name: "Expand all" }));
		expect(screen.getByRole("button", { name: "Tools" })).toHaveAttribute(
			"aria-expanded",
			"true",
		);
		expect(screen.getByRole("button", { name: "Inner" })).toHaveAttribute(
			"aria-expanded",
			"true",
		);

		await user.click(screen.getByRole("button", { name: "Collapse all" }));
		expect(screen.queryByRole("button", { name: "Inner" })).toBeNull();
		expect(screen.getByRole("button", { name: "Tools" })).toHaveAttribute(
			"aria-expanded",
			"false",
		);
		expect(screen.getByRole("button", { name: "Expand all" })).toBeVisible();
	});

	it("has no expand-all button without folders", () => {
		tree({ children: [mail] });
		expect(screen.queryByRole("button")).toBeNull();
	});

	it("works with blocked storage", async () => {
		const user = userEvent.setup();
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		try {
			tree({ children: [tools] });
			await user.click(screen.getByRole("button", { name: "Tools" }));
			expect(screen.getByRole("link", { name: "GitLab" })).toBeVisible();
		} finally {
			vi.restoreAllMocks();
		}
	});
});

function explorer(
	id: string,
	config: object,
	data?: ExplorerData,
): ContentWidget {
	return {
		id,
		kind: "favourite_explorer",
		width: 2,
		height: "auto",
		config: { rootFolderId: "work", size: "M", showIfPublic: false, ...config },
		data,
	};
}

async function openEditor(sections: Widget[], more = {}) {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections) },
		"GET /api/favourites": { body: favourites },
		...more,
	});
	const app = renderApp("/ann/home?edit=true");
	await screen.findByRole("button", { name: "Save and exit" });
	return { api, ...app };
}

describe("the favourite explorer on a dashboard", () => {
	it("shows the server's tree in view mode", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[section("Work", {}, explorer("e1", {}, { children: [mail] }))],
					{ isOwner: false, id: undefined },
				),
			},
		});
		renderApp("/ann/home");
		expect(
			await screen.findByRole("link", {
				name: "https://mail.example.org/inbox",
			}),
		).toBeVisible();
	});

	it("is added through the picker and needs a folder", async () => {
		const { api, user, router } = await openEditor([section("Work")], {
			"PUT /api/dashboards/d1/layout": ({ body }: { body: unknown }) => ({
				body: {
					version: 4,
					sections: (body as { sections: Widget[] }).sections,
				},
			}),
		});
		await user.click(screen.getByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(picker).getByRole("button", { name: /Favourite explorer/ }),
		);

		const dialog = await screen.findByRole("dialog", {
			name: "Add favourite explorer",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();
		const folder = within(dialog).getByLabelText("Folder");
		await waitFor(() => expect(folder).toBeEnabled());
		expect(
			within(folder)
				.getAllByRole("option")
				.map((o) => o.textContent),
		).toEqual([
			"Choose a folder",
			"Favourites",
			"Favourites / Work",
			"Favourites / Work / Tools",
			"Favourites / Work / Tools / Inner",
		]);
		await user.selectOptions(folder, "Favourites / Work");
		await user.selectOptions(within(dialog).getByLabelText("Size"), "Large");
		await user.click(add);

		// The draft shows the folder from the favourites.
		const toolsButton = await screen.findByRole("button", { name: "Tools" });
		expect(toolsButton).toHaveClass("py-1.5");
		expect(screen.getByText("https://mail.example.org/inbox")).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "favourite_explorer",
					config: { rootFolderId: "work", size: "L", showIfPublic: false },
				},
			],
		});
	});

	it("adds nothing when its settings are cancelled", async () => {
		const { user } = await openEditor([section("Work")]);
		await user.click(screen.getByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(picker).getByRole("button", { name: /Favourite explorer/ }),
		);
		const dialog = await screen.findByRole("dialog", {
			name: "Add favourite explorer",
		});
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(screen.queryByTestId(/^widget-/)).toBeNull();
	});

	it("shows a deleted folder and lets another one be chosen", async () => {
		const { user } = await openEditor([
			section("Work", {}, explorer("e1", { rootFolderId: "gone" })),
		]);
		expect(
			await screen.findByText("This folder no longer exists"),
		).toBeVisible();
		await user.click(
			screen.getByRole("button", { name: "Edit Favourite explorer" }),
		);
		const dialog = await screen.findByRole("dialog", {
			name: "Edit favourite explorer",
		});
		const folder = within(dialog).getByLabelText("Folder");
		await waitFor(() => expect(folder).toHaveValue(""));
		const apply = within(dialog).getByRole("button", { name: "Apply" });
		expect(apply).toBeDisabled();
		await user.selectOptions(folder, "Favourites / Work / Tools");
		await user.click(apply);
		expect(await screen.findByText("GitLab")).toBeVisible();
	});
});
