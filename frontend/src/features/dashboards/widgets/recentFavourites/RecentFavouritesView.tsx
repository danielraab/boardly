import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { LinkView } from "../link/LinkView";
import { fullDate, relativeTime } from "../relativeTime";
import { sizeStyles, type WidgetSize } from "../sizes";
import type { RecentData } from "./recent";

/** The date is one step smaller than the title, like a link's URL. */
const dateText: Record<WidgetSize, string> = {
	S: "text-xs",
	M: "text-xs",
	L: "text-sm",
};

/**
 * The links of a recent favourites widget, each with when it was added, below
 * its title as a heading of headingLevel when it has one. In edit mode, links
 * don't navigate. With onDetails, an info button beside each link calls it
 * with the link's ID. now is the time the dates are relative to.
 */
export function RecentFavouritesView({
	data,
	days,
	size,
	editing,
	title = null,
	headingLevel = 2,
	onDetails,
	now = new Date(),
}: {
	data: RecentData;
	days: number | null;
	size: WidgetSize;
	editing: boolean;
	title?: string | null;
	headingLevel?: number;
	onDetails?: (id: string) => void;
	now?: Date;
}) {
	const { t, i18n } = useTranslation();
	const style = sizeStyles[size];

	let body: ReactNode;
	if ("missing" in data) {
		body = (
			<p className="text-muted">
				{t("dashboards.widgets.recentFavourites.missing")}
			</p>
		);
	} else if (data.links.length === 0) {
		body = (
			<p className="text-muted">
				{days === null
					? t("dashboards.widgets.recentFavourites.empty")
					: t("dashboards.widgets.recentFavourites.emptyDays", {
							count: days,
						})}
			</p>
		);
	} else {
		body = (
			<ul>
				{data.links.map((link) => {
					const added = new Date(link.createdAt);
					return (
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
								aside={
									<time
										dateTime={link.createdAt}
										title={fullDate(added, i18n.language)}
										className={`ml-auto shrink-0 pl-2 whitespace-nowrap text-muted ${dateText[size]}`}
									>
										{relativeTime(added, now, i18n.language)}
									</time>
								}
							/>
						</li>
					);
				})}
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
