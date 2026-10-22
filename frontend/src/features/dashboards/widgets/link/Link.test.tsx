import {
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { FavouriteNode } from "#/features/favourites/api";
import { apiError, mockApi, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";
import type { LinkConfig } from "./LinkSettings";
import { LinkView } from "./LinkView";

function view(fields: Partial<Parameters<typeof LinkView>[0]> = {}) {
	return render(
		<LinkView
			url="https://mail.example.org/"
			title={null}
			imageId={null}
			icon={null}
			target="_blank"
			size="M"
			editing={false}
			{...fields}
		/>,
	);
}

describe("LinkView", () => {
	it("shows the title and the URL on two lines, cut off when too long", () => {
		view({ title: "Mail" });
		const link = screen.getByRole("link");
		expect(link).toHaveAttribute("href", "https://mail.example.org/");
		expect(link).toHaveAttribute("title", "Mail\nhttps://mail.example.org/");
		expect(screen.getByText("Mail")).toHaveClass("truncate", "font-medium");
		expect(screen.getByText("https://mail.example.org/")).toHaveClass(
			"truncate",
			"text-muted",
		);
		expect(link).toHaveClass("min-w-0");
	});

	it("shows only the URL without a title", () => {
		view();
		const link = screen.getByRole("link", {
			name: "https://mail.example.org/",
		});
		expect(link.querySelectorAll(".truncate")).toHaveLength(1);
		expect(link).toHaveAttribute("title", "https://mail.example.org/");
	});

	it("shows the image, else the icon, else a globe", () => {
		const { container, rerender } = view({ imageId: "img1", icon: "mail" });
		expect(container.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/img1",
		);
		const props = {
			url: "https://mail.example.org/",
			title: null,
			target: "_blank" as const,
			size: "M" as const,
			editing: false,
		};
		rerender(<LinkView {...props} imageId={null} icon="mail" />);
		expect(container.querySelector("[data-icon='mail']")).not.toBeNull();
		rerender(<LinkView {...props} imageId={null} icon={null} />);
		expect(container.querySelector("[data-icon='globe']")).not.toBeNull();
	});

	it("opens in a new tab or in the same tab", () => {
		const { rerender } = view();
		expect(screen.getByRole("link")).toHaveAttribute("target", "_blank");
		expect(screen.getByRole("link")).toHaveAttribute(
			"rel",
			"noopener noreferrer",
		);
		rerender(
			<LinkView
				url="https://mail.example.org/"
				title={null}
				imageId={null}
				icon={null}
				target="_self"
				size="L"
				editing={false}
			/>,
		);
		expect(screen.getByRole("link")).toHaveAttribute("target", "_self");
		expect(screen.getByRole("link")).not.toHaveAttribute("rel");
		expect(screen.getByRole("link")).toHaveClass("text-base", "py-1.5");
	});

	it("doesn't navigate in edit mode", () => {
		view({ editing: true });
		const link = screen.getByRole("link");
		expect(link).toHaveAttribute("aria-disabled", "true");
		expect(fireEvent.click(link)).toBe(false);
	});
});

const mail: FavouriteNode = {
	id: "mail",
	kind: "link",
	title: "Mail",
	url: "https://mail.example.org/",
	iconImageId: "mailicon",
};
const favourites: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		{ id: "home", kind: "folder", title: "Home", children: [] },
		{ id: "work", kind: "folder", title: "Work", children: [mail] },
	],
};

function linkWidget(
	config: Partial<LinkConfig>,
	data?: unknown,
): ContentWidget {
	return {
		id: "l1",
		kind: "link",
		width: 2,
		height: "auto",
		config: {
			source: "url",
			url: "https://example.org/",
			title: null,
			imageId: null,
			icon: null,
			target: "_blank",
			size: "M",
			...config,
		},
		data,
	};
}

async function openEditor(sections: Widget[], more = {}) {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/u/ann/home": { body: viewData(sections) },
		"GET /api/favourites": { body: favourites },
		"PUT /api/dashboards/d1/layout": ({ body }: { body: unknown }) => ({
			body: { version: 4, sections: (body as { sections: Widget[] }).sections },
		}),
		...more,
	});
	const app = renderApp("/ann/home?edit=true");
	await screen.findByRole("button", { name: "Save and exit" });
	return { api, ...app };
}

async function openNewLink(user: UserEvent) {
	await user.click(screen.getByRole("button", { name: "Add widget" }));
	const picker = await screen.findByRole("dialog", { name: "Add widget" });
	await user.click(within(picker).getByRole("button", { name: /^Link/ }));
	return screen.findByRole("dialog", { name: "Add link" });
}

async function savedConfig(
	user: UserEvent,
	api: Awaited<ReturnType<typeof openEditor>>["api"],
	router: Awaited<ReturnType<typeof openEditor>>["router"],
) {
	await user.click(screen.getByRole("button", { name: "Save and exit" }));
	await waitFor(() => expect(router.state.location.search).toEqual({}));
	const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
	return (
		(call.body as { sections: Widget[] }).sections[0] as {
			children: ContentWidget[];
		}
	).children[0].config;
}

describe("the link widget on a dashboard", () => {
	it("shows the favourite's current values in view mode", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[
						section(
							"Work",
							{},
							linkWidget(
								{
									source: "favourite",
									favouriteId: "mail",
									favouriteTitle: "Mail",
									url: "https://mail.example.org/",
								},
								{
									url: "https://webmail.example.org/",
									title: "Webmail",
									iconImageId: "webmailicon",
								},
							),
						),
					],
					{ isOwner: false, id: undefined },
				),
			},
		});
		renderApp("/ann/home");
		const link = await screen.findByRole("link", { name: /Webmail/ });
		expect(link).toHaveAttribute("href", "https://webmail.example.org/");
		expect(link.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/webmailicon",
		);
	});

	it("adds a favourite through the picker, with its website icon", async () => {
		const { api, user, router } = await openEditor([section("Work")]);
		const dialog = await openNewLink(user);
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(
			within(dialog).getByRole("radio", { name: "Favourite" }),
		).toBeChecked();
		const picker = await within(dialog).findByRole("radiogroup", {
			name: "Favourite",
		});
		// Links only: the empty folder Home isn't offered.
		expect(within(picker).queryByText("Home")).toBeNull();
		await user.click(
			within(picker).getByRole("button", { name: "Expand Work" }),
		);
		await user.click(within(picker).getByRole("radio", { name: "Mail" }));
		// The favourite's known favicon is shown without a request.
		expect(
			within(dialog)
				.getByRole("button", { name: "Icon: Website icon" })
				.querySelector("img"),
		).toHaveAttribute("src", "/api/images/mailicon");
		expect(api.callsTo("POST", "/api/favicons")).toHaveLength(0);
		await user.click(add);

		const link = await screen.findByRole("link", { name: /Mail/ });
		expect(link.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/mailicon",
		);
		expect(await savedConfig(user, api, router)).toEqual({
			source: "favourite",
			favouriteId: "mail",
			favouriteTitle: "Mail",
			url: "https://mail.example.org/",
			title: null,
			iconSource: "favicon",
			imageId: null,
			icon: null,
			target: "_blank",
			size: "M",
		});
	});

	it("validates a URL and looks up its website icon", async () => {
		const { api, user } = await openEditor([section("Work")], {
			"POST /api/favicons": ({ body }: { body: unknown }) =>
				(body as { url: string }).url === "https://github.com"
					? { body: { id: "gh" } }
					: apiError(404, "favicon_not_found"),
		});
		const dialog = await openNewLink(user);
		await user.click(within(dialog).getByLabelText("URL"));
		const url = within(dialog).getByRole("textbox", { name: "URL" });
		const add = within(dialog).getByRole("button", { name: "Add" });

		// Pasted, so each URL is one change.
		await user.click(url);
		await user.paste("javascript:alert(1)");
		expect(
			within(dialog).getByText(
				"Enter a web address that starts with http:// or https://.",
			),
		).toBeVisible();
		expect(add).toBeDisabled();

		await user.clear(url);
		await user.paste("https://github.com");
		const trigger = within(dialog).getByRole("button", {
			name: "Icon: Website icon",
		});
		await waitFor(() =>
			expect(trigger.querySelector("img")).toHaveAttribute(
				"src",
				"/api/images/gh",
			),
		);
		await user.clear(url);
		await user.paste("https://nothing.example/");
		// Only URLs that stayed unchanged are looked up.
		await waitFor(() =>
			expect(api.callsTo("POST", "/api/favicons")).toHaveLength(2),
		);
		expect(trigger.querySelector("[data-icon='globe']")).not.toBeNull();

		await user.click(trigger);
		const chooser = await screen.findByRole("dialog", { name: "Icon" });
		expect(
			within(chooser).getByRole("tab", { name: "Website icon" }),
		).toHaveAttribute("aria-selected", "true");
		expect(
			within(chooser).getByText(
				"No website icon found for nothing.example. A globe is shown instead.",
			),
		).toBeVisible();
	});

	it("chooses a symbol by filter and with arrow keys", async () => {
		const { api, user, router } = await openEditor([
			section("Work", {}, linkWidget({ url: "https://nothing.example/" })),
		]);
		await user.click(screen.getByRole("button", { name: "Edit Link" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit link" });
		await user.click(
			within(dialog).getByRole("button", { name: "Icon: Website icon" }),
		);
		const chooser = await screen.findByRole("dialog", { name: "Icon" });
		await user.click(within(chooser).getByRole("tab", { name: "Symbol" }));
		const filter = within(chooser).getByRole("searchbox");
		await user.click(filter);
		await user.paste("cod");
		const symbols = within(chooser).getByRole("radiogroup", {
			name: "Symbols",
		});
		expect(within(symbols).getAllByRole("radio")).toHaveLength(1);
		await user.click(within(symbols).getByRole("radio", { name: "Code" }));
		expect(
			within(dialog).getByRole("button", { name: "Icon: Code" }),
		).toBeVisible();
		// Arrow keys move through the grid and choose as they go.
		await user.clear(filter);
		within(symbols).getByRole("radio", { name: "Code" }).focus();
		await user.keyboard("{ArrowRight}");
		expect(
			within(symbols).getByRole("radio", { name: "Processor" }),
		).toBeChecked();
		await user.click(within(chooser).getByRole("button", { name: "Done" }));
		await waitFor(() =>
			expect(screen.queryByRole("dialog", { name: "Icon" })).toBeNull(),
		);
		await user.selectOptions(
			within(dialog).getByLabelText("Open in"),
			"Same tab",
		);
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));
		expect(await savedConfig(user, api, router)).toEqual({
			source: "url",
			url: "https://nothing.example/",
			title: null,
			iconSource: "symbol",
			imageId: null,
			icon: "cpu",
			target: "_self",
			size: "M",
		});
	});

	it("uploads an own image, shows upload errors, and switches to none", async () => {
		let uploads = 0;
		const { api, user, router } = await openEditor([section("Work")], {
			"POST /api/images": ({ init }: { init: RequestInit }) => {
				uploads++;
				expect(new Headers(init.headers).get("Content-Type")).toBe(
					"application/octet-stream",
				);
				return uploads === 1
					? apiError(400, "unsupported_image")
					: { status: 201, body: { id: "up1" } };
			},
		});
		const dialog = await openNewLink(user);
		await user.click(within(dialog).getByLabelText("URL"));
		await user.click(within(dialog).getByRole("textbox", { name: "URL" }));
		await user.paste("https://example.org/");
		await user.click(
			within(dialog).getByRole("button", { name: "Icon: Website icon" }),
		);
		const chooser = await screen.findByRole("dialog", { name: "Icon" });
		await user.click(within(chooser).getByRole("tab", { name: "Own image" }));
		const input = within(chooser).getByLabelText("Upload image");

		// The server sniffs the content, so an SVG named like a PNG is rejected.
		await user.upload(
			input,
			new File(["<svg/>"], "logo.png", { type: "image/png" }),
		);
		expect(
			await within(chooser).findByText(
				"This file type isn't supported. Use PNG, JPEG, WebP, GIF or ICO.",
			),
		).toBeVisible();
		expect(
			within(dialog).getByRole("button", { name: "Icon: Website icon" }),
		).toBeVisible();

		await user.upload(
			within(chooser).getByLabelText("Upload image"),
			new File(["png"], "logo.png", { type: "image/png" }),
		);
		const trigger = await within(dialog).findByRole("button", {
			name: "Icon: Own image",
		});
		expect(trigger.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/up1",
		);

		expect(within(chooser).getAllByRole("tab")).toHaveLength(3);
		await user.click(
			within(chooser).getByRole("button", { name: "Remove icon" }),
		);
		expect(
			within(dialog).getByRole("button", { name: "Icon: None" }),
		).toBeVisible();
		const removed = within(chooser).getByRole("button", { name: "No icon" });
		expect(removed).toHaveAttribute("aria-pressed", "true");
		expect(removed).toHaveFocus();

		// Without an icon, the chooser opens on the website icon.
		await user.click(within(chooser).getByRole("button", { name: "Done" }));
		await waitFor(() =>
			expect(screen.queryByRole("dialog", { name: "Icon" })).toBeNull(),
		);
		await user.click(
			within(dialog).getByRole("button", { name: "Icon: None" }),
		);
		const reopened = await screen.findByRole("dialog", { name: "Icon" });
		expect(
			within(reopened).getByRole("tab", { name: "Website icon" }),
		).toHaveAttribute("aria-selected", "true");

		// The image is kept, so it can be chosen again.
		await user.click(within(reopened).getByRole("tab", { name: "Own image" }));
		await user.click(
			within(reopened).getByRole("button", { name: "Use this image" }),
		);
		expect(within(reopened).getByText("In use")).toBeVisible();
		await user.keyboard("{Escape}");
		await waitFor(() =>
			expect(screen.queryByRole("dialog", { name: "Icon" })).toBeNull(),
		);
		// Escape closed only the chooser.
		await user.click(within(dialog).getByRole("button", { name: "Add" }));
		expect(await savedConfig(user, api, router)).toMatchObject({
			iconSource: "image",
			imageId: "up1",
			icon: null,
		});
	});

	it("creates a new favourite in its own dialog and chooses it", async () => {
		let created = 0;
		const nas: FavouriteNode = {
			id: "nas",
			kind: "link",
			title: "NAS",
			url: "https://nas.example.org/",
		};
		const withNas: FavouriteNode = {
			...favourites,
			children: [
				{ id: "home", kind: "folder", title: "Home", children: [nas] },
				...(favourites.children ?? []).slice(1),
			],
		};
		const { api, user } = await openEditor([section("Work")], {
			"GET /api/favourites": () => ({
				body: created > 1 ? withNas : favourites,
			}),
			"POST /api/favourites": () =>
				++created === 1 ? apiError(500, "error") : { status: 201, body: nas },
		});
		const settings = await openNewLink(user);
		// Only two sources: new favourites aren't entered in the settings.
		expect(
			within(settings).getAllByRole("radio", { name: /^(Favourite|URL)$/ }),
		).toHaveLength(2);
		expect(
			within(settings).queryByRole("radio", { name: "New favourite" }),
		).toBeNull();
		await user.click(
			await within(settings).findByRole("button", { name: "New favourite…" }),
		);

		const dialog = await screen.findByRole("dialog", { name: "New link" });
		await user.click(within(dialog).getByRole("textbox", { name: "URL" }));
		await user.paste("https://nas.example.org/");
		await user.type(within(dialog).getByLabelText("Title (optional)"), "NAS");
		const folder = within(dialog).getByRole("radiogroup", { name: "Folder" });
		// The root folder is chosen until another one is.
		expect(
			within(folder).getByRole("radio", { name: "Favourites" }),
		).toBeChecked();
		await user.click(within(folder).getByRole("radio", { name: "Home" }));

		// A failure keeps the favourite dialog open.
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		expect(
			await within(dialog).findByText(
				"Something went wrong. Please try again.",
			),
		).toBeVisible();
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() =>
			expect(screen.queryByRole("dialog", { name: "New link" })).toBeNull(),
		);
		expect(api.callsTo("POST", "/api/favourites").at(-1)?.body).toEqual({
			parentId: "home",
			kind: "link",
			url: "https://nas.example.org/",
			title: "NAS",
			tags: [],
		});

		// Back in the settings, the new favourite is chosen.
		expect(
			await within(settings).findByText("Selected: Favourites / Home / NAS"),
		).toBeVisible();
		await user.click(within(settings).getByRole("button", { name: "Add" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(screen.getByText("NAS")).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Cancel editing" }));
		const confirm = await screen.findByRole("alertdialog", {
			name: "Discard changes?",
		});
		await user.click(within(confirm).getByRole("button", { name: "Discard" }));
		await waitFor(() => expect(screen.queryByText("NAS")).toBeNull());
		// Nothing deletes the favourite.
		expect(api.calls.filter((c) => c.method === "DELETE")).toHaveLength(0);
	});

	it("has no height choice and always fits its content", async () => {
		const { api, user, router } = await openEditor([
			// Stored with a fixed height by an older version.
			section("Work", {}, { ...linkWidget({}), height: 3 }),
		]);
		await user.click(screen.getByRole("button", { name: "Edit Link" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit link" });
		expect(within(dialog).getByLabelText("Width")).toBeVisible();
		expect(within(dialog).queryByLabelText("Fit the content")).toBeNull();
		expect(within(dialog).queryByLabelText("Fixed")).toBeNull();
		await user.click(within(dialog).getByRole("button", { name: "Apply" }));

		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		const sent = (call.body as { sections: Widget[] }).sections[0] as {
			children: ContentWidget[];
		};
		expect(sent.children[0].height).toBe("auto");
	});

	it("falls back to the copies when the favourite is gone", async () => {
		const { user } = await openEditor([
			section(
				"Work",
				{},
				linkWidget({
					source: "favourite",
					favouriteId: "gone",
					favouriteTitle: "Old",
					url: "https://old.example.org/",
				}),
			),
		]);
		expect(await screen.findByText("Old")).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Edit Link" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit link" });
		expect(
			await within(dialog).findByText(
				"This favourite no longer exists. Choose another one.",
			),
		).toBeVisible();
		expect(
			within(dialog).getByRole("button", { name: "Apply" }),
		).toBeDisabled();
	});
});
