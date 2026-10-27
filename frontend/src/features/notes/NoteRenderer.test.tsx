import { readFileSync } from "node:fs";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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

	it("doesn't render links as links", () => {
		render(<NoteRenderer doc={allTypes} />);
		expect(screen.queryByRole("link")).toBeNull();
		expect(screen.getByText("docs")).toBeInTheDocument();
	});
});
