import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { BookmarkCheck, NotebookPen } from "lucide-react";
import { type FormEvent, type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
import { textInput } from "#/components/dialogStyles";
import { focusPage } from "#/components/pageWidths";
import { useMe } from "#/features/auth/session";
import {
	createFavourite,
	type FavouriteNode,
	favouritesQueryKey,
	favouritesQueryOptions,
} from "#/features/favourites/api";
import { LinkFields } from "#/features/favourites/LinkFields";
import { findNode, label } from "#/features/favourites/tree";
import {
	checkLinkTitle,
	checkTag,
	checkUrl,
	type FieldError,
	fieldErrorFromCode,
} from "#/features/favourites/validate";
import {
	createNoteFromText,
	MAX_NOTE_TITLE,
	type NoteCategory,
	noteCategoriesQueryKey,
	noteCategoriesQueryOptions,
	notesQueryKey,
} from "#/features/notes/api";
import { CategoryPicker } from "#/features/notes/categories/CategoryPicker";
import { categoryPath } from "#/features/notes/categories/tree";
import { ApiError } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import {
	type ShareMode,
	type ShareParams,
	sharedLink,
	sharedMode,
	sharedNote,
} from "./sharedLink";

/** The user's default folder while it still exists, otherwise the root. */
function defaultFolder(root: FavouriteNode, shareFolderId: string | null) {
	const node = shareFolderId ? findNode(root, shareFolderId) : undefined;
	return node?.kind === "folder" ? node.id : root.id;
}

/** The user's default category while it still exists, otherwise none. */
function defaultCategory(
	categories: NoteCategory[] | undefined,
	noteCategoryId: string | null,
) {
	return categories?.some((c) => c.id === noteCategoryId)
		? noteCategoryId
		: null;
}

/** What was saved, for the confirmation. */
type Saved =
	| { mode: "link"; folder: string }
	| { mode: "note"; id: string; category: string | null };

/**
 * "Save link" and "Save note": a form for a new link or note, prefilled from
 * what another app shared (the manifest's share target) or empty (the "Add
 * link" shortcut).
 */
export function SharePage({ shared }: { shared: ShareParams }) {
	const { t } = useTranslation();
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const favourites = useQuery(favouritesQueryOptions);
	const categories = useQuery(noteCategoriesQueryOptions);
	const root = favourites.data;
	const { data: me } = useMe();

	const [mode, setMode] = useState<ShareMode>(() => sharedMode(shared));
	usePageTitle(t(mode === "note" ? "share.note.title" : "share.title"));

	const [initial] = useState(() => sharedLink(shared));
	const [url, setUrl] = useState(initial.url);
	const [title, setTitle] = useState(initial.title);
	const [tags, setTags] = useState<string[]>([]);
	const [chosenFolder, setChosenFolder] = useState<string | null>(null);

	const [initialNote] = useState(() => sharedNote(shared));
	const [noteTitle, setNoteTitle] = useState(initialNote.title);
	const [noteText, setNoteText] = useState(initialNote.text);
	const [chosenCategory, setChosenCategory] = useState<{
		id: string | null;
	} | null>(null);

	const [problem, setProblem] = useState<string | null>(null);
	const [saved, setSaved] = useState<Saved | null>(null);

	const folderId =
		chosenFolder ??
		(root ? defaultFolder(root, me?.shareFolderId ?? null) : "");
	const categoryId = chosenCategory
		? chosenCategory.id
		: defaultCategory(categories.data, me?.noteCategoryId ?? null);

	const saveLink = useMutation({
		mutationFn: () =>
			createFavourite({ parentId: folderId, kind: "link", url, title, tags }),
		onSuccess: () => {
			const folder = root && findNode(root, folderId);
			setSaved({
				mode: "link",
				folder:
					!folder || folder.id === root?.id
						? t("favourites.title")
						: label(folder),
			});
			void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
		},
		onError: (e) => {
			if (e instanceof ApiError && e.code === "not_found") {
				setChosenFolder(null);
				void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			}
			const field = e instanceof ApiError && fieldErrorFromCode(e.code, "link");
			setProblem(
				field ? t(`favourites.errors.${field}`) : t("common.genericError"),
			);
		},
	});

	const saveNote = useMutation({
		mutationFn: () =>
			createNoteFromText({ title: noteTitle, text: noteText, categoryId }),
		onSuccess: (note) => {
			setSaved({
				mode: "note",
				id: note.id,
				category: note.categoryId
					? categoryPath(categories.data ?? [], note.categoryId).join(" › ")
					: null,
			});
			void queryClient.invalidateQueries({ queryKey: notesQueryKey });
		},
		onError: (e) => {
			const code = e instanceof ApiError ? e.code : null;
			if (code === "invalid_note") {
				// The only server check the form can't make: the category is gone.
				setChosenCategory({ id: null });
				void queryClient.invalidateQueries({
					queryKey: noteCategoriesQueryKey,
				});
			}
			setProblem(
				code === "invalid_note"
					? t("share.note.categoryGone")
					: code === "invalid_document"
						? t("share.note.tooLarge")
						: t("common.genericError"),
			);
		},
	});

	const pending = saveLink.isPending || saveNote.isPending;

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		if (mode === "note") {
			setProblem(null);
			saveNote.mutate();
			return;
		}
		const found: FieldError | null =
			checkUrl(url) ??
			checkLinkTitle(title) ??
			tags.map(checkTag).find((p) => p !== null) ??
			null;
		setProblem(found && t(`favourites.errors.${found}`));
		if (!found) saveLink.mutate();
	}

	function switchTo(next: ShareMode) {
		setMode(next);
		setProblem(null);
	}

	function saveAnother() {
		setUrl("");
		setTitle("");
		setTags([]);
		setNoteTitle("");
		setNoteText("");
		setProblem(null);
		setSaved(null);
		// Drop the shared values, so a reload doesn't prefill them again.
		void navigate({ to: "/share", search: {}, replace: true });
	}

	if (saved?.mode === "link") {
		return (
			<Confirmation
				icon={
					<BookmarkCheck
						className="mx-auto size-8 text-ink"
						strokeWidth={1.75}
						aria-hidden
					/>
				}
				title={t("share.savedTitle")}
				text={t("share.savedText", { folder: saved.folder })}
				open={
					<Link to="/favourites" className={primaryButton}>
						{t("share.openFavourites")}
					</Link>
				}
				another={t("share.saveAnother")}
				onAnother={saveAnother}
			/>
		);
	}
	if (saved?.mode === "note") {
		return (
			<Confirmation
				icon={
					<NotebookPen
						className="mx-auto size-8 text-ink"
						strokeWidth={1.75}
						aria-hidden
					/>
				}
				title={t("share.note.savedTitle")}
				text={
					saved.category
						? t("share.note.savedText", { category: saved.category })
						: t("share.note.savedTextNone")
				}
				open={
					<Link
						to="/notes/$noteId"
						params={{ noteId: saved.id }}
						className={primaryButton}
					>
						{t("share.note.open")}
					</Link>
				}
				another={t("share.note.saveAnother")}
				onAnother={saveAnother}
			/>
		);
	}

	return (
		<section className={`${focusPage} py-12`}>
			<h1 className="text-3xl font-semibold tracking-tight">
				{t(mode === "note" ? "share.note.title" : "share.title")}
			</h1>
			<p className="mt-2 text-muted">
				{t(mode === "note" ? "share.note.intro" : "share.intro")}
			</p>
			<fieldset className="mt-6">
				<legend className="sr-only">{t("share.mode")}</legend>
				<div className="inline-flex rounded-lg border border-line p-0.5 text-sm">
					{(["link", "note"] as const).map((option) => (
						<label
							key={option}
							className="cursor-pointer rounded-md px-4 py-1.5 font-medium text-muted has-[:checked]:bg-ink has-[:checked]:text-paper has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-accent"
						>
							<input
								type="radio"
								name="share-mode"
								value={option}
								checked={mode === option}
								onChange={() => switchTo(option)}
								className="sr-only"
							/>
							{t(`share.modes.${option}`)}
						</label>
					))}
				</div>
			</fieldset>
			<form className="mt-6 space-y-4" onSubmit={onSubmit} noValidate>
				{mode === "link" ? (
					<LinkFields
						url={url}
						onUrlChange={setUrl}
						title={title}
						onTitleChange={setTitle}
						tags={tags}
						onTagsChange={setTags}
						folder={{ value: folderId, onChange: setChosenFolder }}
						hint
					/>
				) : (
					<>
						<div>
							<label
								htmlFor="share-note-title"
								className="block text-sm font-medium"
							>
								{t("share.note.titleLabel")}
							</label>
							<input
								id="share-note-title"
								value={noteTitle}
								maxLength={MAX_NOTE_TITLE}
								onChange={(e) => setNoteTitle(e.target.value)}
								className={textInput}
							/>
						</div>
						<div>
							<label
								htmlFor="share-note-text"
								className="block text-sm font-medium"
							>
								{t("share.note.textLabel")}
							</label>
							<textarea
								id="share-note-text"
								rows={6}
								value={noteText}
								onChange={(e) => setNoteText(e.target.value)}
								className={`${textInput} resize-y`}
							/>
						</div>
						<CategoryPicker
							id="share-note-category"
							label={t("share.note.category")}
							emptyLabel={t("share.note.noCategory")}
							value={categoryId}
							onChange={(id) => setChosenCategory({ id })}
						/>
					</>
				)}
				{problem && (
					<p role="alert" className="text-sm text-danger">
						{problem}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Link to="/" className={secondaryButton}>
						{t("favourites.dialog.cancel")}
					</Link>
					<button
						type="submit"
						className={primaryButton}
						disabled={pending || (mode === "link" && !root)}
					>
						{pending
							? t("favourites.dialog.saving")
							: t("favourites.dialog.save")}
					</button>
				</div>
			</form>
		</section>
	);
}

function Confirmation({
	icon,
	title,
	text,
	open,
	another,
	onAnother,
}: {
	icon: ReactNode;
	title: string;
	text: string;
	open: ReactNode;
	another: string;
	onAnother: () => void;
}) {
	return (
		<section className={`${focusPage} py-16 text-center`}>
			{icon}
			<h1 className="mt-4 text-3xl font-semibold tracking-tight">{title}</h1>
			<p className="mt-3 text-muted">{text}</p>
			<div className="mt-8 flex flex-wrap items-center justify-center gap-3">
				{open}
				<button type="button" className={secondaryButton} onClick={onAnother}>
					{another}
				</button>
			</div>
		</section>
	);
}
