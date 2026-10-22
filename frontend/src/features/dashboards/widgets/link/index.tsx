import { useQuery } from "@tanstack/react-query";
import { Link } from "lucide-react";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { findNode } from "#/features/favourites/tree";
import type { WidgetKind, WidgetViewProps } from "../registry";
import { type LinkConfig, LinkSettings } from "./LinkSettings";
import { LinkView } from "./LinkView";

/** The current URL and title of a link widget's favourite, from the server. */
type LinkData = { url: string; title: string | null };

/**
 * A favourite's current URL and title: in view mode from the server's data,
 * in edit mode from the owner's favourites. When the favourite is gone, the
 * copies in the config.
 */
function View({ widget, editing }: WidgetViewProps) {
	const config = widget.config as LinkConfig;
	const isFavourite = config.source === "favourite";
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: editing && isFavourite,
	});
	let current: LinkData | undefined;
	if (isFavourite) {
		current = { url: config.url, title: config.favouriteTitle ?? null };
		if (!editing && widget.data) {
			current = widget.data as LinkData;
		} else if (editing && favourites.data && config.favouriteId) {
			const node = findNode(favourites.data, config.favouriteId);
			if (node?.kind === "link" && node.url) {
				current = { url: node.url, title: node.title };
			}
		}
	}
	return (
		<LinkView
			url={current?.url ?? config.url}
			title={config.title ?? current?.title ?? null}
			imageId={config.imageId}
			icon={config.icon}
			target={config.target}
			size={config.size}
			editing={editing}
		/>
	);
}

export const linkKind: WidgetKind = {
	kind: "link",
	icon: Link,
	labelKey: "dashboards.widgets.link.name",
	descriptionKey: "dashboards.widgets.link.description",
	defaultConfig: (): LinkConfig => ({
		source: "favourite",
		url: "",
		title: null,
		imageId: null,
		icon: null,
		target: "_blank",
		size: "M",
	}),
	View,
	Settings: LinkSettings,
};
