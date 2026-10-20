import {
	type AuthenticationResponseJSON,
	browserSupportsWebAuthn,
	browserSupportsWebAuthnAutofill,
	type PublicKeyCredentialCreationOptionsJSON,
	type PublicKeyCredentialRequestOptionsJSON,
	sendSignal,
	startAuthentication,
	startRegistration,
	WebAuthnError,
} from "@simplewebauthn/browser";
import { queryOptions } from "@tanstack/react-query";
import { ApiError, api } from "#/lib/api";
import type { User } from "./session";

export type PasskeyProvider = {
	name: string;
	iconLight: string | null;
	iconDark: string | null;
};

export type Passkey = {
	id: string;
	label: string | null;
	provider: PasskeyProvider | null;
	createdAt: string;
	lastUsedAt: string | null;
};

/** Whether the browser can use passkeys at all. */
export function supportsPasskeys(): boolean {
	return browserSupportsWebAuthn();
}

/** Whether the browser offers passkeys in input autofill (conditional mediation). */
export function supportsPasskeyAutofill(): Promise<boolean> {
	return browserSupportsWebAuthnAutofill();
}

/**
 * Whether the error means the browser prompt was dismissed or replaced by
 * another one; such errors show no message.
 */
export function isCancelled(error: unknown): boolean {
	if (!(error instanceof WebAuthnError)) return false;
	if (error.code === "ERROR_CEREMONY_ABORTED") return true;
	const cause = error.cause as { name?: string } | undefined;
	return cause?.name === "NotAllowedError" || cause?.name === "AbortError";
}

/**
 * Signs in with a passkey: with autofill, it waits for the visitor to pick a
 * passkey in the email field's suggestions; otherwise it opens the browser prompt.
 * A passkey the server doesn't know is reported to the password manager.
 */
export async function signInWithPasskey(autofill = false): Promise<User> {
	const optionsJSON = await api<PublicKeyCredentialRequestOptionsJSON>(
		"/api/auth/passkey/login/options",
		{ method: "POST", json: {} },
	);
	const credential: AuthenticationResponseJSON = await startAuthentication({
		optionsJSON,
		useBrowserAutofill: autofill,
	});
	try {
		return await api<User>("/api/auth/passkey/login/verify", {
			method: "POST",
			json: credential,
		});
	} catch (e) {
		if (
			e instanceof ApiError &&
			e.code === "passkey_unknown" &&
			optionsJSON.rpId
		) {
			sendSignal({
				signalName: "unknownCredential",
				rpID: optionsJSON.rpId,
				credentialID: credential.id,
			}).catch(() => {
				// Not supported by every browser; the signal is only a courtesy.
			});
		}
		throw e;
	}
}

export const passkeysQueryKey = ["passkeys"] as const;

export const passkeysQueryOptions = queryOptions({
	queryKey: passkeysQueryKey,
	queryFn: () => api<Passkey[]>("/api/passkeys"),
});

/** Adds a passkey for the signed-in user: server options, browser prompt, verify. */
export async function registerPasskey(label: string): Promise<Passkey> {
	const optionsJSON = await api<PublicKeyCredentialCreationOptionsJSON>(
		"/api/passkeys/registration/options",
		{ method: "POST", json: {} },
	);
	const credential = await startRegistration({ optionsJSON });
	return api<Passkey>("/api/passkeys/registration/verify", {
		method: "POST",
		json: { credential, label: label.trim() === "" ? null : label },
	});
}

export function deletePasskey(id: string): Promise<void> {
	return api<void>(`/api/passkeys/${encodeURIComponent(id)}`, {
		method: "DELETE",
	});
}
