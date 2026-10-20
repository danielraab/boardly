import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderApp } from "#/test/renderApp";

describe("app shell", () => {
	it.each([
		"/",
		"/login",
		"/no-such-page",
	])("shows the header on %s", async (path) => {
		renderApp(path);
		expect(
			await screen.findByRole("link", { name: "Famodash" }),
		).toHaveAttribute("href", "/");
		expect(screen.getByRole("button", { name: "Language" })).toBeVisible();
		expect(screen.getByRole("button", { name: "User menu" })).toBeVisible();
	});

	it("shows a not-found page with a link home for unknown paths", async () => {
		renderApp("/no-such-page");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeVisible();
		expect(
			screen.getByRole("link", { name: "Back to the start page" }),
		).toHaveAttribute("href", "/");
	});
});

describe("user menu", () => {
	it("opens with the keyboard, closes on Escape and returns focus", async () => {
		const { user } = renderApp("/");
		const avatar = await screen.findByRole("button", { name: "User menu" });

		avatar.focus();
		await user.keyboard("{Enter}");
		expect(await screen.findByRole("menu")).toBeVisible();

		await user.keyboard("{Escape}");
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
		expect(avatar).toHaveFocus();
	});

	it("closes on an outside click", async () => {
		const { user } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "User menu" }));
		expect(await screen.findByRole("menu")).toBeVisible();

		await user.click(document.body);
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});

	it("offers only 'Sign in' when signed out, which goes to /login", async () => {
		const { user, router } = renderApp("/");
		await user.click(await screen.findByRole("button", { name: "User menu" }));

		const items = await screen.findAllByRole("menuitem");
		expect(items.map((i) => i.textContent)).toEqual(["Sign in"]);

		await user.click(items[0]);
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		await waitFor(() => expect(screen.queryByRole("menu")).toBeNull());
	});
});

describe("language switcher", () => {
	it("switches all texts and the lang attribute without reload", async () => {
		const { user } = renderApp("/");
		expect(await screen.findByRole("link", { name: "Sign in" })).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Language" }));
		await user.click(
			await screen.findByRole("menuitemradio", { name: "Deutsch" }),
		);

		expect(await screen.findByRole("link", { name: "Anmelden" })).toBeVisible();
		expect(screen.getByRole("button", { name: "Benutzermenü" })).toBeVisible();
		expect(document.documentElement.lang).toBe("de");
	});
});
