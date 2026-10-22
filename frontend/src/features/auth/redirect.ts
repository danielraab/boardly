/**
 * Returns value if it is a path on this site that the login page may continue
 * to after signing in, and undefined otherwise (absolute and protocol-relative
 * URLs, the login pages themselves, anything that isn't a string).
 */
export function safeRedirect(value: unknown): string | undefined {
	if (typeof value !== "string" || !value.startsWith("/")) return undefined;
	if (value.startsWith("//") || value.startsWith("/\\")) return undefined;
	if (/^\/login(?:[/?#]|$)/.test(value)) return undefined;
	return value;
}
