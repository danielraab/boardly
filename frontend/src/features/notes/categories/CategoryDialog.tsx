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
import { type IconChoice, IconChooser } from "#/components/IconChooser";
import {
	createCategory,
	type NoteCategory,
	noteCategoriesQueryKey,
	updateCategory,
} from "../api";
import { ColourChoices } from "../ColourPicker";
import type { NoteColour } from "../colours";
import { iconChoice, noteIcon } from "../noteIcon";
import { buildTree, flatten, subtreeIds } from "./tree";

/** What the dialog does: create a category below parentId, or edit one. */
export type CategoryDialogTarget =
	| { parentId: string | null }
	| { category: NoteCategory };

export const maxCategoryTitle = 200;

export function CategoryDialog({
	target,
	categories,
	onClose,
}: {
	target: CategoryDialogTarget | null;
	categories: NoteCategory[];
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={target !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100vh-2rem)] overflow-auto`}
				>
					{target && (
						<CategoryForm
							key={"category" in target ? target.category.id : "new"}
							target={target}
							categories={categories}
							onClose={onClose}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function CategoryForm({
	target,
	categories,
	onClose,
}: {
	target: CategoryDialogTarget;
	categories: NoteCategory[];
	onClose: () => void;
}) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const editing = "category" in target ? target.category : null;
	const [title, setTitle] = useState(editing?.title ?? "");
	const [icon, setIcon] = useState<IconChoice>(
		iconChoice(editing?.icon ?? null),
	);
	const [colour, setColour] = useState<NoteColour | null>(
		editing?.color ?? null,
	);
	const [parentId, setParentId] = useState<string | null>(
		editing
			? editing.parentId
			: (target as { parentId: string | null }).parentId,
	);
	const [touched, setTouched] = useState(false);

	// A category can't move into itself or one of its subcategories.
	const excluded = editing
		? subtreeIds(categories, editing.id)
		: new Set<string>();
	const parents = flatten(buildTree(categories, i18n.language)).filter(
		(n) => !excluded.has(n.category.id),
	);

	const trimmed = title.trim();
	const titleError = !trimmed
		? t("notes.categories.dialog.titleRequired")
		: [...trimmed].length > maxCategoryTitle
			? t("notes.categories.dialog.titleTooLong")
			: null;

	const save = useMutation({
		mutationFn: () => {
			const fields = {
				title: trimmed,
				icon: noteIcon(icon),
				color: colour,
				parentId,
			};
			return editing
				? updateCategory(editing.id, fields)
				: createCategory(fields);
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: noteCategoriesQueryKey });
			onClose();
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		setTouched(true);
		if (!titleError) save.mutate();
	}

	return (
		<form onSubmit={onSubmit} noValidate className="space-y-4">
			<Dialog.Title className="text-lg font-semibold">
				{editing
					? t("notes.categories.dialog.editTitle")
					: t("notes.categories.dialog.newTitle")}
			</Dialog.Title>
			<label className="block text-sm font-medium">
				{t("notes.categories.dialog.title")}
				<input
					value={title}
					onChange={(e) => setTitle(e.target.value)}
					onBlur={() => setTouched(true)}
					aria-invalid={touched && titleError !== null}
					aria-describedby={
						touched && titleError ? "category-title-error" : undefined
					}
					className={textInput}
				/>
			</label>
			{touched && titleError && (
				<p
					id="category-title-error"
					role="alert"
					className="-mt-2 text-sm text-danger"
				>
					{titleError}
				</p>
			)}
			<IconChooser value={icon} onChange={setIcon} />
			<fieldset>
				<legend className="text-sm font-medium">
					{t("notes.colours.label")}
				</legend>
				<div className="mt-2">
					<ColourChoices
						value={colour}
						onChange={setColour}
						name="category-colour"
					/>
				</div>
			</fieldset>
			<label className="block text-sm font-medium">
				{t("notes.categories.dialog.parent")}
				<select
					value={parentId ?? ""}
					onChange={(e) => setParentId(e.target.value || null)}
					className={textInput}
				>
					<option value="">{t("notes.categories.dialog.noParent")}</option>
					{parents.map((n) => (
						<option key={n.category.id} value={n.category.id}>
							{`${" ".repeat(n.depth)}${n.category.title}`}
						</option>
					))}
				</select>
			</label>
			{save.isError && (
				<p role="alert" className="text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			<div className="flex justify-end gap-3 pt-2">
				<Dialog.Close className={secondaryButton}>
					{t("notes.categories.dialog.cancel")}
				</Dialog.Close>
				<button
					type="submit"
					className={primaryButton}
					disabled={save.isPending}
				>
					{t("notes.categories.dialog.save")}
				</button>
			</div>
		</form>
	);
}
