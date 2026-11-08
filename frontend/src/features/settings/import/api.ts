import { api } from "#/lib/api";

export type ImportFormat = "karakeep" | "linkwarden" | "netscape" | "boardly";

/** The formats of other apps, imported as bookmarks (and Karakeep notes). */
export type BookmarkFormat = Exclude<ImportFormat, "boardly">;

export const importFormats: readonly ImportFormat[] = [
	"karakeep",
	"linkwarden",
	"netscape",
	"boardly",
];

/** What an import added and left out, as the server reports it. */
export type ImportReport = {
	links: number;
	folders: number;
	notes: number;
	categories: number;
	skipped: {
		duplicates: number;
		duplicateNotes: number;
		invalidUrls: number;
		notLinks: number;
		smartLists: number;
		tooLargeNotes: number;
	};
	shortenedDescriptions: number;
};

/** The error codes of an import besides the generic ones. */
export type ImportErrorCode =
	| "invalid_import_file"
	| "file_too_large"
	| "invalid_folder"
	| "invalid_category";

/** The formats whose files can hold notes. */
export const formatsWithNotes: readonly ImportFormat[] = ["karakeep"];

/**
 * Imports the file into the folder, and its notes into the category (null for
 * the top level). The file is the raw body, like an image upload, because the
 * server's cross-site check doesn't allow form posts. Without skipDuplicates,
 * links and notes that are already saved are imported again.
 */
export function importBookmarks(
	format: BookmarkFormat,
	folderId: string,
	categoryId: string | null,
	file: Blob,
	skipDuplicates: boolean,
) {
	const query = new URLSearchParams({
		format,
		folderId,
		skipDuplicates: String(skipDuplicates),
	});
	if (categoryId) query.set("categoryId", categoryId);
	return api<ImportReport>(`/api/favourites/import?${query}`, {
		method: "POST",
		file,
	});
}

/** The parts of a Boardly export that can be imported. */
export type ImportPart =
	| "favourites"
	| "dashboards"
	| "notes"
	| "media"
	| "feeds";

export const importParts: readonly ImportPart[] = [
	"favourites",
	"dashboards",
	"notes",
	"media",
	"feeds",
];

/** What the import of a Boardly export added and left out, per part. */
export type DataImportReport = {
	favourites: {
		links: number;
		folders: number;
		tags: number;
		skipped: { duplicates: number; invalidUrls: number };
		shortenedDescriptions: number;
	} | null;
	dashboards: { dashboards: number; skippedWidgets: number } | null;
	notes: { notes: number; categories: number; duplicateNotes: number } | null;
	media: { sources: number; duplicates: number; invalid: number } | null;
	feeds?: {
		subscriptions: number;
		duplicates: number;
		invalid: number;
		restoredItems: number;
		skippedItems: number;
	} | null;
	files: {
		images: number;
		attachments: number;
		offlineCopies?: number;
		skipped: number;
	};
};

/**
 * Imports the chosen parts of a Boardly export (its JSON file or its ZIP
 * file) into the account: favourites into the folder (required with
 * "favourites"), notes into the category (null for the top level).
 */
export function importData(
	parts: readonly ImportPart[],
	folderId: string | null,
	categoryId: string | null,
	file: Blob,
	skipDuplicates: boolean,
) {
	const query = new URLSearchParams({
		parts: parts.join(","),
		skipDuplicates: String(skipDuplicates),
	});
	if (folderId) query.set("folderId", folderId);
	if (categoryId) query.set("categoryId", categoryId);
	return api<DataImportReport>(`/api/import?${query}`, {
		method: "POST",
		file,
	});
}
