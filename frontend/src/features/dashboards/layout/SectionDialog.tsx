import { Dialog } from "@base-ui/react/dialog";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { primaryButton, secondaryButton } from "#/features/auth/LoginPage";
import type { Section } from "../api";
import type { SectionSettings } from "./layoutTree";

export const maxSectionTitle = 100;
export const maxRows = 50;

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
	const [width, setWidth] = useState(editing?.width ?? desktopColumns);
	const [border, setBorder] = useState(editing?.border ?? true);
	const [fixed, setFixed] = useState(
		editing !== null && editing.height !== "auto",
	);
	const [rows, setRows] = useState(
		editing && editing.height !== "auto" ? String(editing.height) : "4",
	);
	const [titleTooLong, setTitleTooLong] = useState(false);

	const rowCount = Number(rows);
	const rowsValid =
		Number.isInteger(rowCount) && rowCount >= 1 && rowCount <= maxRows;
	// A width above the desktop count (after lowering it) stays selectable.
	const widths = Array.from(
		{ length: Math.max(desktopColumns, width) },
		(_, i) => i + 1,
	);

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const trimmed = title.trim();
		const tooLong = [...trimmed].length > maxSectionTitle;
		setTitleTooLong(tooLong);
		if (tooLong || (fixed && !rowsValid)) return;
		onApply({
			title: trimmed === "" ? null : trimmed,
			width,
			border,
			height: fixed ? rowCount : "auto",
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
				<div>
					<label htmlFor="section-width" className="block text-sm font-medium">
						{t("dashboards.sectionDialog.widthLabel")}
					</label>
					<select
						id="section-width"
						value={width}
						onChange={(e) => setWidth(Number(e.target.value))}
						className={textInput}
					>
						{widths.map((n) => (
							<option key={n} value={n}>
								{t("dashboards.sectionDialog.widthOption", { count: n })}
							</option>
						))}
					</select>
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
				<fieldset>
					<legend className="text-sm font-medium">
						{t("dashboards.sectionDialog.heightLabel")}
					</legend>
					<div className="mt-2 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
						<label className="inline-flex items-center gap-2">
							<input
								type="radio"
								name="section-height"
								checked={!fixed}
								onChange={() => setFixed(false)}
								className="size-4 accent-ink"
							/>
							{t("dashboards.sectionDialog.heightAuto")}
						</label>
						<label className="inline-flex items-center gap-2">
							<input
								type="radio"
								name="section-height"
								checked={fixed}
								onChange={() => setFixed(true)}
								className="size-4 accent-ink"
							/>
							{t("dashboards.sectionDialog.heightRows")}
						</label>
						{fixed && (
							<label className="inline-flex items-center gap-2">
								<span>{t("dashboards.sectionDialog.rowsLabel")}</span>
								<input
									type="number"
									min={1}
									max={maxRows}
									value={rows}
									onChange={(e) => setRows(e.target.value)}
									aria-invalid={!rowsValid}
									className="w-20 rounded-lg border border-line bg-paper/40 px-3 py-1.5 text-ink outline-none focus-visible:border-ink"
								/>
							</label>
						)}
					</div>
					{fixed && !rowsValid && (
						<p role="alert" className="mt-1 text-sm text-red-700">
							{t("dashboards.sectionDialog.rowsInvalid")}
						</p>
					)}
				</fieldset>
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
					<Dialog.Close className={`${secondaryButton} py-2.5`}>
						{t("dashboards.sectionDialog.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={`${primaryButton} py-2.5`}
						disabled={fixed && !rowsValid}
					>
						{t("dashboards.sectionDialog.apply")}
					</button>
				</div>
			</form>
		</>
	);
}
