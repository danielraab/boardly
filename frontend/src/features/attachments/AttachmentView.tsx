import {
	Download,
	File,
	FileArchive,
	FileAudio,
	FileImage,
	FileSpreadsheet,
	FileText,
	FileVideo,
	FileX,
	LoaderCircle,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { attachmentKind, formatSize, type NoteAttachment } from "./api";

const archiveTypes = [
	"application/zip",
	"application/x-7z-compressed",
	"application/x-tar",
	"application/gzip",
];

/** An icon for the kind of file of a content type. */
export function AttachmentIcon({
	contentType,
	className,
}: {
	contentType: string;
	className?: string;
}) {
	const Icon =
		attachmentKind(contentType) === "image"
			? FileImage
			: attachmentKind(contentType) === "audio"
				? FileAudio
				: attachmentKind(contentType) === "video"
					? FileVideo
					: archiveTypes.includes(contentType)
						? FileArchive
						: contentType.includes("sheet") || contentType === "text/csv"
							? FileSpreadsheet
							: contentType.startsWith("text/") ||
									contentType === "application/pdf" ||
									contentType.includes("document")
								? FileText
								: File;
	return <Icon className={className} aria-hidden />;
}

const fileCard =
	"flex items-center gap-3 rounded-lg border border-line bg-paper/40 px-3 py-2";

/**
 * An attachment block of a note: an image, an audio or video player, or a
 * file card with a download link, and "Attachment missing" for one that
 * doesn't exist. With interactive off, as on the cards of /notes, files and
 * players don't react to activation. onOpenImage opens an image in the image
 * viewer, also with interactive off; without it, the image is inert.
 */
export function AttachmentView({
	attachment,
	interactive = true,
	onOpenImage,
}: {
	attachment: NoteAttachment | undefined;
	interactive?: boolean;
	onOpenImage?: () => void;
}) {
	const { t, i18n } = useTranslation();
	if (!attachment) {
		return (
			<div className={`${fileCard} text-muted`}>
				<FileX className="size-5 shrink-0" aria-hidden />
				<span>{t("attachments.missing")}</span>
			</div>
		);
	}
	const kind = attachmentKind(attachment.contentType);
	if (kind === "image") {
		const image = (
			<img
				src={attachment.url}
				alt={attachment.name}
				loading="lazy"
				className="h-auto max-w-full rounded-md"
			/>
		);
		if (!onOpenImage) return image;
		return (
			<button
				type="button"
				onClick={onOpenImage}
				aria-label={t("attachments.viewImage", { name: attachment.name })}
				className="pointer-events-auto block max-w-full cursor-zoom-in rounded-md focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			>
				{image}
			</button>
		);
	}
	if (kind === "audio") {
		return (
			// biome-ignore lint/a11y/useMediaCaption: user files have no captions
			<audio
				controls
				preload="metadata"
				src={attachment.url}
				aria-label={attachment.name}
				className={`w-full ${interactive ? "" : "pointer-events-none"}`}
				tabIndex={interactive ? undefined : -1}
			/>
		);
	}
	if (kind === "video") {
		return (
			// biome-ignore lint/a11y/useMediaCaption: user files have no captions
			<video
				controls
				preload="metadata"
				src={attachment.url}
				aria-label={attachment.name}
				className={`max-w-full rounded-md ${interactive ? "" : "pointer-events-none"}`}
				tabIndex={interactive ? undefined : -1}
			/>
		);
	}
	const size = formatSize(attachment.size, i18n.language);
	return (
		<div className={fileCard}>
			<AttachmentIcon
				contentType={attachment.contentType}
				className="size-5 shrink-0 text-muted"
			/>
			<span className="min-w-0 flex-1 truncate">
				{attachment.name}
				<span className="text-muted"> · {size}</span>
			</span>
			{interactive && (
				<a
					href={attachment.url}
					download={attachment.name}
					aria-label={t("attachments.download", { name: attachment.name })}
					title={t("attachments.downloadShort")}
					className="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted hover:bg-ink/5 hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
				>
					<Download className="size-4" aria-hidden />
				</a>
			)}
		</div>
	);
}

/** Where a file that is still uploading will appear. */
export function UploadingView({ name }: { name: string }) {
	const { t } = useTranslation();
	return (
		<div className={`${fileCard} text-muted`} role="status">
			<LoaderCircle
				className="size-5 shrink-0 animate-spin motion-reduce:animate-none"
				aria-hidden
			/>
			<span className="min-w-0 flex-1 truncate">
				{t("attachments.uploading", { name })}
			</span>
		</div>
	);
}
