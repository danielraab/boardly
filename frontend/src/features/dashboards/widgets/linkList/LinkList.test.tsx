import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { FavouriteNode } from "#/features/favourites/api";
import { mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import type { LinkListData, ListLink } from "./flatten";
import { type LinkListLayout, LinkListView } from "./LinkListView";

const mail: ListLink = {
	id: "mail",
	title: "Mail",
	url: "https://mail.example.org/inbox",
	iconImageId: "img1",
};
const wiki: ListLink = {
	id: "wiki",
	title: null,
	url: "https://wiki.example.org/",
};

function view(
	data: LinkListData,
	fields: {
		layout?: LinkListLayout;
		editing?: boolean;
		title?: string;
		size?: "S" | "M" | "L";
	} = {},
) {
	return render(
		<LinkListView
			data={data}
			layout={fields.layout ?? "list"}
			size={fields.size ?? "M"}
			editing={fields.editing ?? false}
			title={fields.title}
			headingLevel={4}
		/>,
	);
}

describe("LinkListView", () => {
	it("shows title and URL in the layout list", () => {
		view({ links: [mail, wiki] });
		const link = screen.getByRole("link", { name: /Mail/ });
		expect(link).toHaveTextContent("Mailhttps://mail.example.org/inbox");
		expect(link).toHaveAttribute("href", "https://mail.example.org/inbox");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img1",
		);
		// Without a favicon, a globe.
		const other = screen.getByRole("link", {
			name: "https://wiki.example.org/",
		});
		expect(other.querySelector("[data-icon='globe']")).not.toBeNull();
	});

	it("shows only the title in the layout compact, with the URL as tooltip", () => {
		view({ links: [mail, wiki] }, { layout: "compact" });
		const link = screen.getByRole("link", { name: "Mail" });
		expect(link).toHaveTextContent(/^Mail$/);
		expect(link).toHaveAttribute(
			"title",
			"Mail\nhttps://mail.example.org/inbox",
		);
		expect(link).toHaveAttribute("target", "_blank");
		expect(
			screen.getByRole("link", { name: "https://wiki.example.org/" }),
		).toBeVisible();
	});

	it("shows tiles in a wrapping grid, sized by the size", () => {
		const { container } = view(
			{ links: [mail, wiki] },
			{ layout: "tiles", size: "L" },
		);
		expect(container.querySelector("ul")?.className).toContain(
			"grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))]",
		);
		const link = screen.getByRole("link", { name: "Mail" });
		expect(link).toHaveAttribute("target", "_blank");
		expect(link.querySelector("img")).toHaveClass("size-10");
		expect(screen.getByText("Mail")).toHaveClass("truncate");
	});

	it("doesn't navigate in edit mode", async () => {
		const user = userEvent.setup();
		for (const layout of ["list", "compact", "tiles"] as const) {
			const { unmount } = view({ links: [mail] }, { layout, editing: true });
			const link = screen.getByRole("link", { name: /Mail/ });
			expect(link).toHaveAttribute("aria-disabled", "true");
			await user.click(link);
			unmount();
		}
	});

	it("shows a title as a heading, and none without", () => {
		const { unmount } = view({ links: [mail] }, { title: "Work links" });
		expect(
			screen.getByRole("heading", { level: 4, name: "Work links" }),
		).toHaveClass("truncate");
		unmount();
		view({ links: [mail] });
		expect(screen.queryByRole("heading")).toBeNull();
	});

	it("shows an empty and a missing folder", () => {
		const { unmount } = view({ links: [] });
		expect(screen.getByText("No links in this folder")).toBeVisible();
		unmount();
		view({ missing: true });
		expect(screen.getByText("This folder no longer exists")).toBeVisible();
	});
});

const favourites: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{
			id: "work",
			kind: "folder",
			title: "Work",
			children: [
				{ id: "mail", kind: "link", title: "Mail", url: mail.url },
				{
					id: "tools",
					kind: "folder",
					title: "Tools",
					children: [
						{
							id: "gitlab",
							kind: "link",
							title: "GitLab",
							url: "https://gitlab.example.org/",
						},
					],
				},
			],
		},
	],
};

function linkList(id: string, config: object, data?: LinkListData) {
	return {
		id,
		kind: "link_list",
		width: 2,
		height: "auto",
		config: {
			folderId: "work",
			title: null,
			includeSubfolders: false,
			layout: "list",
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
	} satisfies ContentWidget;
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

describe("the link list on a dashboard", () => {
	it("shows the server's links in view mode", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[section("Work", {}, linkList("l1", {}, { links: [wiki] }))],
					{ isOwner: false, id: undefined },
				),
			},
		});
		renderApp("/ann/home");
		expect(
			await screen.findByRole("link", { name: "https://wiki.example.org/" }),
		).toBeVisible();
	});

	it("is added through the picker, needs a folder, and saves its settings", async () => {
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
		await user.click(within(picker).getByRole("button", { name: /Link list/ }));

		const dialog = await screen.findByRole("dialog", {
			name: "Add link list",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		expect(
			within(dialog).getByLabelText("Show items from subfolders"),
		).not.toBeChecked();
		expect(within(dialog).getByLabelText("Layout")).toHaveValue("list");
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();

		const folder = await within(dialog).findByRole("radiogroup", {
			name: "Folder",
		});
		await user.click(within(folder).getByRole("radio", { name: "Work" }));
		await user.type(within(dialog).getByLabelText("Title"), "Work links");
		await user.click(
			within(dialog).getByLabelText("Show items from subfolders"),
		);
		await user.selectOptions(within(dialog).getByLabelText("Layout"), "Tiles");
		await user.click(add);

		// The draft shows the folder's links, including the subfolder's.
		expect(
			await screen.findByRole("heading", { level: 3, name: "Work links" }),
		).toBeVisible();
		expect(screen.getByRole("link", { name: "Mail" })).toBeVisible();
		expect(screen.getByRole("link", { name: "GitLab" })).toBeVisible();
		expect(screen.queryByText("Tools")).toBeNull();

		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "link_list",
					config: {
						folderId: "work",
						title: "Work links",
						includeSubfolders: true,
						layout: "tiles",
						size: "M",
						showIfPublic: false,
					},
				},
			],
		});
	});

	it("shows a deleted folder and lets another one be chosen", async () => {
		const { user } = await openEditor([
			section("Work", {}, linkList("l1", { folderId: "gone" })),
		]);
		expect(
			await screen.findByText("This folder no longer exists"),
		).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Edit Link list" }));
		const dialog = await screen.findByRole("dialog", {
			name: "Edit link list",
		});
		const apply = within(dialog).getByRole("button", { name: "Apply" });
		expect(apply).toBeDisabled();
		const folder = await within(dialog).findByRole("radiogroup", {
			name: "Folder",
		});
		await user.click(
			within(folder).getByRole("button", { name: "Expand Work" }),
		);
		await user.click(within(folder).getByRole("radio", { name: "Tools" }));
		await user.click(apply);
		expect(await screen.findByRole("link", { name: /GitLab/ })).toBeVisible();
	});
});
