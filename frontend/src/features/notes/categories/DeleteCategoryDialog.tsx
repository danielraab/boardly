import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { dangerButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import {
	deleteCategory,
	type Note,
	type NoteCategory,
	noteCategoriesQueryKey,
	notesQueryKey,
} from "../api";

/**
 * Confirms deleting a category, saying how many of its notes become
 * uncategorized and that its subcategories move up. onDeleted runs after the
 * category is gone.
 */
export function DeleteCategoryDialog({
	category,
	categories,
	notes,
	onClose,
	onDeleted,
}: {
	category: NoteCategory | null;
	categories: NoteCategory[];
	notes: Note[];
	onClose: () => void;
	onDeleted: (category: NoteCategory) => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const remove = useMutation({
		mutationFn: (c: NoteCategory) => deleteCategory(c.id),
		onSuccess: (_, c) => {
			void queryClient.invalidateQueries({ queryKey: noteCategoriesQueryKey });
			void queryClient.invalidateQueries({ queryKey: notesQueryKey });
			onDeleted(c);
			onClose();
		},
	});
	const noteCount = category
		? notes.filter((n) => n.categoryId === category.id).length
		: 0;
	const subcategories = category
		? categories.filter((c) => c.parentId === category.id).length
		: 0;

	return (
		<AlertDialog.Root
			open={category !== null}
			onOpenChange={(open) => {
				if (!open) {
					remove.reset();
					onClose();
				}
			}}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					{category && (
						<>
							<AlertDialog.Title className="text-lg font-semibold">
								{t("notes.categories.deleteDialog.title")}
							</AlertDialog.Title>
							<AlertDialog.Description className="mt-1 space-y-1 text-sm text-muted">
								<span className="block">
									{t("notes.categories.deleteDialog.text", {
										title: category.title,
									})}
								</span>
								{noteCount > 0 && (
									<span className="block">
										{t("notes.categories.deleteDialog.notes", {
											count: noteCount,
										})}
									</span>
								)}
								{subcategories > 0 && (
									<span className="block">
										{t("notes.categories.deleteDialog.subcategories", {
											count: subcategories,
										})}
									</span>
								)}
							</AlertDialog.Description>
							{remove.isError && (
								<p role="alert" className="mt-3 text-sm text-danger">
									{t("common.genericError")}
								</p>
							)}
							<div className="mt-6 flex justify-end gap-3">
								<AlertDialog.Close className={secondaryButton}>
									{t("notes.categories.deleteDialog.cancel")}
								</AlertDialog.Close>
								<button
									type="button"
									className={dangerButton}
									disabled={remove.isPending}
									onClick={() => remove.mutate(category)}
								>
									{t("notes.categories.deleteDialog.confirm")}
								</button>
							</div>
						</>
					)}
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
