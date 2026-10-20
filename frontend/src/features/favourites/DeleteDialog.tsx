import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	dialogBackdrop,
	dialogPopup,
} from "#/components/dialogStyles";
import { secondaryButton } from "#/features/auth/LoginPage";
import {
	type DeleteMode,
	deleteFavourite,
	type FavouriteNode,
	favouritesQueryKey,
} from "./api";
import { label } from "./tree";

export type DeleteTarget = { node: FavouriteNode; parentLabel: string };

export function DeleteDialog({
	target,
	onClose,
}: {
	target: DeleteTarget | null;
	onClose: () => void;
}) {
	return (
		<AlertDialog.Root
			open={target !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					{target && (
						<DeleteForm
							key={target.node.id}
							target={target}
							onClose={onClose}
						/>
					)}
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}

function DeleteForm({
	target,
	onClose,
}: {
	target: DeleteTarget;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const { node, parentLabel } = target;
	const name = label(node);
	const count = node.children?.length ?? 0;
	// A non-empty folder has no default: the user has to choose what happens to its contents.
	const [mode, setMode] = useState<DeleteMode | null>(null);

	const remove = useMutation({
		mutationFn: () =>
			deleteFavourite(node.id, count > 0 ? (mode ?? undefined) : undefined),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
			onClose();
		},
	});

	const text =
		node.kind === "link"
			? t("favourites.deleteDialog.linkText", { name })
			: count === 0
				? t("favourites.deleteDialog.emptyFolderText", { name })
				: t("favourites.deleteDialog.folderText", { name, count });

	return (
		<>
			<AlertDialog.Title className="text-lg font-semibold">
				{node.kind === "link"
					? t("favourites.deleteDialog.linkTitle")
					: t("favourites.deleteDialog.folderTitle")}
			</AlertDialog.Title>
			<AlertDialog.Description className="mt-1 text-sm text-muted">
				{text}
			</AlertDialog.Description>
			{count > 0 && (
				<fieldset className="mt-4 space-y-2">
					<legend className="sr-only">{text}</legend>
					{(["delete", "move"] as const).map((option) => (
						<label
							key={option}
							className="flex cursor-pointer items-start gap-3 rounded-lg border border-line px-3 py-2.5 text-sm has-[:checked]:border-ink"
						>
							<input
								type="radio"
								name="delete-mode"
								value={option}
								checked={mode === option}
								onChange={() => setMode(option)}
								className="mt-0.5 accent-ink"
							/>
							{option === "delete"
								? t("favourites.deleteDialog.deleteContents")
								: t("favourites.deleteDialog.moveContents", {
										parent: parentLabel,
									})}
						</label>
					))}
				</fieldset>
			)}
			{remove.isError && (
				<p role="alert" className="mt-3 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			<div className="mt-6 flex justify-end gap-3">
				<AlertDialog.Close className={`${secondaryButton} py-2.5`}>
					{t("favourites.deleteDialog.cancel")}
				</AlertDialog.Close>
				<button
					type="button"
					className={dangerButton}
					disabled={remove.isPending || (count > 0 && mode === null)}
					onClick={() => remove.mutate()}
				>
					{t("favourites.deleteDialog.confirm")}
				</button>
			</div>
		</>
	);
}
