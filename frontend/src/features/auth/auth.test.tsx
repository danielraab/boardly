import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { apiError, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const never = () => new Promise<never>(() => {});

describe("route protection", () => {
	it("redirects signed-out visitors from /settings to /login", async () => {
		const { router } = renderApp("/settings");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		expect(
			await screen.findByRole("heading", { name: "Sign in" }),
		).toBeVisible();
	});

	it("redirects signed-in users from /login to /", async () => {
		mockApi({ "GET /api/me": signedIn({ name: "Ann" }) });
		const { router } = renderApp("/login");
		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { name: "Welcome, Ann" }),
		).toBeVisible();
	});

	it("sends signed-in users from /login to a local redirect path", async () => {
		mockApi({ "GET /api/me": signedIn({ name: "Ann" }) });
		const { router } = renderApp("/login?redirect=%2Fsettings");
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/settings"),
		);
	});

	it("shows neither landing nor home at / while the session is loading", async () => {
		mockApi({ "GET /api/me": never });
		renderApp("/");
		expect(await screen.findByText("Loading…")).toBeInTheDocument();
		expect(screen.queryByRole("heading", { name: "Boardly" })).toBeNull();
		expect(screen.queryByRole("heading", { name: /Welcome/ })).toBeNull();
	});

	it("switches to signed out when a request returns 401", async () => {
		const api = mockApi({ "GET /api/me": signedIn() });
		const { user, router } = renderApp("/settings");
		const input = await screen.findByLabelText("Display name");

		api.set({ "PATCH /api/me": apiError(401, "unauthenticated") });
		await user.type(input, "Ann");
		await user.click(screen.getByRole("button", { name: "Save" }));

		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
		await user.click(screen.getByRole("button", { name: "User menu" }));
		const items = await screen.findAllByRole("menuitem");
		expect(items.map((i) => i.textContent)).toEqual(["Sign in"]);
	});
});

describe("login page", () => {
	it("sends a link and shows the inbox state with a disabled resend", async () => {
		const api = mockApi({
			"POST /api/auth/magic-link": {
				status: 202,
				body: { status: "accepted" },
			},
		});
		const { user } = renderApp("/login");

		await user.type(
			await screen.findByLabelText("Email address"),
			"ann@example.org",
		);
		await user.click(screen.getByRole("button", { name: "Send sign-in link" }));

		expect(
			await screen.findByRole("heading", { name: "Check your inbox" }),
		).toBeVisible();
		expect(screen.getByText("ann@example.org")).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Send a new link in 60 s" }),
		).toBeDisabled();
		expect(api.callsTo("POST", "/api/auth/magic-link")[0].body).toEqual({
			email: "ann@example.org",
			locale: "en",
		});
	});

	it("requests the email in the current language", async () => {
		const api = mockApi({ "POST /api/auth/magic-link": { status: 202 } });
		const { user } = renderApp("/login");
		await user.click(await screen.findByRole("button", { name: "Language" }));
		await user.click(
			await screen.findByRole("menuitemradio", { name: "Deutsch" }),
		);
		await user.type(
			await screen.findByLabelText("E-Mail-Adresse"),
			"ann@example.org",
		);
		await user.click(
			screen.getByRole("button", { name: "Anmeldelink senden" }),
		);
		await screen.findByRole("heading", { name: "Schau in dein Postfach" });
		expect(api.callsTo("POST", "/api/auth/magic-link")[0].body).toMatchObject({
			locale: "de",
		});
	});

	it("shows how long to wait when rate limited", async () => {
		mockApi({
			"POST /api/auth/magic-link": apiError(429, "rate_limited", {
				"Retry-After": "120",
			}),
		});
		const { user } = renderApp("/login");
		await user.type(
			await screen.findByLabelText("Email address"),
			"ann@example.org",
		);
		await user.click(screen.getByRole("button", { name: "Send sign-in link" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"Too many requests. Try again in 120 s.",
		);
		expect(
			screen.getByRole("button", { name: "Send sign-in link" }),
		).toBeDisabled();
	});

	it("shows an error for an invalid address", async () => {
		mockApi({ "POST /api/auth/magic-link": apiError(400, "invalid_email") });
		const { user } = renderApp("/login");
		await user.type(await screen.findByLabelText("Email address"), "nope");
		await user.click(screen.getByRole("button", { name: "Send sign-in link" }));
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"Enter a valid email address.",
		);
	});
});

describe("one-time code", () => {
	// The label names the slot group and its first slot.
	const firstSlot = () =>
		screen.getByRole("textbox", { name: "One-time code from the email" });

	async function requestEmail(path = "/login") {
		const app = renderApp(path);
		await app.user.type(
			await screen.findByLabelText("Email address"),
			"ann@example.org",
		);
		await app.user.click(
			screen.getByRole("button", { name: "Send sign-in link" }),
		);
		await screen.findByRole("heading", { name: "Check your inbox" });
		return app;
	}

	it("signs in with the code from the email", async () => {
		const api = mockApi({
			"POST /api/auth/magic-link": { status: 202 },
			"POST /api/auth/magic-link/code": signedIn({ name: "Ann" }),
		});
		const { user, router } = await requestEmail();

		await user.type(firstSlot(), "k7q4mz");
		await user.click(screen.getByRole("button", { name: "Sign in with code" }));

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { name: "Welcome, Ann" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/auth/magic-link/code")[0].body).toEqual({
			email: "ann@example.org",
			code: "K7Q4MZ",
		});
	});

	it("shows the code in six upper-case slots and takes a pasted hyphen", async () => {
		mockApi({ "POST /api/auth/magic-link": { status: 202 } });
		const { user } = await requestEmail();
		const sign = screen.getByRole("button", { name: "Sign in with code" });
		expect(sign).toBeDisabled();

		await user.click(firstSlot());
		await user.paste("k7q-4mz");

		const slots = [
			firstSlot(),
			...[2, 3, 4, 5, 6].map((n) =>
				screen.getByLabelText(`Character ${n} of 6`),
			),
		];
		expect(slots.map((s) => (s as HTMLInputElement).value)).toEqual([
			"K",
			"7",
			"Q",
			"4",
			"M",
			"Z",
		]);
		expect(sign).toBeEnabled();
	});

	it("continues to the redirect path", async () => {
		mockApi({
			"POST /api/auth/magic-link": { status: 202 },
			"POST /api/auth/magic-link/code": signedIn(),
		});
		const { user, router } = await requestEmail("/login?redirect=%2Fsettings");
		await user.type(firstSlot(), "K7Q4MZ");
		await user.click(screen.getByRole("button", { name: "Sign in with code" }));
		await waitFor(() =>
			expect(router.state.location.pathname).toBe("/settings"),
		);
	});

	it("explains a wrong code and lets the visitor try again", async () => {
		mockApi({
			"POST /api/auth/magic-link": { status: 202 },
			"POST /api/auth/magic-link/code": apiError(400, "invalid_code"),
		});
		const { user, router } = await requestEmail();
		const field = firstSlot();
		await user.type(field, "AAAAAA");
		await user.click(screen.getByRole("button", { name: "Sign in with code" }));

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This code is wrong or has expired.",
		);
		expect(field).toBeVisible();
		expect(router.state.location.pathname).toBe("/login");
	});
});

describe("verify page", () => {
	const user = { id: "u1", email: "ann@example.org", name: null };
	const confirmationRequired = apiError(403, "confirmation_required");

	/** The verify endpoint answers like the server: unconfirmed requests need the (absent) browser cookie. */
	const otherBrowser = ({ body }: { body: unknown }) =>
		(body as { confirm: boolean }).confirm
			? { body: user }
			: confirmationRequired;

	it("signs in right away in the browser that requested the link", async () => {
		const api = mockApi({ "POST /api/auth/magic-link/verify": { body: user } });
		const { router } = renderApp("/login/verify#token=abc123");

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { name: "Welcome, ann@example.org" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/auth/magic-link/verify")).toEqual([
			expect.objectContaining({ body: { token: "abc123", confirm: false } }),
		]);
	});

	it("asks for confirmation elsewhere and strips the token from the URL", async () => {
		const api = mockApi({ "POST /api/auth/magic-link/verify": otherBrowser });
		const { router } = renderApp("/login/verify#token=abc123");

		expect(
			await screen.findByRole("button", { name: "Complete sign-in" }),
		).toBeVisible();
		await waitFor(() => expect(router.state.location.hash).toBe(""));
		expect(api.callsTo("POST", "/api/auth/magic-link/verify")).toHaveLength(1);
		expect(router.state.location.pathname).toBe("/login/verify");
	});

	it("signs in after the confirmation click and goes to /", async () => {
		const api = mockApi({ "POST /api/auth/magic-link/verify": otherBrowser });
		const { user: u, router } = renderApp("/login/verify#token=abc123");
		await u.click(
			await screen.findByRole("button", { name: "Complete sign-in" }),
		);

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { name: "Welcome, ann@example.org" }),
		).toBeVisible();
		expect(
			api.callsTo("POST", "/api/auth/magic-link/verify").map((c) => c.body),
		).toEqual([
			{ token: "abc123", confirm: false },
			{ token: "abc123", confirm: true },
		]);
	});

	it("explains an invalid link and links back to /login", async () => {
		mockApi({
			"POST /api/auth/magic-link/verify": apiError(400, "invalid_token"),
		});
		renderApp("/login/verify#token=used");
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This sign-in link is invalid or has expired.",
		);
		expect(
			screen.getByRole("link", { name: "Request a new link" }),
		).toHaveAttribute("href", "/login");
		expect(
			screen.queryByRole("button", { name: "Complete sign-in" }),
		).toBeNull();
	});

	it("explains a link that turns out invalid after confirming", async () => {
		mockApi({
			"POST /api/auth/magic-link/verify": ({ body }) =>
				(body as { confirm: boolean }).confirm
					? apiError(400, "invalid_token")
					: confirmationRequired,
		});
		const { user: u } = renderApp("/login/verify#token=abc123");
		await u.click(
			await screen.findByRole("button", { name: "Complete sign-in" }),
		);
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This sign-in link is invalid or has expired.",
		);
	});

	it("shows the error right away when the link has no token", async () => {
		const api = mockApi();
		renderApp("/login/verify");
		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This sign-in link is invalid or has expired.",
		);
		expect(api.callsTo("POST", "/api/auth/magic-link/verify")).toHaveLength(0);
	});
});
