import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { DRAG_DROP_PASTE } from "@lexical/rich-text";
import { mergeRegister } from "@lexical/utils";
import { useQueryClient } from "@tanstack/react-query";
import {
	$getRoot,
	COMMAND_PRIORITY_EDITOR,
	COMMAND_PRIORITY_LOW,
	createCommand,
	DecoratorNode,
	type LexicalCommand,
	type LexicalNode,
	type NodeKey,
	type SerializedLexicalNode,
	type Spread,
} from "lexical";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useEffect,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
	AttachmentView,
	UploadingView,
} from "#/features/attachments/AttachmentView";
import {
	attachmentsQueryOptions,
	formatSize,
	type NoteAttachment,
	uploadAttachment,
} from "#/features/attachments/api";
import { ApiError } from "#/lib/api";
import { $insertBlocksAtCursor, useLastSelection } from "./insertBlock";

/** What the attachment blocks of an editor show and do. */
type EditorAttachments = {
	/** The attachments the editor knows, by ID. */
	byId: Record<string, NoteAttachment>;
	/** Opens the image with the ID in the image viewer. */
	openImage: (id: string) => void;
};

export const EditorAttachmentsContext = createContext<EditorAttachments>({
	byId: {},
	openImage: () => {},
});

function EditorAttachment({ id }: { id: string }) {
	const { byId, openImage } = useContext(EditorAttachmentsContext);
	return (
		<AttachmentView attachment={byId[id]} onOpenImage={() => openImage(id)} />
	);
}

export type SerializedAttachmentNode = Spread<
	{ attachmentId: string },
	SerializedLexicalNode
>;

/** An attachment block; the document stores it as {type: "attachment", id}. */
export class AttachmentNode extends DecoratorNode<ReactNode> {
	__attachmentId: string;

	static getType(): string {
		return "attachment";
	}

	static clone(node: AttachmentNode): AttachmentNode {
		return new AttachmentNode(node.__attachmentId, node.__key);
	}

	static importJSON(json: SerializedAttachmentNode): AttachmentNode {
		return $createAttachmentNode(json.attachmentId);
	}

	constructor(attachmentId: string, key?: NodeKey) {
		super(key);
		this.__attachmentId = attachmentId;
	}

	exportJSON(): SerializedAttachmentNode {
		return { ...super.exportJSON(), attachmentId: this.__attachmentId };
	}

	createDOM(): HTMLElement {
		return document.createElement("div");
	}

	updateDOM(): false {
		return false;
	}

	isInline(): false {
		return false;
	}

	getAttachmentId(): string {
		return this.getLatest().__attachmentId;
	}

	getTextContent(): string {
		return "";
	}

	decorate(): ReactNode {
		return <EditorAttachment id={this.__attachmentId} />;
	}
}

export function $createAttachmentNode(attachmentId: string): AttachmentNode {
	return new AttachmentNode(attachmentId);
}

export function $isAttachmentNode(
	node: LexicalNode | null | undefined,
): node is AttachmentNode {
	return node instanceof AttachmentNode;
}

export type SerializedUploadingNode = Spread<
	{ uploadId: string; name: string },
	SerializedLexicalNode
>;

/**
 * Where a file that is uploading will appear. It isn't part of the
 * document, so it is never saved.
 */
export class UploadingNode extends DecoratorNode<ReactNode> {
	__uploadId: string;
	__name: string;

	static getType(): string {
		return "attachment-upload";
	}

	static clone(node: UploadingNode): UploadingNode {
		return new UploadingNode(node.__uploadId, node.__name, node.__key);
	}

	static importJSON(json: SerializedUploadingNode): UploadingNode {
		return new UploadingNode(json.uploadId, json.name);
	}

	constructor(uploadId: string, name: string, key?: NodeKey) {
		super(key);
		this.__uploadId = uploadId;
		this.__name = name;
	}

	exportJSON(): SerializedUploadingNode {
		return {
			...super.exportJSON(),
			uploadId: this.__uploadId,
			name: this.__name,
		};
	}

	createDOM(): HTMLElement {
		return document.createElement("div");
	}

	updateDOM(): false {
		return false;
	}

	isInline(): false {
		return false;
	}

	getTextContent(): string {
		return "";
	}

	decorate(): ReactNode {
		return <UploadingView name={this.__name} />;
	}
}

/** Uploads the files and inserts them at the cursor, or at the end. */
export const INSERT_ATTACHMENTS_COMMAND: LexicalCommand<File[]> = createCommand(
	"INSERT_ATTACHMENTS_COMMAND",
);

/** The top-level nodes for which match and filter hold. */
function $findTopLevel<T extends LexicalNode>(
	match: (node: LexicalNode) => node is T,
	filter: (node: T) => boolean,
): T[] {
	return $getRoot()
		.getChildren()
		.filter((n): n is T => match(n) && filter(n));
}

const isUploading = (n: LexicalNode): n is UploadingNode =>
	n instanceof UploadingNode;

/**
 * Handles files that are pasted or dropped into the editor and
 * INSERT_ATTACHMENTS_COMMAND: each file is shown as a placeholder where it
 * goes, uploaded, and then becomes an attachment block, or a message says
 * why it couldn't be uploaded. onUploaded gets each uploaded attachment.
 */
export function AttachmentsPlugin({
	onUploaded,
}: {
	onUploaded: (attachment: NoteAttachment) => void;
}) {
	const [editor] = useLexicalComposerContext();
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const [errors, setErrors] = useState<string[]>([]);
	const lastSelection = useLastSelection(editor);
	const onUploadedRef = useRef(onUploaded);
	onUploadedRef.current = onUploaded;

	const errorMessage = useCallback(
		async (err: unknown, name: string) => {
			if (err instanceof ApiError && err.code === "attachment_too_large") {
				const limits = await queryClient
					.fetchQuery(attachmentsQueryOptions)
					.catch(() => null);
				if (limits)
					return t("attachments.upload.tooLarge", {
						name,
						max: formatSize(limits.maxSize, i18n.language),
					});
			}
			if (err instanceof ApiError && err.code === "unsupported_attachment_type")
				return t("attachments.upload.unsupported", { name });
			return t("attachments.upload.failed", { name });
		},
		[queryClient, t, i18n.language],
	);

	const insert = useCallback(
		async (files: File[]) => {
			if (!editor.isEditable() || files.length === 0) return;
			setErrors([]);
			const uploads = files.map((file) => ({
				file,
				uploadId: crypto.randomUUID(),
			}));
			editor.update(() =>
				$insertBlocksAtCursor(
					uploads.map((u) => new UploadingNode(u.uploadId, u.file.name)),
					lastSelection.current,
				),
			);
			for (const { file, uploadId } of uploads) {
				const isThis = (n: UploadingNode) => n.__uploadId === uploadId;
				try {
					const attachment = await uploadAttachment(file);
					onUploadedRef.current(attachment);
					editor.update(() => {
						for (const node of $findTopLevel(isUploading, isThis))
							node.replace($createAttachmentNode(attachment.id));
					});
				} catch (err) {
					editor.update(() => {
						for (const node of $findTopLevel(isUploading, isThis))
							node.remove();
					});
					const message = await errorMessage(err, file.name);
					setErrors((list) => [...list, message]);
				}
			}
		},
		[editor, errorMessage, lastSelection],
	);

	useEffect(
		() =>
			mergeRegister(
				editor.registerCommand(
					DRAG_DROP_PASTE,
					(files) => {
						void insert(files);
						return true;
					},
					COMMAND_PRIORITY_LOW,
				),
				editor.registerCommand(
					INSERT_ATTACHMENTS_COMMAND,
					(files) => {
						void insert(files);
						return true;
					},
					COMMAND_PRIORITY_EDITOR,
				),
			),
		[editor, insert],
	);

	if (errors.length === 0) return null;
	return (
		<div role="alert" className="mt-3 space-y-1 text-sm text-danger">
			{errors.map((message, i) => (
				// biome-ignore lint/suspicious/noArrayIndexKey: messages are only added
				<p key={i}>{message}</p>
			))}
		</div>
	);
}
