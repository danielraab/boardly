import type { ReactNode } from "react";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { DetailsButton } from "#/features/favourites/FavouriteDetails";
import { sizeStyles, type WidgetSize } from "../sizes";

/** The URL line is one step smaller than the title. */
const urlText: Record<WidgetSize, string> = {
	S: "text-xs",
	M: "text-xs",
	L: "text-sm",
};

/**
 * One link: its icon, then its title and URL on two lines, or only the URL.
 * In edit mode it doesn't navigate. With onDetails, an info button right of
 * the link calls it. aside is shown at the right end of the link.
 */
export function LinkView({
	url,
	title,
	imageId,
	icon,
	target,
	size,
	editing,
	onDetails,
	aside,
}: {
	url: string;
	title: string | null;
	imageId: string | null;
	icon: string | null;
	target: "_blank" | "_self";
	size: WidgetSize;
	editing: boolean;
	onDetails?: () => void;
	aside?: ReactNode;
}) {
	const style = sizeStyles[size];
	const link = (
		<a
			href={url}
			target={target}
			rel={target === "_blank" ? "noopener noreferrer" : undefined}
			title={title ? `${title}\n${url}` : url}
			aria-disabled={editing || undefined}
			onClick={editing ? (e) => e.preventDefault() : undefined}
			className={`flex min-w-0 items-center gap-2 rounded-lg px-1 ${style.row} ${style.text} hover:bg-paper ${onDetails ? "flex-1" : ""}`}
		>
			<ChosenIcon imageId={imageId} icon={icon} className={style.icon} />
			<span className="flex min-w-0 flex-col">
				{title && <span className="truncate font-medium">{title}</span>}
				<span
					className={`truncate ${title ? `${urlText[size]} text-muted` : ""}`}
				>
					{url}
				</span>
			</span>
			{aside}
		</a>
	);
	if (!onDetails) return link;
	return (
		<div className="flex min-w-0 items-center gap-1">
			{link}
			<DetailsButton link={{ title, url }} onClick={onDetails} />
		</div>
	);
}
