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
	breakpoints,
	type Columns,
	createDashboard,
	type Dashboard,
	type DashboardFields,
	dashboardsQueryKey,
	dashboardViewQueryKey,
	defaultColumns,
	maxColumns,
	updateDashboard,
} from "./api";
import { ConfirmSlugChangeDialog } from "./ConfirmSlugChangeDialog";
import { SlugField, type SlugProblem } from "./SlugField";
import { slugError, slugFromTitle } from "./slug";

export const MAX_TITLE = 100;

/** What the dialog does: create a dashboard, or edit one. */
export type DashboardDialogTarget = { dashboard: Dashboard | null };

export function DashboardDialog({
	target,
	userSlug,
	onClose,
}: {
	target: DashboardDialogTarget | null;
	userSlug: string;
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
					{/* Remount per target so the form starts from the dashboard's values. */}
					{target && (
						<DashboardForm
							key={target.dashboard?.id ?? "new"}
							editing={target.dashboard}
							userSlug={userSlug}
							onClose={onClose}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

type TitleProblem =
	| "titleRequired"
	| "titleTooLong"
	| "invalidColumns"
	| "generic";

function DashboardForm({
	editing,
	userSlug,
	onClose,
}: {
	editing: Dashboard | null;
	userSlug: string;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [title, setTitle] = useState(editing?.title ?? "");
	const [slug, setSlug] = useState(editing?.slug ?? "");
	// While creating, the slug follows the title until the user edits it.
	const [slugEdited, setSlugEdited] = useState(editing !== null);
	const [isDefault, setIsDefault] = useState(editing?.isDefault ?? false);
	const [isPublic, setIsPublic] = useState(editing?.isPublic ?? false);
	const [columns, setColumns] = useState<Columns>(() => {
		const { desktopColumns, mobileColumns, mobileBreakpoint } =
			editing ?? defaultColumns;
		return { desktopColumns, mobileColumns, mobileBreakpoint };
	});
	const mobileTooMany = columns.mobileColumns > columns.desktopColumns;
	const [problem, setProblem] = useState<TitleProblem | null>(null);
	const [slugProblem, setSlugProblem] = useState<SlugProblem | null>(null);
	const [confirming, setConfirming] = useState(false);

	const save = useMutation({
		mutationFn: () => {
			const fields: DashboardFields = {
				title,
				slug,
				isDefault,
				isPublic,
				...columns,
			};
			return editing
				? updateDashboard(editing.id, fields)
				: createDashboard(fields);
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: dashboardsQueryKey });
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
			onClose();
		},
		onError: (e) => {
			setConfirming(false);
			const code = e instanceof ApiError ? e.code : "";
			if (code === "slug_taken") setSlugProblem("taken");
			else if (code === "invalid_slug") setSlugProblem("invalid");
			else if (code === "invalid_title") setProblem("titleTooLong");
			else if (code === "invalid_columns") setProblem("invalidColumns");
			else {
				if (code === "not_found") {
					void queryClient.invalidateQueries({ queryKey: dashboardsQueryKey });
				}
				setProblem("generic");
			}
		},
	});

	function onTitleChange(value: string) {
		setTitle(value);
		if (!slugEdited) {
			setSlug(slugFromTitle(value));
			setSlugProblem(null);
		}
	}

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const trimmed = title.trim();
		const found: TitleProblem | null =
			trimmed === ""
				? "titleRequired"
				: [...trimmed].length > MAX_TITLE
					? "titleTooLong"
					: null;
		setProblem(found);
		if (found || slugError(slug) || mobileTooMany) return;
		if (editing && slug !== editing.slug) setConfirming(true);
		else save.mutate();
	}

	const address = (s: string) => `/${userSlug}/${s}`;

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">
				{editing
					? t("dashboards.dialog.editTitle")
					: t("dashboards.dialog.newTitle")}
			</Dialog.Title>
			<form className="mt-5 space-y-4" onSubmit={onSubmit} noValidate>
				<div>
					<label
						htmlFor="dashboard-title"
						className="block text-sm font-medium"
					>
						{t("dashboards.dialog.titleLabel")}
					</label>
					<input
						id="dashboard-title"
						autoComplete="off"
						maxLength={MAX_TITLE}
						value={title}
						onChange={(e) => onTitleChange(e.target.value)}
						className={textInput}
					/>
				</div>
				<SlugField
					id="dashboard-slug"
					label={t("dashboards.dialog.slugLabel")}
					prefix={`/${userSlug}/`}
					value={slug}
					onChange={(value) => {
						setSlug(value);
						setSlugEdited(true);
						setSlugProblem(null);
					}}
					saved={editing?.slug}
					serverProblem={slugProblem}
				/>
				<Toggle
					id="dashboard-default"
					label={t("dashboards.dialog.defaultLabel")}
					hint={t("dashboards.dialog.defaultHint")}
					checked={isDefault}
					onChange={setIsDefault}
				/>
				<Toggle
					id="dashboard-public"
					label={t("dashboards.dialog.publicLabel")}
					hint={t("dashboards.dialog.publicHint")}
					checked={isPublic}
					onChange={setIsPublic}
				/>
				<ColumnFields
					columns={columns}
					onChange={(patch) => setColumns((c) => ({ ...c, ...patch }))}
					mobileTooMany={mobileTooMany}
				/>
				{problem && (
					<p role="alert" className="text-sm text-red-700">
						{problem === "generic"
							? t("common.genericError")
							: t(`dashboards.dialog.${problem}`)}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Dialog.Close className={secondaryButton}>
						{t("dashboards.dialog.cancel")}
					</Dialog.Close>
					<button
						type="submit"
						className={primaryButton}
						disabled={
							save.isPending || slugError(slug) !== null || mobileTooMany
						}
					>
						{save.isPending
							? t("dashboards.dialog.saving")
							: t("dashboards.dialog.save")}
					</button>
				</div>
			</form>
			<ConfirmSlugChangeDialog
				change={
					confirming && editing
						? { from: address(editing.slug), to: address(slug) }
						: null
				}
				onConfirm={() => save.mutate()}
				onCancel={() => setConfirming(false)}
			/>
		</>
	);
}

function Toggle({
	id,
	label,
	hint,
	checked,
	onChange,
}: {
	id: string;
	label: string;
	hint: string;
	checked: boolean;
	onChange: (checked: boolean) => void;
}) {
	return (
		<div className="flex items-start gap-3">
			<input
				id={id}
				type="checkbox"
				checked={checked}
				onChange={(e) => onChange(e.target.checked)}
				aria-describedby={`${id}-hint`}
				className="mt-1 size-4 accent-ink"
			/>
			<div>
				<label htmlFor={id} className="text-sm font-medium">
					{label}
				</label>
				<p id={`${id}-hint`} className="text-sm text-muted">
					{hint}
				</p>
			</div>
		</div>
	);
}

/** The grid settings: desktop and mobile column counts and the breakpoint. */
function ColumnFields({
	columns,
	onChange,
	mobileTooMany,
}: {
	columns: Columns;
	onChange: (patch: Partial<Columns>) => void;
	mobileTooMany: boolean;
}) {
	const { t } = useTranslation();
	const counts = Array.from({ length: maxColumns }, (_, i) => i + 1);
	return (
		<fieldset className="space-y-3">
			<legend className="text-sm font-medium">
				{t("dashboards.dialog.layoutLabel")}
			</legend>
			<div className="grid grid-cols-2 gap-3">
				<div>
					<label htmlFor="dashboard-desktop-columns" className="block text-sm">
						{t("dashboards.dialog.desktopColumnsLabel")}
					</label>
					<select
						id="dashboard-desktop-columns"
						value={columns.desktopColumns}
						onChange={(e) =>
							onChange({ desktopColumns: Number(e.target.value) })
						}
						className={textInput}
					>
						{counts.map((n) => (
							<option key={n} value={n}>
								{n}
							</option>
						))}
					</select>
				</div>
				<div>
					<label htmlFor="dashboard-mobile-columns" className="block text-sm">
						{t("dashboards.dialog.mobileColumnsLabel")}
					</label>
					<select
						id="dashboard-mobile-columns"
						value={columns.mobileColumns}
						onChange={(e) =>
							onChange({ mobileColumns: Number(e.target.value) })
						}
						aria-invalid={mobileTooMany}
						aria-describedby={
							mobileTooMany ? "dashboard-mobile-columns-hint" : undefined
						}
						className={textInput}
					>
						{counts.map((n) => (
							<option key={n} value={n}>
								{n}
							</option>
						))}
					</select>
				</div>
			</div>
			{mobileTooMany && (
				<p id="dashboard-mobile-columns-hint" className="text-sm text-red-700">
					{t("dashboards.dialog.mobileColumnsTooMany")}
				</p>
			)}
			<div>
				<label htmlFor="dashboard-breakpoint" className="block text-sm">
					{t("dashboards.dialog.breakpointLabel")}
				</label>
				<select
					id="dashboard-breakpoint"
					value={columns.mobileBreakpoint}
					onChange={(e) =>
						onChange({ mobileBreakpoint: Number(e.target.value) })
					}
					className={textInput}
				>
					{breakpoints.map((px) => (
						<option key={px} value={px}>
							{t("dashboards.dialog.breakpointOption", { px })}
						</option>
					))}
				</select>
			</div>
		</fieldset>
	);
}
