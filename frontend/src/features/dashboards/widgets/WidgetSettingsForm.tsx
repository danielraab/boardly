import { Dialog } from "@base-ui/react/dialog";
import { type FormEvent, type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import type { ContentWidget } from "../api";
import { WidthField } from "../layout/SizeFields";

/** The most characters of a widget's own title (as the server allows). */
export const maxWidgetTitle = 200;

/**
 * The frame of a widget kind's settings dialog: its title, the kind's own
 * fields, the shared width, and the delete, cancel and apply buttons. `valid`
 * is whether the kind's fields can be applied.
 */
export function WidgetSettingsForm({
	title,
	widget,
	isNew,
	desktopColumns,
	valid,
	onApply,
	onDelete,
	children,
}: {
	title: string;
	widget: ContentWidget;
	isNew: boolean;
	desktopColumns: number;
	valid: boolean;
	onApply: (size: { width: number }) => void;
	onDelete: () => void;
	children: ReactNode;
}) {
	const { t } = useTranslation();
	const [width, setWidth] = useState(widget.width);

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (valid) onApply({ width });
	}

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
			<form className="mt-5 space-y-4" onSubmit={onSubmit} noValidate>
				{children}
				<WidthField
					width={width}
					onChange={setWidth}
					desktopColumns={desktopColumns}
					idPrefix="widget"
					offerFull
				/>
				<div className="flex flex-wrap items-center justify-end gap-3 pt-2">
					{!isNew && (
						<button
							type="button"
							onClick={onDelete}
							className="mr-auto text-sm font-medium text-red-700 hover:underline"
						>
							{t("dashboards.widgetDialog.delete")}
						</button>
					)}
					<Dialog.Close className={secondaryButton}>
						{t("dashboards.widgetDialog.cancel")}
					</Dialog.Close>
					<button type="submit" className={primaryButton} disabled={!valid}>
						{isNew
							? t("dashboards.widgetDialog.add")
							: t("dashboards.widgetDialog.apply")}
					</button>
				</div>
			</form>
		</>
	);
}
