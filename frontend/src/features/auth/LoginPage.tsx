import { useMutation } from "@tanstack/react-query";
import { MailCheck } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { ApiError, api } from "#/lib/api";
import { useCountdown } from "./useCountdown";

const RESEND_SECONDS = 60;

type Problem = "invalidEmail" | "rateLimited" | "generic";

export const primaryButton =
	"inline-flex items-center justify-center rounded-full bg-ink px-6 py-3 font-medium text-paper hover:bg-ink/85 disabled:cursor-not-allowed disabled:bg-ink/40";

export function LoginPage() {
	const { t, i18n } = useTranslation();
	const [email, setEmail] = useState("");
	const [sentTo, setSentTo] = useState<string | null>(null);
	const [problem, setProblem] = useState<Problem | null>(null);
	const countdown = useCountdown();

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

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		void send(email);
	}

	const waiting = countdown.remaining > 0;
	const message =
		problem === "rateLimited"
			? waiting
				? t("login.rateLimited", { seconds: countdown.remaining })
				: null
			: problem === "invalidEmail"
				? t("login.invalidEmail")
				: problem === "generic"
					? t("common.genericError")
					: null;
	const alert = message && (
		<p role="alert" className="mt-4 text-sm text-red-700">
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
				<div className="mt-8 flex flex-wrap items-center justify-center gap-4">
					<button
						type="button"
						className={primaryButton}
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
					autoComplete="username"
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
			{alert}
		</section>
	);
}
