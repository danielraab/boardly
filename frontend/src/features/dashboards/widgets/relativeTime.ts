const dayMs = 24 * 60 * 60 * 1000;

const units: [Intl.RelativeTimeFormatUnit, number][] = [
	["year", 365 * dayMs],
	["month", 30 * dayMs],
	["day", dayMs],
	["hour", 60 * 60 * 1000],
	["minute", 60 * 1000],
];

/**
 * How long ago date was, in the largest unit that fits, like "3 days ago" or
 * "yesterday"; "now" under a minute.
 */
export function relativeTime(date: Date, now: Date, language: string): string {
	const format = new Intl.RelativeTimeFormat(language, { numeric: "auto" });
	const ago = Math.max(0, now.getTime() - date.getTime());
	for (const [unit, ms] of units) {
		if (ago >= ms) return format.format(-Math.floor(ago / ms), unit);
	}
	return format.format(0, "second");
}

/** The full date and time of date, for a tooltip. */
export function fullDate(date: Date, language: string): string {
	return new Intl.DateTimeFormat(language, {
		dateStyle: "medium",
		timeStyle: "short",
	}).format(date);
}
