import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { FavouriteNode } from "#/features/favourites/api";
import { mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import { RecentFavouritesView } from "./RecentFavouritesView";
import type { RecentData, RecentLink } from "./recent";

const now = new Date("2026-10-27T12:00:00Z");
const day = 24 * 60 * 60 * 1000;
const daysAgo = (n: number) => new Date(now.getTime() - n * day).toISOString();

const mail: RecentLink = {
	id: "mail",
	title: "Mail",
	url: "https://mail.example.org/inbox",
	iconImageId: "img1",
	createdAt: daysAgo(2),
};
const wiki: RecentLink = {
	id: "wiki",
	title: null,
	url: "https://wiki.example.org/a/very/long/path",
	createdAt: daysAgo(40),
};

function view(
	data: RecentData,
	fields: {
		days?: number | null;
		editing?: boolean;
		title?: string;
		size?: "S" | "M" | "L";
		onDetails?: (id: string) => void;
	} = {},
) {
	return render(
		<RecentFavouritesView
			data={data}
			days={fields.days ?? null}
			size={fields.size ?? "M"}
			editing={fields.editing ?? false}
			title={fields.title}
			headingLevel={4}
			onDetails={fields.onDetails}
			now={now}
		/>,
	);
}

describe("RecentFavouritesView", () => {
	it("shows icon, title, URL and when each link was added", () => {
		view({ links: [mail, wiki] });
		const link = screen.getByRole("link", { name: /Mail/ });
		expect(link).toHaveTextContent(
			"Mailhttps://mail.example.org/inbox2 days ago",
		);
		expect(link).toHaveAttribute("href", "https://mail.example.org/inbox");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img1",
		);
		const date = within(link).getByText("2 days ago");
		expect(date.tagName).toBe("TIME");
		expect(date).toHaveAttribute("dateTime", mail.createdAt);
		expect(date.getAttribute("title")).toMatch(/2026/);

		// Without a title the URL takes its place, cut off when it is too long.
		const other = screen.getByRole("link", { name: /wiki\.example\.org/ });
		expect(other.querySelector("[data-icon='globe']")).not.toBeNull();
		expect(within(other).getByText(wiki.url)).toHaveClass("truncate");
		expect(within(other).getByText("last month")).toBeVisible();
	});

	it("doesn't navigate in edit mode", () => {
		view({ links: [mail] }, { editing: true });
		expect(screen.getByRole("link", { name: /Mail/ })).toHaveAttribute(
			"aria-disabled",
			"true",
		);
	});

	it("shows a title as a heading, and none without", () => {
		const { unmount } = view({ links: [mail] }, { title: "Recently added" });
		expect(
			screen.getByRole("heading", { level: 4, name: "Recently added" }),
		).toHaveClass("truncate");
		unmount();
		view({ links: [mail] });
		expect(screen.queryByRole("heading")).toBeNull();
	});

	it("sizes icons and text by the size", () => {
		view({ links: [mail] }, { size: "L" });
		expect(
			screen.getByRole("link", { name: /Mail/ }).querySelector("img"),
		).toHaveClass("size-5");
	});

	it("shows the empty states and a missing folder", () => {
		const { unmount } = view({ links: [] });
		expect(screen.getByText("No favourites")).toBeVisible();
		unmount();
		const second = view({ links: [] }, { days: 7 });
		expect(screen.getByText("No favourites in the last 7 days")).toBeVisible();
		second.unmount();
		const third = view({ missing: true });
		expect(screen.getByText("This folder no longer exists")).toBeVisible();
		third.unmount();
	});

	it("has an info button beside each link with onDetails", async () => {
		const onDetails = vi.fn();
		const user = userEvent.setup();
		view({ links: [mail] }, { onDetails });
		const button = screen.getByRole("button", { name: "Details for Mail" });
		expect(button.closest("a")).toBeNull();
		await user.click(button);
		expect(onDetails).toHaveBeenCalledWith("mail");
	});
});

const recentAgo = (n: number) =>
	new Date(Date.now() - n * day - 60_000).toISOString();

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
				{
					id: "mail",
					kind: "link",
					title: "Mail",
					url: mail.url,
					createdAt: recentAgo(3),
				},
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
							createdAt: recentAgo(1),
						},
						{
							id: "old",
							kind: "link",
							title: "Old",
							url: "https://old.example.org/",
							createdAt: recentAgo(100),
						},
					],
				},
			],
		},
	],
};

function recentWidget(id: string, config: object, data?: RecentData) {
	return {
		id,
		kind: "recent_favourites",
		width: 2,
		height: "auto",
		config: {
			folderId: "work",
			limit: 10,
			days: null,
			title: null,
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

describe("the recent favourites widget on a dashboard", () => {
	it("shows the server's links in view mode, without info buttons for visitors", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[section("Work", {}, recentWidget("r1", {}, { links: [mail] }))],
					{ isOwner: false, id: undefined },
				),
			},
		});
		renderApp("/ann/home");
		expect(await screen.findByRole("link", { name: /Mail/ })).toBeVisible();
		expect(screen.queryByRole("button", { name: /^Details for/ })).toBeNull();
	});

	it("opens a link's details for the owner", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/u/ann/home": {
				body: viewData([
					section("Work", {}, recentWidget("r1", {}, { links: [mail] })),
				]),
			},
			"GET /api/favourites": { body: favourites },
		});
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Details for Mail" }),
		);
		expect(await screen.findByRole("dialog", { name: "Mail" })).toBeVisible();
	});

	it("is added through the picker, needs a folder and valid numbers, and saves its settings", async () => {
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
			within(picker).getByRole("button", { name: /Recent favourites/ }),
		);

		const dialog = await screen.findByRole("dialog", {
			name: "Add recent favourites",
		});
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Title")).toHaveValue("");
		expect(within(dialog).getByLabelText("Number of links")).toHaveValue(10);
		expect(within(dialog).getByLabelText("Only the last … days")).toHaveValue(
			null,
		);
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();

		const folder = await within(dialog).findByRole("radiogroup", {
			name: "Folder",
		});
		await user.click(within(folder).getByRole("radio", { name: "Work" }));
		expect(add).toBeEnabled();

		const limit = within(dialog).getByLabelText("Number of links");
		await user.clear(limit);
		await user.type(limit, "101");
		expect(add).toBeDisabled();
		await user.clear(limit);
		await user.type(limit, "5");
		const days = within(dialog).getByLabelText("Only the last … days");
		await user.type(days, "0");
		expect(add).toBeDisabled();
		await user.clear(days);
		await user.type(days, "7");
		await user.type(within(dialog).getByLabelText("Title"), "New");
		await user.click(add);

		// The draft shows the newest links of the last 7 days, subfolders included.
		expect(
			await screen.findByRole("heading", { level: 3, name: "New" }),
		).toBeVisible();
		const links = screen.getAllByRole("link", { name: /GitLab|Mail|Old/ });
		expect(links.map((l) => l.textContent)).toEqual([
			expect.stringMatching(/^GitLab/),
			expect.stringMatching(/^Mail/),
		]);
		expect(screen.queryByText("Tools")).toBeNull();

		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "recent_favourites",
					config: {
						folderId: "work",
						limit: 5,
						days: 7,
						title: "New",
						size: "M",
						showIfPublic: false,
					},
				},
			],
		});
	});

	it("shows a deleted folder and lets another one be chosen", async () => {
		const { user } = await openEditor([
			section("Work", {}, recentWidget("r1", { folderId: "gone" })),
		]);
		expect(
			await screen.findByText("This folder no longer exists"),
		).toBeVisible();
		await user.click(
			screen.getByRole("button", { name: "Edit Recent favourites" }),
		);
		const dialog = await screen.findByRole("dialog", {
			name: "Edit recent favourites",
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
		expect(screen.queryByRole("button", { name: /^Details for/ })).toBeNull();
	});
});
