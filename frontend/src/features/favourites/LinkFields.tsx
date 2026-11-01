import { useTranslation } from "react-i18next";
import { textInput } from "#/components/dialogStyles";
import { DomainHint } from "./DomainHint";
import { FavouritePicker } from "./FavouritePicker";
import { TagInput } from "./TagInput";
import { MAX_LINK_TITLE, MAX_URL } from "./validate";

/**
 * The fields of a link: title, URL, tags and, when folder is given, the folder
 * to save it in. With hint, a note that the page's description and preview are
 * loaded in the background, for new links. Under the URL, a hint with how many
 * links are on its domain, not counting excludeId (the link being edited).
 * Used by the favourites dialog and the "Save link" page.
 */
export function LinkFields({
	url,
	onUrlChange,
	title,
	onTitleChange,
	tags,
	onTagsChange,
	folder,
	hint = false,
	excludeId,
}: {
	url: string;
	onUrlChange: (url: string) => void;
	title: string;
	onTitleChange: (title: string) => void;
	tags: string[];
	onTagsChange: (tags: string[]) => void;
	folder?: { value: string; onChange: (id: string) => void };
	hint?: boolean;
	excludeId?: string;
}) {
	const { t } = useTranslation();
	return (
		<>
			<div>
				<label htmlFor="favourite-title" className="block text-sm font-medium">
					{t("favourites.dialog.titleLabel")}
				</label>
				<input
					id="favourite-title"
					autoComplete="off"
					maxLength={MAX_LINK_TITLE}
					value={title}
					onChange={(e) => onTitleChange(e.target.value)}
					className={textInput}
				/>
			</div>
			<div>
				<label htmlFor="favourite-url" className="block text-sm font-medium">
					{t("favourites.dialog.urlLabel")}
				</label>
				<input
					id="favourite-url"
					type="url"
					inputMode="url"
					autoComplete="off"
					maxLength={MAX_URL}
					placeholder={t("favourites.dialog.urlPlaceholder")}
					value={url}
					onChange={(e) => onUrlChange(e.target.value)}
					aria-describedby="favourite-url-domain"
					className={textInput}
				/>
				<DomainHint id="favourite-url-domain" url={url} excludeId={excludeId} />
			</div>
			<TagInput value={tags} onChange={onTagsChange} />
			{folder && (
				<FavouritePicker
					id="favourite-folder"
					label={t("favourites.dialog.folderLabel")}
					mode="folder"
					value={folder.value}
					onChange={folder.onChange}
				/>
			)}
			{hint && (
				<p className="text-sm text-muted">{t("favourites.dialog.fetchHint")}</p>
			)}
		</>
	);
}
