import { useEffect, useState } from "react";

/** Seconds until the RFC 3339 time `until`; ticks every second while above zero. */
export function useRemainingSeconds(until: string | null) {
	const end = until ? Date.parse(until) : 0;
	const [remaining, setRemaining] = useState(() => secondsUntil(end));

	useEffect(() => {
		setRemaining(secondsUntil(end));
		if (secondsUntil(end) <= 0) return;
		const id = setInterval(() => {
			const left = secondsUntil(end);
			setRemaining(left);
			if (left <= 0) clearInterval(id);
		}, 1000);
		return () => clearInterval(id);
	}, [end]);

	return remaining;
}

function secondsUntil(end: number) {
	return Math.max(0, Math.ceil((end - Date.now()) / 1000));
}

/** Formats seconds as m:ss. */
export function formatCountdown(seconds: number) {
	const m = Math.floor(seconds / 60);
	const s = seconds % 60;
	return `${m}:${String(s).padStart(2, "0")}`;
}
