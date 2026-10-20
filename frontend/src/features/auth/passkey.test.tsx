import { WebAuthnError } from "@simplewebauthn/browser";
import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { apiError, type MockResponse, mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

const webauthn = vi.hoisted(() => ({
	supported: true,
	autofill: false,
	startAuthentication: vi.fn(),
	sendSignal: vi.fn(),
}));

vi.mock("@simplewebauthn/browser", async (importOriginal) => {
	const actual =
		await importOriginal<typeof import("@simplewebauthn/browser")>();
	return {
		...actual,
		browserSupportsWebAuthn: () => webauthn.supported,
		browserSupportsWebAuthnAutofill: async () => webauthn.autofill,
		startAuthentication: webauthn.startAuthentication,
		sendSignal: webauthn.sendSignal,
		WebAuthnAbortService: { cancelCeremony: vi.fn() },
	};
});

const options = { challenge: "c", rpId: "localhost" };
const assertion = { id: "cred-1", rawId: "cred-1", type: "public-key" };
const ann = signedIn({ name: "Ann" }).body;

function cancelled() {
	const cause = new Error("The operation either timed out or was not allowed.");
	cause.name = "NotAllowedError";
	return new WebAuthnError({
		message: cause.message,
		code: "ERROR_PASSTHROUGH_SEE_CAUSE_PROPERTY",
		cause,
	});
}

beforeEach(() => {
	webauthn.supported = true;
	webauthn.autofill = false;
	webauthn.startAuthentication.mockReset();
	webauthn.sendSignal.mockReset().mockResolvedValue(undefined);
});

function mockPasskeyApi(verify: MockResponse = { body: ann }) {
	return mockApi({
		"POST /api/auth/passkey/login/options": { body: options },
		"POST /api/auth/passkey/login/verify": verify,
	});
}

describe("passkey sign-in", () => {
	it("signs in with the button and shows the start page", async () => {
		const api = mockPasskeyApi();
		webauthn.startAuthentication.mockResolvedValue(assertion);
		const { user, router } = renderApp("/login");

		await user.click(
			await screen.findByRole("button", { name: "Sign in with passkey" }),
		);

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(
			await screen.findByRole("heading", { name: "Welcome, Ann" }),
		).toBeVisible();
		expect(webauthn.startAuthentication).toHaveBeenCalledWith({
			optionsJSON: options,
			useBrowserAutofill: false,
		});
		expect(
			api.callsTo("POST", "/api/auth/passkey/login/verify")[0].body,
		).toEqual(assertion);
		expect(api.callsTo("GET", "/api/me")).toHaveLength(1);
	});

	it("shows no error when the prompt is cancelled", async () => {
		const api = mockPasskeyApi();
		webauthn.startAuthentication.mockRejectedValue(cancelled());
		const { user } = renderApp("/login");

		const button = await screen.findByRole("button", {
			name: "Sign in with passkey",
		});
		await user.click(button);

		await waitFor(() => expect(button).toBeEnabled());
		expect(screen.queryByRole("alert")).toBeNull();
		expect(api.callsTo("POST", "/api/auth/passkey/login/verify")).toHaveLength(
			0,
		);
	});

	it("explains a deleted passkey and tells the password manager", async () => {
		mockPasskeyApi(apiError(401, "passkey_unknown"));
		webauthn.startAuthentication.mockResolvedValue(assertion);
		const { user } = renderApp("/login");

		await user.click(
			await screen.findByRole("button", { name: "Sign in with passkey" }),
		);

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"This passkey is no longer valid. Sign in with an email link instead.",
		);
		expect(webauthn.sendSignal).toHaveBeenCalledWith({
			signalName: "unknownCredential",
			rpID: "localhost",
			credentialID: "cred-1",
		});
		expect(screen.getByLabelText("Email address")).toBeEnabled();
	});

	it("explains a passkey that could not be verified", async () => {
		mockPasskeyApi(apiError(401, "passkey_invalid"));
		webauthn.startAuthentication.mockResolvedValue(assertion);
		const { user } = renderApp("/login");

		await user.click(
			await screen.findByRole("button", { name: "Sign in with passkey" }),
		);

		expect(await screen.findByRole("alert")).toHaveTextContent(
			"The passkey could not be verified.",
		);
		expect(webauthn.sendSignal).not.toHaveBeenCalled();
	});

	it("offers passkeys in the email field's autofill", async () => {
		mockPasskeyApi();
		webauthn.autofill = true;
		webauthn.startAuthentication.mockResolvedValue(assertion);
		const { router } = renderApp("/login");

		await waitFor(() => expect(router.state.location.pathname).toBe("/"));
		expect(webauthn.startAuthentication).toHaveBeenCalledWith({
			optionsJSON: options,
			useBrowserAutofill: true,
		});
	});

	it("marks the email field for passkey autofill", async () => {
		mockPasskeyApi();
		renderApp("/login");
		expect(await screen.findByLabelText("Email address")).toHaveAttribute(
			"autocomplete",
			"username webauthn",
		);
	});

	it("hides the button without WebAuthn and keeps the email form", async () => {
		webauthn.supported = false;
		const api = mockApi({ "POST /api/auth/magic-link": { status: 202 } });
		const { user } = renderApp("/login");

		const input = await screen.findByLabelText("Email address");
		expect(input).toHaveAttribute("autocomplete", "username");
		expect(
			screen.queryByRole("button", { name: "Sign in with passkey" }),
		).toBeNull();

		await user.type(input, "ann@example.org");
		await user.click(screen.getByRole("button", { name: "Send sign-in link" }));
		expect(
			await screen.findByRole("heading", { name: "Check your inbox" }),
		).toBeVisible();
		expect(api.callsTo("POST", "/api/auth/magic-link")).toHaveLength(1);
		expect(webauthn.startAuthentication).not.toHaveBeenCalled();
	});
});
