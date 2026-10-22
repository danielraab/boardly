import i18next, { type i18n } from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import { initReactI18next } from "react-i18next";
import de from "#/locales/de.json";
import en from "#/locales/en.json";

export const languages = [
	{ code: "en", label: "English" },
	{ code: "de", label: "Deutsch" },
] as const;

export type LanguageCode = (typeof languages)[number]["code"];

export const LANGUAGE_STORAGE_KEY = "famodash.lang";

export const resources = {
	en: { translation: en },
	de: { translation: de },
} as const;

declare module "i18next" {
	interface CustomTypeOptions {
		resources: (typeof resources)["en"];
	}
}

/**
 * Initializes an i18next instance. The detector reads an explicit choice from
 * localStorage first, then the browser languages; detected languages are not
 * cached, only explicit choices via setLanguage.
 */
export function initI18n(instance: i18n = i18next): i18n {
	instance
		.use(LanguageDetector)
		.use(initReactI18next)
		.init({
			resources,
			supportedLngs: languages.map((l) => l.code),
			fallbackLng: "en",
			interpolation: { escapeValue: false },
			detection: {
				order: ["localStorage", "navigator"],
				lookupLocalStorage: LANGUAGE_STORAGE_KEY,
				caches: [],
				convertDetectedLanguage: (lng) => lng.split("-")[0],
			},
		});

	// The manifest gives the installed app its name and description in this language.
	const syncDocumentLanguage = () => {
		const language = instance.resolvedLanguage ?? "en";
		document.documentElement.lang = language;
		document
			.querySelector('link[rel="manifest"]')
			?.setAttribute("href", `/manifest.${language}.webmanifest`);
	};
	syncDocumentLanguage();
	instance.on("languageChanged", syncDocumentLanguage);
	return instance;
}

/** Switches the language and remembers the explicit choice in this browser. */
export async function setLanguage(
	code: LanguageCode,
	instance: i18n = i18next,
): Promise<void> {
	try {
		localStorage.setItem(LANGUAGE_STORAGE_KEY, code);
	} catch {
		// Storage can be unavailable (private mode); the switch still applies to this visit.
	}
	await instance.changeLanguage(code);
}

export default i18next;
