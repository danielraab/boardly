import { queryOptions } from "@tanstack/react-query";
import type { NoteDoc } from "#/features/notes/document/types";
import { api } from "#/lib/api";
import type { LegalPageName } from "#/lib/legalLinks";

/** A legal page's content; language is null for the fallback file. */
export type LegalContent = {
	page: LegalPageName;
	language: string | null;
	doc: NoteDoc;
};

export const legalPageQueryOptions = (page: LegalPageName, language: string) =>
	queryOptions({
		queryKey: ["legal", page, language],
		queryFn: () =>
			api<LegalContent>(
				`/api/legal/${page}?lang=${encodeURIComponent(language)}`,
			),
		staleTime: Number.POSITIVE_INFINITY,
	});
