import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { primaryButton, secondaryButton } from "#/features/auth/LoginPage";
import { ApiError } from "#/lib/api";
import {
	createFavourite,
	type FavouriteNode,
	favouritesQueryKey,
	updateFavourite,
} from "./api";
import { TagInput } from "./TagInput";
import {
	checkFolderName,
	checkLinkTitle,
	checkTag,
	checkUrl,
	type FieldError,
	fieldErrorFromCode,
	MAX_FOLDER_NAME,
	MAX_LINK_TITLE,
	MAX_URL,
} from "./validate";

/** What the dialog does: create a node in a folder, or edit a node. */
export type NodeDialogTarget =
	| { kind: "folder" | "link"; parent: FavouriteNode; parentLabel: string }
	| { node: FavouriteNode };

export function NodeDialog({
	target,
	onClose,
}: {
	target: NodeDialogTarget | null;
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
					{/* Remount per target so the form starts from the node's values. */}
					{target && (
						<NodeForm
							key={"node" in target ? target.node.id : `new-${target.kind}`}
							target={target}
							onClose={onClose}
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
}: {
	target: NodeDialogTarget;
	onClose: () => void;
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

	const save = useMutation({
		mutationFn: () => {
			if (editing) {
				return updateFavourite(
					editing.id,
					kind === "folder" ? { title } : { title, url, tags },
				);
			}
			const parentId = (target as { parent: FavouriteNode }).parent.id;
			return createFavourite(
				kind === "folder"
					? { parentId, kind, title }
					: { parentId, kind, title, url, tags },
			);
		},
		onSuccess: () => {
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
			{!editing && "parentLabel" in target && (
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
					<>
						<div>
							<label
								htmlFor="favourite-url"
								className="block text-sm font-medium"
							>
								{t("favourites.dialog.urlLabel")}
							</label>
							<input
								id="favourite-url"
								type="url"
								inputMode="url"
								autoComplete="off"
								maxLength={MAX_URL}
								placeholder={t("favourites.dialog.urlPlaceholder")}
								value={url}
								onChange={(e) => setUrl(e.target.value)}
								className={textInput}
							/>
						</div>
						<div>
							<label
								htmlFor="favourite-title"
								className="block text-sm font-medium"
							>
								{t("favourites.dialog.titleLabel")}
							</label>
							<input
								id="favourite-title"
								autoComplete="off"
								maxLength={MAX_LINK_TITLE}
								value={title}
								onChange={(e) => setTitle(e.target.value)}
								className={textInput}
							/>
						</div>
						<TagInput value={tags} onChange={setTags} />
					</>
				)}
				{problem && (
					<p role="alert" className="text-sm text-red-700">
						{problem === "generic"
							? t("common.genericError")
							: t(`favourites.errors.${problem}`)}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Dialog.Close className={`${secondaryButton} py-2.5`}>
						{t("favourites.dialog.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={`${primaryButton} py-2.5`}
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
