import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { NotFound } from "#/components/NotFound";
import { narrowPage } from "#/components/pageWidths";
import { NoteRenderer } from "#/features/notes/NoteRenderer";
import { type LegalPageName, legalLink } from "#/lib/legalLinks";
import { usePageTitle } from "#/lib/usePageTitle";
import { legalPageQueryOptions } from "./api";

/**
 * The imprint or the privacy policy from the operator's Markdown files, in the
 * visitor's language. Pages that aren't configured, or are external URLs,
 * show the not-found page.
 */
export function LegalPage({ page }: { page: LegalPageName }) {
	if (legalLink(page)?.kind !== "page") return <NotFound />;
	return <LegalContentPage page={page} />;
}

function LegalContentPage({ page }: { page: LegalPageName }) {
	const { t, i18n } = useTranslation();
	usePageTitle(t(`legal.${page}`));
	const content = useQuery(
		legalPageQueryOptions(page, i18n.resolvedLanguage ?? i18n.language),
	);

	return (
		<article className={`${narrowPage} py-8`}>
			{content.isPending ? (
				<p className="text-muted">{t("common.loading")}</p>
			) : content.isError ? (
				<div role="alert">
					<p>{t("legal.loadError")}</p>
					<button
						type="button"
						onClick={() => content.refetch()}
						className="mt-3 font-medium underline underline-offset-4"
					>
						{t("legal.retry")}
					</button>
				</div>
			) : (
				<NoteRenderer doc={content.data.doc} openLinks />
			)}
		</article>
	);
}
