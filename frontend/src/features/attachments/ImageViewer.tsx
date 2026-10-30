import { Dialog } from "@base-ui/react/dialog";
import { ChevronLeft, ChevronRight, Download, X } from "lucide-react";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import type { NoteAttachment } from "./api";

const viewerButton =
	"inline-flex size-10 shrink-0 items-center justify-center rounded-full bg-black/50 text-white hover:bg-black/70 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none";

/**
 * Shows images[index] as large as the window allows, with its name, a
 * download button and, for several images, buttons and arrow keys to move
 * between them. Escape, the close button or a click outside the image closes
 * it, and the focus goes back to what opened it. index null is closed.
 */
export function ImageViewer({
	images,
	index,
	onIndexChange,
	onClose,
}: {
	images: NoteAttachment[];
	index: number | null;
	onIndexChange: (index: number) => void;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const image = index === null ? undefined : images[index];
	const several = images.length > 1;
	const move = (step: number) => {
		if (index !== null && several)
			onIndexChange((index + step + images.length) % images.length);
	};

	return (
		<Dialog.Root
			open={image !== undefined}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className="fixed inset-0 z-50 bg-black/85 transition-opacity duration-150 data-[ending-style]:opacity-0 data-[starting-style]:opacity-0 motion-reduce:transition-none" />
				<Dialog.Popup
					aria-label={image?.name}
					className="fixed inset-0 z-50 flex flex-col outline-none"
					onKeyDown={(e) => {
						if (e.key === "ArrowLeft") move(-1);
						else if (e.key === "ArrowRight") move(1);
					}}
				>
					{image && (
						<>
							<div className="flex items-center gap-2 p-3 text-white">
								<Dialog.Title className="min-w-0 flex-1 truncate text-sm font-medium">
									{image.name}
								</Dialog.Title>
								<a
									href={image.url}
									download={image.name}
									aria-label={t("attachments.download", { name: image.name })}
									className={viewerButton}
								>
									<Download className="size-5" aria-hidden />
								</a>
								<Dialog.Close
									aria-label={t("common.close")}
									className={viewerButton}
								>
									<X className="size-5" aria-hidden />
								</Dialog.Close>
							</div>
							{/* A click beside the image closes the viewer. */}
							{/* biome-ignore lint/a11y/useKeyWithClickEvents lint/a11y/noStaticElementInteractions: Escape and the close button do the same */}
							<div
								className="relative flex min-h-0 flex-1 items-center justify-center px-2 pb-4 sm:px-16"
								onClick={(e) => e.target === e.currentTarget && onClose()}
							>
								<img
									src={image.url}
									alt={image.name}
									className="max-h-full max-w-full object-contain"
								/>
								{several && (
									<>
										<button
											type="button"
											className={`${viewerButton} absolute left-2 sm:left-4`}
											aria-label={t("attachments.viewer.previous")}
											onClick={() => move(-1)}
										>
											<ChevronLeft className="size-6" aria-hidden />
										</button>
										<button
											type="button"
											className={`${viewerButton} absolute right-2 sm:right-4`}
											aria-label={t("attachments.viewer.next")}
											onClick={() => move(1)}
										>
											<ChevronRight className="size-6" aria-hidden />
										</button>
									</>
								)}
							</div>
						</>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

/**
 * The state of an image viewer: open(images, id) shows the image with the ID
 * among images, and viewer is the element to render.
 */
export function useImageViewer() {
	const [state, setState] = useState<{
		images: NoteAttachment[];
		index: number;
	} | null>(null);
	const open = useCallback((images: NoteAttachment[], id: string) => {
		const index = images.findIndex((a) => a.id === id);
		if (index >= 0) setState({ images, index });
	}, []);
	const viewer = (
		<ImageViewer
			images={state?.images ?? []}
			index={state?.index ?? null}
			onIndexChange={(index) => setState((s) => s && { ...s, index })}
			onClose={() => setState(null)}
		/>
	);
	return { open, viewer };
}
