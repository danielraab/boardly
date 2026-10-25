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
	skipped: {
		duplicates: number;
		invalidUrls: number;
		notLinks: number;
		smartLists: number;
	};
	shortenedDescriptions: number;
};

/** The error codes of an import besides the generic ones. */
export type ImportErrorCode =
	| "invalid_import_file"
	| "file_too_large"
	| "invalid_folder";

/**
 * Imports the file into the folder. The file is the raw body, like an image
 * upload, because the server's cross-site check doesn't allow form posts.
 */
export function importBookmarks(
	format: ImportFormat,
	folderId: string,
	file: Blob,
) {
	const query = new URLSearchParams({ format, folderId });
	return api<ImportReport>(`/api/favourites/import?${query}`, {
		method: "POST",
		file,
	});
}
