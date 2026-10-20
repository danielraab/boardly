import { useCallback, useEffect, useState } from "react";

/** A seconds countdown: start(n) counts `remaining` down from n to 0. */
export function useCountdown() {
	const [remaining, setRemaining] = useState(0);

	useEffect(() => {
		if (remaining <= 0) return;
		const id = setTimeout(() => setRemaining((r) => r - 1), 1000);
		return () => clearTimeout(id);
	}, [remaining]);

	const start = useCallback((seconds: number) => setRemaining(seconds), []);
	return { remaining, start };
}
