import type { EditorThemeClasses } from "lexical";

/**
 * Classes of the editor's nodes. They match NoteRenderer, so a note looks
 * the same on its card and in the editor; .note-task (styles.css) draws the
 * checkboxes of task items.
 */
export const editorTheme: EditorThemeClasses = {
	paragraph: "min-h-[1lh]",
	heading: {
		h1: "text-2xl font-semibold",
		h2: "text-xl font-semibold",
		h3: "text-lg font-semibold",
	},
	quote: "border-l-2 border-line pl-3 text-muted",
	code: "block rounded-md bg-ink/5 p-3 font-mono text-sm whitespace-pre-wrap",
	hr: "my-2 border-line",
	link: "text-accent underline",
	list: {
		ul: "list-disc space-y-1 pl-6",
		ol: "list-decimal space-y-1 pl-6",
		checklist: "space-y-1 pl-0",
		listitem: "",
		listitemChecked: "note-task note-task-checked",
		listitemUnchecked: "note-task",
		nested: { listitem: "list-none" },
	},
	text: {
		bold: "font-semibold",
		italic: "italic",
		strikethrough: "line-through",
		code: "rounded bg-ink/10 px-1 font-mono text-[0.9em]",
	},
};
