import { Dialog } from "@base-ui/react/dialog";
import { useQuery } from "@tanstack/react-query";
import { ExternalLink, Info, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { ImageIcon, imageUrl } from "#/components/ImageIcon";
import { type FavouriteNode, favouritesQueryOptions } from "./api";
import { NodeDialog } from "./NodeDialog";
import { findNode, label } from "./tree";

/**
 * The info button that opens a link's details, named after the link.
 * className replaces the button's classes, iconClassName sets the icon's size.
 */
export function DetailsButton({
	link,
	onClick,
	className = iconButton,
	iconClassName = "size-4",
}: {
	link: { title: string | null; url?: string };
	onClick: () => void;
	className?: string;
	iconClassName?: string;
}) {
	const { t } = useTranslation();
	const name = t("favourites.details.button", {
		title: link.title || link.url,
	});
	return (
		<button
			type="button"
			onClick={onClick}
			aria-label={name}
			title={name}
			className={className}
		>
			<Info className={iconClassName} aria-hidden />
		</button>
	);
}

/**
 * The owner's link favouriteId from their favourites tree, or undefined while
 * the tree loads or when it isn't a link there.
 */
export function useFavouriteLink(favouriteId: string | null | undefined) {
	const favourites = useQuery(favouritesQueryOptions);
	if (!favourites.data || !favouriteId) return undefined;
	const node = findNode(favourites.data, favouriteId);
	return node?.kind === "link" ? node : undefined;
}

/**
 * The read-only details of the owner's link favouriteId while it is set, with
 * actions to open and edit it. Editing swaps the details for the link's edit
 * dialog and returns to them on close; onSaved runs after a save.
 */
export function FavouriteDetailsDialog({
	favouriteId,
	onClose,
	onSaved,
}: {
	favouriteId: string | null;
	onClose: () => void;
	onSaved?: () => void;
}) {
	const link = useFavouriteLink(favouriteId);
	const [editing, setEditing] = useState(false);
	return (
		<>
			<Dialog.Root
				open={link !== undefined && !editing}
				onOpenChange={(open) => !open && onClose()}
			>
				<Dialog.Portal>
					<Dialog.Backdrop className={dialogBackdrop} />
					<Dialog.Popup className={dialogPopup}>
						{link && <Details link={link} onEdit={() => setEditing(true)} />}
					</Dialog.Popup>
				</Dialog.Portal>
			</Dialog.Root>
			<NodeDialog
				target={editing && link ? { node: link } : null}
				onSaved={onSaved}
				onClose={() => setEditing(false)}
			/>
		</>
	);
}

function Details({
	link,
	onEdit,
}: {
	link: FavouriteNode;
	onEdit: () => void;
}) {
	const { t } = useTranslation();
	const url = link.url ?? "";
	return (
		<>
			<div className="flex items-start gap-2">
				<ImageIcon imageId={link.iconImageId} className="mt-1 size-5" />
				<Dialog.Title className="min-w-0 flex-1 text-lg font-semibold break-words">
					{label(link)}
				</Dialog.Title>
				<Dialog.Close
					aria-label={t("favourites.details.close")}
					className={`${iconButton} -mt-1 -mr-2`}
				>
					<X className="size-4" aria-hidden />
				</Dialog.Close>
			</div>
			<div className="mt-4 space-y-4">
				{link.previewImageId && (
					<img
						src={imageUrl(link.previewImageId)}
						alt={t("favourites.details.previewAlt")}
						className="max-h-48 w-full rounded-lg border border-line object-contain"
					/>
				)}
				<a
					href={url}
					target="_blank"
					rel="noopener noreferrer"
					className="block text-sm break-all hover:underline"
				>
					{url}
				</a>
				{link.description ? (
					<p className="text-sm break-words whitespace-pre-line">
						{link.description}
					</p>
				) : (
					<p className="text-sm text-muted">
						{t("favourites.details.noDescription")}
					</p>
				)}
				{link.tags && link.tags.length > 0 && (
					<ul
						aria-label={t("favourites.details.tags")}
						className="flex flex-wrap gap-1.5"
					>
						{link.tags.map((tag) => (
							<li
								key={tag}
								className="rounded-full bg-paper px-2 py-0.5 text-xs text-muted"
							>
								{tag}
							</li>
						))}
					</ul>
				)}
			</div>
			<div className="mt-6 flex justify-end gap-3">
				<button type="button" className={secondaryButton} onClick={onEdit}>
					{t("favourites.details.edit")}
				</button>
				<a
					href={url}
					target="_blank"
					rel="noopener noreferrer"
					className={primaryButton}
				>
					<ExternalLink className="size-4" aria-hidden />
					{t("favourites.details.open")}
				</a>
			</div>
		</>
	);
}
