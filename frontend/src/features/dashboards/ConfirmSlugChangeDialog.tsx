import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useTranslation } from "react-i18next";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { primaryButton, secondaryButton } from "#/features/auth/LoginPage";

/** Asks before a slug change that breaks links to the old address. */
export function ConfirmSlugChangeDialog({
	change,
	onConfirm,
	onCancel,
}: {
	/** The old and new address, or null when closed. */
	change: { from: string; to: string } | null;
	onConfirm: () => void;
	onCancel: () => void;
}) {
	const { t } = useTranslation();
	return (
		<AlertDialog.Root
			open={change !== null}
			onOpenChange={(open) => !open && onCancel()}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{t("dashboards.confirmSlug.title")}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-1 text-sm text-muted">
						{t("dashboards.confirmSlug.text")}
					</AlertDialog.Description>
					{change && (
						<dl className="mt-4 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
							<dt className="text-muted">{t("dashboards.confirmSlug.from")}</dt>
							<dd className="break-all line-through">{change.from}</dd>
							<dt className="text-muted">{t("dashboards.confirmSlug.to")}</dt>
							<dd className="break-all font-medium">{change.to}</dd>
						</dl>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={`${secondaryButton} py-2.5`}>
							{t("dashboards.confirmSlug.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={`${primaryButton} py-2.5`}
							onClick={onConfirm}
						>
							{t("dashboards.confirmSlug.confirm")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
