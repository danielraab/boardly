import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { mockApi, signedIn } from "#/test/api";
import { stubColorScheme } from "#/test/colorScheme";
import { section, viewData } from "#/test/dashboards";
import { renderApp } from "#/test/renderApp";

const frame = (id: string) => screen.getByTestId(`section-${id}`);

describe("dashboard layout", () => {
	it("uses the desktop or the mobile column count", async () => {
		const viewport = stubColorScheme();
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([section("Wide", { width: 4 })], { isOwner: false }),
			},
		});
		renderApp("/ann/home");
		expect(
			await screen.findByRole("heading", { level: 2, name: "Wide" }),
		).toBeVisible();
		expect(frame("Wide")).toHaveAttribute("data-span", "4");

		viewport.setWidth(600);
		await screen.findByTestId("section-Wide");
		expect(frame("Wide")).toHaveAttribute("data-span", "2");
	});

	it("never renders a section wider than its parent", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([
					section(
						"Outer",
						{ width: 3 },
						section("Inner", { width: 6 }),
						section("Small", { width: 1 }),
					),
				]),
			},
		});
		renderApp("/ann/home");
		await screen.findByRole("heading", { level: 3, name: "Inner" });
		expect(frame("Inner")).toHaveAttribute("data-span", "3");
		expect(frame("Small")).toHaveAttribute("data-span", "1");
	});

	it("renders borders, titles and fixed heights", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([
					section("Bordered"),
					section("Plain", { border: false, title: null, height: 3 }),
				]),
			},
		});
		renderApp("/ann/home");
		await screen.findByRole("heading", { name: "Bordered" });
		expect(frame("Bordered")).toHaveClass("border");
		expect(frame("Plain")).not.toHaveClass("border");
		expect(frame("Plain")).toHaveClass("overflow-y-auto");
		expect(frame("Plain").style.height).toBe(
			"calc(3 * var(--dashboard-row) + 2 * var(--dashboard-gap))",
		);
		// Without a title there is no heading, and no edit controls outside edit mode.
		expect(screen.getAllByRole("heading", { level: 2 })).toHaveLength(1);
		expect(screen.queryByRole("button", { name: /Move/ })).toBeNull();
		expect(screen.queryByRole("button", { name: "Add section" })).toBeNull();
	});

	it("collapses titled sections and remembers them in this browser", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData(
					[
						section("News", { height: 3 }, section("Inner")),
						section("Tools"),
						section("Plain", { title: null }, section("Child")),
					],
					{ isOwner: false },
				),
			},
		});
		const { user } = renderApp("/ann/home");
		const collapse = await screen.findByRole("button", {
			name: "Collapse News",
		});
		expect(collapse).toHaveAttribute("aria-expanded", "true");
		// Only the untitled section "Plain" has no toggle.
		expect(screen.getAllByRole("button", { name: /^Collapse/ })).toHaveLength(
			4,
		);

		await user.click(collapse);
		const expand = screen.getByRole("button", { name: "Expand News" });
		expect(expand).toHaveAttribute("aria-expanded", "false");
		expect(screen.queryByTestId("section-Inner")).toBeNull();
		expect(frame("News").style.height).toBe("");
		expect(frame("News")).not.toHaveClass("overflow-y-auto");
		expect(frame("Tools")).toBeVisible();
		expect(
			JSON.parse(localStorage.getItem("boardly.collapsedSections") ?? ""),
		).toEqual(["News"]);

		await user.click(expand);
		expect(frame("Inner")).toBeVisible();
		expect(localStorage.getItem("boardly.collapsedSections")).toBe("[]");
	});

	it("restores collapsed sections after a reload", async () => {
		localStorage.setItem("boardly.collapsedSections", JSON.stringify(["News"]));
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([
					section("News", {}, section("Inner")),
					section("Tools"),
				]),
			},
		});
		renderApp("/ann/home");
		expect(
			await screen.findByRole("button", { name: "Expand News" }),
		).toBeVisible();
		expect(screen.queryByTestId("section-Inner")).toBeNull();
		expect(
			screen.getByRole("button", { name: "Collapse Tools" }),
		).toBeVisible();
	});

	it("shows collapsed sections expanded in edit mode", async () => {
		localStorage.setItem("boardly.collapsedSections", JSON.stringify(["News"]));
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/u/ann/home": {
				body: viewData([section("News", {}, section("Inner"))]),
			},
		});
		renderApp("/ann/home?edit=true");
		expect(await screen.findByTestId("section-Inner")).toBeVisible();
		expect(
			screen.queryByRole("button", { name: /^(Expand|Collapse)/ }),
		).toBeNull();
		expect(localStorage.getItem("boardly.collapsedSections")).toBe('["News"]');
	});

	it("tells the owner how to start an empty dashboard", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/u/ann/home": { body: viewData([]) },
		});
		renderApp("/ann/home");
		expect(
			await screen.findByText(
				"Open the ⋯ menu and choose “Edit layout” to add the first section.",
			),
		).toBeVisible();
		expect(screen.queryByRole("button", { name: "Add section" })).toBeNull();
	});

	it("shows visitors a plain empty placeholder", async () => {
		mockApi({
			"GET /api/u/ann/home": {
				body: viewData([], { isOwner: false, id: undefined }),
			},
		});
		renderApp("/ann/home");
		expect(
			await screen.findByText("Nothing has been added to this dashboard yet."),
		).toBeVisible();
	});
});
