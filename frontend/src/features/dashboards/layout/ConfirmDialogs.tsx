import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	dialogBackdrop,
	dialogPopup,
} from "#/components/dialogStyles";
import { secondaryButton } from "#/features/auth/LoginPage";
import type { Section } from "../api";
import { countDescendants } from "./layoutTree";

/** Asks before deleting a section and everything inside it. */
export function DeleteSectionDialog({
	section,
	onConfirm,
	onClose,
}: {
	section: Section | null;
	onConfirm: (section: Section) => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const title = section?.title ?? t("dashboards.editor.untitled");
	const count = section ? countDescendants(section) : 0;
	return (
		<Confirm
			open={section !== null}
			title={t("dashboards.deleteSection.title")}
			text={
				count > 0
					? t("dashboards.deleteSection.textWithChildren", { title, count })
					: t("dashboards.deleteSection.text", { title })
			}
			confirm={t("dashboards.deleteSection.confirm")}
			cancel={t("dashboards.deleteSection.cancel")}
			onConfirm={() => section && onConfirm(section)}
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
						<AlertDialog.Close className={`${secondaryButton} py-2.5`}>
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
