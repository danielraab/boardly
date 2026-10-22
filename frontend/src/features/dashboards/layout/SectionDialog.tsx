import { Dialog } from "@base-ui/react/dialog";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import type { Section } from "../api";
import type { SectionSettings } from "./layoutTree";
import { SizeFields, useSizeState } from "./SizeFields";

export const maxSectionTitle = 100;

/** Add a section (section null) or edit one. */
export type SectionDialogTarget = { section: Section | null };

export function SectionDialog({
	target,
	desktopColumns,
	onApply,
	onDelete,
	onClose,
}: {
	target: SectionDialogTarget | null;
	desktopColumns: number;
	onApply: (settings: SectionSettings) => void;
	onDelete: (section: Section) => void;
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
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					{target && (
						<SectionForm
							key={target.section?.id ?? "new"}
							editing={target.section}
							desktopColumns={desktopColumns}
							onApply={onApply}
							onDelete={onDelete}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function SectionForm({
	editing,
	desktopColumns,
	onApply,
	onDelete,
}: {
	editing: Section | null;
	desktopColumns: number;
	onApply: (settings: SectionSettings) => void;
	onDelete: (section: Section) => void;
}) {
	const { t } = useTranslation();
	const [title, setTitle] = useState(editing?.title ?? "");
	const [border, setBorder] = useState(editing?.border ?? true);
	const size = useSizeState(
		editing ?? { width: desktopColumns, height: "auto" },
		desktopColumns,
	);
	const [titleTooLong, setTitleTooLong] = useState(false);

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const trimmed = title.trim();
		const tooLong = [...trimmed].length > maxSectionTitle;
		setTitleTooLong(tooLong);
		if (tooLong || !size.valid) return;
		onApply({
			title: trimmed === "" ? null : trimmed,
			border,
			...size.value(),
		});
	}

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{editing
					? t("dashboards.sectionDialog.editTitle")
					: t("dashboards.sectionDialog.newTitle")}
			</Dialog.Title>
			<form className="mt-5 space-y-4" onSubmit={onSubmit} noValidate>
				<div>
					<label htmlFor="section-title" className="block text-sm font-medium">
						{t("dashboards.sectionDialog.titleLabel")}
					</label>
					<input
						id="section-title"
						autoComplete="off"
						maxLength={maxSectionTitle}
						value={title}
						onChange={(e) => setTitle(e.target.value)}
						aria-describedby="section-title-hint"
						className={textInput}
					/>
					<p id="section-title-hint" className="mt-1 text-sm text-muted">
						{t("dashboards.sectionDialog.titleHint")}
					</p>
				</div>
				<div className="flex items-start gap-3">
					<input
						id="section-border"
						type="checkbox"
						checked={border}
						onChange={(e) => setBorder(e.target.checked)}
						aria-describedby="section-border-hint"
						className="mt-1 size-4 accent-ink"
					/>
					<div>
						<label htmlFor="section-border" className="text-sm font-medium">
							{t("dashboards.sectionDialog.borderLabel")}
						</label>
						<p id="section-border-hint" className="text-sm text-muted">
							{t("dashboards.sectionDialog.borderHint")}
						</p>
					</div>
				</div>
				<SizeFields state={size} idPrefix="section" />
				{titleTooLong && (
					<p role="alert" className="text-sm text-red-700">
						{t("dashboards.sectionDialog.titleTooLong")}
					</p>
				)}
				<div className="flex flex-wrap items-center justify-end gap-3 pt-2">
					{editing && (
						<button
							type="button"
							onClick={() => onDelete(editing)}
							className="mr-auto text-sm font-medium text-red-700 hover:underline"
						>
							{t("dashboards.sectionDialog.delete")}
						</button>
					)}
					<Dialog.Close className={secondaryButton}>
						{t("dashboards.sectionDialog.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={!size.valid}
					>
						{t("dashboards.sectionDialog.apply")}
					</button>
				</div>
			</form>
		</>
	);
}
