import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton } from "#/components/buttonStyles";
import { ApiError, api } from "#/lib/api";
import { appTitle } from "#/lib/appTitle";
import { usePageTitle } from "#/lib/usePageTitle";
import { meQueryKey, resetToSignedOut, type User } from "./session";

/** Reads the token from a location hash like "token=…" (with or without "#"). */
function tokenFromHash(hash: string) {
	return new URLSearchParams(hash.replace(/^#/, "")).get("token") ?? "";
}

function hasCode(error: unknown, code: string) {
	return error instanceof ApiError && error.code === code;
}

/**
 * Redeems a sign-in link. The token comes from the URL fragment. The page first
 * tries to redeem it without confirmation, which the server only allows in the
 * browser that requested the link. Anywhere else (another device, a mail scanner
 * that opens links) it asks for an explicit "Complete sign-in" click, so the
 * link isn't used up by merely opening it.
 */
export function VerifyPage() {
	const { t } = useTranslation();
	usePageTitle(t("verify.title"));
	const location = useLocation();
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const [token] = useState(() => tokenFromHash(location.hash));

	// Remove the token from the address bar and the history entry.
	useEffect(() => {
		if (location.hash) {
			void navigate({ to: "/login/verify", hash: "", replace: true });
		}
	}, [location.hash, navigate]);

	const redeem = useMutation({
		mutationFn: (confirm: boolean) =>
			api<User>("/api/auth/magic-link/verify", {
				method: "POST",
				json: { token, confirm },
			}),
		onSuccess: async (user) => {
			resetToSignedOut(queryClient);
			queryClient.setQueryData(meQueryKey, user);
			await navigate({ to: "/" });
		},
	});

	// Try the same-browser sign-in once on open.
	const tried = useRef(false);
	const { mutate } = redeem;
	useEffect(() => {
		if (!token || tried.current) return;
		tried.current = true;
		mutate(false);
	}, [token, mutate]);

	const needsConfirmation =
		redeem.isError && hasCode(redeem.error, "confirmation_required");
	const failed = !token || (redeem.isError && !needsConfirmation);
	const busy = redeem.isPending || redeem.isSuccess;

	return (
		<section className="mx-auto max-w-md py-16 text-center">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("verify.title")}
			</h1>
			{failed ? (
				<>
					<p role="alert" className="mt-3 text-danger">
						{!token || hasCode(redeem.error, "invalid_token")
							? t("verify.invalid")
							: t("common.genericError")}
					</p>
					<Link
						to="/login"
						className="mt-6 inline-block font-medium underline underline-offset-4"
					>
						{t("verify.requestNew")}
					</Link>
				</>
			) : needsConfirmation ? (
				<>
					<p className="mt-3 text-muted">
						{t("verify.text", { appName: appTitle() })}
					</p>
					<button
						type="button"
						className={`${primaryButton} mt-8`}
						onClick={() => redeem.mutate(true)}
					>
						{t("verify.confirm")}
					</button>
				</>
			) : (
				<output className="mt-6 flex items-center justify-center gap-3 text-muted">
					{busy && (
						<span className="size-5 animate-spin rounded-full border-2 border-line border-t-muted motion-reduce:animate-none" />
					)}
					{t("verify.signingIn")}
				</output>
			)}
		</section>
	);
}
