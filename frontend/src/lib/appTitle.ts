const defaultAppTitle = "Boardly";

/**
 * The app title the server writes into index.html's application-name meta tag
 * (BOARDLY_APP_TITLE), or "Boardly" without one, e.g. in the Vite dev server.
 */
export function appTitle(): string {
	const content = document
		.querySelector('meta[name="application-name"]')
		?.getAttribute("content")
		?.trim();
	return content || defaultAppTitle;
}
