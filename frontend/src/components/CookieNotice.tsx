import { Link } from "@tanstack/react-router";
import { X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useMediaPlayer } from "#/features/media/player";
import { appTitle } from "#/lib/appTitle";
import {
	type CookieNoticeVariant,
	cookieNoticeVariant,
} from "#/lib/consent/settings";
import { legalLink } from "#/lib/legalLinks";

export const COOKIE_NOTICE_STORAGE_KEY = "boardly.cookieNotice.dismissed";

/** Closed for this page load, for browsers that block storage. */
let closedForPage: CookieNoticeVariant | undefined;

function dismissed(variant: CookieNoticeVariant): boolean {
	if (closedForPage === variant) return true;
	try {
		return localStorage.getItem(COOKIE_NOTICE_STORAGE_KEY) === variant;
	} catch {
		return false;
	}
}

function dismiss(variant: CookieNoticeVariant) {
	closedForPage = variant;
	try {
		localStorage.setItem(COOKIE_NOTICE_STORAGE_KEY, variant);
	} catch {
		// Blocked storage: the notice stays closed until the page is reloaded.
	}
}

/** Forgets a notice closed for the page; for tests. */
export function resetCookieNotice() {
	closedForPage = undefined;
}

/**
 * The small notice, while analytics consent is off, that only necessary
 * cookies are used. The server picks the variant; closing it is remembered
 * per variant, so a changed text shows again.
 */
export function CookieNotice() {
	const { t } = useTranslation();
	const { current } = useMediaPlayer();
	const variant = cookieNoticeVariant();
	const [closed, setClosed] = useState(() => !variant || dismissed(variant));
	if (!variant || closed) return null;

	const privacy = legalLink("privacy");
	const appName = appTitle();
	// Above the mini player bar (h-20) while it is open.
	const bottom = current ? "bottom-24" : "bottom-4";

	return (
		<section
			aria-label={t("cookieNotice.label")}
			className={`fixed ${bottom} inset-x-4 z-40 flex items-start gap-2 rounded-lg border border-line bg-surface p-3 text-sm text-ink shadow-[0_8px_24px_-12px_var(--color-shadow)] sm:right-auto sm:left-6 sm:max-w-sm`}
		>
			<p className="min-w-0 flex-1">
				{t("cookieNotice.necessary", { appName })}
				{variant === "cookieless-analytics" && (
					<> {t("cookieNotice.cookieless")}</>
				)}
				{privacy && (
					<>
						{" "}
						{privacy.kind === "page" ? (
							<Link to={privacy.href} className="text-accent underline">
								{t("cookieNotice.privacy")}
							</Link>
						) : (
							<a
								href={privacy.href}
								target="_blank"
								rel="noopener noreferrer"
								className="text-accent underline"
							>
								{t("cookieNotice.privacy")}
							</a>
						)}
					</>
				)}
			</p>
			<button
				type="button"
				aria-label={t("cookieNotice.close")}
				title={t("cookieNotice.close")}
				onClick={() => {
					dismiss(variant);
					setClosed(true);
				}}
				className="-m-1 inline-flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted hover:bg-paper hover:text-ink"
			>
				<X className="size-4" aria-hidden />
			</button>
		</section>
	);
}
