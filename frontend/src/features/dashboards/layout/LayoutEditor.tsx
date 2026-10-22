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
import {
	primaryButton,
	secondaryButton,
	secondaryIconButton,
} from "#/components/buttonStyles";
import {
	type ContentWidget,
	type DashboardViewData,
	dashboardViewQueryKey,
	isLayoutConflict,
	isSection,
	saveLayout,
	type Widget,
} from "../api";
import { DashboardPage } from "../DashboardView";
import { widgetLabel } from "../widgets/registry";
import { DeleteWidgetDialog, DiscardDraftDialog } from "./ConfirmDialogs";
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
	findWidget,
	isSameTree,
	isWithin,
	keyboardSlots,
	layoutReducer,
	listOf,
	type MoveProblem,
	parentOf,
	type Slot,
} from "./layoutTree";
import { SectionDialog, type SectionDialogTarget } from "./SectionDialog";
import { DashboardGrid } from "./SectionGrid";
import { WidgetDialog, WidgetPicker } from "./WidgetDialogs";

type KeyboardDrag = {
	id: string;
	slots: Slot[];
	index: number;
	snapshot: Widget[];
};

/** The settings dialog of a content widget, new or existing. */
type WidgetDialogTarget = {
	widget: ContentWidget;
	isNew: boolean;
	parentId: string;
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
	// The section whose "Add widget" opened the picker.
	const [picking, setPicking] = useState<string | null>(null);
	const [widgetDialog, setWidgetDialog] = useState<WidgetDialogTarget | null>(
		null,
	);
	const [deleting, setDeleting] = useState<Widget | null>(null);
	const [discarding, setDiscarding] = useState(false);
	// Why the last move was refused; shown for a while.
	const [refused, setRefused] = useState<MoveProblem | null>(null);
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

	// The "move refused" message disappears after a while.
	useEffect(() => {
		if (!refused) return;
		const timer = setTimeout(() => setRefused(null), 6000);
		return () => clearTimeout(timer);
	}, [refused]);

	const refusedText = (problem: MoveProblem | null) =>
		problem === "topLevel"
			? t("dashboards.editor.topLevel")
			: t("dashboards.editor.tooDeep");

	function refuse(problem: MoveProblem | null) {
		setRefused(problem ?? "tooDeep");
		setAnnouncement(refusedText(problem));
	}

	const titleOf = (id: string) => {
		const widget = findWidget(draft, id);
		return widget ? widgetLabel(widget, t) : t("dashboards.editor.untitled");
	};

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

	function onHandleKeyDown(event: KeyboardEvent, widget: Widget) {
		if (keyboard?.id !== widget.id) {
			if (event.key !== " " && event.key !== "Enter") return;
			event.preventDefault();
			const slots = keyboardSlots(draft, widget.id);
			const current = currentSlot(draft, widget.id);
			const index = Math.max(
				0,
				slots.findIndex(
					(s) => s.parentId === current?.parentId && s.index === current?.index,
				),
			);
			setKeyboard({ id: widget.id, slots, index, snapshot: draft });
			setAnnouncement(
				t("dashboards.dnd.pickedUp", { title: titleOf(widget.id) }),
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
				const problem = canMove(snapshot, id, slot.parentId);
				if (problem !== null) {
					dispatch({ type: "reset", sections: snapshot });
					setRefocus(id);
					refuse(problem);
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

	// Drop targets are the dragged widget's siblings and every grid outside
	// it. Of the targets under the pointer, the smallest (innermost) wins.
	const collisionDetection: CollisionDetection = (args) => {
		const active = String(args.active.id);
		const droppableContainers = args.droppableContainers.filter((c) => {
			const id = String(c.id);
			const container = parentOfContainer(id);
			// A widget: the dragged one itself or any outside of it.
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
	function targetOf(tree: Widget[], active: string, overId: string) {
		const container = parentOfContainer(overId);
		if (container !== undefined) {
			return {
				parentId: container,
				index: listOf(tree, container).filter((w) => w.id !== active).length,
			};
		}
		const parentId = parentOf(tree, overId);
		if (parentId === undefined) return null;
		const list = listOf(tree, parentId);
		return { parentId, index: list.findIndex((w) => w.id === overId) };
	}

	const announcements: Announcements = {
		onDragStart: ({ active }) =>
			t("dashboards.dnd.pickedUp", { title: titleOf(String(active.id)) }),
		onDragOver: () => undefined,
		onDragMove: () => undefined,
		onDragEnd: ({ active }) => {
			const target = dropTargetRef.current;
			return target?.blocked
				? refusedText(
						canMove(draftRef.current, String(active.id), target.parentId),
					)
				: t("dashboards.dnd.dropped", { title: titleOf(String(active.id)) });
		},
		onDragCancel: ({ active }) =>
			t("dashboards.dnd.cancelled", { title: titleOf(String(active.id)) }),
	};

	const editor: Editor = {
		onAdd: (parentId) => setDialog({ section: null, parentId }),
		onAddWidget: (parentId) => setPicking(parentId),
		onEdit: (id) => {
			const widget = findWidget(draft, id);
			const parentId = parentOf(draft, id) ?? null;
			if (!widget) return;
			if (isSection(widget)) setDialog({ section: widget, parentId });
			else if (parentId !== null)
				setWidgetDialog({ widget, isNew: false, parentId });
		},
		pickedId: keyboard?.id ?? null,
		dropTarget,
		onHandleKeyDown,
		// Focus moving elsewhere cancels a keyboard move. A handle that loses
		// focus because its section moved to another grid has no relatedTarget.
		onHandleBlur: (event, widget) => {
			if (keyboard?.id !== widget.id || event.relatedTarget === null) return;
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
						className={primaryButton}
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
						className={secondaryIconButton}
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
							className={secondaryButton}
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
			{refused && (
				<output className="mb-4 block rounded-lg border border-line bg-surface px-4 py-3 text-sm">
					{refusedText(refused)}
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
							setRefused(canMove(draft, id, target.parentId) ?? "tooDeep");
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
						dispatch({
							type: "update",
							id: dialog.section.id,
							widget: { ...dialog.section, ...settings },
						});
					} else if (dialog) {
						dispatch({
							type: "add",
							parentId: dialog.parentId,
							widget: {
								id: crypto.randomUUID(),
								kind: "section",
								children: [],
								...settings,
							},
						});
					}
					setDialog(null);
				}}
				onDelete={(section) => setDeleting(section)}
			/>
			<WidgetPicker
				open={picking !== null}
				onClose={() => setPicking(null)}
				onPick={(kind) => {
					if (picking === null) return;
					// A new widget is as wide as its section.
					const width = Math.min(
						findSection(draft, picking)?.width ?? view.columns.desktopColumns,
						view.columns.desktopColumns,
					);
					setWidgetDialog({
						widget: {
							id: crypto.randomUUID(),
							kind: kind.kind,
							width,
							height: "auto",
							config: kind.defaultConfig(),
						},
						isNew: true,
						parentId: picking,
					});
					setPicking(null);
				}}
			/>
			<WidgetDialog
				target={widgetDialog}
				desktopColumns={view.columns.desktopColumns}
				onClose={() => setWidgetDialog(null)}
				onApply={(widget) => {
					if (widgetDialog?.isNew) {
						dispatch({ type: "add", parentId: widgetDialog.parentId, widget });
					} else if (widgetDialog) {
						dispatch({ type: "update", id: widget.id, widget });
					}
					setWidgetDialog(null);
				}}
				onDelete={(widget) => setDeleting(widget)}
			/>
			<DeleteWidgetDialog
				widget={deleting}
				onClose={() => setDeleting(null)}
				onConfirm={(widget) => {
					dispatch({ type: "remove", id: widget.id });
					setDeleting(null);
					setDialog(null);
					setWidgetDialog(null);
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
