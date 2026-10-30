import { Menu } from "@base-ui/react/menu";
import { Paperclip } from "lucide-react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { AttachmentIcon } from "#/features/attachments/AttachmentView";
import {
	attachmentKind,
	formatSize,
	type NoteAttachment,
} from "#/features/attachments/api";

/**
 * The paperclip button of a note's editor, with the number of its
 * attachments. Its dropdown lists them in content order; an image opens with
 * onOpenImage and another file in a new tab.
 */
export function AttachmentsButton({
	attachments,
	onOpenImage,
}: {
	attachments: NoteAttachment[];
	onOpenImage: (id: string) => void;
}) {
	const { t, i18n } = useTranslation();

	return (
		<Menu.Root>
			<Menu.Trigger
				className={`${secondaryButton} h-[2.375rem] px-2.5`}
				aria-label={t("attachments.button.label")}
				title={t("attachments.button.label")}
			>
				<Paperclip className="size-4" aria-hidden />
				{attachments.length > 0 && (
					<span aria-hidden className="tabular-nums">
						{attachments.length}
					</span>
				)}
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6} className="z-50">
					<Menu.Popup className={`${menuPopup} w-80 max-w-[calc(100vw-2rem)]`}>
						{attachments.length === 0 && (
							<p className="px-3 py-2 text-muted">
								{t("attachments.button.none")}
							</p>
						)}
						{attachments.map((a) => (
							<Menu.Item
								key={a.id}
								className={menuItem}
								onClick={() => {
									if (attachmentKind(a.contentType) === "image")
										onOpenImage(a.id);
									else window.open(a.url, "_blank", "noopener");
								}}
							>
								{attachmentKind(a.contentType) === "image" ? (
									<img
										src={a.url}
										alt=""
										className="size-8 shrink-0 rounded object-cover"
									/>
								) : (
									<span className="flex size-8 shrink-0 items-center justify-center">
										<AttachmentIcon
											contentType={a.contentType}
											className="size-5 text-muted"
										/>
									</span>
								)}
								<span className="min-w-0 flex-1">
									<span className="block truncate">{a.name}</span>
									<span className="block text-xs text-muted">
										{formatSize(a.size, i18n.language)}
									</span>
								</span>
							</Menu.Item>
						))}
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}
