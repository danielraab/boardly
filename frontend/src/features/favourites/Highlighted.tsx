import { highlight } from "./tree";

/** text with the parts that match a search term marked; plain text without terms. */
export function Highlighted({
	text,
	terms,
}: {
	text: string;
	terms: string[];
}) {
	if (terms.length === 0) return text;
	return highlight(text, terms).map((part, index) =>
		part.match ? (
			// biome-ignore lint/suspicious/noArrayIndexKey: parts have no other identity
			<mark key={index} className="rounded-sm bg-accent/25 text-inherit">
				{part.text}
			</mark>
		) : (
			part.text
		),
	);
}
