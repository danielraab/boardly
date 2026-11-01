import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useDebounced } from "#/lib/useDebounced";
import { favouritesQueryOptions } from "./api";
import { domainCount, domainOf } from "./tree";

const DEBOUNCE_MS = 250;

/**
 * A note under a URL field with how many of the user's links are on the
 * entered URL's domain, linking to /favourites filtered by it in a new tab so
 * the form stays as it is. With excludeId, that link isn't counted. Renders
 * nothing for an invalid URL, while the tree loads, or when no link matches.
 */
export function DomainHint({
	id,
	url,
	excludeId,
}: {
	id?: string;
	url: string;
	excludeId?: string;
}) {
	const { t } = useTranslation();
	const tree = useQuery(favouritesQueryOptions);
	const domain = domainOf(useDebounced(url, DEBOUNCE_MS));
	const count = useMemo(
		() => (tree.data && domain ? domainCount(tree.data, domain, excludeId) : 0),
		[tree.data, domain, excludeId],
	);
	if (!domain || count === 0) return null;
	return (
		<p id={id} className="mt-1 text-sm text-muted">
			<a
				href={`/favourites?domain=${encodeURIComponent(domain)}`}
				target="_blank"
				rel="noopener"
				className="text-accent underline-offset-2 hover:underline"
			>
				{t(
					excludeId
						? "favourites.domainHint.others"
						: "favourites.domainHint.count",
					{ count, domain },
				)}
			</a>
		</p>
	);
}
