import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, Trash2 } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { ImageIcon, imageUrl } from "#/components/ImageIcon";
import { ApiError } from "#/lib/api";
import {
	createFavourite,
	type FavouriteNode,
	favouritesQueryKey,
	favouritesQueryOptions,
	refreshFavicon,
	updateFavourite,
} from "./api";
import { LinkFields } from "./LinkFields";
import { findNode, label } from "./tree";
import {
	checkDescription,
	checkFolderName,
	checkLinkTitle,
	checkTag,
	checkUrl,
	type FieldError,
	fieldErrorFromCode,
	MAX_FOLDER_NAME,
} from "./validate";

/** What the dialog does: create a node in a folder, or edit a node. */
export type NodeDialogTarget =
	| { kind: "folder" | "link"; parent: FavouriteNode; parentLabel: string }
	| { node: FavouriteNode };

/**
 * Creates or edits a favourite. With chooseFolder, a new node's folder can be
 * chosen, starting at the target's parent. onSaved gets the saved node.
 */
export function NodeDialog({
	target,
	onClose,
	chooseFolder = false,
	onSaved,
}: {
	target: NodeDialogTarget | null;
	onClose: () => void;
	chooseFolder?: boolean;
	onSaved?: (node: FavouriteNode) => void;
}) {
	return (
		<Dialog.Root
			open={target !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{/* Remount per target so the form starts from the node's values. */}
					{target && (
						<NodeForm
							key={"node" in target ? target.node.id : `new-${target.kind}`}
							target={target}
							onClose={onClose}
							chooseFolder={chooseFolder}
							onSaved={onSaved}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function NodeForm({
	target,
	onClose,
	chooseFolder,
	onSaved,
}: {
	target: NodeDialogTarget;
	onClose: () => void;
	chooseFolder: boolean;
	onSaved?: (node: FavouriteNode) => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const editing = "node" in target ? target.node : null;
	const kind = editing
		? editing.kind
		: (target as { kind: "folder" | "link" }).kind;

	const [title, setTitle] = useState(editing?.title ?? "");
	const [url, setUrl] = useState(editing?.url ?? "");
	const [tags, setTags] = useState<string[]>(editing?.tags ?? []);
	const [description, setDescription] = useState(editing?.description ?? "");
	const [removePreview, setRemovePreview] = useState(false);
	const [problem, setProblem] = useState<FieldError | "generic" | null>(null);
	const [parentId, setParentId] = useState(
		"parent" in target ? target.parent.id : "",
	);

	const save = useMutation({
		mutationFn: () => {
			if (editing) {
				return updateFavourite(
					editing.id,
					kind === "folder"
						? { title }
						: {
								title,
								url,
								tags,
								description,
								...(removePreview && { removePreview: true }),
							},
				);
			}
			return createFavourite(
				kind === "folder"
					? { parentId, kind, title }
					: { parentId, kind, title, url, tags },
			);
		},
		onSuccess: (node) => {
			onSaved?.(node);
			void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
			onClose();
		},
		onError: (e) => {
			if (e instanceof ApiError && e.code === "not_found") {
				void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			}
			setProblem(
				(e instanceof ApiError && fieldErrorFromCode(e.code, kind)) ||
					"generic",
			);
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const found =
			kind === "folder"
				? checkFolderName(title)
				: (checkUrl(url) ??
					checkLinkTitle(title) ??
					tags.map(checkTag).find((p) => p !== null) ??
					(editing ? checkDescription(description) : null));
		setProblem(found);
		if (!found) save.mutate();
	}

	const heading = editing
		? kind === "folder"
			? t("favourites.dialog.editFolder")
			: t("favourites.dialog.editLink")
		: kind === "folder"
			? t("favourites.dialog.newFolder")
			: t("favourites.dialog.newLink");

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">{heading}</Dialog.Title>
			{!editing && !chooseFolder && "parentLabel" in target && (
				<Dialog.Description className="mt-1 text-sm text-muted">
					{t("favourites.dialog.inFolder", { name: target.parentLabel })}
				</Dialog.Description>
			)}
			<form className="mt-5 space-y-4" onSubmit={onSubmit} noValidate>
				{kind === "folder" ? (
					<div>
						<label
							htmlFor="favourite-name"
							className="block text-sm font-medium"
						>
							{t("favourites.dialog.nameLabel")}
						</label>
						<input
							id="favourite-name"
							autoComplete="off"
							maxLength={MAX_FOLDER_NAME}
							value={title}
							onChange={(e) => setTitle(e.target.value)}
							className={textInput}
						/>
					</div>
				) : (
					<LinkFields
						url={url}
						onUrlChange={setUrl}
						title={title}
						onTitleChange={setTitle}
						tags={tags}
						onTagsChange={setTags}
						folder={
							!editing && chooseFolder
								? { value: parentId, onChange: setParentId }
								: undefined
						}
						hint={!editing}
					/>
				)}
				{editing && kind === "link" && (
					<LinkDetails
						link={editing}
						description={description}
						onDescriptionChange={setDescription}
						removePreview={removePreview}
						onRemovePreview={() => setRemovePreview(true)}
					/>
				)}
				{problem && (
					<p role="alert" className="text-sm text-danger">
						{problem === "generic"
							? t("common.genericError")
							: t(`favourites.errors.${problem}`)}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Dialog.Close className={secondaryButton}>
						{t("favourites.dialog.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={save.isPending}
					>
						{save.isPending
							? t("favourites.dialog.saving")
							: t("favourites.dialog.save")}
					</button>
				</div>
			</form>
		</>
	);
}

/**
 * The parts of an existing link that new links get from the page fetch: its
 * description, its preview (which can be removed on save) and its icon, which
 * can be refreshed right away.
 */
function LinkDetails({
	link,
	description,
	onDescriptionChange,
	removePreview,
	onRemovePreview,
}: {
	link: FavouriteNode;
	description: string;
	onDescriptionChange: (description: string) => void;
	removePreview: boolean;
	onRemovePreview: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	// The tree has the icon after a refresh; the target is a snapshot.
	const tree = useQuery(favouritesQueryOptions);
	const current = tree.data ? findNode(tree.data, link.id) : undefined;
	const iconImageId = current ? current.iconImageId : link.iconImageId;
	const preview = removePreview ? undefined : link.previewImageId;

	const refresh = useMutation({
		mutationFn: () => refreshFavicon(link.url ?? ""),
		onSuccess: () =>
			queryClient.invalidateQueries({ queryKey: favouritesQueryKey }),
	});

	return (
		<>
			<div>
				<label
					htmlFor="favourite-description"
					className="block text-sm font-medium"
				>
					{t("favourites.dialog.descriptionLabel")}
				</label>
				<textarea
					id="favourite-description"
					rows={3}
					value={description}
					onChange={(e) => onDescriptionChange(e.target.value)}
					className={`${textInput} resize-y`}
				/>
			</div>
			<div>
				<p className="text-sm font-medium">
					{t("favourites.dialog.previewLabel")}
				</p>
				{preview ? (
					<div className="mt-2 flex items-end gap-3">
						<img
							src={imageUrl(preview)}
							alt={t("favourites.dialog.previewAlt")}
							className="max-h-32 min-w-0 rounded-lg border border-line object-contain"
						/>
						<button
							type="button"
							className={`${secondaryButton} shrink-0`}
							onClick={onRemovePreview}
						>
							<Trash2 className="size-4" aria-hidden />
							{t("favourites.dialog.removePreview")}
						</button>
					</div>
				) : (
					<p className="mt-1 text-sm text-muted">
						{t("favourites.dialog.noPreview")}
					</p>
				)}
			</div>
			<div>
				<p className="text-sm font-medium">
					{t("favourites.dialog.iconLabel")}
				</p>
				<div className="mt-2 flex items-center gap-3">
					<ImageIcon
						imageId={iconImageId}
						version={refresh.isSuccess ? refresh.submittedAt : undefined}
						className="size-6"
					/>
					<button
						type="button"
						className={secondaryButton}
						disabled={refresh.isPending}
						onClick={() => refresh.mutate()}
					>
						<RefreshCw
							className={`size-4 ${refresh.isPending ? "animate-spin motion-reduce:animate-none" : ""}`}
							aria-hidden
						/>
						{t("favourites.dialog.refreshIcon")}
					</button>
				</div>
				{refresh.isError && (
					<p role="alert" className="mt-2 text-sm text-danger">
						{t(refreshErrorKey(refresh.error), { name: label(link) })}
					</p>
				)}
			</div>
		</>
	);
}

/** The translation key of the message for a failed icon refresh. */
function refreshErrorKey(error: Error) {
	switch (error instanceof ApiError ? error.code : "") {
		case "favicon_not_found":
			return "favourites.errors.iconNotFound";
		case "blocked_address":
			return "favourites.errors.iconBlocked";
		case "rate_limited":
			return "favourites.errors.iconRateLimited";
		default:
			return "favourites.errors.iconFailed";
	}
}
