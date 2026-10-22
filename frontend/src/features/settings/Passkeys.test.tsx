import { act, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Passkey } from "#/features/auth/passkey";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";
import { setTheme, type ThemeId } from "#/theme";

const webauthn = vi.hoisted(() => ({
	supported: true,
	startRegistration: vi.fn(),
}));

vi.mock("@simplewebauthn/browser", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@simplewebauthn/browser")>();
	return {
		...actual,
		browserSupportsWebAuthn: () => webauthn.supported,
		browserSupportsWebAuthnAutofill: async () => false,
		startRegistration: webauthn.startRegistration,
	};
});

const NOW = new Date("2026-03-01T12:00:00Z");

const gpm: Passkey = {
	id: "p1",
	label: null,
	provider: {
		name: "Google Password Manager",
		iconLight: "data:image/svg+xml;base64,bGlnaHQ=",
		iconDark: "data:image/svg+xml;base64,ZGFyaw==",
	},
	createdAt: "2026-02-10T09:00:00Z",
	lastUsedAt: "2026-02-28T18:30:00Z",
};

const unknown: Passkey = {
	id: "p2",
	label: null,
	provider: null,
	createdAt: "2026-01-05T09:00:00Z",
	lastUsedAt: null,
};

function withWindow(secondsLeft: number) {
	return signedIn({
		passkeyEnrollmentUntil: new Date(
			NOW.getTime() + secondsLeft * 1000,
		).toISOString(),
	});
}

beforeEach(() => {
	webauthn.supported = true;
	webauthn.startRegistration.mockReset();
	vi.useFakeTimers({ toFake: ["Date", "setInterval", "clearInterval"] });
	vi.setSystemTime(NOW);
});

afterEach(() => {
	vi.useRealTimers();
});

describe("settings tabs", () => {
	it("selects the Profile tab by default", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		const tabs = await screen.findAllByRole("tab");
		expect(tabs.map((tab) => tab.textContent)).toEqual([
			"Profile",
			"Passkeys",
			"Connected apps",
		]);
		await waitFor(() =>
			expect(tabs[0]).toHaveAttribute("aria-selected", "true"),
		);
		expect(
			within(screen.getByRole("tabpanel")).getByLabelText("Display name"),
		).toBeVisible();
	});

	it("puts the selected tab into the URL", async () => {
		mockApi({ "GET /api/me": signedIn() });
		const { user, router } = renderApp("/settings");
		await user.click(await screen.findByRole("tab", { name: "Passkeys" }));
		await waitFor(() =>
			expect(router.state.location.search).toEqual({ tab: "passkeys" }),
		);
		expect(
			await within(screen.getByRole("tabpanel")).findByText("No passkeys yet"),
		).toBeVisible();
	});

	it("keeps the Passkeys tab after a reload", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings?tab=passkeys");
		const tab = await screen.findByRole("tab", { name: "Passkeys" });
		await waitFor(() => expect(tab).toHaveAttribute("aria-selected", "true"));
	});

	it("falls back to the Profile tab for an unknown tab", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings?tab=nope");
		const tab = await screen.findByRole("tab", { name: "Profile" });
		await waitFor(() => expect(tab).toHaveAttribute("aria-selected", "true"));
	});
});

describe("passkey list", () => {
	it("shows a known provider with its icon and dates", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [gpm] },
		});
		renderApp("/settings?tab=passkeys");

		const item = (await screen.findByText("Google Password Manager")).closest(
			"li",
		) as HTMLElement;
		const img = item.querySelector("img");
		expect(img).toHaveAttribute("src", gpm.provider?.iconLight);
		expect(img).toHaveAttribute("alt", "");
		expect(item).toHaveTextContent("Added Feb 10, 2026");
		expect(item).toHaveTextContent("last used Feb 28, 2026");
	});

	async function providerIcon() {
		const item = (await screen.findByText("Google Password Manager")).closest(
			"li",
		) as HTMLElement;
		return item.querySelector("img");
	}

	const lightOnly: Passkey = {
		...gpm,
		provider: {
			name: "Google Password Manager",
			iconLight: "light.svg",
			iconDark: null,
		},
	};

	it.each<[ThemeId, Passkey, string | null | undefined]>([
		["white", gpm, gpm.provider?.iconLight],
		["fjord", gpm, gpm.provider?.iconDark],
		["black", lightOnly, "light.svg"],
	])(
		"in theme %s shows the matching provider icon",
		async (theme, passkey, src) => {
			setTheme(theme);
			mockApi({
				"GET /api/me": signedIn(),
				"GET /api/passkeys": { body: [passkey] },
			});
			renderApp("/settings?tab=passkeys");
			expect(await providerIcon()).toHaveAttribute("src", src);
		},
	);

	it("switches the provider icon when the theme changes", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [gpm] },
		});
		renderApp("/settings?tab=passkeys");
		const img = await providerIcon();
		expect(img).toHaveAttribute("src", gpm.provider?.iconLight);
		act(() => setTheme("dusk"));
		expect(img).toHaveAttribute("src", gpm.provider?.iconDark);
	});

	it("shows a generic icon and name for an unknown provider", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [unknown] },
		});
		renderApp("/settings?tab=passkeys");

		const item = (await screen.findByText("Passkey")).closest(
			"li",
		) as HTMLElement;
		expect(item.querySelector("img")).toBeNull();
		expect(item.querySelector("svg")).not.toBeNull();
		expect(item).toHaveTextContent("never used");
	});

	it("prefers the label over the provider name", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [{ ...gpm, label: "Work laptop" }] },
		});
		renderApp("/settings?tab=passkeys");
		expect(await screen.findByText("Work laptop")).toBeVisible();
		expect(screen.queryByText("Google Password Manager")).toBeNull();
	});

	it("formats dates in the current language", async () => {
		mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [gpm] },
		});
		renderApp("/settings?tab=passkeys");
		await screen.findByText("Google Password Manager");
		await act(() => i18n.changeLanguage("de"));
		expect(
			await screen.findByText(/Hinzugefügt am 10\.02\.2026/),
		).toBeVisible();
	});

	it("explains passkeys when there are none", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings?tab=passkeys");
		expect(await screen.findByText("No passkeys yet")).toBeVisible();
		expect(
			screen.getByText(/Add a passkey to sign in with one tap/),
		).toBeVisible();
	});
});

describe("enrollment window", () => {
	it("counts down and disables adding when it reaches zero", async () => {
		const api = mockApi({ "GET /api/me": withWindow(240) });
		renderApp("/settings?tab=passkeys");

		const timer = await screen.findByRole("timer");
		expect(timer).toHaveTextContent("4:00");
		expect(screen.getByRole("button", { name: "Add passkey" })).toBeEnabled();

		act(() => vi.advanceTimersByTime(1000));
		expect(timer).toHaveTextContent("3:59");

		api.set({ "GET /api/me": signedIn() });
		act(() => vi.advanceTimersByTime(239_000));

		expect(screen.queryByRole("timer")).toBeNull();
		expect(
			screen.getByText(/To add a passkey, sign in with an email link first/),
		).toBeVisible();
		expect(screen.getByRole("button", { name: "Add passkey" })).toBeDisabled();
		await waitFor(() => expect(api.callsTo("GET", "/api/me")).toHaveLength(2));
	});

	it("sends a sign-in link to the user's own address", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"POST /api/auth/magic-link": { status: 202 },
		});
		const { user } = renderApp("/settings?tab=passkeys");

		expect(
			await screen.findByRole("button", { name: "Add passkey" }),
		).toBeDisabled();
		await user.click(screen.getByRole("button", { name: "Send sign-in link" }));

		expect(
			await screen.findByText(
				"We sent a sign-in link to ann@example.org. Open it to add passkeys.",
			),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/auth/magic-link")[0].body).toEqual({
			email: "ann@example.org",
			locale: "en",
		});
	});

	it("explains that the browser can't add passkeys", async () => {
		webauthn.supported = false;
		mockApi({ "GET /api/me": withWindow(240) });
		renderApp("/settings?tab=passkeys");
		expect(
			await screen.findByText(/This browser doesn't support passkeys/),
		).toBeVisible();
		expect(screen.getByRole("button", { name: "Add passkey" })).toBeDisabled();
	});

	it("adds a passkey with a label", async () => {
		const added: Passkey = { ...gpm, id: "p3", label: "Work laptop" };
		const api = mockApi({
			"GET /api/me": withWindow(240),
			"POST /api/passkeys/registration/options": { body: { challenge: "c" } },
			"POST /api/passkeys/registration/verify": () => {
				api.set({ "GET /api/passkeys": { body: [added] } });
				return { status: 201, body: added };
			},
		});
		webauthn.startRegistration.mockResolvedValue({ id: "cred" });
		const { user } = renderApp("/settings?tab=passkeys");

		await user.click(
			await screen.findByRole("button", { name: "Add passkey" }),
		);
		const dialog = await screen.findByRole("dialog");
		await user.type(
			within(dialog).getByLabelText("Name (optional)"),
			"Work laptop",
		);
		await user.click(within(dialog).getByRole("button", { name: "Continue" }));

		expect(await screen.findByText("Work laptop")).toBeVisible();
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(
			api.callsTo("POST", "/api/passkeys/registration/verify")[0].body,
		).toEqual({ credential: { id: "cred" }, label: "Work laptop" });
		expect(webauthn.startRegistration).toHaveBeenCalledWith({
			optionsJSON: { challenge: "c" },
		});
	});
});

describe("deleting a passkey", () => {
	it("removes the passkey after confirming", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [gpm, unknown] },
			"DELETE /api/passkeys/p1": () => {
				api.set({ "GET /api/passkeys": { body: [unknown] } });
				return { status: 204 };
			},
		});
		const { user } = renderApp("/settings?tab=passkeys");

		await user.click(
			await screen.findByRole("button", {
				name: "Delete passkey Google Password Manager",
			}),
		);
		const dialog = await screen.findByRole("alertdialog");
		expect(dialog).toHaveTextContent(
			"You will no longer be able to sign in with “Google Password Manager”.",
		);
		await user.click(within(dialog).getByRole("button", { name: "Delete" }));

		await waitFor(() =>
			expect(screen.queryByText("Google Password Manager")).toBeNull(),
		);
		expect(screen.getByText("Passkey")).toBeVisible();
		expect(api.callsTo("DELETE", "/api/passkeys/p1")).toHaveLength(1);
	});

	it("keeps the passkey when cancelled", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/passkeys": { body: [gpm] },
		});
		const { user } = renderApp("/settings?tab=passkeys");

		await user.click(
			await screen.findByRole("button", {
				name: "Delete passkey Google Password Manager",
			}),
		);
		const dialog = await screen.findByRole("alertdialog");
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));

		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
		expect(screen.getByText("Google Password Manager")).toBeVisible();
		expect(api.callsTo("DELETE", "/api/passkeys/p1")).toHaveLength(0);
	});
});
