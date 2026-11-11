import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Dialog } from "@base-ui/react/dialog";
import { Menu } from "@base-ui/react/menu";
import {
	closestCenter,
	DndContext,
	type DragEndEvent,
	KeyboardSensor,
	PointerSensor,
	TouchSensor,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import {
	arrayMove,
	SortableContext,
	sortableKeyboardCoordinates,
	useSortable,
	verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import {
	useInfiniteQuery,
	useMutation,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import {
	EllipsisVertical,
	GripVertical,
	Pencil,
	Plus,
	Rss,
	Search,
	Settings2,
	Trash2,
	X,
} from "lucide-react";
import { type FormEvent, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	primaryButton,
	secondaryButton,
} from "#/components/buttonStyles";
import {
	dialogBackdrop,
	dialogPopup,
	textInput,
} from "#/components/dialogStyles";
import { type IconChoice, IconChooser } from "#/components/IconChooser";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { widePage } from "#/components/pageWidths";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import { hostOf } from "#/features/media/api";
import { iconChoice, iconProps, noteIcon } from "#/features/notes/noteIcon";
import { ApiError } from "#/lib/api";
import { useDebounced } from "#/lib/useDebounced";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	createFeed,
	deleteFeed,
	type FeedSubscription,
	type FeedView,
	feedIcon,
	feedItemsQueryKey,
	feedItemsQueryOptions,
	feedsQueryKey,
	feedsQueryOptions,
	feedViews,
	reorderFeeds,
	updateFeed,
} from "./api";
import { FeedItemRow } from "./FeedItemRow";

/** /feeds: the items of the user's feeds, and managing the feeds. */
export function FeedsPage() {
	const { t } = useTranslation();
	usePageTitle(t("feeds.title"));
	const list = useQuery(feedsQueryOptions);
	const [view, setView] = useState<FeedView>("latest");
	const [adding, setAdding] = useState(false);
	const [managing, setManaging] = useState(false);
	const [editing, setEditing] = useState<FeedSubscription | null>(null);
	const [removing, setRemoving] = useState<FeedSubscription | null>(null);
	const [search, setSearch] = useState("");
	const [feedId, setFeedId] = useState("");
	const debounced = useDebounced(search.trim(), searchDelayMs);
	const feeds = list.data ?? [];
	const noFeeds = list.data !== undefined && feeds.length === 0;
	// Clearing applies at once; a removed feed falls back to all feeds.
	const query = search.trim() === "" ? "" : debounced;
	const filterId = feeds.some((f) => f.id === feedId) ? feedId : "";

	return (
		<section className={`${widePage} pt-12 pb-16`}>
			<div className="flex flex-wrap items-end justify-between gap-4">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("feeds.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">{t("feeds.intro")}</p>
				</div>
				<div className="flex flex-wrap gap-2">
					{feeds.length > 0 && (
						<button
							type="button"
							onClick={() => setManaging((m) => !m)}
							aria-expanded={managing}
							className={secondaryButton}
						>
							<Settings2 className="size-4" aria-hidden />
							{t("feeds.manage")}
						</button>
					)}
					<button
						type="button"
						onClick={() => setAdding(true)}
						className={primaryButton}
					>
						<Plus className="size-4" aria-hidden />
						{t("feeds.add")}
					</button>
				</div>
			</div>
			{list.isError && (
				<p role="alert" className="mt-6 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			{managing && feeds.length > 0 && (
				<FeedList feeds={feeds} onEdit={setEditing} onRemove={setRemoving} />
			)}
			{/* biome-ignore lint/a11y/useSemanticElements: a group of view toggles */}
			<div
				role="group"
				aria-label={t("feeds.viewsLabel")}
				className="mt-8 flex flex-wrap gap-1 border-b border-line"
			>
				{feedViews.map((v) => (
					<button
						key={v}
						type="button"
						onClick={() => setView(v)}
						aria-pressed={view === v}
						className="-mb-px border-b-2 border-transparent px-3 py-2 text-sm font-medium text-muted aria-pressed:border-ink aria-pressed:text-ink"
					>
						{t(`feeds.views.${v}`)}
					</button>
				))}
			</div>
			{feeds.length > 0 && (
				<SearchBar
					search={search}
					onSearch={setSearch}
					feedId={filterId}
					onFeed={setFeedId}
					feeds={feeds}
				/>
			)}
			<Items view={view} noFeeds={noFeeds} q={query} feedId={filterId} />
			<AddFeedDialog open={adding} onClose={() => setAdding(false)} />
			<EditFeedDialog feed={editing} onClose={() => setEditing(null)} />
			<RemoveFeedDialog feed={removing} onClose={() => setRemoving(null)} />
		</section>
	);
}

/** How long typing has to pause before the search runs. */
const searchDelayMs = 300;

function SearchBar({
	search,
	onSearch,
	feedId,
	onFeed,
	feeds,
}: {
	search: string;
	onSearch: (value: string) => void;
	feedId: string;
	onFeed: (id: string) => void;
	feeds: FeedSubscription[];
}) {
	const { t } = useTranslation();
	const input = useRef<HTMLInputElement>(null);
	return (
		<div className="mt-4 flex flex-wrap items-end gap-3">
			<div className="min-w-48 flex-[2_1_16rem]">
				<label htmlFor="feed-search" className="block text-sm">
					{t("feeds.search")}
				</label>
				<div className="relative mt-2">
					<Search
						className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted"
						aria-hidden
					/>
					<input
						ref={input}
						id="feed-search"
						type="search"
						value={search}
						onChange={(e) => onSearch(e.target.value)}
						className={`${textInput.replace("mt-2 ", "")} px-9 [&::-webkit-search-cancel-button]:hidden`}
					/>
					{search !== "" && (
						<button
							type="button"
							onClick={() => {
								onSearch("");
								input.current?.focus();
							}}
							aria-label={t("feeds.clearSearch")}
							className={`${iconButton} absolute top-1/2 right-1.5 -translate-y-1/2`}
						>
							<X className="size-4" aria-hidden />
						</button>
					)}
				</div>
			</div>
			<div className="min-w-40 flex-[1_1_12rem]">
				<label htmlFor="feed-filter" className="block text-sm">
					{t("feeds.feedFilter")}
				</label>
				<select
					id="feed-filter"
					value={feedId}
					onChange={(e) => onFeed(e.target.value)}
					className={textInput}
				>
					<option value="">{t("feeds.allFeeds")}</option>
					{feeds.map((f) => (
						<option key={f.id} value={f.id}>
							{f.title}
						</option>
					))}
				</select>
			</div>
		</div>
	);
}

function Items({
	view,
	noFeeds,
	q,
	feedId,
}: {
	view: FeedView;
	noFeeds: boolean;
	q: string;
	feedId: string;
}) {
	const { t } = useTranslation();
	// While typing, the last results stay until the new ones arrive, but
	// another view never shows the items of the previous one.
	const pages = useInfiniteQuery({
		...feedItemsQueryOptions(view, { q, feedId }),
		placeholderData: (previous, previousQuery) =>
			previousQuery?.queryKey[1] === view ? previous : undefined,
	});
	const items = pages.data?.pages.flatMap((p) => p.items) ?? [];

	if (pages.isError) {
		return (
			<p role="alert" className="mt-6 text-sm text-danger">
				{t("common.genericError")}
			</p>
		);
	}
	if (!pages.data) return null;
	if (items.length === 0) {
		if (q || feedId) {
			return (
				<p className="mt-8 text-center text-muted">{t("feeds.noMatches")}</p>
			);
		}
		return view === "latest" && noFeeds ? (
			<div className="mt-8 rounded-xl border border-dashed border-line p-10 text-center">
				<Rss className="mx-auto size-8 text-muted" aria-hidden />
				<p className="mt-3 font-medium">{t("feeds.empty")}</p>
				<p className="mt-1 text-sm text-muted">{t("feeds.emptyHint")}</p>
			</div>
		) : (
			<p className="mt-8 text-center text-muted">{t("feeds.emptyView")}</p>
		);
	}
	return (
		<>
			<ul className="mt-4 flex flex-col gap-1">
				{items.map((item) => (
					<FeedItemRow key={item.id} item={item} showFlags showSummary />
				))}
			</ul>
			{pages.hasNextPage && (
				<div className="mt-6 text-center">
					<button
						type="button"
						disabled={pages.isFetchingNextPage}
						onClick={() => void pages.fetchNextPage()}
						className={secondaryButton}
					>
						{t("feeds.loadMore")}
					</button>
				</div>
			)}
		</>
	);
}

function FeedList({
	feeds,
	onEdit,
	onRemove,
}: {
	feeds: FeedSubscription[];
	onEdit: (feed: FeedSubscription) => void;
	onRemove: (feed: FeedSubscription) => void;
}) {
	const queryClient = useQueryClient();
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(TouchSensor, {
			activationConstraint: { delay: 200, tolerance: 5 },
		}),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);
	const reorder = useMutation({
		mutationFn: (ids: string[]) => reorderFeeds(ids),
		onMutate: (ids) => {
			const byId = new Map(feeds.map((f) => [f.id, f]));
			queryClient.setQueryData<FeedSubscription[]>(
				feedsQueryKey,
				ids.flatMap((id, position) => {
					const f = byId.get(id);
					return f ? [{ ...f, position }] : [];
				}),
			);
		},
		onSettled: () =>
			void queryClient.invalidateQueries({ queryKey: feedsQueryKey }),
	});
	const ids = feeds.map((f) => f.id);

	function onDragEnd({ active, over }: DragEndEvent) {
		if (!over || active.id === over.id) return;
		reorder.mutate(
			arrayMove(
				ids,
				ids.indexOf(String(active.id)),
				ids.indexOf(String(over.id)),
			),
		);
	}

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			onDragEnd={onDragEnd}
		>
			<SortableContext items={ids} strategy={verticalListSortingStrategy}>
				<ul className="mt-8 divide-y divide-line rounded-xl border border-line bg-surface">
					{feeds.map((feed) => (
						<FeedRow
							key={feed.id}
							feed={feed}
							onEdit={() => onEdit(feed)}
							onRemove={() => onRemove(feed)}
						/>
					))}
				</ul>
			</SortableContext>
		</DndContext>
	);
}

function FeedRow({
	feed,
	onEdit,
	onRemove,
}: {
	feed: FeedSubscription;
	onEdit: () => void;
	onRemove: () => void;
}) {
	const { t } = useTranslation();
	const {
		attributes,
		listeners,
		setNodeRef,
		setActivatorNodeRef,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: feed.id });

	return (
		<li
			ref={setNodeRef}
			style={{
				transform: transform
					? `translate3d(0, ${Math.round(transform.y)}px, 0)`
					: undefined,
				transition,
			}}
			className={`flex items-center gap-3 bg-surface px-2 py-3 first:rounded-t-xl last:rounded-b-xl sm:px-3 ${isDragging ? "relative z-10 shadow-md" : ""}`}
		>
			<button
				type="button"
				ref={setActivatorNodeRef}
				{...attributes}
				{...listeners}
				aria-label={t("feeds.drag", { name: feed.title })}
				className="flex size-8 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			>
				<GripVertical className="size-4" aria-hidden />
			</button>
			<span className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-line bg-paper">
				<ChosenIcon
					{...iconProps(feedIcon(feed))}
					fallback={Rss}
					className="size-6"
				/>
			</span>
			<div className="min-w-0 flex-1">
				<p className="truncate font-medium">{feed.title}</p>
				<p className="truncate text-sm text-muted">{hostOf(feed.url)}</p>
				{feed.failing && (
					<p className="truncate text-sm text-danger">{t("feeds.failing")}</p>
				)}
			</div>
			<Menu.Root>
				<Menu.Trigger
					aria-label={t("feeds.actions", { name: feed.title })}
					className={iconButton}
				>
					<EllipsisVertical className="size-4" aria-hidden />
				</Menu.Trigger>
				<Menu.Portal>
					<Menu.Positioner align="end" sideOffset={4} className="z-50">
						<Menu.Popup className={menuPopup}>
							<Menu.Item className={menuItem} onClick={onEdit}>
								<Pencil className="size-4 text-muted" aria-hidden />
								{t("feeds.edit")}
							</Menu.Item>
							<Menu.Item
								className={`${menuItem} text-danger`}
								onClick={onRemove}
							>
								<Trash2 className="size-4" aria-hidden />
								{t("feeds.remove")}
							</Menu.Item>
						</Menu.Popup>
					</Menu.Positioner>
				</Menu.Portal>
			</Menu.Root>
		</li>
	);
}

const addErrors = {
	invalid_url: "feeds.addDialog.errors.invalidUrl",
	feed_unreachable: "feeds.addDialog.errors.unreachable",
	feed_not_found: "feeds.addDialog.errors.notFound",
	already_subscribed: "feeds.addDialog.errors.already",
	rate_limited: "feeds.addDialog.errors.rateLimited",
} as const;

function AddFeedDialog({
	open,
	onClose,
}: {
	open: boolean;
	onClose: () => void;
}) {
	return (
		<Dialog.Root open={open} onOpenChange={(o) => !o && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{open && <AddFeedForm onClose={onClose} />}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function AddFeedForm({ onClose }: { onClose: () => void }) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [url, setUrl] = useState("");
	const add = useMutation({
		mutationFn: () => createFeed(url.trim()),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: feedsQueryKey });
			void queryClient.invalidateQueries({ queryKey: feedItemsQueryKey });
			onClose();
		},
	});
	const code = add.error instanceof ApiError ? add.error.code : null;
	const message =
		code && code in addErrors
			? t(addErrors[code as keyof typeof addErrors])
			: add.error
				? t("common.genericError")
				: null;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		add.mutate();
	}

	return (
		<form onSubmit={onSubmit} className="flex flex-col gap-4">
			<Dialog.Title className="text-lg font-semibold">
				{t("feeds.addDialog.title")}
			</Dialog.Title>
			<div>
				<label htmlFor="add-feed-url" className="block text-sm font-medium">
					{t("feeds.addDialog.urlLabel")}
				</label>
				<input
					id="add-feed-url"
					type="url"
					required
					value={url}
					onChange={(e) => setUrl(e.target.value)}
					aria-describedby="add-feed-url-hint"
					className={textInput}
				/>
				{message ? (
					<p
						id="add-feed-url-hint"
						role="alert"
						className="mt-1 text-sm text-danger"
					>
						{message}
					</p>
				) : (
					<p id="add-feed-url-hint" className="mt-1 text-sm text-muted">
						{t("feeds.addDialog.urlHint")}
					</p>
				)}
			</div>
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onClose} className={secondaryButton}>
					{t("feeds.cancel")}
				</button>
				<button
					type="submit"
					disabled={add.isPending}
					aria-busy={add.isPending}
					className={primaryButton}
				>
					{t("feeds.addDialog.submit")}
				</button>
			</div>
		</form>
	);
}

function EditFeedDialog({
	feed,
	onClose,
}: {
	feed: FeedSubscription | null;
	onClose: () => void;
}) {
	return (
		<Dialog.Root
			open={feed !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={dialogPopup}>
					{feed && <EditForm key={feed.id} feed={feed} onClose={onClose} />}
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function EditForm({
	feed,
	onClose,
}: {
	feed: FeedSubscription;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [title, setTitle] = useState(feed.customTitle ?? "");
	const [icon, setIcon] = useState<IconChoice>(() => iconChoice(feed.icon));
	const save = useMutation({
		mutationFn: () =>
			updateFeed(feed.id, {
				customTitle: title.trim() || null,
				icon: noteIcon(icon),
			}),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: feedsQueryKey });
			void queryClient.invalidateQueries({ queryKey: feedItemsQueryKey });
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
			onClose();
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		save.mutate();
	}

	return (
		<form onSubmit={onSubmit} className="flex flex-col gap-4">
			<Dialog.Title className="text-lg font-semibold">
				{t("feeds.editTitle")}
			</Dialog.Title>
			<div>
				<span className="block text-sm font-medium">
					{t("feeds.iconLabel")}
				</span>
				<div className="mt-2">
					<IconChooser value={icon} onChange={setIcon} />
				</div>
			</div>
			<div>
				<label htmlFor="edit-feed-title" className="block text-sm font-medium">
					{t("feeds.titleLabel")}
				</label>
				<input
					id="edit-feed-title"
					maxLength={200}
					value={title}
					placeholder={feed.title}
					onChange={(e) => setTitle(e.target.value)}
					className={textInput}
				/>
			</div>
			{save.isError && (
				<p role="alert" className="text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			<div className="flex justify-end gap-2">
				<button type="button" onClick={onClose} className={secondaryButton}>
					{t("feeds.cancel")}
				</button>
				<button
					type="submit"
					disabled={save.isPending}
					className={primaryButton}
				>
					{t("feeds.save")}
				</button>
			</div>
		</form>
	);
}

function RemoveFeedDialog({
	feed,
	onClose,
}: {
	feed: FeedSubscription | null;
	onClose: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const remove = useMutation({
		mutationFn: (id: string) => deleteFeed(id),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: feedsQueryKey });
			void queryClient.invalidateQueries({ queryKey: feedItemsQueryKey });
			void queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
			onClose();
		},
	});

	return (
		<AlertDialog.Root
			open={feed !== null}
			onOpenChange={(open) => !open && onClose()}
		>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{t("feeds.removeTitle")}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-2 text-muted">
						{t("feeds.removeText", { name: feed?.title ?? "" })}
					</AlertDialog.Description>
					{remove.isError && (
						<p role="alert" className="mt-3 text-sm text-danger">
							{t("common.genericError")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-2">
						<AlertDialog.Close className={secondaryButton}>
							{t("feeds.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							disabled={remove.isPending}
							onClick={() => feed && remove.mutate(feed.id)}
							className={dangerButton}
						>
							{t("feeds.remove")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
