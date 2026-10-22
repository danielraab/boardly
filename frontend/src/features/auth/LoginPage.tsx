import { WebAuthnAbortService } from "@simplewebauthn/browser";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { KeyRound, MailCheck } from "lucide-react";
import {
	type FormEvent,
	useCallback,
	useEffect,
	useRef,
	useState,
} from "react";
import { Trans, useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { ApiError, api } from "#/lib/api";
import { CODE_LENGTH, CodeField } from "./CodeField";
import {
	isCancelled,
	signInWithPasskey,
	supportsPasskeyAutofill,
	supportsPasskeys,
} from "./passkey";
import { meQueryKey, type User } from "./session";
import { useCountdown } from "./useCountdown";

const RESEND_SECONDS = 60;

type Problem =
	| "invalidEmail"
	| "invalidCode"
	| "rateLimited"
	| "passkeyUnknown"
	| "passkeyInvalid"
	| "generic";

function passkeyProblem(error: unknown): Problem | null {
	if (isCancelled(error)) return null;
	if (error instanceof ApiError && error.code === "passkey_unknown") {
		return "passkeyUnknown";
	}
	if (error instanceof ApiError && error.code === "passkey_invalid") {
		return "passkeyInvalid";
	}
	return "generic";
}

/** Finishes a sign-in made on the login page: stores the user and continues to next. */
function useSignedIn(next: string) {
	const queryClient = useQueryClient();
	const navigate = useNavigate();
	return useCallback(
		async (user: User) => {
			queryClient.setQueryData(meQueryKey, user);
			await navigate({ href: next });
		},
		[queryClient, navigate, next],
	);
}

/**
 * Passkey sign-in for the login page: autofill on the email field (where the
 * browser supports it) and a modal prompt for the button.
 */
function usePasskeySignIn(
	signedIn: (user: User) => Promise<void>,
	onProblem: (problem: Problem | null) => void,
) {
	const [supported] = useState(supportsPasskeys);
	const [pending, setPending] = useState(false);
	const active = useRef(true);

	const startAutofill = useCallback(async () => {
		if (!(await supportsPasskeyAutofill()) || !active.current) return;
		try {
			await signedIn(await signInWithPasskey(true));
		} catch (e) {
			// An aborted autofill request was replaced by the button or the page closed.
			if (!active.current || isCancelled(e)) return;
			onProblem(passkeyProblem(e));
			if (e instanceof ApiError) void startAutofill();
		}
	}, [signedIn, onProblem]);

	useEffect(() => {
		active.current = true;
		if (supported) void startAutofill();
		return () => {
			active.current = false;
			WebAuthnAbortService.cancelCeremony();
		};
	}, [supported, startAutofill]);

	async function signInWithButton() {
		onProblem(null);
		setPending(true);
		try {
			await signedIn(await signInWithPasskey(false));
		} catch (e) {
			if (!active.current) return;
			onProblem(passkeyProblem(e));
			void startAutofill();
		} finally {
			if (active.current) setPending(false);
		}
	}

	return { supported, pending, signInWithButton };
}

/** The login page; next is the local path to continue to after signing in. */
export function LoginPage({ next }: { next: string }) {
	const { t, i18n } = useTranslation();
	const [email, setEmail] = useState("");
	const [sentTo, setSentTo] = useState<string | null>(null);
	const [problem, setProblem] = useState<Problem | null>(null);
	const [code, setCode] = useState("");
	const countdown = useCountdown();
	const signedIn = useSignedIn(next);
	const passkey = usePasskeySignIn(signedIn, setProblem);

	const request = useMutation({
		mutationFn: (address: string) =>
			api("/api/auth/magic-link", {
				method: "POST",
				json: { email: address, locale: i18n.resolvedLanguage },
			}),
	});

	async function send(address: string) {
		setProblem(null);
		try {
			await request.mutateAsync(address);
			setSentTo(address.trim());
			countdown.start(RESEND_SECONDS);
		} catch (e) {
			if (e instanceof ApiError && e.code === "rate_limited") {
				countdown.start(e.retryAfter ?? RESEND_SECONDS);
				setProblem("rateLimited");
			} else if (e instanceof ApiError && e.code === "invalid_email") {
				setProblem("invalidEmail");
			} else {
				setProblem("generic");
			}
		}
	}

	const redeemCode = useMutation({
		mutationFn: (entered: string) =>
			api<User>("/api/auth/magic-link/code", {
				method: "POST",
				json: { email: sentTo, code: entered },
			}),
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		void send(email);
	}

	async function onSubmitCode(event: FormEvent) {
		event.preventDefault();
		setProblem(null);
		try {
			await signedIn(await redeemCode.mutateAsync(code));
		} catch (e) {
			setProblem(
				e instanceof ApiError && e.code === "invalid_code"
					? "invalidCode"
					: "generic",
			);
		}
	}

	const waiting = countdown.remaining > 0;
	const message =
		problem === "rateLimited"
			? waiting
				? t("login.rateLimited", { seconds: countdown.remaining })
				: null
			: problem === "invalidEmail"
				? t("login.invalidEmail")
				: problem === "invalidCode"
					? t("login.invalidCode")
					: problem === "passkeyUnknown"
						? t("login.passkeyUnknown")
						: problem === "passkeyInvalid"
							? t("login.passkeyInvalid")
							: problem === "generic"
								? t("common.genericError")
								: null;
	const alert = message && (
		<p role="alert" className="mt-4 text-sm text-danger">
			{message}
		</p>
	);

	if (sentTo !== null) {
		return (
			<section className="mx-auto max-w-md py-16 text-center">
				<MailCheck
					className="mx-auto size-8 text-ink"
					strokeWidth={1.75}
					aria-hidden
				/>
				<h1 className="mt-4 text-3xl font-semibold tracking-tight">
					{t("login.sentTitle")}
				</h1>
				<p className="mt-3 text-muted">
					<Trans
						i18nKey="login.sentText"
						values={{ email: sentTo }}
						components={{ strong: <strong className="text-ink" /> }}
					/>
				</p>
				<form className="mt-8 text-left" onSubmit={onSubmitCode} noValidate>
					<label
						id="login-code-label"
						htmlFor="login-code"
						className="block text-sm font-medium"
					>
						{t("login.codeLabel")}
					</label>
					<CodeField
						id="login-code"
						value={code}
						onChange={setCode}
						disabled={redeemCode.isPending}
					/>
					<button
						type="submit"
						className={`${primaryButton} mt-4 w-full`}
						disabled={redeemCode.isPending || code.length < CODE_LENGTH}
					>
						{redeemCode.isPending
							? t("login.codeChecking")
							: t("login.codeSubmit")}
					</button>
				</form>
				<div className="mt-6 flex flex-wrap items-center justify-center gap-4">
					<button
						type="button"
						className={secondaryButton}
						disabled={waiting || request.isPending}
						onClick={() => void send(sentTo)}
					>
						{waiting
							? t("login.resendIn", { seconds: countdown.remaining })
							: t("login.resend")}
					</button>
					<button
						type="button"
						className="font-medium underline underline-offset-4"
						onClick={() => {
							setSentTo(null);
							setCode("");
							setProblem(null);
						}}
					>
						{t("login.useOther")}
					</button>
				</div>
				{alert}
			</section>
		);
	}

	return (
		<section className="mx-auto max-w-md py-16 text-center">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("login.title")}
			</h1>
			<p className="mt-3 text-muted">{t("login.intro")}</p>
			<form className="mt-8 text-left" onSubmit={onSubmit} noValidate>
				<label htmlFor="login-email" className="block text-sm font-medium">
					{t("login.emailLabel")}
				</label>
				<input
					id="login-email"
					type="email"
					name="email"
					autoComplete={passkey.supported ? "username webauthn" : "username"}
					required
					value={email}
					onChange={(e) => setEmail(e.target.value)}
					className="mt-2 w-full rounded-lg border border-line bg-surface px-4 py-3 text-ink outline-none focus-visible:border-ink"
				/>
				<button
					type="submit"
					className={`${primaryButton} mt-4 w-full`}
					disabled={request.isPending || (problem === "rateLimited" && waiting)}
				>
					{request.isPending ? t("login.sending") : t("login.submit")}
				</button>
			</form>
			{passkey.supported && (
				<>
					<p className="mt-6 text-sm text-muted">{t("login.or")}</p>
					<button
						type="button"
						className={`${secondaryButton} mt-4 w-full`}
						disabled={passkey.pending}
						onClick={() => void passkey.signInWithButton()}
					>
						<KeyRound className="size-5" strokeWidth={1.75} aria-hidden />
						{t("login.passkey")}
					</button>
				</>
			)}
			{alert}
		</section>
	);
}
