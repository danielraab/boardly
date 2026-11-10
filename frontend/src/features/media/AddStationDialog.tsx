import { Dialog } from "@base-ui/react/dialog";
import { Tabs } from "@base-ui/react/tabs";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
	ArrowUpRight,
	Check,
	LoaderCircle,
	Plus,
	Radio,
	Search,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import {
	createEmbed,
	createStation,
	type MediaSource,
	mediaQueryKey,
	type NewEmbed,
	type NewStation,
	normalizeStreamUrl,
} from "./api";
import {
	radioBrowserHome,
	type StationResult,
	searchStations,
} from "./radioBrowser";

const minQuery = 2;

const tab =
	"rounded-md px-3 py-1.5 text-sm text-muted outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-accent data-[selected]:bg-paper data-[selected]:font-medium data-[selected]:text-ink";

/** The message for a failed URL check, by error code. */
export function checkErrorKey(code: string) {
	switch (code) {
		case "invalid_url":
			return "media.add.errors.invalidUrl" as const;
		case "stream_not_https":
			return "media.add.errors.notHttps" as const;
		case "stream_unreachable":
			return "media.add.errors.unreachable" as const;
		case "embed_unsupported":
			return "media.add.errors.embedUnsupported" as const;
		case "embed_unreachable":
			return "media.add.errors.embedUnreachable" as const;
		default:
			return "common.genericError" as const;
	}
}

/** Adds media: stations from a search or by URL, and embeds by link. */
export function AddStationDialog({
	open,
	onClose,
	library,
}: {
	open: boolean;
	onClose: () => void;
	/** The library, to mark results that are already in it. */
	library: MediaSource[];
}) {
	const { t } = useTranslation();
	return (
		<Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup
					className={`${dialogPopup} flex max-h-[calc(100dvh-2rem)] w-[36rem] flex-col`}
				>
					<Dialog.Title className="text-lg font-semibold">
						{t("media.add.title")}
					</Dialog.Title>
					{open && (
						<Tabs.Root
							defaultValue="search"
							className="mt-4 flex min-h-0 flex-col"
						>
							<Tabs.List className="flex gap-1 border-b border-line pb-2">
								<Tabs.Tab value="search" className={tab}>
									{t("media.add.tabs.search")}
								</Tabs.Tab>
								<Tabs.Tab value="url" className={tab}>
									{t("media.add.tabs.url")}
								</Tabs.Tab>
								<Tabs.Tab value="embed" className={tab}>
									{t("media.add.tabs.embed")}
								</Tabs.Tab>
							</Tabs.List>
							<Tabs.Panel value="search" className="flex min-h-0 flex-col pt-4">
								<StationSearch library={library} onClose={onClose} />
							</Tabs.Panel>
							<Tabs.Panel value="url" className="pt-4">
								<UrlForm onClose={onClose} />
							</Tabs.Panel>
							<Tabs.Panel value="embed" className="pt-4">
								<EmbedForm onClose={onClose} />
							</Tabs.Panel>
						</Tabs.Root>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function useCreateSource<T>(
	create: (input: T) => Promise<MediaSource>,
	onDone: () => void,
) {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: create,
		onSuccess: (created) => {
			queryClient.setQueryData<MediaSource[]>(mediaQueryKey, (list) =>
				list ? [...list, created] : list,
			);
			void queryClient.invalidateQueries({ queryKey: mediaQueryKey });
			onDone();
		},
	});
}

function StationSearch({
	library,
	onClose,
}: {
	library: MediaSource[];
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const [input, setInput] = useState("");
	// The last submitted search; typing alone sends nothing.
	const [query, setQuery] = useState("");
	const search = useQuery({
		queryKey: ["radio-browser", query],
		queryFn: ({ signal }) => searchStations(query, signal),
		enabled: query.length >= minQuery,
		staleTime: 5 * 60_000,
	});
	const create = useCreateSource<NewStation>(createStation, onClose);
	const added = new Set(library.map((s) => normalizeStreamUrl(s.url)));
	const term = input.trim();

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (term.length < minQuery) return;
		// The same search again tries again, for example after an error.
		if (term === query) void search.refetch();
		else setQuery(term);
	}

	return (
		<>
			<form onSubmit={onSubmit} className="flex items-stretch gap-2">
				<label htmlFor="station-search" className="sr-only">
					{t("media.add.searchLabel")}
				</label>
				<input
					id="station-search"
					type="search"
					value={input}
					onChange={(e) => setInput(e.target.value)}
					placeholder={t("media.add.searchPlaceholder")}
					autoComplete="off"
					className={`${textInput.replace("mt-2 ", "")} min-w-0 flex-1`}
				/>
				<button
					type="submit"
					aria-label={t("media.add.searchButton")}
					className={`${secondaryButton} shrink-0 max-sm:px-3`}
					disabled={term.length < minQuery}
				>
					<Search className="size-4" aria-hidden />
					<span className="max-sm:hidden">{t("media.add.searchButton")}</span>
				</button>
			</form>
			<p className="mt-2 text-xs text-muted">
				{t("media.add.notice.before")}
				<a
					href={radioBrowserHome}
					target="_blank"
					rel="noopener noreferrer"
					aria-label={`radio-browser.info ${t("common.opensInNewTab")}`}
					className="inline-flex items-center gap-0.5 text-accent underline"
				>
					radio-browser.info
					<ArrowUpRight className="size-3" aria-hidden />
				</a>
				{t("media.add.notice.after")}
			</p>
			{create.isError && (
				<p role="alert" className="mt-2 text-sm text-danger">
					{t(
						checkErrorKey(
							create.error instanceof ApiError ? create.error.code : "",
						),
					)}
				</p>
			)}
			<div className="mt-3 min-h-0 flex-1 overflow-y-auto" aria-live="polite">
				{query.length >= minQuery && search.isFetching && !search.data && (
					<p className="flex items-center gap-2 py-6 text-sm text-muted">
						<LoaderCircle
							className="size-4 animate-spin motion-reduce:animate-none"
							aria-hidden
						/>
						{t("media.add.searching")}
					</p>
				)}
				{query.length >= minQuery && search.isError && (
					<p role="alert" className="py-6 text-sm text-danger">
						{t(
							search.error instanceof ApiError &&
								search.error.code === "rate_limited"
								? "media.add.rateLimited"
								: "media.add.directoryDown",
						)}
					</p>
				)}
				{query.length >= minQuery && search.data?.length === 0 && (
					<p className="py-6 text-sm text-muted">{t("media.add.noResults")}</p>
				)}
				{search.data && search.data.length > 0 && query.length >= minQuery && (
					<ul className="divide-y divide-line">
						{search.data.map((station) => (
							<ResultRow
								key={station.id}
								station={station}
								added={added.has(normalizeStreamUrl(station.url))}
								busy={create.isPending && create.variables?.url === station.url}
								disabled={create.isPending}
								onAdd={() =>
									create.mutate({
										title: station.name,
										url: station.url,
										logoUrl: station.logoUrl ?? undefined,
									})
								}
							/>
						))}
					</ul>
				)}
			</div>
		</>
	);
}

function ResultRow({
	station,
	added,
	busy,
	disabled,
	onAdd,
}: {
	station: StationResult;
	added: boolean;
	busy: boolean;
	disabled: boolean;
	onAdd: () => void;
}) {
	const { t } = useTranslation();
	const [logoFailed, setLogoFailed] = useState(false);
	const details = [
		station.country,
		station.tags.join(", "),
		station.bitrate ? t("media.add.bitrate", { kbps: station.bitrate }) : null,
	].filter(Boolean);

	return (
		<li
			className={`flex items-center gap-3 py-2 ${station.playable ? "" : "opacity-60"}`}
		>
			<span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-line bg-paper">
				{/* Through Boardly's server, so the logo's host doesn't see the user. */}
				{station.logo && !logoFailed ? (
					<img
						src={station.logo}
						alt=""
						loading="lazy"
						referrerPolicy="no-referrer"
						onError={() => setLogoFailed(true)}
						className="size-full object-contain"
					/>
				) : (
					<Radio className="size-5 text-muted" aria-hidden />
				)}
			</span>
			<div className="min-w-0 flex-1">
				<p className="truncate text-sm font-medium">{station.name}</p>
				<p className="truncate text-xs text-muted">
					{station.playable ? details.join(" · ") : t("media.add.notPlayable")}
				</p>
			</div>
			{added ? (
				<span className="inline-flex shrink-0 items-center gap-1 text-sm text-muted">
					<Check className="size-4" aria-hidden />
					{t("media.add.added")}
				</span>
			) : (
				station.playable && (
					<button
						type="button"
						onClick={onAdd}
						disabled={disabled}
						aria-label={t("media.add.addNamed", { name: station.name })}
						className={`${secondaryButton} shrink-0 px-3 py-1.5 max-sm:px-2`}
					>
						{busy ? (
							<LoaderCircle
								className="size-4 animate-spin motion-reduce:animate-none"
								aria-hidden
							/>
						) : (
							<Plus className="size-4" aria-hidden />
						)}
						<span className="max-sm:hidden">{t("media.add.add")}</span>
					</button>
				)
			)}
		</li>
	);
}

function UrlForm({ onClose }: { onClose: () => void }) {
	const { t } = useTranslation();
	const [title, setTitle] = useState("");
	const [url, setUrl] = useState("");
	const create = useCreateSource<NewStation>(createStation, onClose);
	const error =
		create.error instanceof ApiError
			? create.error.code
			: create.error
				? "error"
				: null;
	const titleError = error === "invalid_title";

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		create.mutate({ title, url: url.trim() });
	}

	return (
		<form onSubmit={onSubmit} className="flex flex-col gap-4">
			<div>
				<label htmlFor="station-name" className="block text-sm font-medium">
					{t("media.add.nameLabel")}
				</label>
				<input
					id="station-name"
					required
					value={title}
					maxLength={200}
					onChange={(e) => setTitle(e.target.value)}
					aria-invalid={titleError || undefined}
					className={textInput}
				/>
			</div>
			<div>
				<label htmlFor="station-url" className="block text-sm font-medium">
					{t("media.add.urlLabel")}
				</label>
				<input
					id="station-url"
					type="url"
					required
					value={url}
					onChange={(e) => setUrl(e.target.value)}
					placeholder="https://"
					aria-invalid={(error !== null && !titleError) || undefined}
					aria-describedby="station-url-hint"
					className={textInput}
				/>
				{error !== null ? (
					<p
						id="station-url-hint"
						role="alert"
						className="mt-1 text-sm text-danger"
					>
						{titleError
							? t("media.add.errors.invalidTitle")
							: t(checkErrorKey(error))}
					</p>
				) : (
					<p id="station-url-hint" className="mt-1 text-sm text-muted">
						{t("media.add.urlHint")}
					</p>
				)}
			</div>
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onClose} className={secondaryButton}>
					{t("media.cancel")}
				</button>
				<button
					type="submit"
					disabled={create.isPending}
					className={primaryButton}
				>
					{create.isPending && (
						<LoaderCircle
							className="size-4 animate-spin motion-reduce:animate-none"
							aria-hidden
						/>
					)}
					{create.isPending ? t("media.add.checking") : t("media.add.add")}
				</button>
			</div>
		</form>
	);
}

function EmbedForm({ onClose }: { onClose: () => void }) {
	const { t } = useTranslation();
	const [title, setTitle] = useState("");
	const [url, setUrl] = useState("");
	const create = useCreateSource<NewEmbed>(createEmbed, onClose);
	const error =
		create.error instanceof ApiError
			? create.error.code
			: create.error
				? "error"
				: null;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		create.mutate({ url: url.trim(), title: title.trim() || undefined });
	}

	return (
		<form onSubmit={onSubmit} className="flex flex-col gap-4">
			<div>
				<label htmlFor="embed-link" className="block text-sm font-medium">
					{t("media.add.embedLinkLabel")}
				</label>
				<input
					id="embed-link"
					required
					value={url}
					onChange={(e) => setUrl(e.target.value)}
					placeholder="https://open.spotify.com/"
					autoComplete="off"
					aria-invalid={error !== null || undefined}
					aria-describedby="embed-link-hint"
					className={textInput}
				/>
				{error !== null ? (
					<p
						id="embed-link-hint"
						role="alert"
						className="mt-1 text-sm text-danger"
					>
						{t(
							error === "invalid_title"
								? "media.add.errors.invalidTitle"
								: checkErrorKey(error),
						)}
					</p>
				) : (
					<p id="embed-link-hint" className="mt-1 text-sm text-muted">
						{t("media.add.embedHint")}
					</p>
				)}
			</div>
			<div>
				<label htmlFor="embed-name" className="block text-sm font-medium">
					{t("media.add.embedNameLabel")}
				</label>
				<input
					id="embed-name"
					value={title}
					maxLength={200}
					onChange={(e) => setTitle(e.target.value)}
					className={textInput}
				/>
			</div>
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onClose} className={secondaryButton}>
					{t("media.cancel")}
				</button>
				<button
					type="submit"
					disabled={create.isPending}
					className={primaryButton}
				>
					{create.isPending && (
						<LoaderCircle
							className="size-4 animate-spin motion-reduce:animate-none"
							aria-hidden
						/>
					)}
					{create.isPending ? t("media.add.checking") : t("media.add.add")}
				</button>
			</div>
		</form>
	);
}
