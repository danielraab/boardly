import { DRAG_DROP_PASTE } from "@lexical/rich-text";
import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { $getRoot, $isElementNode } from "lexical";
import { describe, expect, it, vi } from "vitest";
import type { NoteAttachment } from "#/features/attachments/api";
import { apiError, type MockResponse } from "#/test/api";
import { editorOf } from "#/test/lexical";
import { fakeNotesServer, note, paragraphs } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { Block } from "./document/types";

const attachment = (
	id: string,
	name: string,
	contentType = "application/pdf",
): NoteAttachment => ({
	id,
	name,
	contentType,
	size: 2048,
	url: `/api/attachments/${id}/${name}`,
});

const roses = attachment("a-roses", "roses.jpg", "image/jpeg");
const plan = attachment("a-plan", "plan.pdf");

/**
 * Opens the note n1 with the blocks and attachments, on a server that stores
 * uploads as attachments, or answers upload with its response.
 */
async function openNote(
	blocks: Block[],
	attachments: NoteAttachment[] = [],
	upload?: MockResponse,
) {
	let next = 1;
	const server = fakeNotesServer(
		[note("n1", { content: { v: 1, blocks }, attachments })],
		[],
		{
			"POST /api/attachments": ({ url }) => {
				if (upload) return upload;
				const name = url.searchParams.get("name") ?? "";
				const type = name.endsWith(".png") ? "image/png" : "application/pdf";
				return { status: 201, body: attachment(`up${next++}`, name, type) };
			},
			"GET /api/attachments": {
				body: {
					attachments: [],
					maxSize: 5 * 1024 * 1024,
					allowedTypes: ["image/png", "application/pdf"],
				},
			},
		},
	);
	const app = renderApp("/notes/n1");
	const content = await screen.findByRole("textbox", { name: "Content" });
	return { ...server, ...app, content };
}

const file = (name: string) => new File(["x"], name);

/** Chooses files in the toolbar's file picker. */
function chooseFiles(...files: File[]) {
	fireEvent.change(screen.getByTestId("attachment-input"), {
		target: { files },
	});
}

/** The blocks of the saved note, once the save after version is done. */
async function savedBlocks(
	server: { notes: { version: number; content: { blocks: Block[] } }[] },
	version: number,
) {
	await waitFor(() => expect(server.notes[0].version).toBe(version), {
		timeout: 3000,
	});
	return server.notes[0].content.blocks;
}

const text = (t: string): Block => ({
	type: "paragraph",
	content: [{ type: "text", text: t }],
});

describe("attachments button", () => {
	it("lists the note's attachments once each and shows their number", async () => {
		await openNote(
			[
				{ type: "attachment", id: roses.id },
				{ type: "attachment", id: plan.id },
				{ type: "attachment", id: roses.id },
			],
			[roses, plan],
		);
		const button = screen.getByRole("button", { name: "Attachments" });
		expect(button).toHaveTextContent("2");
		fireEvent.click(button);
		const menu = await screen.findByRole("menu");
		const items = within(menu)
			.getAllByRole("menuitem")
			.map((i) => i.textContent);
		expect(items).toEqual(["roses.jpg2 KB", "plan.pdf2 KB"]);
		expect(within(menu).queryByRole("separator")).not.toBeInTheDocument();
	});

	it("says when there are none", async () => {
		await openNote(paragraphs("Garden").blocks);
		fireEvent.click(screen.getByRole("button", { name: "Attachments" }));
		expect(await screen.findByText("No attachments")).toBeInTheDocument();
	});

	it("no longer lists an attachment whose block was deleted", async () => {
		const server = await openNote(
			[text("Taxes"), { type: "attachment", id: plan.id }],
			[plan],
		);
		await act(async () => {
			editorOf(server.content).update(
				() => $getRoot().getLastChild()?.remove(),
				{
					discrete: true,
				},
			);
		});
		expect(await savedBlocks(server, 2)).toEqual([text("Taxes")]);
		fireEvent.click(screen.getByRole("button", { name: "Attachments" }));
		expect(await screen.findByText("No attachments")).toBeInTheDocument();
	});

	it("opens images in the viewer", async () => {
		await openNote([{ type: "attachment", id: roses.id }], [roses]);
		fireEvent.click(screen.getByRole("button", { name: "Attachments" }));
		fireEvent.click(
			await screen.findByRole("menuitem", { name: /roses\.jpg/ }),
		);
		expect(
			await screen.findByRole("dialog", { name: "roses.jpg" }),
		).toBeInTheDocument();
	});
});

describe("adding attachments", () => {
	it("offers the button in the toolbar right after the link", async () => {
		await openNote([text("Garden")]);
		const buttons = within(screen.getByRole("toolbar"))
			.getAllByRole("button")
			.map((b) => b.getAttribute("aria-label"));
		const link = buttons.indexOf("Link");
		expect(buttons.slice(link, link + 2)).toEqual(["Link", "Add attachment"]);
	});

	it("opens the file picker with the allowed types", async () => {
		await openNote([text("Garden")]);
		const input = screen.getByTestId("attachment-input");
		const click = vi.spyOn(input, "click");
		fireEvent.click(screen.getByRole("button", { name: "Add attachment" }));
		expect(click).toHaveBeenCalled();
		await waitFor(() =>
			expect(input).toHaveAttribute("accept", "image/png,application/pdf"),
		);
	});

	it("inserts them at the end when the editor never had a cursor", async () => {
		const server = await openNote([text("Garden")]);
		chooseFiles(file("a.pdf"), file("b.pdf"));
		expect(await savedBlocks(server, 2)).toEqual([
			text("Garden"),
			{ type: "attachment", id: "up1" },
			{ type: "attachment", id: "up2" },
		]);
		expect(await screen.findByText("b.pdf")).toBeInTheDocument();
		expect(
			screen.getByRole("button", { name: "Attachments" }),
		).toHaveTextContent("2");
	});

	it("inserts them at the cursor", async () => {
		const server = await openNote([text("first"), text("second")]);
		const editor = editorOf(server.content);
		await act(async () => {
			editor.update(
				() => {
					const first = $getRoot().getFirstChild();
					if ($isElementNode(first)) first.selectEnd();
				},
				{ discrete: true },
			);
		});
		chooseFiles(file("plan.pdf"));
		expect(await savedBlocks(server, 2)).toEqual([
			text("first"),
			{ type: "attachment", id: "up1" },
			text("second"),
		]);
	});

	it("uploads pasted and dropped files", async () => {
		const server = await openNote([]);
		await act(async () => {
			editorOf(server.content).dispatchCommand(DRAG_DROP_PASTE, [
				file("screenshot.png"),
			]);
		});
		expect(await savedBlocks(server, 2)).toContainEqual({
			type: "attachment",
			id: "up1",
		});
		expect(
			await screen.findByRole("img", { name: "screenshot.png" }),
		).toBeInTheDocument();
	});

	it("says why a file wasn't added", async () => {
		const server = await openNote(
			[text("Garden")],
			[],
			apiError(413, "attachment_too_large"),
		);
		chooseFiles(file("video.mp4"));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"video.mp4 is larger than 5 MB and wasn't added.",
		);
		expect(screen.queryByRole("status")).not.toBeInTheDocument();
		expect(server.notes[0].content.blocks).toEqual([text("Garden")]);
	});

	it("says when a type isn't allowed", async () => {
		await openNote([], [], apiError(400, "unsupported_attachment_type"));
		chooseFiles(file("notes.zip"));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"notes.zip wasn't added: this type of file can't be attached.",
		);
	});
});
