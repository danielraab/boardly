import { useEffect, useState } from "react";

/** Waits until value has been stable for delay ms. */
export function useDebounced<T>(value: T, delay: number) {
	const [debounced, setDebounced] = useState(value);
	useEffect(() => {
		const id = setTimeout(() => setDebounced(value), delay);
		return () => clearTimeout(id);
	}, [value, delay]);
	return debounced;
}
