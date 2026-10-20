import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

describe("home page", () => {
	it("greets a user by name and hides the landing page", async () => {
		mockApi({ "GET /api/me": signedIn({ name: "Ann" }) });
		renderApp("/");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Welcome, Ann" }),
		).toBeVisible();
		expect(screen.getByText("Your dashboards will appear here")).toBeVisible();
		expect(screen.queryByRole("heading", { name: "Famodash" })).toBeNull();
		expect(screen.queryByText("Screenshot coming soon")).toBeNull();
	});

	it("greets a user without a name by email", async () => {
		mockApi({ "GET /api/me": signedIn({ email: "bob@example.org" }) });
		renderApp("/");
		expect(
			await screen.findByRole("heading", {
				level: 1,
				name: "Welcome, bob@example.org",
			}),
		).toBeVisible();
	});
});
