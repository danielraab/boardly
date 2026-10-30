import { createContext, type ReactNode, useContext } from "react";
import { AttachmentView } from "#/features/attachments/AttachmentView";
import type { NoteAttachment } from "#/features/attachments/api";
import { useImageViewer } from "#/features/attachments/ImageViewer";
import {
	sizeStyles,
	type WidgetSize,
} from "#/features/dashboards/widgets/sizes";
import { inlineText } from "./document/text";
import {
	attachmentIds,
	type Block,
	type Inline,
	type ListBlock,
	type Mark,
	type NoteDoc,
	type TextRun,
} from "./document/types";

/** The attachments of the rendered note, and how they react. */
const RenderedAttachments = createContext<{
	byId: Record<string, NoteAttachment>;
	interactive: boolean;
	openImage?: (id: string) => void;
}>({ byId: {}, interactive: false });

function Attachment({ id }: { id: string }) {
	const { byId, interactive, openImage } = useContext(RenderedAttachments);
	return (
		<AttachmentView
			attachment={byId[id]}
			interactive={interactive}
			onOpenImage={openImage && (() => openImage(id))}
		/>
	);
}

/** Tailwind classes of the marks. */
const markClass: Record<Mark, string> = {
	bold: "font-semibold",
	italic: "italic",
	strike: "line-through",
	code: "rounded bg-ink/10 px-1 font-mono text-[0.9em]",
};

function Run({ run }: { run: TextRun }) {
	if (!run.marks?.length) return run.text;
	return (
		<span className={run.marks.map((m) => markClass[m]).join(" ")}>
			{run.text}
		</span>
	);
}

function Inlines({ content }: { content: Inline[] }) {
	return content.map((inline, i) => {
		// The content is only rendered, never reordered.
		const key = i;
		if (inline.type === "break") return <br key={key} />;
		if (inline.type === "link")
			// Inert on cards: activating it starts editing like the rest.
			return (
				<span key={key} className="text-accent underline">
					{inline.content.map((run, j) => (
						// biome-ignore lint/suspicious/noArrayIndexKey: rendered once, never reordered
						<Run key={j} run={run} />
					))}
				</span>
			);
		return <Run key={key} run={inline} />;
	});
}

/** Called when a task item's checkbox is toggled. */
type OnToggleTask = (id: string, checked: boolean) => void;

function List({
	list,
	onToggleTask,
}: {
	list: ListBlock;
	onToggleTask?: OnToggleTask;
}) {
	const items = list.items.map((item, i) => {
		const key = item.id ?? i;
		const nested = item.children && (
			<List list={item.children} onToggleTask={onToggleTask} />
		);
		if (list.style !== "task") {
			return (
				<li key={key}>
					<Inlines content={item.content} />
					{nested}
				</li>
			);
		}
		return (
			<li key={key} className="list-none">
				<span className="flex items-start gap-2">
					{/* Inert without onToggleTask. Above the card's edit button with it. */}
					<input
						type="checkbox"
						checked={item.checked === true}
						disabled={!onToggleTask}
						readOnly={!onToggleTask}
						tabIndex={onToggleTask ? undefined : -1}
						onChange={(e) => {
							if (item.id) onToggleTask?.(item.id, e.target.checked);
						}}
						onClick={(e) => e.stopPropagation()}
						aria-label={inlineText(item.content)}
						className="pointer-events-auto mt-[0.2em] size-[1.143em] shrink-0 accent-accent disabled:opacity-100"
					/>
					<span
						className={item.checked ? "text-muted line-through" : undefined}
					>
						<Inlines content={item.content} />
					</span>
				</span>
				{nested}
			</li>
		);
	});
	if (list.style === "ordered")
		return <ol className="list-decimal space-y-0.5 pl-6">{items}</ol>;
	return (
		<ul
			className={
				list.style === "task"
					? "space-y-0.5 pl-0 [&_ul]:pl-6"
					: "list-disc space-y-0.5 pl-6"
			}
		>
			{items}
		</ul>
	);
}

function BlockView({
	block,
	onToggleTask,
}: {
	block: Block;
	onToggleTask?: OnToggleTask;
}): ReactNode {
	switch (block.type) {
		case "paragraph":
			return (
				<p className="min-h-[1lh]">
					<Inlines content={block.content} />
				</p>
			);
		case "heading": {
			const Tag = `h${block.level + 1}` as "h2" | "h3" | "h4";
			// Relative to the content's text size: lg, base and sm at text-sm.
			const size = {
				1: "text-[1.286em]/[1.556]",
				2: "text-[1.143em]/[1.5]",
				3: "text-[1em]",
			}[block.level];
			return (
				<Tag className={`${size} font-semibold`}>
					<Inlines content={block.content} />
				</Tag>
			);
		}
		case "quote":
			return (
				<blockquote className="border-l-2 border-line pl-3 text-muted">
					<Inlines content={block.content} />
				</blockquote>
			);
		case "code":
			return (
				<pre className="overflow-hidden rounded-md bg-ink/5 p-2 font-mono text-[0.857em]/[1.333] whitespace-pre-wrap">
					{block.text}
				</pre>
			);
		case "divider":
			return <hr className="border-line" />;
		case "list":
			return <List list={block} onToggleTask={onToggleTask} />;
		case "attachment":
			return <Attachment id={block.id} />;
	}
}

/**
 * A note's content as read-only React elements, for the cards of /notes and
 * the note widget. Headings start one level below the card's title. Links
 * don't react to activation; checkboxes only do with onToggleTask. A size
 * sets the text size as for widgets; without one it is the cards' text-sm.
 * attachments are the note's; images open in the image viewer. With
 * interactiveAttachments, as in widgets, files can be downloaded and audio and
 * video played, while on the cards activating them starts editing like the
 * rest.
 */
export function NoteRenderer({
	doc,
	attachments = [],
	interactiveAttachments = false,
	onToggleTask,
	size,
}: {
	doc: NoteDoc;
	attachments?: NoteAttachment[];
	interactiveAttachments?: boolean;
	onToggleTask?: OnToggleTask;
	size?: WidgetSize;
}) {
	const text = size ? sizeStyles[size].text : "text-sm";
	const images = useImageViewer();
	const byId = Object.fromEntries(attachments.map((a) => [a.id, a]));
	const openImage = (id: string) =>
		images.open(
			attachmentIds(doc)
				.map((a) => byId[a])
				.filter((a) => a?.contentType.startsWith("image/")),
			id,
		);
	return (
		<RenderedAttachments
			value={{
				byId,
				interactive: interactiveAttachments,
				openImage,
			}}
		>
			<div className={`space-y-1.5 ${text} break-words`}>
				{doc.blocks.map((block, i) => (
					// biome-ignore lint/suspicious/noArrayIndexKey: rendered once, never reordered
					<BlockView key={i} block={block} onToggleTask={onToggleTask} />
				))}
			</div>
			{images.viewer}
		</RenderedAttachments>
	);
}
