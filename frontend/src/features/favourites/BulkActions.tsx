import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FolderInput, Tag, TagsIcon, Trash2, X } from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import {
	type BulkOp,
	bulkFavourites,
	type FavouriteNode,
	favouritesQueryKey,
	MAX_BULK,
} from "./api";
import { FavouritePicker } from "./FavouritePicker";
import { TagInput } from "./TagInput";

export type TagAction = "addTags" | "removeTags" | "setTags";
export type BulkDialogKind = TagAction | "move";

const titles = {
	addTags: "favourites.bulk.addTagsTitle",
	removeTags: "favourites.bulk.removeTagsTitle",
	setTags: "favourites.bulk.setTagsTitle",
	move: "favourites.bulk.moveTitle",
} as const;

const barButton = `${secondaryButton} bg-surface px-3`;

/** The bar shown while anything is selected: counts and the bulk actions. */
export function SelectionBar({
	links,
	folders,
	moveCount,
	onAction,
	onDelete,
}: {
	links: number;
	folders: number;
	/** How many nodes a move would send. */
	moveCount: number;
	onAction: (kind: BulkDialogKind) => void;
	onDelete: () => void;
}) {
	const { t } = useTranslation();
	const parts = [
		links > 0 && t("favourites.linkCount", { count: links }),
		folders > 0 && t("favourites.folderCount", { count: folders }),
	].filter(Boolean);
	const tooMany = links > MAX_BULK || moveCount > MAX_BULK;
	const noLinks = links === 0 || links > MAX_BULK;
	return (
		<div className="sticky top-2 z-10 mb-3 rounded-xl border border-line bg-paper p-3 shadow-[0_8px_24px_-12px_var(--color-shadow)]">
			<p className="text-sm font-medium" aria-live="polite">
				{t("favourites.bulk.selected", {
					items: parts.join(t("favourites.bulk.and")),
				})}
			</p>
			{tooMany && (
				<p className="mt-1 text-sm text-danger">
					{t("favourites.bulk.tooMany", { max: MAX_BULK })}
				</p>
			)}
			<div className="mt-2 flex flex-wrap gap-2">
				<button
					type="button"
					className={barButton}
					disabled={noLinks}
					onClick={() => onAction("addTags")}
				>
					<Tag className="size-4" aria-hidden />
					{t("favourites.bulk.addTags")}
				</button>
				<button
					type="button"
					className={barButton}
					disabled={noLinks}
					onClick={() => onAction("removeTags")}
				>
					<X className="size-4" aria-hidden />
					{t("favourites.bulk.removeTags")}
				</button>
				<button
					type="button"
					className={barButton}
					disabled={noLinks}
					onClick={() => onAction("setTags")}
				>
					<TagsIcon className="size-4" aria-hidden />
					{t("favourites.bulk.setTags")}
				</button>
				<button
					type="button"
					className={barButton}
					disabled={moveCount === 0 || moveCount > MAX_BULK}
					onClick={() => onAction("move")}
				>
					<FolderInput className="size-4" aria-hidden />
					{t("favourites.bulk.move")}
				</button>
				<button
					type="button"
					className={`${barButton} hover:text-danger`}
					disabled={noLinks}
					onClick={onDelete}
				>
					<Trash2 className="size-4" aria-hidden />
					{t("favourites.bulk.delete", { count: links })}
				</button>
			</div>
		</div>
	);
}

export type BulkDialogTarget =
	| {
			kind: TagAction;
			links: FavouriteNode[];
			/** The tags of the links with how many have each, for removing. */
			tags: { name: string; count: number }[];
	  }
	| {
			kind: "move";
			nodes: FavouriteNode[];
			/** Folders that can't be the target, with everything below them. */
			blocked: ReadonlySet<string>;
	  };

/** The dialog of a tag action or "Move to…" for the selected nodes. */
export function BulkDialog({
	target,
	onClose,
}: {
	target: BulkDialogTarget | null;
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={target !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{target && (
						<BulkForm key={target.kind} target={target} onClose={onClose} />
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function BulkForm({
	target,
	onClose,
}: {
	target: BulkDialogTarget;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [tags, setTags] = useState<string[]>([]);
	const [parentId, setParentId] = useState("");

	const run = useMutation({
		mutationFn: (op: BulkOp) => bulkFavourites(op),
		onSuccess: async () => {
			await queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
			onClose();
		},
		onError: (e) => {
			// Something changed elsewhere: show the current tree.
			if (e instanceof ApiError && e.status === 404) {
				void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			}
		},
	});

	let body: ReactNode;
	let submit: (() => void) | null = null;
	let confirm = t("favourites.bulk.apply");
	if (target.kind === "move") {
		const ids = target.nodes.map((n) => n.id);
		body = (
			<FavouritePicker
				id="bulk-move"
				label={t("favourites.dialog.folderLabel")}
				mode="folder"
				value={parentId}
				onChange={setParentId}
				disabledIds={target.blocked}
			/>
		);
		if (parentId) submit = () => run.mutate({ action: "move", ids, parentId });
		confirm = t("favourites.bulk.moveConfirm");
	} else {
		const ids = target.links.map((n) => n.id);
		if (target.kind === "removeTags") {
			body = (
				<fieldset>
					<legend className="block text-sm font-medium">
						{t("favourites.bulk.removeLabel")}
					</legend>
					<div className="mt-2 max-h-60 space-y-1 overflow-auto">
						{target.tags.map(({ name, count }) => (
							<label
								key={name}
								className="flex cursor-pointer items-center gap-3 rounded-md px-2 py-1.5 text-sm hover:bg-paper"
							>
								<input
									type="checkbox"
									checked={tags.includes(name)}
									onChange={(e) =>
										setTags(
											e.target.checked
												? [...tags, name]
												: tags.filter((tag) => tag !== name),
										)
									}
									className="accent-ink"
								/>
								<span className="min-w-0 flex-1 truncate">{name}</span>
								<span className="text-xs text-muted tabular-nums">{count}</span>
							</label>
						))}
						{target.tags.length === 0 && (
							<p className="text-sm text-muted">
								{t("favourites.bulk.noTags")}
							</p>
						)}
					</div>
				</fieldset>
			);
			if (tags.length > 0) {
				submit = () => run.mutate({ action: "removeTags", ids, tags });
			}
		} else {
			body = (
				<>
					<TagInput value={tags} onChange={setTags} />
					{target.kind === "setTags" && (
						<p className="mt-3 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm">
							{t("favourites.bulk.setWarning")}
						</p>
					)}
				</>
			);
			if (tags.length > 0 || target.kind === "setTags") {
				submit = () => run.mutate({ action: target.kind, ids, tags });
			}
		}
	}
	const count =
		target.kind === "move" ? target.nodes.length : target.links.length;

	return (
		<form
			onSubmit={(e) => {
				e.preventDefault();
				submit?.();
			}}
		>
			<Dialog.Title className="text-lg font-semibold">
				{t(titles[target.kind])}
			</Dialog.Title>
			<Dialog.Description className="mt-1 text-sm text-muted">
				{target.kind === "move"
					? t("favourites.bulk.appliesToItems", { count })
					: t("favourites.bulk.appliesToLinks", { count })}
			</Dialog.Description>
			<div className="mt-4">{body}</div>
			{run.isError && (
				<p role="alert" className="mt-3 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			<div className="mt-6 flex justify-end gap-3">
				<Dialog.Close className={secondaryButton}>
					{t("favourites.dialog.cancel")}
				</Dialog.Close>
				<button
					type="submit"
					className={target.kind === "setTags" ? dangerButton : primaryButton}
					disabled={!submit || run.isPending}
				>
					{confirm}
				</button>
			</div>
		</form>
	);
}
