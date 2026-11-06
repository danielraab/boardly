// The analytics consent banner and settings, built on vanilla-cookieconsent.
// Boardly builds the whole library config itself, so texts, look and the
// privacy link always match the app. Loaded only when consent is on.
import "vanilla-cookieconsent/dist/cookieconsent.css";
import "./consent.css";
import i18next, { type TFunction } from "i18next";
import * as CookieConsent from "vanilla-cookieconsent";
import { appTitle } from "#/lib/appTitle";
import { legalLink } from "#/lib/legalLinks";
import { getThemeMode, onThemeChange } from "#/theme";
import type { ConsentOptions } from "./index";
import { type ConsentSettings, cookiePattern } from "./settings";

/** The consent category the snippet's scripts carry in data-category. */
const analytics = "analytics";

/** Marks links inside the banner that open an in-app page. */
const routeAttr = "data-boardly-route";

/** Undoes start(); set while the manager runs. */
let stop: (() => void) | undefined;

function escapeHtml(s: string): string {
	return s
		.replace(/&/g, "&amp;")
		.replace(/</g, "&lt;")
		.replace(/>/g, "&gt;")
		.replace(/"/g, "&quot;")
		.replace(/'/g, "&#39;");
}

/** The privacy policy link as HTML, or "" without a privacy policy. */
function privacyLinkHtml(t: TFunction): string {
	const link = legalLink("privacy");
	if (!link) return "";
	const text = escapeHtml(t("consent.privacy"));
	const href = escapeHtml(link.href);
	return link.kind === "page"
		? `<a href="${href}" ${routeAttr}>${text}</a>`
		: `<a href="${href}" target="_blank" rel="noopener noreferrer">${text}</a>`;
}

/** The banner and settings texts in one language, from Boardly's translations. */
export function translation(
	t: TFunction,
	settings: ConsentSettings,
): CookieConsent.Translation {
	const vars = { service: settings.service, appName: appTitle() };
	// i18next doesn't escape (escapeValue: false), but the library takes HTML.
	const tr = t as unknown as (
		key: `consent.${string}`,
		options: object,
	) => string;
	const html = (key: `consent.${string}`) => escapeHtml(tr(key, vars));
	const privacy = privacyLinkHtml(t);
	return {
		consentModal: {
			title: html("consent.title"),
			description: html("consent.text"),
			acceptAllBtn: html("consent.accept"),
			acceptNecessaryBtn: html("consent.reject"),
			showPreferencesBtn: html("consent.settings"),
			footer: privacy,
		},
		preferencesModal: {
			title: html("consent.settingsTitle"),
			acceptAllBtn: html("consent.acceptAll"),
			acceptNecessaryBtn: html("consent.rejectAll"),
			savePreferencesBtn: html("consent.save"),
			closeIconLabel: html("consent.close"),
			sections: [
				{
					description: privacy
						? `${html("consent.intro")} ${privacy}`
						: html("consent.intro"),
				},
				{
					title: html("consent.necessaryTitle"),
					description: html("consent.necessaryText"),
					linkedCategory: "necessary",
				},
				{
					title: html("consent.analyticsTitle"),
					description: html("consent.analyticsText"),
					linkedCategory: analytics,
					...(settings.cookies.length > 0 && {
						cookieTable: {
							headers: { name: html("consent.cookie") },
							body: settings.cookies.map((name) => ({
								name: escapeHtml(name),
							})),
						},
					}),
				},
			],
		},
	};
}

/**
 * The domains a cookie for hostname can be set on, without the host itself
 * and the last label: `a.dash.example.org` → [`dash.example.org`,
 * `example.org`]. None for IP addresses and single-label hosts.
 */
export function parentDomains(hostname: string): string[] {
	if (/^[\d.]+$/.test(hostname) || hostname.includes(":")) return [];
	const labels = hostname.split(".");
	const out: string[] = [];
	for (let i = 1; i < labels.length - 1; i++)
		out.push(labels.slice(i).join("."));
	return out;
}

/**
 * Deletes the cookies matching the configured names, host-only and on the
 * host and every parent domain, where analytics tools usually set them.
 * Expiring a cookie on a domain the page can't use is a no-op.
 */
/**
 * Writes a cookie string. The Cookie Store API isn't in every browser yet,
 * and expiring cookies on other domains needs the raw attributes.
 */
export function writeCookie(cookie: string) {
	// biome-ignore lint/suspicious/noDocumentCookie: see above
	document.cookie = cookie;
}

export function clearAnalyticsCookies(
	names: string[],
	hostname = location.hostname,
) {
	const patterns = names.map(cookiePattern);
	const present = document.cookie
		.split(";")
		.map((c) => c.split("=")[0]?.trim() ?? "")
		.filter((name) => name && patterns.some((p) => p.test(name)));
	const expired = "=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
	for (const name of present) {
		writeCookie(name + expired);
		for (const domain of [hostname, ...parentDomains(hostname)]) {
			writeCookie(`${name}${expired}; domain=.${domain}`);
		}
	}
}

function syncDarkMode() {
	document.documentElement.classList.toggle(
		"cc--darkmode",
		getThemeMode() === "dark",
	);
}

/** Runs the consent manager: shows the banner when the visitor hasn't chosen in this revision. */
export async function start(
	settings: ConsentSettings,
	options: ConsentOptions,
) {
	if (stop) return;

	syncDarkMode();
	const stopTheme = onThemeChange(syncDarkMode);

	// In-app links (the privacy page) open without reloading the app.
	const onClick = (event: MouseEvent) => {
		const a = (event.target as Element | null)?.closest?.(
			`#cc-main a[${routeAttr}]`,
		);
		if (!a || event.defaultPrevented || event.button !== 0) return;
		if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)
			return;
		event.preventDefault();
		options.navigate(a.getAttribute("href") ?? "/");
	};
	document.addEventListener("click", onClick);

	const onLanguage = () => {
		void CookieConsent.setLanguage(language(), true);
	};
	i18next.on("languageChanged", onLanguage);

	stop = () => {
		stopTheme();
		document.removeEventListener("click", onClick);
		i18next.off("languageChanged", onLanguage);
		document.documentElement.classList.remove("cc--darkmode");
		CookieConsent.reset(true);
		stop = undefined;
	};

	const language = () => (i18next.resolvedLanguage === "de" ? "de" : "en");

	await CookieConsent.run({
		revision: settings.revision,
		hideFromBots: false,
		disablePageInteraction: false,
		cookie: {
			name: "boardly_consent",
			expiresAfterDays: 182,
			sameSite: "Lax",
			secure: location.protocol === "https:",
		},
		guiOptions: {
			consentModal: {
				layout: "box",
				position: "bottom right",
				equalWeightButtons: true,
			},
			preferencesModal: { layout: "box", equalWeightButtons: true },
		},
		categories: {
			necessary: { enabled: true, readOnly: true },
			[analytics]: {
				autoClear: {
					cookies: settings.cookies.map((name) => ({
						name: cookiePattern(name),
					})),
				},
			},
		},
		onChange: ({ changedCategories }) => {
			if (
				changedCategories.includes(analytics) &&
				!CookieConsent.acceptedCategory(analytics)
			) {
				// Scripts that already ran keep running until the page reloads.
				clearAnalyticsCookies(settings.cookies);
				location.reload();
			}
		},
		language: {
			default: language(),
			translations: {
				en: translation(i18next.getFixedT("en"), settings),
				de: translation(i18next.getFixedT("de"), settings),
			},
		},
	});
}

/** Stops the manager and forgets its state and the stored choice; for tests. */
export function stopConsent() {
	stop?.();
}

/** Opens the consent settings dialog. */
export function showSettings() {
	CookieConsent.showPreferences();
}
