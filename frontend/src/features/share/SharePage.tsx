import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { BookmarkCheck } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { primaryButton, secondaryButton } from "#/components/buttonStyles";
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
import { ApiError } from "#/lib/api";
import { usePageTitle } from "#/lib/usePageTitle";
import { type ShareParams, sharedLink } from "./sharedLink";

/** The folder last saved into on this page, remembered in this browser. */
export const SHARE_FOLDER_STORAGE_KEY = "boardly.share.folder";

function storedFolder(): string | null {
	try {
		return localStorage.getItem(SHARE_FOLDER_STORAGE_KEY);
	} catch {
		return null;
	}
}

function rememberFolder(id: string) {
	try {
		localStorage.setItem(SHARE_FOLDER_STORAGE_KEY, id);
	} catch {
		// Storage can be unavailable (private mode); the next share starts at the root.
	}
}

/** The remembered folder while it still exists, otherwise the root. */
function defaultFolder(root: FavouriteNode): string {
	const stored = storedFolder();
	const node = stored ? findNode(root, stored) : undefined;
	return node?.kind === "folder" ? node.id : root.id;
}

/**
 * "Save link": a form for a new link, prefilled from what another app shared
 * (the manifest's share target) or empty (the "Add link" shortcut).
 */
export function SharePage({ shared }: { shared: ShareParams }) {
	const { t } = useTranslation();
	usePageTitle(t("share.title"));
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const favourites = useQuery(favouritesQueryOptions);
	const root = favourites.data;

	const [initial] = useState(() => sharedLink(shared));
	const [url, setUrl] = useState(initial.url);
	const [title, setTitle] = useState(initial.title);
	const [tags, setTags] = useState<string[]>([]);
	const [chosenFolder, setChosenFolder] = useState<string | null>(null);
	const [problem, setProblem] = useState<FieldError | "generic" | null>(null);
	const [savedIn, setSavedIn] = useState<string | null>(null);

	const folderId = chosenFolder ?? (root ? defaultFolder(root) : "");

	const save = useMutation({
		mutationFn: () =>
			createFavourite({ parentId: folderId, kind: "link", url, title, tags }),
		onSuccess: () => {
			rememberFolder(folderId);
			const folder = root && findNode(root, folderId);
			setSavedIn(
				!folder || folder.id === root?.id
					? t("favourites.title")
					: label(folder),
			);
			void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			void queryClient.invalidateQueries({ queryKey: ["tags"] });
		},
		onError: (e) => {
			if (e instanceof ApiError && e.code === "not_found") {
				setChosenFolder(null);
				void queryClient.invalidateQueries({ queryKey: favouritesQueryKey });
			}
			setProblem(
				(e instanceof ApiError && fieldErrorFromCode(e.code, "link")) ||
					"generic",
			);
		},
	});

	function onSubmit(event: FormEvent) {
		event.preventDefault();
		const found =
			checkUrl(url) ??
			checkLinkTitle(title) ??
			tags.map(checkTag).find((p) => p !== null) ??
			null;
		setProblem(found);
		if (!found) save.mutate();
	}

	function saveAnother() {
		setUrl("");
		setTitle("");
		setTags([]);
		setProblem(null);
		setSavedIn(null);
		// Drop the shared values, so a reload doesn't prefill them again.
		void navigate({ to: "/share", search: {}, replace: true });
	}

	if (savedIn !== null) {
		return (
			<section className="mx-auto max-w-md py-16 text-center">
				<BookmarkCheck
					className="mx-auto size-8 text-ink"
					strokeWidth={1.75}
					aria-hidden
				/>
				<h1 className="mt-4 text-3xl font-semibold tracking-tight">
					{t("share.savedTitle")}
				</h1>
				<p className="mt-3 text-muted">
					{t("share.savedText", { folder: savedIn })}
				</p>
				<div className="mt-8 flex flex-wrap items-center justify-center gap-3">
					<Link to="/favourites" className={primaryButton}>
						{t("share.openFavourites")}
					</Link>
					<button
						type="button"
						className={secondaryButton}
						onClick={saveAnother}
					>
						{t("share.saveAnother")}
					</button>
				</div>
			</section>
		);
	}

	return (
		<section className="mx-auto max-w-md py-12">
			<h1 className="text-3xl font-semibold tracking-tight">
				{t("share.title")}
			</h1>
			<p className="mt-2 text-muted">{t("share.intro")}</p>
			<form className="mt-8 space-y-4" onSubmit={onSubmit} noValidate>
				<LinkFields
					url={url}
					onUrlChange={setUrl}
					title={title}
					onTitleChange={setTitle}
					tags={tags}
					onTagsChange={setTags}
					folder={{ value: folderId, onChange: setChosenFolder }}
				/>
				{problem && (
					<p role="alert" className="text-sm text-danger">
						{problem === "generic"
							? t("common.genericError")
							: t(`favourites.errors.${problem}`)}
					</p>
				)}
				<div className="flex justify-end gap-3 pt-2">
					<Link to="/" className={secondaryButton}>
						{t("favourites.dialog.cancel")}
					</Link>
					<button
						type="submit"
						className={primaryButton}
						disabled={save.isPending || !root}
					>
						{save.isPending
							? t("favourites.dialog.saving")
							: t("favourites.dialog.save")}
					</button>
				</div>
			</form>
		</section>
	);
}
