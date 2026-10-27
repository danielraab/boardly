import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { ImageIcon } from "#/components/ImageIcon";
import { DetailsButton } from "#/features/favourites/FavouriteDetails";
import { LinkView } from "../link/LinkView";
import { sizeStyles, type WidgetSize } from "../sizes";
import type { LinkListData, ListLink } from "./flatten";

export type LinkListLayout = "list" | "compact" | "tiles";

export const linkListLayouts: LinkListLayout[] = ["list", "compact", "tiles"];

/** Icon size, minimum tile width and label size of the tiles of each size. */
const tileStyles: Record<
	WidgetSize,
	{ icon: string; grid: string; text: string }
> = {
	S: {
		icon: "size-6",
		grid: "grid-cols-[repeat(auto-fill,minmax(4.5rem,1fr))]",
		text: "text-xs",
	},
	M: {
		icon: "size-8",
		grid: "grid-cols-[repeat(auto-fill,minmax(6rem,1fr))]",
		text: "text-xs",
	},
	L: {
		icon: "size-10",
		grid: "grid-cols-[repeat(auto-fill,minmax(7.5rem,1fr))]",
		text: "text-sm",
	},
};

/** The props of a link that opens in a new tab, and not at all in edit mode. */
function anchorProps(link: ListLink, editing: boolean) {
	return {
		href: link.url,
		target: "_blank",
		rel: "noopener noreferrer",
		title: link.title ? `${link.title}\n${link.url}` : link.url,
		"aria-disabled": editing || undefined,
		onClick: editing
			? (e: { preventDefault: () => void }) => e.preventDefault()
			: undefined,
	};
}

/**
 * The links of a link list in its layout, below its title as a heading of
 * headingLevel when it has one. In edit mode, links don't navigate. With
 * onDetails, an info button beside each link (on a tile, in its top-right
 * corner) calls it with the link's ID.
 */
export function LinkListView({
	data,
	layout,
	size,
	editing,
	title = null,
	tagFilter = false,
	headingLevel = 2,
	onDetails,
}: {
	data: LinkListData;
	layout: LinkListLayout;
	size: WidgetSize;
	editing: boolean;
	title?: string | null;
	/** Whether the links are filtered by a tag, for the empty message. */
	tagFilter?: boolean;
	headingLevel?: number;
	onDetails?: (id: string) => void;
}) {
	const { t } = useTranslation();
	const style = sizeStyles[size];
	const tile = tileStyles[size];

	let body: ReactNode;
	if ("missing" in data) {
		body = (
			<p className="text-muted">
				{t(
					data.missing === "tag"
						? "dashboards.widgets.linkList.missingTag"
						: "dashboards.widgets.linkList.missing",
				)}
			</p>
		);
	} else if (data.links.length === 0) {
		body = (
			<p className="text-muted">
				{t(
					tagFilter
						? "dashboards.widgets.linkList.emptyTag"
						: "dashboards.widgets.linkList.empty",
				)}
			</p>
		);
	} else if (layout === "tiles") {
		body = (
			<ul className={`grid gap-1 ${tile.grid}`}>
				{data.links.map((link) => (
					<li key={link.id} className="relative min-w-0">
						<a
							{...anchorProps(link, editing)}
							className={`flex min-w-0 flex-col items-center gap-1 rounded-lg p-2 ${tile.text} hover:bg-paper`}
						>
							<ImageIcon imageId={link.iconImageId} className={tile.icon} />
							<span className="w-full truncate text-center">
								{link.title || link.url}
							</span>
						</a>
						{onDetails && (
							<DetailsButton
								link={link}
								onClick={() => onDetails(link.id)}
								className="absolute top-0.5 right-0.5 inline-flex items-center justify-center rounded p-1 text-muted hover:bg-paper hover:text-ink"
								iconClassName="size-3.5"
							/>
						)}
					</li>
				))}
			</ul>
		);
	} else if (layout === "compact") {
		body = (
			<ul>
				{data.links.map((link) => (
					<li
						key={link.id}
						className={
							onDetails ? "flex min-w-0 items-center gap-1" : undefined
						}
					>
						<a
							{...anchorProps(link, editing)}
							className={`flex min-w-0 items-center gap-2 rounded-lg px-1 ${style.row} hover:bg-paper ${onDetails ? "flex-1" : ""}`}
						>
							<ImageIcon imageId={link.iconImageId} className={style.icon} />
							<span className="truncate">{link.title || link.url}</span>
						</a>
						{onDetails && (
							<DetailsButton link={link} onClick={() => onDetails(link.id)} />
						)}
					</li>
				))}
			</ul>
		);
	} else {
		body = (
			<ul>
				{data.links.map((link) => (
					<li key={link.id}>
						<LinkView
							url={link.url}
							title={link.title}
							imageId={link.iconImageId ?? null}
							icon={null}
							target="_blank"
							size={size}
							editing={editing}
							onDetails={onDetails && (() => onDetails(link.id))}
						/>
					</li>
				))}
			</ul>
		);
	}

	const Heading = `h${headingLevel}` as "h2";
	return (
		<div className={style.text}>
			{title && <Heading className="truncate font-medium">{title}</Heading>}
			{body}
		</div>
	);
}
