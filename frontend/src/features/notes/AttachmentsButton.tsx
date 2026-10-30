import { Menu } from "@base-ui/react/menu";
import { useQuery } from "@tanstack/react-query";
import { Paperclip, Plus, X } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { secondaryButton } from "#/components/buttonStyles";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { AttachmentIcon } from "#/features/attachments/AttachmentView";
import {
	attachmentKind,
	attachmentsQueryOptions,
	formatSize,
	type NoteAttachment,
} from "#/features/attachments/api";

/**
 * The paperclip button of a note's editor, with the number of its
 * attachments. Its dropdown lists them in content order; an image opens with
 * onOpenImage and another file in a new tab. "Remove from note" calls
 * onRemove, and "Add attachment" lets the user choose files for onAdd.
 */
export function AttachmentsButton({
	attachments,
	onAdd,
	onRemove,
	onOpenImage,
	disabled = false,
}: {
	attachments: NoteAttachment[];
	onAdd: (files: File[]) => void;
	onRemove: (id: string) => void;
	onOpenImage: (id: string) => void;
	disabled?: boolean;
}) {
	const { t, i18n } = useTranslation();
	const fileInput = useRef<HTMLInputElement>(null);
	const [open, setOpen] = useState(false);
	// The allowed types are only needed once the dropdown was opened.
	const [used, setUsed] = useState(false);
	const limits = useQuery({ ...attachmentsQueryOptions, enabled: used });

	return (
		<>
			<Menu.Root
				open={open}
				onOpenChange={(next) => {
					setOpen(next);
					if (next) setUsed(true);
				}}
			>
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
						<Menu.Popup
							className={`${menuPopup} w-80 max-w-[calc(100vw-2rem)]`}
						>
							{attachments.length === 0 && (
								<p className="px-3 py-2 text-muted">
									{t("attachments.button.none")}
								</p>
							)}
							{attachments.map((a) => (
								<div key={a.id} className="flex items-center gap-1">
									<Menu.Item
										className={`${menuItem} min-w-0 flex-1`}
										aria-label={t("attachments.button.open", { name: a.name })}
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
									<Menu.Item
										className={`${menuItem} w-auto shrink-0 px-2`}
										disabled={disabled}
										aria-label={t("attachments.button.remove", {
											name: a.name,
										})}
										title={t("attachments.button.removeShort")}
										onClick={() => onRemove(a.id)}
									>
										<X className="size-4" aria-hidden />
									</Menu.Item>
								</div>
							))}
							<Menu.Separator className="my-1 border-t border-line" />
							<Menu.Item
								className={menuItem}
								disabled={disabled}
								onClick={() => fileInput.current?.click()}
							>
								<Plus className="size-4" aria-hidden />
								{t("attachments.button.add")}
							</Menu.Item>
						</Menu.Popup>
					</Menu.Positioner>
				</Menu.Portal>
			</Menu.Root>
			<input
				ref={fileInput}
				type="file"
				multiple
				hidden
				accept={limits.data?.allowedTypes.join(",")}
				data-testid="attachment-input"
				onChange={(e) => {
					const files = [...(e.target.files ?? [])];
					e.target.value = "";
					if (files.length) onAdd(files);
				}}
			/>
		</>
	);
}
