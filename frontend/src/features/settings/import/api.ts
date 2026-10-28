import { api } from "#/lib/api";

export type ImportFormat = "karakeep" | "linkwarden" | "netscape";

export const importFormats: readonly ImportFormat[] = [
	"karakeep",
	"linkwarden",
	"netscape",
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
	format: ImportFormat,
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
