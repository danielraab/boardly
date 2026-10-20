import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

async function openMenuText(user: ReturnType<typeof renderApp>["user"]) {
	await user.click(screen.getByRole("button", { name: "User menu" }));
	const menu = await screen.findByRole("menu");
	const text = menu.textContent ?? "";
	await user.keyboard("{Escape}");
	await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	return text;
}

describe("settings profile tab", () => {
	it("shows the email and an empty name field", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		expect(
			await screen.findByRole("heading", { name: "Settings" }),
		).toBeVisible();
		expect(screen.getByText("ann@example.org")).toBeVisible();
		expect(screen.getByLabelText("Display name")).toHaveValue("");
		expect(screen.getByRole("button", { name: "Save" })).toBeDisabled();
	});

	it("sets the name and updates the menu without a reload", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": ({ body }) =>
				signedIn({ name: (body as { name: string }).name.trim() }),
		});
		const { user } = renderApp("/settings");

		await user.type(await screen.findByLabelText("Display name"), "Ann Smith");
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(screen.getByLabelText("Display name")).toHaveValue("Ann Smith");
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			name: "Ann Smith",
		});
		expect(await openMenuText(user)).toContain("Ann Smith");
		expect(api.callsTo("GET", "/api/me")).toHaveLength(1);
	});

	it("clears the name with null", async () => {
		const api = mockApi({
			"GET /api/me": signedIn({ name: "Ann" }),
			"PATCH /api/me": signedIn({ name: null }),
		});
		const { user } = renderApp("/settings");

		const input = await screen.findByLabelText("Display name");
		expect(input).toHaveValue("Ann");
		await user.clear(input);
		await user.click(screen.getByRole("button", { name: "Save" }));

		expect(await screen.findByText("Saved.")).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({ name: null });
		const menu = await openMenuText(user);
		expect(menu).not.toContain("Ann ");
		expect(menu.startsWith("ann@example.org")).toBe(true);
	});

	it("explains a name that is too long", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": apiError(400, "invalid_name"),
		});
		const { user } = renderApp("/settings");
		await user.type(
			await screen.findByLabelText("Display name"),
			"x".repeat(101),
		);
		await user.click(screen.getByRole("button", { name: "Save" }));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The name can be at most 100 characters long.",
		);
		expect(within(document.body).queryByText("Saved.")).toBeNull();
	});
});
