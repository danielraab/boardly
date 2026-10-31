import { Dialog } from "@base-ui/react/dialog";
import { Tabs } from "@base-ui/react/tabs";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, Check, LoaderCircle, Plus, Radio } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import { useDebounced } from "#/lib/useDebounced";
import {
	createStation,
	type MediaSource,
	mediaQueryKey,
	type NewStation,
	normalizeStreamUrl,
} from "./api";
import {
	radioBrowserHome,
	type StationResult,
	searchStations,
} from "./radioBrowser";

/** Typing pauses this long before a search is sent. */
const searchDelay = 300;
const minQuery = 2;

const tab =
	"rounded-md px-3 py-1.5 text-sm text-muted outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-accent data-[selected]:bg-paper data-[selected]:font-medium data-[selected]:text-ink";

/** The message for a failed URL check, by error code. */
function checkErrorKey(code: string) {
	switch (code) {
		case "invalid_url":
			return "media.add.errors.invalidUrl" as const;
		case "stream_not_https":
			return "media.add.errors.notHttps" as const;
		case "stream_unreachable":
			return "media.add.errors.unreachable" as const;
		default:
			return "common.genericError" as const;
	}
}

/** Adds stations: from a radio-browser.info search, or by URL. */
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
							</Tabs.List>
							<Tabs.Panel value="search" className="flex min-h-0 flex-col pt-4">
								<StationSearch library={library} onClose={onClose} />
							</Tabs.Panel>
							<Tabs.Panel value="url" className="pt-4">
								<UrlForm onClose={onClose} />
							</Tabs.Panel>
						</Tabs.Root>
					)}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function useCreateStation(onDone: () => void) {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (station: NewStation) => createStation(station),
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
	const query = useDebounced(input.trim(), searchDelay);
	const search = useQuery({
		queryKey: ["radio-browser", query],
		queryFn: ({ signal }) => searchStations(query, signal),
		enabled: query.length >= minQuery,
		staleTime: 5 * 60_000,
	});
	const create = useCreateStation(onClose);
	const added = new Set(library.map((s) => normalizeStreamUrl(s.url)));

	return (
		<>
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
				className={`${textInput} mt-0`}
			/>
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
						{t("media.add.directoryDown")}
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
				{/* The browser only loads https logos on an https page. */}
				{station.logoUrl?.startsWith("https://") && !logoFailed ? (
					<img
						src={station.logoUrl}
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
						className={`${secondaryButton} shrink-0 px-3 py-1.5`}
					>
						{busy ? (
							<LoaderCircle
								className="size-4 animate-spin motion-reduce:animate-none"
								aria-hidden
							/>
						) : (
							<Plus className="size-4" aria-hidden />
						)}
						{t("media.add.add")}
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
	const create = useCreateStation(onClose);
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
