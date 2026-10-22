import { useDroppable } from "@dnd-kit/core";
import {
	rectSortingStrategy,
	SortableContext,
	useSortable,
} from "@dnd-kit/sortable";
import { GripVertical, LayoutGrid, Plus, Settings2 } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
	type ContentWidget,
	isSection,
	type Section,
	type Widget,
} from "../api";
import { findKind, widgetKinds, widgetLabel } from "../widgets/registry";
import { containerId, useEditor } from "./editor";
import { maxDepth } from "./layoutTree";

/** One row unit and the gap between grid cells; S/M/L can change them later. */
const gridVars = {
	"--dashboard-row": "4rem",
	"--dashboard-gap": "1rem",
} as CSSProperties;

/**
 * The top level of a dashboard: a grid with `columns` columns. Each widget is
 * at most as wide as its parent, and a section lays out its children on a
 * grid with as many columns as it is wide.
 */
export function DashboardGrid({
	sections,
	columns,
}: {
	sections: Widget[];
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

/** Content widgets can be added inside sections when any kind exists. */
const canAddWidgets = (parentId: string | null) =>
	parentId !== null && widgetKinds.length > 0;

function SectionList({
	sections,
	columns,
	parentId,
	level,
}: {
	sections: Widget[];
	columns: number;
	parentId: string | null;
	level: number;
}) {
	const { t } = useTranslation();
	const editor = useEditor();
	const addSection = level <= maxDepth;
	const addWidget = canAddWidgets(parentId);
	const grid = (
		<Grid columns={columns} parentId={parentId}>
			{sections.map((widget) =>
				isSection(widget) ? (
					<SectionBlock
						key={widget.id}
						section={widget}
						parentColumns={columns}
						level={level}
					/>
				) : (
					<ContentBlock
						key={widget.id}
						widget={widget}
						parentColumns={columns}
					/>
				),
			)}
			{editor && (addSection || addWidget) && (
				<div
					className="col-span-full flex flex-wrap gap-2"
					data-parent={parentId ?? "root"}
				>
					{addSection && (
						<AddButton
							label={t("dashboards.editor.addSection")}
							onClick={() => editor.onAdd(parentId)}
						/>
					)}
					{addWidget && parentId !== null && (
						<AddButton
							label={t("dashboards.editor.addWidget")}
							onClick={() => editor.onAddWidget(parentId)}
						/>
					)}
				</div>
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
					widget={props.section}
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
			{(section.children.length > 0 ||
				(editor && (level < maxDepth || canAddWidgets(section.id)))) && (
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
	widget,
	picked,
	activatorRef,
	listeners,
}: {
	widget: Widget;
	picked: boolean;
	activatorRef: (el: HTMLButtonElement | null) => void;
	listeners: ReturnType<typeof useSortable>["listeners"];
}) {
	const { t } = useTranslation();
	const editor = useEditor();
	const title = widgetLabel(widget, t);
	return (
		<button
			type="button"
			ref={activatorRef}
			{...listeners}
			// The keyboard is handled by the editor, not by a dnd-kit sensor.
			onKeyDown={(event) => editor?.onHandleKeyDown(event, widget)}
			onBlur={(event) => editor?.onHandleBlur(event, widget)}
			aria-label={t("dashboards.editor.moveSection", { title })}
			aria-pressed={picked}
			className={`cursor-grab touch-none rounded-md p-1 text-muted hover:bg-paper hover:text-ink ${picked ? "bg-paper text-ink ring-2 ring-ink" : ""}`}
		>
			<GripVertical className="size-4" aria-hidden />
		</button>
	);
}

function AddButton({ label, onClick }: { label: string; onClick: () => void }) {
	return (
		<button
			type="button"
			onClick={onClick}
			className="inline-flex min-w-0 flex-1 items-center justify-center gap-2 rounded-xl border border-dashed border-line px-4 py-3 text-sm text-muted hover:border-ink hover:text-ink"
		>
			<Plus className="size-4 shrink-0" aria-hidden />
			<span className="truncate">{label}</span>
		</button>
	);
}

function ContentBlock(props: { widget: ContentWidget; parentColumns: number }) {
	return useEditor() ? (
		<SortableContent {...props} />
	) : (
		<WidgetFrame {...props} />
	);
}

function SortableContent(props: {
	widget: ContentWidget;
	parentColumns: number;
}) {
	const editor = useEditor();
	const {
		setNodeRef,
		setActivatorNodeRef,
		listeners,
		transform,
		transition,
		isDragging,
	} = useSortable({ id: props.widget.id });
	return (
		<WidgetFrame
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
					widget={props.widget}
					picked={editor?.pickedId === props.widget.id}
					activatorRef={(el) => {
						setActivatorNodeRef(el);
						editor?.registerHandle(props.widget.id, el);
					}}
					listeners={listeners}
				/>
			}
		/>
	);
}

/**
 * A content widget's cell: it fills its grid cell, has its height, and
 * scrolls when its content doesn't fit. In edit mode it gets a dashed outline
 * and a bar with the drag handle and the edit button.
 */
function WidgetFrame({
	widget,
	parentColumns,
	frameRef,
	dragStyle,
	handle,
}: {
	widget: ContentWidget;
	parentColumns: number;
	frameRef?: (el: HTMLElement | null) => void;
	dragStyle?: CSSProperties;
	handle?: ReactNode;
}) {
	const { t } = useTranslation();
	const editor = useEditor();
	const kind = findKind(widget.kind);
	const span = Math.min(widget.width, parentColumns);
	const rows = widget.height === "auto" ? null : widget.height;
	const label = widgetLabel(widget, t);
	const dropBefore = editor?.dropTarget?.overId === widget.id;
	const outline = !editor
		? ""
		: dropBefore
			? `rounded-xl p-2 outline-2 -outline-offset-2 outline-dashed ${editor.dropTarget?.blocked ? "outline-red-700" : "outline-ink"}`
			: "rounded-xl p-2 outline-2 outline-dashed -outline-offset-2 outline-line";
	const Icon = kind?.icon ?? LayoutGrid;

	return (
		<div
			ref={frameRef}
			data-testid={`widget-${widget.id}`}
			data-span={span}
			className={`flex min-w-0 flex-col gap-2 ${outline}`}
			style={{
				gridColumn: `span ${span} / span ${span}`,
				height: rows
					? `calc(${rows} * var(--dashboard-row) + ${rows - 1} * var(--dashboard-gap))`
					: undefined,
				...dragStyle,
			}}
		>
			{editor && (
				<div className="flex min-w-0 items-center gap-2">
					{handle}
					<Icon className="size-4 shrink-0 text-muted" aria-hidden />
					<span className="min-w-0 flex-1 truncate text-sm text-muted">
						{label}
					</span>
					<button
						type="button"
						onClick={() => editor.onEdit(widget.id)}
						aria-label={t("dashboards.editor.editWidget", { title: label })}
						className="rounded-md p-1 text-muted hover:bg-paper hover:text-ink"
					>
						<Settings2 className="size-4" aria-hidden />
					</button>
				</div>
			)}
			<div className="min-h-0 min-w-0 flex-1 overflow-auto">
				{kind && <kind.View widget={widget} editing={editor !== null} />}
			</div>
		</div>
	);
}
