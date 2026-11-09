import {
	DndContext,
	type DragMoveEvent,
	type DragStartEvent,
	PointerSensor,
	TouchSensor,
	useDraggable,
	useSensor,
	useSensors,
} from "@dnd-kit/core";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { $dfs, mergeRegister } from "@lexical/utils";
import {
	COMMAND_PRIORITY_NORMAL,
	KEY_ARROW_DOWN_COMMAND,
	KEY_ARROW_UP_COMMAND,
	type NodeKey,
	SKIP_SELECTION_FOCUS_TAG,
} from "lexical";
import { GripVertical } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { isTaskItem } from "./plugins";
import {
	$moveTaskBlock,
	$selectedTaskItem,
	$taskBlocks,
	dropGap,
} from "./taskMove";

/** The width of a handle, which sits in the gutter left of its task item. */
const handleSize = 24;

type Handle = { key: NodeKey; top: number; left: number };

type Box = { top: number; left: number; width: number; height: number };

type Drag = {
	key: NodeKey;
	gap: number;
	/** The block being moved. */
	source: Box;
	/** Where the block would land. */
	line: { top: number; left: number; width: number };
};

function TaskHandle({ handle }: { handle: Handle }) {
	const { t } = useTranslation();
	// Not focusable: Alt+Up and Alt+Down are the keyboard way to move a task.
	const { attributes, listeners, setNodeRef } = useDraggable({
		id: handle.key,
		attributes: { tabIndex: -1 },
	});
	return (
		<button
			type="button"
			ref={setNodeRef}
			{...attributes}
			{...listeners}
			aria-label={t("notes.editor.moveTask")}
			// Keeps the cursor in the editor and the focus where it is.
			onMouseDown={(e) => e.preventDefault()}
			style={{ top: handle.top, left: handle.left }}
			className="pointer-events-auto absolute flex size-6 cursor-grab touch-none items-center justify-center rounded-md text-muted hover:text-ink"
		>
			<GripVertical className="size-4" aria-hidden />
		</button>
	);
}

function startY(event: Event): number {
	if ("clientY" in event) return (event as MouseEvent).clientY;
	return (event as TouchEvent).touches[0]?.clientY ?? 0;
}

/**
 * The drag handles of the task items, in a layer over the content so that
 * they aren't part of the document, and moving a task with Alt+Up and
 * Alt+Down. Render it in a positioned element around the content.
 */
export function TaskHandlesPlugin() {
	const [editor] = useLexicalComposerContext();
	const overlay = useRef<HTMLDivElement>(null);
	const pointerY = useRef(0);
	const [editable, setEditable] = useState(() => editor.isEditable());
	const [handles, setHandles] = useState<Handle[]>([]);
	const [drag, setDrag] = useState<Drag | null>(null);
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(TouchSensor, {
			activationConstraint: { delay: 200, tolerance: 5 },
		}),
	);

	const measure = useCallback(() => {
		const origin = overlay.current?.getBoundingClientRect();
		if (!origin) return;
		const keys = editor.getEditorState().read(() =>
			$dfs()
				.filter(({ node }) => isTaskItem(node))
				.map(({ node }) => node.getKey()),
		);
		const next = keys.flatMap((key) => {
			const rect = editor.getElementByKey(key)?.getBoundingClientRect();
			if (!rect) return [];
			return [
				{
					key,
					// Centres the handle on the checkbox of the first line.
					top: rect.top - origin.top - 1,
					left: rect.left - origin.left - handleSize,
				},
			];
		});
		setHandles((current) =>
			current.length === next.length &&
			current.every(
				(h, i) =>
					h.key === next[i].key &&
					h.top === next[i].top &&
					h.left === next[i].left,
			)
				? current
				: next,
		);
	}, [editor]);

	useEffect(() => {
		let observer: ResizeObserver | undefined;
		return mergeRegister(
			editor.registerEditableListener((value) => {
				setEditable(value);
				if (!value) setDrag(null);
			}),
			editor.registerUpdateListener(({ dirtyElements, dirtyLeaves }) => {
				if (dirtyElements.size > 0 || dirtyLeaves.size > 0) measure();
			}),
			editor.registerRootListener((root) => {
				observer?.disconnect();
				observer = undefined;
				if (!root) return;
				measure();
				if (typeof ResizeObserver === "undefined") return;
				observer = new ResizeObserver(measure);
				observer.observe(root);
			}),
			() => observer?.disconnect(),
		);
	}, [editor, measure]);

	useEffect(() => {
		const move = (direction: -1 | 1) => (event: KeyboardEvent) => {
			if (!event.altKey || event.shiftKey || event.ctrlKey || event.metaKey)
				return false;
			if (!editor.isEditable()) return false;
			const key = $selectedTaskItem();
			if (!key) return false;
			event.preventDefault();
			const list = $taskBlocks(key);
			if (list) $moveTaskBlock(key, list.from + (direction < 0 ? -1 : 2));
			return true;
		};
		// Above the arrow key handling of Lexical's task lists.
		return mergeRegister(
			editor.registerCommand(
				KEY_ARROW_UP_COMMAND,
				move(-1),
				COMMAND_PRIORITY_NORMAL,
			),
			editor.registerCommand(
				KEY_ARROW_DOWN_COMMAND,
				move(1),
				COMMAND_PRIORITY_NORMAL,
			),
		);
	}, [editor]);

	const dragged = drag !== null;
	useEffect(() => {
		if (!dragged) return;
		// The delta of a drag is adjusted for scrolling, so it can't tell where
		// the pointer is on the screen.
		const track = (event: PointerEvent) => {
			pointerY.current = event.clientY;
		};
		// Escape only cancels the drag: it would also end inline editing or close
		// the dialog around the editor, so it stops here and ends the gesture
		// the way a cancelled pointer does.
		const cancel = (event: KeyboardEvent) => {
			if (event.key !== "Escape") return;
			event.preventDefault();
			event.stopPropagation();
			document.dispatchEvent(new Event("pointercancel"));
			document.dispatchEvent(new Event("touchcancel"));
		};
		window.addEventListener("pointermove", track, true);
		window.addEventListener("keydown", cancel, true);
		return () => {
			window.removeEventListener("pointermove", track, true);
			window.removeEventListener("keydown", cancel, true);
		};
	}, [dragged]);

	/** Where the task item key would land with the pointer where it is now. */
	const place = useCallback(
		(key: NodeKey): Drag | null => {
			const origin = overlay.current?.getBoundingClientRect();
			const list = editor.getEditorState().read(() => $taskBlocks(key));
			if (!origin || !list) return null;
			const rects = list.blocks.map((block) => {
				const first = editor.getElementByKey(block[0]);
				const last = editor.getElementByKey(block[block.length - 1]);
				const top = first?.getBoundingClientRect();
				return {
					top: top?.top ?? 0,
					left: top?.left ?? 0,
					right: top?.right ?? 0,
					bottom: last?.getBoundingClientRect().bottom ?? 0,
				};
			});
			const gap = dropGap(pointerY.current, rects, list.firstGap);
			const from = rects[list.from];
			return {
				key,
				gap,
				source: {
					top: from.top - origin.top,
					left: from.left - origin.left,
					width: from.right - from.left,
					height: from.bottom - from.top,
				},
				line: {
					top:
						(gap < rects.length ? rects[gap].top : rects[gap - 1].bottom) -
						origin.top -
						1,
					left: from.left - origin.left,
					width: from.right - from.left,
				},
			};
		},
		[editor],
	);

	const onDragStart = (event: DragStartEvent) => {
		pointerY.current = startY(event.activatorEvent);
		setDrag(place(String(event.active.id)));
	};
	const onDragMove = (event: DragMoveEvent) => {
		setDrag(place(String(event.active.id)));
	};
	const onDragEnd = (event: DragMoveEvent) => {
		const key = String(event.active.id);
		const target = place(key);
		setDrag(null);
		if (!target) return;
		editor.update(
			() => {
				if (editor.isEditable()) $moveTaskBlock(key, target.gap);
			},
			{ tag: SKIP_SELECTION_FOCUS_TAG },
		);
	};

	return (
		<div ref={overlay} className="pointer-events-none absolute inset-0">
			{/* Only with handles: it adds a status region to the page. */}
			{editable && (handles.length > 0 || drag) && (
				<DndContext
					sensors={sensors}
					onDragStart={onDragStart}
					onDragMove={onDragMove}
					onDragEnd={onDragEnd}
					onDragCancel={() => setDrag(null)}
				>
					{handles.map((handle) => (
						<TaskHandle key={handle.key} handle={handle} />
					))}
				</DndContext>
			)}
			{drag && (
				<>
					<div
						style={drag.source}
						className="absolute rounded-md bg-accent/10"
					/>
					<div
						style={drag.line}
						className="absolute h-0.5 rounded-full bg-accent"
					/>
				</>
			)}
		</div>
	);
}
