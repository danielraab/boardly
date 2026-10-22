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
	ChevronRight,
	Folder,
	FolderPlus,
	GripVertical,
	Pencil,
	Plus,
	Star,
	Trash2,
} from "lucide-react";
import {
	type KeyboardEvent,
	type ReactNode,
	useEffect,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { iconButton, secondaryButton } from "#/components/buttonStyles";
import { ImageIcon } from "#/components/ImageIcon";
import { useMe } from "#/features/auth/session";
import {
	type FavouriteNode,
	favouritesQueryKey,
	favouritesQueryOptions,
	moveFavourite,
} from "./api";
import { DeleteDialog, type DeleteTarget } from "./DeleteDialog";
import { NodeDialog, type NodeDialogTarget } from "./NodeDialog";
import {
	applyMove,
	currentSlot,
	findNode,
	findParent,
	flatten,
	type Indicator,
	indicatorFor,
	isInSubtree,
	keyboardSlots,
	label,
	pointerSlot,
	ROOT_END,
	type Slot,
	sameSlot,
} from "./tree";

const INDENT_REM = 1.5;

type Actions = {
	add: (kind: "folder" | "link", parent: FavouriteNode) => void;
	edit: (node: FavouriteNode) => void;
	remove: (node: FavouriteNode) => void;
};

export function FavouritesPage() {
	const { t } = useTranslation();
	const { data: me } = useMe();
	const tree = useQuery({ ...favouritesQueryOptions, enabled: !!me });
	const [editing, setEditing] = useState<NodeDialogTarget | null>(null);
	const [deleting, setDeleting] = useState<DeleteTarget | null>(null);

	// The session can end while the page is open (401 from any request).
	if (me === null) return <Navigate to="/login" />;
	if (!me) return null;

	const root = tree.data;
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
			<div className="mt-8">
				{tree.isPending ? (
					<p className="text-muted">{t("common.loading")}</p>
				) : tree.isError ? (
					<p role="alert" className="text-sm text-danger">
						{t("common.genericError")}
					</p>
				) : (tree.data.children ?? []).length === 0 ? (
					<div className="rounded-xl border border-dashed border-line bg-surface px-6 py-12 text-center">
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
				) : (
					<TreeView
						root={tree.data}
						actions={actions}
						folderLabel={folderLabel}
					/>
				)}
			</div>
			<NodeDialog target={editing} onClose={() => setEditing(null)} />
			<DeleteDialog target={deleting} onClose={() => setDeleting(null)} />
		</section>
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
	actions,
	folderLabel,
}: {
	root: FavouriteNode;
	actions: Actions;
	folderLabel: (folder: FavouriteNode) => string;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(new Set());
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
			<ul className="rounded-xl border border-line bg-surface py-1">
				{flatten(root, collapsed).map(({ node, depth }) => (
					<TreeRow
						key={node.id}
						node={node}
						depth={depth}
						indicator={indicator?.rowId === node.id ? indicator : null}
						dimmed={!!draggedId && isInSubtree(root, draggedId, node.id)}
						picked={keyboard?.id === node.id}
						expanded={!collapsed.has(node.id)}
						onToggle={() =>
							setCollapsed((prev) => {
								const next = new Set(prev);
								if (!next.delete(node.id)) next.add(node.id);
								return next;
							})
						}
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
	actions: Actions;
}) {
	const { t } = useTranslation();
	const { setNodeRef: setDropRef } = useDroppable({ id: node.id });
	const {
		setNodeRef: setDragRef,
		setActivatorNodeRef,
		attributes,
		listeners,
	} = useDraggable({ id: node.id });
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
			}}
			data-testid={`row-${node.id}`}
			className={`relative mx-1 flex items-center gap-1 rounded-lg py-1 pr-1 ${indicator?.edge === "inside" ? "bg-accent/10 ring-2 ring-accent" : ""} ${dimmed ? "opacity-40" : ""}`}
			style={{ paddingLeft: `${depth * INDENT_REM + 0.25}rem` }}
		>
			{indicator && indicator.edge !== "inside" && (
				<span
					aria-hidden
					className={`pointer-events-none absolute right-2 h-0.5 rounded-full bg-accent ${indicator.edge === "top" ? "-top-px" : "-bottom-px"}`}
					style={{ left: `${indicator.depth * INDENT_REM + 2.25}rem` }}
				/>
			)}
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
			<div className="ml-1.5 flex min-w-0 flex-1 items-center gap-2">
				{isFolder ? (
					<span className="truncate font-medium">{name}</span>
				) : (
					<a
						href={node.url}
						target="_blank"
						rel="noopener noreferrer"
						className="truncate hover:underline"
					>
						{name}
					</a>
				)}
				{node.tags?.map((tag) => (
					<span
						key={tag}
						className="hidden shrink-0 rounded-full bg-paper px-2 py-0.5 text-xs text-muted sm:inline"
					>
						{tag}
					</span>
				))}
			</div>
			<div className="flex shrink-0 items-center">{actionButtons}</div>
		</li>
	);
}
