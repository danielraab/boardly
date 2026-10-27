import { CheckListPlugin } from "@lexical/react/LexicalCheckListPlugin";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { HorizontalRulePlugin } from "@lexical/react/LexicalHorizontalRulePlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { $docToLexical, $lexicalToDoc, editorNodes } from "../document/lexical";
import { isLinkUrl, type NoteDoc } from "../document/types";
import {
	EditablePlugin,
	ListTabPlugin,
	ModClickLinkPlugin,
	TaskIdPlugin,
	TaskShortcutPlugin,
} from "./plugins";
import { Toolbar } from "./Toolbar";
import { editorTheme } from "./theme";
import { noteTransformers } from "./transformers";

/**
 * The rich text editor of a note. It starts from initial and reports every
 * change of the content as a document; later changes of initial are ignored.
 */
export function NoteEditor({
	initial,
	onChange,
	editable = true,
}: {
	initial: NoteDoc;
	onChange: (doc: NoteDoc) => void;
	editable?: boolean;
}) {
	const { t } = useTranslation();
	const initialRef = useRef(initial);
	const placeholder = t("notes.editor.placeholder");

	return (
		<LexicalComposer
			initialConfig={{
				namespace: "note",
				nodes: editorNodes,
				theme: editorTheme,
				editable,
				editorState: () => $docToLexical(initialRef.current),
				onError: (error) => {
					throw error;
				},
			}}
		>
			<div className="sticky top-0 z-10 -mx-2 mb-3 rounded-lg bg-inherit px-2 py-1">
				<Toolbar disabled={!editable} />
			</div>
			<div className="relative">
				<RichTextPlugin
					contentEditable={
						<ContentEditable
							aria-label={t("notes.editor.content")}
							aria-placeholder={placeholder}
							placeholder={
								<div className="pointer-events-none absolute top-0 left-0 text-muted select-none">
									{placeholder}
								</div>
							}
							className="min-h-64 space-y-2 break-words outline-none"
						/>
					}
					ErrorBoundary={LexicalErrorBoundary}
				/>
			</div>
			<HistoryPlugin />
			<ListPlugin />
			<CheckListPlugin />
			<LinkPlugin validateUrl={isLinkUrl} />
			<HorizontalRulePlugin />
			<MarkdownShortcutPlugin transformers={noteTransformers} />
			<TaskShortcutPlugin />
			<TaskIdPlugin />
			<ListTabPlugin />
			<ModClickLinkPlugin />
			<EditablePlugin editable={editable} />
			<OnChangePlugin
				ignoreSelectionChange
				ignoreHistoryMergeTagChange
				onChange={(state) => onChange(state.read($lexicalToDoc))}
			/>
		</LexicalComposer>
	);
}
