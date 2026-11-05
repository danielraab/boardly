import { Dialog } from "@base-ui/react/dialog";
import { Tabs } from "@base-ui/react/tabs";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ExternalLink, RefreshCw, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import {
	type FavouriteNode,
	type OfflineCopy,
	offlineCopyQueryKey,
	offlineCopyQueryOptions,
	offlinePartUrl,
	refreshOfflineCopy,
} from "./api";
import { label } from "./tree";

const tab =
	"rounded-md px-3 py-1.5 text-sm text-muted outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-accent data-[active]:bg-paper data-[active]:font-medium data-[active]:text-ink data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[disabled]:hover:text-muted";

type Part = "article" | "screenshot";

/**
 * The offline copy of the owner's link while link is set: its article and
 * screenshot, the time it was saved, and a refresh. A pending copy is
 * checked again every few seconds while the dialog is open.
 */
export function OfflineCopyDialog({
	link,
	onClose,
}: {
	link: FavouriteNode | null;
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={link !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} flex h-[calc(100dvh-2rem)] w-[64rem] flex-col`}
				>
					{link && <OfflineCopyView link={link} />}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function OfflineCopyView({ link }: { link: FavouriteNode }) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const copy = useQuery(offlineCopyQueryOptions(link.id));
	const refresh = useMutation({
		mutationFn: () => refreshOfflineCopy(link.id),
		onSuccess: (data) =>
			queryClient.setQueryData(offlineCopyQueryKey(link.id), data),
	});
	const title = label(link);
	const url = link.url ?? "";
	const data = copy.data;
	const pending = data?.status === "pending";

	return (
		<>
			<div className="flex items-start gap-2">
				<Dialog.Title className="min-w-0 flex-1 text-lg font-semibold break-words">
					{title}
				</Dialog.Title>
				<Dialog.Close
					aria-label={t("favourites.offline.close")}
					className={`${iconButton} -mt-1 -mr-2`}
				>
					<X className="size-4" aria-hidden />
				</Dialog.Close>
			</div>
			<div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-2">
				<div className="min-w-0 flex-1 text-sm text-muted" aria-live="polite">
					{data && <Status copy={data} language={i18n.language} />}
					{refresh.error && (
						<p role="alert" className="text-danger">
							{refresh.error instanceof ApiError && refresh.error.status === 429
								? t("favourites.offline.rateLimited")
								: t("favourites.offline.refreshFailed")}
						</p>
					)}
				</div>
				<div className="flex gap-3">
					<button
						type="button"
						className={secondaryButton}
						disabled={!data || pending || refresh.isPending}
						onClick={() => refresh.mutate()}
					>
						<RefreshCw
							className={`size-4 ${pending ? "motion-safe:animate-spin" : ""}`}
							aria-hidden
						/>
						{t("favourites.offline.refresh")}
					</button>
					<a
						href={url}
						target="_blank"
						rel="noopener noreferrer"
						className={primaryButton}
					>
						<ExternalLink className="size-4" aria-hidden />
						{t("favourites.offline.openOriginal")}
					</a>
				</div>
			</div>
			{data && (data.hasArticle || data.hasScreenshot) && (
				<Parts linkId={link.id} title={title} copy={data} />
			)}
		</>
	);
}

/** The state line: when the copy was saved, or why there is none. */
function Status({ copy, language }: { copy: OfflineCopy; language: string }) {
	const { t } = useTranslation();
	const saved = copy.capturedAt && (
		<p>
			{t("favourites.offline.saved", {
				date: new Intl.DateTimeFormat(language, {
					dateStyle: "medium",
					timeStyle: "short",
				}).format(new Date(copy.capturedAt)),
			})}
		</p>
	);
	switch (copy.status) {
		case "pending":
			return (
				<>
					<p>{t("favourites.offline.pending")}</p>
					{saved}
				</>
			);
		case "failed":
			return (
				<>
					<p className="text-danger">
						{t("favourites.offline.failed")}
						{copy.error && ` ${t(`favourites.offline.reasons.${copy.error}`)}`}
					</p>
					{saved}
				</>
			);
		case "none":
			return <p>{t("favourites.offline.none")}</p>;
		default:
			return saved || null;
	}
}

/** The article and screenshot tabs; a part the copy lacks is disabled. */
function Parts({
	linkId,
	title,
	copy,
}: {
	linkId: string;
	title: string;
	copy: OfflineCopy;
}) {
	const { t } = useTranslation();
	const [chosen, setChosen] = useState<Part | null>(null);
	const available: Record<Part, boolean> = {
		article: copy.hasArticle,
		screenshot: copy.hasScreenshot,
	};
	const selected: Part =
		chosen && available[chosen]
			? chosen
			: copy.hasArticle
				? "article"
				: "screenshot";
	return (
		<Tabs.Root
			value={selected}
			onValueChange={(value) => setChosen(value as Part)}
			className="mt-4 flex min-h-0 flex-1 flex-col"
		>
			<Tabs.List className="flex gap-1 border-b border-line pb-2">
				<Tabs.Tab value="article" disabled={!copy.hasArticle} className={tab}>
					{t("favourites.offline.article")}
				</Tabs.Tab>
				<Tabs.Tab
					value="screenshot"
					disabled={!copy.hasScreenshot}
					className={tab}
				>
					{t("favourites.offline.screenshot")}
				</Tabs.Tab>
			</Tabs.List>
			<Tabs.Panel value="article" className="min-h-0 flex-1 pt-4">
				{copy.hasArticle && (
					<iframe
						title={t("favourites.offline.articleFrame", { title })}
						src={offlinePartUrl(linkId, "article", copy.capturedAt)}
						// No scripts and no access to Boardly: the article may only
						// open its links in a new tab.
						sandbox="allow-popups allow-popups-to-escape-sandbox"
						referrerPolicy="no-referrer"
						className="size-full rounded-lg border border-line bg-white"
					/>
				)}
			</Tabs.Panel>
			<Tabs.Panel
				value="screenshot"
				className="min-h-0 flex-1 overflow-auto pt-4"
			>
				{copy.hasScreenshot && (
					<img
						src={offlinePartUrl(linkId, "screenshot", copy.capturedAt)}
						alt={t("favourites.offline.screenshotAlt", { title })}
						className="w-full rounded-lg border border-line"
					/>
				)}
			</Tabs.Panel>
		</Tabs.Root>
	);
}
