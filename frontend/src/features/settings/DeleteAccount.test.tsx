import { act, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "#/i18n";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { closeCancelledNotice } from "./accountDeletion";

const NOW = new Date("2026-11-01T10:00:00Z");

function withWindow(secondsLeft: number, user = {}) {
	return signedIn({
		deletionWindowUntil: new Date(
			NOW.getTime() + secondsLeft * 1000,
		).toISOString(),
		...user,
	});
}

const scheduled = signedIn({ deletionScheduledAt: "2026-11-08T10:00:00Z" });

beforeEach(() => {
	vi.useFakeTimers({ toFake: ["Date", "setInterval", "clearInterval"] });
	vi.setSystemTime(NOW);
	closeCancelledNotice();
});

afterEach(() => {
	vi.useRealTimers();
});

async function section() {
	return within(await screen.findByRole("region", { name: "Delete account" }));
}

describe("delete account section", () => {
	it("offers the deletion with a countdown and the export hint", async () => {
		mockApi({ "GET /api/me": withWindow(120) });
		renderApp("/settings");
		const s = await section();
		expect(s.getByRole("timer")).toHaveTextContent("2:00");
		expect(s.getByRole("button", { name: "Delete account…" })).toBeVisible();
		expect(s.getByText(/export your data first/)).toBeVisible();
	});

	it("only explains the new sign-in when the window is closed", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		const s = await section();
		expect(
			s.getByText(/sign out and sign in again with an email link/),
		).toBeVisible();
		expect(s.queryByRole("button")).toBeNull();
		expect(s.queryByRole("timer")).toBeNull();
	});

	it("closes the window at zero without a reload", async () => {
		mockApi({ "GET /api/me": withWindow(3) });
		renderApp("/settings");
		const s = await section();
		expect(s.getByRole("timer")).toHaveTextContent("0:03");
		act(() => vi.advanceTimersByTime(3000));
		expect(s.queryByRole("button", { name: "Delete account…" })).toBeNull();
		expect(
			s.getByText(/sign out and sign in again with an email link/),
		).toBeVisible();
	});

	it("links the export hint to the Import / Export tab", async () => {
		mockApi({ "GET /api/me": withWindow(120) });
		const { user, router } = renderApp("/settings");
		const s = await section();
		await user.click(s.getByRole("link", { name: "Go to Import / Export" }));
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ tab: "data" }),
		);
		expect(
			screen.getByRole("tab", { name: "Import / Export" }),
		).toHaveAttribute("aria-selected", "true");
	});

	it("schedules the deletion after typing the email address", async () => {
		const api = mockApi({
			"GET /api/me": withWindow(120),
			"DELETE /api/me": scheduled,
		});
		const { user } = renderApp("/settings");
		const s = await section();
		await user.click(s.getByRole("button", { name: "Delete account…" }));

		const dialog = await screen.findByRole("dialog");
		const d = within(dialog);
		expect(dialog).toHaveTextContent("November 8, 2026");
		expect(dialog).toHaveTextContent("in 7 days");
		expect(dialog).toHaveTextContent("other devices are signed out");
		expect(dialog).toHaveTextContent("stay revoked even if you cancel");
		expect(dialog).toHaveTextContent("/ann/… becomes available to other users");
		expect(
			d.getByRole("link", { name: "Go to Import / Export" }),
		).toBeVisible();

		const confirm = d.getByRole("button", { name: "Delete account" });
		expect(confirm).toBeDisabled();
		const field = d.getByLabelText(/Type your email address/);
		await user.type(field, "bob@example.org");
		expect(confirm).toBeDisabled();
		await user.clear(field);
		await user.type(field, " ANN@example.org ");
		expect(confirm).toBeEnabled();
		await user.click(confirm);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(api.callsTo("DELETE", "/api/me")[0].body).toEqual({
			email: "ANN@example.org",
			locale: "en",
		});
		expect(
			await screen.findByText(
				/Your account will be deleted on November 8, 2026/,
				{
					selector: "span",
				},
			),
		).toBeVisible();
		expect(
			(await section()).getByRole("button", { name: "Cancel deletion" }),
		).toBeVisible();
	});

	it("sends the current language", async () => {
		await i18n.changeLanguage("de");
		const api = mockApi({
			"GET /api/me": withWindow(120),
			"DELETE /api/me": scheduled,
		});
		const { user } = renderApp("/settings");
		const s = within(
			await screen.findByRole("region", { name: "Konto löschen" }),
		);
		await user.click(s.getByRole("button", { name: "Konto löschen…" }));
		const d = within(await screen.findByRole("dialog"));
		await user.type(d.getByLabelText(/E-Mail-Adresse/), "ann@example.org");
		await user.click(d.getByRole("button", { name: "Konto löschen" }));
		await waitFor(() =>
			expect(api.callsTo("DELETE", "/api/me")[0]?.body).toEqual({
				email: "ann@example.org",
				locale: "de",
			}),
		);
	});

	it("explains a closed window from the server", async () => {
		mockApi({
			"GET /api/me": withWindow(120),
			"DELETE /api/me": apiError(403, "deletion_window_closed"),
		});
		const { user } = renderApp("/settings");
		const s = await section();
		await user.click(s.getByRole("button", { name: "Delete account…" }));
		const d = within(await screen.findByRole("dialog"));
		await user.type(
			d.getByLabelText(/Type your email address/),
			"ann@example.org",
		);
		await user.click(d.getByRole("button", { name: "Delete account" }));
		expect(await d.findByRole("alert")).toHaveTextContent(
			/sign in again with an email link/,
		);
	});
});

describe("deletion banner", () => {
	it("is not shown without a scheduled deletion", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		await section();
		expect(
			screen.queryByRole("button", { name: "Cancel deletion" }),
		).toBeNull();
	});

	it("shows the date on every page and cancels the deletion", async () => {
		const api = mockApi({
			"GET /api/me": scheduled,
			"GET /api/favourites": {
				body: { id: "root", kind: "folder", children: [] },
			},
			"POST /api/me/restore": signedIn(),
		});
		const { user } = renderApp("/favourites");
		const banner = await screen.findByText(
			/Your account will be deleted on November 8, 2026/,
		);
		expect(banner).toBeVisible();

		await user.click(screen.getByRole("button", { name: "Cancel deletion" }));
		await waitFor(() =>
			expect(api.callsTo("POST", "/api/me/restore")).toHaveLength(1),
		);
		expect(
			await screen.findByText(/Connected apps stay revoked/),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Cancel deletion" }),
		).toBeNull();
		expect(
			screen.getByRole("link", { name: "Go to Connected apps" }),
		).toHaveAttribute("href", "/settings?tab=apps");

		await user.click(screen.getByRole("button", { name: "Close" }));
		expect(screen.queryByText(/Connected apps stay revoked/)).toBeNull();
	});

	it("cancels from the settings section too", async () => {
		mockApi({
			"GET /api/me": scheduled,
			"POST /api/me/restore": signedIn(),
		});
		const { user } = renderApp("/settings");
		const s = await section();
		expect(s.getByText(/will be deleted on November 8, 2026/)).toBeVisible();
		await user.click(s.getByRole("button", { name: "Cancel deletion" }));
		expect(
			await screen.findByText(/Connected apps stay revoked/),
		).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Cancel deletion" }),
		).toBeNull();
	});
});
