import {
	type Announcements,
	type CollisionDetection,
	closestCenter,
	DndContext,
	DragOverlay,
	PointerSensor,
	pointerWithin,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useBlocker } from "@tanstack/react-router";
import { Check, GripVertical, X } from "lucide-react";
import {
	type KeyboardEvent,
	useEffect,
	useReducer,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/features/auth/LoginPage";
import {
	type DashboardViewData,
	dashboardViewQueryKey,
	isLayoutConflict,
	type Section,
	saveLayout,
} from "../api";
import { DashboardPage } from "../DashboardView";
import { DeleteSectionDialog, DiscardDraftDialog } from "./ConfirmDialogs";
import {
	type DropTarget,
	type Editor,
	EditorContext,
	parentOfContainer,
} from "./editor";
import {
	canMove,
	currentSlot,
	findSection,
	isSameTree,
	isWithin,
	keyboardSlots,
	layoutReducer,
	parentOf,
	type Slot,
} from "./layoutTree";
import { SectionDialog, type SectionDialogTarget } from "./SectionDialog";
import { DashboardGrid } from "./SectionGrid";

type KeyboardDrag = {
	id: string;
	slots: Slot[];
	index: number;
	snapshot: Section[];
};

/**
 * Edit mode: a draft of the layout that is saved in one request. `onExit`
 * leaves edit mode; it is only called once nothing would be lost.
 */
export function LayoutEditor({
	view,
	columns,
	onExit,
}: {
	view: DashboardViewData;
	columns: number;
	onExit: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	// The version the draft is based on; a newer saved one means a conflict.
	const [baseVersion, setBaseVersion] = useState(view.layoutVersion);
	const [saved, setSaved] = useState(view.sections);
	const [draft, dispatch] = useReducer(layoutReducer, view.sections);
	const dirty = !isSameTree(draft, saved);

	const [dialog, setDialog] = useState<
		(SectionDialogTarget & { parentId: string | null }) | null
	>(null);
	const [deleting, setDeleting] = useState<Section | null>(null);
	const [discarding, setDiscarding] = useState(false);
	const [tooDeep, setTooDeep] = useState(false);
	const [announcement, setAnnouncement] = useState("");

	// Leaving on purpose (saved or discarded) must not be blocked.
	const leaving = useRef(false);
	const dirtyRef = useRef(dirty);
	dirtyRef.current = dirty;
	const blocker = useBlocker({
		shouldBlockFn: () => !leaving.current && dirtyRef.current,
		enableBeforeUnload: () => !leaving.current && dirtyRef.current,
		withResolver: true,
	});

	const save = useMutation({
		mutationFn: () => saveLayout(view.id ?? "", baseVersion, draft),
		onSuccess: (layout) => {
			queryClient.setQueriesData<DashboardViewData>(
				{ queryKey: dashboardViewQueryKey },
				(old) =>
					old && old.id === view.id
						? {
								...old,
								sections: layout.sections,
								layoutVersion: layout.version,
							}
						: old,
			);
			setSaved(layout.sections);
			leave();
		},
	});

	/** After a conflict: drops the draft and starts over from the current layout. */
	async function reload() {
		await queryClient.invalidateQueries({ queryKey: dashboardViewQueryKey });
		const fresh = queryClient
			.getQueriesData<DashboardViewData>({ queryKey: dashboardViewQueryKey })
			.map(([, data]) => data)
			.find((data) => data?.id === view.id);
		if (!fresh) return;
		setBaseVersion(fresh.layoutVersion);
		setSaved(fresh.sections);
		dispatch({ type: "reset", sections: fresh.sections });
		save.reset();
	}

	function leave() {
		leaving.current = true;
		onExit();
	}

	// The "nested too deep" message disappears after a while.
	useEffect(() => {
		if (!tooDeep) return;
		const timer = setTimeout(() => setTooDeep(false), 6000);
		return () => clearTimeout(timer);
	}, [tooDeep]);

	function blockTooDeep() {
		setTooDeep(true);
		setAnnouncement(t("dashboards.editor.tooDeep"));
	}

	const titleOf = (id: string) =>
		findSection(draft, id)?.title ?? t("dashboards.editor.untitled");

	// --- Keyboard moves: pick up, step through slots, drop. ---
	const [keyboard, setKeyboard] = useState<KeyboardDrag | null>(null);
	const [refocus, setRefocus] = useState<string | null>(null);
	const handles = useRef(new Map<string, HTMLButtonElement>());

	// Moving a section to another parent remounts it; keep focus on its handle.
	useEffect(() => {
		if (!refocus) return;
		handles.current.get(refocus)?.focus();
		setRefocus(null);
	}, [refocus]);

	function describeSlot(slot: Slot, valid: boolean) {
		const parent =
			slot.parentId === null
				? t("dashboards.dnd.dashboard")
				: titleOf(slot.parentId);
		return t(valid ? "dashboards.dnd.position" : "dashboards.dnd.tooDeep", {
			position: slot.index + 1,
			parent,
		});
	}

	function onHandleKeyDown(event: KeyboardEvent, section: Section) {
		if (keyboard?.id !== section.id) {
			if (event.key !== " " && event.key !== "Enter") return;
			event.preventDefault();
			const slots = keyboardSlots(draft, section.id);
			const current = currentSlot(draft, section.id);
			const index = Math.max(
				0,
				slots.findIndex(
					(s) => s.parentId === current?.parentId && s.index === current?.index,
				),
			);
			setKeyboard({ id: section.id, slots, index, snapshot: draft });
			setAnnouncement(
				t("dashboards.dnd.pickedUp", { title: titleOf(section.id) }),
			);
			return;
		}
		const { id, slots, snapshot } = keyboard;
		switch (event.key) {
			case "ArrowUp":
			case "ArrowLeft":
			case "ArrowDown":
			case "ArrowRight": {
				event.preventDefault();
				const step =
					event.key === "ArrowUp" || event.key === "ArrowLeft" ? -1 : 1;
				const index = Math.min(
					Math.max(keyboard.index + step, 0),
					slots.length - 1,
				);
				const slot = slots[index];
				const valid = canMove(snapshot, id, slot.parentId) === null;
				if (valid) {
					dispatch({
						type: "reset",
						sections: layoutReducer(snapshot, { type: "move", id, ...slot }),
					});
					setRefocus(id);
				}
				setKeyboard({ ...keyboard, index });
				setAnnouncement(describeSlot(slot, valid));
				break;
			}
			case " ":
			case "Enter": {
				event.preventDefault();
				const slot = slots[keyboard.index];
				setKeyboard(null);
				if (canMove(snapshot, id, slot.parentId) !== null) {
					dispatch({ type: "reset", sections: snapshot });
					setRefocus(id);
					blockTooDeep();
					return;
				}
				setAnnouncement(t("dashboards.dnd.dropped", { title: titleOf(id) }));
				break;
			}
			case "Escape":
				event.preventDefault();
				dispatch({ type: "reset", sections: snapshot });
				setKeyboard(null);
				setRefocus(id);
				setAnnouncement(t("dashboards.dnd.cancelled", { title: titleOf(id) }));
				break;
		}
	}

	// --- Pointer moves: dnd-kit. Inside one grid the sortable strategy shows
	// the new order; a move to another grid only highlights the target until
	// the drop, so the layout doesn't reflow under the pointer.
	const [activeId, setActiveId] = useState<string | null>(null);
	const [dropTarget, setDropTarget] = useState<DropTarget | null>(null);
	const dropTargetRef = useRef<DropTarget | null>(null);
	const showDropTarget = (target: DropTarget | null) => {
		dropTargetRef.current = target;
		setDropTarget(target);
	};
	const draftRef = useRef(draft);
	draftRef.current = draft;
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
	);

	// Drop targets are the dragged section's siblings and every grid outside
	// it. Of the targets under the pointer, the smallest (innermost) wins.
	const collisionDetection: CollisionDetection = (args) => {
		const active = String(args.active.id);
		const droppableContainers = args.droppableContainers.filter((c) => {
			const id = String(c.id);
			const container = parentOfContainer(id);
			// A section: the dragged one itself or any outside of it.
			if (container === undefined) {
				return id === active || !isWithin(draftRef.current, id, active);
			}
			// A grid: the top level, or the children of a section outside of it.
			return (
				container === null || !isWithin(draftRef.current, container, active)
			);
		});
		const hits = pointerWithin({ ...args, droppableContainers });
		if (hits.length === 0) {
			return closestCenter({ ...args, droppableContainers });
		}
		const area = (id: string | number) => {
			const rect = args.droppableRects.get(id);
			return rect ? rect.width * rect.height : Number.POSITIVE_INFINITY;
		};
		return hits.sort((a, b) => area(a.id) - area(b.id));
	};

	/** The slot a drop on overId means. */
	function targetOf(tree: Section[], active: string, overId: string) {
		const container = parentOfContainer(overId);
		if (container !== undefined) {
			const list =
				container === null
					? tree
					: (findSection(tree, container)?.children ?? []);
			return {
				parentId: container,
				index: list.filter((s) => s.id !== active).length,
			};
		}
		const parentId = parentOf(tree, overId);
		if (parentId === undefined) return null;
		const list =
			parentId === null ? tree : (findSection(tree, parentId)?.children ?? []);
		return { parentId, index: list.findIndex((s) => s.id === overId) };
	}

	const announcements: Announcements = {
		onDragStart: ({ active }) =>
			t("dashboards.dnd.pickedUp", { title: titleOf(String(active.id)) }),
		onDragOver: () => undefined,
		onDragMove: () => undefined,
		onDragEnd: ({ active }) =>
			dropTargetRef.current?.blocked
				? t("dashboards.editor.tooDeep")
				: t("dashboards.dnd.dropped", { title: titleOf(String(active.id)) }),
		onDragCancel: ({ active }) =>
			t("dashboards.dnd.cancelled", { title: titleOf(String(active.id)) }),
	};

	const editor: Editor = {
		onAdd: (parentId) => setDialog({ section: null, parentId }),
		onEdit: (id) => {
			const section = findSection(draft, id);
			if (section)
				setDialog({ section, parentId: parentOf(draft, id) ?? null });
		},
		pickedId: keyboard?.id ?? null,
		dropTarget,
		onHandleKeyDown,
		// Focus moving elsewhere cancels a keyboard move. A handle that loses
		// focus because its section moved to another grid has no relatedTarget.
		onHandleBlur: (event, section) => {
			if (keyboard?.id !== section.id || event.relatedTarget === null) return;
			dispatch({ type: "reset", sections: keyboard.snapshot });
			setKeyboard(null);
		},
		registerHandle: (id, el) => {
			if (el) handles.current.set(id, el);
			else handles.current.delete(id);
		},
	};

	function cancel() {
		if (dirty) setDiscarding(true);
		else leave();
	}

	return (
		<DashboardPage
			title={view.dashboard.title}
			actions={
				<>
					<button
						type="button"
						onClick={() => save.mutate()}
						disabled={save.isPending}
						className={`${primaryButton} gap-2 px-4 py-2 text-sm`}
					>
						<Check className="size-4" aria-hidden />
						{save.isPending
							? t("dashboards.editor.saving")
							: t("dashboards.editor.saveAndExit")}
					</button>
					<button
						type="button"
						onClick={cancel}
						aria-label={t("dashboards.editor.cancel")}
						title={t("dashboards.editor.cancel")}
						className={`${secondaryButton} p-2`}
					>
						<X className="size-4" aria-hidden />
					</button>
				</>
			}
		>
			{save.isError &&
				(isLayoutConflict(save.error) ? (
					<div
						role="alert"
						className="mb-4 flex flex-wrap items-center gap-3 rounded-lg border border-line bg-surface px-4 py-3 text-sm"
					>
						<p className="min-w-0 flex-1">{t("dashboards.editor.conflict")}</p>
						<button
							type="button"
							className={`${secondaryButton} px-4 py-1.5 text-sm`}
							onClick={() => void reload()}
						>
							{t("dashboards.editor.reload")}
						</button>
					</div>
				) : (
					<p role="alert" className="mb-4 text-sm text-red-700">
						{t("dashboards.editor.saveFailed")}
					</p>
				))}
			{tooDeep && (
				<output className="mb-4 block rounded-lg border border-line bg-surface px-4 py-3 text-sm">
					{t("dashboards.editor.tooDeep")}
				</output>
			)}
			<EditorContext.Provider value={editor}>
				<DndContext
					sensors={sensors}
					collisionDetection={collisionDetection}
					accessibility={{
						announcements,
						screenReaderInstructions: {
							draggable: t("dashboards.dnd.instructions"),
						},
					}}
					onDragStart={({ active }) => {
						setKeyboard(null);
						setActiveId(String(active.id));
					}}
					onDragOver={({ active, over }) => {
						const id = String(active.id);
						const overId = over ? String(over.id) : null;
						const target = overId && targetOf(draft, id, overId);
						// Moves inside the same grid are left to the sortable strategy.
						if (!target || target.parentId === parentOf(draft, id)) {
							showDropTarget(null);
							return;
						}
						showDropTarget({
							...target,
							overId: parentOfContainer(overId) === undefined ? overId : null,
							blocked: canMove(draft, id, target.parentId) !== null,
						});
					}}
					onDragEnd={({ active, over }) => {
						const id = String(active.id);
						const target = dropTargetRef.current;
						if (target?.blocked) {
							setTooDeep(true);
						} else if (target) {
							dispatch({ type: "move", id, ...target });
						} else if (over && String(over.id) !== id) {
							const reorder = targetOf(draft, id, String(over.id));
							if (reorder && reorder.parentId === parentOf(draft, id)) {
								dispatch({ type: "move", id, ...reorder });
							}
						}
						showDropTarget(null);
						setActiveId(null);
					}}
					onDragCancel={() => {
						showDropTarget(null);
						setActiveId(null);
					}}
				>
					<DashboardGrid sections={draft} columns={columns} />
					<DragOverlay dropAnimation={null}>
						{activeId && (
							<div className="inline-flex max-w-80 items-center gap-2 rounded-lg border border-line bg-surface px-3 py-2 shadow-lg">
								<GripVertical
									className="size-4 shrink-0 text-muted"
									aria-hidden
								/>
								<span className="truncate">{titleOf(activeId)}</span>
							</div>
						)}
					</DragOverlay>
				</DndContext>
			</EditorContext.Provider>
			<output aria-live="assertive" className="sr-only">
				{announcement}
			</output>
			<SectionDialog
				target={dialog}
				desktopColumns={view.columns.desktopColumns}
				onClose={() => setDialog(null)}
				onApply={(settings) => {
					if (dialog?.section) {
						dispatch({ type: "update", id: dialog.section.id, settings });
					} else if (dialog) {
						dispatch({
							type: "add",
							parentId: dialog.parentId,
							section: { id: crypto.randomUUID(), children: [], ...settings },
						});
					}
					setDialog(null);
				}}
				onDelete={(section) => setDeleting(section)}
			/>
			<DeleteSectionDialog
				section={deleting}
				onClose={() => setDeleting(null)}
				onConfirm={(section) => {
					dispatch({ type: "remove", id: section.id });
					setDeleting(null);
					setDialog(null);
				}}
			/>
			<DiscardDraftDialog
				open={discarding || blocker.status === "blocked"}
				onClose={() => {
					setDiscarding(false);
					if (blocker.status === "blocked") blocker.reset();
				}}
				onConfirm={() => {
					setDiscarding(false);
					leaving.current = true;
					if (blocker.status === "blocked") blocker.proceed();
					else onExit();
				}}
			/>
		</DashboardPage>
	);
}
