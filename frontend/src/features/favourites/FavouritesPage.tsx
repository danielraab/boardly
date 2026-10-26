import { Menu } from "@base-ui/react/menu";
import {
	type Announcements,
	DndContext,
	type DragEndEvent,
	type DragMoveEvent,
	DragOverlay,
	PointerSensor,
	pointerWithin,
	useDraggable,
	useDroppable,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate } from "@tanstack/react-router";
import {
	Check,
	ChevronRight,
	ChevronsDownUp,
	ChevronsUpDown,
	Folder,
	FolderPlus,
	GripVertical,
	ListChecks,
	Pencil,
	Plus,
	Search,
	SearchX,
	Star,
	Tag as TagIcon,
	Tags as TagsIcon,
	Trash2,
	X,
} from "lucide-react";
import {
	type KeyboardEvent,
	type ReactNode,
	useCallback,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
	iconButton,
	secondaryButton,
	secondaryIconButton,
} from "#/components/buttonStyles";
import { ImageIcon } from "#/components/ImageIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { Toast, type ToastMessage } from "#/components/Toast";
import { useMe } from "#/features/auth/session";
import { usePageTitle } from "#/lib/usePageTitle";
import { useStoredIdSet } from "#/lib/useStoredIdSet";
import {
	bulkFavourites,
	type FavouriteNode,
	favouritesQueryKey,
	favouritesQueryOptions,
	moveFavourite,
} from "./api";
import {
	BulkDialog,
	type BulkDialogKind,
	type BulkDialogTarget,
	SelectionBar,
} from "./BulkActions";
import { DeleteDialog, type DeleteTarget } from "./DeleteDialog";
import { ManageTagsDialog, type TagChange } from "./ManageTagsDialog";
import { NodeDialog, type NodeDialogTarget } from "./NodeDialog";
import {
	applyMove,
	type CheckState,
	checkState,
	currentSlot,
	filterTree,
	findNode,
	findParent,
	flatten,
	folderPath,
	highlight,
	type Indicator,
	indicatorFor,
	isFiltering,
	isInSubtree,
	keyboardSlots,
	label,
	linkCounts,
	moveRoots,
	pointerSlot,
	pruneSelection,
	ROOT_END,
	type Slot,
	sameSlot,
	searchTerms,
	selectedLinks,
	selectionCounts,
	selectionTagCounts,
	selectRange,
	tagCounts,
	toggleSelection,
	withoutNodes,
} from "./tree";

type Actions = {
	add: (kind: "folder" | "link", parent: FavouriteNode) => void;
	edit: (node: FavouriteNode) => void;
	remove: (node: FavouriteNode) => void;
	filterTag: (tag: string) => void;
};

export type FavouritesSearch = {
	q?: string;
	tags?: string[];
	folder?: string;
};

/** Changes the page's search params; replace keeps the change out of the history. */
export type SearchChange = (
	patch: Partial<FavouritesSearch>,
	replace?: boolean,
) => void;

const HIGHLIGHT_MS = 2000;

export function FavouritesPage({
	search,
	onSearchChange,
}: {
	search: FavouritesSearch;
	onSearchChange: SearchChange;
}) {
	const { t } = useTranslation();
	usePageTitle(t("favourites.title"));
	const { data: me } = useMe();
	const tree = useQuery({ ...favouritesQueryOptions, enabled: !!me });
	const [editing, setEditing] = useState<NodeDialogTarget | null>(null);
	const [deleting, setDeleting] = useState<DeleteTarget | null>(null);
	const [remembered, setRemembered] = useStoredIdSet(
		`boardly.favourites.collapsed.${me?.id ?? ""}`,
	);
	const [text, setText] = useState(search.q ?? "");
	const searchInput = useRef<HTMLInputElement>(null);
	const [highlighted, setHighlighted] = useState<string | null>(null);
	const queryClient = useQueryClient();
	const [selecting, setSelecting] = useState(false);
	const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
	const lastClicked = useRef<number | null>(null);
	const [bulkDialog, setBulkDialog] = useState<BulkDialogTarget | null>(null);
	const [managingTags, setManagingTags] = useState(false);
	const [toast, setToast] = useState<ToastMessage | null>(null);
	const [bulkError, setBulkError] = useState(false);
	// Links deleted in bulk are hidden until the delete is sent (see commitDelete).
	const [pendingIds, setPendingIds] = useState<ReadonlySet<string>>(new Set());
	const pendingDelete = useRef<string[] | null>(null);

	// The URL changes the text only from outside the field (back, clear).
	useEffect(() => {
		if (document.activeElement !== searchInput.current) {
			setText(search.q ?? "");
		}
	}, [search.q]);

	const root = useMemo(
		() => tree.data && withoutNodes(tree.data, pendingIds),
		[tree.data, pendingIds],
	);
	const tags = useMemo(() => search.tags ?? [], [search.tags]);
	const filter = useMemo(
		() => ({ terms: searchTerms(text), tags }),
		[text, tags],
	);
	const filtering = isFiltering(filter);
	const shown = useMemo(
		() => (root ? filterTree(root, filter) : undefined),
		[root, filter],
	);
	// While filtering, every shown folder starts expanded and nothing is remembered.
	const filterKey = JSON.stringify(filter);
	const [filterCollapsed, setFilterCollapsed] = useState<{
		key: string;
		ids: ReadonlySet<string>;
	}>({ key: "", ids: new Set() });
	const collapsed = filtering
		? filterCollapsed.key === filterKey
			? filterCollapsed.ids
			: new Set<string>()
		: remembered;
	const setCollapsed = (ids: ReadonlySet<string>) =>
		filtering
			? setFilterCollapsed({ key: filterKey, ids })
			: setRemembered(ids);

	// Forget folders that no longer exist.
	useEffect(() => {
		if (!root) return;
		const folders = new Set(folderIds(root));
		if ([...remembered].some((id) => !folders.has(id))) {
			setRemembered(new Set([...remembered].filter((id) => folders.has(id))));
		}
	}, [root, remembered, setRemembered]);

	// ?folder= opens that folder once and then leaves the URL.
	const openedFolder = useRef<string | null>(null);
	useEffect(() => {
		if (!root || !search.folder || openedFolder.current === search.folder) {
			return;
		}
		openedFolder.current = search.folder;
		const path = folderPath(root, search.folder);
		const target = findNode(root, search.folder);
		if (target?.kind === "folder" && path.length > 0) {
			const next = new Set(remembered);
			for (const id of path) next.delete(id);
			setRemembered(next);
			setHighlighted(target.id);
		}
		onSearchChange({ folder: undefined }, true);
	}, [root, search.folder, remembered, setRemembered, onSearchChange]);

	useEffect(() => {
		if (!highlighted) return;
		const timer = setTimeout(() => setHighlighted(null), HIGHLIGHT_MS);
		return () => clearTimeout(timer);
	}, [highlighted]);

	// Deleted or moved elsewhere: no longer selected.
	useEffect(() => {
		if (root) setSelected((prev) => pruneSelection(root, prev));
	}, [root]);

	/** Sends the pending bulk delete; keepalive when the page is being left. */
	const commitDelete = useCallback(
		(keepalive = false) => {
			const ids = pendingDelete.current;
			if (!ids) return;
			pendingDelete.current = null;
			const unhide = () =>
				setPendingIds((prev) => {
					const next = new Set(prev);
					for (const id of ids) next.delete(id);
					return next;
				});
			bulkFavourites({ action: "delete", ids }, keepalive)
				.then(async () => {
					void queryClient.invalidateQueries({ queryKey: ["tags"] });
					await queryClient.invalidateQueries({
						queryKey: favouritesQueryKey,
					});
					unhide();
				})
				.catch(() => {
					unhide();
					setBulkError(true);
				});
		},
		[queryClient],
	);

	// Leaving the page or the app sends a pending delete.
	useEffect(() => {
		const onPageHide = () => commitDelete(true);
		window.addEventListener("pagehide", onPageHide);
		return () => {
			window.removeEventListener("pagehide", onPageHide);
			commitDelete(true);
		};
	}, [commitDelete]);

	// Escape ends selection mode, unless a dialog, menu or field has it.
	useEffect(() => {
		if (!selecting) return;
		const onKeyDown = (event: globalThis.KeyboardEvent) => {
			const target = event.target as HTMLElement | null;
			if (
				event.key !== "Escape" ||
				target?.closest("input:not([type=checkbox]), textarea, select") ||
				document.querySelector("[role=dialog], [role=alertdialog], [role=menu]")
			) {
				return;
			}
			setSelecting(false);
			setSelected(new Set());
		};
		window.addEventListener("keydown", onKeyDown);
		return () => window.removeEventListener("keydown", onKeyDown);
	}, [selecting]);

	// "/" focuses the search field, unless the user is typing or in a dialog.
	useEffect(() => {
		const onKeyDown = (event: globalThis.KeyboardEvent) => {
			if (event.key !== "/" || event.ctrlKey || event.metaKey || event.altKey) {
				return;
			}
			const target = event.target as HTMLElement | null;
			if (
				target?.closest("input, textarea, select, [contenteditable]") ||
				document.querySelector("[role=dialog], [role=alertdialog]")
			) {
				return;
			}
			event.preventDefault();
			searchInput.current?.focus();
		};
		window.addEventListener("keydown", onKeyDown);
		return () => window.removeEventListener("keydown", onKeyDown);
	}, []);

	// The session can end while the page is open (401 from any request).
	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;

	const folderLabel = (folder: FavouriteNode) =>
		folder.id === root?.id ? t("favourites.title") : label(folder);
	const actions: Actions = {
		add: (kind, parent) =>
			setEditing({ kind, parent, parentLabel: folderLabel(parent) }),
		edit: (node) => setEditing({ node }),
		remove: (node) => {
			const parent = root && findParent(root, node.id)?.parent;
			setDeleting({ node, parentLabel: parent ? folderLabel(parent) : "" });
		},
		filterTag: (tag) => {
			if (!tags.some((chosen) => sameTag(chosen, tag))) {
				onSearchChange({ tags: [...tags, tag] });
			}
		},
	};
	const changeText = (value: string) => {
		setText(value);
		onSearchChange({ q: value.trim() === "" ? undefined : value }, true);
	};
	const setTags = (next: string[]) =>
		onSearchChange({ tags: next.length > 0 ? next : undefined });
	// Keeps the tag filter pointing at tags that exist after managing them.
	const onTagChange = (change: TagChange) => {
		const name = change.kind === "renamed" ? change.from : change.name;
		if (!tags.some((chosen) => sameTag(chosen, name))) return;
		const rest = tags.filter((chosen) => !sameTag(chosen, name));
		const to = change.kind === "renamed" ? change.to : null;
		setTags(
			to && !rest.some((chosen) => sameTag(chosen, to)) ? [...rest, to] : rest,
		);
	};
	const clearFilters = () => {
		setText("");
		onSearchChange({ q: undefined, tags: undefined });
	};

	const isEmpty = root && (root.children ?? []).length === 0;
	const shownFolders = shown ? folderIds(shown) : [];
	const anyExpanded = shownFolders.some((id) => !collapsed.has(id));

	const stopSelecting = () => {
		setSelecting(false);
		setSelected(new Set());
		lastClicked.current = null;
	};
	const toggleRow = (index: number, shiftKey: boolean) => {
		if (!shown) return;
		const rows = flatten(shown, collapsed).map((item) => item.node);
		const node = rows[index];
		const on = checkState(node, selected) !== "checked";
		setSelected(
			shiftKey && lastClicked.current !== null
				? selectRange(rows, lastClicked.current, index, selected, on)
				: toggleSelection(node, selected, on),
		);
		lastClicked.current = index;
	};
	const counts = root
		? selectionCounts(root, selected)
		: { links: 0, folders: 0 };
	const roots = root ? moveRoots(root, selected) : [];
	const openBulk = (kind: BulkDialogKind) => {
		if (!root) return;
		setBulkError(false);
		setBulkDialog(
			kind === "move"
				? {
						kind,
						nodes: roots,
						blocked: new Set(
							roots.filter((n) => n.kind === "folder").map((n) => n.id),
						),
					}
				: {
						kind,
						links: selectedLinks(root, selected),
						tags: selectionTagCounts(root, selected),
					},
		);
	};
	const deleteSelected = () => {
		if (!root) return;
		const ids = selectedLinks(root, selected).map((n) => n.id);
		// An earlier delete still waiting for its undo is sent first.
		commitDelete();
		pendingDelete.current = ids;
		setPendingIds((prev) => new Set([...prev, ...ids]));
		setSelected(new Set());
		lastClicked.current = null;
		setBulkError(false);
		setToast({
			id: Date.now(),
			text: t("favourites.bulk.deleted", { count: ids.length }),
			action: {
				label: t("favourites.bulk.undo"),
				onClick: () => {
					pendingDelete.current = null;
					setPendingIds((prev) => {
						const next = new Set(prev);
						for (const id of ids) next.delete(id);
						return next;
					});
					setToast(null);
				},
			},
		});
	};

	return (
		<section className="pt-12">
			<div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
				<div>
					<h1 className="text-3xl font-semibold tracking-tight">
						{t("favourites.title")}
					</h1>
					<p className="mt-2 max-w-[60ch] text-muted">
						{t("favourites.intro")}
					</p>
				</div>
				{root && (
					<div className="flex shrink-0 gap-2">
						<button
							type="button"
							className={secondaryButton}
							onClick={() => actions.add("folder", root)}
						>
							<FolderPlus className="size-4" aria-hidden />
							{t("favourites.addFolder")}
						</button>
						<button
							type="button"
							className={secondaryButton}
							onClick={() => actions.add("link", root)}
						>
							<Plus className="size-4" aria-hidden />
							{t("favourites.addLink")}
						</button>
					</div>
				)}
			</div>
			{root && !isEmpty && (
				<div className="mt-8">
					<div className="flex items-center gap-2">
						<div className="relative min-w-0 flex-1">
							<Search
								className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted"
								aria-hidden
							/>
							<input
								ref={searchInput}
								type="search"
								value={text}
								onChange={(event) => changeText(event.target.value)}
								aria-label={t("favourites.search.label")}
								placeholder={t("favourites.search.placeholder")}
								className="w-full rounded-lg border border-line bg-paper/40 py-2 pr-3 pl-9 text-sm outline-none focus:border-ink"
							/>
						</div>
						<TagFilter
							available={tagCounts(root)}
							chosen={tags}
							onChange={setTags}
						/>
						{shownFolders.length > 0 && (
							<button
								type="button"
								className={secondaryIconButton}
								aria-label={t(
									anyExpanded
										? "favourites.collapseAll"
										: "favourites.expandAll",
								)}
								title={t(
									anyExpanded
										? "favourites.collapseAll"
										: "favourites.expandAll",
								)}
								onClick={() =>
									setCollapsed(new Set(anyExpanded ? shownFolders : []))
								}
							>
								{anyExpanded ? (
									<ChevronsDownUp className="size-4" aria-hidden />
								) : (
									<ChevronsUpDown className="size-4" aria-hidden />
								)}
							</button>
						)}
					</div>
					<div className="mt-3 flex flex-wrap items-center gap-2 text-sm">
						{tags.map((tag) => (
							<span
								key={tag}
								className="inline-flex items-center gap-1 rounded-full bg-paper py-0.5 pr-1 pl-2.5 text-muted"
							>
								{tag}
								<button
									type="button"
									className="inline-flex size-5 items-center justify-center rounded-full hover:bg-surface hover:text-ink"
									aria-label={t("favourites.tagFilter.remove", { tag })}
									onClick={() =>
										setTags(tags.filter((chosen) => chosen !== tag))
									}
								>
									<X className="size-3" aria-hidden />
								</button>
							</span>
						))}
						<div className="ml-auto flex items-center gap-4">
							{filtering && shown && (
								<button
									type="button"
									className="text-muted underline-offset-2 hover:text-ink hover:underline"
									onClick={clearFilters}
								>
									{t("favourites.clearFilters")}
								</button>
							)}
							{selecting && shown && (
								<button
									type="button"
									className="text-muted underline-offset-2 hover:text-ink hover:underline"
									onClick={() =>
										setSelected(
											(shown.children ?? []).reduce(
												(acc, node) => toggleSelection(node, acc, true),
												selected,
											),
										)
									}
								>
									{t("favourites.select.all")}
								</button>
							)}
							<button
								type="button"
								className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 font-medium hover:bg-surface"
								onClick={() => setManagingTags(true)}
							>
								<TagsIcon className="size-4" aria-hidden />
								{t("favourites.manageTags.button")}
							</button>
							<button
								type="button"
								aria-pressed={selecting}
								className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 font-medium hover:bg-surface aria-pressed:bg-surface"
								onClick={() =>
									selecting ? stopSelecting() : setSelecting(true)
								}
							>
								{selecting ? (
									t("favourites.select.done")
								) : (
									<>
										<ListChecks className="size-4" aria-hidden />
										{t("favourites.select.start")}
									</>
								)}
							</button>
						</div>
					</div>
				</div>
			)}
			<div className="mt-4">
				{tree.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : tree.isError ? (
					<p role="alert" className="text-sm text-danger">
						{t("common.genericError")}
					</p>
				) : isEmpty ? (
					<div className="mt-4 rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
						<Star
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-4 text-lg font-semibold">
							{t("favourites.emptyTitle")}
						</h2>
						<p className="mx-auto mt-1 max-w-[45ch] text-muted">
							{t("favourites.emptyText")}
						</p>
					</div>
				) : !shown ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
						<SearchX
							className="mx-auto size-8 text-muted"
							strokeWidth={1.75}
							aria-hidden
						/>
						<h2 className="mt-4 text-lg font-semibold">
							{t("favourites.noMatch")}
						</h2>
						<button
							type="button"
							className={`${secondaryButton} mt-4`}
							onClick={clearFilters}
						>
							{t("favourites.clearFilters")}
						</button>
					</div>
				) : (
					<>
						{filtering && !selecting && (
							<p className="mb-3 text-sm text-muted">
								{t("favourites.reorderHint")}
							</p>
						)}
						{bulkError && (
							<p role="alert" className="mb-3 text-sm text-danger">
								{t("favourites.bulk.deleteFailed")}
							</p>
						)}
						{selecting && selected.size > 0 && (
							<SelectionBar
								links={counts.links}
								folders={counts.folders}
								moveCount={roots.length}
								onAction={openBulk}
								onDelete={deleteSelected}
							/>
						)}
						<TreeView
							root={shown}
							collapsed={collapsed}
							onCollapsedChange={setCollapsed}
							movable={!filtering && !selecting}
							selection={
								selecting
									? {
											state: (node) => checkState(node, selected),
											onToggle: toggleRow,
										}
									: undefined
							}
							terms={filter.terms}
							highlighted={highlighted}
							actions={actions}
							folderLabel={folderLabel}
						/>
					</>
				)}
			</div>
			<NodeDialog target={editing} onClose={() => setEditing(null)} />
			<DeleteDialog target={deleting} onClose={() => setDeleting(null)} />
			<BulkDialog target={bulkDialog} onClose={() => setBulkDialog(null)} />
			<ManageTagsDialog
				open={managingTags}
				onClose={() => setManagingTags(false)}
				onFilter={(tag) => {
					setManagingTags(false);
					onSearchChange({ tags: [tag] });
				}}
				onChange={onTagChange}
			/>
			<Toast
				toast={toast}
				onClose={() => {
					setToast(null);
					commitDelete();
				}}
			/>
		</section>
	);
}

const sameTag = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();

/** The IDs of every folder below the root. */
function folderIds(root: FavouriteNode): string[] {
	const out: string[] = [];
	const visit = (folder: FavouriteNode) => {
		for (const child of folder.children ?? []) {
			if (child.kind !== "folder") continue;
			out.push(child.id);
			visit(child);
		}
	};
	visit(root);
	return out;
}

/** A menu of the user's tags with link counts; links must have every checked tag. */
function TagFilter({
	available,
	chosen,
	onChange,
}: {
	available: { name: string; count: number }[];
	chosen: string[];
	onChange: (tags: string[]) => void;
}) {
	const { t } = useTranslation();
	if (available.length === 0) return null;
	return (
		<Menu.Root>
			<Menu.Trigger className={secondaryButton}>
				<TagIcon className="size-4" aria-hidden />
				{chosen.length > 0
					? t("favourites.tagFilter.buttonCount", { count: chosen.length })
					: t("favourites.tagFilter.button")}
			</Menu.Trigger>
			<Menu.Portal>
				<Menu.Positioner align="end" sideOffset={6}>
					<Menu.Popup className={`${menuPopup} max-h-80 overflow-y-auto`}>
						{available.map(({ name, count }) => {
							const checked = chosen.some((tag) => sameTag(tag, name));
							return (
								<Menu.CheckboxItem
									key={name}
									checked={checked}
									onCheckedChange={(next) =>
										onChange(
											next
												? [...chosen, name]
												: chosen.filter((tag) => !sameTag(tag, name)),
										)
									}
									className={menuItem}
								>
									<span className="flex size-4 items-center justify-center">
										<Menu.CheckboxItemIndicator>
											<Check className="size-4" aria-hidden />
										</Menu.CheckboxItemIndicator>
									</span>
									<span className="min-w-0 flex-1 truncate">{name}</span>
									<span className="text-xs text-muted tabular-nums">
										{count}
									</span>
								</Menu.CheckboxItem>
							);
						})}
					</Menu.Popup>
				</Menu.Positioner>
			</Menu.Portal>
		</Menu.Root>
	);
}

/** Where the pointer is vertically, from the event that started the drag. */
function startY(event: Event): number | null {
	if ("clientY" in event && typeof event.clientY === "number") {
		return event.clientY;
	}
	if ("touches" in event) {
		return (event as TouchEvent).touches[0]?.clientY ?? null;
	}
	return null;
}

type KeyboardDrag = { id: string; slots: Slot[]; index: number };

function TreeView({
	root,
	collapsed,
	onCollapsedChange,
	movable,
	selection,
	terms,
	highlighted,
	actions,
	folderLabel,
}: {
	root: FavouriteNode;
	collapsed: ReadonlySet<string>;
	onCollapsedChange: (ids: ReadonlySet<string>) => void;
	/** Off while filtering: positions in a filtered tree aren't real ones. */
	movable: boolean;
	/** In selection mode: each row's checkbox, toggled by its row index. */
	selection?: {
		state: (node: FavouriteNode) => CheckState;
		onToggle: (index: number, shiftKey: boolean) => void;
	};
	terms: string[];
	highlighted: string | null;
	actions: Actions;
	folderLabel: (folder: FavouriteNode) => string;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const counts = useMemo(() => linkCounts(root), [root]);
	const [activeId, setActiveId] = useState<string | null>(null);
	const [pointer, setPointer] = useState<Slot | null>(null);
	const [keyboard, setKeyboard] = useState<KeyboardDrag | null>(null);
	const [announcement, setAnnouncement] = useState("");
	const [refocus, setRefocus] = useState<string | null>(null);
	const handles = useRef(new Map<string, HTMLButtonElement>());

	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
	);

	const move = useMutation({
		mutationFn: (v: { id: string; parentId: string; position: number }) =>
			moveFavourite(v.id, v.parentId, v.position),
		onMutate: async (v) => {
			await queryClient.cancelQueries({ queryKey: favouritesQueryKey });
			const previous =
				queryClient.getQueryData<FavouriteNode>(favouritesQueryKey);
			if (previous) {
				queryClient.setQueryData(favouritesQueryKey, (tree: FavouriteNode) =>
					applyMoveSafe(tree, v),
				);
			}
			return { previous };
		},
		onError: (_error, _v, context) => {
			if (context?.previous) {
				queryClient.setQueryData(favouritesQueryKey, context.previous);
			}
		},
		onSettled: () =>
			queryClient.invalidateQueries({ queryKey: favouritesQueryKey }),
	});

	// Keyboard moves keep focus on the moved node's handle.
	useEffect(() => {
		if (!refocus) return;
		handles.current.get(refocus)?.focus();
		setRefocus(null);
	}, [refocus]);

	const name = (id: string) => {
		const node = findNode(root, id);
		return node ? label(node) : "";
	};

	function commit(id: string, slot: Slot | null) {
		if (!slot || sameSlot(currentSlot(root, id), { ...slot, into: false })) {
			return;
		}
		move.mutate({ id, parentId: slot.parentId, position: slot.position });
	}

	function describe(slot: Slot) {
		const folder = findNode(root, slot.parentId);
		const folderName = folder ? folderLabel(folder) : "";
		return slot.into
			? t("favourites.dnd.into", { folder: folderName })
			: t("favourites.dnd.position", {
					position: slot.position + 1,
					folder: folderName,
				});
	}

	function slotFor(event: DragMoveEvent | DragEndEvent): Slot | null {
		const { active, over, delta, activatorEvent } = event;
		const y = startY(activatorEvent);
		if (!over || y === null) return null;
		const { top, height } = over.rect;
		const fraction = height > 0 ? (y + delta.y - top) / height : 0.5;
		return pointerSlot(
			root,
			collapsed,
			String(active.id),
			String(over.id),
			fraction,
		);
	}

	function onHandleKeyDown(event: KeyboardEvent, node: FavouriteNode) {
		if (keyboard?.id !== node.id) {
			if (event.key !== " " && event.key !== "Enter") return;
			event.preventDefault();
			const slots = keyboardSlots(root, collapsed, node.id);
			const current = currentSlot(root, node.id);
			const index = Math.max(
				0,
				slots.findIndex((s) => sameSlot(s, current)),
			);
			setKeyboard({ id: node.id, slots, index });
			setAnnouncement(t("favourites.dnd.pickedUp", { name: label(node) }));
			return;
		}
		switch (event.key) {
			case "ArrowUp":
			case "ArrowDown": {
				event.preventDefault();
				const step = event.key === "ArrowUp" ? -1 : 1;
				const index = Math.min(
					Math.max(keyboard.index + step, 0),
					keyboard.slots.length - 1,
				);
				setKeyboard({ ...keyboard, index });
				setAnnouncement(describe(keyboard.slots[index]));
				break;
			}
			case " ":
			case "Enter":
				event.preventDefault();
				setKeyboard(null);
				commit(node.id, keyboard.slots[keyboard.index]);
				setRefocus(node.id);
				setAnnouncement(t("favourites.dnd.dropped", { name: label(node) }));
				break;
			case "Escape":
				event.preventDefault();
				setKeyboard(null);
				setAnnouncement(t("favourites.dnd.cancelled", { name: label(node) }));
				break;
		}
	}

	const announcements: Announcements = {
		onDragStart: ({ active }) =>
			t("favourites.dnd.pickedUp", { name: name(String(active.id)) }),
		onDragOver: () => undefined,
		onDragEnd: ({ active }) =>
			t("favourites.dnd.dropped", { name: name(String(active.id)) }),
		onDragCancel: ({ active }) =>
			t("favourites.dnd.cancelled", { name: name(String(active.id)) }),
	};

	const draggedId = activeId ?? keyboard?.id ?? null;
	const slot = activeId
		? pointer
		: keyboard
			? keyboard.slots[keyboard.index]
			: null;
	const indicator =
		draggedId && slot ? indicatorFor(root, collapsed, draggedId, slot) : null;
	const reset = () => {
		setActiveId(null);
		setPointer(null);
	};

	return (
		<DndContext
			sensors={sensors}
			collisionDetection={pointerWithin}
			accessibility={{
				announcements,
				screenReaderInstructions: {
					draggable: t("favourites.dnd.instructions"),
				},
			}}
			onDragStart={({ active }) => {
				setKeyboard(null);
				setActiveId(String(active.id));
			}}
			onDragMove={(event) => {
				const next = slotFor(event);
				setPointer((prev) => (sameSlot(prev, next) ? prev : next));
			}}
			onDragEnd={(event) => {
				commit(String(event.active.id), slotFor(event));
				reset();
			}}
			onDragCancel={reset}
		>
			{move.isError && (
				<p role="alert" className="mb-3 text-sm text-danger">
					{t("favourites.errors.moveFailed")}
				</p>
			)}
			{/* Deep rows scroll sideways inside the frame instead of squeezing their
			    names to nothing. Each row is as wide as it needs, but at least as
			    wide as the frame, so shallow rows keep their buttons at its edge. */}
			<div className="@container overflow-x-auto rounded-xl border border-line bg-surface">
				<ul className="w-max min-w-full py-1 [--indent:0.75rem] sm:[--indent:1.5rem]">
					{flatten(root, collapsed).map(({ node, depth }, index) => (
						<TreeRow
							key={node.id}
							node={node}
							depth={depth}
							indicator={indicator?.rowId === node.id ? indicator : null}
							dimmed={!!draggedId && isInSubtree(root, draggedId, node.id)}
							picked={keyboard?.id === node.id}
							expanded={!collapsed.has(node.id)}
							onToggle={() => {
								const next = new Set(collapsed);
								if (!next.delete(node.id)) next.add(node.id);
								onCollapsedChange(next);
							}}
							movable={movable}
							check={
								selection && {
									state: selection.state(node),
									onToggle: (shiftKey) => selection.onToggle(index, shiftKey),
								}
							}
							terms={terms}
							highlighted={highlighted === node.id}
							linkCount={counts.get(node.id) ?? 0}
							onHandleKeyDown={(event) => onHandleKeyDown(event, node)}
							onHandleBlur={() => {
								if (keyboard?.id === node.id) setKeyboard(null);
							}}
							registerHandle={(el) => {
								if (el) handles.current.set(node.id, el);
								else handles.current.delete(node.id);
							}}
							actions={actions}
						/>
					))}
				</ul>
			</div>
			<RootEnd visible={activeId !== null} />
			<output aria-live="assertive" className="sr-only">
				{announcement}
			</output>
			<DragOverlay dropAnimation={null}>
				{activeId && (
					<div className="inline-flex max-w-80 items-center gap-2 rounded-lg border border-line bg-surface px-3 py-2 shadow-lg">
						<GripVertical className="size-4 shrink-0 text-muted" aria-hidden />
						<span className="truncate">{name(activeId)}</span>
					</div>
				)}
			</DragOverlay>
		</DndContext>
	);
}

/** applyMove that ignores a move the server would reject anyway. */
function applyMoveSafe(
	tree: FavouriteNode,
	v: { id: string; parentId: string; position: number },
) {
	if (isInSubtree(tree, v.id, v.parentId)) return tree;
	return applyMove(tree, v.id, v.parentId, v.position);
}

function RootEnd({ visible }: { visible: boolean }) {
	const { t } = useTranslation();
	const { setNodeRef, isOver } = useDroppable({ id: ROOT_END });
	return (
		<div
			ref={setNodeRef}
			className={
				visible
					? `mt-2 rounded-lg border border-dashed px-3 py-3 text-center text-sm text-muted ${isOver ? "border-accent bg-accent/10" : "border-line"}`
					: "h-0"
			}
		>
			{visible && t("favourites.dropAtEnd")}
		</div>
	);
}

function TreeRow({
	node,
	depth,
	indicator,
	dimmed,
	picked,
	expanded,
	onToggle,
	onHandleKeyDown,
	onHandleBlur,
	registerHandle,
	movable,
	check,
	terms,
	highlighted,
	linkCount,
	actions,
}: {
	node: FavouriteNode;
	depth: number;
	indicator: Indicator | null;
	dimmed: boolean;
	picked: boolean;
	expanded: boolean;
	onToggle: () => void;
	onHandleKeyDown: (event: KeyboardEvent) => void;
	onHandleBlur: () => void;
	registerHandle: (el: HTMLButtonElement | null) => void;
	movable: boolean;
	check?: { state: CheckState; onToggle: (shiftKey: boolean) => void };
	terms: string[];
	highlighted: boolean;
	linkCount: number;
	actions: Actions;
}) {
	const { t } = useTranslation();
	const rowRef = useRef<HTMLLIElement | null>(null);
	useEffect(() => {
		// jsdom has no scrollIntoView.
		if (highlighted) rowRef.current?.scrollIntoView?.({ block: "center" });
	}, [highlighted]);
	const { setNodeRef: setDropRef } = useDroppable({ id: node.id });
	const {
		setNodeRef: setDragRef,
		setActivatorNodeRef,
		attributes,
		listeners,
	} = useDraggable({ id: node.id, disabled: !movable });
	const name = label(node);
	const isFolder = node.kind === "folder";

	let actionButtons: ReactNode = (
		<>
			<button
				type="button"
				className={iconButton}
				aria-label={t("favourites.edit", { name })}
				onClick={() => actions.edit(node)}
			>
				<Pencil className="size-4" aria-hidden />
			</button>
			<button
				type="button"
				className={`${iconButton} hover:text-danger`}
				aria-label={t("favourites.delete", { name })}
				onClick={() => actions.remove(node)}
			>
				<Trash2 className="size-4" aria-hidden />
			</button>
		</>
	);
	if (isFolder) {
		actionButtons = (
			<>
				<button
					type="button"
					className={iconButton}
					aria-label={t("favourites.addFolderTo", { name })}
					onClick={() => actions.add("folder", node)}
				>
					<FolderPlus className="size-4" aria-hidden />
				</button>
				<button
					type="button"
					className={iconButton}
					aria-label={t("favourites.addLinkTo", { name })}
					onClick={() => actions.add("link", node)}
				>
					<Plus className="size-4" aria-hidden />
				</button>
				{actionButtons}
			</>
		);
	}

	return (
		<li
			ref={(el) => {
				setDropRef(el);
				setDragRef(el);
				rowRef.current = el;
			}}
			data-testid={`row-${node.id}`}
			data-highlighted={highlighted || undefined}
			className={`relative mx-1 flex w-min min-w-[calc(100cqw-0.5rem)] items-center gap-1 rounded-lg py-1 pr-1 transition-colors motion-reduce:transition-none ${indicator?.edge === "inside" || highlighted ? "bg-accent/10 ring-2 ring-accent" : ""} ${dimmed ? "opacity-40" : ""}`}
			style={{ paddingLeft: `calc(var(--indent) * ${depth} + 0.25rem)` }}
		>
			{indicator && indicator.edge !== "inside" && (
				<span
					aria-hidden
					className={`pointer-events-none absolute right-2 h-0.5 rounded-full bg-accent ${indicator.edge === "top" ? "-top-px" : "-bottom-px"}`}
					style={{ left: `calc(var(--indent) * ${indicator.depth} + 2.25rem)` }}
				/>
			)}
			{check ? (
				<span className="inline-flex size-7 shrink-0 items-center justify-center">
					<input
						type="checkbox"
						readOnly
						checked={check.state === "checked"}
						ref={(el) => {
							if (el) el.indeterminate = check.state === "mixed";
						}}
						aria-label={t("favourites.select.row", { name })}
						className="size-4 accent-ink"
						onClick={(event) => check.onToggle(event.shiftKey)}
					/>
				</span>
			) : movable ? (
				<button
					type="button"
					ref={(el) => {
						setActivatorNodeRef(el);
						registerHandle(el);
					}}
					{...attributes}
					{...listeners}
					aria-pressed={picked}
					aria-label={t("favourites.move", { name })}
					className={`${iconButton} cursor-grab touch-none active:cursor-grabbing ${picked ? "bg-accent/20 text-ink" : ""}`}
					onKeyDown={onHandleKeyDown}
					onBlur={onHandleBlur}
				>
					<GripVertical className="size-4" aria-hidden />
				</button>
			) : (
				<span className="size-7 shrink-0" />
			)}
			{isFolder ? (
				<button
					type="button"
					className={iconButton}
					aria-expanded={expanded}
					aria-label={t(
						expanded ? "favourites.collapse" : "favourites.expand",
						{
							name,
						},
					)}
					onClick={onToggle}
				>
					<ChevronRight
						className={`size-4 transition-transform motion-reduce:transition-none ${expanded ? "rotate-90" : ""}`}
						aria-hidden
					/>
				</button>
			) : (
				<span className="size-8 shrink-0" />
			)}
			{isFolder ? (
				<Folder className="size-4 shrink-0 text-muted" aria-hidden />
			) : (
				<ImageIcon imageId={node.iconImageId} className="size-4" />
			)}
			<div className="ml-1.5 flex min-w-24 flex-1 items-center gap-2 sm:min-w-32">
				{isFolder ? (
					<>
						<span className="truncate font-medium">
							<Highlighted text={name} terms={terms} />
						</span>
						<span className="shrink-0 text-xs text-muted tabular-nums">
							<span aria-hidden>{linkCount}</span>
							<span className="sr-only">
								{t("favourites.linkCount", { count: linkCount })}
							</span>
						</span>
					</>
				) : (
					<a
						href={node.url}
						target="_blank"
						rel="noopener noreferrer"
						className="truncate hover:underline"
					>
						<Highlighted text={name} terms={terms} />
					</a>
				)}
				{node.tags?.map((tag) => (
					<button
						type="button"
						key={tag}
						className="hidden shrink-0 rounded-full bg-paper px-2 py-0.5 text-xs text-muted hover:text-ink sm:inline"
						aria-label={t("favourites.tagFilter.add", { tag })}
						onClick={() => actions.filterTag(tag)}
					>
						{tag}
					</button>
				))}
			</div>
			<div className="flex shrink-0 items-center">{actionButtons}</div>
		</li>
	);
}

function Highlighted({ text, terms }: { text: string; terms: string[] }) {
	if (terms.length === 0) return text;
	return highlight(text, terms).map((part, index) =>
		part.match ? (
			// biome-ignore lint/suspicious/noArrayIndexKey: parts have no other identity
			<mark key={index} className="rounded-sm bg-accent/25 text-inherit">
				{part.text}
			</mark>
		) : (
			part.text
		),
	);
}
