import { readFileSync } from "node:fs";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AppIcon, STAR_COLOR } from "./AppIcon";

const favicon = new DOMParser().parseFromString(
	readFileSync("public/favicon.svg", "utf8"),
	"image/svg+xml",
);

function shapes(root: ParentNode) {
	return [...root.querySelectorAll("path, rect, circle")].map((el) =>
		["d", "x", "y", "width", "height", "rx", "cx", "cy", "r"]
			.map((name) => el.getAttribute(name))
			.join(" "),
	);
}

describe("AppIcon", () => {
	it("is decorative and drawn in the text colour with a rose star", () => {
		const { container } = render(<AppIcon className="size-6" />);
		const svg = container.querySelector("svg") as SVGElement;
		expect(svg).toHaveAttribute("aria-hidden", "true");
		expect(svg.querySelector("g")).toHaveAttribute("stroke", "currentColor");
		expect(svg.lastElementChild).toHaveAttribute("fill", STAR_COLOR);
	});

	it("gives each icon its own mask", () => {
		const { container } = render(
			<>
				<AppIcon className="size-6" />
				<AppIcon className="size-6" />
			</>,
		);
		const ids = [...container.querySelectorAll("mask")].map((m) => m.id);
		expect(new Set(ids).size).toBe(2);
	});

	it("matches the drawing in favicon.svg", () => {
		const { container } = render(<AppIcon className="size-6" />);
		expect(shapes(favicon)).toEqual(shapes(container));
	});

	it("draws the favicon outline dark in a light and light in a dark browser", () => {
		const style = favicon.querySelector("style")?.textContent ?? "";
		expect(style).toMatch(/^svg\{color:#18181b\}/);
		expect(style).toContain(
			"@media (prefers-color-scheme:dark){svg{color:#f4f4f5}}",
		);
	});
});
