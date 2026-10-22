import { screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderApp } from "#/test/renderApp";
import { features } from "./features";

describe("landing page", () => {
	it("shows the hero with name, tagline and a sign-in call to action", async () => {
		renderApp("/");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Boardly" }),
		).toBeVisible();
		expect(
			screen.getByText(
				"Your favourites and dashboards, self-hosted and in one place.",
			),
		).toBeVisible();
		expect(screen.getByRole("link", { name: "Sign in" })).toHaveAttribute(
			"href",
			"/login",
		);
	});

	it("shows the screenshot placeholder", async () => {
		renderApp("/");
		expect(await screen.findByText("Screenshot coming soon")).toBeVisible();
	});

	it("renders one card per feature with translated texts", async () => {
		renderApp("/");
		const list = await screen.findByRole("list");
		const items = within(list).getAllByRole("listitem");
		expect(items).toHaveLength(features.length);
		expect(
			within(items[0]).getByRole("heading", { name: "Dashboards" }),
		).toBeVisible();
	});
});
