import { apiDownload } from "#/lib/api";

/**
 * Downloads all data of the signed-in user: the JSON document, or with
 * withFiles a ZIP archive that also holds the image and attachment files. A
 * rate-limited export throws an ApiError with status 429 and `retryAfter`.
 */
export async function exportData(withFiles: boolean) {
	const { blob, fileName } = await apiDownload(
		`/api/me/export?files=${withFiles}`,
	);
	return {
		blob,
		fileName: fileName ?? `boardly-export.${withFiles ? "zip" : "json"}`,
	};
}

/** Lets the browser save the blob as a file with the given name. */
export function saveFile(blob: Blob, fileName: string) {
	const url = URL.createObjectURL(blob);
	const link = document.createElement("a");
	link.href = url;
	link.download = fileName;
	link.style.display = "none";
	document.body.append(link);
	link.click();
	link.remove();
	// Revoked later, so the download has started before the URL goes away.
	setTimeout(() => URL.revokeObjectURL(url), 60_000);
}
