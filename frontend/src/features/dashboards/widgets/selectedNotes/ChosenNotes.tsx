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
	ArrowDown,
	ArrowUp,
	GripVertical,
	TriangleAlert,
	X,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { secondaryIconButton } from "#/components/buttonStyles";
import type { Note, NoteCategory } from "#/features/notes/api";
import { displayTitle } from "#/features/notes/document/text";
import { NoteChoiceLabel } from "../NoteChoiceLabel";
import { notePath } from "./selected";

/**
 * The chosen notes in their order, each with its category path, a drag
 * handle and buttons to move it up, down or remove it. IDs without a note in notes are shown as
 * notes that no longer exist.
 */
export function ChosenNotes({
	noteIds,
	notes,
	categories,
	onChange,
}: {
	noteIds: string[];
	notes: Note[];
	categories: NoteCategory[];
	onChange: (noteIds: string[]) => void;
}) {
	const sensors = useSensors(
		useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
		useSensor(TouchSensor, {
			activationConstraint: { delay: 200, tolerance: 5 },
		}),
		useSensor(KeyboardSensor, {
			coordinateGetter: sortableKeyboardCoordinates,
		}),
	);
	const byID = new Map(notes.map((n) => [n.id, n]));
	const move = (from: number, to: number) =>
		onChange(arrayMove(noteIds, from, to));

	function onDragEnd({ active, over }: DragEndEvent) {
		if (!over || active.id === over.id) return;
		move(noteIds.indexOf(String(active.id)), noteIds.indexOf(String(over.id)));
	}

	if (noteIds.length === 0) return null;
	return (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			onDragEnd={onDragEnd}
		>
			<SortableContext items={noteIds} strategy={verticalListSortingStrategy}>
				<ul className="flex flex-col gap-1">
					{noteIds.map((id, i) => (
						<ChosenNote
							key={id}
							id={id}
							note={byID.get(id)}
							categories={categories}
							first={i === 0}
							last={i === noteIds.length - 1}
							onUp={() => move(i, i - 1)}
							onDown={() => move(i, i + 1)}
							onRemove={() => onChange(noteIds.filter((n) => n !== id))}
						/>
					))}
				</ul>
			</SortableContext>
		</DndContext>
	);
}

function ChosenNote({
	id,
	note,
	categories,
	first,
	last,
	onUp,
	onDown,
	onRemove,
}: {
	id: string;
	note: Note | undefined;
	categories: NoteCategory[];
	first: boolean;
	last: boolean;
	onUp: () => void;
	onDown: () => void;
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
	} = useSortable({ id });
	const title = note ? displayTitle(note) : null;
	const name = note
		? (title ?? t("notes.untitled"))
		: t("dashboards.widgets.selectedNotes.gone");
	const path = note ? notePath(note, categories) : [];
	const button = `${secondaryIconButton} size-8 shrink-0 disabled:opacity-40`;

	return (
		<li
			ref={setNodeRef}
			style={{
				transform: transform
					? `translate3d(0, ${Math.round(transform.y)}px, 0)`
					: undefined,
				transition,
			}}
			className={`flex items-center gap-1 rounded-lg border border-line bg-surface py-1 pr-1 pl-0.5 ${isDragging ? "relative z-10 shadow-md" : ""}`}
		>
			<button
				type="button"
				ref={setActivatorNodeRef}
				{...attributes}
				{...listeners}
				aria-label={t("dashboards.widgets.selectedNotes.drag", { name })}
				className="flex size-8 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			>
				<GripVertical className="size-4" aria-hidden />
			</button>
			<span
				title={path.length > 0 ? `${name} · ${path.join(" › ")}` : name}
				className={`flex min-w-0 flex-1 items-center gap-2 text-sm ${note?.archived ? "opacity-70" : ""}`}
			>
				{note ? (
					<NoteChoiceLabel note={note} title={title} path={path} />
				) : (
					<>
						<TriangleAlert
							className="size-4 shrink-0 text-danger"
							aria-hidden
						/>
						<span className="min-w-0 flex-1 truncate text-muted italic">
							{name}
						</span>
					</>
				)}
			</span>
			<button
				type="button"
				onClick={onUp}
				disabled={first}
				aria-label={t("dashboards.widgets.selectedNotes.moveUp", { name })}
				className={button}
			>
				<ArrowUp className="size-4" aria-hidden />
			</button>
			<button
				type="button"
				onClick={onDown}
				disabled={last}
				aria-label={t("dashboards.widgets.selectedNotes.moveDown", { name })}
				className={button}
			>
				<ArrowDown className="size-4" aria-hidden />
			</button>
			<button
				type="button"
				onClick={onRemove}
				aria-label={t("dashboards.widgets.selectedNotes.remove", { name })}
				className={button}
			>
				<X className="size-4" aria-hidden />
			</button>
		</li>
	);
}
