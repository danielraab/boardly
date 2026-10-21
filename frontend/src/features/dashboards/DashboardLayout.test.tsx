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
