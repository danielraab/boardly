import { readFileSync } from "node:fs";
import {
	fireEvent,
	render,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { NoteDoc } from "./document/types";
import { NoteRenderer } from "./NoteRenderer";

const allTypes: NoteDoc = JSON.parse(
	readFileSync(
		"../backend/internal/notes/testdata/docs/valid/all-types.json",
		"utf8",
	),
);

describe("NoteRenderer", () => {
	it("renders every block type", () => {
		const { container } = render(<NoteRenderer doc={allTypes} />);
		expect(screen.getByRole("heading", { name: "This week" }).tagName).toBe(
			"H3",
		);
		expect(screen.getByText("bold")).toHaveClass("font-semibold");
		expect(screen.getByText("italic")).toHaveClass("italic");
		expect(screen.getByText("gone")).toHaveClass("line-through");
		expect(screen.getByText("both")).toHaveClass("font-semibold", "italic");
		expect(container.querySelector("blockquote")).toHaveTextContent("A quote");
		expect(container.querySelector("pre")).toHaveTextContent(
			"go test ./... make check",
		);
		expect(container.querySelector("hr")).not.toBeNull();
		expect(container.querySelector("ol")).toHaveTextContent("two a");
		expect(container.querySelector("br")).not.toBeNull();
	});

	it("shows task items as disabled checkboxes in their state", () => {
		render(<NoteRenderer doc={allTypes} />);
		const review = screen.getByRole("checkbox", { name: "review PR" });
		expect(review).not.toBeChecked();
		expect(review).toBeDisabled();
		expect(screen.getByRole("checkbox", { name: "deploy" })).toBeChecked();
		expect(screen.getByText("deploy")).toHaveClass("line-through");
		expect(
			screen.getByRole("checkbox", { name: "staging" }),
		).toBeInTheDocument();
	});

	it("toggles task items with onToggleTask", () => {
		const onToggleTask = vi.fn();
		const outer = vi.fn();
		render(
			// biome-ignore lint/a11y/noStaticElementInteractions: only listens for bubbling clicks
			// biome-ignore lint/a11y/useKeyWithClickEvents: only listens for bubbling clicks
			<div onClick={outer}>
				<NoteRenderer doc={allTypes} onToggleTask={onToggleTask} />
			</div>,
		);
		const review = screen.getByRole("checkbox", { name: "review PR" });
		expect(review).toBeEnabled();
		expect(review).not.toHaveAttribute("tabindex");
		fireEvent.click(review);
		expect(outer).not.toHaveBeenCalled();
		expect(onToggleTask).toHaveBeenCalledWith("t1", true);
		fireEvent.click(screen.getByRole("checkbox", { name: "deploy" }));
		expect(onToggleTask).toHaveBeenLastCalledWith(expect.any(String), false);
	});

	it("sets the text size from a widget size, and text-sm without one", () => {
		const root = (size?: "S" | "M" | "L") =>
			render(<NoteRenderer doc={allTypes} size={size} />).container
				.firstElementChild;
		expect(root()).toHaveClass("text-sm");
		expect(root("S")).toHaveClass("text-xs");
		expect(root("M")).toHaveClass("text-sm");
		expect(root("L")).toHaveClass("text-base");
	});

	it("doesn't render links as links", () => {
		render(<NoteRenderer doc={allTypes} />);
		expect(screen.queryByRole("link")).toBeNull();
		expect(screen.getByText("docs")).toBeInTheDocument();
	});
});

describe("NoteRenderer attachments", () => {
	const roses = {
		id: "a1",
		name: "roses.jpg",
		contentType: "image/jpeg",
		size: 2048,
		url: "/api/attachments/a1/roses.jpg",
	};
	const tulips = { ...roses, id: "a2", name: "tulips.jpg", url: "/t.jpg" };
	const plan = {
		id: "a3",
		name: "report.pdf",
		contentType: "application/pdf",
		size: 300 * 1024,
		url: "/api/attachments/a3/report.pdf",
	};
	const doc: NoteDoc = {
		v: 1,
		blocks: [
			{ type: "attachment", id: "a1" },
			{ type: "attachment", id: "a3" },
			{ type: "attachment", id: "a2" },
			{ type: "attachment", id: "gone" },
		],
	};

	it("shows images, file cards and missing attachments", () => {
		render(
			<NoteRenderer
				doc={doc}
				attachments={[roses, plan, tulips]}
				interactiveAttachments
			/>,
		);
		expect(screen.getByRole("img", { name: "roses.jpg" })).toHaveAttribute(
			"src",
			roses.url,
		);
		expect(screen.getByText("report.pdf")).toHaveTextContent(
			"report.pdf · 300 KB",
		);
		expect(
			screen.getByRole("link", { name: "Download report.pdf" }),
		).toHaveAttribute("href", plan.url);
		expect(screen.getByText("Attachment missing")).toBeInTheDocument();
	});

	it("is inert on cards", () => {
		render(<NoteRenderer doc={doc} attachments={[roses, plan, tulips]} />);
		expect(
			screen.queryByRole("button", { name: "View roses.jpg" }),
		).not.toBeInTheDocument();
		expect(
			screen.queryByRole("link", { name: "Download report.pdf" }),
		).not.toBeInTheDocument();
	});

	it("opens images in the viewer and moves between them", async () => {
		render(
			<NoteRenderer
				doc={doc}
				attachments={[roses, plan, tulips]}
				interactiveAttachments
			/>,
		);
		const open = screen.getByRole("button", { name: "View roses.jpg" });
		open.focus();
		fireEvent.click(open);
		const viewer = await screen.findByRole("dialog", { name: "roses.jpg" });
		expect(
			within(viewer).getByRole("link", { name: "Download roses.jpg" }),
		).toHaveAttribute("href", roses.url);

		fireEvent.keyDown(viewer, { key: "ArrowRight" });
		expect(
			await screen.findByRole("dialog", { name: "tulips.jpg" }),
		).toBeInTheDocument();
		fireEvent.click(screen.getByRole("button", { name: "Next" }));
		expect(
			await screen.findByRole("dialog", { name: "roses.jpg" }),
		).toBeInTheDocument();

		fireEvent.keyDown(document.activeElement ?? document.body, {
			key: "Escape",
		});
		await waitFor(() =>
			expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
		);
		await waitFor(() => expect(open).toHaveFocus());
	});
});
