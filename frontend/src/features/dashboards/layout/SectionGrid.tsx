import { useDroppable } from "@dnd-kit/core";
import {
	rectSortingStrategy,
	SortableContext,
	useSortable,
} from "@dnd-kit/sortable";
import { GripVertical, Plus, Settings2 } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { Section } from "../api";
import { containerId, useEditor } from "./editor";
import { maxDepth } from "./layoutTree";

/** One row unit and the gap between grid cells; S/M/L can change them later. */
const gridVars = {
	"--dashboard-row": "4rem",
	"--dashboard-gap": "1rem",
} as CSSProperties;

/**
 * The top level of a dashboard: a grid with `columns` columns. Each section is
 * at most as wide as its parent and lays out its children on a grid with as
 * many columns as it is wide.
 */
export function DashboardGrid({
	sections,
	columns,
}: {
	sections: Section[];
	columns: number;
}) {
	return (
		<div style={gridVars}>
			<SectionList
				sections={sections}
				columns={columns}
				parentId={null}
				level={1}
			/>
		</div>
	);
}

function SectionList({
	sections,
	columns,
	parentId,
	level,
}: {
	sections: Section[];
	columns: number;
	parentId: string | null;
	level: number;
}) {
	const editor = useEditor();
	const grid = (
		<Grid columns={columns} parentId={parentId}>
			{sections.map((section) => (
				<SectionBlock
					key={section.id}
					section={section}
					parentColumns={columns}
					level={level}
				/>
			))}
			{editor && level <= maxDepth && (
				<AddSectionButton
					parentId={parentId}
					onClick={() => editor.onAdd(parentId)}
				/>
			)}
		</Grid>
	);
	if (!editor) return grid;
	return (
		<SortableContext
			id={containerId(parentId)}
			items={sections.map((s) => s.id)}
			strategy={rectSortingStrategy}
		>
			{grid}
		</SortableContext>
	);
}

function Grid({
	columns,
	parentId,
	children,
}: {
	columns: number;
	parentId: string | null;
	children: ReactNode;
}) {
	const editor = useEditor();
	const style: CSSProperties = {
		gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
		gap: "var(--dashboard-gap)",
	};
	if (!editor) {
		return (
			<div className="grid" style={style}>
				{children}
			</div>
		);
	}
	const target = editor.dropTarget;
	const highlight =
		target?.parentId === parentId
			? target.blocked
				? "dashboard-drop-blocked"
				: "dashboard-drop"
			: undefined;
	return (
		<DroppableGrid
			id={containerId(parentId)}
			style={style}
			highlight={highlight}
		>
			{children}
		</DroppableGrid>
	);
}

/** Outline classes of a grid that is the target of a drag from another grid. */
const highlights = {
	"dashboard-drop": "rounded-lg outline-2 outline-offset-4 outline-ink",
	"dashboard-drop-blocked":
		"rounded-lg outline-2 outline-offset-4 outline-red-700",
};

/** In edit mode every grid is a drop target, so empty sections accept drops. */
function DroppableGrid({
	id,
	style,
	highlight,
	children,
}: {
	id: string;
	style: CSSProperties;
	highlight?: keyof typeof highlights;
	children: ReactNode;
}) {
	const { setNodeRef } = useDroppable({ id });
	return (
		<div
			ref={setNodeRef}
			className={`grid ${highlight ? highlights[highlight] : ""}`}
			style={style}
			data-drop={highlight}
		>
			{children}
		</div>
	);
}

function SectionBlock(props: {
	section: Section;
	parentColumns: number;
	level: number;
}) {
	return useEditor() ? (
		<SortableSection {...props} />
	) : (
		<SectionFrame {...props} />
	);
}

function SortableSection(props: {
	section: Section;
	parentColumns: number;
	level: number;
}) {
	const editor = useEditor();
	const {
		setNodeRef,
		setActivatorNodeRef,
		listeners,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: props.section.id });
	return (
		<SectionFrame
			{...props}
			frameRef={setNodeRef}
			dragStyle={{
				transform: transform
					? `translate3d(${transform.x}px, ${transform.y}px, 0)`
					: undefined,
				transition,
				opacity: isDragging ? 0.4 : undefined,
			}}
			handle={
				<DragHandle
					section={props.section}
					picked={editor?.pickedId === props.section.id}
					activatorRef={(el) => {
						setActivatorNodeRef(el);
						editor?.registerHandle(props.section.id, el);
					}}
					listeners={listeners}
				/>
			}
		/>
	);
}

/** A section's box: title, border, height and its children's grid. */
function SectionFrame({
	section,
	parentColumns,
	level,
	frameRef,
	dragStyle,
	handle,
}: {
	section: Section;
	parentColumns: number;
	level: number;
	frameRef?: (el: HTMLElement | null) => void;
	dragStyle?: CSSProperties;
	handle?: ReactNode;
}) {
	const { t } = useTranslation();
	const editor = useEditor();
	const span = Math.min(section.width, parentColumns);
	const rows = section.height === "auto" ? null : section.height;
	const Heading = `h${Math.min(level + 1, 6)}` as "h2";
	const title = section.title ?? t("dashboards.editor.untitled");

	const box = section.border
		? "rounded-xl border border-line bg-surface p-4"
		: editor
			? "rounded-xl p-2"
			: "";
	// In edit mode a borderless section gets a dashed outline so it stays
	// visible. A drop in front of the section colors the outline.
	const dropBefore = editor?.dropTarget?.overId === section.id;
	const dropColor = editor?.dropTarget?.blocked
		? "outline-red-700"
		: "outline-ink";
	const outline = !editor
		? ""
		: dropBefore
			? `outline-2 -outline-offset-2 ${dropColor} ${section.border ? "" : "outline-dashed"}`
			: section.border
				? ""
				: "outline-2 outline-dashed -outline-offset-2 outline-line";

	return (
		<section
			ref={frameRef}
			aria-label={section.title ?? undefined}
			data-testid={`section-${section.id}`}
			data-span={span}
			className={`flex min-w-0 flex-col gap-3 ${box} ${outline} ${rows ? "overflow-y-auto" : ""}`}
			style={{
				gridColumn: `span ${span} / span ${span}`,
				height: rows
					? `calc(${rows} * var(--dashboard-row) + ${rows - 1} * var(--dashboard-gap))`
					: undefined,
				...dragStyle,
			}}
		>
			{(section.title || editor) && (
				<div className="flex min-w-0 items-center gap-2">
					{handle}
					{section.title ? (
						<Heading className="min-w-0 flex-1 truncate font-semibold">
							{section.title}
						</Heading>
					) : (
						<span className="min-w-0 flex-1 truncate text-sm text-muted">
							{title}
						</span>
					)}
					{editor && (
						<button
							type="button"
							onClick={() => editor.onEdit(section.id)}
							aria-label={t("dashboards.editor.editSection", { title })}
							className="rounded-md p-1 text-muted hover:bg-paper hover:text-ink"
						>
							<Settings2 className="size-4" aria-hidden />
						</button>
					)}
				</div>
			)}
			{(section.children.length > 0 || (editor && level < maxDepth)) && (
				<SectionList
					sections={section.children}
					columns={span}
					parentId={section.id}
					level={level + 1}
				/>
			)}
		</section>
	);
}

function DragHandle({
	section,
	picked,
	activatorRef,
	listeners,
}: {
	section: Section;
	picked: boolean;
	activatorRef: (el: HTMLButtonElement | null) => void;
	listeners: ReturnType<typeof useSortable>["listeners"];
}) {
	const { t } = useTranslation();
	const editor = useEditor();
	const title = section.title ?? t("dashboards.editor.untitled");
	return (
		<button
			type="button"
			ref={activatorRef}
			{...listeners}
			// The keyboard is handled by the editor, not by a dnd-kit sensor.
			onKeyDown={(event) => editor?.onHandleKeyDown(event, section)}
			onBlur={(event) => editor?.onHandleBlur(event, section)}
			aria-label={t("dashboards.editor.moveSection", { title })}
			aria-pressed={picked}
			className={`cursor-grab touch-none rounded-md p-1 text-muted hover:bg-paper hover:text-ink ${picked ? "bg-paper text-ink ring-2 ring-ink" : ""}`}
		>
			<GripVertical className="size-4" aria-hidden />
		</button>
	);
}

function AddSectionButton({
	parentId,
	onClick,
}: {
	parentId: string | null;
	onClick: () => void;
}) {
	const { t } = useTranslation();
	return (
		<button
			type="button"
			onClick={onClick}
			data-parent={parentId ?? "root"}
			className="col-span-full inline-flex items-center justify-center gap-2 rounded-xl border border-dashed border-line px-4 py-3 text-sm text-muted hover:border-ink hover:text-ink"
		>
			<Plus className="size-4" aria-hidden />
			{t("dashboards.editor.addSection")}
		</button>
	);
}
