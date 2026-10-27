import type { ReactNode } from "react";
import { inlineText } from "./document/text";
import type {
	Block,
	Inline,
	ListBlock,
	Mark,
	NoteDoc,
	TextRun,
} from "./document/types";

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
			// Inert on cards: activating the card opens the note.
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

function List({ list }: { list: ListBlock }) {
	const items = list.items.map((item, i) => {
		const key = item.id ?? i;
		const nested = item.children && <List list={item.children} />;
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
					{/* Inert on cards: activating the card opens the note. */}
					<input
						type="checkbox"
						checked={item.checked === true}
						disabled
						readOnly
						tabIndex={-1}
						aria-label={inlineText(item.content)}
						className="mt-[0.2em] size-4 shrink-0 accent-accent disabled:opacity-100"
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

function BlockView({ block }: { block: Block }): ReactNode {
	switch (block.type) {
		case "paragraph":
			return (
				<p className="min-h-[1lh]">
					<Inlines content={block.content} />
				</p>
			);
		case "heading": {
			const Tag = `h${block.level + 1}` as "h2" | "h3" | "h4";
			const size = { 1: "text-lg", 2: "text-base", 3: "text-sm" }[block.level];
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
				<pre className="overflow-hidden rounded-md bg-ink/5 p-2 font-mono text-xs whitespace-pre-wrap">
					{block.text}
				</pre>
			);
		case "divider":
			return <hr className="border-line" />;
		case "list":
			return <List list={block} />;
	}
}

/**
 * A note's content as read-only React elements, for the cards of /notes.
 * Headings start one level below the card's title. Links and checkboxes
 * don't react to activation.
 */
export function NoteRenderer({ doc }: { doc: NoteDoc }) {
	return (
		<div className="space-y-1.5 text-sm break-words">
			{doc.blocks.map((block, i) => (
				// biome-ignore lint/suspicious/noArrayIndexKey: rendered once, never reordered
				<BlockView key={i} block={block} />
			))}
		</div>
	);
}
