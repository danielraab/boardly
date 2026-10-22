import { Dialog } from "@base-ui/react/dialog";
import type { FormEvent, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import type { ContentWidget, SectionHeight } from "../api";
import { SizeFields, useSizeState } from "../layout/SizeFields";

/**
 * The frame of a widget kind's settings dialog: its title, the kind's own
 * fields, the shared width and height, and the delete, cancel and apply
 * buttons. `valid` is whether the kind's fields can be applied. A fitContent
 * widget has no height choice and is always as tall as its content.
 */
export function WidgetSettingsForm({
	title,
	widget,
	isNew,
	desktopColumns,
	valid,
	fitContent = false,
	onApply,
	onDelete,
	children,
}: {
	title: string;
	widget: ContentWidget;
	isNew: boolean;
	desktopColumns: number;
	valid: boolean;
	fitContent?: boolean;
	onApply: (size: { width: number; height: SectionHeight }) => void;
	onDelete: () => void;
	children: ReactNode;
}) {
	const { t } = useTranslation();
	const size = useSizeState(widget, desktopColumns);

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (!valid) return;
		if (fitContent) onApply({ width: size.width, height: "auto" });
		else if (size.valid) onApply(size.value());
	}

	return (
		<>
			<Dialog.Title className="text-lg font-semibold">{title}</Dialog.Title>
			<form className="mt-5 space-y-4" onSubmit={onSubmit} noValidate>
				{children}
				<SizeFields state={size} idPrefix="widget" height={!fitContent} />
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
					<button
						type="submit"
						className={primaryButton}
						disabled={!valid || (!fitContent && !size.valid)}
					>
						{isNew
							? t("dashboards.widgetDialog.add")
							: t("dashboards.widgetDialog.apply")}
					</button>
				</div>
			</form>
		</>
	);
}
