import { Dialog } from "@base-ui/react/dialog";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Search, Trash2 } from "lucide-react";
import {
	type KeyboardEvent,
	type ReactNode,
	useEffect,
	useRef,
	useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
	dangerButton,
	iconButton,
	secondaryButton,
} from "#/components/buttonStyles";
import { dialogBackdrop, dialogPopup } from "#/components/dialogStyles";
import { ApiError } from "#/lib/api";
import {
	deleteTag,
	favouritesQueryKey,
	listTags,
	renameTag,
	type TagUsage,
	tagUsageQueryKey,
} from "./api";
import { MAX_TAG } from "./validate";

/** What happened to a tag's name, so the page can update its tag filter. */
export type TagChange =
	| { kind: "renamed"; from: string; to: string }
	| { kind: "deleted"; name: string };

const smallButton =
	"rounded-md px-2.5 py-1 text-sm font-medium disabled:opacity-50";

/**
 * All of the user's tags with their link counts, to rename, merge or delete
 * them. Activating a count filters the favourites by that tag.
 */
export function ManageTagsDialog({
	open,
	onClose,
	onFilter,
	onChange,
}: {
	open: boolean;
	onClose: () => void;
	onFilter: (tag: string) => void;
	onChange: (change: TagChange) => void;
}) {
	const { t } = useTranslation();
	return (
		<Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
			<Dialog.Portal>
				<Dialog.Backdrop className={dialogBackdrop} />
				<Dialog.Popup className={`${dialogPopup} w-[32rem]`}>
					<Dialog.Title className="text-lg font-semibold">
						{t("favourites.manageTags.title")}
					</Dialog.Title>
					{open && <TagList onFilter={onFilter} onChange={onChange} />}
					<div className="mt-6 flex justify-end">
						<Dialog.Close className={secondaryButton}>
							{t("common.close")}
						</Dialog.Close>
					</div>
				</Dialog.Popup>
			</Dialog.Portal>
		</Dialog.Root>
	);
}

function TagList({
	onFilter,
	onChange,
}: {
	onFilter: (tag: string) => void;
	onChange: (change: TagChange) => void;
}) {
	const { t } = useTranslation();
	const tags = useQuery({ queryKey: tagUsageQueryKey, queryFn: listTags });
	const [query, setQuery] = useState("");

	if (tags.isPending) {
		return <p className="mt-4 text-sm text-muted">{t("common.loading")}</p>;
	}
	if (tags.isError) {
		return (
			<p role="alert" className="mt-4 text-sm text-danger">
				{t("common.genericError")}
			</p>
		);
	}
	if (tags.data.length === 0) {
		return (
			<p className="mt-4 text-sm text-muted">
				{t("favourites.manageTags.empty")}
			</p>
		);
	}
	const q = query.trim().toLowerCase();
	const shown = tags.data.filter((tag) => tag.name.toLowerCase().includes(q));
	return (
		<>
			<div className="relative mt-4">
				<Search
					className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted"
					aria-hidden
				/>
				<input
					type="search"
					value={query}
					onChange={(e) => setQuery(e.target.value)}
					aria-label={t("favourites.manageTags.search")}
					placeholder={t("favourites.manageTags.search")}
					className="w-full rounded-lg border border-line bg-paper/40 py-2 pr-3 pl-9 text-sm outline-none focus:border-ink"
				/>
			</div>
			<ul className="mt-3 max-h-80 overflow-y-auto">
				{shown.map((tag) => (
					<TagRow
						key={tag.id}
						tag={tag}
						all={tags.data}
						onFilter={onFilter}
						onChange={onChange}
					/>
				))}
			</ul>
		</>
	);
}

type RowMode =
	| { kind: "view" }
	| { kind: "rename"; name: string }
	| { kind: "merge"; name: string; into: string }
	| { kind: "delete" };

function TagRow({
	tag,
	all,
	onFilter,
	onChange,
}: {
	tag: TagUsage;
	all: TagUsage[];
	onFilter: (tag: string) => void;
	onChange: (change: TagChange) => void;
}) {
	const { t } = useTranslation();
	const queryClient = useQueryClient();
	const [mode, setMode] = useState<RowMode>({ kind: "view" });
	const row = useRef<HTMLLIElement>(null);
	// A question with its buttons can wrap below the list's visible area.
	useEffect(() => {
		if (mode.kind === "merge" || mode.kind === "delete") {
			row.current?.scrollIntoView?.({ block: "nearest" });
		}
	}, [mode.kind]);
	const refresh = () => {
		void queryClient.invalidateQueries({ queryKey: tagUsageQueryKey });
		void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
		void queryClient.invalidateQueries({ queryKey: ["tags"] });
	};

	const rename = useMutation({
		mutationFn: (v: { name: string; merge: boolean }) =>
			renameTag(tag.id, v.name, v.merge),
		onSuccess: (renamed) => {
			setMode({ kind: "view" });
			onChange({ kind: "renamed", from: tag.name, to: renamed.name });
			refresh();
		},
		onError: (e, v) => {
			if (e instanceof ApiError && e.code === "tag_exists") {
				const into =
					all.find(
						(other) =>
							other.id !== tag.id &&
							other.name.toLowerCase() === v.name.trim().toLowerCase(),
					)?.name ?? v.name.trim();
				setMode({ kind: "merge", name: v.name, into });
			}
		},
	});
	const remove = useMutation({
		mutationFn: () => deleteTag(tag.id),
		onSuccess: () => {
			onChange({ kind: "deleted", name: tag.name });
			refresh();
		},
	});

	const failed = (rename.isError && mode.kind !== "merge") || remove.isError;
	let content: ReactNode;
	if (mode.kind === "rename") {
		const name = mode.name;
		const save = () => {
			// Enter disables the input, which blurs it: save only once.
			if (rename.isPending) return;
			if (name.trim() === "" || name.trim() === tag.name) {
				setMode({ kind: "view" });
				return;
			}
			rename.mutate({ name, merge: false });
		};
		content = (
			<input
				// biome-ignore lint/a11y/noAutofocus: the input replaces the button that was just activated
				autoFocus
				value={name}
				maxLength={MAX_TAG}
				aria-label={t("favourites.manageTags.newName", { name: tag.name })}
				onChange={(e) => setMode({ kind: "rename", name: e.target.value })}
				onKeyDown={(e: KeyboardEvent) => {
					if (e.key === "Enter") {
						e.preventDefault();
						save();
					}
					if (e.key === "Escape") {
						// Only cancels the rename, doesn't close the dialog.
						e.preventDefault();
						e.stopPropagation();
						setMode({ kind: "view" });
					}
				}}
				onBlur={save}
				disabled={rename.isPending}
				className="min-w-0 flex-1 rounded-md border border-ink bg-paper/40 px-2 py-1 text-sm outline-none"
			/>
		);
	} else if (mode.kind === "merge") {
		content = (
			<div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-sm">
				<span className="min-w-0">
					{t("favourites.manageTags.mergeQuestion", {
						from: tag.name,
						into: mode.into,
					})}
				</span>
				<button
					type="button"
					className={`${smallButton} bg-ink text-paper`}
					disabled={rename.isPending}
					onClick={() => rename.mutate({ name: mode.name, merge: true })}
				>
					{t("favourites.manageTags.merge")}
				</button>
				<button
					type="button"
					className={`${smallButton} hover:bg-paper`}
					onClick={() => setMode({ kind: "view" })}
				>
					{t("favourites.dialog.cancel")}
				</button>
			</div>
		);
	} else if (mode.kind === "delete") {
		content = (
			<div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-sm">
				<span className="min-w-0">
					{t("favourites.manageTags.deleteQuestion", {
						name: tag.name,
						count: tag.linkCount,
					})}
				</span>
				<button
					type="button"
					className={`${dangerButton} px-2.5 py-1`}
					disabled={remove.isPending}
					onClick={() => remove.mutate()}
				>
					{t("favourites.manageTags.delete")}
				</button>
				<button
					type="button"
					className={`${smallButton} hover:bg-paper`}
					onClick={() => setMode({ kind: "view" })}
				>
					{t("favourites.dialog.cancel")}
				</button>
			</div>
		);
	} else {
		content = (
			<>
				<span className="min-w-0 flex-1 truncate text-sm">{tag.name}</span>
				<button
					type="button"
					className="rounded-md px-2 py-0.5 text-xs text-muted tabular-nums hover:bg-paper hover:text-ink"
					aria-label={t("favourites.manageTags.show", {
						name: tag.name,
						count: tag.linkCount,
					})}
					onClick={() => onFilter(tag.name)}
				>
					{tag.linkCount}
				</button>
				<button
					type="button"
					className={iconButton}
					aria-label={t("favourites.manageTags.rename", { name: tag.name })}
					onClick={() => {
						rename.reset();
						setMode({ kind: "rename", name: tag.name });
					}}
				>
					<Pencil className="size-4" aria-hidden />
				</button>
				<button
					type="button"
					className={`${iconButton} hover:text-danger`}
					aria-label={t("favourites.manageTags.deleteTag", { name: tag.name })}
					onClick={() => {
						remove.reset();
						setMode({ kind: "delete" });
					}}
				>
					<Trash2 className="size-4" aria-hidden />
				</button>
			</>
		);
	}

	return (
		<li ref={row} className="border-b border-line/60 py-1.5 last:border-b-0">
			<div className="flex min-h-8 items-center gap-1">{content}</div>
			{failed && (
				<p role="alert" className="mt-1 text-sm text-danger">
					{rename.error instanceof ApiError &&
					rename.error.code === "invalid_tag"
						? t("favourites.manageTags.invalidName")
						: t("common.genericError")}
				</p>
			)}
		</li>
	);
}
