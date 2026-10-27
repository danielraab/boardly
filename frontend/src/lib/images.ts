import { api } from "#/lib/api";

/** Uploads an image file and returns its ID. */
export async function uploadImage(file: Blob) {
	const { id } = await api<{ id: string }>("/api/images", {
		method: "POST",
		file,
	});
	return id;
}
