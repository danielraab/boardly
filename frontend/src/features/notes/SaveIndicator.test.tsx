import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { SaveIndicator } from "./SaveIndicator";

const noop = () => {};

describe("SaveIndicator", () => {
	it("keeps the text as the status when compact", () => {
		render(<SaveIndicator status="saved" onRetry={noop} compact />);
		expect(screen.getByRole("status")).toHaveTextContent("Saved");
		expect(screen.getByRole("status").querySelector("svg")).not.toBeNull();
	});

	it("shows no icon unless compact", () => {
		render(<SaveIndicator status="saving" onRetry={noop} />);
		expect(screen.getByRole("status")).toHaveTextContent("Saving…");
		expect(screen.getByRole("status").querySelector("svg")).toBeNull();
	});

	it("offers Reload for a conflict", async () => {
		const onReload = vi.fn();
		render(
			<SaveIndicator
				status="conflict"
				onRetry={noop}
				onReload={onReload}
				compact
			/>,
		);
		expect(screen.getByRole("status")).toHaveTextContent("Changed elsewhere");
		await userEvent.click(screen.getByRole("button", { name: "Reload" }));
		expect(onReload).toHaveBeenCalledOnce();
	});

	it("shows nothing for a conflict without onReload", () => {
		render(<SaveIndicator status="conflict" onRetry={noop} />);
		expect(screen.queryByRole("status")).toBeNull();
	});
});
