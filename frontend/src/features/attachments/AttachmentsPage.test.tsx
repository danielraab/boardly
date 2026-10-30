import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn, signedOut } from "#/test/api";
import { note } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { Attachment } from "./api";

const day = 24 * 60 * 60 * 1000;

const attachment = (
	id: string,
	name: string,
	fields: Partial<Attachment> = {},
): Attachment => ({
	id,
	name,
	contentType: "application/pdf",
	size: 1024 * 1024,
	url: `/api/attachments/${id}/${name}`,
	createdAt: "2026-10-01T12:00:00.000Z",
	deletedAt: null,
	deleteAt: null,
	keep: false,
	noteIds: [],
	...fields,
});

const report = attachment("a-report", "report.pdf", {
	noteIds: ["taxes"],
	createdAt: "2026-10-03T12:00:00.000Z",
});
const old = attachment("a-old", "old.png", {
	contentType: "image/png",
	size: 512 * 1024,
	deletedAt: "2026-10-01T12:00:00.000Z",
	deleteAt: new Date(
		Date.parse("2026-10-01T12:00:00.000Z") + 30 * day,
	).toISOString(),
});
const kept = attachment("a-kept", "kept.pdf", {
	size: 1024 * 1024,
	keep: true,
	deletedAt: "2026-10-01T12:00:00.000Z",
	createdAt: "2026-09-01T12:00:00.000Z",
});

function server(attachments: Attachment[] = [report, old, kept]) {
	const list = structuredClone(attachments);
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/attachments": () => ({
			body: {
				attachments: list,
				maxSize: 5 * 1024 * 1024,
				allowedTypes: ["application/pdf"],
			},
		}),
		"GET /api/notes": {
			body: [
				note("taxes", { title: "Taxes", archived: true }),
				note("garden", { title: "Garden" }),
			],
		},
		...Object.fromEntries(
			list.flatMap((a) => [
				[
					`PATCH /api/attachments/${a.id}`,
					({ body }: { body: unknown }) => {
						const { keep } = body as { keep: boolean };
						Object.assign(a, {
							keep,
							deletedAt: keep ? a.deletedAt : null,
							deleteAt: null,
						});
						return { body: a };
					},
				],
				[
					`DELETE /api/attachments/${a.id}`,
					() => {
						list.splice(list.indexOf(a), 1);
						return { status: 204 };
					},
				],
			]),
		),
	});
	return { api, list };
}

/** The entry of the attachment named name. */
async function entry(name: string) {
	const link = await screen.findByText(name);
	const li = link.closest("li");
	if (!li) throw new Error(`no entry for ${name}`);
	return li;
}

describe("attachments page", () => {
	it("lists the attachments with their notes, size and status", async () => {
		server();
		renderApp("/attachments");
		expect(
			await screen.findByRole("heading", { name: "Attachments" }),
		).toBeInTheDocument();
		expect(
			await screen.findByText("3 attachments · 2.5 MB"),
		).toBeInTheDocument();
		await waitFor(() => expect(document.title).toBe("Attachments - Boardly"));

		const names = screen
			.getAllByRole("listitem")
			.filter((li) => li.parentElement?.parentElement?.tagName === "SECTION")
			.map((li) => li.querySelector("a, button")?.textContent);
		expect(names).toEqual(["report.pdf", "old.png", "kept.pdf"]);

		const taxes = within(await entry("report.pdf")).getByRole("link", {
			name: "Taxes",
		});
		expect(taxes).toHaveAttribute("href", "/notes/taxes");
		expect(
			within(await entry("report.pdf")).getByText("Archived"),
		).toBeVisible();

		const marked = await entry("old.png");
		expect(
			within(marked).getByText("Not used in any note"),
		).toBeInTheDocument();
		expect(
			within(marked).getByText("Will be deleted on Oct 31, 2026"),
		).toBeInTheDocument();

		expect(
			within(await entry("kept.pdf")).getByText(/Kept · marked on Oct 1, 2026/),
		).toBeInTheDocument();
	});

	it("goes to the note from its link", async () => {
		server();
		const { router } = renderApp("/attachments");
		fireEvent.click(
			within(await entry("report.pdf")).getByRole("link", { name: "Taxes" }),
		);
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/notes/taxes"),
		);
	});

	it("sends signed-out visitors to /login", async () => {
		mockApi({ "GET /api/me": signedOut });
		const { router } = renderApp("/attachments");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("filters by use, mark and keep", async () => {
		server();
		const { router } = renderApp("/attachments");
		const unused = await screen.findByRole("button", { name: "Unused 2" });
		fireEvent.click(unused);
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ filter: "unused" }),
		);
		expect(unused).toHaveAttribute("aria-pressed", "true");
		expect(screen.queryByText("report.pdf")).not.toBeInTheDocument();
		expect(screen.getByText("old.png")).toBeInTheDocument();
		expect(screen.getByText("kept.pdf")).toBeInTheDocument();

		fireEvent.click(
			screen.getByRole("button", { name: "Marked for deletion 1" }),
		);
		await waitFor(() => expect(screen.queryByText("kept.pdf")).toBeNull());
		expect(screen.getByText("old.png")).toBeInTheDocument();

		fireEvent.click(screen.getByRole("button", { name: "Kept 1" }));
		await waitFor(() => expect(screen.queryByText("old.png")).toBeNull());
		expect(screen.getByText("kept.pdf")).toBeInTheDocument();
	});

	it("shows everything for an unknown filter", async () => {
		server();
		renderApp("/attachments?filter=nonsense");
		expect(
			await screen.findByRole("button", { name: "All 3" }),
		).toHaveAttribute("aria-pressed", "true");
		expect(screen.getByText("report.pdf")).toBeInTheDocument();
	});

	it("says when a filter has nothing and when there are no attachments", async () => {
		server([report]);
		renderApp("/attachments?filter=kept");
		expect(await screen.findByText("No attachments here.")).toBeInTheDocument();
	});

	it("shows an empty page without attachments", async () => {
		server([]);
		renderApp("/attachments");
		expect(await screen.findByText("No attachments yet")).toBeInTheDocument();
	});

	it("keeps a marked attachment", async () => {
		const { api } = server();
		renderApp("/attachments");
		const marked = await entry("old.png");
		fireEvent.click(within(marked).getByRole("button", { name: "Keep" }));
		expect(await within(marked).findByText(/^Kept/)).toBeInTheDocument();
		expect(api.callsTo("PATCH", "/api/attachments/a-old")[0].body).toEqual({
			keep: true,
		});
		expect(
			within(marked).getByRole("button", { name: "Don't keep" }),
		).toBeInTheDocument();
	});

	it("says when keeping fails", async () => {
		const { api } = server();
		api.set({ "PATCH /api/attachments/a-old": apiError(500, "internal") });
		renderApp("/attachments");
		const marked = await entry("old.png");
		fireEvent.click(within(marked).getByRole("button", { name: "Keep" }));
		expect(await within(marked).findByRole("alert")).toHaveTextContent(
			"The change couldn't be saved.",
		);
	});

	it("deletes an unused attachment after confirmation", async () => {
		server();
		renderApp("/attachments");
		const marked = await entry("old.png");
		fireEvent.click(
			within(marked).getByRole("button", { name: "Delete old.png" }),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent("“old.png” will be deleted.");
		fireEvent.click(within(dialog).getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(screen.queryByText("old.png")).toBeNull());
		expect(screen.getByText("2 attachments · 2 MB")).toBeInTheDocument();
	});

	it("doesn't delete an attachment in use", async () => {
		server();
		renderApp("/attachments");
		const used = await entry("report.pdf");
		const remove = within(used).getByRole("button", {
			name: "Delete report.pdf",
		});
		expect(remove).toBeDisabled();
		expect(remove).toHaveAccessibleDescription("Still used in notes");
	});
});
