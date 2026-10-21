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

describe("settings profile slug", () => {
	it("changes the slug after confirmation and updates the cache", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": ({ body }) =>
				signedIn({ slug: (body as { slug: string }).slug }),
		});
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		expect(input).toHaveValue("ann");
		await user.clear(input);
		await user.type(input, "AnnSmith");
		expect(input).toHaveValue("annsmith");
		expect(
			screen.getByText("Links that use the old address will stop working."),
		).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog", {
			name: "Change address?",
		});
		expect(within(confirm).getByText("/ann/…")).toBeVisible();
		expect(within(confirm).getByText("/annsmith/…")).toBeVisible();
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);

		expect(
			await screen.findByText("Your dashboards are at /annsmith/…"),
		).toBeVisible();
		expect(api.callsTo("PATCH", "/api/me")[0].body).toEqual({
			slug: "annsmith",
		});
	});

	it("saves nothing when the confirmation is cancelled", async () => {
		const api = mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.type(input, "2");
		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog");
		await user.click(within(confirm).getByRole("button", { name: "Cancel" }));

		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(input).toHaveValue("ann2");
		expect(api.callsTo("PATCH", "/api/me")).toHaveLength(0);
	});

	it("blocks reserved words", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.clear(input);
		await user.type(input, "login");
		expect(screen.getByRole("alert")).toHaveTextContent(
			"This address is reserved by Famodash.",
		);
		expect(screen.getByRole("button", { name: "Save address" })).toBeDisabled();
	});

	it("shows a taken slug at the field", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"PATCH /api/me": apiError(409, "slug_taken"),
		});
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Address");
		await user.clear(input);
		await user.type(input, "bob");
		await user.click(screen.getByRole("button", { name: "Save address" }));
		const confirm = await screen.findByRole("alertdialog");
		await user.click(
			within(confirm).getByRole("button", { name: "Change address" }),
		);
		expect(
			await screen.findByText("This address is already taken."),
		).toBeVisible();
		expect(input).toHaveValue("bob");
	});
});
