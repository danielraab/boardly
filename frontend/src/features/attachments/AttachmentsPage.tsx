import { AlertDialog } from "@base-ui/react/alert-dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Paperclip, ShieldCheck, Trash2 } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { dangerButton, secondaryButton } from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { widePage } from "#/components/pageWidths";
import { type Note, notesQueryOptions } from "#/features/notes/api";
import { displayTitle } from "#/features/notes/document/text";
import { NoteMarks } from "#/features/notes/NoteMarks";
import { usePageTitle } from "#/lib/usePageTitle";
import { AttachmentIcon } from "./AttachmentView";
import {
	type Attachment,
	type AttachmentList,
	attachmentKind,
	attachmentsQueryKey,
	attachmentsQueryOptions,
	deleteAttachment,
	formatSize,
	setAttachmentKeep,
} from "./api";
import { useImageViewer } from "./ImageViewer";

export const attachmentFilters = [
	"all",
	"in-use",
	"unused",
	"marked",
	"kept",
] as const;

export type AttachmentFilter = (typeof attachmentFilters)[number];

/** The filter search param of /attachments; "all" is the default. */
export type AttachmentsSearch = { filter?: Exclude<AttachmentFilter, "all"> };

const matches: Record<AttachmentFilter, (a: Attachment) => boolean> = {
	all: () => true,
	"in-use": (a) => a.noteIds.length > 0,
	unused: (a) => a.noteIds.length === 0,
	marked: (a) => a.deletedAt !== null && !a.keep,
	kept: (a) => a.keep,
};

/**
 * /attachments: all of the user's attachments with the notes that use them,
 * filters, and keeping and deleting them.
 */
export function AttachmentsPage({
	search,
	onSearchChange,
}: {
	search: AttachmentsSearch;
	onSearchChange: (search: AttachmentsSearch) => void;
}) {
	const { t, i18n } = useTranslation();
	usePageTitle(t("attachments.page.title"));
	const list = useQuery(attachmentsQueryOptions);
	const notes = useQuery(notesQueryOptions);
	const images = useImageViewer();
	const filter: AttachmentFilter = search.filter ?? "all";
	const all = list.data?.attachments ?? [];
	const shown = all.filter(matches[filter]);
	const notesById = new Map((notes.data ?? []).map((n) => [n.id, n]));
	const total = all.reduce((sum, a) => sum + a.size, 0);
	const imageList = shown.filter(
		(a) => attachmentKind(a.contentType) === "image",
	);

	return (
		<section className={`${widePage} pt-12 pb-16`}>
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("attachments.page.title")}
			</h1>
			<p className="mt-2 max-w-[60ch] text-muted">
				{t("attachments.page.intro")}
			</p>
			{list.data && (
				<p className="mt-4 text-sm font-medium">
					{t("attachments.page.summary", {
						count: all.length,
						size: formatSize(total, i18n.language),
					})}
				</p>
			)}
			{list.isError && (
				<p role="alert" className="mt-6 text-sm text-danger">
					{t("common.genericError")}
				</p>
			)}
			{list.data && all.length === 0 && (
				<div className="mt-8 rounded-xl border border-dashed border-line p-10 text-center">
					<Paperclip className="mx-auto size-8 text-muted" aria-hidden />
					<p className="mt-3 font-medium">{t("attachments.page.empty")}</p>
					<p className="mt-1 text-sm text-muted">
						{t("attachments.page.emptyHint")}
					</p>
				</div>
			)}
			{list.data && all.length > 0 && (
				<>
					<fieldset className="mt-6 flex flex-wrap gap-2">
						<legend className="sr-only">
							{t("attachments.page.filters.label")}
						</legend>
						{attachmentFilters.map((f) => (
							<button
								key={f}
								type="button"
								aria-pressed={f === filter}
								onClick={() =>
									onSearchChange({ filter: f === "all" ? undefined : f })
								}
								className={`${secondaryButton} px-3 py-1.5 aria-pressed:border-ink aria-pressed:bg-surface`}
							>
								{t(`attachments.page.filters.${f}`)}{" "}
								<span className="text-muted tabular-nums">
									{all.filter(matches[f]).length}
								</span>
							</button>
						))}
					</fieldset>
					{shown.length === 0 ? (
						<p className="mt-6 rounded-xl border border-dashed border-line p-10 text-center text-muted">
							{t("attachments.page.emptyFilter")}
						</p>
					) : (
						<ul className="mt-6 divide-y divide-line rounded-xl border border-line bg-surface">
							{shown.map((a) => (
								<AttachmentEntry
									key={a.id}
									attachment={a}
									notes={a.noteIds
										.map((id) => notesById.get(id))
										.filter((n) => n !== undefined)}
									onOpenImage={() => images.open(imageList, a.id)}
								/>
							))}
						</ul>
					)}
				</>
			)}
			{images.viewer}
		</section>
	);
}

/** Replaces an attachment in the cached list, or removes it with null. */
function useSetCached() {
	const queryClient = useQueryClient();
	return (id: string, next: Attachment | null) =>
		queryClient.setQueryData<AttachmentList>(attachmentsQueryKey, (list) =>
			list
				? {
						...list,
						attachments: next
							? list.attachments.map((a) => (a.id === id ? next : a))
							: list.attachments.filter((a) => a.id !== id),
					}
				: list,
		);
}

function AttachmentEntry({
	attachment: a,
	notes,
	onOpenImage,
}: {
	attachment: Attachment;
	notes: Note[];
	onOpenImage: () => void;
}) {
	const { t, i18n } = useTranslation();
	const queryClient = useQueryClient();
	const setCached = useSetCached();
	const [deleting, setDeleting] = useState(false);
	const date = new Intl.DateTimeFormat(i18n.language, { dateStyle: "medium" });
	const inUse = a.noteIds.length > 0;
	const keep = useMutation({
		mutationFn: (value: boolean) => setAttachmentKeep(a.id, value),
		onSuccess: (updated) => setCached(a.id, updated),
		onError: () =>
			void queryClient.invalidateQueries({ queryKey: attachmentsQueryKey }),
	});
	const isImage = attachmentKind(a.contentType) === "image";
	const nameClass =
		"block truncate text-left font-medium hover:underline focus-visible:underline focus-visible:outline-none";

	return (
		<li className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:gap-4">
			<div className="flex min-w-0 items-center gap-3 md:flex-1">
				{isImage ? (
					<img
						src={a.url}
						alt=""
						className="size-12 shrink-0 rounded-md border border-line object-cover"
					/>
				) : (
					<span className="flex size-12 shrink-0 items-center justify-center rounded-md border border-line">
						<AttachmentIcon
							contentType={a.contentType}
							className="size-6 text-muted"
						/>
					</span>
				)}
				<div className="min-w-0 flex-1">
					{isImage ? (
						<button type="button" className={nameClass} onClick={onOpenImage}>
							{a.name}
						</button>
					) : (
						<a
							href={a.url}
							target="_blank"
							rel="noopener noreferrer"
							className={nameClass}
						>
							{a.name}
						</a>
					)}
					<p className="truncate text-sm text-muted">
						{a.contentType} · {formatSize(a.size, i18n.language)} ·{" "}
						{date.format(new Date(a.createdAt))}
					</p>
				</div>
			</div>
			<div className="min-w-0 text-sm md:w-56">
				{notes.length === 0 ? (
					<p className="text-muted">{t("attachments.page.notUsed")}</p>
				) : (
					<ul className="space-y-1">
						{notes.map((n) => (
							<li key={n.id} className="flex min-w-0 items-center gap-2">
								<Link
									to="/notes/$noteId"
									params={{ noteId: n.id }}
									className="min-w-0 truncate text-accent underline"
								>
									{displayTitle(n) ?? t("notes.untitled")}
								</Link>
								<NoteMarks pinned={false} archived={n.archived} />
							</li>
						))}
					</ul>
				)}
				{a.keep ? (
					<p className="mt-1 flex items-center gap-1 text-muted">
						<ShieldCheck className="size-4 shrink-0" aria-hidden />
						{t("attachments.page.kept")}
						{a.deletedAt &&
							` · ${t("attachments.page.markedOn", {
								date: date.format(new Date(a.deletedAt)),
							})}`}
					</p>
				) : (
					a.deleteAt && (
						<p className="mt-1 text-danger">
							{t("attachments.page.deleteOn", {
								date: date.format(new Date(a.deleteAt)),
							})}
						</p>
					)
				)}
				{keep.isError && (
					<p role="alert" className="mt-1 text-danger">
						{t("attachments.page.changeFailed")}
					</p>
				)}
			</div>
			<div className="flex shrink-0 gap-2">
				<button
					type="button"
					className={`${secondaryButton} px-3`}
					disabled={keep.isPending}
					onClick={() => keep.mutate(!a.keep)}
				>
					{t(a.keep ? "attachments.page.dontKeep" : "attachments.page.keep")}
				</button>
				<button
					type="button"
					className={`${secondaryButton} px-3`}
					disabled={inUse}
					aria-label={t("attachments.page.deleteName", { name: a.name })}
					title={inUse ? t("attachments.page.inUse") : undefined}
					aria-describedby={inUse ? `in-use-${a.id}` : undefined}
					onClick={() => setDeleting(true)}
				>
					<Trash2 className="size-4" aria-hidden />
					{t("attachments.page.delete")}
				</button>
				{inUse && (
					<span id={`in-use-${a.id}`} hidden>
						{t("attachments.page.inUse")}
					</span>
				)}
			</div>
			<DeleteAttachmentDialog
				open={deleting}
				attachment={a}
				onClose={() => setDeleting(false)}
				onDeleted={() => setCached(a.id, null)}
			/>
		</li>
	);
}

function DeleteAttachmentDialog({
	open,
	attachment,
	onClose,
	onDeleted,
}: {
	open: boolean;
	attachment: Attachment;
	onClose: () => void;
	onDeleted: () => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const remove = useMutation({
		mutationFn: () => deleteAttachment(attachment.id),
		onSuccess: () => {
			onClose();
			onDeleted();
		},
		onError: () =>
			void queryClient.invalidateQueries({ queryKey: attachmentsQueryKey }),
	});
	return (
		<AlertDialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
			<AlertDialog.Portal>
				<AlertDialog.Backdrop className={dialogBackdrop} />
				<AlertDialog.Popup className={dialogPopup}>
					<AlertDialog.Title className="text-lg font-semibold">
						{t("attachments.page.deleteDialog.title")}
					</AlertDialog.Title>
					<AlertDialog.Description className="mt-1 text-sm text-muted">
						{t("attachments.page.deleteDialog.text", { name: attachment.name })}
					</AlertDialog.Description>
					{remove.isError && (
						<p role="alert" className="mt-3 text-sm text-danger">
							{t("attachments.page.changeFailed")}
						</p>
					)}
					<div className="mt-6 flex justify-end gap-3">
						<AlertDialog.Close className={secondaryButton}>
							{t("attachments.page.deleteDialog.cancel")}
						</AlertDialog.Close>
						<button
							type="button"
							className={dangerButton}
							disabled={remove.isPending}
							onClick={() => remove.mutate()}
						>
							{t("attachments.page.deleteDialog.confirm")}
						</button>
					</div>
				</AlertDialog.Popup>
			</AlertDialog.Portal>
		</AlertDialog.Root>
	);
}
