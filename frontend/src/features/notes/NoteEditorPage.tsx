import { AlertDialog } from "@base-ui/react/alert-dialog";
import { Menu } from "@base-ui/react/menu";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Ellipsis, RotateCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	secondaryButton,
	secondaryIconButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { type IconChoice, IconChooser } from "#/components/IconChooser";
import { menuItem, menuPopup } from "#/components/menuStyles";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	deleteNote,
	type Note,
	type NoteFields,
	noteCategoriesQueryOptions,
	notesQueryKey,
} from "./api";
import { ColourButton } from "./ColourPicker";
import { CategoryBadge } from "./categories/CategoryBadge";
import { noteBackground } from "./colours";
import { displayTitle } from "./document/text";
import { emptyDoc } from "./document/types";
import { NoteEditor } from "./editor/NoteEditor";
import { iconChoice, noteIcon } from "./noteIcon";
import { type SaveStatus, useAutosave } from "./useAutosave";

/**
 * The editor page of a note, or of a new one (note null) that is created on
 * the first change, starting in initialCategory. onCreated gets the new note;
 * onReload loads the note again after a conflict.
 */
export function NoteEditorPage({
	note,
	initialCategory = null,
	onCreated,
	onReload,
}: {
	note: Note | null;
	initialCategory?: string | null;
	onCreated: (note: Note) => void;
	onReload: () => void;
}) {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const categories = useQuery(noteCategoriesQueryOptions);
	const [draft, setDraft] = useState<NoteFields>(() => ({
		title: note?.title ?? null,
		icon: note?.icon ?? null,
		color: note?.color ?? null,
		categoryId: note ? note.categoryId : initialCategory,
		content: note?.content ?? emptyDoc(),
	}));
	const [titleText, setTitleText] = useState(note?.title ?? "");
	const [icon, setIcon] = useState<IconChoice>(iconChoice(draft.icon));
	const [noteId, setNoteId] = useState(note?.id ?? null);
	const [deleting, setDeleting] = useState(false);
	const { status, change, retry, discard } = useAutosave({
		note,
		draft,
		onCreated: (created) => {
			setNoteId(created.id);
			onCreated(created);
		},
	});
	const readOnly = status === "conflict";

	const update = (fields: Partial<NoteFields>) => {
		setDraft((d) => ({ ...d, ...fields }));
		change(fields);
	};

	const name = displayTitle(draft) ?? t("notes.untitled");
	usePageTitle(name);

	return (
		<section
			className={`mt-6 mb-16 rounded-2xl border border-line p-4 sm:p-6 ${draft.color ? "" : "bg-surface"}`}
			style={{ background: noteBackground(draft.color) }}
		>
			<div className="flex flex-wrap items-center gap-2">
				<Link to="/notes" className={`${secondaryButton} px-3`}>
					<ArrowLeft className="size-4" aria-hidden />
					{t("notes.editor.back")}
				</Link>
				<div className="ml-auto flex items-center gap-2">
					<SaveIndicator status={status} onRetry={retry} />
					<ColourButton
						value={draft.color}
						onChange={(color) => update({ color })}
						disabled={readOnly}
					/>
					<Menu.Root>
						<Menu.Trigger
							className={secondaryIconButton}
							aria-label={t("notes.editor.menu")}
						>
							<Ellipsis className="size-4" aria-hidden />
						</Menu.Trigger>
						<Menu.Portal>
							<Menu.Positioner align="end" sideOffset={6}>
								<Menu.Popup className={menuPopup}>
									<Menu.Item
										className={menuItem}
										disabled={noteId === null}
										onClick={() => setDeleting(true)}
									>
										<Trash2 className="size-4" aria-hidden />
										{t("notes.editor.delete")}
									</Menu.Item>
								</Menu.Popup>
							</Menu.Positioner>
						</Menu.Portal>
					</Menu.Root>
				</div>
			</div>

			{readOnly && (
				<div
					role="alert"
					className="mt-4 flex flex-wrap items-center gap-3 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm"
				>
					<span className="flex-1">{t("notes.editor.conflict")}</span>
					<button type="button" className={secondaryButton} onClick={onReload}>
						<RotateCw className="size-4" aria-hidden />
						{t("notes.editor.reload")}
					</button>
				</div>
			)}

			<div className="mt-6 flex items-center gap-3">
				<IconChooser
					variant="button"
					value={icon}
					onChange={(choice) => {
						setIcon(choice);
						if (!readOnly) update({ icon: noteIcon(choice) });
					}}
				/>
				<input
					value={titleText}
					readOnly={readOnly}
					onChange={(e) => {
						setTitleText(e.target.value);
						update({ title: e.target.value.trim() || null });
					}}
					placeholder={t("notes.editor.title")}
					aria-label={t("notes.editor.title")}
					maxLength={200}
					className="min-w-0 flex-1 bg-transparent text-2xl font-semibold outline-none placeholder:text-muted"
				/>
			</div>
			<div className="mt-3">
				<CategoryBadge
					categoryId={draft.categoryId}
					categories={categories.data ?? []}
					onChange={(categoryId) => update({ categoryId })}
					showAdd
					disabled={readOnly}
				/>
			</div>
			<div className="mt-6">
				<NoteEditor
					initial={draft.content}
					editable={!readOnly}
					onChange={(content) => update({ content })}
				/>
			</div>

			<DeleteNoteDialog
				open={deleting}
				noteId={noteId}
				name={name}
				onClose={() => setDeleting(false)}
				onDeleted={() => {
					discard();
					void navigate({ to: "/notes" });
				}}
			/>
		</section>
	);
}

function SaveIndicator({
	status,
	onRetry,
}: {
	status: SaveStatus;
	onRetry: () => void;
}) {
	const { t } = useTranslation();
	if (status === "idle" || status === "conflict") return null;
	if (status === "failed") {
		return (
			<span
				role="status"
				className="flex items-center gap-2 text-sm text-danger"
			>
				{t("notes.editor.status.failed")}
				<button
					type="button"
					className={`${secondaryButton} px-3 py-1`}
					onClick={onRetry}
				>
					{t("notes.editor.status.retry")}
				</button>
			</span>
		);
	}
	return (
		<span role="status" className="text-sm text-muted">
			{status === "saving"
				? t("notes.editor.status.saving")
				: t("notes.editor.status.saved")}
		</span>
	);
}

function DeleteNoteDialog({
	open,
	noteId,
	name,
	onClose,
	onDeleted,
}: {
	open: boolean;
	noteId: string | null;
	name: string;
	onClose: () => void;
	onDeleted: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const remove = useMutation({
		mutationFn: (id: string) => deleteNote(id),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: notesQueryKey });
			onDeleted();
		},
	});
	return (
		<AlertDialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{t("notes.editor.deleteDialog.title")}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-1 text-sm text-muted">
						{t("notes.editor.deleteDialog.text", { title: name })}
					</AlertDialog.Description>
					{remove.isError && (
						<p role="alert" className="mt-3 text-sm text-danger">
							{t("common.genericError")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{t("notes.editor.deleteDialog.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={dangerButton}
							disabled={remove.isPending || noteId === null}
							onClick={() => noteId && remove.mutate(noteId)}
						>
							{t("notes.editor.deleteDialog.confirm")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
