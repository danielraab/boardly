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

	it("shows a title as a heading, with expand all on its line", () => {
		render(
			<ExplorerTree
				widgetId="w1"
				size="M"
				data={{ children: [tools, mail] }}
				editing={false}
				title="Work links"
				headingLevel={4}
			/>,
		);
		const heading = screen.getByRole("heading", {
			level: 4,
			name: "Work links",
		});
		expect(heading).toHaveClass("truncate");
		expect(heading.parentElement).toContainElement(
			screen.getByRole("button", { name: "Expand all" }),
		);
	});

	it("shows no heading without a title, and the title with a missing folder", () => {
		const { unmount } = tree({ children: [tools] });
		expect(screen.queryByRole("heading")).toBeNull();
		unmount();
		render(
			<ExplorerTree
				widgetId="w1"
				size="M"
				data={{ missing: true }}
				editing={false}
				title="Work links"
			/>,
		);
		expect(screen.getByRole("heading", { name: "Work links" })).toBeVisible();
		expect(screen.getByText("This folder no longer exists")).toBeVisible();
	});

	it("has no expand-all button without folders", () => {
		tree({ children: [mail] });
		expect(screen.queryByRole("button")).toBeNull();
	});

	it("shows an add button left of expand all, in the same size", async () => {
		const user = userEvent.setup();
		const onAdd = vi.fn();
		render(
			<ExplorerTree
				widgetId="w1"
				size="M"
				data={{ children: [tools, mail] }}
				editing={false}
				onAdd={onAdd}
			/>,
		);
		const add = screen.getByRole("button", { name: "Add favourite" });
		const expand = screen.getByRole("button", { name: "Expand all" });
		expect(add.nextElementSibling).toBe(expand);
		expect(add.className).toBe(expand.className);
		await user.click(add);
		expect(onAdd).toHaveBeenCalledOnce();
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
		const folder = await within(dialog).findByRole("radiogroup", {
			name: "Folder",
		});
		const radios = () =>
			within(folder)
				.getAllByRole("radio")
				.map((r) => r.closest("label")?.textContent);
		// Folders only, collapsed, with the root first.
		expect(radios()).toEqual(["Favourites", "Work"]);
		await user.click(
			within(folder).getByRole("button", { name: "Expand Work" }),
		);
		expect(radios()).toEqual(["Favourites", "Work", "Tools"]);
		await user.click(within(folder).getByRole("radio", { name: "Work" }));
		expect(
			within(dialog).getByText("Selected: Favourites / Work"),
		).toBeVisible();
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		await user.type(within(dialog).getByLabelText("Title"), "  Work links ");
		await user.selectOptions(within(dialog).getByLabelText("Size"), "Large");
		await user.click(add);

		// A level below the section "Work", like a section beside it.
		expect(
			await screen.findByRole("heading", { level: 3, name: "Work links" }),
		).toBeVisible();

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
					config: {
						rootFolderId: "work",
						title: "Work links",
						size: "L",
						showIfPublic: false,
					},
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
		const folder = await within(dialog).findByRole("radiogroup", {
			name: "Folder",
		});
		expect(within(folder).queryByRole("radio", { checked: true })).toBeNull();
		const apply = within(dialog).getByRole("button", { name: "Apply" });
		expect(apply).toBeDisabled();
		// Found by searching, shown with the path of its folder.
		await user.type(within(dialog).getByRole("searchbox"), "too");
		const tools = within(folder).getByRole("radio", { name: "Tools" });
		expect(tools).toHaveAccessibleDescription("Favourites / Work");
		await user.click(tools);
		await user.click(apply);
		expect(await screen.findByText("GitLab")).toBeVisible();
	});
});

describe("adding a favourite from the explorer", () => {
	const viewRoutes = (sections: Widget[], fields = {}) => ({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections, fields) },
		"GET /api/favourites": { body: favourites },
		"GET /api/tags": { body: [] },
	});

	it("adds a link to the explorer's folder and refreshes the dashboard", async () => {
		const api = mockApi({
			...viewRoutes([
				section("Work", {}, explorer("e1", {}, { children: [mail] })),
			]),
			"POST /api/favourites": ({ body }) => ({
				status: 201,
				body: { id: "new", ...(body as object) },
			}),
		});
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Add favourite" }),
		);

		const dialog = await screen.findByRole("dialog", { name: "New link" });
		const folder = within(dialog).getByRole("radiogroup", { name: "Folder" });
		expect(within(folder).getByRole("radio", { name: "Work" })).toBeChecked();
		await user.click(within(dialog).getByRole("textbox", { name: "URL" }));
		await user.paste("https://example.com/");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));

		await waitFor(() =>
			expect(screen.queryByRole("dialog", { name: "New link" })).toBeNull(),
		);
		expect(api.callsTo("POST", "/api/favourites").at(-1)?.body).toEqual({
			parentId: "work",
			kind: "link",
			url: "https://example.com/",
			title: "",
			tags: [],
		});
		await waitFor(() =>
			expect(api.callsTo("GET", "/api/u/ann/home")).toHaveLength(2),
		);
	});

	it("has no add button for visitors", async () => {
		mockApi(
			viewRoutes(
				[section("Work", {}, explorer("e1", {}, { children: [mail] }))],
				{ isOwner: false, id: undefined },
			),
		);
		renderApp("/ann/home");
		await screen.findByRole("link", { name: "https://mail.example.org/inbox" });
		expect(screen.queryByRole("button", { name: "Add favourite" })).toBeNull();
	});

	it("has no add button for a deleted folder", async () => {
		mockApi(
			viewRoutes([
				section(
					"Work",
					{},
					explorer("e1", { rootFolderId: "gone" }, { missing: true }),
				),
			]),
		);
		renderApp("/ann/home");
		await screen.findByText("This folder no longer exists");
		expect(screen.queryByRole("button", { name: "Add favourite" })).toBeNull();
	});

	it("has no add button in edit mode", async () => {
		await openEditor([section("Work", {}, explorer("e1", {}))]);
		await screen.findByText("https://mail.example.org/inbox");
		expect(screen.queryByRole("button", { name: "Add favourite" })).toBeNull();
	});
});

describe("explorer details", () => {
	const viewRoutes = (sections: Widget[], fields = {}) => ({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections, fields) },
		"GET /api/favourites": { body: favourites },
	});
	const workExplorer = () =>
		section("Work", {}, explorer("e1", {}, { children: [tools, mail] }));

	it("has a small info button on link rows only", async () => {
		const onDetails = vi.fn();
		const user = userEvent.setup();
		render(
			<ExplorerTree
				widgetId="w1"
				size="M"
				data={{ children: [tools, mail] }}
				editing={false}
				onDetails={onDetails}
			/>,
		);
		const button = screen.getByRole("button", {
			name: "Details for https://mail.example.org/inbox",
		});
		expect(button.closest("a")).toBeNull();
		expect(button.querySelector("svg")).toHaveClass("size-3.5");
		// Folders have none.
		expect(
			screen.getAllByRole("button", { name: /^Details for/ }),
		).toHaveLength(1);
		await user.click(button);
		expect(onDetails).toHaveBeenCalledWith("mail");
	});

	it("opens a link's details for the owner", async () => {
		mockApi(viewRoutes([workExplorer()]));
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", {
				name: "Details for https://mail.example.org/inbox",
			}),
		);
		expect(
			await screen.findByRole("dialog", {
				name: "https://mail.example.org/inbox",
			}),
		).toBeVisible();
	});

	it("has no info buttons for visitors", async () => {
		mockApi(viewRoutes([workExplorer()], { isOwner: false, id: undefined }));
		renderApp("/ann/home");
		await screen.findByRole("link", { name: "https://mail.example.org/inbox" });
		expect(screen.queryByRole("button", { name: /^Details for/ })).toBeNull();
	});

	it("has no info buttons in edit mode", async () => {
		await openEditor([section("Work", {}, explorer("e1", {}))]);
		await screen.findByText("https://mail.example.org/inbox");
		expect(screen.queryByRole("button", { name: /^Details for/ })).toBeNull();
	});
});

describe("explorer title link", () => {
	const viewRoutes = (sections: Widget[], fields = {}) => ({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections, fields) },
		"GET /api/favourites": { body: favourites },
	});
	const titled = (config = {}, data: ExplorerData = { children: [mail] }) =>
		section(
			"Work",
			{},
			explorer("e1", { title: "Work links", ...config }, data),
		);

	it("links the owner to the folder on the favourites page", async () => {
		mockApi(viewRoutes([titled()]));
		const { user, router } = renderApp("/ann/home");
		const title = await screen.findByRole("link", { name: "Work links" });
		expect(title).toHaveAttribute("href", "/favourites?folder=work");
		await user.click(title);
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/favourites"),
		);
	});

	it("links to the favourites page for the tree's root", async () => {
		mockApi(viewRoutes([titled({ rootFolderId: "root" })]));
		renderApp("/ann/home");
		expect(
			await screen.findByRole("link", { name: "Work links" }),
		).toHaveAttribute("href", "/favourites");
	});

	it("is plain text for visitors", async () => {
		mockApi(viewRoutes([titled()], { isOwner: false, id: undefined }));
		renderApp("/ann/home");
		await screen.findByRole("heading", { name: "Work links" });
		expect(screen.queryByRole("link", { name: "Work links" })).toBeNull();
	});

	it("is plain text in edit mode", async () => {
		await openEditor([titled()]);
		await screen.findByRole("heading", { name: "Work links" });
		expect(screen.queryByRole("link", { name: "Work links" })).toBeNull();
	});

	it("is plain text for a deleted folder", async () => {
		mockApi(viewRoutes([titled({ rootFolderId: "gone" }, { missing: true })]));
		renderApp("/ann/home");
		await screen.findByText("This folder no longer exists");
		expect(screen.queryByRole("link", { name: "Work links" })).toBeNull();
	});
});
