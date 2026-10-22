import { Dialog } from "@base-ui/react/dialog";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import type { ContentWidget } from "../api";
import { findKind, type WidgetKind, widgetKinds } from "../widgets/registry";

/** Lists the widget kinds; picking one closes the picker. */
export function WidgetPicker({
	open,
	onPick,
	onClose,
}: {
	open: boolean;
	onPick: (kind: WidgetKind) => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	return (
		<Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					<Dialog.Title className="text-lg font-semibold">
						{t("dashboards.widgetPicker.title")}
					</Dialog.Title>
					<Dialog.Description className="mt-1 text-sm text-muted">
						{t("dashboards.widgetPicker.text")}
					</Dialog.Description>
					<ul className="mt-5 space-y-2">
						{widgetKinds.map((kind) => (
							<li key={kind.kind}>
								<button
									type="button"
									onClick={() => onPick(kind)}
									className="flex w-full items-start gap-3 rounded-lg border border-line px-4 py-3 text-left hover:border-ink"
								>
									<kind.icon className="mt-0.5 size-5 shrink-0" aria-hidden />
									<span className="min-w-0">
										<span className="block font-medium">
											{t(kind.labelKey)}
										</span>
										<span className="block text-sm text-muted">
											{t(kind.descriptionKey)}
										</span>
									</span>
								</button>
							</li>
						))}
					</ul>
					<div className="mt-6 flex justify-end">
						<Dialog.Close className={secondaryButton}>
							{t("dashboards.widgetDialog.cancel")}
						</Dialog.Close>
					</div>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/** Opens the settings of a content widget's kind. */
export function WidgetDialog({
	target,
	desktopColumns,
	onApply,
	onDelete,
	onClose,
}: {
	target: { widget: ContentWidget; isNew: boolean } | null;
	desktopColumns: number;
	onApply: (widget: ContentWidget) => void;
	onDelete: (widget: ContentWidget) => void;
	onClose: () => void;
}) {
	const kind = target ? findKind(target.widget.kind) : undefined;
	return (
		<Dialog.Root
			open={target !== null && kind !== undefined}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} max-h-[calc(100dvh-2rem)] overflow-y-auto`}
				>
					{target && kind && (
						<kind.Settings
							key={target.widget.id}
							widget={target.widget}
							isNew={target.isNew}
							desktopColumns={desktopColumns}
							onApply={onApply}
							onDelete={() => onDelete(target.widget)}
						/>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}
