import { AutoFocusPlugin } from "@lexical/react/LexicalAutoFocusPlugin";
import { CheckListPlugin } from "@lexical/react/LexicalCheckListPlugin";
import { LexicalComposer } from "@lexical/react/LexicalComposer";
import { ContentEditable } from "@lexical/react/LexicalContentEditable";
import { EditorRefPlugin } from "@lexical/react/LexicalEditorRefPlugin";
import { LexicalErrorBoundary } from "@lexical/react/LexicalErrorBoundary";
import { HistoryPlugin } from "@lexical/react/LexicalHistoryPlugin";
import { HorizontalRulePlugin } from "@lexical/react/LexicalHorizontalRulePlugin";
import { LinkPlugin } from "@lexical/react/LexicalLinkPlugin";
import { ListPlugin } from "@lexical/react/LexicalListPlugin";
import { MarkdownShortcutPlugin } from "@lexical/react/LexicalMarkdownShortcutPlugin";
import { OnChangePlugin } from "@lexical/react/LexicalOnChangePlugin";
import { RichTextPlugin } from "@lexical/react/LexicalRichTextPlugin";
import type { LexicalEditor } from "lexical";
import { useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { NoteAttachment } from "#/features/attachments/api";
import { useImageViewer } from "#/features/attachments/ImageViewer";
import { $docToLexical, $lexicalToDoc, editorNodes } from "../document/lexical";
import { attachmentIds, isLinkUrl, type NoteDoc } from "../document/types";
import { AttachmentsPlugin, EditorAttachmentsContext } from "./attachments";
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
 * Without toolbar, it is the compact editor of a card on /notes. attachments
 * are those of the note; files pasted or dropped into it are uploaded and
 * passed to onAttachmentAdded.
 */
export function NoteEditor({
	initial,
	onChange,
	attachments = [],
	onAttachmentAdded,
	editable = true,
	toolbar = true,
	autoFocus = false,
}: {
	initial: NoteDoc;
	onChange: (doc: NoteDoc) => void;
	attachments?: NoteAttachment[];
	onAttachmentAdded?: (attachment: NoteAttachment) => void;
	editable?: boolean;
	toolbar?: boolean;
	/** Puts the cursor at the end of the content on mount. */
	autoFocus?: boolean;
}) {
	const { t } = useTranslation();
	const initialRef = useRef(initial);
	const placeholder = t("notes.editor.placeholder");
	const lexicalRef = useRef<LexicalEditor | null>(null);
	const [byId, setById] = useState(() =>
		Object.fromEntries(attachments.map((a) => [a.id, a])),
	);
	const images = useImageViewer();
	const context = useMemo(
		() => ({
			byId,
			openImage: (id: string) => {
				const doc = lexicalRef.current?.getEditorState().read($lexicalToDoc);
				const shown = attachmentIds(doc ?? { v: 1, blocks: [] })
					.map((a) => byId[a])
					.filter((a) => a?.contentType.startsWith("image/"));
				images.open(shown, id);
			},
		}),
		[byId, images.open],
	);

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
			{toolbar && (
				<div className="sticky top-0 z-10 -mx-2 mb-3 rounded-lg bg-inherit px-2 py-1">
					<Toolbar disabled={!editable} />
				</div>
			)}
			<EditorAttachmentsContext value={context}>
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
								className={`${toolbar ? "min-h-64" : "min-h-16 text-sm"} space-y-2 break-words outline-none`}
							/>
						}
						ErrorBoundary={LexicalErrorBoundary}
					/>
				</div>
			</EditorAttachmentsContext>
			<AttachmentsPlugin
				onUploaded={(attachment) => {
					setById((known) => ({ ...known, [attachment.id]: attachment }));
					onAttachmentAdded?.(attachment);
				}}
			/>
			<EditorRefPlugin editorRef={lexicalRef} />
			{images.viewer}
			{autoFocus && <AutoFocusPlugin defaultSelection="rootEnd" />}
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
