import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useTranslation } from "react-i18next";
import { dangerButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { isSection, type Widget } from "../api";
import { widgetLabel } from "../widgets/registry";
import { countDescendants } from "./layoutTree";

/** Asks before deleting a widget, and for a section everything inside it. */
export function DeleteWidgetDialog({
	widget,
	onConfirm,
	onClose,
}: {
	widget: Widget | null;
	onConfirm: (widget: Widget) => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const title = widget ? widgetLabel(widget, t) : "";
	const count = widget ? countDescendants(widget) : 0;
	const keys =
		widget === null || isSection(widget)
			? "dashboards.deleteSection"
			: "dashboards.deleteWidget";
	return (
		<Confirm
			open={widget !== null}
			title={t(`${keys}.title`)}
			text={
				count > 0
					? t("dashboards.deleteSection.textWithChildren", { title, count })
					: t(`${keys}.text`, { title })
			}
			confirm={t(`${keys}.confirm`)}
			cancel={t(`${keys}.cancel`)}
			onConfirm={() => widget && onConfirm(widget)}
			onClose={onClose}
		/>
	);
}

/** Asks before dropping unsaved changes. */
export function DiscardDraftDialog({
	open,
	onConfirm,
	onClose,
}: {
	open: boolean;
	onConfirm: () => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	return (
		<Confirm
			open={open}
			title={t("dashboards.discard.title")}
			text={t("dashboards.discard.text")}
			confirm={t("dashboards.discard.confirm")}
			cancel={t("dashboards.discard.cancel")}
			onConfirm={onConfirm}
			onClose={onClose}
		/>
	);
}

function Confirm({
	open,
	title,
	text,
	confirm,
	cancel,
	onConfirm,
	onClose,
}: {
	open: boolean;
	title: string;
	text: string;
	confirm: string;
	cancel: string;
	onConfirm: () => void;
	onClose: () => void;
}) {
	return (
		<AlertDialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{title}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-1 text-sm text-muted">
						{text}
					</AlertDialog.Description>
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{cancel}
						</AlertDialog.Close>
						<button type="button" className={dangerButton} onClick={onConfirm}>
							{confirm}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
