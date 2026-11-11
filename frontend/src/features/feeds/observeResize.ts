/**
 * One ResizeObserver shared by all callers, so that a long list of rows
 * doesn't create one observer per row. Returns a function that stops
 * observing. Does nothing where ResizeObserver doesn't exist.
 */
const callbacks = new Map<Element, () => void>();
let observer: ResizeObserver | undefined;

export function observeResize(el: Element, callback: () => void): () => void {
	if (typeof ResizeObserver === "undefined") return () => {};
	observer ??= new ResizeObserver((entries) => {
		for (const entry of entries) callbacks.get(entry.target)?.();
	});
	callbacks.set(el, callback);
	observer.observe(el);
	return () => {
		callbacks.delete(el);
		observer?.unobserve(el);
	};
}
