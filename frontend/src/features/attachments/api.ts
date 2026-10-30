import { queryOptions } from "@tanstack/react-query";
import { api } from "#/lib/api";

/** An attachment as a note returns it (see "Notes API"). */
export type NoteAttachment = {
	id: string;
	name: string;
	contentType: string;
	size: number;
	url: string;
};

/** An attachment as the attachments API returns it. */
export type Attachment = NoteAttachment & {
	createdAt: string;
	/** The deletion mark, or null. */
	deletedAt: string | null;
	/** When it will be deleted, or null when it isn't marked or is kept. */
	deleteAt: string | null;
	keep: boolean;
	noteIds: string[];
};

export type AttachmentList = {
	attachments: Attachment[];
	/** The largest attachment, in bytes. */
	maxSize: number;
	/** The content types that can be attached. */
	allowedTypes: string[];
};

export const attachmentsQueryKey = ["attachments"] as const;

const attachmentPath = (id: string) =>
	`/api/attachments/${encodeURIComponent(id)}`;

export const attachmentsQueryOptions = queryOptions({
	queryKey: attachmentsQueryKey,
	queryFn: () => api<AttachmentList>("/api/attachments"),
});

/** Uploads a file as an attachment named after it. */
export function uploadAttachment(file: File) {
	return api<Attachment>(
		`/api/attachments?name=${encodeURIComponent(file.name)}`,
		{ method: "POST", file },
	);
}

/** Sets or removes the keep flag of an attachment. */
export function setAttachmentKeep(id: string, keep: boolean) {
	return api<Attachment>(attachmentPath(id), {
		method: "PATCH",
		json: { keep },
	});
}

/**
 * Deletes an attachment. One a note still uses throws an ApiError with the
 * code attachment_in_use.
 */
export function deleteAttachment(id: string) {
	return api<void>(attachmentPath(id), { method: "DELETE" });
}

/** The kind of an attachment, for how it is shown. */
export type AttachmentKind = "image" | "audio" | "video" | "file";

export function attachmentKind(contentType: string): AttachmentKind {
	const group = contentType.split("/")[0];
	return group === "image" || group === "audio" || group === "video"
		? group
		: "file";
}

/** A size in bytes as "300 KB" or "2.5 MB". */
export function formatSize(bytes: number, locale?: string): string {
	const units = ["B", "KB", "MB", "GB"];
	let value = bytes;
	let unit = 0;
	while (value >= 1024 && unit < units.length - 1) {
		value /= 1024;
		unit++;
	}
	const digits = unit === 0 || value >= 10 ? 0 : 1;
	return `${new Intl.NumberFormat(locale, { maximumFractionDigits: digits }).format(value)} ${units[unit]}`;
}
