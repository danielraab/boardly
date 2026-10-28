import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Pencil } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { FavouritePicker } from "#/features/favourites/FavouritePicker";
import { ApiError } from "#/lib/api";
import {
	type AppToken,
	type AppTokenPatch,
	appNames,
	appTokensQueryKey,
	updateAppToken,
} from "./appTokens/api";
import { RootCategoryField } from "./CreateAppTokenDialog";

/**
 * "Edit": changes the root folder and, for Karakeep, the root category of an
 * active token. The app keeps its API key.
 */
export function EditAppTokenDialog({ token }: { token: AppToken }) {
	const { t } = useTranslation();
	const [open, setOpen] = useState(false);
	return (
		<Dialog.Root open={open} onOpenChange={setOpen}>
			<Dialog.Trigger
				className={iconButton}
				aria-label={t("settings.apps.edit", { name: token.name })}
			>
				<Pencil className="size-4" aria-hidden />
			</Dialog.Trigger>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					<EditForm token={token} onSaved={() => setOpen(false)} />
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function EditForm({
	token,
	onSaved,
}: {
	token: AppToken;
	onSaved: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [rootId, setRootId] = useState(token.rootId ?? "");
	const [rootCategoryId, setRootCategoryId] = useState(token.rootCategoryId);

	// Only what changed is sent.
	const patch: AppTokenPatch = {};
	if (rootId !== "" && rootId !== token.rootId) patch.rootId = rootId;
	if (token.app === "karakeep" && rootCategoryId !== token.rootCategoryId) {
		patch.rootCategoryId = rootCategoryId;
	}
	const unchanged = Object.keys(patch).length === 0;

	const save = useMutation({
		mutationFn: () => updateAppToken(token.id, patch),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: appTokensQueryKey });
			onSaved();
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate();
	}

	const code = save.error instanceof ApiError ? save.error.code : null;
	const error = !save.error
		? null
		: code === "invalid_root"
			? t("settings.apps.form.rootInvalid")
			: code === "invalid_root_category"
				? t("settings.apps.form.rootCategoryInvalid")
				: code === "token_inactive"
					? t("settings.apps.editForm.inactive")
					: t("common.genericError");

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{t("settings.apps.editForm.title")}
			</Dialog.Title>
			<Dialog.Description className="mt-1 text-sm text-muted">
				{t("settings.apps.editForm.text")}
			</Dialog.Description>
			<p className="mt-4 text-sm">
				<span className="font-medium">{token.name}</span> ·{" "}
				{appNames[token.app]}
			</p>
			<form className="mt-5 space-y-5" onSubmit={onSubmit}>
				<div>
					<FavouritePicker
						id="app-token-edit-root"
						label={t("settings.apps.form.root")}
						mode="folder"
						value={rootId}
						onChange={setRootId}
					/>
					<p className="mt-1 text-sm text-muted">
						{t("settings.apps.form.rootHint")}
					</p>
				</div>
				{token.app === "karakeep" && (
					<RootCategoryField
						id="app-token-edit-root-category"
						value={rootCategoryId}
						onChange={setRootCategoryId}
					/>
				)}
				{error && (
					<p role="alert" className="text-sm text-danger">
						{error}
					</p>
				)}
				<div className="flex justify-end gap-3">
					<Dialog.Close className={secondaryButton}>
						{t("settings.apps.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={save.isPending || unchanged}
					>
						{save.isPending
							? t("settings.apps.editForm.saving")
							: t("settings.apps.editForm.save")}
					</button>
				</div>
			</form>
		</>
	);
}
