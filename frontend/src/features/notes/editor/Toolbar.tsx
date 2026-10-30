import { Popover } from "@base-ui/react/popover";
import { $isLinkNode, $toggleLink } from "@lexical/link";
import {
	$isListNode,
	INSERT_CHECK_LIST_COMMAND,
	INSERT_ORDERED_LIST_COMMAND,
	INSERT_UNORDERED_LIST_COMMAND,
	ListNode,
	REMOVE_LIST_COMMAND,
} from "@lexical/list";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { $createHeadingNode, $isHeadingNode } from "@lexical/rich-text";
import { $setBlocksType } from "@lexical/selection";
import { $findMatchingParent, $getNearestNodeOfType } from "@lexical/utils";
import { useQuery } from "@tanstack/react-query";
import {
	$createParagraphNode,
	$getSelection,
	$isRangeSelection,
	$setSelection,
	type BaseSelection,
	FORMAT_TEXT_COMMAND,
	type TextFormatType,
} from "lexical";
import {
	Bold,
	Code,
	FilePlus2,
	Heading,
	Italic,
	Link,
	List,
	ListChecks,
	ListOrdered,
	Strikethrough,
} from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import { menuPopup } from "#/components/menuStyles";
import { attachmentsQueryOptions } from "#/features/attachments/api";
import { isLinkUrl } from "../document/types";
import { INSERT_ATTACHMENTS_COMMAND } from "./attachments";

type ListKind = "bullet" | "number" | "check";

/** The formatting at the selection, for the pressed state of the buttons. */
type Active = {
	heading: boolean;
	list: ListKind | null;
	formats: Set<TextFormatType>;
	link: string | null;
};

const none: Active = {
	heading: false,
	list: null,
	formats: new Set(),
	link: null,
};

const formats = [
	{ format: "bold", icon: Bold, key: "bold" },
	{ format: "italic", icon: Italic, key: "italic" },
	{ format: "strikethrough", icon: Strikethrough, key: "strike" },
	{ format: "code", icon: Code, key: "code" },
] as const;

const lists = [
	{
		kind: "bullet",
		icon: List,
		key: "bullet",
		command: INSERT_UNORDERED_LIST_COMMAND,
	},
	{
		kind: "number",
		icon: ListOrdered,
		key: "ordered",
		command: INSERT_ORDERED_LIST_COMMAND,
	},
	{
		kind: "check",
		icon: ListChecks,
		key: "task",
		command: INSERT_CHECK_LIST_COMMAND,
	},
] as const;

const toolButton =
	"inline-flex size-8 shrink-0 items-center justify-center rounded-md text-muted hover:bg-ink/10 hover:text-ink disabled:opacity-40 aria-pressed:bg-ink/10 aria-pressed:text-ink data-[popup-open]:bg-ink/10";

/** Keeps the editor's selection when a toolbar button is pressed. */
const keepSelection = (event: { preventDefault: () => void }) =>
	event.preventDefault();

/** The formatting toolbar of the note editor. */
export function Toolbar({ disabled }: { disabled: boolean }) {
	const { t } = useTranslation();
	const [editor] = useLexicalComposerContext();
	const [active, setActive] = useState<Active>(none);

	useEffect(
		() =>
			editor.registerUpdateListener(({ editorState }) => {
				editorState.read(() => {
					const selection = $getSelection();
					if (!$isRangeSelection(selection)) return setActive(none);
					const anchor = selection.anchor.getNode();
					const block = anchor.getTopLevelElement();
					const list = $getNearestNodeOfType(anchor, ListNode);
					const link = $findMatchingParent(anchor, $isLinkNode);
					setActive({
						heading: $isHeadingNode(block),
						list: $isListNode(list) ? list.getListType() : null,
						formats: new Set(
							formats
								.map((f) => f.format as TextFormatType)
								.filter((f) => selection.hasFormat(f)),
						),
						link: $isLinkNode(link) ? link.getURL() : null,
					});
				});
			}),
		[editor],
	);

	const toggleHeading = () =>
		editor.update(() => {
			const selection = $getSelection();
			if (!$isRangeSelection(selection)) return;
			$setBlocksType(selection, () =>
				active.heading ? $createParagraphNode() : $createHeadingNode("h2"),
			);
		});

	return (
		<div
			role="toolbar"
			aria-label={t("notes.editor.toolbar.label")}
			className="flex flex-wrap items-center gap-0.5"
		>
			<button
				type="button"
				className={toolButton}
				aria-label={t("notes.editor.toolbar.heading")}
				title={t("notes.editor.toolbar.heading")}
				aria-pressed={active.heading}
				disabled={disabled}
				onMouseDown={keepSelection}
				onClick={toggleHeading}
			>
				<Heading className="size-4" aria-hidden />
			</button>
			{lists.map(({ kind, icon: Icon, key, command }) => (
				<button
					key={kind}
					type="button"
					className={toolButton}
					aria-label={t(`notes.editor.toolbar.${key}`)}
					title={t(`notes.editor.toolbar.${key}`)}
					aria-pressed={active.list === kind}
					disabled={disabled}
					onMouseDown={keepSelection}
					onClick={() =>
						active.list === kind
							? editor.dispatchCommand(REMOVE_LIST_COMMAND, undefined)
							: editor.dispatchCommand(command, undefined)
					}
				>
					<Icon className="size-4" aria-hidden />
				</button>
			))}
			<span className="mx-1 h-5 w-px bg-line" aria-hidden />
			{formats.map(({ format, icon: Icon, key }) => (
				<button
					key={format}
					type="button"
					className={toolButton}
					aria-label={t(`notes.editor.toolbar.${key}`)}
					title={t(`notes.editor.toolbar.${key}`)}
					aria-pressed={active.formats.has(format)}
					disabled={disabled}
					onMouseDown={keepSelection}
					onClick={() => editor.dispatchCommand(FORMAT_TEXT_COMMAND, format)}
				>
					<Icon className="size-4" aria-hidden />
				</button>
			))}
			<LinkButton current={active.link} disabled={disabled} />
			<AttachButton disabled={disabled} />
		</div>
	);
}

/** Lets the user choose files and inserts them at the cursor, or at the end. */
function AttachButton({ disabled }: { disabled: boolean }) {
	const { t } = useTranslation();
	const [editor] = useLexicalComposerContext();
	const fileInput = useRef<HTMLInputElement>(null);
	const limits = useQuery({ ...attachmentsQueryOptions, enabled: !disabled });

	return (
		<>
			<button
				type="button"
				className={toolButton}
				aria-label={t("notes.editor.toolbar.attach")}
				title={t("notes.editor.toolbar.attach")}
				disabled={disabled}
				onMouseDown={keepSelection}
				onClick={() => fileInput.current?.click()}
			>
				<FilePlus2 className="size-4" aria-hidden />
			</button>
			<input
				ref={fileInput}
				type="file"
				multiple
				hidden
				accept={limits.data?.allowedTypes.join(",")}
				data-testid="attachment-input"
				onChange={(e) => {
					const files = [...(e.target.files ?? [])];
					e.target.value = "";
					if (files.length)
						editor.dispatchCommand(INSERT_ATTACHMENTS_COMMAND, files);
				}}
			/>
		</>
	);
}

/** Opens a small form to add, change or remove the link at the selection. */
function LinkButton({
	current,
	disabled,
}: {
	current: string | null;
	disabled: boolean;
}) {
	const { t } = useTranslation();
	const [editor] = useLexicalComposerContext();
	const [open, setOpen] = useState(false);
	const [url, setUrl] = useState("");
	const [error, setError] = useState(false);
	// The form takes the focus, so the selection the link is for is kept here.
	const [selection, setSelection] = useState<BaseSelection | null>(null);

	const toggleLink = (value: string | null) =>
		editor.update(() => {
			if (selection) $setSelection(selection.clone());
			$toggleLink(value);
		});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const value = url.trim();
		if (!isLinkUrl(value)) {
			setError(true);
			return;
		}
		toggleLink(value);
		setOpen(false);
	}

	return (
		<Popover.Root
			open={open}
			onOpenChange={(next) => {
				setOpen(next);
				if (next) {
					setUrl(current ?? "https://");
					setError(false);
					setSelection(
						editor
							.getEditorState()
							.read(() => $getSelection()?.clone() ?? null),
					);
				}
			}}
		>
			<Popover.Trigger
				className={toolButton}
				aria-label={t("notes.editor.toolbar.link")}
				title={t("notes.editor.toolbar.link")}
				aria-pressed={current !== null}
				disabled={disabled}
				onMouseDown={keepSelection}
			>
				<Link className="size-4" aria-hidden />
			</Popover.Trigger>
			<Popover.Portal>
				<Popover.Positioner sideOffset={6} align="start" className="z-50">
					<Popover.Popup
						className={`${menuPopup} w-80 max-w-[calc(100vw-2rem)] p-3`}
					>
						<Popover.Title className="sr-only">
							{t("notes.editor.toolbar.link")}
						</Popover.Title>
						<form onSubmit={onSubmit} noValidate>
							<label className="block text-sm font-medium">
								{t("notes.editor.link.label")}
								<input
									type="url"
									value={url}
									onChange={(e) => {
										setUrl(e.target.value);
										setError(false);
									}}
									aria-invalid={error}
									aria-describedby={error ? "note-link-error" : undefined}
									className={textInput}
								/>
							</label>
							{error && (
								<p
									id="note-link-error"
									role="alert"
									className="mt-2 text-sm text-danger"
								>
									{t("notes.editor.link.invalid")}
								</p>
							)}
							<div className="mt-3 flex justify-end gap-2">
								{current !== null && (
									<button
										type="button"
										className={secondaryButton}
										onClick={() => {
											toggleLink(null);
											setOpen(false);
										}}
									>
										{t("notes.editor.link.remove")}
									</button>
								)}
								<button type="submit" className={primaryButton}>
									{t("notes.editor.link.apply")}
								</button>
							</div>
						</form>
					</Popover.Popup>
				</Popover.Positioner>
			</Popover.Portal>
		</Popover.Root>
	);
}
