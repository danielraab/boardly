import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { type IconChoice, IconChooser } from "./IconChooser";

function Chooser({ withFavicon }: { withFavicon: boolean }) {
	const [value, setValue] = useState<IconChoice>({
		source: "none",
		imageId: null,
		icon: null,
	});
	return (
		<IconChooser
			value={value}
			onChange={setValue}
			favicon={withFavicon ? { id: null, loading: false } : undefined}
			host={withFavicon ? "example.org" : null}
		/>
	);
}

describe("IconChooser", () => {
	it("offers the website icon tab when given a favicon", async () => {
		render(<Chooser withFavicon />);
		fireEvent.click(screen.getByRole("button", { name: /^Icon:/ }));
		const tabs = await screen.findAllByRole("tab");
		expect(tabs.map((t) => t.textContent)).toEqual([
			"Website icon",
			"Own image",
			"Symbol",
		]);
	});

	it("without a favicon offers only own image and symbol, starting on symbol", async () => {
		render(<Chooser withFavicon={false} />);
		fireEvent.click(screen.getByRole("button", { name: /^Icon:/ }));
		const tabs = await screen.findAllByRole("tab");
		expect(tabs.map((t) => t.textContent)).toEqual(["Own image", "Symbol"]);
		expect(screen.getByRole("tab", { name: "Symbol" })).toHaveAttribute(
			"aria-selected",
			"true",
		);
		fireEvent.click(screen.getByRole("radio", { name: "Home" }));
		expect(
			screen.getByRole("button", { name: "Icon: Home" }),
		).toBeInTheDocument();
	});
});
