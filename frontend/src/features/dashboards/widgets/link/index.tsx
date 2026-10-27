import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "lucide-react";
import { useState } from "react";
import { shownIcon } from "#/components/IconChooser";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { useCurrentDashboard } from "#/features/dashboards/currentDashboard";
import { favouritesQueryOptions } from "#/features/favourites/api";
import { FavouriteDetailsDialog } from "#/features/favourites/FavouriteDetails";
import { findNode } from "#/features/favourites/tree";
import type { WidgetKind, WidgetViewProps } from "../registry";
import { iconSourceOf, type LinkConfig, LinkSettings } from "./LinkSettings";
import { LinkView } from "./LinkView";
import { useFavicon } from "./useFavicon";

/**
 * What the server resolves: a favourite's current URL and title, and the
 * website icon of the link's site.
 */
type LinkData = { url?: string; title?: string | null; iconImageId?: string };

/**
 * A favourite's current URL and title and the website icon: in view mode from
 * the server's data, in edit mode from the owner's favourites and a favicon
 * lookup. When the favourite is gone, the copies in the config. In view mode
 * the owner can open the favourite's details.
 */
function View({ widget, editing }: WidgetViewProps) {
	const queryClient = useQueryClient();
	const config = widget.config as LinkConfig;
	const isFavourite = config.source === "favourite";
	const isOwner = useCurrentDashboard()?.isOwner ?? false;
	const [showDetails, setShowDetails] = useState(false);
	const favourites = useQuery({
		...favouritesQueryOptions,
		enabled: (editing || isOwner) && isFavourite,
	});
	const data = widget.data as LinkData | undefined;
	let url = config.url;
	let favouriteTitle: string | null = null;
	let knownIcon: string | undefined;
	if (isFavourite) {
		favouriteTitle = config.favouriteTitle ?? null;
		if (!editing && data?.url) {
			url = data.url;
			favouriteTitle = data.title ?? null;
		} else if (editing && favourites.data && config.favouriteId) {
			const node = findNode(favourites.data, config.favouriteId);
			if (node?.kind === "link" && node.url) {
				url = node.url;
				favouriteTitle = node.title;
				knownIcon = node.iconImageId;
			}
		}
	}
	const canShowDetails =
		isOwner &&
		!editing &&
		isFavourite &&
		!!favourites.data &&
		!!config.favouriteId &&
		findNode(favourites.data, config.favouriteId)?.kind === "link";
	const source = iconSourceOf(config);
	// A draft has no data yet, so edit mode looks the website icon up itself.
	const lookedUp = useFavicon(
		editing && source === "favicon" ? url : null,
		knownIcon,
	);
	const faviconId = editing ? lookedUp.id : (data?.iconImageId ?? null);
	const shown = shownIcon(
		{ source, imageId: config.imageId, icon: config.icon },
		faviconId,
	);
	return (
		<>
			<LinkView
				url={url}
				title={config.title ?? favouriteTitle}
				imageId={shown.imageId}
				icon={shown.icon}
				target={config.target}
				size={config.size}
				editing={editing}
				onDetails={canShowDetails ? () => setShowDetails(true) : undefined}
			/>
			{canShowDetails && (
				<FavouriteDetailsDialog
					favouriteId={showDetails ? (config.favouriteId ?? null) : null}
					onSaved={() =>
						void queryClient.invalidateQueries({
							queryKey: dashboardViewQueryKey,
						})
					}
					onClose={() => setShowDetails(false)}
				/>
			)}
		</>
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
		iconSource: "favicon",
		imageId: null,
		icon: null,
		target: "_blank",
		size: "M",
	}),
	View,
	Settings: LinkSettings,
};
