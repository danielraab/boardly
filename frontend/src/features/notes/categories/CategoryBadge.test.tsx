import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { ReadOnlyCategoryBadge } from "./CategoryBadge";

describe("ReadOnlyCategoryBadge", () => {
	it("shows the category with its tooltip and opens nothing", async () => {
		render(
			<ReadOnlyCategoryBadge
				category={{ title: "Garden", icon: null, color: "green" }}
				tooltip="Home › Garden"
			/>,
		);
		const badge = screen.getByText("Garden").parentElement as HTMLElement;
		expect(badge).toHaveAttribute("title", "Home › Garden");
		expect(screen.queryByRole("button")).toBeNull();
		await userEvent.click(badge);
		expect(screen.queryByRole("menu")).toBeNull();
	});
});
