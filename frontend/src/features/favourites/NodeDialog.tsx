import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import {
	createFavourite,
	type FavouriteNode,
	favouritesQueryKey,
	updateFavourite,
} from "./api";
import { LinkFields } from "./LinkFields";
import {
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
	const [problem, setProblem] = useState<FieldError | "generic" | null>(null);
	const [parentId, setParentId] = useState(
		"parent" in target ? target.parent.id : "",
	);

	const save = useMutation({
		mutationFn: () => {
			if (editing) {
				return updateFavourite(
					editing.id,
					kind === "folder" ? { title } : { title, url, tags },
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
					null);
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
