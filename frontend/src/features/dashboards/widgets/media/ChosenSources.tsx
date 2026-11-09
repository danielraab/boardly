import { Combobox } from "@base-ui/react/combobox";
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
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { secondaryIconButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import { ChosenIcon } from "#/components/icons/ChosenIcon";
import { menuItem, menuPopup } from "#/components/menuStyles";
import type { MediaSource } from "#/features/media/api";
import { sourceFallbackIcon } from "#/features/media/sourceIcon";
import { iconProps } from "#/features/notes/noteIcon";

/** A source that can be chosen: a station, or a feed for the feed widget. */
type Choice = Pick<MediaSource, "id" | "title" | "icon"> &
	Partial<Pick<MediaSource, "kind">>;

/** Where the chooser's texts are: each kind that uses it has the same keys. */
type Labels = "dashboards.widgets.media" | "dashboards.widgets.feed";

/** The most sources a media widget can show. */
export const maxMediaSources = 50;

const fold = (s: string) =>
	s
		.normalize("NFD")
		.replace(/\p{Diacritic}/gu, "")
		.toLowerCase();

/**
 * The chosen sources in their order, each with a drag handle and buttons to
 * move it up, down or remove it. IDs without a source are shown as stations
 * that no longer exist.
 */
export function ChosenSources({
	sourceIds,
	sources,
	onChange,
	labels = "dashboards.widgets.media",
}: {
	sourceIds: string[];
	sources: Choice[];
	onChange: (sourceIds: string[]) => void;
	labels?: Labels;
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
	const byID = new Map(sources.map((s) => [s.id, s]));
	const move = (from: number, to: number) =>
		onChange(arrayMove(sourceIds, from, to));

	function onDragEnd({ active, over }: DragEndEvent) {
		if (!over || active.id === over.id) return;
		move(
			sourceIds.indexOf(String(active.id)),
			sourceIds.indexOf(String(over.id)),
		);
	}

	if (sourceIds.length === 0) return null;
	return (
		<DndContext
			sensors={sensors}
			collisionDetection={closestCenter}
			onDragEnd={onDragEnd}
		>
			<SortableContext items={sourceIds} strategy={verticalListSortingStrategy}>
				<ul className="flex flex-col gap-1">
					{sourceIds.map((id, i) => (
						<ChosenSource
							key={id}
							id={id}
							source={byID.get(id)}
							labels={labels}
							first={i === 0}
							last={i === sourceIds.length - 1}
							onUp={() => move(i, i - 1)}
							onDown={() => move(i, i + 1)}
							onRemove={() => onChange(sourceIds.filter((s) => s !== id))}
						/>
					))}
				</ul>
			</SortableContext>
		</DndContext>
	);
}

function ChosenSource({
	id,
	source,
	labels,
	first,
	last,
	onUp,
	onDown,
	onRemove,
}: {
	id: string;
	source: Choice | undefined;
	labels: Labels;
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
	const name = source?.title ?? t(`${labels}.gone`);
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
				aria-label={t(`${labels}.drag`, { name })}
				className="flex size-8 shrink-0 cursor-grab touch-none items-center justify-center rounded-md text-muted hover:text-ink focus-visible:ring-2 focus-visible:ring-accent focus-visible:outline-none"
			>
				<GripVertical className="size-4" aria-hidden />
			</button>
			<span className="flex min-w-0 flex-1 items-center gap-2 text-sm">
				{source ? (
					<>
						<ChosenIcon
							{...iconProps(source.icon)}
							fallback={sourceFallbackIcon(source)}
							className="size-4"
						/>
						<span className="min-w-0 flex-1 truncate">{name}</span>
					</>
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
				aria-label={t(`${labels}.moveUp`, { name })}
				className={button}
			>
				<ArrowUp className="size-4" aria-hidden />
			</button>
			<button
				type="button"
				onClick={onDown}
				disabled={last}
				aria-label={t(`${labels}.moveDown`, { name })}
				className={button}
			>
				<ArrowDown className="size-4" aria-hidden />
			</button>
			<button
				type="button"
				onClick={onRemove}
				aria-label={t(`${labels}.remove`, { name })}
				className={button}
			>
				<X className="size-4" aria-hidden />
			</button>
		</li>
	);
}

/**
 * A search over the owner's sources that aren't chosen yet. Choosing one
 * calls onAdd with its ID and clears the search.
 */
export function SourceSearch({
	id,
	sources,
	chosen,
	onAdd,
	labels = "dashboards.widgets.media",
}: {
	id: string;
	sources: Choice[];
	chosen: string[];
	onAdd: (sourceId: string) => void;
	labels?: Labels;
}) {
	const { t } = useTranslation();
	const [input, setInput] = useState("");
	const full = chosen.length >= maxMediaSources;
	const query = fold(input.trim());
	const choices = sources.filter(
		(s) => !chosen.includes(s.id) && fold(s.title).includes(query),
	);
	const byID = new Map(choices.map((s) => [s.id, s]));

	return (
		<Combobox.Root
			items={choices.map((s) => s.id)}
			filter={null}
			value={null}
			onValueChange={(sourceId: string | null) => {
				if (sourceId === null) return;
				onAdd(sourceId);
				setInput("");
			}}
			inputValue={input}
			onInputValueChange={setInput}
			disabled={full}
		>
			<Combobox.Input
				id={id}
				placeholder={t(`${labels}.addPlaceholder`)}
				aria-describedby={full ? `${id}-full` : undefined}
				className={textInput}
			/>
			{full && (
				<p id={`${id}-full`} className="mt-1 text-sm text-muted">
					{t(`${labels}.full`, { max: maxMediaSources })}
				</p>
			)}
			<Combobox.Portal>
				<Combobox.Positioner sideOffset={4} className="z-50">
					<Combobox.Popup
						className={`${menuPopup} max-h-64 w-[var(--anchor-width)] overflow-y-auto`}
					>
						<Combobox.Empty className="px-3 py-2 text-muted empty:hidden">
							{t(`${labels}.noMatches`)}
						</Combobox.Empty>
						<Combobox.List>
							{(sourceId: string) => {
								const source = byID.get(sourceId);
								if (!source) return null;
								return (
									<Combobox.Item
										key={sourceId}
										value={sourceId}
										className={menuItem}
									>
										<ChosenIcon
											{...iconProps(source.icon)}
											fallback={sourceFallbackIcon(source)}
											className="size-4"
										/>
										<span className="min-w-0 flex-1 truncate">
											{source.title}
										</span>
									</Combobox.Item>
								);
							}}
						</Combobox.List>
					</Combobox.Popup>
				</Combobox.Positioner>
			</Combobox.Portal>
		</Combobox.Root>
	);
}
