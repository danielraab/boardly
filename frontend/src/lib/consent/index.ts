// Starts the analytics consent manager when the operator turned consent on.
// The manager and vanilla-cookieconsent are a separate chunk, so instances
// without consent never download them.
import { consentSettings } from "./settings";

export type ConsentOptions = {
	/** Opens an in-app path, such as the privacy page, without a reload. */
	navigate: (path: string) => void;
};

type Manager = typeof import("./manager");

let manager: Promise<Manager> | undefined;
let started: ConsentOptions | undefined;

function load(): Promise<Manager> {
	manager ??= import("./manager");
	return manager;
}

/** Shows the consent banner when needed; does nothing when consent is off. */
export async function startConsent(options: ConsentOptions): Promise<void> {
	const settings = consentSettings();
	if (!settings) return;
	started = options;
	await (await load()).start(settings, options);
}

/** Opens the consent settings dialog; does nothing when consent is off. */
export async function openConsentSettings(): Promise<void> {
	const settings = consentSettings();
	if (!settings) return;
	const m = await load();
	// start() does nothing when it already ran; without it the dialog can't open.
	await m.start(
		settings,
		started ?? { navigate: (path) => location.assign(path) },
	);
	m.showSettings();
}
