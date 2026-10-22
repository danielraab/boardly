import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ImageIcon } from "./ImageIcon";

describe("ImageIcon", () => {
	it("shows the image, and a globe when there is none or it fails", () => {
		const { container, rerender } = render(
			<ImageIcon imageId="a" className="size-4" />,
		);
		const img = container.querySelector("img");
		expect(img).toHaveAttribute("src", "/api/images/a");

		fireEvent.error(img as HTMLImageElement);
		expect(container.querySelector("img")).toBeNull();
		expect(container.querySelector("[data-icon='globe']")).not.toBeNull();

		// Another image gets its own chance.
		rerender(<ImageIcon imageId="b" className="size-4" />);
		expect(container.querySelector("img")).toHaveAttribute(
			"src",
			"/api/images/b",
		);

		rerender(<ImageIcon imageId={null} className="size-4" />);
		expect(container.querySelector("[data-icon='globe']")).not.toBeNull();
	});
});
