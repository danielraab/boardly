import { useState } from "react";
import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import type { SectionHeight } from "../api";

export const maxRows = 50;

/** The width and height of a widget while its dialog is open. */
export function useSizeState(
	initial: { width: number; height: SectionHeight },
	desktopColumns: number,
) {
	const [width, setWidth] = useState(initial.width);
	const [fixed, setFixed] = useState(initial.height !== "auto");
	const [rows, setRows] = useState(
		initial.height === "auto" ? "4" : String(initial.height),
	);
	const rowCount = Number(rows);
	const rowsValid =
		Number.isInteger(rowCount) && rowCount >= 1 && rowCount <= maxRows;
	return {
		width,
		setWidth,
		fixed,
		setFixed,
		rows,
		setRows,
		desktopColumns,
		valid: !fixed || rowsValid,
		rowsValid,
		value: (): { width: number; height: SectionHeight } => ({
			width,
			height: fixed ? rowCount : "auto",
		}),
	};
}

export type SizeState = ReturnType<typeof useSizeState>;

/**
 * The width select and the height choice, shared by every widget dialog.
 * Without height, only the width is offered.
 */
export function SizeFields({
	state,
	idPrefix,
	height = true,
}: {
	state: SizeState;
	idPrefix: string;
	height?: boolean;
}) {
	const { t } = useTranslation();
	// A width above the desktop count (after lowering it) stays selectable.
	const widths = Array.from(
		{ length: Math.max(state.desktopColumns, state.width) },
		(_, i) => i + 1,
	);
	return (
		<>
			<div>
				<label
					htmlFor={`${idPrefix}-width`}
					className="block text-sm font-medium"
				>
					{t("dashboards.sectionDialog.widthLabel")}
				</label>
				<select
					id={`${idPrefix}-width`}
					value={state.width}
					onChange={(e) => state.setWidth(Number(e.target.value))}
					className={textInput}
				>
					{widths.map((n) => (
						<option key={n} value={n}>
							{t("dashboards.sectionDialog.widthOption", { count: n })}
						</option>
					))}
				</select>
			</div>
			{height && (
				<fieldset>
					<legend className="text-sm font-medium">
						{t("dashboards.sectionDialog.heightLabel")}
					</legend>
					<div className="mt-2 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm">
						<label className="inline-flex items-center gap-2">
							<input
								type="radio"
								name={`${idPrefix}-height`}
								checked={!state.fixed}
								onChange={() => state.setFixed(false)}
								className="size-4 accent-ink"
							/>
							{t("dashboards.sectionDialog.heightAuto")}
						</label>
						<label className="inline-flex items-center gap-2">
							<input
								type="radio"
								name={`${idPrefix}-height`}
								checked={state.fixed}
								onChange={() => state.setFixed(true)}
								className="size-4 accent-ink"
							/>
							{t("dashboards.sectionDialog.heightRows")}
						</label>
						{state.fixed && (
							<label className="inline-flex items-center gap-2">
								<span>{t("dashboards.sectionDialog.rowsLabel")}</span>
								<input
									type="number"
									min={1}
									max={maxRows}
									value={state.rows}
									onChange={(e) => state.setRows(e.target.value)}
									aria-invalid={!state.rowsValid}
									className="w-20 rounded-lg border border-line bg-paper/40 px-3 py-1.5 text-ink outline-none focus-visible:border-ink"
								/>
							</label>
						)}
					</div>
					{state.fixed && !state.rowsValid && (
						<p role="alert" className="mt-1 text-sm text-red-700">
							{t("dashboards.sectionDialog.rowsInvalid")}
						</p>
					)}
				</fieldset>
			)}
		</>
	);
}
